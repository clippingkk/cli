package sdr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/amazon-ion/ion-go/ion"
)

const (
	kfxContainerHeaderSize = 18
	kfxEntityHeaderSize    = 10
	maxKFXContainerSize    = 64 << 20
	maxKFXEntities         = 100_000
	maxYJSymbolID          = 2048
)

type kfxEntity struct {
	id       string
	typeName string
	value    any
}

type kfxSection struct {
	position int64
	text     string
}

type kfxPage struct {
	position int64
	label    string
}

type kfxPositionChunk struct {
	position int64
	eid      int64
	offset   int64
	length   int64
}

type kfxBook struct {
	title    string
	sections []kfxSection
	pages    []kfxPage
}

// assembleKFX decodes the text and navigation fragments needed to map Kindle
// annotation positions. It deliberately does not attempt to convert the book
// to another publication format.
func assembleKFX(data []byte) (kfxBook, error) {
	entities, err := decodeKFXContainer(data)
	if err != nil {
		return kfxBook{}, err
	}
	book := kfxBook{}
	textPools := make(map[string][]string)
	sectionStarts := make(map[string]int64)
	var positionEntities []kfxEntity
	var contentEntities []kfxEntity
	var navigationEntities []kfxEntity
	for _, entity := range entities {
		switch entity.typeName {
		case "$145":
			if pool := kfxStringList(kfxMap(entity.value)["$146"]); len(pool) > 0 {
				textPools[entity.id] = pool
			}
		case "$259", "$260":
			contentEntities = append(contentEntities, entity)
		case "$265":
			collectKFXSectionStarts(entity.value, sectionStarts)
		case "$609":
			positionEntities = append(positionEntities, entity)
		case "$258", "$490", "$538":
			if book.title == "" {
				book.title = findKFXTitle(entity.value)
			}
		case "$391":
			navigationEntities = append(navigationEntities, entity)
		}
	}
	chunksByEID := make(map[int64][]kfxPositionChunk)
	for _, entity := range positionEntities {
		sectionName := kfxString(kfxMap(entity.value)["$174"])
		start, ok := sectionStarts[sectionName]
		if !ok {
			continue
		}
		chunks, err := decodeKFXPositionMap(kfxMap(entity.value)["$181"], start)
		if err != nil {
			return kfxBook{}, fmt.Errorf("decode KFX position map %s: %w", sectionName, err)
		}
		for _, chunk := range chunks {
			chunksByEID[chunk.eid] = append(chunksByEID[chunk.eid], chunk)
		}
	}
	for eid := range chunksByEID {
		sort.Slice(chunksByEID[eid], func(i, j int) bool {
			return chunksByEID[eid][i].offset < chunksByEID[eid][j].offset
		})
	}
	textOffsets := make(map[int64]int64)
	for _, entity := range contentEntities {
		collectKFXContent(entity.value, textPools, chunksByEID, textOffsets, &book.sections)
	}
	for _, entity := range navigationEntities {
		collectKFXPages(entity.value, chunksByEID, &book.pages)
	}
	if len(book.sections) == 0 {
		return kfxBook{}, fmt.Errorf("KFX container contains no readable text fragments")
	}
	sort.Slice(book.sections, func(i, j int) bool { return book.sections[i].position < book.sections[j].position })
	sort.Slice(book.pages, func(i, j int) bool { return book.pages[i].position < book.pages[j].position })
	return book, nil
}

func decodeKFXContainer(data []byte) ([]kfxEntity, error) {
	if len(data) < kfxContainerHeaderSize || string(data[:4]) != "CONT" {
		return nil, fmt.Errorf("invalid KFX CONT signature")
	}
	if len(data) > maxKFXContainerSize {
		return nil, fmt.Errorf("KFX container exceeds %d byte limit", maxKFXContainerSize)
	}
	version := binary.LittleEndian.Uint16(data[4:6])
	if version != 1 && version != 2 {
		return nil, fmt.Errorf("unsupported KFX container version %d", version)
	}
	headerLength := uint64(binary.LittleEndian.Uint32(data[6:10]))
	infoOffset := uint64(binary.LittleEndian.Uint32(data[10:14]))
	infoLength := uint64(binary.LittleEndian.Uint32(data[14:18]))
	if headerLength < kfxContainerHeaderSize || headerLength > uint64(len(data)) {
		return nil, fmt.Errorf("invalid KFX header length %d", headerLength)
	}
	infoData, err := boundedKFXSlice(data, infoOffset, infoLength, "container info")
	if err != nil {
		return nil, err
	}
	infoValue, err := decodeKFXIon(infoData, nil)
	if err != nil {
		return nil, fmt.Errorf("decode KFX container info: %w", err)
	}
	info, ok := infoValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("KFX container info is not a struct")
	}
	if compression, _ := kfxInt(info["$410"]); compression != 0 {
		return nil, fmt.Errorf("unsupported KFX container compression %d", compression)
	}
	if drm, _ := kfxInt(info["$411"]); drm != 0 {
		return nil, fmt.Errorf("DRM-protected KFX containers are unsupported")
	}
	indexOffset, indexOK := kfxUint(info["$413"])
	indexLength, lengthOK := kfxUint(info["$414"])
	if !indexOK || !lengthOK || indexLength%24 != 0 {
		return nil, fmt.Errorf("invalid KFX entity index")
	}
	if indexLength/24 > maxKFXEntities {
		return nil, fmt.Errorf("KFX entity index exceeds entry limit")
	}
	indexData, err := boundedKFXSlice(data, indexOffset, indexLength, "entity index")
	if err != nil {
		return nil, err
	}

	var documentSymbols []byte
	if symbolLength, ok := kfxUint(info["$416"]); ok && symbolLength > 0 {
		symbolOffset, offsetOK := kfxUint(info["$415"])
		if !offsetOK {
			return nil, fmt.Errorf("KFX document symbol table has no offset")
		}
		documentSymbols, err = boundedKFXSlice(data, symbolOffset, symbolLength, "document symbol table")
		if err != nil {
			return nil, err
		}
	}
	symbols, err := kfxSymbolTable(documentSymbols)
	if err != nil {
		return nil, fmt.Errorf("decode KFX document symbols: %w", err)
	}

	result := make([]kfxEntity, 0, indexLength/24)
	for offset := 0; offset < len(indexData); offset += 24 {
		idSID := uint64(binary.LittleEndian.Uint32(indexData[offset : offset+4]))
		typeSID := uint64(binary.LittleEndian.Uint32(indexData[offset+4 : offset+8]))
		entityOffset := binary.LittleEndian.Uint64(indexData[offset+8 : offset+16])
		entityLength := binary.LittleEndian.Uint64(indexData[offset+16 : offset+24])
		serialized, err := boundedKFXSlice(data, headerLength+entityOffset, entityLength, "entity")
		if err != nil {
			return nil, fmt.Errorf("KFX entity %d: %w", offset/24, err)
		}
		if len(serialized) < kfxEntityHeaderSize || string(serialized[:4]) != "ENTY" {
			return nil, fmt.Errorf("KFX entity %d has invalid ENTY signature", offset/24)
		}
		if entityVersion := binary.LittleEndian.Uint16(serialized[4:6]); entityVersion != 1 {
			return nil, fmt.Errorf("KFX entity %d has unsupported version %d", offset/24, entityVersion)
		}
		entityHeaderLength := uint64(binary.LittleEndian.Uint32(serialized[6:10]))
		if entityHeaderLength < kfxEntityHeaderSize || entityHeaderLength > uint64(len(serialized)) {
			return nil, fmt.Errorf("KFX entity %d has invalid header length", offset/24)
		}
		entityInfoData := serialized[kfxEntityHeaderSize:entityHeaderLength]
		if len(entityInfoData) > 0 {
			entityInfoValue, err := decodeKFXIon(entityInfoData, documentSymbols)
			if err != nil {
				return nil, fmt.Errorf("decode KFX entity %d info: %w", offset/24, err)
			}
			if entityInfo, ok := entityInfoValue.(map[string]any); ok {
				if compression, _ := kfxInt(entityInfo["$410"]); compression != 0 {
					return nil, fmt.Errorf("KFX entity %d uses unsupported compression %d", offset/24, compression)
				}
				if drm, _ := kfxInt(entityInfo["$411"]); drm != 0 {
					return nil, fmt.Errorf("KFX entity %d is DRM-protected", offset/24)
				}
			}
		}
		typeName := kfxSymbolByID(symbols, typeSID)
		var value any
		if typeName == "$417" || typeName == "$418" {
			value = append([]byte(nil), serialized[entityHeaderLength:]...)
		} else {
			value, err = decodeKFXIon(serialized[entityHeaderLength:], documentSymbols)
			if err != nil {
				return nil, fmt.Errorf("decode KFX entity %d payload: %w", offset/24, err)
			}
		}
		result = append(result, kfxEntity{
			id:       kfxSymbolByID(symbols, idSID),
			typeName: typeName,
			value:    value,
		})
	}
	return result, nil
}

func boundedKFXSlice(data []byte, offset, length uint64, label string) ([]byte, error) {
	if offset > uint64(len(data)) || length > uint64(len(data))-offset {
		return nil, fmt.Errorf("%s range is outside the KFX container", label)
	}
	return data[offset : offset+length], nil
}

func yjSharedSymbols() ion.SharedSymbolTable {
	symbols := make([]string, maxYJSymbolID-9)
	for sid := 10; sid <= maxYJSymbolID; sid++ {
		symbols[sid-10] = fmt.Sprintf("$%d", sid)
	}
	return ion.NewSharedSymbolTable("YJ_symbols", 10, symbols)
}

func kfxIonPrelude() ([]byte, error) {
	var output bytes.Buffer
	w := ion.NewBinaryWriter(&output, yjSharedSymbols())
	if err := w.WriteInt(0); err != nil {
		return nil, err
	}
	if err := w.Finish(); err != nil {
		return nil, err
	}
	encoded := output.Bytes()
	if len(encoded) == 0 || encoded[len(encoded)-1] != 0x20 {
		return nil, fmt.Errorf("create KFX Ion symbol prelude")
	}
	return append([]byte(nil), encoded[:len(encoded)-1]...), nil
}

func decodeKFXIon(data, documentSymbols []byte) (any, error) {
	var stream []byte
	if len(documentSymbols) > 0 {
		stream = append(stream, 0xe0, 0x01, 0x00, 0xea)
		stream = append(stream, stripIonVersion(documentSymbols)...)
	} else {
		prelude, err := kfxIonPrelude()
		if err != nil {
			return nil, err
		}
		stream = append(stream, prelude...)
	}
	stream = append(stream, stripIonVersion(data)...)
	reader := ion.NewReaderCat(bytes.NewReader(stream), ion.NewCatalog(yjSharedSymbols()))
	return ion.NewDecoder(reader).Decode()
}

func stripIonVersion(data []byte) []byte {
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{0xe0, 0x01, 0x00, 0xea}) {
		return data[4:]
	}
	return data
}

func kfxSymbolTable(documentSymbols []byte) (ion.SymbolTable, error) {
	var stream []byte
	if len(documentSymbols) > 0 {
		stream = append(stream, 0xe0, 0x01, 0x00, 0xea)
		stream = append(stream, stripIonVersion(documentSymbols)...)
	} else {
		prelude, err := kfxIonPrelude()
		if err != nil {
			return nil, err
		}
		stream = append(stream, prelude...)
	}
	stream = append(stream, 0x20)
	reader := ion.NewReaderCat(bytes.NewReader(stream), ion.NewCatalog(yjSharedSymbols()))
	if !reader.Next() {
		if reader.Err() != nil {
			return nil, reader.Err()
		}
		return nil, fmt.Errorf("decode KFX symbol table")
	}
	return reader.SymbolTable(), nil
}

func kfxSymbolByID(table ion.SymbolTable, sid uint64) string {
	if table != nil {
		if symbol, ok := table.FindByID(sid); ok {
			return symbol
		}
	}
	return fmt.Sprintf("$%d", sid)
}

func kfxUint(value any) (uint64, bool) {
	n, ok := kfxInt(value)
	if !ok || n < 0 {
		return 0, false
	}
	return uint64(n), true
}

func kfxInt(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case uint64:
		if v <= uint64(^uint64(0)>>1) {
			return int64(v), true
		}
	}
	return 0, false
}

func kfxString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case *string:
		if v != nil {
			return *v
		}
	case *ion.SymbolToken:
		if v != nil && v.Text != nil {
			return *v.Text
		}
	}
	return ""
}

func kfxMap(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func kfxList(value any) []any {
	result, _ := value.([]any)
	return result
}

func kfxStringList(value any) []string {
	values := kfxList(value)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text := kfxString(value); text != "" {
			result = append(result, text)
		} else {
			result = append(result, "")
		}
	}
	return result
}

func findKFXTitle(value any) string {
	var title string
	walkKFX(value, func(field string, child any) {
		if title != "" {
			return
		}
		if field == "$153" || strings.EqualFold(field, "title") {
			title = kfxString(child)
			return
		}
		if object := kfxMap(child); kfxString(object["$492"]) == "title" {
			title = kfxString(object["$307"])
		}
	})
	return title
}

func collectKFXSectionStarts(value any, starts map[string]int64) {
	for _, sectionValue := range kfxList(kfxMap(value)["$181"]) {
		section := kfxMap(sectionValue)
		name := kfxString(section["$174"])
		start, ok := kfxInt(section["$184"])
		if name != "" && ok && start >= 0 {
			starts[name] = start
		}
	}
}

func decodeKFXPositionMap(value any, sectionStart int64) ([]kfxPositionChunk, error) {
	entries := kfxList(value)
	var result []kfxPositionChunk
	var position, eid, eidOffset int64
	for index, entry := range entries {
		var nextPosition, nextEID, nextOffset int64
		if values := kfxList(entry); len(values) >= 2 && len(values) <= 3 {
			positionDelta, positionOK := kfxInt(values[0])
			eidValue, eidOK := kfxInt(values[1])
			if !positionOK || !eidOK || positionDelta < 0 || eidValue < 0 {
				return nil, fmt.Errorf("invalid list entry %d", index)
			}
			nextPosition = position + positionDelta
			nextEID = eidValue
			if len(values) == 3 {
				var ok bool
				nextOffset, ok = kfxInt(values[2])
				if !ok || nextOffset < 0 {
					return nil, fmt.Errorf("invalid offset in entry %d", index)
				}
			}
		} else if delta, ok := kfxInt(entry); ok && delta >= 0 {
			nextPosition = position + delta
			nextEID = eid + 1
		} else {
			return nil, fmt.Errorf("invalid entry %d", index)
		}
		if index > 0 {
			length := nextPosition - position
			if length < 0 {
				return nil, fmt.Errorf("position moved backwards at entry %d", index)
			}
			result = append(result, kfxPositionChunk{
				position: sectionStart + position,
				eid:      eid,
				offset:   eidOffset,
				length:   length,
			})
		}
		position, eid, eidOffset = nextPosition, nextEID, nextOffset
	}
	return result, nil
}

func collectKFXContent(value any, pools map[string][]string, chunks map[int64][]kfxPositionChunk,
	offsets map[int64]int64, sections *[]kfxSection,
) {
	switch current := value.(type) {
	case map[string]any:
		if eid, ok := kfxInt(current["$155"]); ok {
			if reference := kfxMap(current["$145"]); reference != nil {
				poolName := kfxString(reference["name"])
				index, indexOK := kfxInt(reference["$403"])
				pool := pools[poolName]
				if indexOK && index >= 0 && index < int64(len(pool)) {
					text := pool[index]
					offset := offsets[eid]
					if position, found := kfxPositionForEID(chunks, eid, offset); found {
						*sections = append(*sections, kfxSection{position: position, text: text})
					}
					offsets[eid] += int64(utf8.RuneCountInString(text))
				}
			}
		}
		for _, child := range current {
			collectKFXContent(child, pools, chunks, offsets, sections)
		}
	case []any:
		for _, child := range current {
			collectKFXContent(child, pools, chunks, offsets, sections)
		}
	}
}

func kfxPositionForEID(chunks map[int64][]kfxPositionChunk, eid, offset int64) (int64, bool) {
	for _, chunk := range chunks[eid] {
		if offset >= chunk.offset && offset < chunk.offset+chunk.length {
			return chunk.position + offset - chunk.offset, true
		}
	}
	return 0, false
}

func collectKFXPages(value any, chunks map[int64][]kfxPositionChunk, pages *[]kfxPage) {
	if kfxString(kfxMap(value)["$235"]) != "$237" {
		return
	}
	walkKFX(value, func(_ string, child any) {
		entry := kfxMap(child)
		labelObject := kfxMap(entry["$241"])
		positionObject := kfxMap(entry["$246"])
		label := kfxString(labelObject["$244"])
		eid, eidOK := kfxInt(positionObject["$155"])
		offset, offsetOK := kfxInt(positionObject["$143"])
		if !offsetOK {
			offset = 0
		}
		if label == "" || !eidOK {
			return
		}
		if position, found := kfxPositionForEID(chunks, eid, offset); found {
			*pages = append(*pages, kfxPage{position: position, label: label})
		}
	})
}

func walkKFX(value any, visit func(field string, value any)) {
	switch v := value.(type) {
	case map[string]any:
		visit("", v)
		for field, child := range v {
			visit(field, child)
			walkKFX(child, visit)
		}
	case []any:
		visit("", v)
		for _, child := range v {
			walkKFX(child, visit)
		}
	}
}

func (book kfxBook) textAt(start, end int64) (string, bool) {
	if start > end {
		start, end = end, start
	}
	var output strings.Builder
	covered := false
	for i, section := range book.sections {
		sectionRunes := []rune(section.text)
		sectionEnd := section.position + int64(len(sectionRunes))
		if i+1 < len(book.sections) && book.sections[i+1].position < sectionEnd {
			sectionEnd = book.sections[i+1].position
		}
		if sectionEnd <= start || section.position > end {
			continue
		}
		left := max(start, section.position) - section.position
		right := min(end+1, sectionEnd) - section.position
		if left >= 0 && right > left && right <= int64(len(sectionRunes)) {
			output.WriteString(string(sectionRunes[left:right]))
			covered = true
		}
	}
	text := strings.TrimSpace(strings.Join(strings.Fields(output.String()), " "))
	return text, covered && utf8.ValidString(text)
}

func (book kfxBook) pageAt(position int64) string {
	index := sort.Search(len(book.pages), func(i int) bool { return book.pages[i].position > position }) - 1
	if index >= 0 && book.pages[index].label != "" {
		return "#" + book.pages[index].label
	}
	return "#" + itoa64(position)
}
