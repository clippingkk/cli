package models

import (
	"encoding/json"
	"time"
)

// ClippingItem represents a single clipping from Kindle
type ClippingItem struct {
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	PageAt    string    `json:"pageAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// MarshalJSON implements custom JSON marshaling to maintain RFC3339 format
func (c ClippingItem) MarshalJSON() ([]byte, error) {
	type Alias ClippingItem
	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"createdAt"`
	}{
		Alias:     (*Alias)(&c),
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling to parse RFC3339 format
func (c *ClippingItem) UnmarshalJSON(data []byte) error {
	type Alias ClippingItem
	aux := &struct {
		*Alias
		CreatedAt string `json:"createdAt"`
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var err error
	c.CreatedAt, err = time.Parse(time.RFC3339, aux.CreatedAt)
	return err
}

// ClippingInput represents the input format for GraphQL mutations
type ClippingInput struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	BookID    string `json:"bookID"`
	PageAt    string `json:"pageAt"`
	CreatedAt string `json:"createdAt"`
	Source    string `json:"source"`
}

// ToClippingInput converts ClippingItem to ClippingInput for API calls
func (c ClippingItem) ToClippingInput() ClippingInput {
	return ClippingInput{
		Title:     c.Title,
		Content:   c.Content,
		BookID:    "0", // Default book ID
		PageAt:    c.PageAt,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
		Source:    "kindle",
	}
}