# CK-CLI [![codecov](https://codecov.io/gh/clippingkk/cli/branch/master/graph/badge.svg?token=68N24T6T9P)](https://codecov.io/gh/clippingkk/cli)

High-performance command-line tool for parsing Kindle clippings into structured JSON and syncing with ClippingKK web service.

[![video guide](http://img.youtube.com/vi/y4pgU9zIpxA/0.jpg)](http://www.youtube.com/watch?v=y4pgU9zIpxA "ClippingKK 命令行工具上传使用")

## Installation

### Homebrew (macOS/Linux)
```bash
brew install clippingkk/ck-cli/ck-cli
```

### Binary
Download from [releases](https://github.com/clippingkk/cli/releases) or use package managers:
- **Debian/Ubuntu**: `sudo dpkg -i ck-cli_*.deb`
- **RHEL/Fedora**: `sudo rpm -i ck-cli_*.rpm`
- **Arch**: `yay -S ck-cli-bin`

### Go
```bash
go install github.com/clippingkk/cli/cmd/ck-cli@latest
```

### Docker
```bash
docker pull ghcr.io/clippingkk/ck-cli:latest
```

## Usage

```bash
# Parse to JSON
ck-cli parse -i "My Clippings.txt" -o output.json

# Parse from stdin
cat "My Clippings.txt" | ck-cli parse > output.json

# Extract unique titles
cat "My Clippings.txt" | ck-cli parse | jq -r .[].title | sort -u
```

**Options:**
- `-i, --input`: Input file path (default: stdin)
- `-o, --output`: Output file path or `http` for web sync (default: stdout)

**Output format:**
```json
[{
  "title": "Book Title",
  "content": "Highlighted text",
  "pageAt": "78",
  "createdAt": "2019-03-27T19:57:26Z"
}]
```
### Web Sync

```bash
# Authenticate (get token from https://clippingkk.annatarhe.com)
ck-cli login --token "YOUR_TOKEN"

# Sync to ClippingKK
ck-cli parse -i "My Clippings.txt" -o http
```

Configuration stored in `~/.ck-cli.toml`.

## Development

**Requirements:** Go 1.21+

```bash
git clone https://github.com/clippingkk/cli.git
cd cli
make build    # Build binary
make test     # Run tests
make lint     # Run linter
```

See [Makefile](./Makefile) for all commands.

## Features

- Multi-language parsing (Chinese, English, etc.)
- Flexible I/O (files, stdin/stdout, web sync)
- High-performance processing of large files
- Direct ClippingKK web service integration
- Cross-platform (macOS, Linux, Windows)

## Contributing

Contributions welcome. Please follow [Conventional Commits](https://www.conventionalcommits.org/) format:
- `feat(scope): description`
- `fix(scope): description`
- `docs: description`

See [CLAUDE.md](./CLAUDE.md) for development guidelines.

## License
[MIT](https://choosealicense.com/licenses/mit/)
