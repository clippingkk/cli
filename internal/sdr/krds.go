package sdr

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var krdsSignature = []byte{0, 0, 0, 0, 0, 0x1a, 0xb1, 0x26}

const (
	krdsBoolean     int8 = 0
	krdsInt         int8 = 1
	krdsLong        int8 = 2
	krdsUTF         int8 = 3
	krdsDouble      int8 = 4
	krdsShort       int8 = 5
	krdsFloat       int8 = 6
	krdsByte        int8 = 7
	krdsChar        int8 = 9
	krdsObjectBegin int8 = -2
	krdsObjectEnd   int8 = -1
)

const maxKRDSValues = 1_000_000

type krdsDecoder struct {
	data []byte
	off  int
}

// DecodeSidecar decodes the annotation and page-map portions of a Kindle
// reader-data-store file.
func DecodeSidecar(data []byte) (Sidecar, error) {
	d := &krdsDecoder{data: data}
	if len(data) < len(krdsSignature) || string(data[:len(krdsSignature)]) != string(krdsSignature) {
		return Sidecar{}, fmt.Errorf("invalid KRDS signature")
	}
	d.off = len(krdsSignature)
	first, err := d.next(nil)
	if err != nil {
		return Sidecar{}, fmt.Errorf("decode KRDS version: %w", err)
	}
	if n, ok := asInt64(first); !ok || n != 1 {
		return Sidecar{}, fmt.Errorf("unsupported KRDS version %v", first)
	}
	countValue, err := d.next(nil)
	if err != nil {
		return Sidecar{}, fmt.Errorf("decode KRDS object count: %w", err)
	}
	count, ok := asCount(countValue)
	if !ok {
		return Sidecar{}, fmt.Errorf("invalid KRDS object count %v", countValue)
	}

	var result Sidecar
	for i := 0; i < count; i++ {
		value, err := d.next(nil)
		if err != nil {
			return Sidecar{}, fmt.Errorf("decode KRDS object %d: %w", i, err)
		}
		object, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if cache, ok := object["annotation.cache.object"].(map[string]any); ok {
			appendAnnotations(&result, cache)
		}
		if apnx, ok := object["apnx.key"].(map[string]any); ok {
			result.PageMap = pageMapFromObject(apnx)
		}
	}
	return result, nil
}

func (d *krdsDecoder) next(forced *int8) (any, error) {
	var datatype int8
	if forced != nil {
		datatype = *forced
	} else {
		b, err := d.byte()
		if err != nil {
			return nil, err
		}
		datatype = int8(b)
	}

	switch datatype {
	case krdsBoolean:
		b, err := d.byte()
		if err != nil {
			return nil, err
		}
		if b > 1 {
			return nil, fmt.Errorf("invalid boolean %d at offset %d", b, d.off-1)
		}
		return b == 1, nil
	case krdsInt:
		b, err := d.take(4)
		if err != nil {
			return nil, err
		}
		return int64(int32(binary.BigEndian.Uint32(b))), nil
	case krdsLong:
		b, err := d.take(8)
		if err != nil {
			return nil, err
		}
		return int64(binary.BigEndian.Uint64(b)), nil
	case krdsUTF:
		emptyType := krdsBoolean
		empty, err := d.next(&emptyType)
		if err != nil {
			return nil, err
		}
		if empty.(bool) {
			return "", nil
		}
		b, err := d.take(2)
		if err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint16(b))
		text, err := d.take(n)
		if err != nil {
			return nil, err
		}
		return string(text), nil
	case krdsDouble:
		b, err := d.take(8)
		if err != nil {
			return nil, err
		}
		return math.Float64frombits(binary.BigEndian.Uint64(b)), nil
	case krdsShort:
		b, err := d.take(2)
		if err != nil {
			return nil, err
		}
		return int64(int16(binary.BigEndian.Uint16(b))), nil
	case krdsFloat:
		b, err := d.take(4)
		if err != nil {
			return nil, err
		}
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b))), nil
	case krdsByte:
		b, err := d.byte()
		return int64(int8(b)), err
	case krdsChar:
		b, err := d.byte()
		return string([]byte{b}), err
	case krdsObjectBegin:
		nameType := krdsUTF
		nameValue, err := d.next(&nameType)
		if err != nil {
			return nil, fmt.Errorf("decode object name: %w", err)
		}
		name := nameValue.(string)
		values := make([]any, 0, 8)
		for {
			if d.off >= len(d.data) {
				return nil, fmt.Errorf("unterminated object %q", name)
			}
			if int8(d.data[d.off]) == krdsObjectEnd {
				d.off++
				break
			}
			if len(values) >= maxKRDSValues {
				return nil, fmt.Errorf("object %q exceeds value limit", name)
			}
			value, err := d.next(nil)
			if err != nil {
				return nil, fmt.Errorf("decode object %q: %w", name, err)
			}
			values = append(values, value)
		}
		return map[string]any{name: decodeKRDSObject(name, values)}, nil
	default:
		return nil, fmt.Errorf("unknown KRDS datatype %d at offset %d", datatype, d.off-1)
	}
}

func (d *krdsDecoder) byte() (byte, error) {
	b, err := d.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (d *krdsDecoder) take(n int) ([]byte, error) {
	if n < 0 || d.off > len(d.data)-n {
		return nil, fmt.Errorf("truncated KRDS data at offset %d", d.off)
	}
	b := d.data[d.off : d.off+n]
	d.off += n
	return b, nil
}

func decodeKRDSObject(name string, values []any) any {
	pop := func() (any, bool) {
		if len(values) == 0 {
			return nil, false
		}
		v := values[0]
		values = values[1:]
		return v, true
	}

	switch name {
	case "saved.avl.interval.tree":
		countValue, ok := pop()
		count, valid := asCount(countValue)
		if !ok || !valid || count > len(values) {
			return values
		}
		return append([]any(nil), values[:count]...)
	case "annotation.personal.bookmark", "annotation.personal.highlight",
		"annotation.personal.note", "annotation.personal.clip_article",
		"annotation.personal.handwritten_note", "annotation.personal.sticky_note",
		"annotation.personal.underline":
		if len(values) < 5 {
			return values
		}
		obj := map[string]any{
			"startPosition":        values[0],
			"endPosition":          values[1],
			"creationTime":         values[2],
			"lastModificationTime": values[3],
			"template":             values[4],
		}
		if name == "annotation.personal.note" && len(values) > 5 {
			obj["note"] = values[5]
		}
		return obj
	case "annotation.cache.object":
		return decodeAnnotationCache(values)
	case "apnx.key":
		return decodeAPNX(values)
	default:
		return values
	}
}

func decodeAnnotationCache(values []any) map[string]any {
	result := make(map[string]any)
	if len(values) == 0 {
		return result
	}
	count, ok := asCount(values[0])
	if !ok {
		return result
	}
	values = values[1:]
	classes := map[int64]string{
		0: "bookmark", 1: "highlight", 2: "note", 3: "clip_article",
		10: "handwritten_note", 11: "sticky_note", 13: "underline",
	}
	for i := 0; i < count && len(values) >= 2; i++ {
		kindCode, ok := asInt64(values[0])
		tree, treeOK := objectValue(values[1], "saved.avl.interval.tree")
		values = values[2:]
		kind, known := classes[kindCode]
		if !ok || !treeOK || !known {
			continue
		}
		items, ok := tree.([]any)
		if !ok {
			continue
		}
		decoded := make([]map[string]any, 0, len(items))
		fullName := "annotation.personal." + kind
		for _, item := range items {
			if value, ok := objectValue(item, fullName); ok {
				if obj, ok := value.(map[string]any); ok {
					decoded = append(decoded, obj)
				}
			}
		}
		result[kind] = decoded
	}
	return result
}

func decodeAPNX(values []any) map[string]any {
	result := make(map[string]any)
	if len(values) < 4 {
		return result
	}
	result["asin"] = values[0]
	result["cdeType"] = values[1]
	result["sidecarAvailable"] = values[2]
	count, ok := asCount(values[3])
	if !ok || len(values) < 4+count {
		return result
	}
	positions := make([]int64, 0, count)
	for _, value := range values[4 : 4+count] {
		if position, ok := asPosition(value); ok {
			positions = append(positions, position)
		}
	}
	result["oPNToPosition"] = positions
	return result
}

func appendAnnotations(result *Sidecar, cache map[string]any) {
	for _, kind := range []AnnotationType{AnnotationHighlight, AnnotationNote, AnnotationUnderline} {
		items, ok := cache[string(kind)].([]map[string]any)
		if !ok {
			continue
		}
		for _, item := range items {
			start, startOK := asPosition(item["startPosition"])
			end, endOK := asPosition(item["endPosition"])
			created, createdOK := asMilliseconds(item["creationTime"])
			if !startOK || !endOK || !createdOK {
				continue
			}
			modified, _ := asMilliseconds(item["lastModificationTime"])
			note, _ := item["note"].(string)
			result.Annotations = append(result.Annotations, Annotation{
				Type: kind, StartPosition: start, EndPosition: end,
				CreationTime: created, ModificationTime: modified, Note: note,
			})
		}
	}
}

func pageMapFromObject(obj map[string]any) PageMap {
	positions, ok := obj["oPNToPosition"].([]int64)
	if !ok {
		return PageMap{}
	}
	return PageMap{Positions: append([]int64(nil), positions...)}
}

func objectValue(value any, name string) (any, bool) {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	v, ok := obj[name]
	return v, ok
}

func asCount(value any) (int, bool) {
	n, ok := asInt64(value)
	if !ok || n < 0 || n > maxKRDSValues {
		return 0, false
	}
	return int(n), true
}

func asInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}

func asPosition(value any) (int64, bool) {
	if n, ok := asInt64(value); ok {
		return n, true
	}
	if text, ok := value.(string); ok {
		if separator := strings.LastIndexByte(text, ':'); separator >= 0 {
			text = text[separator+1:]
		}
		if n, err := strconv.ParseInt(text, 10, 64); err == nil && n >= 0 {
			return n, true
		}
	}
	return 0, false
}

func asMilliseconds(value any) (time.Time, bool) {
	n, ok := asInt64(value)
	if !ok {
		return time.Time{}, false
	}
	return time.UnixMilli(n).UTC(), true
}
