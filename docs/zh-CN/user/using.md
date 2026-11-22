# 使用 vfox-plus

> [English](../../en/user/using.md)

应用有五个去向：主页、SDK、插件、任务与设置。宽窗口在侧边栏，窄窗口在底部栏。

## 主页

- vfox 版本与可执行文件位置，带刷新按钮。
- 有在用版本的插件列表，点开进入该插件的 SDK 页。
- 健康检查（**健康**）：`vfox doctor` 发现漂移或旧安装时，可**检查健康**、**纳入**、**重装**。
- 终端里如何使用这些版本：给出写入 `~/.bashrc` 的那一行，带复制按钮。

## SDK

已安装插件列表，标出当前版本。链接版本显示其目录，卸载只删链接；别名当版本用。点一个看详情：描述、主页与两个页签（应用认识的插件有三个页签，
见 [Java、Node.js、Flutter、Gradle、Go 与 Python](#javanodejsfluttergradlego-与-python)）。宽窗口里详情始终是打开的，
你没选时是在用的那个插件。

**已安装**列出已装版本，最新在前。当前版本有标记，其它版本可设为全局默认。
每个版本还可固定到某个项目（在其目录下）、复制安装路径、重装、命名别名或卸载。移除插件本身会
删掉它的全部版本，操作前会确认。

**可安装**搜索能装的版本：按文本过滤，只看带备注（如 LTS）的或只看已安装的。
安装显示实时下载进度，可取消。列表里每个版本都有**安装**，旁边的菜单里有**安装并使用**。列表上方那一行
末尾的菜单（**更多添加方式**）是其他添加方式：**按名称安装…** 用 vfox 能理解的名称安装列表里没有的版本，
例如用 `24` 安装最新的 Node.js 24，勾选**安装后使用**可一次装好即用；**探测并链接**把机器上已有的纳入管理
（`vfox link --detect`）；**链接已有…**按你写的目录与版本名链接。

### Java、Node.js、Flutter、Gradle、Go 与 Python

插件 [vfox-java](https://github.com/version-fox/vfox-java)、
[vfox-nodejs](https://github.com/version-fox/vfox-nodejs)、
[vfox-flutter](https://github.com/version-fox/vfox-flutter)、
[vfox-gradle](https://github.com/version-fox/vfox-gradle)、
[vfox-golang](https://github.com/version-fox/vfox-golang) 与 [vfox-python](https://github.com/version-fox/vfox-python) 的功能不止列出版本，vfox-plus
把这些功能都提供出来：桌面应用与网关的 Web 界面都有，手机端 App 也有，差别见[与桌面应用的区别](android.md#与桌面应用的区别)。用别名添加的插件，如
`vfox add --alias coffee java`，与它所自出的插件提供同样的功能：

- **Java 的发行版。** 单独的 `vfox search java` 只列出 jdk.java.net 上的 OpenJDK。列表上方的
  按钮（**发行版：OpenJDK**）会打开 31 个发行版的列表，例如 Temurin、Zulu、Liberica、Corretto 与
  GraalVM，可以按名称、简称或厂商筛选；**全部发行版**一次列出所有发行版约 2700 个版本，并标明每个版本
  属于哪个发行版。你的选择会被记住，下次打开该插件和重启之后仍然生效；它保存在桌面应用的设置里，所以
  Web 界面和手机端 App 显示的是同一个选择。这和 `vfox search java tem` 是一回事。
- **筛选 Chip。** 列表上方的 Chip 按插件对版本的说明来收窄列表：Java 有 LTS 与 JavaFX，Node.js 有 LTS，
  Flutter 有通道（stable、beta、dev、OpenHarmony）与架构（x64、arm64），Python 有 GraalPy 与 GraalPy Community 两种实现。同一行的 Chip 是“或”的关系，
  不同行互相收窄。
- **按名称安装。** 有些名称由 vfox 在安装时解析：`24` 是最新的 Node.js 24，`latest` 是所有版本中最新的，
  `stable` 与 `beta` 是 Flutter 通道当前指向的版本，`21-tem` 是最新的 Temurin 21，`21-zulu-fx` 是带
  JavaFX 的 Zulu 21，`28.ea-open` 是抢先体验版的 OpenJDK，`3.13` 是最新的 Python 3.13，`3.13t` 是它的自由线程版，`latest` 是最新的 Gradle、Go 或标准 CPython 版本。对话框会为该插件给出这样的名称，并说明
  各自的含义。通道会被它指向的版本替换，之后在**已安装**里看到的就是那个版本名。
- **已安装的名称。** 已安装的 Java 版本会标出发行版（带 JavaFX 时也标出），已安装的 Flutter 版本会标出架构。链接版本保留你起的名称（如 `17-system`）。

| 中文界面里的 Java | 中文界面里的选项 |
| --- | --- |
| ![中文界面里 Java 的可安装版本](https://iili.io/nmih2xS.png) | ![中文界面里 Node.js 的选项](https://iili.io/nms7NXj.png) |

**选项**页签放的是这个插件特有的内容：

- **插件变量**：插件读取的变量，并提供已检查过的镜像可选：Java 与 Flutter 的 GitHub 镜像、Node.js 的
  下载镜像、Flutter 的存储镜像、Gradle 镜像列表、Go 镜像与 Python 的 python.org 和 uv-build 镜像，它们会加到本应用运行的每条 vfox 命令里，应用会检查地址以 `http://` 或
  `https://` 开头。
- **版本文件**：一个开关，下面是该插件读取的文件：Java 的 `.sdkmanrc`，Node.js 的 `.node-version` 与
  `.nvmrc`（vfox 1.1.0 上双向生效：离开项目切回全局），Gradle 的 `gradle/wrapper/gradle-wrapper.properties`，Python 的 `.python-version`；
  Node.js 与 Python 还有“文件里写的版本怎么选”的选项。
- **须知**：容易让人意外、又没有按钮能说明的几件事，例如不写发行版的 Java 是 jdk.java.net 上的 OpenJDK，只保留
  最新的版本；又如 vfox 管理的 Flutter 不能原地升级，又如部分版本号（如 `3.13`）不会选中自由线程版（`3.13t`）或预发布版。名称怎么写、常用的名称，在**按名称安装**对话框里。
- **纳入、漂移与盖住**：机器上已有 JDK、Gradle、Flutter、Go 或 Python 时，**探测并链接**纳入管理，不必重下；SDK 被原地改动时主页**健康**与 `vfox use` 报告漂移（**重装**，旧安装用**纳入**）；别的工具盖住 vfox 的版本时健康检查指明是谁与做法（`path.mode`、`path.preserve`）。
- **文档**：该插件讲解自身的页面，使用你的语言。

### Maven 与 PHP

插件 [vfox-maven](https://github.com/version-fox/vfox-maven) 与 [vfox-php](https://github.com/version-fox/vfox-php) 的功能同样不止列出版本：桌面应用与 Web 界面都有，手机端 App 也有：

- **按名称安装。** `latest` 安装最新的稳定版；预发布版按名称安装，如 `4.0.0-rc-7`。PHP 的 `latest` 是最新的发行版，在 Windows 上是它的线程安全构建。
- **PHP 的构建。** 在 Linux 和 macOS 上，不带后缀的版本是非线程安全构建，`-zts` 名称指线程安全构建，如 `8.5.12-zts`；`vfox search php` 不列出 `-zts` 构建。在 Windows 上，不带后缀的版本是线程安全构建，`-nts` 指非线程安全构建。
- **筛选 Chip。** Windows 上 PHP 有一个“构建”筛选，带 TS 与 NTS 两个 Chip。Linux 和 macOS 没有这个筛选。
- **构建选项。** 在 Linux 和 macOS 上，**选项**页签列出 PEAR、mhash 与 PCRE JIT，各有一个**更改**按钮。PHP 7.4 及以后默认不包含 PEAR，`PHP_WITH_PEAR=yes` 才会构建 `pear` 与 `pecl`；PHP 8.4 及以后默认不包含 mhash 函数，`PHP_WITH_MHASH=yes` 才会构建。两者设置后 configure 都会打印弃用警告。PCRE JIT 默认开启。
- **自由文本的选项。** `PHP_CONFIGURE_OPTIONS` 与 `MAKEFLAGS` 接受任意文本，所以**选项**页签不提供它们。请在*设置 → 环境变量*里以 `KEY=VALUE` 的形式设置；应用会把这一列表传给每条 vfox 命令。
- **探测并链接。** 会纳入 `M2_HOME` 与 `MAVEN_HOME` 指向的 Maven，以及系统软件包、Homebrew、SDKMAN、asdf、mise、Scoop 与 Chocolatey 装的 Maven；Maven 1 探测不到，请用**链接已有…**。PHP 也会纳入机器上已有的 PHP，需要 vfox 1.1.0 或更高版本。
- **Java 运行时。** Maven 需要 Java 运行时，但插件不会安装它：把 JDK 放进 `PATH` 或设置 `JAVA_HOME`。Maven 1.1 只能在 JDK 8 上运行。
- **构建耗时。** 在 Linux 和 macOS 上，PHP 从源码编译，需要一段时间。编译前会先检查依赖并列出缺少的软件包，缺少时不会开始编译。
- **Windows 运行库。** 在 Windows 上，缺少 Visual C++ 2015-2022 运行库时，安装会在 Windows 请求授权后将其添加。

### Dart、Kotlin 与 .NET

插件 [vfox-dart](https://github.com/version-fox/vfox-dart)、
[vfox-kotlin](https://github.com/version-fox/vfox-kotlin) 与
[vfox-dotnet](https://github.com/version-fox/vfox-dotnet) 在三个界面里也提供这些功能：

- **Dart 的通道。** `stable`、`dev` 与 `beta` 安装该通道上最新的版本，在安装时解析。同时发布在多个通道上的版本会为每个通道各列一次，所以 `vfox search dart` 可能出现两个相同的版本号，注记里写有通道；**通道**筛选可以选择其一。`latest` 是列表中版本号最大的版本，可能是 dev 构建；最新的稳定版请用 `stable`。插件不安装 arm64 构建，尽管 Dart 的官方归档里有 Linux、macOS 和 Windows 的 arm64 SDK。
- **Kotlin。** 版本要完整写出，如 `2.1.0`；`latest` 是 Kotlin 发布页面上最新的版本。若该页面无法读取，搜索与安装会报出原因，而不会猜测版本。有预编译包的平台会一并安装 Kotlin/Native。`kotlinc` 与 `kotlinc-native` 需要 PATH 上有 Java 17 或更高版本，插件不会安装。
- **.NET 的名称与变体。** 通道或发行版如 `8`、`8.0` 或 `8.0.34` 会安装 SDK；`-runtime` 与 `-aspnetcore` 安装较小的发行版，**发行版**筛选可以按它们收窄列表。通道与 `latest` 选的是最新稳定版，只有通道里没有稳定版时才选预览版；`11.0.0-rc.3` 这样的完整版本号直接指定预览版。找不到适用于本机平台的匹配版本时，安装会报错并指出该版本，且不安装任何东西。插件会设置 `DOTNET_ROOT` 与 `PATH`；Linux 上的 SDK 还需要 ICU、libssl 与 zlib。
- **纳入与变量。** **探测并链接**可纳入机器上已有的 SDK —— Dart SDK（包括 Flutter 自带的）、来自 SDKMAN、asdf、mise、Homebrew、IDE 自带与 `KOTLIN_HOME` 的 Kotlin 编译器，以及 `DOTNET_ROOT` 与 `DOTNET_HOME` 指向的 .NET SDK —— 需要 vfox 1.1.0 或更高版本。这三个插件不读取变量，也没有旧版版本文件，所以**选项**页签只有说明、可安装的名称与文档。

### CMake

插件 [vfox-cmake](https://github.com/version-fox/vfox-cmake) 在三个界面里也提供这些功能：

- **名称。** 版本要完整写出，如 `3.31.6`；候选版写成 `4.5.0-rc1`。`3.31` 这样的版本线不是发行版，安装会失败。`latest` 安装最新的正式版。
- **候选版。** 插件把候选版排在它所属的正式版之后；还没有正式版的候选版排在最后，所以只要列表里有正式版，`latest` 就是正式版。**候选版**筛选为 `rc1` 到 `rc6` 各设一个 Chip，保留以它结尾的版本。
- **探测并链接。** 会纳入 `PATH`、`CMAKE_HOME`、常见的系统位置里的 CMake，以及 Homebrew、Scoop 或 Chocolatey 装的 CMake。`CMAKE_HOME` 请在*设置 → 环境变量*里以 `KEY=VALUE` 的形式设置：应用会把这一列表传给每条 vfox 命令。
- **要求。** 插件需要 vfox 1.1.0 或更高版本，并要能访问 jsDelivr（版本列表）与 cmake.org（安装包），没有可设置的镜像。它不读取变量，也没有旧版版本文件，所以**选项**页签只有说明、可安装的名称与文档。

### Erlang 与 Clang

插件 [vfox-erlang](https://github.com/version-fox/vfox-erlang) 与 [vfox-clang](https://github.com/version-fox/vfox-clang) 在三个界面里也提供这些功能。两者都只在 Linux 上检查过。

- **预编译的 Erlang/OTP。** 在 Linux 上，**预编译构建**选项选一个 Ubuntu 版本（20.04、22.04 或 24.04），就用它的预编译构建安装 Erlang/OTP，而不从源码编译。此时版本列表是预编译构建，按发行版号列出，如 `29.1.2`；名称 `OTP-29.1.2` 也会安装该发行版的预编译构建。选项留空则从源码编译，需要很长时间，并需要插件源码编译页面中列出的编译工具。选择了预编译构建时，`latest` 是列表中最新的发行版。搜索缓存按插件保存，与选项无关，所以使用预编译构建前请关闭它，见插件的预编译页面。
- **Erlang 的设置。** `OTP_CONFIGURE_ARGS` 把参数传给 configure，如 `--without-javac`；`MAKEFLAGS` 设置 make 的选项，如 `-j8`。请在*设置 → 环境变量*里以 `KEY=VALUE` 的形式设置。
- **文档。** **选项**页签链接到 vfox-erlang 的文档网站（英文与简体中文），以及它的预编译页面、源码编译页面和 README。
- **Clang。** `latest` 安装列表中标为 latest 的版本。**Conda 频道**选项选择 pixi 安装 clang 所用的 conda 频道，预设为 prefix.dev 与清华镜像；留空则为 conda-forge。每次安装先下载 pixi，通常需要约一分钟。
- **Clang 的 GitHub 地址。** 插件用 `GITHUB_URL` 获取版本列表与 pixi 的下载。它没有预设，请在*设置 → 环境变量*里把它设为保持 GitHub 路径的镜像。
- **家目录。** 源码编译 Erlang/OTP 时，家目录的路径含单引号会失败；预编译构建可以在含空格的路径下安装。Clang 在家目录的路径含空格时无法安装。

### Elixir

插件 [vfox-elixir](https://github.com/version-fox/vfox-elixir) 在三个界面里提供这些功能。它只在 Linux 上检查过。

- **先装 Erlang/OTP。** Elixir 用 Erlang/OTP 编译，运行时也需要 PATH 上有 `erl`。vfox 不会为 Elixir 安装 Erlang/OTP。源码构建在 PATH 上没有 `erlc` 时，会使用 vfox 已安装的 Erlang/OTP 的 `erlc`。请先用 erlang 插件安装 Erlang/OTP，再与 Elixir 一起运行，例如 `vfox exec erlang@28.4.2 elixir@1.20.5 -- elixir --version`。单独运行 `vfox exec elixir@1.20.5 -- elixir --version` 会以 `exec: erl: not found` 失败。
- **预编译构建。** 在 Linux 上，**预编译构建**选项用 hex.pm 的预编译构建安装 Elixir，而不从源码编译。大多数名称带 Erlang/OTP 的主版本，如 `1.20.5-otp-29`；少数没有 `-otp-N` 后缀，如 `1.16`，这样的构建可能无法用你的 Erlang/OTP 运行。选项不改变 `vfox search elixir`，它列出的是发行版：`vfox search elixir hex` 才列出预编译构建。选项留空则从源码编译：源码构建约需一分钟，并且家目录的路径中不能有空格；预编译构建不受此限制。
- **版本。** `latest` 是列表的第一行：Linux 上是 `1.20-latest`，检查时它运行为 1.20.5。使用预编译构建时 `latest` 会失败，请输入 `1.20.5-otp-29` 这样的构建名称。请完整写出版本，如 `1.20.5`。源码构建中，`1.16` 这样的部分版本号不是发行版，因为它的标签不存在。
- **搜索。** 无论选项如何，`vfox search elixir` 都列出发行版；`vfox search elixir hex` 列出预编译构建。插件文档说 vfox 对每次搜索的列表最多保留 12 小时。
- **文档。** **选项**页签链接到 vfox-elixir 的文档网站（英文与简体中文），以及它的 README。

### etcd、Deno、Groovy、Scala 与 Zig

插件 [vfox-etcd](https://github.com/version-fox/vfox-etcd)、[vfox-deno](https://github.com/version-fox/vfox-deno)、[vfox-groovy](https://github.com/version-fox/vfox-groovy)、[vfox-scala](https://github.com/version-fox/vfox-scala) 与 [vfox-zig](https://github.com/version-fox/vfox-zig) 在三个界面里也提供这些功能，但受限于它们当前的列表：

- **名称。** Deno、Groovy、Scala 与 Zig 的 `latest` 安装列表中的第一个版本（Zig 的 `latest` 是最新的稳定版），Zig 的 `nightly` 安装其 master 分支的开发版。Groovy 接受完整的版本名，如 `4.0.33`，不接受 `4` 这样的部分版本号；它的列表把每个预发布版排在对应的正式版之下。Scala 接受完整的版本名，如 `3.10.1`，不接受 `3` 或 `2.13` 这样的部分版本号。Deno 还接受 `2`、`2.1` 这样的版本线，安装该线上最新的发行版。Zig 接受完整的版本名，如 `0.17.1`，以及备注中的 Mach 名称，如 `mach-latest`，不接受 `0.17` 这样的部分版本号。etcd 接受完整的版本名，如 `3.7.3`，可以带 `v`，但不接受 `3.5` 这样的部分版本号；`latest` 安装列表的第一行，即最新的发行版。
- **归档镜像。** Groovy 的**归档镜像**选项提供 dlcdn.apache.org，它是 Apache 归档的另一个副本，archive.apache.org 较慢时可以使用。它只保留当前的发行版，所以较旧的版本不会出现在列表中。没有提供清华镜像：检查时它能列出版本，但其下载被拒绝。
- **阶段。** Groovy 的**阶段**筛选可以选出预发布版或孵化版。
- **下载。** Scala 2.10.4 及更早的版本从 scala-lang.org 下载；Scala 2.10.5 及之后的版本，以及全部 Scala 3，从 GitHub 下载。从 GitHub 下载时，若 Scala 发布了 `.sha256` 文件，就据此校验；没有时安装会说明未校验。列表中标为 LTS 的 Scala 3.9.0 以其版本号安装。
- **JDK。** Groovy 与 Scala 运行需要 JDK，插件都不安装。检查时，Scala 2.10.4 无法在所检查的 JDK 25 上运行。
- **校验和。** Deno 与 Groovy 在发行版提供校验文件时据此校验（Deno 为 `.sha256sum`，Groovy 为 `.sha256`），没有时会说明未校验。etcd 用其发行版的 `SHA256SUMS` 文件校验。没有该文件的 3.4.0 之前的发行版（如 0.3.0），会用插件固定的校验和检查，安装时会说明；3.4.0 及以后没有该文件的发行版会被拒绝。
- **Deno。** 列表从 0.36.0 开始，它是 Linux x86-64 上第一个有下载文件的发行版。插件在解压后为 Deno 设置可执行位，所以 0.36.0 可以运行。0.2.3 之前的插件安装的 0.36.0 没有可执行位：请重新安装它。
- **Zig。** 列表需要 ziglang.org：它无法访问时，整个列表都会失败。machengine.org 是可选的：无法访问时，列表只显示官方发行版，并给出警告。列表中没有的名称会被拒绝，什么也不会安装。vfox 1.1.0 的 `exec` 不接受 `latest`、`nightly` 或 Mach 名称，请用 `vfox list` 显示的版本名；`vfox use` 可以接受这些名称。ziglang.org 没有发布的 Mach 构建从 pkg.hexops.org 下载，仍然按索引中的校验和检查。
- **etcd。** 失败构建由 `BUILD_WITH_FAILPOINT=yes` 开启，它会按所选版本从源码构建，并启用 failpoint。需要 PATH 上有 Git、Go 和 GNU Make，并且比安装发行版耗时更长。仅在 Linux 上检查过：3.5.12 能构建出带 failpoint 的版本，3.5.5 会失败。请用 `GOFAIL_HTTP` 检查结果。只有值 `yes` 会开启该构建：`1` 与 `YES` 会被拒绝，`no` 或留空则安装官方构建。**选项**页签不为它提供开关。
- **文档。** etcd 的**选项**页签链接到 vfox-etcd 的文档网站（英文与简体中文）、它的文档与 README。vfox-deno 与 vfox-groovy 的**选项**页签链接各自的文档（英文与简体中文）以及 README。

## 插件

- 搜索仓库，按官方/社区过滤。
- 每张卡片说明插件是否已装，可打开其主页。
- 从仓库、URL 或本地 `.zip`/`.lua` 文件添加插件，一键更新全部插件。
- 已装插件提供更新与移除。

## 任务

安装（含安装并使用、重装、链接与探测链接、别名设置与删除、doctor 纳入）、卸载、切换、添加、移除、更新都会成为一条任务，最新在前。每条任务显示
状态（排队、运行、成功、失败、已取消），运行中的安装有实时进度，输出日志附
带丢弃行数；失败任务的完整日志保留在磁盘并显示位置。失败或取消的任务可重试，已完成的可以清除。在别的页面时任务完成会有
消息提示。

## 设置

- **外观**：浅色、深色或跟随系统；用桌面强调色或自选色板；语言（English 或简体中文）。
- **其他设备**：同一网络中的手机、网关和其他电脑是否可以通过 HTTPS 连接，没有窗口时是否为它们
  继续运行（轻量模式）。见[其他设备与轻量模式](remote.md)。
- **vfox**：用哪个可执行文件（自动探测，也可自选）与下载镜像等额外环境变量。
- **vfox 配置**：代理、仓库源、存储目录、旧版版本文件、缓存与超时、诊断（`diagnostics.onUse`）与 PATH 顺序（默认 `path.mode` 为 `baseline`，`path.preserve`）——开关、菜单与
  文本框；清空文本框即恢复默认。别名（`vfox alias`）与链接版本（`vfox link`）从 vfox 读取。
- **关于**：版本、源码与依赖许可证。

## 找不到 vfox 时

找不到 vfox 可执行文件，或找到的不能用、版本早于 vfox 1.1.0 时，每个页面都会说明如何安装 vfox，并提供
定位可执行文件与重试。设置页始终可用，可手动指向 vfox。
