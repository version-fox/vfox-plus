# 插件配置档案

> [English](../../en/dev/profiles.md)

对每个插件，vfox 列出、安装、选择和删除版本的方式都一样，但插件能做的不止这些，而且 vfox 的输出里看不出来：`vfox search java tem` 列出 Java 的一个发行版，而 `vfox search java` 只列出 OpenJDK；`nodejs@24` 安装最新的 Node.js 24；插件还会读取把下载换到镜像上的环境变量。**插件配置档案**（profile，下称档案）把这些写成数据：一份 JSON 文件，vfox+ 的所有界面都读它。适配一个插件，或插件的新版本，只改数据，不改应用。

格式定义在 [schema.json](../../../shared/profile/schema.json)，文件在 `"$schema"` 里写上它，编辑器就能据此提示和检查。

## 档案从哪里来

对已安装的插件，档案的每一部分取自下面第一个带有它的来源：

| 优先级 | 来源 | 位置 |
| --- | --- | --- |
| 1 | 用户 | 设置文件旁的 `profiles/<插件>.json`（Linux 为 `~/.config/vfox-plus/profiles/`，Windows 为 `%APPDATA%\vfox-plus\profiles\`） |
| 2 | 插件自带 | 已安装插件里的 `lib/vfox-plus.json`，vfox 打包插件时会带上它，随插件一起更新 |
| 3 | 在线目录 | 本仓库主分支上的 [`shared/profile`](../../../shared/profile) |
| 4 | 应用内置 | 应用发布时带上的同一批文件 |
| 5 | 推断 | 只用于 3 和 4 都没有描述的插件 |

每一部分整段取用：带 `variables` 的文件会替换后面来源的全部变量，指南、提示等其他部分保持不变。空列表（如 `"tips": []`）表示隐藏这一部分。

查找插件时先用 vfox 里的名称，找不到再用插件自己声明的名称，因此用别名添加的插件（`vfox add --alias`）仍能用上它的档案。

**在线目录。** 应用最多每天一次（上次失败则一小时后）经设置里的更新镜像（如 `https://gh-proxy.org/`）获取 `index.json`，其中列出每份档案及其文件的 SHA-256。只下载与内置不同的文件，按索引校验后存进用户缓存目录（`vfox-plus/profiles`）。当应用内置的档案发生变化时缓存作废，避免旧缓存盖过应用更新带来的新档案。

**推断。** 对目录和应用都没有描述的插件，其他来源没写的部分会推断出来：变量取自已安装插件里读取设置项的 `os.getenv` 调用（`VFOX_*`、镜像、基础地址、代理），版本文件取自 `metadata.lua` 的 `PLUGIN.legacyFilenames`，指南与文档取自 `vfox info` 的描述和主页。任何插件，只要档案里没有筛选项，都会按列出的版本生成（注记，以及共有的 `-后缀`）。

**能力**由插件文件决定：除非插件有 hooks 目录却没有 `hooks/detect.lua`（单文件插件则是 `main.lua` 里没有 `Detect` 函数），否则提供 `vfox link --detect`。

## 格式

档案是一个不超过 256 KiB 的 JSON 对象：

```json
{
  "$schema": "https://raw.githubusercontent.com/version-fox/vfox-plus/main/shared/profile/schema.json",
  "schema": 2,
  "plugin": "example",
  "variables": [
    {
      "key": "VFOX_EXAMPLE_MIRROR",
      "label": {"en": "Download mirror", "zh": "下载镜像"},
      "help": {"en": "A mirror of example.org/dist.", "zh": "example.org/dist 的镜像。"},
      "presets": [{"value": "https://mirror.example.org/dist", "name": "Example"}]
    }
  ],
  "guide": {"en": "Write the release as the list shows it.", "zh": "按列表中的样子写版本。"}
}
```

`plugin` 写明用户文件或目录文件描述的是哪个插件；插件自带的文件可以省略，它描述的就是所在的插件。不认识的字段会被忽略，因此更新格式的文件仍可读取。读不懂的部分只损失它自己：坏的条目丢掉该条目，整段没有可用内容时丢掉该段，改由下一个来源提供。

**文本**可以是一个字符串（所有语言都显示它），或按语言给出的对象：`{"en": "Mirror", "zh": "镜像"}`。应用支持英文和简体中文，缺中文时显示英文。

**平台。** 列表里的大部分条目可带 `"when": {"os": [...], "arch": [...]}`，取值同 `GOOS` 与 `GOARCH`（字符串或列表；缺省表示全部）。平台指 vfox 运行所在的平台。

| 部分 | 内容 |
| --- | --- |
| `search` | `vfox search` 里插件名后面的参数：`label`、`default`（不带参数时 vfox 列出的那个选项）、`choices`（`{value, name, maker, aliases, popular, everything}`）。带 `when` 的多个 search 让不同平台各用各的，取第一个匹配当前平台的 |
| `naming` | 版本名的读法（见下文），或 `"plain"` |
| `shorthands` | 除列表里的版本外，`vfox install <插件>@<名称>` 接受的名称：`{name, meaning}` |
| `variables` | 插件读取的环境变量（见下文） |
| `legacyFiles` | vfox 借助插件读取的版本文件 |
| `legacyStrategy` | vfox 的 `legacyVersionFile.strategy` 设置是否影响插件处理这些文件 |
| `facets` | 版本列表的筛选项：`{id, label, kind, values, pattern}`；`kind` 为 `note`（注记按逗号和分号拆开的部分）、`suffix`（版本号最后一个 `-` 之后的部分）或 `pattern`（`pattern` 的第一个分组或 `value` 分组在版本号里匹配到的内容） |
| `guide` | 几句话说明版本名怎么写 |
| `tips` | 首次安装前值得知道的事 |
| `docs` | 文档：`{title, url}`，`url` 为 `https://` 地址，或按语言给出 |

### 变量

`key`、`label`、`help`、`advanced`（只在特殊系统上需要）、`example`（空输入框里的提示）以及 `kind`：

| `kind` | 取值 | 字段 |
| --- | --- | --- |
| `url`（默认） | `http://` 或 `https://` 地址 | `presets`，只接受 `https://` |
| `choice` | 预设值之一 | `presets`：`{value, name}` |
| `toggle` | 开或关 | `on`（默认 `"1"`）、`off`（空表示不设置） |
| `text` | 一行文本 | `presets` |

### 版本名

`naming` 把版本名拆开显示，如 `21.0.12.1+1-tem` 显示 Temurin，`3.47.7-x64` 显示 x64：

```json
"naming": {
  "sources": [{"value": "tem", "name": "Temurin"}, {"value": "open", "name": "OpenJDK"}],
  "defaultSource": "open",
  "rules": [
    {"pattern": "^(?<version>.*?)(?:-(?<source>{sources}))?(?:-(?<flavor>fx))?$", "labels": {"fx": "JavaFX"}}
  ]
}
```

按顺序取第一个 `pattern`（Go 的正则表达式，RE2 语法）匹配版本名的规则。分组 `version` 是版本号，没有该分组时取整个名称；分组 `source` 指向 `sources` 之一，匹配不到时用 `defaultSource`；其他匹配到内容的命名分组各添加一个标记，名称按 `labels`（以小写后的匹配文本为键）给出。`flavors` 添加固定标记。`{sources}` 代表所有来源值组成的多选分支。

## 档案不能做什么

档案只是数据，无论来自哪里都遵守同样的规则：

- 不能提供改变程序运行方式的变量：`PATH`、`HOME` 及其他用户目录，shell（`BASH_ENV`、`PROMPT_COMMAND` 等），动态加载器（`LD_*`、`DYLD_*`），`VFOX_HOME` 与 `__VFOX_*`，语言运行时选项（`NODE_OPTIONS`、`JAVA_TOOL_OPTIONS`、`PYTHONPATH` 等），编译器和构建工具（`CC`、`MAKEFLAGS` 等），代理和证书，Git、npm 与访问令牌。完整列表见 [security.go](../../../mygo/core/profile/security.go)。这类设置由用户在设置页的环境变量里自己填写。
- 文档链接和 `url` 变量的预设地址只接受 `https://`。搜索选项和简写必须是单个命令行参数：不能以 `-` 开头，不能含空格。
- 应用没有内置的值（来自插件、在线目录或用户文件），设置前会展示并请用户确认。手动输入的值是用户自己的选择，不再确认。

## 描述你的插件

在插件里添加 `lib/vfox-plus.json`：vfox 打包插件时会带上 `lib`，档案随插件到达用户机器、离线可用，任何装了该插件的 vfox plus 都能用上。在本仓库的检出目录里对照插件检查它：

```bash
go -C shared/tooling run ./cmd/vfxtool profile lint --plugin /path/to/vfox-example
```

或在插件的 CI 里：

```yaml
jobs:
  profile:
    uses: version-fox/vfox-plus/.github/workflows/profile-lint.yml@main
```

应用读不懂的内容、同一平台上重复的键、插件除档案和文档外任何文件都没有提到的变量，都会让检查失败。

## 修改本仓库的档案

version-fox 组织的插件配置档案在 [`shared/profile/plugins`](../../../shared/profile/plugins)。修改后运行：

```bash
go -C shared/tooling run ./cmd/vfxtool profile lint --strict
go -C shared/tooling run ./cmd/vfxtool profile index
```

`index` 生成 `shared/profile/index.json`；它过期时 `core/profile` 的测试会失败。已安装的应用一天内就会从在线目录拿到改动，下个版本会内置它。
