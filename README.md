# CK-CLI [![codecov](https://codecov.io/gh/clippingkk/cli/branch/master/graph/badge.svg?token=68N24T6T9P)](https://codecov.io/gh/clippingkk/cli)

`ck-cli`(clippingkk-cli) is a TUI(Terminal User Interface) to parse `My Clippings.txt` that clippings in Amazon Kindle to user friendly data struct.

## Installation

download latest version from [release page](https://github.com/clippingkk/cli/releases) and add to `$PATH`

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

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)
