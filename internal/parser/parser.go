package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/clippingkk/cli/internal/models"
)

// Language represents the detected language of Kindle clippings
type Language int

const (
	// LanguageEnglish represents English clippings
	LanguageEnglish Language = iota
	// LanguageChinese represents Chinese clippings
	LanguageChinese
)

var (
	// BOM pattern for UTF-8 BOM removal
	bomPattern = regexp.MustCompile(`\ufeff`)
	
	// Location patterns for different languages
	englishLocationPattern = regexp.MustCompile(`\d+(-?\d+)?`)
	chineseLocationPattern = regexp.MustCompile(`#?\d+(-?\d+)?`)
	
	// Chinese character detection pattern
	chinesePattern = regexp.MustCompile(`[\x{4E00}-\x{9FFF}\x{3000}-\x{303F}]`)
	
	// Date parsing patterns for different languages
	englishDateFormat = "Monday, January 2, 2006 3:4:5 PM"
	chineseDateFormat = "2006-1-2 3:4:5 PM"
)

// ParseOptions contains configuration for parsing
type ParseOptions struct {
	// RemoveBOM whether to remove UTF-8 BOM
	RemoveBOM bool
}

// DefaultParseOptions returns default parsing options
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		RemoveBOM: true,
	}
}

// Parse parses Kindle clippings text and returns structured data
func Parse(input string, opts ...ParseOptions) ([]models.ClippingItem, error) {
	options := DefaultParseOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	// Remove BOM if requested
	if options.RemoveBOM {
		input = bomPattern.ReplaceAllString(input, "")
	}

	// Trim and validate input
	input = strings.TrimSpace(input)
	if input == "" {
		return []models.ClippingItem{}, nil
	}

	// Detect language
	language := detectLanguage(input)

	// Split into clipping groups
	groups := splitIntoGroups(input)

	// Parse each group
	var result []models.ClippingItem
	for _, group := range groups {
		item, err := parseGroup(group, language)
		if err != nil {
			// Skip invalid clippings but continue processing
			continue
		}
		if item != nil {
			result = append(result, *item)
		}
	}

	return result, nil
}

// detectLanguage detects the language of the clippings
func detectLanguage(input string) Language {
	if strings.Contains(input, "Your Highlight on") {
		return LanguageEnglish
	}
	return LanguageChinese
}

// splitIntoGroups splits the input into clipping groups using the separator
func splitIntoGroups(input string) [][]string {
	const separator = "========"
	
	lines := strings.Split(input, "\n")
	var groups [][]string
	var currentGroup []string

	for _, line := range lines {
		if strings.Contains(line, separator) {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = []string{}
			}
		} else {
			currentGroup = append(currentGroup, line)
		}
	}

	// Add the last group if it exists
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// parseGroup parses a single clipping group
func parseGroup(group []string, language Language) (*models.ClippingItem, error) {
	// Validate group structure (minimum 4 lines: title, info, empty, content)
	if len(group) < 4 {
		return nil, fmt.Errorf("invalid group structure: not enough lines")
	}

	// Remove BOM from title
	title := parseTitle(bomPattern.ReplaceAllString(group[0], ""))
	if title == "" {
		return nil, fmt.Errorf("empty title")
	}

	// Parse location and date from info line
	location, createdAt, err := parseInfo(group[1], language)
	if err != nil {
		return nil, fmt.Errorf("failed to parse info: %w", err)
	}

	// Get content (skip empty line at index 2)
	content := strings.TrimSpace(group[3])
	if content == "" {
		return nil, fmt.Errorf("empty content")
	}

	return &models.ClippingItem{
		Title:     title,
		Content:   content,
		PageAt:    location,
		CreatedAt: createdAt,
	}, nil
}

// parseTitle extracts and cleans the book title
func parseTitle(line string) string {
	// Remove parentheses and content within them
	stopWords := []string{"(", "（"}
	title := strings.TrimSpace(line)

	for _, stop := range stopWords {
		if idx := strings.Index(title, stop); idx != -1 {
			title = title[:idx]
		}
	}

	// Remove trailing closing parentheses
	title = strings.TrimSuffix(title, ")")
	title = strings.TrimSuffix(title, "）")

	return strings.TrimSpace(title)
}

// parseInfo parses the info line to extract location and date
func parseInfo(line string, language Language) (string, time.Time, error) {
	// Split by pipe character
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return "", time.Time{}, fmt.Errorf("invalid info line format")
	}

	// Parse location
	locationSection := strings.TrimSpace(parts[0])
	var locationPattern *regexp.Regexp
	
	switch language {
	case LanguageEnglish:
		locationPattern = englishLocationPattern
	case LanguageChinese:
		locationPattern = chineseLocationPattern
	}

	matches := locationPattern.FindStringSubmatch(locationSection)
	var location string
	if len(matches) > 0 {
		pageAt := matches[0]
		if !strings.HasPrefix(pageAt, "#") {
			pageAt = "#" + pageAt
		}
		location = pageAt
	} else {
		location = ""
	}

	// Parse date from the last part
	dateSection := strings.TrimSpace(parts[len(parts)-1])
	dateSection = strings.Replace(dateSection, "Added on ", "", 1)
	dateSection = strings.Replace(dateSection, "添加于 ", "", 1)

	var createdAt time.Time
	var err error

	switch language {
	case LanguageEnglish:
		createdAt, err = parseEnglishDate(dateSection)
	case LanguageChinese:
		createdAt, err = parseChineseDate(dateSection)
	}

	if err != nil {
		// Return default time if parsing fails
		createdAt = time.Unix(0, 0).UTC()
	}

	return location, createdAt, nil
}

// parseEnglishDate parses English date format
func parseEnglishDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	return time.Parse(englishDateFormat, dateStr)
}

// parseChineseDate parses Chinese date format
func parseChineseDate(dateStr string) (time.Time, error) {
	// Determine AM/PM
	var ampm string
	if strings.Contains(dateStr, "上午") {
		ampm = "AM"
	} else {
		ampm = "PM"
	}

	// Replace Chinese characters with separators
	dateStr = chinesePattern.ReplaceAllString(dateStr, "-")
	
	// Remove multiple dashes
	multipleDashPattern := regexp.MustCompile(`-{2,}`)
	dateStr = multipleDashPattern.ReplaceAllString(dateStr, "")
	dateStr = strings.TrimSpace(dateStr)
	
	// Add AM/PM suffix
	dateStr = dateStr + " " + ampm

	return time.Parse(chineseDateFormat, dateStr)
}

// validateUTF8 checks if the input is valid UTF-8
func validateUTF8(input string) error {
	if !utf8.ValidString(input) {
		return fmt.Errorf("input is not valid UTF-8")
	}
	return nil
}