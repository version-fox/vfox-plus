# vfox+

> [English](../../en/dev/implementation-mygo.md)

vfox-plus 使用 [mygo](https://github.com/egoist/mygo)（Go 桌面应用框架）及其
**原生用户界面**（`ui` 包）构建：无 Web 视图、无 HTML、无 JavaScript。mygo 从
flexbox 布局的元素出发，用自研渲染器绘制窗口，经 GTK 3 打开，无需 C 编译器：
经 [purego](https://github.com/ebitengine/purego) 调用 GTK、GLib、Pango 与 Cairo，
故应用是仅链接 C 库、启动时加载 GTK 3 的约 12 MB 单 Go 可执行文件。mygo 没有
Material 3 Expressive 观感的部件；其元素与无样式部件之上全部为本项目 Go 手写：

| 目录 | 职责 |
| --- | --- |
| [`core/`](../../../mygo/core) | 模型、进程运行器、vfox 客户端及其输出解析、设置存储、带 ICU 消息的翻译目录、插件配置档案（`core/profile`：vfox-java、vfox-nodejs 与 vfox-flutter 的特有之处）——无 UI |
| [`state/`](../../../mygo/state) | 可观察控制器、任务队列（一次一个改变状态的命令，流式输出、下载进度、取消、重试） |
| [`ui/`](../../../mygo/ui) | 应用：窗口、带导航的 shell、页面、对话框与端到端场景 |
| [`ui/kit/`](../../../mygo/ui/kit) | 工具包：基于 mygo 元素与无样式部件的 Material 3 Expressive 组件 |
| [`ui/mcu/`](../../../mygo/ui/mcu) | 配色：Material 颜色工具的 Go 移植（HCT、CAM16、tonal spot） |
| [`ui/shapes/`](../../../mygo/ui/shapes) | 形状：`androidx.graphics.shapes` 圆角多边形及其形变的 Go 移植 |
| [`assets/`](../../../mygo/assets) | 二进制嵌入物：图标、翻译与许可证（构建时收集） |
| [`pair/`](../../../mygo/pair) | `vfox-plus pair` / `vfox-plus serve`：浏览器配对到本机（一次性链接），或为容器与远端实例常驻服务；配对后的浏览器得到完整 Web 界面（`/app`：SDK、仓库、带实时日志的任务、设置） |
| [`main.go`](../../../mygo/main.go) | 组装模型与窗口；分发 `pair` 子命令 |

分层见[架构(architecture.md)：`core` 不知道其它层，`state` 只知道
`core`，`ui` 在上。`core`、`state`、`ui/mcu` 与 `ui/shapes` 是无 UI 工具包的纯 Go，
解析器跑 [`fixtures`(fixtures.md) 通过。

## 依赖

- Go（锁定在 [`versions.env`](../../../mygo/versions.env)）；mygo 需要 Go 1.27。

- 运行时 GTK 3、fontconfig 与字体，以及 vfox 1.0.0 或更高：

  ```sh
  sudo apt install libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk
  ```

- C 编译器仅单元测试竞态检测需要（`go test -race` 需 cgo）；应用构建不用。

怎样在 Ubuntu 26.04（也包括 WSL2）上装齐这些，见[开发环境](environment.md)。

## 日常命令

`vfxtool` 是构建、测试、打包应用的单一 Go 工具
（`../shared/tooling/cmd/vfxtool`）。构建一次放 `PATH`，或每条命令前加
`go -C ../shared/tooling run ./cmd/vfxtool`：

```sh
go -C ../shared/tooling build -o ~/.local/bin/vfxtool ./cmd/vfxtool
```

| 任务 | 命令 |
| --- | --- |
| 收集二进制嵌入的图标、翻译与许可证 | `vfxtool prepare`（`MYGO_SKIP_LICENSES=1` 跳过需 `go mod download` 的 Go 模块许可证） |
| 构建（`build/vfox-plus`） | `vfxtool build build/vfox-plus` |
| 构建无头服务（`build/vfox-plus-serve`） | `vfxtool build --package ./cmd/vfox-plus-serve build/vfox-plus-serve`（同样的 `pair`、`serve`、`service` 命令，无用户界面，小约三分之一） |
| 交叉编译 Windows | `vfxtool build --goos windows --goarch amd64\|arm64 build/vfox-plus.exe` |
| 运行 | `vfxtool build build/vfox-plus && build/vfox-plus` |
| 格式检查（CI 有差异即失败） | `golangci-lint fmt --diff` |
| Lint（CI 有发现即失败） | `golangci-lint run` 与 `golangci-lint run --build-tags e2e` |
| 单元测试 | `CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./...` |
| 覆盖率门禁 | `vfxtool coverage coverage.out 85` |
| `.deb`、`.tar.gz`、`.msix`、`.msixbundle` | 见[打包](#打包) |

`vfxtool prepare` 把 [`shared/icons.json`](../../../shared/icons.json) 的图标及其 SVG
（[`shared/icons`](../../../shared/icons)，描边与填充各一，见
[用户界面如何工作](#用户界面如何工作)）、[`shared/i18n`](../../../shared/i18n) 的翻译目录复制到
`mygo/assets` 并在那里写许可证；只有 Go 模块许可证需要的 `go mod download` 才用网络。结果不提交；
[`assets/assets.go`](../../../mygo/assets/assets.go) 的 `//go:embed` 缺失即构建失败。

发布二进制用 `CGO_ENABLED=0`、位置无关（`-buildmode=pie`，Windows 除外）、strip、
关 mygo 原生界面 inspector，约 12 MB。

## 沙箱

`vfxtool sandbox` 在一次性 Docker 容器跑 CI 检查、端到端测试与打包。只需 Docker，
不改变本机：镜像带 Go、golangci-lint、GTK 3、Xvfb 与 vfox（扩展共享基础镜像，见
[`shared/sandbox`(sandbox.md)），只读看仓库，在缓存卷保留
Go 模块与构建缓存。

| 命令 | 作用 |
| --- | --- |
| `vfxtool sandbox check` | 格式化、golangci-lint（含端到端标签）、`go vet`、带竞态检测的单元测试与覆盖率门禁 |
| `vfxtool sandbox e2e` | 端到端场景，夹具与真实 vfox 各一 |
| `vfxtool sandbox package` | 发布构建与冒烟测试、`.deb`、`.tar.gz`、`.msix`/`.msixbundle`（lintian 检查），再装入干净 Ubuntu 26.04 容器 |
| `vfxtool sandbox measure` | Home 页与列出 Node.js 版本时的内存 |
| `vfxtool sandbox screenshots` | 各页截图，以及 Java、Flutter 与 Node.js 选项的截图 |
| `vfxtool sandbox all` | `check`、`e2e` 与 `package` |
| `vfxtool sandbox shell` | 沙箱内交互 shell |
| `vfxtool sandbox clean` | 删除镜像、缓存卷与结果 |

结果写入 `dist/sandbox/mygo/<命令>/`。默认服务器慢时设 `SANDBOX_GOPROXY`
（如 `https://goproxy.cn`）与 `SANDBOX_UBUNTU_MIRROR`（Ubuntu 归档）。

## 测试

- **单元测试**覆盖 `core`、`state`、配色与形状。门禁：`core` 与 `state` 合计语句
  85%，`ui` 75%，其组件与页面测试把 `ui` 带到约 90%。
  解析器跑 [`shared/fixtures`(fixtures.md) **全部**用例。
- **组件与页面测试**无窗口跑 mygo 视图，用其 `ui.Tester` 在内存布局绘制、点击、输入、
  推进自有时钟（[`ui/kit`](../../../mygo/ui/kit/behaviour_test.go)）。
  [`ui/app_test.go`](../../../mygo/ui/app_test.go) 对捕获的 vfox 输出跑完整应用，每流一测：Home、
  导航、SDK 列表与详情、安装、使用、卸载、仓库、任务、设置、窄布局、vfox 缺失。设
  `VFOX_PLUS_SNAPSHOTS=<目录>` 让测试写页面 PNG。
- **Golden 测试**比较 Go 配色形状与移植来源 JavaScript 库。[`ui/mcu/testdata`](../../../mygo/ui/mcu/testdata)
  存 82 种子色双主题配色，bit 一致；[`ui/shapes/testdata`](../../../mygo/ui/shapes/testdata) 存形状轮廓，
  尺寸 2% 内一致。各 `testdata` 说明重生成方法：配色需 Node.js 跑 JS 参考实现，形状
  `go run generate.go` 即可，无需 Node.js。
- **端到端场景**在 `VFOX_PLUS_E2E_SCENARIO` 置位时跑在构建的应用中：`fake` 对夹具假 vfox，
  `real` 对带离线测试插件及其服务器的真实 vfox，`measure` 测内存，`screenshots` 截图。代码
  在 [`ui/e2e*.go`](../../../mygo/ui)，仅 `e2e` 构建标签编译：

  ```sh
  vfxtool build --e2e build/vfox-plus-e2e
  xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- vfxtool e2e-run fake build/vfox-plus-e2e
  VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- \
    vfxtool e2e-run real build/vfox-plus-e2e
  ```

  `vfxtool e2e-run` 准备全新空家目录、假 vfox 或带离线测试插件及服务器的真实 vfox，
  启动应用读其 `e2e: …` 报告；场景通过应用 exit 0。场景在两帧之间主线程点按、填写、
  读取组件注册的节点（角色、名称、点击或输入作用），走真实组件、状态、进程 spawn、
  流式、带 `SIGTERM`（Windows 上 kill）的取消、设置文件与 vfox 本体。
  `vfxtool screenshots` 跑三遍 `screenshots` 场景（见[截图](contributing.md#截图)）。

## 用户界面如何工作

mygo 每帧有变化即调用一次 state 的 view 函数，视图重建窗口元素：容器
（`ui.Row`、`ui.Column`、`ui.Scroll`、`ui.List`）、文本、无观感但处理指针键盘焦点
的部件（`ui.ButtonBase`、`ui.SwitchBase`、`ui.TextInputBase`、`ui.PopoverBase`、
`ui.DialogBase`、`ui.TooltipBase`、`ui.ToastViewportBase`）。帧间保持靠 mygo 在元素
身份（兄弟中位置或 key）下存续。[`ui/kit`](../../../mygo/ui/kit) 在其上实现 Material 3 Expressive
观感，页面由其组件搭建。

- **组件**是 `kit.Ctx`（mygo 上下文与窗口环境）的函数，就地构建元素并返回最外层，
  调用者用元素方法排布（`Grow`、`Margin`、`Width` 等）：`kit.Button(k, kit.ButtonSpec{…})`。
  动作回调在帧构建后由 mygo 执行。
- **动效**是每动画值一阻尼弹簧，Material 3 Expressive 动效方案的 spatial 与 effects
  弹簧（[`motion.go`](../../../mygo/ui/kit/motion.go)），交 mygo `AnimateTo`：精确解，中途变目标保持
  速度，只在动时绘制，静止窗口空闲。桌面要求减少动态效果则值直达。对话框菜单 pose
  弹簧弹出，页面让路 mygo 元素转场（fade through、shared axis）。加载与波浪进度在
  painter 动，不重建视图。
- **浮层**——菜单、tooltip、对话框、snack bar——是 mygo popover、tooltip、dialog 与
  toast，管焦点、Escape 与外部按压；kit 定样式。
- **列表**虚拟：只构建可见行，记行高，千像素跳跃与单步同价。Node.js 版本列表 872 项。
- **registry**（[`registry.go`](../../../mygo/ui/kit/registry.go)）是某次构建组件注册节点的扁平表：
  角色、名称、状态、所在列表项或对话框、点击或输入作用。端到端场景用它找操作部件；
  不是无障碍树（mygo 给屏幕阅读器另有一套，见[局限](#局限)）。
- **颜色**运行时从种子色或桌面强调色按 Material 颜色工具（[`ui/mcu`](../../../mygo/ui/mcu)，
  tonal spot 2021）生成。mygo 自有部件（文本框、滚动条、tooltip）经其主题跟随。
- **形状**——头像、统计块、swatch、加载指示器——是 Material 3 Expressive 多边形与形变，
  按路径绘制（[`ui/shapes`](../../../mygo/ui/shapes)）。
- **文本**由 Pango 用桌面 sans-serif 排版绘制：Ubuntu 上 Ubuntu Sans，中文字体按系统
  选（Noto Sans CJK），字重按 type scale，桌面文本缩放生效。
- **图标**为 [`shared/icons`](../../../shared/icons) 里 Material Symbols Rounded 的 SVG，由 mygo 的
  `ui.Icon` 以文本颜色绘制，每个符号一个描边、一个填充；导航与按钮的填充动画是两者的交叉淡入淡出
  （[`ui/kit/icon.go`](../../../mygo/ui/kit/icon.go)）。SVG 在首次绘制该图标时才解析。

## 打包

发布二进制自包含一切。

Linux 上，共享工具 `vfxtool build-deb` 打包为
`vfox-plus_<版本>_amd64.deb` 与 `.tar.gz`，含
[`shared/packaging/linux`](../../../shared/packaging/linux) 的桌面入口、AppStream 元数据与图标；
版本取 [`VERSION`](../../../mygo/VERSION)，有测试保证与应用显示一致。应用窗口名
`dev.vfox.VfoxPlus`（Wayland 应用 ID 与 X11 窗口类取自程序名，应用在 mygo 启动 GTK 前
用 `g_set_prgname` 设置），与桌面入口对应。Go 模块、Go 标准库、移植代码与图标
许可证装为 `/usr/share/doc/vfox-plus/THIRD-PARTY-LICENSES` 并在应用内展示
（[`NOTICE.txt`](../../../mygo/NOTICE.txt) 存移植声明；其余 `vfxtool golicenses` 收集，
  不兼容 Apache License 2.0 即失败）。

包依赖 GTK 3（含 mygo 加载的 GLib、Pango、Cairo 等）、fontconfig 与字体
（`fonts-ubuntu` 或 `fonts-noto-core`），推荐中文 `fonts-noto-cjk`、桌面门户与打开链接
的 `xdg-utils`。列表即 `vfxtool runtask` 的 `Depends`
（`shared/tooling/internal/runtask`），CI 装入干净 Ubuntu 容器证明完备。

Windows 11 上，`vfxtool build-msix` 把交叉编译的可执行文件
（`vfxtool build --goos windows --goarch amd64|arm64`）打包为未签名
`vfox-plus_<版本>_<架构>.msix` 与
`vfox-plus_<版本>.msixbundle`，含
[`shared/packaging/windows`](../../../shared/packaging/windows) 的清单、徽标与 bundle 布局。
用户在管理员 PowerShell 用 `Add-AppxPackage -AllowUnsigned` 安装未签名 bundle；
安装包目前尚未签名，日后会提供官方签名的安装包。
CI 在 Linux 交叉编译打包并用 `vfxtool validate-msix` 校验，在 Windows Server 2025（x64）与 ARM 版 Windows 11（arm64）上校验并安装/卸载 bundle。

### Docker 镜像

[`Dockerfile`](../../../mygo/Dockerfile) 从仓库根构建发布镜像：最小 Ubuntu 运行时，含 vfox 与
无头服务（`vfox-plus-serve serve`，镜像入口），无需显示服务。工具链版本默认取
[`versions.env`](../../../mygo/versions.env)。

```sh
docker build -f mygo/Dockerfile -t vfox-plus .
docker run --rm -p 8080:8080 vfox-plus
```

打开它输出的配对链接，把 `0.0.0.0` 换成可达地址。链接是 `https`：容器首次启动
生成自签名证书并打印指纹，请在浏览器核对。`-v vfox-plus-data:/home/vfoxplus`
卷在多次运行之间保留已安装的 SDK。另一台机器的浏览器在容器可达的网络上同样配对。

## 内存

`vfxtool sandbox measure` 测量：Docker 内 Ubuntu 26.04，Xvfb，1280×800 窗口，单进程，
发布构建。

| 状态 | RSS | PSS |
| --- | ---: | ---: |
| Home 页 | 58 MB | 45 MB |
| 列出 Node.js 版本 | 67 MB | 53 MB |

mygo 用处理器绘制，应用阻止 GTK 加载 OpenGL：GDK 打开 OpenGL 连接即映射 Mesa，软件驱动
再加其编译器 LLVM，比应用本身还大（同构建 GDK 打开时 Home 页 PSS 约 78 MB，关闭约
43 MB，其它相同）。这种体量界面 CPU 绘制每帧零点几毫秒，不需要 GPU，内存不依赖本机
驱动。剩下主要是带区域数据的可执行文件、Go 堆、GTK 及其 C 库。版本列表只构建可见行，
增加 8 MB。

## 局限

- **OpenGL 故意关闭。**mygo 有 GPU 即走 OpenGL 绘制，加载 Mesa，软件驱动再加其编译器
  LLVM：GDK 打开 OpenGL 连接即映射约 35 MB PSS。vfox+ 页面 CPU 绘制零点几毫秒，故应用
  在启动 GTK 前设 `MYGO_GPU=0` 与 `GDK_GL=disable`，走 CPU，除非用户设 `MYGO_GPU`
  （`MYGO_GPU=1` 走 OpenGL）。缩放进入的对话框与缩放的窗口是 CPU 最贵的帧；今日机器一帧
  内画完。
- **标题栏是桌面的。**GTK 画标题栏（GNOME Wayland 客户端侧、X11 窗口管理器），随桌面风格；
  mygo 可把标题栏交给应用，本实现暂未用。
- **缩放**走 GTK 3：整数倍，来自 `GDK_SCALE` 或桌面，加桌面文本缩放。
- **mygo 年轻**（写作时 v0.6.3，API 尚变）：版本锁 [`go.mod`](../../../mygo/go.mod)，页面跟现行版本。
- **无障碍**走 mygo ATK 对象，GTK 桥到 AT-SPI，Orca 读窗口；场景用的 registry 不是它。
  kit 组件给元素命名（图标按钮 tooltip 即其名），mygo 读屏支持未在此用 Orca 试过。
- **输入法**走 GTK 3（IBus、Fcitx、Wayland text input）。
- **Linux x86-64 跑全、测全、打包。**mygo 另有 Windows 与 macOS 后端，应用可为它们及
  Linux arm64 编译（`vfxtool build --goos windows --goarch amd64` 等）。Windows 11 打包为
  未签名 `.msixbundle`（x64 与 arm64），CI 校验；安装为 `Add-AppxPackage -AllowUnsigned`
  侧加载。`core` 已跨平台：进程终止（Unix `SIGTERM`、Windows kill）、`PATH` 分隔
  （`:` 与 `;`）、vfox 查找（`vfox` 与 `vfox.exe` 及 Windows 回退目录）、设置位置
  （`~/.config` 与 `%APPDATA%`）。
- **`.apk` 在 `android/`，不在此。**mygo 无 Android 后端，界面不能是由本代码构建的
  原生 Android 应用。Android 客户端是包住配对服务（`vfox-plus serve`）的
  WebView：手机经[远端协议](remote-protocol.md)驱动 PC 上的 vfox-plus 实例，
  所示 Web 界面遵循[设计系统](design-system.md)（见
  [远端前端](architecture.md#远端前端)）。
