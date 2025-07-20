# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`ck-cli` (ClippingKK CLI) is a high-performance command-line tool written in Go that parses Amazon Kindle's `My Clippings.txt` file into structured JSON format and syncs highlights to the ClippingKK web service.

### Key Features
- **Multi-language Support**: Parses clippings in Chinese, English, and other languages
- **Flexible I/O**: Read from files or stdin, output to files, stdout, or sync to web
- **High Performance**: Optimized parser handles large clipping files efficiently
- **Cloud Sync**: Direct integration with ClippingKK web service via GraphQL
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Structured Output**: Clean JSON format for easy processing

## Key Commands

### Building and Development
```bash
# Standard build
make build
# or
go build -o ck-cli ./cmd/ck-cli

# Install to GOPATH/bin
make install
# or
go install ./cmd/ck-cli

# Run tests
make test
# or
go test ./...

# Run tests with coverage
make test-coverage

# Format code
make fmt
# or
go fmt ./...

# Lint code
make lint
# or
golangci-lint run

# Run the CLI
./ck-cli [arguments]
# or
go run ./cmd/ck-cli [arguments]
```

### Cross-Platform Building
```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux
make build-windows  
make build-macos

# Install development dependencies
make dev-setup

# Release with GoReleaser
make release-dry    # dry run
make release        # actual release
```

### Testing and Development
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestName ./internal/parser

# Run benchmarks
make bench
# or
go test -bench=. ./...

# Test with example data
make run-example
make test-parse-stdin
```

### Running the Application
```bash
# Parse a Kindle clippings file to JSON
ck-cli parse --input "/path/to/My Clippings.txt" --output "/path/output.json"

# Parse from stdin to stdout
cat "My Clippings.txt" | ck-cli parse

# Sync to ClippingKK web service (requires login first)
ck-cli login --token "YOUR_TOKEN"
ck-cli parse --input "/path/to/My Clippings.txt" --output http

# Compose with Unix tools
cat ./My\ Clippings.txt | ck-cli parse | jq .[].title | sort | uniq
```

## Architecture and Code Structure

### Project Structure
```
cmd/ck-cli/         # Main CLI application entry point
internal/
├── commands/       # CLI command implementations (login, parse)
├── config/         # Configuration management (TOML files)
├── http/           # HTTP client and GraphQL integration
├── models/         # Data models (ClippingItem, etc.)
└── parser/         # Kindle clippings parser (core logic)
```

### Core Components

1. **Main CLI (`cmd/ck-cli/main.go`)**: Application entry point
   - Uses `urfave/cli/v2` framework for command structure
   - Handles graceful shutdown with context
   - Version and build info injection

2. **Parser Module (`internal/parser/parser.go`)**: The heart of the application
   - Handles multi-language parsing (Chinese, English)
   - Uses regex patterns to extract clipping components  
   - Converts Kindle's format to `ClippingItem` structs
   - Key struct: `ClippingItem` with fields: `Title`, `Content`, `PageAt`, `CreatedAt`

3. **HTTP/GraphQL Integration (`internal/http/client.go`)**:
   - Syncs parsed clippings to ClippingKK web service
   - Uses GraphQL mutations for data upload
   - Handles chunked uploads with concurrency control
   - Proper error handling and retry logic

4. **Configuration (`internal/config/config.go`)**:
   - Manages `.ck-cli.toml` in user's home directory
   - Stores HTTP endpoint and authentication headers
   - TOML format with `pelletier/go-toml/v2`

5. **Commands (`internal/commands/`)**: 
   - `login.go`: Authentication flow and token management
   - `parse.go`: Main parsing and output logic
   - Clean separation of CLI logic from business logic

### Data Flow
1. Input: Kindle's "My Clippings.txt" file (UTF-8 encoded)
2. Processing: Parser extracts structured data using language-specific regex patterns
3. Output: JSON array of clipping objects or direct sync to web service

### Key Technical Details

- **CLI Framework**: Uses `urfave/cli/v2` for robust command handling
- **HTTP Client**: Custom HTTP client with proper context handling
- **Concurrency**: Controlled concurrent uploads with semaphores
- **Error Handling**: Structured error handling with context
- **Date Parsing**: Handles multiple date formats across languages
- **Regex Patterns**: Language-specific patterns for parsing clipping headers
- **JSON Serialization**: Standard library JSON with custom marshaling

### Testing Approach

- Unit tests in `*_test.go` files alongside source code
- Test fixtures cover edge cases: multiple languages, special characters, various formats
- Table-driven tests for comprehensive coverage
- Integration tests for command-line interface

### Build and Release

- **GoReleaser**: Multi-platform builds and releases
- **Docker**: Container builds for easy deployment
- **Package Managers**: Homebrew, APT, RPM, AUR support
- **CI/CD**: GitHub Actions for testing and releases

### Dependencies

- `github.com/urfave/cli/v2`: CLI framework
- `github.com/pelletier/go-toml/v2`: TOML configuration
- Standard library for HTTP, JSON, regex, time handling

### Important Patterns

- Clean architecture with internal packages
- Interface-based design for testability
- Context-aware operations for cancellation
- Structured logging and error handling
- Configuration with sensible defaults

## Development Guidelines

### Code Style
- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `golangci-lint` for comprehensive linting
- Write tests for all new functionality
- Document exported functions and types

### Testing
- Write table-driven tests where appropriate
- Include both positive and negative test cases
- Test error conditions and edge cases
- Use test fixtures for parser testing

### Performance
- Parser optimized for large clipping files
- Concurrent HTTP uploads for better sync performance
- Minimal memory allocation in hot paths

## Commit Guidelines

- Follow [Conventional Commits](https://www.conventionalcommits.org/) format
- Use scopes: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build`
- Examples:
  - `feat(parser): add support for Japanese clippings`
  - `fix(http): handle network timeouts properly`
  - `refactor(config): simplify TOML configuration`
  - `docs: update installation instructions`
  - `test(parser): add edge case for malformed dates`
  - `perf(http): optimize concurrent upload performance`
  - `build: update Go version to 1.21`

### Pull Request Process

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`make test`)
5. Run linter (`make lint`)
6. Format code (`make fmt`)
7. Commit using Conventional Commits format
8. Push to your branch and open a Pull Request

## Error Handling Best Practices

- Always wrap errors with context using `fmt.Errorf("context: %w", err)`
- Check errors immediately after function calls
- Use early returns for error conditions
- Provide meaningful error messages that help debugging

## Common Tasks

### Adding a New Command
1. Create a new file in `internal/commands/`
2. Implement the command following the pattern in existing commands
3. Register the command in `cmd/ck-cli/main.go`
4. Add tests in `internal/commands/*_test.go`
5. Update documentation

### Modifying the Parser
1. Update regex patterns in `internal/parser/parser.go`
2. Add test cases to `internal/parser/parser_test.go`
3. Test with real clipping files in multiple languages
4. Ensure backward compatibility

### Updating GraphQL Schema
1. Modify queries/mutations in `internal/http/client.go`
2. Update request/response structs
3. Test against the live API
4. Handle API errors gracefully