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
把这些功能都提供出来，桌面应用、Web 界面和安卓应用都一样。用别名添加的插件，如
`vfox add --alias coffee java`，与它所自出的插件提供同样的功能：

- **Java 的发行版。** 单独的 `vfox search java` 只列出 jdk.java.net 上的 OpenJDK。列表上方的
  按钮（**发行版：OpenJDK**）会打开 31 个发行版的列表，例如 Temurin、Zulu、Liberica、Corretto 与
  GraalVM，可以按名称、简称或厂商筛选；**全部发行版**一次列出所有发行版约 2700 个版本，并标明每个版本
  属于哪个发行版。你的选择会被记住，下次打开该插件和重启之后仍然生效；它保存在桌面应用的设置里，所以
  Web 界面和安卓应用显示的是同一个选择。这和 `vfox search java tem` 是一回事。
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

插件 [vfox-maven](https://github.com/version-fox/vfox-maven) 与 [vfox-php](https://github.com/version-fox/vfox-php) 的功能同样不止列出版本，桌面应用、Web 界面和安卓应用都一样：

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

- **Dart 的通道。** `stable`、`dev` 与 `beta` 安装该通道上最新的版本，在安装时解析。同时发布在多个通道上的版本会为每个通道各列一次，所以 `vfox search dart` 可能出现两个相同的版本号，注记里写有通道；**通道**筛选可以选择其一。`latest` 是列表中版本号最大的版本，可能是 dev 构建；最新的稳定版请用 `stable`。任何系统都不支持 arm64。
- **Kotlin。** 版本要完整写出，如 `2.1.0`；`latest` 是 Kotlin 发布页面上最新的版本。有预编译包的平台会一并安装 Kotlin/Native。`kotlinc` 与 `kotlinc-native` 需要 PATH 上有 Java 17 或更高版本，插件不会安装。
- **.NET 的名称与变体。** 通道或发行版如 `8`、`8.0` 或 `8.0.34` 会安装 SDK；`-runtime` 与 `-aspnetcore` 安装较小的发行版，**发行版**筛选可以按它们收窄列表。通道与 `latest` 选的是最新稳定版，只有通道里没有稳定版时才选预览版；`11.0.0-rc.3` 这样的完整版本号直接指定预览版。插件会设置 `DOTNET_ROOT` 与 `PATH`；Linux 上的 SDK 还需要 ICU、libssl 与 zlib。
- **纳入与变量。** **探测并链接**可纳入机器上已有的 SDK —— Dart SDK（包括 Flutter 自带的）、来自 SDKMAN、asdf、mise、Homebrew、IDE 自带与 `KOTLIN_HOME` 的 Kotlin 编译器，以及 `DOTNET_ROOT` 与 `DOTNET_HOME` 指向的 .NET SDK —— 需要 vfox 1.1.0 或更高版本。这三个插件不读取变量，也没有旧版版本文件，所以**选项**页签只有说明、可安装的名称与文档。

### CMake

插件 [vfox-cmake](https://github.com/version-fox/vfox-cmake) 在三个界面里也提供这些功能：

- **名称。** 版本要完整写出，如 `3.31.6`；候选版写成 `4.5.0-rc1`。`3.31` 这样的版本线不是发行版，安装会失败。`latest` 安装最新的正式版。
- **候选版。** 插件把候选版排在它所属的正式版之后；还没有正式版的候选版排在最后，所以只要列表里有正式版，`latest` 就是正式版。**候选版**筛选为 `rc1` 到 `rc6` 各设一个 Chip，保留以它结尾的版本。
- **探测并链接。** 会纳入 `PATH`、`CMAKE_HOME`、常见的系统位置里的 CMake，以及 Homebrew、Scoop 或 Chocolatey 装的 CMake。`CMAKE_HOME` 请在*设置 → 环境变量*里以 `KEY=VALUE` 的形式设置：应用会把这一列表传给每条 vfox 命令。
- **要求。** 插件需要 vfox 1.1.0 或更高版本，并要能访问 jsDelivr（版本列表）与 cmake.org（安装包），没有可设置的镜像。它不读取变量，也没有旧版版本文件，所以**选项**页签只有说明、可安装的名称与文档。

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
- **网页界面**：vfox-plus 是否同时向本机浏览器提供它的界面，没有窗口时是否继续运行
  （轻量模式），以及同一网络中的手机和其他电脑是否可以通过 HTTPS 连接。见
  [网页界面与其他设备](remote.md)。
- **vfox**：用哪个可执行文件（自动探测，也可自选）与下载镜像等额外环境变量。
- **vfox 配置**：代理、仓库源、存储目录、旧版版本文件、缓存与超时、诊断（`diagnostics.onUse`）与 PATH 顺序（默认 `path.mode` 为 `baseline`，`path.preserve`）——开关、菜单与
  文本框；清空文本框即恢复默认。别名（`vfox alias`）与链接版本（`vfox link`）从 vfox 读取。
- **关于**：版本、源码与依赖许可证。

## 找不到 vfox 时

找不到 vfox 可执行文件，或找到的不能用、版本早于 vfox 1.1.0 时，每个页面都会说明如何安装 vfox，并提供
定位可执行文件与重试。设置页始终可用，可手动指向 vfox。
