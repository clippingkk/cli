package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/clippingkk/cli/internal/models"
	"github.com/clippingkk/cli/internal/parser"
)

// This test file validates the parse command against all fixture files.
// It complements the existing parser tests in internal/parser/parser_test.go
// by testing the full integration with actual fixture data.

// TestParseBasicValidation performs basic validation that the parser works
func TestParseBasicValidation(t *testing.T) {
	// Simple test to ensure the parser is working
	input := `Test Book (Test Author)
- Your Highlight on page 1 | location 1-1 | Added on Monday, January 1, 2024 12:00:00 PM

Test content
==========`

	clippings, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Basic parse test failed: %v", err)
	}

	if len(clippings) != 1 {
		t.Fatalf("Expected 1 clipping, got %d", len(clippings))
	}

	if clippings[0].Title != "Test Book" {
		t.Errorf("Expected title 'Test Book', got '%s'", clippings[0].Title)
	}

	if clippings[0].Content != "Test content" {
		t.Errorf("Expected content 'Test content', got '%s'", clippings[0].Content)
	}

	t.Log("✅ Basic parser validation passed")
}

// TestParseAllFixtures tests all fixture files against their expected results
func TestParseAllFixtures(t *testing.T) {
	// Define test cases for all fixture files
	testCases := []struct {
		name       string
		txtFile    string
		resultFile string
	}{
		{"English", "../../fixtures/clippings_en.txt", "../../fixtures/clippings_en.result.json"},
		{"Chinese", "../../fixtures/clippings_zh.txt", "../../fixtures/clippings_zh.result.json"},
		{"Other", "../../fixtures/clippings_other.txt", "../../fixtures/clippings_other.result.json"},
		{"Rare", "../../fixtures/clippings_rare.txt", "../../fixtures/clippings_rare.result.json"},
		{"Ric", "../../fixtures/clippings_ric.txt", "../../fixtures/clippings_ric.result.json"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check if files exist
			if _, err := os.Stat(tc.txtFile); os.IsNotExist(err) {
				t.Skipf("Input file %s does not exist, skipping", tc.txtFile)
				return
			}
			if _, err := os.Stat(tc.resultFile); os.IsNotExist(err) {
				t.Skipf("Result file %s does not exist, skipping", tc.resultFile)
				return
			}

			// Read input file
			inputData, err := os.ReadFile(tc.txtFile)
			if err != nil {
				t.Fatalf("Failed to read input file %s: %v", tc.txtFile, err)
			}

			// Read expected result file
			expectedData, err := os.ReadFile(tc.resultFile)
			if err != nil {
				t.Fatalf("Failed to read result file %s: %v", tc.resultFile, err)
			}

			// Parse expected results
			var expectedClippings []models.ClippingItem
			if err := json.Unmarshal(expectedData, &expectedClippings); err != nil {
				t.Fatalf("Failed to unmarshal expected results from %s: %v", tc.resultFile, err)
			}

			// Parse input using the parser
			actualClippings, err := parser.Parse(string(inputData))
			if err != nil {
				t.Fatalf("Parser failed for %s: %v", tc.txtFile, err)
			}

			// Compare lengths
			if len(actualClippings) != len(expectedClippings) {
				t.Errorf("Length mismatch for %s: expected %d clippings, got %d",
					tc.txtFile, len(expectedClippings), len(actualClippings))

				// Log first few clippings for debugging
				t.Logf("Expected first clipping: %+v", expectedClippings[0])
				if len(actualClippings) > 0 {
					t.Logf("Actual first clipping: %+v", actualClippings[0])
				}
				return
			}

			// Compare each clipping
			for i, expected := range expectedClippings {
				actual := actualClippings[i]

				if actual.Title != expected.Title {
					t.Errorf("Title mismatch at index %d for %s: expected '%s', got '%s'",
						i, tc.txtFile, expected.Title, actual.Title)
				}

				if actual.Content != expected.Content {
					t.Errorf("Content mismatch at index %d for %s: expected '%s', got '%s'",
						i, tc.txtFile, expected.Content, actual.Content)
				}

				if actual.PageAt != expected.PageAt {
					t.Errorf("PageAt mismatch at index %d for %s: expected '%s', got '%s'",
						i, tc.txtFile, expected.PageAt, actual.PageAt)
				}

				// Compare timestamps (allowing for timezone differences by comparing UTC)
				if !actual.CreatedAt.UTC().Equal(expected.CreatedAt.UTC()) {
					t.Errorf("CreatedAt mismatch at index %d for %s: expected '%s', got '%s'",
						i, tc.txtFile, expected.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
						actual.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
				}
			}

			t.Logf("✅ Successfully validated %d clippings for %s", len(actualClippings), tc.name)
		})
	}
}

// TestParseFixturesDiscovery automatically discovers all fixture files
func TestParseFixturesDiscovery(t *testing.T) {
	// Get all .txt files in fixtures directory
	txtFiles, err := filepath.Glob("../../fixtures/*.txt")
	if err != nil {
		t.Fatalf("Failed to find fixture files: %v", err)
	}

	if len(txtFiles) == 0 {
		t.Skip("No fixture .txt files found")
	}

	for _, txtFile := range txtFiles {
		t.Run(filepath.Base(txtFile), func(t *testing.T) {
			// Determine corresponding result file
			baseName := txtFile[:len(txtFile)-4] // Remove .txt extension
			resultFile := baseName + ".result.json"

			// Check if result file exists
			if _, err := os.Stat(resultFile); os.IsNotExist(err) {
				t.Skipf("Result file %s does not exist, skipping", resultFile)
				return
			}

			// Read input file
			inputData, err := os.ReadFile(txtFile)
			if err != nil {
				t.Fatalf("Failed to read input file %s: %v", txtFile, err)
			}

			// Read expected result file
			expectedData, err := os.ReadFile(resultFile)
			if err != nil {
				t.Fatalf("Failed to read result file %s: %v", resultFile, err)
			}

			// Parse expected results
			var expectedClippings []models.ClippingItem
			if err := json.Unmarshal(expectedData, &expectedClippings); err != nil {
				t.Fatalf("Failed to unmarshal expected results from %s: %v", resultFile, err)
			}

			// Parse input using the parser
			actualClippings, err := parser.Parse(string(inputData))
			if err != nil {
				t.Fatalf("Parser failed for %s: %v", txtFile, err)
			}

			// Basic validation
			if len(actualClippings) != len(expectedClippings) {
				t.Errorf("Expected %d clippings, got %d for %s", len(expectedClippings), len(actualClippings), txtFile)
			}

			// Detailed comparison for first few items
			maxCompare := len(expectedClippings)
			if len(actualClippings) < maxCompare {
				maxCompare = len(actualClippings)
			}

			for i := 0; i < maxCompare; i++ {
				expected := expectedClippings[i]
				actual := actualClippings[i]

				if actual.Title != expected.Title {
					t.Errorf("Title mismatch at index %d: expected '%s', got '%s'", i, expected.Title, actual.Title)
				}

				if actual.Content != expected.Content {
					t.Errorf("Content mismatch at index %d: expected '%s', got '%s'", i, expected.Content, actual.Content)
				}

				if actual.PageAt != expected.PageAt {
					t.Errorf("PageAt mismatch at index %d: expected '%s', got '%s'", i, expected.PageAt, actual.PageAt)
				}
			}

			t.Logf("Processed %s: %d clippings", filepath.Base(txtFile), len(actualClippings))
		})
	}
}

// TestParseFixturesJSON validates JSON serialization matches expected format
func TestParseFixturesJSON(t *testing.T) {
	testCases := []struct {
		name       string
		txtFile    string
		resultFile string
	}{
		{"English", "../../fixtures/clippings_en.txt", "../../fixtures/clippings_en.result.json"},
		{"Chinese", "../../fixtures/clippings_zh.txt", "../../fixtures/clippings_zh.result.json"},
		{"Other", "../../fixtures/clippings_other.txt", "../../fixtures/clippings_other.result.json"},
		{"Rare", "../../fixtures/clippings_rare.txt", "../../fixtures/clippings_rare.result.json"},
		{"Ric", "../../fixtures/clippings_ric.txt", "../../fixtures/clippings_ric.result.json"},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_json", func(t *testing.T) {
			if _, err := os.Stat(tc.txtFile); os.IsNotExist(err) {
				t.Skipf("Input file %s does not exist, skipping", tc.txtFile)
				return
			}
			if _, err := os.Stat(tc.resultFile); os.IsNotExist(err) {
				t.Skipf("Result file %s does not exist, skipping", tc.resultFile)
				return
			}

			// Read and parse input
			inputData, err := os.ReadFile(tc.txtFile)
			if err != nil {
				t.Fatalf("Failed to read input file %s: %v", tc.txtFile, err)
			}

			actualClippings, err := parser.Parse(string(inputData))
			if err != nil {
				t.Fatalf("Parser failed for %s: %v", tc.txtFile, err)
			}

			// Marshal to JSON
			actualJSON, err := json.MarshalIndent(actualClippings, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal actual results to JSON: %v", err)
			}

			// Read expected JSON
			expectedJSON, err := os.ReadFile(tc.resultFile)
			if err != nil {
				t.Fatalf("Failed to read expected JSON: %v", err)
			}

			// Parse both JSONs to normalize formatting
			var actualNormalized, expectedNormalized interface{}

			if err := json.Unmarshal(actualJSON, &actualNormalized); err != nil {
				t.Fatalf("Failed to unmarshal actual JSON: %v", err)
			}

			if err := json.Unmarshal(expectedJSON, &expectedNormalized); err != nil {
				t.Fatalf("Failed to unmarshal expected JSON: %v", err)
			}

			// Re-marshal both to ensure consistent formatting
			actualNormalizedJSON, _ := json.MarshalIndent(actualNormalized, "", "  ")
			expectedNormalizedJSON, _ := json.MarshalIndent(expectedNormalized, "", "  ")

			if string(actualNormalizedJSON) != string(expectedNormalizedJSON) {
				t.Errorf("JSON output mismatch for %s", tc.txtFile)
				// Only show first 500 chars to avoid overwhelming output
				expectedStr := string(expectedNormalizedJSON)
				actualStr := string(actualNormalizedJSON)
				if len(expectedStr) > 500 {
					expectedStr = expectedStr[:500] + "..."
				}
				if len(actualStr) > 500 {
					actualStr = actualStr[:500] + "..."
				}
				t.Logf("Expected (first 500 chars):\n%s", expectedStr)
				t.Logf("Actual (first 500 chars):\n%s", actualStr)
			} else {
				t.Logf("✅ JSON serialization matches for %s", tc.name)
			}
		})
	}
}
