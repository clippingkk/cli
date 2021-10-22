package core

import "time"

type TClippingItem struct {
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	PageAt    string    `json:"pageAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type KindleClippingFileLines int

const (
	KindleClippingFileLinesTitle KindleClippingFileLines = iota + 1
	KindleClippingFileLinesInfo
	KindleClippingFileLinesContent
)

const (
	KindleDateTimeENLayout = "Monday, January 2, 2006 3:4:5 PM"
	KindleDateTimeZHLayout = "2006-1-2 3:4:5"
)
