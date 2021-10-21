package core

import "time"

type TClippingItem struct {
	Title     string
	Content   string
	PageAt    string
	CreatedAt time.Time
}

type KindleClippingFileLines int

const (
	KindleClippingFileLinesTitle KindleClippingFileLines = iota + 1
	KindleClippingFileLinesInfo
	KindleClippingFileLinesContent
)

const (
	KindleDateTimeLayout = "Monday, January 2, 2006 3:4:5 PM"
)
