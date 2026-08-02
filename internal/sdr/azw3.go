package sdr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	mobiCompressionNone    = 1
	mobiCompressionPalmDOC = 2
	mobiCompressionHUFF    = 0x4448
	missingSection         = uint32(0xffffffff)
)

type palmDatabase struct {
	name     string
	identity string
	sections [][]byte
}

type mobiBook struct {
	db           *palmDatabase
	start        int
	title        string
	compression  uint16
	textRecords  int
	version      uint32
	huffOffset   uint32
	huffCount    uint32
	fdst         uint32
	fdstCount    uint32
	fragment     uint32
	skeleton     uint32
	trailerFlags uint16
}

type indexTag struct {
	tag            byte
	valuesPerEntry byte
	mask           byte
	endFlag        byte
}

type indexEntry struct {
	text string
	tags map[byte][]uint64
}

type skeletonEntry struct {
	fragmentCount int
	position      int
	length        int
}

type fragmentEntry struct {
	insertPosition int
	position       int
	length         int
}

// AssembleBook reads an unencrypted KF8 book and returns its title and
// reconstructed main markup flow. Annotation positions index this byte slice.
func AssembleBook(data []byte) (string, []byte, error) {
	db, err := parsePalmDatabase(data)
	if err != nil {
		return "", nil, err
	}
	book, err := findKF8Book(db)
	if err != nil {
		return "", nil, err
	}
	raw, err := book.rawMarkup()
	if err != nil {
		return "", nil, err
	}
	assembled, err := book.assembleKF8(raw)
	if err != nil {
		return "", nil, err
	}
	return book.title, assembled, nil
}

func parsePalmDatabase(data []byte) (*palmDatabase, error) {
	if len(data) < 78 {
		return nil, fmt.Errorf("book is too small to be a Palm database")
	}
	recordCount := int(binary.BigEndian.Uint16(data[76:78]))
	if recordCount == 0 || recordCount > 100_000 {
		return nil, fmt.Errorf("invalid Palm record count %d", recordCount)
	}
	tableEnd := 78 + recordCount*8
	if tableEnd > len(data) {
		return nil, fmt.Errorf("truncated Palm record table")
	}
	offsets := make([]int, recordCount+1)
	for i := 0; i < recordCount; i++ {
		offset := uint64(binary.BigEndian.Uint32(data[78+i*8 : 82+i*8]))
		if offset > uint64(len(data)) {
			return nil, fmt.Errorf("Palm record %d offset is outside the file", i)
		}
		offsets[i] = int(offset)
		if i > 0 && offsets[i] < offsets[i-1] {
			return nil, fmt.Errorf("Palm record offsets are not ordered")
		}
	}
	if offsets[0] < tableEnd {
		return nil, fmt.Errorf("first Palm record overlaps the record table")
	}
	offsets[recordCount] = len(data)
	sections := make([][]byte, recordCount)
	for i := range sections {
		sections[i] = data[offsets[i]:offsets[i+1]]
	}
	name := strings.TrimRight(string(data[:32]), "\x00")
	identity := string(data[60:68])
	if identity != "BOOKMOBI" && identity != "TEXtREAd" {
		return nil, fmt.Errorf("unsupported Palm identity %q", identity)
	}
	return &palmDatabase{name: name, identity: identity, sections: sections}, nil
}

func findKF8Book(db *palmDatabase) (*mobiBook, error) {
	if len(db.sections) == 0 {
		return nil, fmt.Errorf("book has no records")
	}
	firstBook, firstErr := parseMobiHeader(db, 0)
	if firstErr == nil && firstBook.version == 8 {
		return firstBook, nil
	}
	for i, section := range db.sections {
		if len(section) == 8 && string(section) == "BOUNDARY" && i+1 < len(db.sections) {
			book, err := parseMobiHeader(db, i+1)
			if err != nil {
				return nil, fmt.Errorf("parse hybrid KF8 header: %w", err)
			}
			return book, nil
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, fmt.Errorf("book has no KF8 content (Mobi7 is not supported)")
}

func parseMobiHeader(db *palmDatabase, start int) (*mobiBook, error) {
	if start < 0 || start >= len(db.sections) {
		return nil, fmt.Errorf("MOBI header record is missing")
	}
	header := db.sections[start]
	if len(header) < 40 || string(header[16:20]) != "MOBI" {
		return nil, fmt.Errorf("record %d is not a MOBI header", start)
	}
	headerLength := binary.BigEndian.Uint32(header[20:24])
	if headerLength < 0x18 || uint64(headerLength)+16 > uint64(len(header)) {
		return nil, fmt.Errorf("invalid MOBI header length %d", headerLength)
	}
	crypto := binary.BigEndian.Uint16(header[12:14])
	if crypto != 0 {
		return nil, fmt.Errorf("book is DRM-encrypted")
	}
	book := &mobiBook{
		db: db, start: start,
		compression: binary.BigEndian.Uint16(header[0:2]),
		textRecords: int(binary.BigEndian.Uint16(header[8:10])),
		version:     binary.BigEndian.Uint32(header[36:40]),
		title:       db.name,
		huffOffset:  missingSection, fdst: missingSection,
		fragment: missingSection, skeleton: missingSection,
	}
	if book.textRecords < 0 || start+book.textRecords >= len(db.sections) {
		return nil, fmt.Errorf("MOBI text record count %d exceeds the file", book.textRecords)
	}
	if len(header) >= 0x5c {
		offset := binary.BigEndian.Uint32(header[0x54:0x58])
		length := binary.BigEndian.Uint32(header[0x58:0x5c])
		if uint64(offset)+uint64(length) <= uint64(len(header)) && length > 0 {
			book.title = decodeMobiText(header[offset:offset+length], binary.BigEndian.Uint32(header[28:32]))
		}
	}
	if updated := updatedTitle(header, headerLength, binary.BigEndian.Uint32(header[28:32])); updated != "" {
		book.title = updated
	}
	if strings.TrimSpace(book.title) == "" {
		book.title = db.name
	}
	if len(header) >= 0x78 {
		book.huffOffset = binary.BigEndian.Uint32(header[0x70:0x74])
		book.huffCount = binary.BigEndian.Uint32(header[0x74:0x78])
	}
	if len(header) >= 0xc8 {
		book.fdst = binary.BigEndian.Uint32(header[0xc0:0xc4])
		book.fdstCount = binary.BigEndian.Uint32(header[0xc4:0xc8])
		if book.fdstCount <= 1 {
			book.fdst = missingSection
		}
	}
	if len(header) >= 0x100 {
		book.trailerFlags = binary.BigEndian.Uint16(header[0xf2:0xf4])
		book.fragment = binary.BigEndian.Uint32(header[0xf8:0xfc])
		book.skeleton = binary.BigEndian.Uint32(header[0xfc:0x100])
	}
	return book, nil
}

func updatedTitle(header []byte, mobiHeaderLength, codepage uint32) string {
	if len(header) < 0x84 || binary.BigEndian.Uint32(header[0x80:0x84])&0x40 == 0 {
		return ""
	}
	offset := int(mobiHeaderLength) + 16
	if offset+12 > len(header) || string(header[offset:offset+4]) != "EXTH" {
		return ""
	}
	length := int(binary.BigEndian.Uint32(header[offset+4 : offset+8]))
	count := int(binary.BigEndian.Uint32(header[offset+8 : offset+12]))
	if length < 12 || offset+length > len(header) || count > 100_000 {
		return ""
	}
	position := offset + 12
	for i := 0; i < count && position+8 <= offset+length; i++ {
		kind := binary.BigEndian.Uint32(header[position : position+4])
		size := int(binary.BigEndian.Uint32(header[position+4 : position+8]))
		if size < 8 || position+size > offset+length {
			break
		}
		if kind == 503 {
			return decodeMobiText(header[position+8:position+size], codepage)
		}
		position += size
	}
	return ""
}

func decodeMobiText(data []byte, codepage uint32) string {
	if codepage == 65001 || utf8.Valid(data) {
		return strings.TrimSpace(strings.TrimRight(string(data), "\x00"))
	}
	// MOBI commonly uses Windows-1252. Decode its non-ISO control range.
	cp1252 := [...]rune{0x20ac, 0x0081, 0x201a, 0x0192, 0x201e, 0x2026, 0x2020, 0x2021,
		0x02c6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008d, 0x017d, 0x008f,
		0x0090, 0x2018, 0x2019, 0x201c, 0x201d, 0x2022, 0x2013, 0x2014,
		0x02dc, 0x2122, 0x0161, 0x203a, 0x0153, 0x009d, 0x017e, 0x0178}
	var out strings.Builder
	for _, b := range data {
		switch {
		case b >= 0x80 && b <= 0x9f:
			out.WriteRune(cp1252[b-0x80])
		default:
			out.WriteRune(rune(b))
		}
	}
	return strings.TrimSpace(strings.TrimRight(out.String(), "\x00"))
}

func (b *mobiBook) absolute(relative uint32) (int, error) {
	if relative == missingSection {
		return -1, fmt.Errorf("required MOBI section is absent")
	}
	index := uint64(relative) + uint64(b.start)
	if index >= uint64(len(b.db.sections)) {
		return -1, fmt.Errorf("MOBI section %d is outside the file", index)
	}
	return int(index), nil
}

func (b *mobiBook) rawMarkup() ([]byte, error) {
	var huff *huffDecoder
	if b.compression == mobiCompressionHUFF {
		index, err := b.absolute(b.huffOffset)
		if err != nil {
			return nil, fmt.Errorf("locate HUFF table: %w", err)
		}
		if b.huffCount < 2 || uint64(index)+uint64(b.huffCount) > uint64(len(b.db.sections)) {
			return nil, fmt.Errorf("invalid HUFF/CDIC section count %d", b.huffCount)
		}
		huff = &huffDecoder{}
		if err := huff.loadHUFF(b.db.sections[index]); err != nil {
			return nil, err
		}
		for i := 1; i < int(b.huffCount); i++ {
			if err := huff.loadCDIC(b.db.sections[index+i]); err != nil {
				return nil, err
			}
		}
	}

	var out bytes.Buffer
	for i := 1; i <= b.textRecords; i++ {
		record, err := trimTrailingData(b.db.sections[b.start+i], b.trailerFlags)
		if err != nil {
			return nil, fmt.Errorf("trim text record %d: %w", i, err)
		}
		var decoded []byte
		switch b.compression {
		case mobiCompressionNone:
			decoded = append([]byte(nil), record...)
		case mobiCompressionPalmDOC:
			decoded, err = decompressPalmDOC(record)
		case mobiCompressionHUFF:
			decoded, err = huff.unpack(record, 0)
		default:
			return nil, fmt.Errorf("unsupported MOBI compression 0x%x", b.compression)
		}
		if err != nil {
			return nil, fmt.Errorf("decompress text record %d: %w", i, err)
		}
		out.Write(decoded)
	}
	return out.Bytes(), nil
}

func trimTrailingData(data []byte, flags uint16) ([]byte, error) {
	record := data
	trailers := 0
	shifted := flags
	for shifted > 1 {
		if shifted&2 != 0 {
			trailers++
		}
		shifted >>= 1
	}
	for i := 0; i < trailers; i++ {
		size, err := trailingEntrySize(record)
		if err != nil || size <= 0 || size > len(record) {
			return nil, fmt.Errorf("invalid trailing entry size")
		}
		record = record[:len(record)-size]
	}
	if flags&1 != 0 {
		if len(record) == 0 {
			return nil, fmt.Errorf("missing multibyte trailer")
		}
		size := int(record[len(record)-1]&3) + 1
		if size > len(record) {
			return nil, fmt.Errorf("invalid multibyte trailer size")
		}
		record = record[:len(record)-size]
	}
	return record, nil
}

func trailingEntrySize(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("empty record")
	}
	start := len(data) - 4
	if start < 0 {
		start = 0
	}
	value := 0
	for _, b := range data[start:] {
		if b&0x80 != 0 {
			value = 0
		}
		value = (value << 7) | int(b&0x7f)
	}
	return value, nil
}

func decompressPalmDOC(input []byte) ([]byte, error) {
	output := make([]byte, 0, len(input)*2)
	for position := 0; position < len(input); {
		code := input[position]
		position++
		switch {
		case code >= 1 && code <= 8:
			count := int(code)
			if position+count > len(input) {
				return nil, fmt.Errorf("truncated PalmDOC literal")
			}
			output = append(output, input[position:position+count]...)
			position += count
		case code < 0x80:
			output = append(output, code)
		case code >= 0xc0:
			output = append(output, ' ', code^0x80)
		default:
			if position >= len(input) {
				return nil, fmt.Errorf("truncated PalmDOC back-reference")
			}
			pair := uint16(code)<<8 | uint16(input[position])
			position++
			distance := int((pair >> 3) & 0x7ff)
			length := int(pair&7) + 3
			if distance == 0 || distance > len(output) {
				return nil, fmt.Errorf("invalid PalmDOC back-reference distance %d", distance)
			}
			for i := 0; i < length; i++ {
				output = append(output, output[len(output)-distance])
			}
		}
	}
	return output, nil
}

type huffCode struct {
	length   int
	terminal bool
	maxCode  uint32
}

type huffPhrase struct {
	data     []byte
	terminal bool
	expanded []byte
	busy     bool
}

type huffDecoder struct {
	lookup  [256]huffCode
	minCode [33]uint32
	maxCode [33]uint32
	phrases []huffPhrase
}

func (h *huffDecoder) loadHUFF(data []byte) error {
	if len(data) < 24 || string(data[:8]) != "HUFF\x00\x00\x00\x18" {
		return fmt.Errorf("invalid HUFF header")
	}
	offset1 := int(binary.BigEndian.Uint32(data[8:12]))
	offset2 := int(binary.BigEndian.Uint32(data[12:16]))
	if offset1 < 0 || offset1+256*4 > len(data) || offset2 < 0 || offset2+64*4 > len(data) {
		return fmt.Errorf("HUFF tables are truncated")
	}
	for i := 0; i < 256; i++ {
		value := binary.BigEndian.Uint32(data[offset1+i*4 : offset1+i*4+4])
		length := int(value & 0x1f)
		if length == 0 || length > 32 {
			return fmt.Errorf("invalid HUFF code length %d", length)
		}
		max := uint64(value >> 8)
		max = ((max + 1) << (32 - length)) - 1
		h.lookup[i] = huffCode{length: length, terminal: value&0x80 != 0, maxCode: uint32(max)}
	}
	for length := 1; length <= 32; length++ {
		min := binary.BigEndian.Uint32(data[offset2+(length-1)*8 : offset2+(length-1)*8+4])
		max := binary.BigEndian.Uint32(data[offset2+(length-1)*8+4 : offset2+length*8])
		h.minCode[length] = uint32(uint64(min) << (32 - length))
		h.maxCode[length] = uint32(((uint64(max) + 1) << (32 - length)) - 1)
	}
	return nil
}

func (h *huffDecoder) loadCDIC(data []byte) error {
	if len(data) < 16 || string(data[:8]) != "CDIC\x00\x00\x00\x10" {
		return fmt.Errorf("invalid CDIC header")
	}
	phraseCount := int(binary.BigEndian.Uint32(data[8:12]))
	bits := int(binary.BigEndian.Uint32(data[12:16]))
	if bits < 0 || bits > 16 || phraseCount < len(h.phrases) {
		return fmt.Errorf("invalid CDIC phrase table")
	}
	count := 1 << bits
	if remaining := phraseCount - len(h.phrases); count > remaining {
		count = remaining
	}
	if 16+count*2 > len(data) {
		return fmt.Errorf("truncated CDIC offsets")
	}
	for i := 0; i < count; i++ {
		offset := int(binary.BigEndian.Uint16(data[16+i*2 : 18+i*2]))
		if 18+offset > len(data) {
			return fmt.Errorf("CDIC phrase offset is outside the record")
		}
		lengthFlag := binary.BigEndian.Uint16(data[16+offset : 18+offset])
		length := int(lengthFlag & 0x7fff)
		start := 18 + offset
		if start+length > len(data) {
			return fmt.Errorf("truncated CDIC phrase")
		}
		h.phrases = append(h.phrases, huffPhrase{
			data: append([]byte(nil), data[start:start+length]...), terminal: lengthFlag&0x8000 != 0,
		})
	}
	return nil
}

func (h *huffDecoder) unpack(input []byte, depth int) ([]byte, error) {
	if depth > 64 {
		return nil, fmt.Errorf("HUFF phrase recursion limit exceeded")
	}
	padded := make([]byte, len(input)+8)
	copy(padded, input)
	bitsLeft := len(input) * 8
	position := 0
	remaining := 32
	window := binary.BigEndian.Uint64(padded[:8])
	var output bytes.Buffer
	for {
		if remaining <= 0 {
			position += 4
			if position+8 > len(padded) {
				break
			}
			window = binary.BigEndian.Uint64(padded[position : position+8])
			remaining += 32
		}
		code := uint32(window >> remaining)
		entry := h.lookup[code>>24]
		length := entry.length
		maxCode := entry.maxCode
		if !entry.terminal {
			for length <= 32 && code < h.minCode[length] {
				length++
			}
			if length > 32 {
				return nil, fmt.Errorf("invalid HUFF code")
			}
			maxCode = h.maxCode[length]
		}
		remaining -= length
		bitsLeft -= length
		if bitsLeft < 0 {
			break
		}
		phraseIndex := uint64(maxCode-code) >> (32 - length)
		if phraseIndex >= uint64(len(h.phrases)) {
			return nil, fmt.Errorf("HUFF phrase index %d is outside the dictionary", phraseIndex)
		}
		phrase := &h.phrases[phraseIndex]
		if phrase.terminal {
			output.Write(phrase.data)
			continue
		}
		if phrase.busy {
			return nil, fmt.Errorf("recursive HUFF dictionary cycle")
		}
		if phrase.expanded == nil {
			phrase.busy = true
			expanded, err := h.unpack(phrase.data, depth+1)
			phrase.busy = false
			if err != nil {
				return nil, err
			}
			phrase.expanded = expanded
		}
		output.Write(phrase.expanded)
	}
	return output.Bytes(), nil
}

func (b *mobiBook) assembleKF8(raw []byte) ([]byte, error) {
	flow := raw
	if b.fdst != missingSection {
		index, err := b.absolute(b.fdst)
		if err != nil {
			return nil, fmt.Errorf("locate FDST: %w", err)
		}
		section := b.db.sections[index]
		if len(section) < 12 || string(section[:4]) != "FDST" {
			return nil, fmt.Errorf("invalid FDST record")
		}
		count := int(binary.BigEndian.Uint32(section[8:12]))
		if count <= 0 || 12+count*8 > len(section) {
			return nil, fmt.Errorf("invalid FDST flow count %d", count)
		}
		start := int(binary.BigEndian.Uint32(section[12:16]))
		end := len(raw)
		if count > 1 {
			end = int(binary.BigEndian.Uint32(section[20:24]))
		}
		if start < 0 || end < start || end > len(raw) {
			return nil, fmt.Errorf("FDST main flow range is invalid")
		}
		flow = raw[start:end]
	}
	if b.skeleton == missingSection || b.fragment == missingSection {
		return append([]byte(nil), flow...), nil
	}
	skeletonIndex, err := b.absolute(b.skeleton)
	if err != nil {
		return nil, fmt.Errorf("locate skeleton index: %w", err)
	}
	fragmentIndex, err := b.absolute(b.fragment)
	if err != nil {
		return nil, fmt.Errorf("locate fragment index: %w", err)
	}
	skeletonRows, err := parseMobiIndex(b.db.sections, skeletonIndex)
	if err != nil {
		return nil, fmt.Errorf("parse skeleton index: %w", err)
	}
	fragmentRows, err := parseMobiIndex(b.db.sections, fragmentIndex)
	if err != nil {
		return nil, fmt.Errorf("parse fragment index: %w", err)
	}
	skeletons := make([]skeletonEntry, 0, len(skeletonRows))
	for _, row := range skeletonRows {
		one := row.tags[1]
		six := row.tags[6]
		if len(one) < 1 || len(six) < 2 {
			return nil, fmt.Errorf("skeleton index entry is missing required tags")
		}
		skeletons = append(skeletons, skeletonEntry{fragmentCount: int(one[0]), position: int(six[0]), length: int(six[1])})
	}
	fragments := make([]fragmentEntry, 0, len(fragmentRows))
	for _, row := range fragmentRows {
		position, err := strconv.ParseUint(row.text, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid fragment insert position %q", row.text)
		}
		six := row.tags[6]
		if len(six) < 2 {
			return nil, fmt.Errorf("fragment index entry is missing tag 6")
		}
		fragments = append(fragments, fragmentEntry{insertPosition: int(position), position: int(six[0]), length: int(six[1])})
	}
	var assembled bytes.Buffer
	fragmentPointer := 0
	for _, skeleton := range skeletons {
		base := skeleton.position + skeleton.length
		if skeleton.position < 0 || base < skeleton.position || base > len(flow) {
			return nil, fmt.Errorf("skeleton range is outside the main flow")
		}
		part := append([]byte(nil), flow[skeleton.position:base]...)
		for i := 0; i < skeleton.fragmentCount; i++ {
			if fragmentPointer >= len(fragments) {
				return nil, fmt.Errorf("skeleton references missing fragments")
			}
			fragment := fragments[fragmentPointer]
			fragmentPointer++
			end := base + fragment.length
			if fragment.length < 0 || end < base || end > len(flow) {
				return nil, fmt.Errorf("fragment range is outside the main flow")
			}
			insert := fragment.insertPosition - skeleton.position
			if insert < 0 || insert > len(part) {
				return nil, fmt.Errorf("fragment insert position is outside its skeleton")
			}
			withFragment := make([]byte, 0, len(part)+fragment.length)
			withFragment = append(withFragment, part[:insert]...)
			withFragment = append(withFragment, flow[base:end]...)
			withFragment = append(withFragment, part[insert:]...)
			part = withFragment
			base = end
		}
		assembled.Write(part)
	}
	if assembled.Len() == 0 {
		return nil, fmt.Errorf("KF8 skeleton index produced no text")
	}
	return assembled.Bytes(), nil
}

func parseMobiIndex(sections [][]byte, index int) ([]indexEntry, error) {
	if index < 0 || index >= len(sections) {
		return nil, fmt.Errorf("INDX record is missing")
	}
	main := sections[index]
	headerLength, recordCount, _, err := parseINDXHeader(main)
	if err != nil {
		return nil, err
	}
	controlBytes, tags, err := parseTAGX(main, headerLength)
	if err != nil {
		return nil, err
	}
	if recordCount < 0 || index+recordCount >= len(sections) {
		return nil, fmt.Errorf("INDX record count exceeds the file")
	}
	entries := make([]indexEntry, 0)
	for record := 1; record <= recordCount; record++ {
		data := sections[index+record]
		_, entryCount, idxt, err := parseINDXHeader(data)
		if err != nil {
			return nil, err
		}
		if idxt < 0 || idxt+4+entryCount*2 > len(data) || string(data[idxt:idxt+4]) != "IDXT" {
			return nil, fmt.Errorf("invalid IDXT table")
		}
		positions := make([]int, entryCount+1)
		for i := 0; i < entryCount; i++ {
			positions[i] = int(binary.BigEndian.Uint16(data[idxt+4+i*2 : idxt+6+i*2]))
		}
		positions[entryCount] = idxt
		for i := 0; i < entryCount; i++ {
			start, end := positions[i], positions[i+1]
			if start < 0 || start >= end || end > len(data) {
				return nil, fmt.Errorf("invalid INDX entry range")
			}
			textLength := int(data[start])
			valueStart := start + 1 + textLength
			if valueStart > end {
				return nil, fmt.Errorf("truncated INDX entry text")
			}
			tagMap, err := decodeIndexTags(data, valueStart, end, controlBytes, tags)
			if err != nil {
				return nil, err
			}
			entries = append(entries, indexEntry{text: string(data[start+1 : start+1+textLength]), tags: tagMap})
		}
	}
	return entries, nil
}

func parseINDXHeader(data []byte) (headerLength, count, idxt int, err error) {
	if len(data) < 56 || string(data[:4]) != "INDX" {
		return 0, 0, 0, fmt.Errorf("invalid INDX header")
	}
	headerLength = int(binary.BigEndian.Uint32(data[4:8]))
	idxt = int(binary.BigEndian.Uint32(data[20:24]))
	count = int(binary.BigEndian.Uint32(data[24:28]))
	if headerLength < 0 || headerLength > len(data) || count < 0 || count > 1_000_000 {
		return 0, 0, 0, fmt.Errorf("invalid INDX header values")
	}
	return headerLength, count, idxt, nil
}

func parseTAGX(data []byte, start int) (int, []indexTag, error) {
	if start < 0 || start+12 > len(data) || string(data[start:start+4]) != "TAGX" {
		return 0, nil, fmt.Errorf("missing TAGX table")
	}
	length := int(binary.BigEndian.Uint32(data[start+4 : start+8]))
	controlBytes := int(binary.BigEndian.Uint32(data[start+8 : start+12]))
	if length < 12 || start+length > len(data) || controlBytes <= 0 || controlBytes > 32 {
		return 0, nil, fmt.Errorf("invalid TAGX header")
	}
	tags := make([]indexTag, 0, (length-12)/4)
	for position := start + 12; position+4 <= start+length; position += 4 {
		tags = append(tags, indexTag{data[position], data[position+1], data[position+2], data[position+3]})
	}
	return controlBytes, tags, nil
}

func decodeIndexTags(data []byte, start, end, controlByteCount int, table []indexTag) (map[byte][]uint64, error) {
	if start < 0 || start+controlByteCount > end || end > len(data) {
		return nil, fmt.Errorf("truncated INDX control bytes")
	}
	type pendingTag struct {
		tag            byte
		count          int
		byteLength     int
		valuesPerEntry int
	}
	pending := make([]pendingTag, 0)
	controlIndex := 0
	dataPosition := start + controlByteCount
	for _, tag := range table {
		if tag.endFlag == 1 {
			controlIndex++
			continue
		}
		if controlIndex >= controlByteCount {
			return nil, fmt.Errorf("TAGX control byte index overflow")
		}
		masked := data[start+controlIndex] & tag.mask
		if masked == 0 {
			continue
		}
		item := pendingTag{tag: tag.tag, valuesPerEntry: int(tag.valuesPerEntry)}
		if masked == tag.mask {
			if bitCount(tag.mask) > 1 {
				value, consumed, err := variableWidth(data, dataPosition, end)
				if err != nil {
					return nil, err
				}
				dataPosition += consumed
				item.byteLength = int(value)
			} else {
				item.count = 1
			}
		} else {
			value, mask := masked, tag.mask
			for mask&1 == 0 {
				mask >>= 1
				value >>= 1
			}
			item.count = int(value)
		}
		pending = append(pending, item)
	}
	result := make(map[byte][]uint64)
	for _, item := range pending {
		values := make([]uint64, 0)
		if item.byteLength > 0 {
			limit := dataPosition + item.byteLength
			if limit > end {
				return nil, fmt.Errorf("INDX tag values exceed the entry")
			}
			for dataPosition < limit {
				value, consumed, err := variableWidth(data, dataPosition, limit)
				if err != nil {
					return nil, err
				}
				dataPosition += consumed
				values = append(values, value)
			}
		} else {
			count := item.count * item.valuesPerEntry
			for i := 0; i < count; i++ {
				value, consumed, err := variableWidth(data, dataPosition, end)
				if err != nil {
					return nil, err
				}
				dataPosition += consumed
				values = append(values, value)
			}
		}
		result[item.tag] = values
	}
	return result, nil
}

func variableWidth(data []byte, offset, limit int) (uint64, int, error) {
	var value uint64
	for consumed := 0; consumed < 10; consumed++ {
		position := offset + consumed
		if position >= limit || position >= len(data) {
			return 0, 0, fmt.Errorf("truncated variable-width integer")
		}
		b := data[position]
		if value > (^uint64(0) >> 7) {
			return 0, 0, fmt.Errorf("variable-width integer overflow")
		}
		value = (value << 7) | uint64(b&0x7f)
		if b&0x80 != 0 {
			return value, consumed + 1, nil
		}
	}
	return 0, 0, fmt.Errorf("variable-width integer is too long")
}

func bitCount(value byte) int {
	count := 0
	for value != 0 {
		count += int(value & 1)
		value >>= 1
	}
	return count
}
