# clippingkk-cli [![codecov](https://codecov.io/gh/clippingkk/cli/branch/master/graph/badge.svg?token=68N24T6T9P)](https://codecov.io/gh/clippingkk/cli)

命令行解析 kindle 文件 `My Clippings.txt`


手动从 `release` 中下载对应的二进制文件，例如：

```
curl -L \
	-o ck-cli \
	https://github.com/clippingkk/cli/releases/download/v1.0.1/clippingkk-cli_1.0.1_darwin_amd64
```

## 使用方式

参数:

- `--i` optional. 表示 **input** 填入希望解析的文件路径. 如无该参数则通过 stdin 获取
- `--o` optional. 表示 **output** 填入输出文件路径. 如无该参数则输出至 stdout

例子：

```bash
ck-cli -i /path/to/My Clippings.txt -o /path/output.json
cat My Clippings.txt | ck-cli -o /path/output.json
cat My Clippings.txt | ck-cli > file.json
```

```bash
cat ./core/clippings_en.txt | go run cmd/cli.go | jq .[15]
```

```json
{
  "title": "凤凰项目 一个IT运维的传奇故事",
  "content": "创建约束理论的艾利·高德拉特告诉我们，在瓶颈之外的任何地方作出的改进都是假象。难以置信，但千真万确！在瓶颈之后作出任何改进都是徒劳的，因为只能干等着瓶颈把工作传送过来。而在瓶颈之前作出的任何改进则只会导致瓶颈处堆积更多的库存",
  "pageAt": "78",
  "createdAt": "2019-03-27T19:57:26Z"
}
```

可以组合更多 *nix 的命令进行更多处理，例如：

```bash
cat ./core/clippings_en.txt | ck-cli | jq .[].title | sort | uniq
# 返回内容:
# "Bad Blood: Secrets and Lies in a Silicon Valley Startup"
# "凤凰项目 一个IT运维的传奇故事"
# "论法的精神"
```

