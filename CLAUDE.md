# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ClippingKK CLI (`ck-cli`) is a Rust-based Terminal User Interface tool that parses Amazon Kindle's "My Clippings.txt" file into structured JSON format. It supports synchronization with the ClippingKK web service for cloud storage of reading highlights.

## Key Commands

### Building and Development
```bash
# Standard build
cargo build

# Release build (optimized)
cargo build --release

# Run tests
cargo test

# Run benchmarks
cargo bench

# Format code
cargo fmt

# Lint code
cargo clippy --all-features

# Run the CLI
cargo run -- [arguments]
```

### Testing
```bash
# Run all tests with verbose output
cargo test --verbose --all-features --workspace

# Run specific test
cargo test test_name

# Generate code coverage (requires cargo-tarpaulin)
cargo +nightly tarpaulin --verbose --all-features --workspace --timeout 120 --out Xml
```

### Running the Application
```bash
# Parse a Kindle clippings file to JSON
cargo run -- parse -i "/path/to/My Clippings.txt" -o "/path/output.json"

# Parse from stdin to stdout
cat "My Clippings.txt" | cargo run -- parse

# Sync to ClippingKK web service (requires login first)
cargo run -- login --token "YOUR_TOKEN"
cargo run -- parse --input "/path/to/My Clippings.txt" --output http
```

## Architecture and Code Structure

### Core Components

1. **Parser Module (`src/parser.rs`)**: The heart of the application
   - Handles multi-language parsing (Chinese, English, Japanese)
   - Uses regex patterns to extract clipping components
   - Converts Kindle's format to `TClippingItem` structs
   - Key struct: `TClippingItem` with fields: `title`, `content`, `pageAt`, `createdAt`

2. **HTTP/GraphQL Integration (`src/http.rs`, `src/graphql.rs`)**: 
   - Syncs parsed clippings to ClippingKK web service
   - Uses GraphQL mutations for data upload
   - Handles authentication via Bearer tokens

3. **Configuration (`src/config.rs`)**: 
   - Manages `.ck-cli.toml` in user's home directory
   - Stores HTTP endpoint and authentication headers

4. **Authentication (`src/auth.rs`)**: 
   - Interactive login flow with QR code display
   - Token management for API access

### Data Flow
1. Input: Kindle's "My Clippings.txt" file (UTF-8 encoded)
2. Processing: Parser extracts structured data using language-specific regex patterns
3. Output: JSON array of clipping objects or direct sync to web service

### Key Technical Details

- **Async Runtime**: Uses Tokio for async operations
- **Error Handling**: Returns `Result<T, Box<dyn Error>>` for main operations
- **Date Parsing**: Handles multiple date formats across languages using chrono
- **Regex Patterns**: Language-specific patterns for parsing clipping headers
- **JSON Serialization**: Uses serde for type-safe JSON handling

### Testing Approach

- Unit tests in `tests/tests.rs` validate parsing against fixture files
- Fixtures cover edge cases: multiple languages, special characters, various formats
- Benchmarks in `benches/parse.rs` measure parsing performance using Criterion

### CI/CD Workflows

1. **Code Coverage**: Runs on master branch and PRs, reports to Codecov
2. **Benchmark Comparison**: Compares performance metrics on PRs
3. **Release Builds**: Automated multi-platform builds for releases

### Important Patterns

- The parser uses a state machine approach to handle multi-line clippings
- Language detection is based on regex pattern matching of clipping headers
- HTTP client reuses connections for batch uploads
- Configuration persists between sessions in TOML format