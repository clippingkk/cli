package core

import (
	"bytes"
	"errors"
	"regexp"
	"time"
)

type FileLanuages int

const (
	FileLanuagesEn FileLanuages = iota
	FileLanuagesZh
)

type clippingsParser struct {
	file           []byte
	lines          [][][]byte
	language       FileLanuages
	separator      []byte
	locationRegexp *regexp.Regexp
}

func NewClippingParser(src []byte) clippingsParser {
	return clippingsParser{
		file:      bytes.Trim(src, "\xef\xbb\xbf"),
		lines:     [][][]byte{},
		separator: []byte("========"),
	}
}

func (c *clippingsParser) Prepare() error {
	lines := bytes.Split(c.file, []byte("\n"))
	temp := make([][]byte, 0)

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if bytes.Contains(line, c.separator) {
			c.lines = append(c.lines, temp)
			temp = [][]byte{}
		}
		if len(line) > 0 && !bytes.Contains(line, c.separator) {
			temp = append(temp, line)
		}
	}

	if bytes.Contains(c.file, []byte("Your Highlight on")) {
		c.locationRegexp = regexp.MustCompile(`\d+(-?\d+)?`)
		c.language = FileLanuagesEn
	} else if bytes.Contains(c.file, []byte("您在")) {
		c.locationRegexp = regexp.MustCompile(`#?\d+(-?\d+)?`)
		c.language = FileLanuagesZh
	} else {
		return errors.New("哎呀呀，暂不支持非中英文的内容呢~")
	}

	return nil
}

func (c *clippingsParser) DoParse() (result []TClippingItem, err error) {
	for _, dataset := range c.lines {
		item := TClippingItem{}
		title := c.exactTitlte(dataset[0])
		item.Title = title

		pageAt, date, err := c.exactInfo(dataset[1])
		if err != nil {
			return result, err
		}
		item.PageAt = pageAt
		item.CreatedAt = date

		item.Content = string(dataset[2])
		result = append(result, item)
	}

	//   return this.result.filter(item => item.content && item.content !== "")
	return
}

func (c clippingsParser) exactTitlte(line []byte) string {
	STOP_WORDS := [][]byte{[]byte("("), []byte("（")}
	title := line

	for _, s := range STOP_WORDS {
		title = bytes.Split(title, s)[0]
	}

	title = bytes.TrimSpace(title)
	return string(title)
}

func (c clippingsParser) exactInfo(line []byte) (pageAt string, date time.Time, err error) {
	l := bytes.Split(line, []byte("|"))
	locationSection := l[0]
	dateSection := l[len(l)-1]
	locationResult := c.locationRegexp.FindStringSubmatch(string(locationSection))

	pageAt = locationResult[0]

	dateSection = bytes.Replace(dateSection, []byte("Added on "), []byte(""), 1)
	dateSection = bytes.Replace(dateSection, []byte("添加于 "), []byte(""), 1)

	dateSection = bytes.TrimSpace(dateSection)

	date, err = time.Parse(KindleDateTimeLayout, string(dateSection))
	return
}