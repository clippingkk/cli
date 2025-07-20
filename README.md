# CK-CLI [![codecov](https://codecov.io/gh/clippingkk/cli/branch/master/graph/badge.svg?token=68N24T6T9P)](https://codecov.io/gh/clippingkk/cli)

`ck-cli` (ClippingKK CLI) is a high-performance command-line tool written in Go that parses Amazon Kindle's `My Clippings.txt` file into structured JSON format and syncs highlights to the ClippingKK web service.

[![video guide](http://img.youtube.com/vi/y4pgU9zIpxA/0.jpg)](http://www.youtube.com/watch?v=y4pgU9zIpxA "ClippingKK 命令行工具上传使用")

## Installation

### Homebrew (macOS/Linux)
```bash
brew install clippingkk/ck-cli/ck-cli
```

### Direct Download
Download the latest version from the [release page](https://github.com/clippingkk/cli/releases) and add to your `$PATH`.

### Package Managers

**Debian/Ubuntu:**
```bash
# Download .deb package from releases page
sudo dpkg -i ck-cli_*_linux_amd64.deb
```

**Red Hat/CentOS/Fedora:**
```bash
# Download .rpm package from releases page  
sudo rpm -i ck-cli_*_linux_amd64.rpm
```

**Arch Linux:**
```bash
yay -S ck-cli-bin
```

### Go Install
If you have Go installed:
```bash
go install github.com/clippingkk/cli/cmd/ck-cli@latest
```

### Docker
```bash
docker pull ghcr.io/clippingkk/ck-cli:latest
# Use with volume mount for file access
docker run --rm -v $(pwd):/data ghcr.io/clippingkk/ck-cli:latest parse --input /data/My\ Clippings.txt
```

## Usage

### Parse

```bash
ck-cli parse -i /path/to/My Clippings.txt -o /path/output.json
cat My Clippings.txt | ck-cli parse -o /path/output.json
cat My Clippings.txt | ck-cli parse > file.json
```

Arguments:

|    key |   value |   type |   desc |
| ------ | ------- | ------ | ------ |
| input(-i) | /path/to/My Clippings.txt | file path | if empty it will read from stdin |
| output(-o) | /path/to/output.json | file path | if empty it will put to stdout |

Result:

output format is json. and it will be like this:

```json
[{
  "title": "凤凰项目 一个IT运维的传奇故事",
  "content": "创建约束理论的艾利·高德拉特告诉我们，在瓶颈之外的任何地方作出的改进都是假象。难以置信，但千真万确！在瓶颈之后作出任何改进都是徒劳的，因为只能干等着瓶颈把工作传送过来。而在瓶颈之前作出的任何改进则只会导致瓶颈处堆积更多的库存",
  "pageAt": "78",
  "createdAt": "2019-03-27T19:57:26Z"
}]
```

You can compose any *nix command to process the result, like this:

```bash
cat ./core/clippings_en.txt | ck-cli parse | jq .[].title | sort | uniq
# result text should be like this:
# "Bad Blood: Secrets and Lies in a Silicon Valley Startup"
# "凤凰项目 一个IT运维的传奇故事"
# "论法的精神"
```
### Compose with ClippingKK Http Service

you can pass cli token to local config

```bash
ck-cli --token "COPY FROM https://clippingkk.annatarhe.com" login
cat ~/.ck-cli.toml
```

You can also just parse file and put it to server with token for once:

```bash
ck-cli parse --input /path/to/My Clippings.txt --output http
```

the `http` in `output` is magic word and it will send parsed clippings to server.

you can manually define where should it send and the http request headers by edit config in `~/.ck-cli.toml`

If you want integration with CI service, you can set config as secret. and to do something you want

## Development

### Prerequisites
- Go 1.21 or later
- Make (optional, for convenience)

### Building from Source
```bash
# Clone the repository
git clone https://github.com/clippingkk/cli.git
cd cli

# Build the CLI
make build
# or
go build -o ck-cli ./cmd/ck-cli

# Run tests
make test
# or  
go test ./...
```

### Development Commands
```bash
make help           # Show all available commands
make build          # Build for current platform
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linter
make build-all      # Cross-compile for all platforms
make dev-setup      # Install development dependencies
```

### Project Structure
```
cmd/ck-cli/         # Main CLI application
internal/
├── commands/       # CLI command implementations
├── config/         # Configuration management
├── http/           # HTTP client and GraphQL integration
├── models/         # Data models
└── parser/         # Kindle clippings parser
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)
