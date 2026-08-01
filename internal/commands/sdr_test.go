package commands

import (
	"strings"
	"testing"
	"time"

	"github.com/clippingkk/cli/internal/sdr"
)

func TestRenderSDRText(t *testing.T) {
	report := sdr.Report{Books: []sdr.BookResult{{
		Title: "原则",
		Highlights: []sdr.Highlight{{
			Type: sdr.AnnotationNote, Text: "selected text", Note: "remember",
			PageAt: "#12", CreatedAt: time.Date(2026, 7, 1, 8, 30, 0, 0, time.UTC),
		}},
	}}}
	var output strings.Builder
	if err := renderSDRText(&output, report); err != nil {
		t.Fatalf("renderSDRText() error = %v", err)
	}
	want := "原则\n==\n\n[note] selected text\n  Note: remember\n  Location: #12 | Created: 2026-07-01T08:30:00Z\n\n"
	if output.String() != want {
		t.Fatalf("renderSDRText() = %q, want %q", output.String(), want)
	}
}
