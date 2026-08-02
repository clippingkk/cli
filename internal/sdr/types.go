package sdr

import "time"

// AnnotationType identifies a Kindle text annotation.
type AnnotationType string

const (
	AnnotationHighlight AnnotationType = "highlight"
	AnnotationNote      AnnotationType = "note"
	AnnotationUnderline AnnotationType = "underline"
)

// Annotation is the subset of a KRDS annotation used by the extractor.
type Annotation struct {
	Type             AnnotationType
	StartPosition    int64
	EndPosition      int64
	CreationTime     time.Time
	ModificationTime time.Time
	Note             string
}

// PageMap maps assembled-text positions to printed pages.
type PageMap struct {
	Positions []int64
}

// Sidecar is annotation data decoded from Kindle reader-data-store files.
type Sidecar struct {
	Annotations []Annotation
	PageMap     PageMap
}

// Highlight is a recovered text annotation and its display metadata.
type Highlight struct {
	Title         string
	Text          string
	ExactText     string
	Type          AnnotationType
	StartPosition int64
	EndPosition   int64
	CreatedAt     time.Time
	Note          string
	PageAt        string
}

// BookResult is the extraction result for one Kindle book.
type BookResult struct {
	BookPath   string
	SidecarDir string
	Title      string
	Highlights []Highlight
}

// Report summarizes a path scan.
type Report struct {
	Books    []BookResult
	Warnings []string
	Decoded  int
}
