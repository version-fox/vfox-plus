# 架构

> [English](../architecture.md)

vfox-plus 是 [vfox](https://vfox.dev) 命令行工具的图形化前端。它使用
[mygo](https://github.com/egoist/mygo)（Go 桌面应用框架）及其原生用户界面构建：
无 Web 视图、无 HTML、无 JavaScript。mygo 从 flexbox 布局的元素出发，用自研渲染器
绘制窗口，经 GTK 3 打开，无需 C 编译器。

## 仓库布局

```text
vfox-plus/
├── docs/                  规约：架构、vfox CLI 契约、界面、设计系统、国际化
│   ├── architecture.md    本文档
│   ├── vfox-cli-contract.md  GUI 如何驱动 vfox：命令、环境、解析
│   ├── ui-spec.md         界面、流程和状态
│   ├── design-system.md   Material 3 Expressive 标记、组件和动效
│   ├── i18n.md            翻译与术语
│   ├── memory.md          应用程序的内存占用，同一方法测量
│   └── zh-CN/             上述文档的简体中文译文
├── shared/                应用使用的机器可读资源
│   ├── i18n/              翻译目录（ARB：带 ICU 消息的 JSON）
│   ├── icons.json         界面使用的 Material Symbols
│   ├── fixtures/          捕获的 vfox 输出及期望解析结果
│   ├── test-plugin/       对真实 vfox 做端到端测试的离线 vfox 插件
│   ├── packaging/linux/   桌面入口、AppStream 元数据和图标，`vfxtool build-deb` 打包
│   ├── packaging/windows/ MSIX 清单、徽标和 bundle 布局，`vfxtool build-msix` 为 Windows 11 打包
│   ├── tooling/           `vfxtool`：构建、测试、打包一切的 Go 工具
│   └── sandbox/           Docker 沙箱的共享部分
├── mygo/                  应用程序（Go）
└── .github/               CI 工作流：仓库检查、构建、内存报告和沙箱
```

应用从 `../shared` 读取共享资源，从不复制，保证翻译或夹具只改一处。

## 分层

应用分三层，依赖只向下。

```text
┌────────────────────────────────────────────────────────────────────┐
│ UI         页面、组件、主题、动效、平台集成                        │
├────────────────────────────────────────────────────────────────────┤
│ State      可观察控制器：设置、vfox 可用性、                        │
│            已安装 SDK、插件仓库、配置、任务队列                    │
├────────────────────────────────────────────────────────────────────┤
│ Core       模型、进程运行器、vfox 客户端与输出解析、                 │
│            设置存储——无 UI 框架                                    │
└────────────────────────────────────────────────────────────────────┘
                                  │ 子进程
                                  ▼
                           vfox 命令行
```

**Core** 是不依赖 UI 框架的纯 Go，包含：

- 领域模型（`InstalledSdk`、`PluginInfo`、`AvailablePlugin`、
  `AvailableVersion`、`ConfigEntry`、`UseScope`）；
- *进程运行器*接口，按平台进程 API 实现；测试用脚本化替身，未来远端后端（见
  下）可提供另一实现；
- *vfox 客户端*：组装命令行、运行只读查询并解析输出，把改变状态的操作描述为
  *命令*（值）而不直接执行；
- *设置存储*；
- 带 ICU 消息的*翻译目录*（`core/i18n`）。

它用 `shared/fixtures` 测试。

**State** 把 core 结果变成 UI 可观察状态。最重要的是*任务队列*：每个改变状态的
命令（安装、卸载、使用、取消使用、添加、移除、更新）都排队**一次执行一个**，
因为 vfox 不锁数据目录。每个任务流式输出进程内容、提取下载进度、保留有界日志
并总结失败。任务结束重载已安装 SDK。

**UI** 按[界面规约](ui-spec.md)和[设计系统](design-system.md)渲染状态。桌面集成
（浏览器、文件选择器、剪贴板、强调色）藏在小*平台服务*接口后，UI 测试可替换。
Material 3 Expressive 组件在 `ui/kit`，基于 mygo 元素与无样式部件；
配色（`ui/mcu`，Material 颜色工具的 Go 移植）与形状（`ui/shapes`，
`androidx.graphics.shapes` 的 Go 移植）和 `core`、`state` 一样是无 UI 框架的纯 Go。

源码中三层是 `mygo/core`、`mygo/state`、`mygo/ui`
（`ui/kit`、`ui/mcu`、`ui/shapes` 属 UI 层）；`core` 不知道其它层，`state` 只知道
`core`。

## 与 vfox 对话

GUI 从不直接读写 vfox 文件，一切走命令行，见
[CLI 契约](vfox-cli-contract.md)。这让 GUI 跨 vfox 版本保持正确，也让 vfox 保持
自身不变式。契约还列出解析器必须容忍的现行 vfox 输出 quirks。

只读查询并发跑且便宜（每次约 10 ms）；只有 `vfox search` 和 `vfox available`
可能联网。

## 设置

GUI 自身设置存为 JSON：

- Linux：`$XDG_CONFIG_HOME/vfox-plus/settings.json`（默认
  `~/.config/vfox-plus/settings.json`）；
- Windows：`%APPDATA%/vfox-plus/settings.json`。

| 键 | 类型 | 默认 | 含义 |
| --- | --- | --- | --- |
| `theme` | string | `"system"` | `system`、`light` 或 `dark`。 |
| `useSystemColor` | boolean | `true` | 从桌面强调色派生颜色。 |
| `seedColor` | integer (ARGB) | `0xFFF28C1B` | 不用强调色时的种子颜色。 |
| `language` | string | `"system"` | `system`、`english` 或 `simplifiedChinese`。 |
| `vfoxExecutable` | string | 无 | vfox 路径；无表示自动探测。 |
| `extraEnvironment` | object | `{}` | 给 vfox 的额外环境变量，如镜像。 |
| `railExpanded` | boolean | `false` | 导航栏是否显示标签。 |

未知键必须保留或忽略，不得拒绝。

vfox 配置（代理、仓库、存储目录等）**不是**这些设置的一部分：它属于 vfox，经
`vfox config` 编辑。

## 查找 vfox

未配置可执行文件时，GUI 先搜 `PATH`，再搜桌面启动器 `PATH` 常缺的常见安装位置：

- Linux：`/usr/local/bin`、`/usr/bin`、`~/.local/bin`、`~/bin`、
  `~/.linuxbrew/bin`、`/home/linuxbrew/.linuxbrew/bin` 和 `/snap/bin`，找 `vfox`；
- Windows：`%ProgramFiles%/vfox`、`%LOCALAPPDATA%/Programs/vfox`、
  `%LOCALAPPDATA%/Microsoft/WindowsApps`、`~/scoop/shims` 和 `~/.vfox/bin`，找
  `vfox.exe`（`PATH` 按 `;` 分隔）。

用户也可自选可执行文件。vfox 缺失或损坏时，GUI 说明如何安装，并保持设置可达。

## 依赖

运行时依赖必须属于 Apache 软件基金会许可证分类的 A 或 B 类
（见 https://www.apache.org/legal/resolved）：ASF 产品允许的即可——宽松许可证
（如 Apache-2.0、MIT、BSD、Unlicense）或弱 copyleft 许可证（如 MPL-2.0、
EPL-2.0）。X 类（GPL、AGPL 等强 copyleft）排除在外。
B 类依赖按自身许可证及其条件发布（声明保留、源码可得）；应用原样链接。应用框架
mygo 为 MIT 许可证，其链接的 Go 模块 purego 为 Apache License 2.0，文本塑形
`go-text/typesetting` 为 Unlicense 或 BSD，`golang.org/x` 模块与 Go 标准库为 BSD；
`mygo/ui/mcu` 与 `mygo/ui/shapes` 中的移植代码来自 Apache License 2.0 与 MIT 许可证
的库，声明随应用发布（[`mygo/NOTICE.txt`](../../mygo/NOTICE.txt)）。应用不链接 LGPL 的
GTK 3、GLib、Pango 和 Cairo：mygo 启动时从系统加载，包按
`vfxtool runtask`（`shared/tooling/internal/runtask`）声明依赖。应用不含 X 类
强 copyleft 代码；若有 B 类弱 copyleft 依赖，按其许可证发布。`shared/tooling`
中的构建工具（`vfxtool`，不随应用发布）还用 `go-github`（内存报告的 GitHub API）、
`nfpm`（Debian 打包库）、`andybalholm/brotli`（图标字体的 WOFF2 压缩）、
`go-licenses`（许可证分类）与 `go-getter`（包预取的断点续传），均为 A 类。
Windows `.msix` 与 `.msixbundle` 无额外运行时依赖：Go 可执行文件自包含，包自带徽标。
优先用平台与技术官方库而非第三方包，依赖声明旁注明用途。

## 内存

低内存占用是项目目标。应用：

- 列表懒加载（仅 Node.js 版本列表就近 900 条）；
- 只打包使用的图标（两个图标字体子集共约 36 KB）；
- 不可见动画停止，尊重平台的减少动态效果设置；
- 不缓存 vfox 可廉价重产的数据。

[内存页](memory.md)记录应用占用。

## 未来：远端前端

连接跑 vfox 机器的 Web UI 与 Android 客户端仍可能。进程运行器与 vfox 客户端是接缝：
远端实现转发同样命令到机器上的 agent，状态与 UI 层不变。CLI 契约届时成为协议。
