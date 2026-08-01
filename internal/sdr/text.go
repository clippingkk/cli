package sdr

import (
	"html"
	"regexp"
	"sort"
	"strings"
)

const maxWordSnap = 40

var (
	markupTagPattern = regexp.MustCompile(`<[^>]*>`)
	spacePattern     = regexp.MustCompile(`\s+`)
)

func recoverText(assembled []byte, start, end int64) (string, string) {
	if start > end {
		start, end = end, start
	}
	start = clampPosition(start, len(assembled))
	end = clampPosition(end, len(assembled))
	exact := cleanMarkup(assembled[start:end])
	left, right := snapRange(assembled, int(start), int(end))
	return cleanMarkup(assembled[left:right]), exact
}

func clampPosition(position int64, length int) int64 {
	if position < 0 {
		return 0
	}
	if position > int64(length) {
		return int64(length)
	}
	return position
}

func snapRange(data []byte, start, end int) (int, int) {
	left := start
	steps := 0
	for left > 0 && !asciiWhitespace(data[left-1]) && steps < maxWordSnap {
		left--
		steps++
	}
	if steps >= maxWordSnap {
		left = start
	}
	right := end
	steps = 0
	for right < len(data) && !asciiWhitespace(data[right]) && steps < maxWordSnap {
		right++
		steps++
	}
	if steps >= maxWordSnap {
		right = end
	}
	return left, right
}

func asciiWhitespace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

func cleanMarkup(data []byte) string {
	text := string(data)
	text = markupTagPattern.ReplaceAllString(text, " ")
	if close := strings.IndexByte(text, '>'); close >= 0 {
		open := strings.IndexByte(text, '<')
		if open < 0 || close < open {
			text = text[close+1:]
		}
	}
	if open := strings.LastIndexByte(text, '<'); open >= 0 && !strings.Contains(text[open:], ">") {
		text = text[:open]
	}
	text = html.UnescapeString(text)
	return strings.TrimSpace(spacePattern.ReplaceAllString(text, " "))
}

func pageAt(pageMap PageMap, position int64) string {
	if len(pageMap.Positions) > 0 {
		page := sort.Search(len(pageMap.Positions), func(i int) bool {
			return pageMap.Positions[i] > position
		}) - 1
		if page >= 0 {
			return "#" + itoa64(int64(page))
		}
	}
	return "#" + itoa64(position)
}

func itoa64(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
