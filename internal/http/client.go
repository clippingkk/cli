package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/clippingkk/cli/internal/config"
	"github.com/clippingkk/cli/internal/models"
)

const (
	// ChunkSize is the number of clippings to send per request
	ChunkSize = 20
	// MaxConcurrency is the maximum number of concurrent requests
	MaxConcurrency = 10
	// RequestTimeout is the timeout for individual HTTP requests
	RequestTimeout = 30 * time.Second
)

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	OperationName string      `json:"operationName"`
	Query         string      `json:"query"`
	Variables     interface{} `json:"variables"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{}      `json:"data"`
	Errors []GraphQLError   `json:"errors"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message   string                 `json:"message"`
	Locations []GraphQLLocation      `json:"locations"`
	Path      []interface{}          `json:"path"`
	Extensions map[string]interface{} `json:"extensions"`
}

// GraphQLLocation represents error location
type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// CreateClippingsResponse represents the response from createClippings mutation
type CreateClippingsResponse struct {
	CreateClippings []CreateClippingResult `json:"createClippings"`
}

// CreateClippingResult represents a single clipping creation result
type CreateClippingResult struct {
	ID int64 `json:"id"`
}

// CreateClippingsVariables represents variables for createClippings mutation
type CreateClippingsVariables struct {
	Payload []models.ClippingInput `json:"payload"`
	Visible bool                   `json:"visible"`
}

const createClippingsMutation = `
mutation createClippings($payload: [ClippingInput!]!, $visible: Boolean) {
	createClippings(payload: $payload, visible: $visible) {
		id
	}
}
`

// Client represents an HTTP client for ClippingKK API
type Client struct {
	httpClient *http.Client
	config     *config.Config
	endpoint   string
	headers    map[string]string
}

// NewClient creates a new HTTP client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
		config:   cfg,
		endpoint: cfg.HTTP.Endpoint,
		headers:  cfg.HTTP.Headers,
	}
}

// SyncToServer uploads clippings to the ClippingKK server
func (c *Client) SyncToServer(ctx context.Context, clippings []models.ClippingItem, endpoint string) error {
	// Use provided endpoint or fall back to config
	targetEndpoint := c.endpoint
	if endpoint != "" && endpoint != "http" {
		targetEndpoint = endpoint
	}

	if targetEndpoint == "" || targetEndpoint == "http" {
		return fmt.Errorf("no valid endpoint configured")
	}

	// Split clippings into chunks
	chunks := chunkClippings(clippings, ChunkSize)
	
	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, MaxConcurrency)
	
	// Use WaitGroup to wait for all goroutines
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	fmt.Printf("Uploading %d clippings in %d chunks...\n", len(clippings), len(chunks))

	for i, chunk := range chunks {
		wg.Add(1)
		go func(chunkIndex int, chunkData []models.ClippingInput) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := c.uploadChunk(ctx, targetEndpoint, chunkData, chunkIndex+1); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("chunk %d failed: %w", chunkIndex+1, err))
				mu.Unlock()
			} else {
				fmt.Printf("✅ Chunk %d/%d completed: %d items\n", chunkIndex+1, len(chunks), len(chunkData))
			}
		}(i, convertToClippingInputs(chunk))
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("upload failed with %d errors: %v", len(errors), errors[0])
	}

	fmt.Printf("🎉 Successfully uploaded %d clippings!\n", len(clippings))
	return nil
}

// uploadChunk uploads a single chunk of clippings
func (c *Client) uploadChunk(ctx context.Context, endpoint string, chunk []models.ClippingInput, chunkIndex int) error {
	request := GraphQLRequest{
		OperationName: "createClippings",
		Query:         createClippingsMutation,
		Variables: CreateClippingsVariables{
			Payload: chunk,
			Visible: true,
		},
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(graphqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", graphqlResp.Errors[0].Message)
	}

	return nil
}

// chunkClippings splits clippings into chunks of specified size
func chunkClippings(clippings []models.ClippingItem, chunkSize int) [][]models.ClippingItem {
	var chunks [][]models.ClippingItem
	
	for i := 0; i < len(clippings); i += chunkSize {
		end := i + chunkSize
		if end > len(clippings) {
			end = len(clippings)
		}
		chunks = append(chunks, clippings[i:end])
	}
	
	return chunks
}

// convertToClippingInputs converts ClippingItem slice to ClippingInput slice
func convertToClippingInputs(items []models.ClippingItem) []models.ClippingInput {
	inputs := make([]models.ClippingInput, len(items))
	for i, item := range items {
		inputs[i] = item.ToClippingInput()
	}
	return inputs
}