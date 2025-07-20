package parser

import (
	"strings"
	"testing"
	"time"
)

func TestParseEnglishClippings(t *testing.T) {
	input := `The Great Gatsby (F. Scott Fitzgerald)
- Your Highlight on page 7 | location 100-101 | Added on Monday, April 1, 2024 2:30:45 PM

In his blue gardens men and girls came and went like moths among the whisperings and the champagne and the stars.
==========
Another Book (Author Name)  
- Your Highlight on page 15 | location 200-205 | Added on Tuesday, April 2, 2024 3:45:30 PM

This is another highlight from a different book.
==========`

	clippings, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(clippings) != 2 {
		t.Fatalf("Expected 2 clippings, got %d", len(clippings))
	}

	// Test first clipping
	first := clippings[0]
	if first.Title != "The Great Gatsby" {
		t.Errorf("Expected title 'The Great Gatsby', got '%s'", first.Title)
	}

	if first.PageAt != "#7" {
		t.Errorf("Expected pageAt '#7', got '%s'", first.PageAt)
	}

	expectedContent := "In his blue gardens men and girls came and went like moths among the whisperings and the champagne and the stars."
	if first.Content != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, first.Content)
	}

	// Test date parsing
	expectedTime := time.Date(2024, 4, 1, 14, 30, 45, 0, time.UTC)
	if !first.CreatedAt.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, first.CreatedAt)
	}
}

func TestParseChineseClippings(t *testing.T) {
	input := `深度工作 (卡尔·纽波特)
- 您在位置 #42-43的标注 | 添加于 2024年4月1日星期一 下午2:30:45

专注力就像肌肉一样，使用后会疲劳。
==========`

	clippings, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(clippings) != 1 {
		t.Fatalf("Expected 1 clipping, got %d", len(clippings))
	}

	first := clippings[0]
	if first.Title != "深度工作" {
		t.Errorf("Expected title '深度工作', got '%s'", first.Title)
	}

	if first.PageAt != "#42-43" {
		t.Errorf("Expected pageAt '#42-43', got '%s'", first.PageAt)
	}
}

func TestParseTitleWithParentheses(t *testing.T) {
	input := `Some Book (Author Name) (Series: Book 1)
- Your Highlight on page 7 | location 100-101 | Added on Monday, April 1, 2024 2:30:45 PM

Some content here.
==========`

	clippings, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(clippings) != 1 {
		t.Fatalf("Expected 1 clipping, got %d", len(clippings))
	}

	// Should extract title before first parenthesis
	expected := "Some Book"
	if clippings[0].Title != expected {
		t.Errorf("Expected title '%s', got '%s'", expected, clippings[0].Title)
	}
}

func TestParseBOMRemoval(t *testing.T) {
	// Input with UTF-8 BOM
	input := "\ufeffThe Great Gatsby (F. Scott Fitzgerald)\n- Your Highlight on page 7 | location 100-101 | Added on Monday, April 1, 2024 2:30:45 PM\n\nSome content.\n=========="

	clippings, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(clippings) != 1 {
		t.Fatalf("Expected 1 clipping, got %d", len(clippings))
	}

	// Title should not contain BOM
	if strings.Contains(clippings[0].Title, "\ufeff") {
		t.Errorf("Title contains BOM: '%s'", clippings[0].Title)
	}
}

func TestParseEmptyInput(t *testing.T) {
	clippings, err := Parse("")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(clippings) != 0 {
		t.Fatalf("Expected 0 clippings for empty input, got %d", len(clippings))
	}
}

func TestParseInvalidInput(t *testing.T) {
	// Invalid input - not enough lines
	input := `Some Title
Invalid structure`

	clippings, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should return empty result for invalid groups
	if len(clippings) != 0 {
		t.Fatalf("Expected 0 clippings for invalid input, got %d", len(clippings))
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected Language
	}{
		{"Your Highlight on page", LanguageEnglish},
		{"您在位置", LanguageChinese},
		{"Some other text", LanguageChinese}, // Default to Chinese
	}

	for _, test := range tests {
		result := detectLanguage(test.input)
		if result != test.expected {
			t.Errorf("detectLanguage('%s') = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestParseTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Title", "Simple Title"},
		{"Title (Author)", "Title"},
		{"Title (Author) (Series)", "Title"},
		{"Title（作者）", "Title"},
		{"Title) with trailing paren", "Title) with trailing paren"},
	}

	for _, test := range tests {
		result := parseTitle(test.input)
		if result != test.expected {
			t.Errorf("parseTitle('%s') = '%s', expected '%s'", test.input, result, test.expected)
		}
	}
}