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
  [CLI 契约](vfox-cli-contract.md)）；
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
| `doctor` | stdout | `vfox doctor --json` 的 `ok`、`count` 与 `findings`。 |
| `aliases` | stdout | 每个别名的 `sdk`、`alias` 与 `target`。 |
| `versionDetail` | stdout | `vfox info --json` 的 `name`、`version`、`installed`、`origin`、`target` 与 `home`。 |

解析器跳过不认识的行，不失败；错误摘要规则见
[CLI 契约](vfox-cli-contract.md#错误)。

## version-fox 组织的插件

`vfxtool capture` 最后才添加 vfox-java 与 vfox-flutter，这样其他用例的输出里就不会列出它们，并捕获它们的
搜索：`search-java`（OpenJDK，不带参数的列表）、`search-java-tem`（一个发行版）、`search-java-all`
（全部发行版，2683 个版本）与 `search-flutter`（Linux x64 上 460 个版本：备注是通道，附带工具是 Dart SDK，
还有架构后缀与 OpenHarmony 构建）。它们用 vfox-java 0.6.5、vfox-flutter 0.2.5 与 vfox-nodejs 0.5.4 捕获，`core/profile` 的
档案据此测试：`search-java-all` 里每个版本名带的简称，档案都认识。

`doctor`、`doctor-json`（无安装时、在已激活的 shell 里的 `vfox doctor [--json]`，由 `__VFOX_SHELL` 表明）、`alias-list`（无别名）与 `info-json`
（`vfox info --json nodejs`）覆盖 vfox 1.1.0 新增只读命令。`doctor-json-not-activated` 是在未激活的 shell 里（即应用运行 vfox 的方式）的 `vfox doctor --json`：它发现 `path.not-activated` 并以 1 退出；`__VFOX_SHELL` 为空时假 vfox 用它回答。`vfox list` 带 `(linked → …)` 与 `[alias: …]` 后缀时解析只取版本号。

它们的版本是伪造的，和 `list` 里的 Node.js、Maven 一样：工具为每个版本建一个目录，不下载任何东西。
`list-profiles` 与 `current-profiles` 是一台装了 Java（正在用 21.0.12.1+1-tem，另有 25.0.4+9-librca-fx 与
25.0.2+10）和 Flutter（正在用 3.47.7-x64，另有 3.44.9-x64），同时装着 Node.js 与 Maven 的机器的 `list` 与
`current`；`info-java` 与 `info-flutter` 是这两个插件对自己的描述。设置了 `VFOX_PLUS_FAKE_INVENTORY=profiles`
时，假 vfox 用它们回答 `list` 与 `current`，这两个插件的截图需要它；搜索与插件信息则始终由它回答。

`vfxtool capture` 也最后才添加 vfox-python，所用的是 `VFOX_PYTHON_PLUGIN` 指定的插件 zip：采集时注册表列的是
0.6.3，而配置档案跟随的是 0.6.4，zip 正是它。`search-python` 在 Linux x64 上列出 999 个版本，其中 GraalPy 构建带有
备注 `GraalPy` 与 `GraalPy Community`，配置档案的筛选靠它们区分。`info-python` 是插件对自己的描述。`list-python`
与 `current-python` 是 Python 页面所用的机器：装着 Java、Flutter、Node.js 与 Maven 的那台机器，另外装着 Python
3.15.2 与 3.14.4，其中 3.15.2 正在用。设置了 `VFOX_PLUS_FAKE_INVENTORY=python` 时，假 vfox 为 `list` 与 `current`
返回它们。`cases.json` 列出了解析器必须从它们读出的内容。

`vfxtool capture -plugins` 为 profile 里尚无捕获的八个插件补上捕获：etcd、Erlang/OTP、Deno、Groovy、Scala、Zig、clang 与 Elixir。每个插件捕获 `search-<插件>` 与 `info-<插件>`；另捕获它们那台机器的 `list-plugins` 与 `current-plugins`：Erlang/OTP 29.1.2 正在用，clang 23.1.6 与 etcd 3.7.4 也已安装。每个插件都从 `VFOX_PLUGIN_ZIPS` 指定的 zip（`<目录>/vfox-<插件>.zip`）添加，即 profile 所跟随的发行版。该模式把这些捕获加到已有的夹具里，其余保持不变。设置了 `VFOX_PLUS_FAKE_INVENTORY=plugins` 时，假 vfox 为 `list` 与 `current` 返回那台机器的输出；端到端场景 `plugins` 需要它。`cases.json` 记录它们的期望值。

## 跟进 vfox 新版

1. 装新版 vfox 跑 `vfxtool capture`。需联网，约一分钟，写入 `vfox-<版本>/`。
2. 从旧版复制 `cases.json`，新旧输出 `diff -r` 对比并调整期望值。输出不同保留旧版
   目录，解析器对两版都测（vfox 1.0.13 把错误从标准输出搬到标准错误）。
3. 跑解析器测试，如 `cd mygo && go test ./core/...`。
4. 输出格式变化记入
   [CLI 契约](vfox-cli-contract.md)。

## 假 vfox

`vfxtool fake-vfox` 对 GUI 表现得像 vfox 1.1.0：只读命令用 `vfox-1.1.0/`（或
`VFOX_PLUS_FIXTURES`）捕获回答，假装在标准错误打进度条安装版本，无 hook 环境跑
`vfox use` 即按 vfox 失败。端到端工具把 vfxtool 二进制复制为 `vfox` 文件安装：
以该文件名执行即回答 vfox 命令而非跑子命令（见 `cmd/vfxtool`）。它还检查 CLI 契约
对调用者的要求：`NO_COLOR=1`、无激活 shell 的 `__VFOX_PID`、非项目命令工作目录 `/`。
无状态。端到端测试用它不碰真实 vfox 试应用，端到端测试与 `vfxtool measure` 都用。
