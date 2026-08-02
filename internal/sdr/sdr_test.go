package sdr

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/amazon-ion/ion-go/ion"
)

func TestDecodeSidecarAndRecoverText(t *testing.T) {
	created := time.Date(2026, 5, 7, 13, 59, 53, 233_000_000, time.UTC)
	data := buildKRDSFixture(t, []fixtureAnnotation{
		{kind: 1, start: "3", end: "8", created: created},
		{kind: 2, start: "9", end: "14", created: created.Add(time.Minute), note: "remember this"},
		{kind: 13, start: "15", end: "20", created: created.Add(2 * time.Minute)},
	}, []int64{0, 9, 20})

	sidecar, err := DecodeSidecar(data)
	if err != nil {
		t.Fatalf("DecodeSidecar() error = %v", err)
	}
	if len(sidecar.Annotations) != 3 {
		t.Fatalf("got %d annotations, want 3", len(sidecar.Annotations))
	}
	if sidecar.Annotations[1].Note != "remember this" {
		t.Fatalf("note = %q", sidecar.Annotations[1].Note)
	}
	if got := pageAt(sidecar.PageMap, 12); got != "#1" {
		t.Fatalf("pageAt() = %q, want #1", got)
	}
	if got := pageAt(PageMap{}, 12); got != "#12" {
		t.Fatalf("pageAt() fallback = %q, want #12", got)
	}

	markup := []byte(`<p>one &amp; two three</p>`)
	start := int64(bytes.Index(markup, []byte("one")) + 1)
	end := int64(bytes.Index(markup, []byte("two")) + 2)
	readable, exact := recoverText(markup, start, end)
	if readable != "one & two" {
		t.Fatalf("recoverText() readable = %q", readable)
	}
	if exact == readable || exact == "" {
		t.Fatalf("recoverText() exact = %q, want an unsnapped non-empty slice", exact)
	}
}

func TestDecodeSidecarRejectsTruncatedData(t *testing.T) {
	if _, err := DecodeSidecar(append([]byte(nil), krdsSignature...)); err == nil {
		t.Fatal("DecodeSidecar() accepted truncated data")
	}
}

func TestPalmDOCDecompression(t *testing.T) {
	// "hello hello": literal "hello", encoded space+letter, then a back-reference.
	input := []byte{5, 'h', 'e', 'l', 'l', 'o', 0xe8, 0x80, 0x31}
	got, err := decompressPalmDOC(input)
	if err != nil {
		t.Fatalf("decompressPalmDOC() error = %v", err)
	}
	if string(got) != "hello hello" {
		t.Fatalf("decompressPalmDOC() = %q", got)
	}
}

func TestHUFFDecompression(t *testing.T) {
	huff := make([]byte, 24+256*4+64*4)
	copy(huff[:8], []byte("HUFF\x00\x00\x00\x18"))
	binary.BigEndian.PutUint32(huff[8:12], 24)
	binary.BigEndian.PutUint32(huff[12:16], uint32(24+256*4))
	for i := 0; i < 256; i++ {
		binary.BigEndian.PutUint32(huff[24+i*4:28+i*4], 0x181)
	}
	cdic := make([]byte, 26)
	copy(cdic[:8], []byte("CDIC\x00\x00\x00\x10"))
	binary.BigEndian.PutUint32(cdic[8:12], 2)
	binary.BigEndian.PutUint32(cdic[12:16], 1)
	binary.BigEndian.PutUint16(cdic[16:18], 4)
	binary.BigEndian.PutUint16(cdic[18:20], 7)
	binary.BigEndian.PutUint16(cdic[20:22], 0x8001)
	cdic[22] = 'A'
	binary.BigEndian.PutUint16(cdic[23:25], 0x8001)
	cdic[25] = 'B'

	decoder := &huffDecoder{}
	if err := decoder.loadHUFF(huff); err != nil {
		t.Fatalf("loadHUFF() error = %v", err)
	}
	if err := decoder.loadCDIC(cdic); err != nil {
		t.Fatalf("loadCDIC() error = %v", err)
	}
	got, err := decoder.unpack([]byte{0xff}, 0)
	if err != nil {
		t.Fatalf("unpack() error = %v", err)
	}
	if string(got) != "AAAAAAAA" {
		t.Fatalf("HUFF unpack = %q", got)
	}
}

func TestKF8IndexAssembly(t *testing.T) {
	flow := []byte("<p></p>hello")
	skeletonMain, skeletonExtra := buildIndexFixture(t, []indexTag{
		{tag: 1, valuesPerEntry: 1, mask: 1},
		{tag: 6, valuesPerEntry: 2, mask: 2},
	}, "s", 3, []uint64{1, 0, 7})
	fragmentMain, fragmentExtra := buildIndexFixture(t, []indexTag{
		{tag: 6, valuesPerEntry: 2, mask: 1},
	}, "3", 1, []uint64{7, 5})
	db := &palmDatabase{sections: [][]byte{flow, skeletonMain, skeletonExtra, fragmentMain, fragmentExtra}}
	book := &mobiBook{db: db, skeleton: 1, fragment: 3, fdst: missingSection}
	got, err := book.assembleKF8(flow)
	if err != nil {
		t.Fatalf("assembleKF8() error = %v", err)
	}
	if string(got) != "<p>hello</p>" {
		t.Fatalf("assembleKF8() = %q", got)
	}
}

func TestAssembleBookRejectsDRMAndMobi7(t *testing.T) {
	markup := []byte("<p>text</p>")
	drm := buildKF8Fixture(t, "DRM", markup)
	headerOffset := int(binary.BigEndian.Uint32(drm[78:82]))
	binary.BigEndian.PutUint16(drm[headerOffset+12:headerOffset+14], 1)
	if _, _, err := AssembleBook(drm); err == nil || !strings.Contains(err.Error(), "DRM") {
		t.Fatalf("AssembleBook(DRM) error = %v", err)
	}

	mobi7 := buildKF8Fixture(t, "Old", markup)
	headerOffset = int(binary.BigEndian.Uint32(mobi7[78:82]))
	binary.BigEndian.PutUint32(mobi7[headerOffset+36:headerOffset+40], 6)
	if _, _, err := AssembleBook(mobi7); err == nil || !strings.Contains(err.Error(), "Mobi7") {
		t.Fatalf("AssembleBook(Mobi7) error = %v", err)
	}
}

func TestExtractPathEndToEnd(t *testing.T) {
	dir := t.TempDir()
	bookPath := filepath.Join(dir, "Example.azw3")
	sidecarDir := filepath.Join(dir, "Example.sdr")
	if err := os.Mkdir(sidecarDir, 0o755); err != nil {
		t.Fatal(err)
	}
	markup := []byte(`<p>alpha beta gamma</p>`)
	if err := os.WriteFile(bookPath, buildKF8Fixture(t, "Example Title", markup), 0o600); err != nil {
		t.Fatal(err)
	}
	start := int64(bytes.Index(markup, []byte("beta")))
	created := time.Date(2026, 7, 1, 8, 30, 0, 0, time.UTC)
	sidecar := buildKRDSFixture(t, []fixtureAnnotation{{kind: 1, start: itoa64(start), end: itoa64(start + 4), created: created}}, []int64{0, start})
	if err := os.WriteFile(filepath.Join(sidecarDir, "annotations.azw3r"), sidecar, 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := ExtractPath(dir)
	if err != nil {
		t.Fatalf("ExtractPath() error = %v; warnings = %v", err, report.Warnings)
	}
	if report.Decoded != 1 || len(report.Books) != 1 || len(report.Books[0].Highlights) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	highlight := report.Books[0].Highlights[0]
	if highlight.Title != "Example Title" || highlight.Text != "beta" || highlight.PageAt != "#1" {
		t.Fatalf("unexpected highlight: %+v", highlight)
	}
	if !highlight.CreatedAt.Equal(created) {
		t.Fatalf("createdAt = %s, want %s", highlight.CreatedAt, created)
	}

	direct, err := ExtractPath(sidecarDir)
	if err != nil || direct.Decoded != 1 {
		t.Fatalf("direct ExtractPath() = %+v, %v", direct, err)
	}
}

func TestExtractPathKFXEndToEnd(t *testing.T) {
	dir := t.TempDir()
	bookPath := filepath.Join(dir, "Example.kfx")
	sidecarDir := filepath.Join(dir, "Example.sdr")
	if err := os.Mkdir(sidecarDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bookPath, buildKFXContainerFixture(t, "旅途测试", "甲乙丙丁"), 0o600); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 8, 2, 9, 30, 0, 0, time.UTC)
	sidecar := buildKRDSFixture(t, []fixtureAnnotation{
		{kind: 1, start: "fixture:2", end: "fixture:3", created: created},
		{kind: 2, start: "fixture:3", end: "fixture:3", created: created.Add(time.Second), note: "重点"},
	}, nil)
	if err := os.WriteFile(filepath.Join(sidecarDir, "annotations.yjr"), sidecar, 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := ExtractPath(dir)
	if err != nil {
		t.Fatalf("ExtractPath() error = %v; warnings = %v", err, report.Warnings)
	}
	if report.Decoded != 1 || len(report.Books) != 1 || len(report.Books[0].Highlights) != 1 {
		t.Fatalf("unexpected KFX report: %+v", report)
	}
	highlight := report.Books[0].Highlights[0]
	if highlight.Title != "旅途测试" || highlight.Text != "乙丙" || highlight.ExactText != "乙丙" {
		t.Fatalf("unexpected KFX highlight text: %+v", highlight)
	}
	if highlight.Note != "重点" || highlight.PageAt != "#7" || !highlight.CreatedAt.Equal(created) {
		t.Fatalf("unexpected KFX highlight metadata: %+v", highlight)
	}

	direct, err := ExtractPath(bookPath)
	if err != nil || direct.Decoded != 1 || len(direct.Books[0].Highlights) != 1 {
		t.Fatalf("direct KFX ExtractPath() = %+v, %v", direct, err)
	}
}

func TestExtractPathKFXEmptyAnnotations(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Empty.kfx"), buildKFXContainerFixture(t, "空标注", "没有标注"), 0o600); err != nil {
		t.Fatal(err)
	}
	sidecarDir := filepath.Join(dir, "Empty.sdr")
	if err := os.Mkdir(sidecarDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sidecarDir, "empty.yjr"), buildKRDSFixture(t, nil, nil), 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := ExtractPath(sidecarDir)
	if err == nil || !strings.Contains(err.Error(), "no highlighted text") {
		t.Fatalf("ExtractPath(empty KFX) error = %v", err)
	}
	if report.Decoded != 1 || len(report.Books) != 1 || len(report.Books[0].Highlights) != 0 {
		t.Fatalf("unexpected empty KFX report: %+v", report)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], "annotation cache is empty") {
		t.Fatalf("empty KFX warnings = %v", report.Warnings)
	}
}

func TestExtractPathKFXRecoversBadFileFallback(t *testing.T) {
	dir := t.TempDir()
	bookPath := filepath.Join(dir, "Recovered.kfx")
	if err := os.WriteFile(bookPath, buildKFXContainerFixture(t, "恢复测试", "甲乙丙丁"), 0o600); err != nil {
		t.Fatal(err)
	}
	sidecarDir := filepath.Join(dir, "Recovered.sdr")
	if err := os.Mkdir(sidecarDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sidecarDir, "active.yjr"), buildKRDSFixture(t, nil, nil), 0o600); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 8, 2, 3, 4, 5, 0, time.UTC)
	backup := buildKRDSFixture(t, []fixtureAnnotation{
		{kind: 1, start: "AVgCAAAoAAAA:1", end: "AVgCAAAoAAAA:2", created: created},
	}, nil)
	if err := os.WriteFile(filepath.Join(sidecarDir, "backup.yjr.bad_file"), backup, 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := ExtractPath(sidecarDir)
	if err != nil {
		t.Fatalf("ExtractPath(fallback KFX) error = %v; warnings = %v", err, report.Warnings)
	}
	if report.Decoded != 1 || len(report.Books) != 1 || len(report.Books[0].Highlights) != 1 {
		t.Fatalf("unexpected fallback KFX report: %+v", report)
	}
	highlight := report.Books[0].Highlights[0]
	if highlight.Text != "甲乙" || !highlight.CreatedAt.Equal(created) {
		t.Fatalf("fallback highlight = %+v", highlight)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], "recovered annotations") {
		t.Fatalf("fallback warnings = %v", report.Warnings)
	}
}

func TestKFXRejectsMalformedData(t *testing.T) {
	valid := buildKFXContainerFixture(t, "Malformed", "text")
	tests := []struct {
		name string
		data []byte
	}{
		{name: "signature", data: []byte("NOPE")},
		{name: "truncated", data: valid[:20]},
		{name: "unsupported version", data: append([]byte(nil), valid...)},
	}
	binary.LittleEndian.PutUint16(tests[2].data[4:6], 99)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := assembleKFX(test.data); err == nil {
				t.Fatal("assembleKFX() accepted malformed data")
			}
		})
	}
}

func TestKFXTextAtAcrossFragments(t *testing.T) {
	book := kfxBook{sections: []kfxSection{
		{position: 10, text: "甲乙"},
		{position: 12, text: "丙丁"},
	}}
	text, exact := book.textAt(11, 12)
	if !exact || text != "乙丙" {
		t.Fatalf("textAt() = %q, %v; want 乙丙, true", text, exact)
	}
}

type fixtureAnnotation struct {
	kind    int64
	start   string
	end     string
	created time.Time
	note    string
}

func buildKRDSFixture(t *testing.T, annotations []fixtureAnnotation, pages []int64) []byte {
	t.Helper()
	var output bytes.Buffer
	output.Write(krdsSignature)
	writeKRDSInt(&output, 1)
	writeKRDSInt(&output, 2)

	writeKRDSObjectStart(&output, "annotation.cache.object")
	writeKRDSInt(&output, int64(len(annotations)))
	for _, annotation := range annotations {
		writeKRDSInt(&output, annotation.kind)
		writeKRDSObjectStart(&output, "saved.avl.interval.tree")
		writeKRDSInt(&output, 1)
		name := map[int64]string{1: "highlight", 2: "note", 13: "underline"}[annotation.kind]
		writeKRDSObjectStart(&output, "annotation.personal."+name)
		writeKRDSString(&output, annotation.start)
		writeKRDSString(&output, annotation.end)
		writeKRDSLong(&output, annotation.created.UnixMilli())
		writeKRDSLong(&output, annotation.created.UnixMilli())
		writeKRDSString(&output, "template")
		if annotation.kind == 2 {
			writeKRDSString(&output, annotation.note)
		}
		output.WriteByte(0xff)
		output.WriteByte(0xff)
	}
	output.WriteByte(0xff)

	writeKRDSObjectStart(&output, "apnx.key")
	writeKRDSString(&output, "asin")
	writeKRDSString(&output, "EBOK")
	writeKRDSBool(&output, true)
	writeKRDSInt(&output, int64(len(pages)))
	for _, page := range pages {
		writeKRDSLong(&output, page)
	}
	output.WriteByte(0xff)
	return output.Bytes()
}

func writeKRDSObjectStart(output *bytes.Buffer, name string) {
	output.WriteByte(0xfe)
	writeKRDSStringValue(output, name)
}

func writeKRDSString(output *bytes.Buffer, value string) {
	output.WriteByte(byte(krdsUTF))
	writeKRDSStringValue(output, value)
}

func writeKRDSStringValue(output *bytes.Buffer, value string) {
	output.WriteByte(0)
	_ = binary.Write(output, binary.BigEndian, uint16(len(value)))
	output.WriteString(value)
}

func writeKRDSInt(output *bytes.Buffer, value int64) {
	output.WriteByte(byte(krdsInt))
	_ = binary.Write(output, binary.BigEndian, int32(value))
}

func writeKRDSLong(output *bytes.Buffer, value int64) {
	output.WriteByte(byte(krdsLong))
	_ = binary.Write(output, binary.BigEndian, value)
}

func writeKRDSBool(output *bytes.Buffer, value bool) {
	output.WriteByte(byte(krdsBoolean))
	if value {
		output.WriteByte(1)
	} else {
		output.WriteByte(0)
	}
}

func buildKF8Fixture(t *testing.T, title string, markup []byte) []byte {
	t.Helper()
	header := make([]byte, 0x118+len(title))
	binary.BigEndian.PutUint16(header[0:2], mobiCompressionNone)
	binary.BigEndian.PutUint16(header[8:10], 1)
	copy(header[16:20], []byte("MOBI"))
	binary.BigEndian.PutUint32(header[20:24], 0x108)
	binary.BigEndian.PutUint32(header[28:32], 65001)
	binary.BigEndian.PutUint32(header[36:40], 8)
	binary.BigEndian.PutUint32(header[0x54:0x58], 0x118)
	binary.BigEndian.PutUint32(header[0x58:0x5c], uint32(len(title)))
	binary.BigEndian.PutUint32(header[0x70:0x74], missingSection)
	binary.BigEndian.PutUint32(header[0xc0:0xc4], missingSection)
	binary.BigEndian.PutUint32(header[0xc4:0xc8], 1)
	binary.BigEndian.PutUint32(header[0xf8:0xfc], missingSection)
	binary.BigEndian.PutUint32(header[0xfc:0x100], missingSection)
	copy(header[0x118:], []byte(title))

	sections := [][]byte{header, markup}
	recordTableEnd := 78 + len(sections)*8
	total := recordTableEnd
	for _, section := range sections {
		total += len(section)
	}
	file := make([]byte, total)
	copy(file[:32], []byte("Fixture"))
	copy(file[60:68], []byte("BOOKMOBI"))
	binary.BigEndian.PutUint16(file[76:78], uint16(len(sections)))
	offset := recordTableEnd
	for i, section := range sections {
		binary.BigEndian.PutUint32(file[78+i*8:82+i*8], uint32(offset))
		copy(file[offset:], section)
		offset += len(section)
	}
	return file
}

type fixtureKFXEntity struct {
	idSID   uint32
	typeSID uint32
	data    []byte
}

func buildKFXContainerFixture(t *testing.T, title, text string) []byte {
	t.Helper()
	textLength := int64(utf8.RuneCountInString(text))
	entities := []fixtureKFXEntity{
		buildKFXEntityFixture(t, 900, 145, map[string]any{
			"$146": []any{text},
		}),
		buildKFXEntityFixture(t, 901, 259, map[string]any{
			"$146": []any{map[string]any{
				"$145": map[string]any{"name": "$900", "$403": int64(0)},
				"$155": int64(2), "$159": "$269",
			}},
		}),
		buildKFXEntityFixture(t, 348, 265, map[string]any{
			"$181": []any{map[string]any{"$144": textLength + 1, "$174": "$902", "$184": int64(0)}},
		}),
		buildKFXEntityFixture(t, 902, 609, map[string]any{
			"$174": "$902",
			"$181": []any{[]any{int64(0), int64(1)}, []any{int64(1), int64(2)}, []any{textLength, int64(0)}},
		}),
		buildKFXEntityFixture(t, 348, 490, map[string]any{
			"$491": []any{map[string]any{
				"$495": "kindle_title_metadata",
				"$258": []any{map[string]any{"$492": "title", "$307": title}},
			}},
		}),
		buildKFXEntityFixture(t, 903, 391, map[string]any{
			"$235": "$237",
			"$247": []any{map[string]any{
				"$241": map[string]any{"$244": "7"},
				"$246": map[string]any{"$155": int64(2), "$143": int64(0)},
			}},
		}),
	}

	indexLength := len(entities) * 24
	entityOffset := uint64(0)
	index := make([]byte, indexLength)
	for i, entity := range entities {
		offset := i * 24
		binary.LittleEndian.PutUint32(index[offset:offset+4], entity.idSID)
		binary.LittleEndian.PutUint32(index[offset+4:offset+8], entity.typeSID)
		binary.LittleEndian.PutUint64(index[offset+8:offset+16], entityOffset)
		binary.LittleEndian.PutUint64(index[offset+16:offset+24], uint64(len(entity.data)))
		entityOffset += uint64(len(entity.data))
	}
	containerInfo := encodeKFXIonFixture(t, map[string]any{
		"$409": "fixture", "$410": int64(0), "$411": int64(0), "$412": int64(4096),
		"$413": int64(kfxContainerHeaderSize), "$414": int64(indexLength), "$416": int64(0),
	})
	headerLength := kfxContainerHeaderSize + len(index) + len(containerInfo)
	result := make([]byte, headerLength)
	copy(result[:4], "CONT")
	binary.LittleEndian.PutUint16(result[4:6], 2)
	binary.LittleEndian.PutUint32(result[6:10], uint32(headerLength))
	binary.LittleEndian.PutUint32(result[10:14], uint32(kfxContainerHeaderSize+len(index)))
	binary.LittleEndian.PutUint32(result[14:18], uint32(len(containerInfo)))
	copy(result[kfxContainerHeaderSize:], index)
	copy(result[kfxContainerHeaderSize+len(index):], containerInfo)
	for _, entity := range entities {
		result = append(result, entity.data...)
	}
	return result
}

func buildKFXEntityFixture(t *testing.T, idSID, typeSID uint32, value any) fixtureKFXEntity {
	t.Helper()
	info := encodeKFXIonFixture(t, map[string]any{"$410": int64(0), "$411": int64(0)})
	payload := encodeKFXIonFixture(t, value)
	headerLength := kfxEntityHeaderSize + len(info)
	data := make([]byte, headerLength)
	copy(data[:4], "ENTY")
	binary.LittleEndian.PutUint16(data[4:6], 1)
	binary.LittleEndian.PutUint32(data[6:10], uint32(headerLength))
	copy(data[kfxEntityHeaderSize:], info)
	data = append(data, payload...)
	return fixtureKFXEntity{idSID: idSID, typeSID: typeSID, data: data}
}

func encodeKFXIonFixture(t *testing.T, value any) []byte {
	t.Helper()
	var output bytes.Buffer
	w := ion.NewBinaryWriter(&output, yjSharedSymbols())
	if err := ion.MarshalTo(w, value); err != nil {
		t.Fatal(err)
	}
	if err := w.Finish(); err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), output.Bytes()...)
}

func buildIndexFixture(t *testing.T, tags []indexTag, text string, control byte, values []uint64) ([]byte, []byte) {
	t.Helper()
	tagLength := 12 + len(tags)*4
	main := make([]byte, 56+tagLength)
	copy(main[:4], []byte("INDX"))
	binary.BigEndian.PutUint32(main[4:8], 56)
	binary.BigEndian.PutUint32(main[24:28], 1)
	copy(main[56:60], []byte("TAGX"))
	binary.BigEndian.PutUint32(main[60:64], uint32(tagLength))
	binary.BigEndian.PutUint32(main[64:68], 1)
	for i, tag := range tags {
		position := 68 + i*4
		main[position] = tag.tag
		main[position+1] = tag.valuesPerEntry
		main[position+2] = tag.mask
		main[position+3] = tag.endFlag
	}

	entry := []byte{byte(len(text))}
	entry = append(entry, text...)
	entry = append(entry, control)
	for _, value := range values {
		if value >= 0x80 {
			t.Fatalf("fixture value %d requires multi-byte VWI", value)
		}
		entry = append(entry, byte(value)|0x80)
	}
	const entryStart = 56
	idxt := entryStart + len(entry)
	extra := make([]byte, idxt+6)
	copy(extra[:4], []byte("INDX"))
	binary.BigEndian.PutUint32(extra[4:8], 56)
	binary.BigEndian.PutUint32(extra[20:24], uint32(idxt))
	binary.BigEndian.PutUint32(extra[24:28], 1)
	copy(extra[entryStart:], entry)
	copy(extra[idxt:idxt+4], []byte("IDXT"))
	binary.BigEndian.PutUint16(extra[idxt+4:idxt+6], entryStart)
	return main, extra
}

func TestDiscoverWarnsForMissingKFXBook(t *testing.T) {
	dir := t.TempDir()
	sidecar := filepath.Join(dir, "KFX.sdr")
	if err := os.Mkdir(sidecar, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sidecar, "data.yjr"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := ExtractPath(dir)
	if err == nil || !strings.Contains(err.Error(), "no supported") || len(report.Warnings) == 0 {
		t.Fatalf("ExtractPath() report=%+v err=%v", report, err)
	}
}
