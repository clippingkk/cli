# clippingkk-cli [![codecov](https://codecov.io/gh/clippingkk/cli/branch/master/graph/badge.svg?token=68N24T6T9P)](https://codecov.io/gh/clippingkk/cli)

命令行解析 kindle 文件 `My Clippings.txt`


手动从 `release` 中下载对应的二进制文件，例如：

```
curl -L \
	-o ck-cli \
	https://github.com/clippingkk/cli/releases/download/v1.0.1/clippingkk-cli_1.0.1_darwin_amd64
```

## 使用方式

```bash
ck-cli -i /path/to/My Clippings.txt -o /path/output.json
cat My Clippings.txt | ck-cli -o /path/output.json
cat My Clippings.txt | ck-cli > file.json
```

遵循 unix pipe & redirect 规范

```bash
cat ./core/clippings_en.txt | ck-cli | jq .[].title | uniq
# 返回内容:
# "Bad Blood: Secrets and Lies in a Silicon Valley Startup"
# "论法的精神"
# "凤凰项目 一个IT运维的传奇故事"
# "论法的精神"
```

