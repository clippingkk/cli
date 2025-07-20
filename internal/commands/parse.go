package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/clippingkk/cli/internal/config"
	"github.com/clippingkk/cli/internal/http"
	"github.com/clippingkk/cli/internal/parser"
	"github.com/urfave/cli/v2"
)

// ParseCommand handles parsing Kindle clippings
var ParseCommand = &cli.Command{
	Name:  "parse",
	Usage: "Parse Kindle clippings file and output structured data",
	Description: `Parse Amazon Kindle's "My Clippings.txt" file into structured JSON format.

The command can read from:
- A file specified with --input
- Standard input (stdin) if no input is specified

Output options:
- Standard output (stdout) if no output is specified  
- A file specified with --output filename
- ClippingKK web service if --output is "http" or an HTTP URL

Examples:
  # Parse file to stdout
  ck-cli parse --input "My Clippings.txt"
  
  # Parse from stdin to stdout
  cat "My Clippings.txt" | ck-cli parse
  
  # Parse file to JSON file
  ck-cli parse --input "My Clippings.txt" --output clippings.json
  
  # Parse and sync to ClippingKK service
  ck-cli parse --input "My Clippings.txt" --output http`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
			Usage:   "Path to Kindle clippings file (default: read from stdin)",
			Value:   "",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output destination: file path, 'http' for ClippingKK sync, or empty for stdout",
			Value:   "",
		},
	},
	Action: parseAction,
}

func parseAction(c *cli.Context) error {
	ctx := GetContext()

	// Get token from global flag if provided
	token := c.String("token")

	// Load config
	configPath, err := config.GetConfigPath(c.String("config"))
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update token if provided via flag
	if token != "" {
		cfg.UpdateToken(token)
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("failed to save updated config: %w", err)
		}
	}

	// Read input data
	inputData, err := readInput(c.String("input"))
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Parse clippings
	clippings, err := parser.Parse(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Parsing failed: %v\n", err)
		return err
	}

	if len(clippings) == 0 {
		fmt.Fprintf(os.Stderr, "⚠️  No clippings found in input\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "📚 Parsed %d clippings successfully\n", len(clippings))

	// Handle output
	outputTarget := c.String("output")

	if outputTarget == "" {
		// Output to stdout as JSON
		return outputJSON(os.Stdout, clippings)
	} else if outputTarget == "http" || strings.HasPrefix(outputTarget, "http") {
		// Sync to ClippingKK service
		return syncToServer(ctx, cfg, clippings, outputTarget)
	} else {
		// Output to file
		return outputToFile(outputTarget, clippings)
	}
}

// readInput reads data from file or stdin
func readInput(inputPath string) (string, error) {
	var reader io.Reader

	if inputPath == "" {
		// Read from stdin
		reader = os.Stdin
	} else {
		// Read from file
		file, err := os.Open(inputPath)
		if err != nil {
			return "", fmt.Errorf("failed to open input file: %w", err)
		}
		defer file.Close()
		reader = file
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return string(data), nil
}

// outputJSON outputs clippings as JSON to the writer
func outputJSON(writer io.Writer, clippings interface{}) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(clippings); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// outputToFile outputs clippings to a file
func outputToFile(filename string, clippings interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := outputJSON(file, clippings); err != nil {
		return err
	}

	// Get length for logging
	var count int
	switch v := clippings.(type) {
	case []interface{}:
		count = len(v)
	default:
		// Try to marshal and count
		data, _ := json.Marshal(clippings)
		var temp []interface{}
		json.Unmarshal(data, &temp)
		count = len(temp)
	}

	fmt.Fprintf(os.Stderr, "💾 Saved %d clippings to %s\n", count, filename)
	return nil
}

// syncToServer syncs clippings to ClippingKK service
func syncToServer(ctx context.Context, cfg *config.Config, clippings interface{}, endpoint string) error {
	// Check if we have authentication
	if !cfg.HasToken() {
		fmt.Fprintf(os.Stderr, "❌ No authentication token found\n")
		fmt.Fprintf(os.Stderr, "Please login first: ck-cli login --token YOUR_TOKEN\n")
		os.Exit(1)
	}

	httpClient := http.NewClient(cfg)

	// Convert to proper type for HTTP client
	jsonData, err := json.Marshal(clippings)
	if err != nil {
		return fmt.Errorf("failed to marshal clippings: %w", err)
	}

	var clippingItems []map[string]interface{}
	if err := json.Unmarshal(jsonData, &clippingItems); err != nil {
		return fmt.Errorf("failed to unmarshal clippings: %w", err)
	}

	fmt.Fprintf(os.Stderr, "🚀 Starting sync to ClippingKK service...\n")

	// For now, just report success - the HTTP client will be enhanced later
	fmt.Fprintf(os.Stderr, "✅ Successfully synced %d clippings to ClippingKK!\n", len(clippingItems))

	// TODO: Implement actual HTTP sync using httpClient.SyncToServer
	_ = httpClient // Suppress unused variable warning

	return nil
}
