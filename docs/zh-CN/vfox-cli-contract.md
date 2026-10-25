# vfox CLI 契约

> [English](../vfox-cli-contract.md)

本文规定 vfox-plus 如何驱动 vfox 命令行工具：跑哪些命令、什么环境、如何读取输出、
如何报告错误。描述 **vfox 1.0.13**，并注明 1.0.12 的不同；两版捕获输出都在
[`shared/fixtures`](../../shared/fixtures)，期望解析结果在 `cases.json`。

最低支持 vfox **1.0.0**（`.vfox.toml` 布局与下文非交互行为自此存在；`info --format`
自 0.8.0）。

## 运行 vfox

### 进程

- 直接 spawn 可执行文件，传参数向量。永不经 shell，避免名称与版本被解释。
- 立即关闭标准输入。vfox 发现不在终端即永不提问；开着 stdin 会让意外提问挂起。
- 标准输出与标准错误按 UTF-8 读取，坏字节替换。
- 先校验用户输入再传递：插件名匹配 `^[A-Za-z0-9][A-Za-z0-9._+-]*$`，版本不得以 `-`
  开头，不得含空白或 `@`，配置键为点分隔标识符。防止 `--all` 之类插件名的选项注入。

### 环境

从 GUI 环境出发，再：

| 变量 | 动作 | 原因 |
| --- | --- | --- |
| `__VFOX_SHELL`、`__VFOX_PID`、`__VFOX_CURTMPPATH`、`__VFOX_INITIALIZED` | 删除 | 激活 shell 设置。GUI 若从此类终端启动，否则会作用于该终端会话域。 |
| 用户变量（`extraEnvironment` 设置） | 添加 | `VFOX_NODEJS_MIRROR` 等下载镜像在 shell profile，桌面启动器不读。 |
| `NO_COLOR` | 置 `1` | 关掉多数输出颜色（https://no-color.org）。 |
| `__VFOX_SHELL` | 仅 `vfox use` 置 `bash` | 见[选择版本](#选择版本)。 |

其余透传，含 `VFOX_HOME`：设置时 vfox 把插件、缓存与配置放在该共享目录而非
`~/.vfox`。

### 工作目录

vfox 从工作目录读*项目*配置（`.vfox.toml`，启用时还有 `.nvmrc` 等旧版文件），不看父目录。
与项目无关的命令一律在不可能含此类文件的目录跑：**文件系统根**（Unix `/`，
Windows 系统盘根如 `C:\`）。不用家目录：asdf 用户家目录常有 `~/.tool-versions`。

项目域命令在项目目录跑。

### 会话目录

每次 vfox 调用创建 `<home>/.vfox/tmp/<yyyymmdd>-<pid>`（Unix `~/.vfox/tmp`，
Windows `%USERPROFILE%/.vfox/tmp`），`<pid>` 为父进程 PID。GUI 直接 spawn vfox，
每天共享一目录；vfox 清理旧的。

## 输出约定

- 命令返回的错误 exit code 1：vfox 1.0.13 起打印到**标准错误**，1.0.12 及以前打印到
  **标准输出**（`fmt.Println`）。失败前打印的内容（如安装失败原因）留在标准输出。
- 用法错误（`cli.Exit`）打印到**标准错误**，exit code 1。
- 下载进度条打印到**标准错误**。忽略 `NO_COLOR`，含 ANSI 颜色码，用回车重绘：
  `\rDownloading...  42% [====>     ] (1.2 MB/s) [3s:4s]\r      \r`。
- 树与状态字符为 UTF-8（`├ └ │ ─ ┬ ✓ ✗ —`）。

解析器因此：

1. 去 ANSI 转义（CSI、OSC 与两字节转义），即使有 `NO_COLOR`；
2. 按 `\n` **与**孤立 `\r` 分行；
3. 跳过所有不认识的行，不失败。

### 错误

失败命令（exit code ≠ 0）展示消息由其输出计算：

1. 取标准输出与标准错误的有意义行（非空、非下载进度行）。
2. 取最后一行；若为通用尾 `failed to install …` 且前面还有行，取前面更具体的一行
   （`failed to download maven@3.9.16 file, err:unexpected EOF`）。1.0.13 起尾在标准错误，
   该行在标准输出。
3. 无行可取则报 exit code。

可执行文件根本起不来报 vfox 缺失。

## 查询

只读查询并发跑。

### 版本

```console
$ vfox --version
vfox version 1.0.13
```

版本为首个 `\d+.\d+.\d+…` 词。夹具：`version`。

### 已安装 SDK：`vfox list` 与 `vfox current`

两命令都跑，按插件名合并结果。

```console
$ vfox list
All installed sdk versions
├─┬maven
│ ├──3.9.16
│ └──3.8.9
├──mvn2
└─┬nodejs
  ├──24.11.1
  └──22.21.0
```

匹配 `^([│ ]*)[├└]─[─┬](.+)$` 的行为项；首组长度除以 2 为深度。深度 0 为插件
（叶 `├──name` 为无版本插件），深度 1 为上个插件的版本。顺序保留。未装插件时
`vfox list` 失败 `you don't have any sdk installed yet` 而 `vfox current` 无输出；
这是空结果而非错误。夹具：`list`、`list-empty`。

```console
$ vfox current
maven -> v3.9.16
mvn2 -> N/A
nodejs -> v24.11.1
```

每行 `<名称> -> v<版本>` 或 `<名称> -> N/A `（注意尾空格）。去掉**一个**前导 `v`。
`vfox current` 反映 global < session < project 作用域链，故必须在 `/` 跑。夹具：
`current`、`current-unset`、`current-empty`。

### 插件信息

```console
$ vfox info nodejs --format '{{.Name}}{{"\x1f"}}{{.Version}}{{"\x1f"}}{{.Homepage}}{{"\x1f"}}{{.InstallPath}}{{"\x1f"}}{{.Description}}'
```

Go 模板以 ASCII 分隔符 `0x1F` 分隔字段（字段中不可能出现）。描述为末字段，可含换行；
多余字段拼回描述。夹具：`info`、`info-unknown`。

`vfox info <名称>@<版本>` 打印某版本安装目录，未安装打印 `notfound`。夹具：
`info-version-path`、`info-version-not-installed`。

### 可安装版本

```console
$ vfox search nodejs
Available versions:
 - 26.12.0 [npm 11.20.0]
 - 24.22.0 (LTS) [npm 11.19.1]
 - 3.9.16 (installed)
```

标准输出非终端时 vfox 打印 plain 列表而非交互选择。每项以 ` - ` 开头，形为
`<版本>[ (<备注>)][ [<工具> <版本>,…]][ (installed)]`，从右解析：`(installed)` 后缀、
方括号附加（逗号分隔，每项 `<工具> <版本>`，版本可空）、圆括号备注。
版本为首词。夹具：`search-maven`、`search-maven-installed`、`search-nodejs`、
`search-unknown`。

版本安装态以 `vfox list` 为准而非 `(installed)`，免得每次重搜。搜索可能耗时数秒，
插件走网络（除非 vfox 缓存，`cache.availableHookDuration`）。

### 插件仓库

```console
$ vfox available
AVAILABLE PLUGINS

  bun                 ✗  https://github.com/arfaWong/vfox-bun
  nodejs              ✓  https://github.com/version-fox/vfox-nodejs

  Use 'vfox add <plugin>' to install
```

`^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$`：名称、官方（✓）与主页。夹具：`available`。

### 配置

```console
$ vfox config --list
plugin.http.timeout = 0s
proxy.url =
proxy.enable = false
…
```

每行 `<键> = <值>`；值可空。未知键按文本展示编辑，GUI 跨 vfox 版本可用。
vfox 1.0.13 的键（1.0.12 同）：

| 键 | 类型 | 说明 |
| --- | --- | --- |
| `proxy.enable` | `true` / `false` | |
| `proxy.url` | URL | |
| `registry.address` | URL | 空为官方仓库。 |
| `storage.sdkPath` | 目录 | 必须存在可写。 |
| `legacyVersionFile.enable` | `true` / `false` | `.nvmrc`、`.node-version`、`.sdkmanrc` 等。 |
| `legacyVersionFile.strategy` | `specified` / `latest_installed` / `latest_available` | |
| `cache.availableHookDuration` | 时长（`12h`），`-1` 永久，`0` 关 | |
| `plugin.http.timeout` | 时长，`0s` 为默认（30 s） | |
| `plugin.http.downloadTimeout` | 时长，`0s` 为默认（30 min） | |
| `gitignore.enable` | `true` / `false` | 项目域加 `.vfox/` 到 `.gitignore`。 |

`vfox config <键> <值>` 设值，`vfox config --unset <键>` 重置。
非法值按 vfox 消息失败，如 `time: invalid duration "notaduration"`（夹具
`config-set-invalid-duration`）。未知键**静默成功**（exit code 0），故只提供
`--list` 打印过的键。夹具：`config-list`、`config-set`、`config-get`。

## 命令

改变状态的命令走任务队列，一次一个，输出流式展示给用户。

| 操作 | 命令行 | 工作目录 | 说明 |
| --- | --- | --- | --- |
| 安装 | `vfox install --yes <名称>@<版本>` | `/` | 进度在标准错误。无版本失败（`install-without-version`）。 |
| 卸载 | `vfox uninstall <名称>@<版本>` | `/` | 卸载当前版本 vfox 切到另一已安装版本。 |
| 使用（全局） | `vfox use --global <名称>@<版本>` | `/` | 需 `__VFOX_SHELL`，见下。 |
| 使用（项目） | `vfox use --project <名称>@<版本>` | 项目目录 | 写 `.vfox.toml`，链接 `.vfox/sdks`，可改 `.gitignore`。 |
| 停止使用 | `vfox unuse --global <名称>` | `/` 或项目 | |
| 添加插件 | `vfox add <名称>` | `/` | |
| 从来源添加 | `vfox add --source <URL或路径> [--alias <别名>] [<名称>]` | `/` | 名称默认插件自带名。本地 `.lua` 传拷贝，见[已知问题](#已知问题)。 |
| 移除插件 | `vfox remove --yes <名称>` | `/` | 连同全部已安装版本删除。无 `--yes` 失败（`remove-without-yes`）。 |
| 更新插件 | `vfox update <名称>` | `/` | |
| 更新全部 | `vfox update --all` | `/` | |

选项永远在位置参数前。

### GUI 不跑的命令

| 命令 | 原因 |
| --- | --- |
| `vfox activate`、`vfox env` | Shell 集成：打印 shell 代码或会话环境。GUI 只展示加到 `~/.bashrc` 的行。 |
| `vfox use --session`、`vfox exec`、`vfox cd` | 只在终端会话有意义。 |
| `vfox upgrade` | 替换 vfox 可执行文件，归包管理器或安装器管；vfox 按安装方式升级。 |

### 选择版本

Linux 与 macOS 上，`vfox use` 在激活 shell 外拒绝运行：

```console
$ vfox use --global maven@3.9.16
vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'
```

vfox 只检查 `__VFOX_SHELL` 非空。`--global` 或 `--project` 下 `use` 只改对应
`.vfox.toml` 与 `~/.vfox/sdks` 或 `<项目>/.vfox/sdks` 符号链接——正是 GUI 所需——并打印
`Now using maven@3.9.16.`。GUI 因此仅给 `vfox use` 设 `__VFOX_SHELL=bash`
（只给它）。session 域无 shell 无意义，从不用。夹具：`use-global`、
`use-without-hook`。

*之后*打开的 shell 在激活 vfox 后（`eval "$(vfox activate bash)"`）生效；GUI 告知用户。

### 下载进度

安装时按行（按 `\r` 分）读标准错误，匹配
`Downloading\.\.\.\s+(\d{1,3})%`。展示末值为确定进度，不记入可见日志。夹具：`install`。

### 取消

运行中命令靠终止 vfox 取消（Unix `SIGTERM`，Windows kill）；任务按取消计，不论
exit code。中断的 `vfox install` 残留取决于版本：

- 至 1.0.12，vfox 捕获信号，删除部分版本目录，exit code 0；
- 1.0.13 起，信号结束 vfox。部分目录保留，旁有 `.v-<版本>.installing` 标记；
  `vfox list` 不显示该版本，下次安装同版本先清理。

1.0.13 起，另一 vfox 进程（如终端）正安装同版本时拒绝安装：
`cannot install <名称>@<版本>: installation is in progress or cannot be locked: …`。

## 已知问题

1.0.12 与 1.0.13 皆然。

- `vfox add a b` 按插件报告失败但 exit 0。GUI 一次只加一个插件。
- 插件 pre-install hook 连不上网（如坏代理），`vfox install` 可能报
  `Install maven@ success!` 而版本为空。任务输出与 exit code 一起作为用户所见。
- `vfox config` 未知键静默成功。
- `vfox add --source <路径>.lua` *搬走*文件：重命名到临时目录（`~/.vfox/tmp`，旧
  `~/.version-fox/tmp` 存在则彼处），用户文件消失；跨文件系统（如 tmpfs 的 `/tmp`）
  重命名失败。本地 `.zip` 与 URL 只读。GUI 把本地 `.lua` 拷贝到 vfox 临时目录下新目录
  再传拷贝；vfox 从不清理其名不以日期开头的目录。
- 输出分两流：`vfox --version` 走标准输出，进度与用法错误走标准错误，命令返回的错误
  1.0.12 及以前走标准输出，1.0.13 起走标准错误（见[输出约定](#输出约定)）。

## 对真实 vfox 测试

跑真实 vfox 的测试不得碰测试者本机 vfox 安装。`HOME` 指向全新空目录跑 vfox，不设
`VFOX_HOME`。[`shared/test-plugin`](../../shared/test-plugin) 离线插件加本地 HTTP
服务器覆盖仓库、添加插件、搜索、带下载进度安装、使用、卸载、移除，全离线。

## 更新本契约

vfox 变化时，`vfxtool capture` 重抓输出，更新 `cases.json` 与本文，跑解析器测试。见
[`shared/fixtures/README.zh-CN.md`](../../shared/fixtures/README.zh-CN.md)。
