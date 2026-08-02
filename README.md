# CK-CLI [![codecov](https://codecov.io/gh/clippingkk/cli/branch/master/graph/badge.svg?token=68N24T6T9P)](https://codecov.io/gh/clippingkk/cli)

High-performance command-line tool for parsing Kindle clippings into structured JSON and syncing with ClippingKK web service.

[![video guide](http://img.youtube.com/vi/y4pgU9zIpxA/0.jpg)](http://www.youtube.com/watch?v=y4pgU9zIpxA "ClippingKK 命令行工具上传使用")

## Installation

go to release page and download one.

## Usage

```bash
# Parse to JSON
ck-cli parse -i "My Clippings.txt" -o output.json

# Parse from stdin
cat "My Clippings.txt" | ck-cli parse > output.json

# Extract unique titles
cat "My Clippings.txt" | ck-cli parse | jq -r .[].title | sort -u

# Extract highlights from a mounted Kindle
ck-cli sdr --path "/Volumes/Kindle/documents"

# Extract one book as ClippingItem JSON
ck-cli sdr --path "/path/to/Book.sdr" --json
```

**Parse options:**

- `-i, --input`: Input file path (default: stdin)
- `-o, --output`: Output file path or `http` for web sync (default: stdout)

**Output format:**

```json
[{
  "title": "Book Title",
  "content": "Highlighted text",
  "pageAt": "#78",
  "createdAt": "2019-03-27T19:57:26Z"
}]
```

### Kindle `.sdr` Highlights

Recent Kindle sidecars store annotations as positions rather than embedding the
highlighted words. The `sdr` command pairs `.azw3r` and `.yjr` sidecars with
their sibling AZW3/KF8 or KFX books and reconstructs the selected text locally:

```bash
# Recursively scan a Kindle root or documents directory
ck-cli sdr --path "/Volumes/Kindle/documents"

# Process a single sidecar directory or book
ck-cli sdr --path "/path/to/Book.sdr"
ck-cli sdr --path "/path/to/Book.azw3" --json
ck-cli sdr --path "/path/to/Book.kfx" --json
```

`--path` accepts a Kindle root/documents tree, one `.sdr` directory, or one
`.azw3`, `.azw`, KF8-containing `.mobi`, or `.kfx` file. The default output is
readable text grouped by book. `--json` emits the same `title`, `content`,
`pageAt`, and `createdAt` schema as `parse`; printed APNX or KFX navigation pages
are preferred, with the raw annotation position used as a fallback.

The implementation is read-only, offline, and written natively in Go—Python and
KindleUnpack are not runtime dependencies. It supports unencrypted AZW3/KF8
books with `.azw3r` sidecars and unencrypted KFX books with `.yjr` sidecars.
DRM-protected books and Mobi7 remain unsupported; `.yjf` reading statistics are
ignored because they do not contain highlight selections.

### Web Sync

```bash
# Authenticate (get token from https://clippingkk.annatarhe.com)
ck-cli login --token "YOUR_TOKEN"

# Sync to ClippingKK
ck-cli parse -i "My Clippings.txt" -o http
```

Configuration stored in `~/.ck-cli.toml`.

## Development

**Requirements:** Go 1.24+

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
- Native Kindle `.sdr` highlight extraction for `.azw3r` and `.yjr`
- Cross-platform (macOS, Linux, Windows)

## Contributing

See [CLAUDE.md](./CLAUDE.md) for development guidelines.

## License

[MIT](https://choosealicense.com/licenses/mit/)

The `.sdr` implementation was informed by the published format research in
[kindle-reading-dashboard](https://github.com/zevisvei/kindle-reading-dashboard)
and the container behavior documented by
[KindleUnpack](https://github.com/kevinhendricks/KindleUnpack). No code from
either GPLv3 project is bundled or required.

Binary Amazon Ion values in KFX containers are decoded with the Apache-2.0
licensed [Amazon Ion Go](https://github.com/amazon-ion/ion-go) library.
