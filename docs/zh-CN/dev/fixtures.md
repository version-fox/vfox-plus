# vfox 输出夹具

> [English](../../en/dev/fixtures.md)

vfox 打印面向人类的文本，GUI 负责解析。本目录存放真实 vfox 运行捕获的文本，
及正确解析器应产出的结果。解析器在单元测试中跑这些文件，与 vfox 输出含义保持一致。

## 布局

```mermaid
---
config:
  theme: base
  themeVariables:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', 'Noto Sans', 'Noto Sans CJK SC', 'PingFang SC', 'Microsoft YaHei', sans-serif"
    lineColor: "#7C79A8"
    primaryColor: "#FFE9D0"
    primaryBorderColor: "#F28C1B"
    primaryTextColor: "#3B2200"
    clusterBkg: "#F4F2FF"
    clusterBorder: "#B9B3F5"
    titleColor: "#1E1470"
    edgeLabelBackground: "#FFFFFF"
  flowchart:
    curve: basis
    wrappingWidth: 320
    subGraphTitleMargin:
      top: 10
      bottom: 8
---
flowchart TB
    accTitle: vfox 输出夹具
    accDescr: vfxtool capture 从已安装的 vfox 写出夹具；vfxtool fake-vfox 回放它们，解析器单元测试读取它们。
    vfox(["已安装的 <b>vfox</b>"])
    subgraph fx["<b>fixtures/vfox-#lt;版本#gt;/</b>"]
        direction LR
        cases("<b>cases.json</b><br/>命令、exit code<br/>与期望结果") ~~~ out("<b>#lt;用例#gt;.stdout</b><br/>命令标准输出") ~~~ err("<b>#lt;用例#gt;.stderr</b><br/>命令标准错误")
    end
    fake(["<b>vfxtool fake-vfox</b><br/>vfox 替身"])
    tests(["解析器<br/>单元测试"])
    vfox -->|"<code>vfxtool capture</code>"| fx
    fx -->|"回放"| fake
    fx -->|"读取"| tests

    classDef real fill:#17123F,stroke:#17123F,color:#FFFFFF,stroke-width:1.5px
    classDef file fill:#FFE9D0,stroke:#F28C1B,color:#3B2200,stroke-width:1.5px,rx:12,ry:12
    classDef user fill:#E4E0FF,stroke:#5A4BE8,color:#1E1470,stroke-width:1.5px
    class vfox real
    class cases,out,err file
    class fake,tests user
    style fx fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
    linkStyle default stroke-width:2px
```

## 输出如何捕获

`vfxtool capture` 跑每个命令时：

- `NO_COLOR=1`，stdin 与 stdout 都不是终端——正是 GUI 跑 vfox 的方式（见
  [CLI 契约(vfox-cli-contract.md)）；
- `HOME=/tmp/vfox-sb/home` 一次性沙箱，不设 `VFOX_HOME`，故不碰真实 vfox 安装，
  捕获路径稳定；
- 工作目录 `/`。

部分版本靠建目录伪造，让 `vfox list` 打印的树每插件多版本，无需下载。

部分捕获含回车、转义序列或字段分隔符 `0x1F`。为保持纯文本，`vfxtool capture` 把
tab 与换行外的控制字符写成 `\x` 加两位小写十六进制（回车变 `\x0d`），反斜杠写成
`\\`。解析前按从左到右单遍替换两种转义还原，得到 vfox 打印的精确字节。

## `cases.json`

| 字段 | 含义 |
| --- | --- |
| `name` | `.stdout` 与 `.stderr` 的基名。 |
| `command` | 传给 `vfox` 的参数。 |
| `environment` | 除上表外额外设置的变量。 |
| `exitCode` | 命令 exit code。 |
| `expect` | 期望解析结果，按解析器键。空对象表示该用例仅作参考保留。 |

各解析器期望结果：

| 键 | 输入 | 期望值 |
| --- | --- | --- |
| `version` | stdout | 版本号。 |
| `current` | stdout | 插件名到当前版本；无生效版本为 `null`。 |
| `list` | stdout | 插件名到已安装版本，按输出顺序。 |
| `search` | stdout | 版本 `count` 与按序号选的 `items`。 |
| `available` | stdout | 插件 `count` 与按序号选的 `items`。 |
| `pluginInfo` | stdout | `--format` 模板各字段。 |
| `installPath` | stdout | 某版本目录，`notfound` 为 `null`。 |
| `config` | stdout | 每条 `key = value`，按输出顺序。 |
| `finalProgress` | stderr 行 | 末个下载进度（0–1），无则 `null`。 |
| `error` | stdout + stderr | 展示给用户的一行错误摘要。 |

解析器跳过不认识的行，不失败；错误摘要规则见
[CLI 契约(vfox-cli-contract.md#错误)。

## 跟进 vfox 新版

1. 装新版 vfox 跑 `vfxtool capture`。需联网，约一分钟，写入 `vfox-<版本>/`。
2. 从旧版复制 `cases.json`，新旧输出 `diff -r` 对比并调整期望值。输出不同保留旧版
   目录，解析器对两版都测（vfox 1.0.13 把错误从标准输出搬到标准错误）。
3. 跑解析器测试，如 `cd mygo && go test ./core/...`。
4. 输出格式变化记入
   [CLI 契约(vfox-cli-contract.md)。

## 假 vfox

`vfxtool fake-vfox` 对 GUI 表现得像 vfox 1.0.13：只读命令用 `vfox-1.0.13/`（或
`VFOX_PLUS_FIXTURES`）捕获回答，假装在标准错误打进度条安装版本，无 hook 环境跑
`vfox use` 即按 vfox 失败。端到端工具把 vfxtool 二进制复制为 `vfox` 文件安装：
以该文件名执行即回答 vfox 命令而非跑子命令（见 `cmd/vfxtool`）。它还检查 CLI 契约
对调用者的要求：`NO_COLOR=1`、无激活 shell 的 `__VFOX_PID`、非项目命令工作目录 `/`。
无状态。端到端测试用它不碰真实 vfox 试应用，端到端测试与 `vfxtool measure` 都用。
