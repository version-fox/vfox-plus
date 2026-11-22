# vfox CLI 契约

> [English](../../en/dev/vfox-cli-contract.md)

本文规定 vfox-plus 如何驱动 vfox 命令行工具：跑哪些命令、什么环境、如何读取输出、
如何报告错误。描述 **vfox 1.1.0**；捕获输出在
[`shared/fixtures`](../../../shared/fixtures)，期望解析结果在 `cases.json`。

最低支持 vfox **1.1.0**：1.1.0 新增 link、doctor、alias、path.mode、install --use
命令与插件 Detect、Fingerprint 钩子，应用依赖它们，更早版本不支持。

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
| `NO_COLOR` | 置 `1` | 关掉多数输出颜色（<https://no-color.org>）。 |
| `__VFOX_SHELL` | 仅 `vfox use` 与 `vfox uninstall` 置 `bash` | 见[选择版本](#选择版本)。 |

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

- 命令返回的错误 exit code 1 打印到**标准错误**。失败前打印的内容（如安装失败原因）留在标准输出。
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

插件钩子抛出的错误会连同 Lua 堆栈一起打印：

```text
plugin [PreInstall] method error: ~/.vfox/plugin/java/hooks/pre_install.lua:42: No JDK found for 99-tem on linux/amd64. …
stack traceback:
	[G]: in function 'error'
	~/.vfox/plugin/java/hooks/pre_install.lua:42: in main chunk
	[G]: ?
```

所以第 1 条还会去掉 `stack traceback:` 行与其后以制表符缩进的行，并去掉 `plugin [<hook>] method error:` 里的文件名与行号：
留下的是插件说的那句话 `No JDK found for 99-tem …`，而不是 `[G]: ?`。这对按名称安装很重要，因为名称写错了要由插件来报告。
夹具：`install-plugin-error`。

可执行文件根本起不来报 vfox 缺失。

## 查询

只读查询并发跑。

### 版本

```console
$ vfox --version
vfox version 1.1.0
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

每行 `<名称> -> v<版本>` 或 `<名称> -> N/A` 加一个空格（注意尾空格）。去掉**一个**前导 `v`。
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

#### 搜索参数

```console
$ vfox search java
$ vfox search java tem
$ vfox search java all
```

插件名后面的内容会作为 `ctx.args` 交给插件的 `Available` 钩子；多数插件忽略它，但 vfox-java 接受发行版的
简称或全称（`tem`、`temurin`）或 `all`，列出那个发行版，而不是不带参数时列出的 jdk.java.net 上的 OpenJDK
（Linux loong64 上是 `loongnix`、`lnxabi1` 与 `lnxabi2`）。vfox 按参数列表分别缓存答案。应用传一个参数，取自
插件[配置档案](architecture.md#插件配置档案)里的选择项，默认项不传参数。每个参数按插件名的方式检查：不能为空、
不能以 `-` 开头、不含空白与控制字符。`java all` 的列表有 2683 项，备注的形式有 `LTS`、`JavaFX` 与
`LTS, JavaFX`，版本名带发行版：`21.0.12.1+1-tem`、`25.0.4+9-librca-fx`。夹具：`search-java`、
`search-java-tem`、`search-java-all`、`search-flutter`（它的备注是通道，附带工具为 `[dart <版本>]`，最早的版本
是空的 `[dart ]`，分支的构建带 `(OpenHarmony)`）。

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
vfox 1.1.0 的键：

| 键 | 类型 | 说明 |
| --- | --- | --- |
| `proxy.enable` | `true` / `false` | |
| `proxy.url` | URL | |
| `registry.address` | URL | 空为官方仓库。 |
| `storage.sdkPath` | 目录 | 必须存在可写。 |
| `legacyVersionFile.enable` | `true` / `false` | `.nvmrc`、`.node-version`、`.sdkmanrc`、`gradle/wrapper/gradle-wrapper.properties` 等。 |
| `legacyVersionFile.strategy` | `specified` / `latest_installed` / `latest_available` | |
| `cache.availableHookDuration` | 时长（`12h`），`-1` 永久，`0` 关 | |
| `plugin.http.timeout` | 时长，`0s` 为默认（30 s） | |
| `plugin.http.downloadTimeout` | 时长，`0s` 为默认（30 min） | |
| `gitignore.enable` | `true` / `false` | 项目域加 `.vfox/` 到 `.gitignore`。 |
| `diagnostics.onUse` | `true` / `false` | `vfox use` 时警告被盖住与漂移的 SDK。 |
| `diagnostics.ignore` | 列表，如 `[]` | 不报告的检查 ID。 |
| `path.mode` | `legacy` / `baseline` / `aggressive` | 1.1.0 起默认 `baseline`。 |
| `path.preserve` | 目录列表 | `baseline` 与 `aggressive` 下保留在前的目录。 |
| `aliases` | 映射，如 `map[]` | 版本别名；经 `vfox alias` 编辑，勿手改。 |

`vfox config <键> <值>` 设值，`vfox config --unset <键>` 重置。
1.1.0 起只写被设置的键。非法值按 vfox 消息失败，如 `cache.availableHookDuration: time: invalid duration "notaduration"`（夹具
`config-set-invalid-duration`）。设 section、列表或表格为键会失败；删不存在的键成功且不重写文件。未知键**静默成功**（exit code 0），故只提供
`--list` 打印过的键。夹具：`config-list`、`config-set`、`config-get`。

### 健康、别名与链接版本

`vfox doctor [--json] [<sdk>[@<version>]]` 检查问题；`--json` 输出 `{"ok", "findings"}`，仅未忽略的 error 才 exit 1。`--adopt` 记录旧安装以便后查漂移。夹具：`doctor`、`doctor-json`。

`vfox alias list [<sdk>]` 每行 `<sdk>@<别名> -> <sdk>@<版本>`；`set` 命名已安装版本，`unset` 删除。夹具：`alias-list`。

`vfox list` 给链接版本加后缀，解析只取版本号。`vfox info --json` 用 `origin` 与 `target` 区分链接与安装。夹具：`info-json`。详见英文版。

## 命令

改变状态的命令走任务队列，一次一个，输出流式展示给用户。

| 操作 | 命令行 | 工作目录 | 说明 |
| --- | --- | --- | --- |
| 安装 | `vfox install --yes <名称>@<版本>` | `/` | 进度在标准错误。无版本失败（`install-without-version`）。版本可以是插件能解析的任何名称，见[插件能解析的名称](#插件能解析的名称)。 |
| 卸载 | `vfox uninstall <名称>@<版本>` | `/` | 卸载当前版本 vfox 切到另一已安装版本；切换需 `__VFOX_SHELL`，没有它则卸载后以状态 1 退出。 |
| 使用（全局） | `vfox use --global <名称>@<版本>` | `/` | 需 `__VFOX_SHELL`，见下。 |
| 使用（项目） | `vfox use --project <名称>@<版本>` | 项目目录 | 写 `.vfox.toml`，链接 `.vfox/sdks`，可改 `.gitignore`。 |
| 停止使用 | `vfox unuse --global <名称>` | `/` 或项目 | |
| 添加插件 | `vfox add <名称>` | `/` | |
| 从来源添加 | `vfox add --source <URL或路径> [--alias <别名>] [<名称>]` | `/` | 名称默认插件自带名。本地 `.lua` 传拷贝，见[已知问题](#已知问题)。 |
| 移除插件 | `vfox remove --yes <名称>` | `/` | 连同全部已安装版本删除。无 `--yes` 失败（`remove-without-yes`）。 |
| 更新插件 | `vfox update <名称>` | `/` | |
| 更新全部 | `vfox update --all` | `/` | |

选项永远在位置参数前。

### 插件能解析的名称

`<版本>` 不只是搜索列表里的名称。插件的 `PreInstall` 钩子把收到的内容变成安装包，而 version-fox 组织的
插件能理解的不止列出的名称：vfox-nodejs 接受版本的开头（`24`、`24.21`）与 `latest`；vfox-flutter 接受通道
`stable` 与 `beta`（可带架构后缀），并以通道指向的版本命名安装；vfox-java 接受版本加发行版（`21-tem`、
`temurin-21`、`21.0.8-tem`、`21-zulu-fx`）或抢先体验版（`28.ea-open`）；vfox-gradle 接受 `latest`；vfox-golang 接受 `latest` 与 `1.27rc3` 这样的候选版；vfox-python 接受 `latest`、`3.13` 或 `3.13t` 这样的部分版本号，以及 `graalpy-24.2.0` 这样的完整 GraalPy 名称。vfox-maven 与 vfox-php 接受 `latest`；vfox-php 在 Linux 和 macOS 上还接受 `-zts` 名称，如 `8.5.12-zts`，在 Windows 上接受 `-nts` 名称。vfox-dart 接受 `stable`、`dev` 与 `beta` 通道，它们解析为该通道上最新的版本，而一个发布在多个通道上的版本会为每个通道各列一次；vfox-kotlin 接受 `latest`（它是最新的发行版）与完整的版本名；vfox-dotnet 接受通道或发行版（`8`、`8.0`、`8.0.34`），可带 `-runtime` 或 `-aspnetcore` 后缀，预览版要用完整版本如 `11.0.0-rc.3`；通道与 `latest` 选的是最新的稳定版，只有通道里没有稳定版时才选预览版；vfox-cmake 接受 `latest`（它是最新的正式版，候选版排在它所属的正式版之后）与完整的版本名，不接受 `3.31` 这样的版本线。vfox-erlang 接受列表中的名称：`29.1.2` 这样的发行版，设置了 `USE_PREBUILT_OTP` 时也接受 `OTP-29.1.2` 这样的名称；`latest` 是列表中最新的发行版。列表中没有的名称会在下载前被拒绝。两类名称不做此检查：`master` 与 `maint-` 分支（它们通过分支地址获取，完整的源码安装未检查），以及以 `latest` 结尾的名称（见 #34）。预编译安装按 `29.1.2` 记录，`vfox exec` 不会把 `OTP-29.1.2` 映射到它。`USE_PREBUILT_OTP` 接受 14.04 至 26.04 共七个 Ubuntu 版本；选项页签提供 20.04、22.04 与 24.04。vfox-clang 接受 `latest`，即列表标为 latest 的版本。vfox-deno 接受 `latest` 与 `2`、`2.1` 这样的版本线，后者解析为该线上最新的发行版；插件在解压后为 Deno 设置可执行位，所以 0.36.0 可以运行，0.2.3 之前的插件安装的 0.36.0 需要重新安装。vfox-scala 接受 `latest`（列表中的第一个版本），LTS 版本 `3.9.0` 以其版本号安装。vfox-zig 接受 `latest`（最新的稳定版）、`nightly`（master 分支的开发版）以及 `mach-latest` 这样的 Mach 名称；vfox 1.1.0 的 exec 不接受这些名称，需要用版本名。vfox-groovy 接受 `latest`（最新的正式版）与完整的版本名，如 `4.0.33`，不接受 `4` 这样的部分版本号。vfox-etcd 接受完整的发行版名称，如 `3.7.3`，可以带 `v`，以及列表第一行的 `latest`，但不接受 `3.5` 这样的部分版本号。vfox-elixir 接受 `latest`（列表第一行，Linux 上是 `1.20-latest`；使用预编译构建时 `latest` 会在下载前停止）、`1.20.5` 这样的完整版本名，以及使用预编译构建时列表显示的 `1.20.5-otp-29` 这样的名称，它们带有所需的 Erlang/OTP 版本；没有 `-otp-N` 后缀的名称（如 `1.20.5`）不会列出，如果手动输入，安装会警告它可能无法用你的 Erlang/OTP 运行。所以应用会安装用户输入的任何名称，
按其他版本一样检查（不能为空、不能以 `-` 开头、不含空白、不含 `@`），并在之后从 `vfox list` 取已安装的名称，
因为通道不是已安装的名称。每个插件的[配置档案](architecture.md#插件配置档案)列出值得推荐的名称。

### 插件的环境

插件从 vfox 进程的环境里读环境变量，所以应用把设置里的额外环境传给每条 vfox 命令（见[环境](#环境)）：
`VFOX_JAVA_GITHUB_MIRROR` 与 `VFOX_FLUTTER_GITHUB_MIRROR`（加在 GitHub 地址前面的前缀，必须以 `http://` 或
`https://` 开头；vfox 会把其他值当成磁盘上的文件）、`VFOX_NODEJS_MIRROR`、`FLUTTER_STORAGE_BASE_URL` 与
`VFOX_NODEJS_LIBC`（`musl` 或 `glibc`）、`VFOX_GRADLE_MIRRORS`、`VFOX_GOLANG_MIRROR`（以 `/` 结尾）、`VFOX_PYTHON_MIRROR`、`VFOX_PYTHON_UV_BUILD_MIRROR`、`VFOX_PYTHON_USE_UV_BUILD`（`0` 或 `1`）、`VFOX_PYTHON_UV_LIBC`（`gnu` 或 `musl`）与 `VFOX_PYTHON_REGISTER_PY_LAUNCHER`（Windows 上设为 `0` 则关闭启动器注册）。`PHP_WITH_PEAR`（PHP 7.4 及以后）、`PHP_WITH_MHASH`（PHP 8.4 及以后）与 `PHP_WITHOUT_PCRE_JIT`（取 `yes` 或 `no`）是 vfox-php 在 Linux 和 macOS 上的构建选项；PHP 7.4 之前默认构建 PEAR，用 `PHP_WITHOUT_PEAR` 关闭。插件的选项页签把这些开关写进同一个列表。`PHP_CONFIGURE_OPTIONS` 与 `MAKEFLAGS` 接受任意文本，所以选项页签不提供它们：在*设置 → 环境变量*里以 `KEY=VALUE` 的形式设置，应用会把这一列表传给每条 vfox 命令。vfox-dart、vfox-kotlin、vfox-dotnet、vfox-cmake、vfox-deno、vfox-scala 与 vfox-zig 不读取上述任何变量：它们的选项页签只有说明、可安装的名称与文档。vfox-groovy 读取 `VFOX_GROOVY_MIRROR`，选项页签提供 dlcdn.apache.org 这一个只保留当前发行版的 Apache 归档镜像。vfox-elixir 读取 `VFOX_ELIXIR_MIRROR`，Linux 上选项页签以 `hex` 值提供预编译构建。vfox-erlang 读取 `USE_PREBUILT_OTP`（Linux 上选项页签提供 `ubuntu-20.04`、`ubuntu-22.04` 与 `ubuntu-24.04`），`OTP_CONFIGURE_ARGS` 与 `MAKEFLAGS` 为自由文本，在*设置 → 环境变量*里设置。vfox-clang 读取 `Conda_Forge`（选项页签提供两个镜像），以及 `GITHUB_URL`（在*设置 → 环境变量*里设置，没有预设）。vfox-etcd 读取 `BUILD_WITH_FAILPOINT`，但选项页签不提供它，因为只有部分发行版能带 failpoint 构建；只有值 `yes` 会开启该构建，`1` 与 `YES` 会被拒绝，`no` 或留空则安装官方构建。vfox-cmake 的探测钩子读取 `CMAKE_HOME`，和 `PHP_CONFIGURE_OPTIONS` 一样在*设置 → 环境变量*里设置。vfox 按参数列表把搜索的答案缓存 `cache.availableHookDuration`，
与环境无关，所以更换镜像后，列表要等缓存过期才会反映出来。

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
`Now using maven@3.9.16.`。GUI 因此给 `vfox use` 与 `vfox uninstall` 设 `__VFOX_SHELL=bash`
（只给它们）。卸载当前版本只有带它才会切换到另一已安装版本；没有它则 vfox 打印切换信息、以状态 1 退出，且没有当前版本。session 域无 shell 无意义，从不用。夹具：`use-global`、
`use-without-hook`。

*之后*打开的 shell 在激活 vfox 后（`eval "$(vfox activate bash)"`）生效；GUI 告知用户。

### 下载进度

安装时按行（按 `\r` 分）读标准错误，匹配
`Downloading\.\.\.\s+(\d{1,3})%`。展示末值为确定进度，不记入可见日志。夹具：`install`。

### 取消

运行中命令靠终止 vfox 取消（Unix `SIGTERM`，Windows kill）；任务按取消计，不论
exit code。中断的 `vfox install` 残留取决于版本：

- 信号结束 vfox。部分目录保留，旁有 `.v-<版本>.installing` 标记；
  `vfox list` 不显示该版本，下次安装同版本先清理。

另一 vfox 进程（如终端）正安装同版本时拒绝安装：
`cannot install <名称>@<版本>: installation is in progress or cannot be locked: …`。

## 已知问题

- `vfox add a b` 按插件报告失败但 exit 0。GUI 一次只加一个插件。
- 插件 pre-install hook 连不上网（如坏代理），`vfox install` 可能报
  `Install maven@ success!` 而版本为空。任务输出与 exit code 一起作为用户所见。
- `vfox config` 未知键静默成功。
- `vfox add --source <路径>.lua` *搬走*文件：重命名到临时目录（`~/.vfox/tmp`，旧
  `~/.version-fox/tmp` 存在则彼处），用户文件消失；跨文件系统（如 tmpfs 的 `/tmp`）
  重命名失败。本地 `.zip` 与 URL 只读。GUI 把本地 `.lua` 拷贝到 vfox 临时目录下新目录
  再传拷贝；vfox 从不清理其名不以日期开头的目录。
- 输出分两流：`vfox --version` 走标准输出，进度与用法错误走标准错误，命令返回的错误
  走标准错误（见[输出约定](#输出约定)）。

## 对真实 vfox 测试

跑真实 vfox 的测试不得碰测试者本机 vfox 安装。`HOME` 指向全新空目录跑 vfox，不设
`VFOX_HOME`。[`shared/test-plugin`](../../../shared/test-plugin) 离线插件加本地 HTTP
服务器覆盖仓库、添加插件、搜索、带下载进度安装、使用、卸载、移除，全离线。

## 更新本契约

vfox 变化时，`vfxtool capture` 重抓输出，更新 `cases.json` 与本文，跑解析器测试。见
[`shared/fixtures/README.zh-CN.md`](../../../shared/fixtures/README.zh-CN.md)。
