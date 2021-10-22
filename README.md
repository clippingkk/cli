# clippingkk-cli [![codecov](https://codecov.io/gh/clippingkk/cli/branch/master/graph/badge.svg?token=68N24T6T9P)](https://codecov.io/gh/clippingkk/cli)

a cli to parse or upload `My Clippings.txt`

## Usage

```bash
ck-cli -i /path/to/My Clippings.txt -o /path/output.json
cat My Clippings.txt | ck-cli -o /path/output.json
cat My Clippings.txt | ck-cli > file.json
```

