# 架构

> [English](../../en/dev/architecture.md)

vfox-plus 是 [vfox](https://vfox.dev) 命令行工具的图形化前端。它使用
[mygo](https://github.com/egoist/mygo)（Go 桌面应用框架）及其原生用户界面构建：
无 Web 视图、无 HTML、无 JavaScript。mygo 从 flexbox 布局的元素出发，用自研渲染器
绘制窗口，经 GTK 3 打开，无需 C 编译器。

## 仓库布局

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
    accTitle: 仓库布局
    accDescr: docs 存放文档，shared 存放应用使用的资源，mygo 与 android 是应用，.github 是 CI 工作流。
    subgraph docs["<b>docs/</b> · 全部文档，先按语言、再按受众分类"]
        direction LR
        d1("<b>en/user/</b><br/>用户指南：安装、使用、<br/>远端与安卓") ~~~ d2("<b>en/dev/</b><br/>开发者规约<br/>（本文档在这里）") ~~~ d3("<b>zh-CN/user/</b><br/>简体中文用户指南") ~~~ d4("<b>zh-CN/dev/</b><br/>简体中文开发者规约")
    end
    subgraph shared["<b>shared/</b> · 应用使用的机器可读资源"]
        direction LR
        s1("<b>i18n/</b><br/>翻译目录<br/>（ARB：带 ICU 消息的 JSON）") ~~~ s2("<b>icons.json</b><br/>界面使用的<br/>Material Symbols") ~~~ s3("<b>fixtures/</b><br/>捕获的 vfox 输出<br/>及期望解析结果") ~~~ s4("<b>test-plugin/</b><br/>对真实 vfox 做端到端测试的<br/>离线 vfox 插件")
        s5("<b>packaging/linux/</b><br/>桌面入口、AppStream 元数据和图标，<br/><code>vfxtool build-deb</code> 打包") ~~~ s6("<b>packaging/windows/</b><br/>MSIX 清单、徽标和 bundle 布局，<br/>为 Windows 11 打包，<br/><code>vfxtool build-msix</code>") ~~~ s7("<b>tooling/</b><br/><code>vfxtool</code>：构建、测试、<br/>打包一切的 Go 工具") ~~~ s8("<b>sandbox/</b><br/>Docker 沙箱的共享部分")
    end
    subgraph code["<b>应用</b>与<b>自动化</b>"]
        direction LR
        m("<b>mygo/</b><br/>桌面应用及其配对服务（Go）；<br/><code>cmd/vfox-plus-serve</code> 构建<br/>无界面 headless 服务") ~~~ a("<b>android/</b><br/>安卓远端客户端<br/>（配对服务的 WebView 套壳）") ~~~ g("<b>.github/</b><br/>CI 工作流：仓库检查、构建、<br/>内存报告和沙箱")
    end
    docs ~~~ shared ~~~ code

    classDef docs fill:#E4E0FF,stroke:#5A4BE8,color:#1E1470,stroke-width:1.5px,rx:12,ry:12
    classDef shared fill:#FFE9D0,stroke:#F28C1B,color:#3B2200,stroke-width:1.5px,rx:12,ry:12
    classDef app fill:#D9F2EA,stroke:#13836B,color:#06332A,stroke-width:1.5px,rx:12,ry:12
    classDef ci fill:#ECEBF3,stroke:#8E8BA8,color:#2B2840,stroke-width:1.5px,rx:12,ry:12
    class d1,d2,d3,d4 docs
    class s1,s2,s3,s4,s5,s6,s7,s8 shared
    class m,a app
    class g ci
    style docs fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
    style shared fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
    style code fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
```

应用从 `shared/` 读取共享资源，从不复制，保证翻译或夹具只改一处。

## 分层

应用分三层，依赖只向下。

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
    accTitle: 应用的分层
    accDescr: UI 依赖 State，State 依赖 Core，Core 以子进程运行 vfox 命令行。
    subgraph UI["<b>UI</b> · mygo/ui"]
        direction LR
        ui1(["页面"]) ~~~ ui2(["组件<br/>ui/kit"]) ~~~ ui3(["主题<br/>ui/mcu · ui/shapes"]) ~~~ ui4(["动效"]) ~~~ ui5(["平台集成"])
    end
    subgraph State["<b>State</b> · mygo/state"]
        direction LR
        st1(["设置"]) ~~~ st2(["vfox 可用性"]) ~~~ st3(["已安装 SDK"]) ~~~ st4(["插件仓库"]) ~~~ st5(["配置"]) ~~~ st6(["任务队列"])
    end
    subgraph Core["<b>Core</b> · mygo/core · 无 UI 框架"]
        direction LR
        co1(["模型"]) ~~~ co2(["进程运行器"]) ~~~ co3(["vfox 客户端与<br/>输出解析"]) ~~~ co4(["设置存储"])
    end
    UI --> State --> Core
    Core -->|"子进程"| vfox(["vfox 命令行"])

    classDef ui fill:#E4E0FF,stroke:#5A4BE8,color:#1E1470,stroke-width:1.5px
    classDef state fill:#FFE9D0,stroke:#F28C1B,color:#3B2200,stroke-width:1.5px
    classDef core fill:#D9F2EA,stroke:#13836B,color:#06332A,stroke-width:1.5px
    classDef ext fill:#17123F,stroke:#17123F,color:#FFFFFF,stroke-width:1.5px
    class ui1,ui2,ui3,ui4,ui5 ui
    class st1,st2,st3,st4,st5,st6 state
    class co1,co2,co3,co4 core
    class vfox ext
    style UI fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
    style State fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
    style Core fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
    linkStyle default stroke-width:2px
```

**Core** 是不依赖 UI 框架的纯 Go，包含：

- 领域模型（`InstalledSdk`、`PluginInfo`、`AvailablePlugin`、
  `AvailableVersion`、`ConfigEntry`、`UseScope`）；
- *进程运行器*接口，按平台进程 API 实现；测试用脚本化替身，未来远端后端（见
  下）可提供另一实现；
- *vfox 客户端*：组装命令行、运行只读查询并解析输出，把改变状态的操作描述为
  *命令*（值）而不直接执行；
- *设置存储*；
- 带 ICU 消息的*翻译目录*（`core/i18n`）；
- *插件配置档案*（`core/profile`），见下。

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

## 插件配置档案

vfox 对每个插件都用同样的方式列出、安装、选择与移除版本，但 version-fox 组织的插件提供的更多，
这些在 vfox 的输出里一点也看不出来：`vfox search java tem` 列出 Java 的一个发行版，而 `vfox search java`
只列出 OpenJDK；`nodejs@24` 安装最新的 Node.js 24，`flutter@stable` 安装某个通道的版本；每个插件还会读取
把下载指向镜像的环境变量与 `.nvmrc` 之类的版本文件。不了解这些，界面只能提供所有插件共有的功能，
第一版的 Web 界面就是这样只显示了 Java 的 OpenJDK 列表（[#43](https://github.com/version-fox/vfox-plus/issues/43)）。

*配置档案*（`core/profile`）把这些写成三个界面共用的纯数据，覆盖 `java`、`nodejs` 与 `flutter`：

| 部分 | 内容 |
| --- | --- |
| 搜索 | `vfox search` 里插件名后面的参数：`java` 有 Foojay 的 31 个发行版与 `all`（Linux loong64 上改为 Loongnix JDK），以及 vfox 不带参数时用的那个 |
| 简写 | `vfox install <插件>@<名称>` 除列出的版本外接受的名称 |
| 变量 | 插件读取的环境变量，附已检查过的镜像，以及每个变量接受什么（`http://` 或 `https://` 地址，或少数几个值） |
| 版本文件 | vfox 通过该插件读取的文件，以及 vfox 设置 `legacyVersionFile.strategy` 对它们是否有影响 |
| 筛选维度 | 适合已列出版本的备注与名称的筛选：`LTS`、`JavaFX`、Flutter 的通道与架构 |
| 命名 | 版本名称的构成，所以 `21.0.12.1+1-tem` 显示 Temurin，`3.47.7-x64` 显示 x64 |
| 说明、须知、文档 | 几句关于名称的说明、容易让人意外的地方，以及该插件文档的链接 |

档案里的文字用两种语言写在数据旁边，而不放进翻译目录，因为 Web 界面通过 HTTP 获取它们，无头服务器也不带
翻译目录。没有档案的插件行为与从前相同。注册表是 `state.AppModel.Profiles`；端到端测试为它们的离线测试
插件注册了一份档案。

`SdkDetailController` 保存所选的搜索（`SetSearch` 会忘掉之前的列表与筛选维度；有人要列表时才加载）与所选
筛选维度；`AppModel.SetPluginVariable` 按变量检查一个值，并把它存入设置里的额外环境变量，每个 vfox 进程
都带着它运行。

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
二维码编码器 `skip2/go-qrcode`（MIT）把 `vfox-plus serve` 的配对链接打印成终端
二维码供安卓 App 扫描；`mygo/ui/mcu` 与 `mygo/ui/shapes` 中的移植代码来自 Apache License 2.0 与 MIT 许可证
的库，声明随应用发布（[`mygo/NOTICE.txt`](../../../mygo/NOTICE.txt)）。应用不链接 LGPL 的
GTK 3、GLib、Pango 和 Cairo：mygo 启动时从系统加载，包按
`vfxtool runtask`（`shared/tooling/internal/runtask`）声明依赖。应用不含 X 类
强 copyleft 代码；若有 B 类弱 copyleft 依赖，按其许可证发布。`shared/tooling`
中的构建工具（`vfxtool`，不随应用发布）还用 `go-github`（内存报告的 GitHub API）、
`nfpm`（Debian 打包库）、`andybalholm/brotli`（图标字体的 WOFF2 压缩）、
`go-licenses`（许可证分类）、`go-getter`（包预取的断点续传）、`golang.org/x/image`
（绘制 Windows 徽标的矢量光栅化库，BSD-3-Clause）与 `golang.org/x/sys`
（`vfxtool sign-msix` 的 Windows 调用：提权用的令牌、证书库；BSD-3-Clause），均为 A 类。
Windows `.msix` 与 `.msixbundle` 无额外运行时依赖：Go 可执行文件自包含，包自带徽标。
优先用平台与技术官方库而非第三方包，依赖声明旁注明用途。

## 内存

低内存占用是项目目标。应用：

- 列表懒加载（仅 Node.js 版本列表就近 900 条）；
- 每个已完成任务只在内存保留最新 500 行（运行时保留 2000 行），已完成任务
  最多记 50 条；失败任务的完整日志落盘，只保留最新 20 个文件；
- 只打包使用的图标（两个图标字体子集共约 36 KB）；
- 不可见动画停止，尊重平台的减少动态效果设置；
- 不缓存 vfox 可廉价重产的数据。

[内存页](memory.md)记录应用占用。

## 远端前端

每个用户一个 server 拥有应用状态：桌面应用、后台 daemon 或回环 `serve`
把自己登记在设置旁的 `server.json` 里，其它各方挂靠它，不再另起模型。`pair` 从活着的 server 增发一次性链接；只有无 server 的
headless `pair` 才起自己的只读快照服务。

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
    accTitle: pair 与 serve 由哪个 server 应答
    accDescr: 有共享 server 在运行时，pair 与回环 serve 都用它；没有时，pair 起只读快照服务，serve 起自己的模型并成为共享 server，桌面与 service start 也是如此。监听其它网卡的 serve 一律起自己的 HTTPS 服务。浏览器与安卓 App 与应答的 server 配对。
    pair(["<b>pair</b>"]) --> q1{"有共享 server<br/>在运行？"}
    srv(["<b>serve</b><br/>回环监听"]) --> q2{"有共享 server<br/>在运行？"}
    rem(["<b>serve</b><br/>非回环监听"]) --> own("一律起自己的 server：<br/>HTTPS 与二维码，<br/>需要 <code>--allow-remote</code>")
    subgraph shared["<b>共享 server</b> · 每个用户至多一个，登记在 <code>server.json</code>"]
        direction LR
        d("<b>桌面</b><br/>开启远端访问且<br/>无 server 在运行：<br/>内嵌回环服务") ~~~ s("<b>service start</b><br/>分离的回环<br/><code>serve</code>") ~~~ l("<b>serve</b><br/>没找到共享 server<br/>的回环监听")
    end
    q1 -->|"是：增发链接"| shared
    q2 -->|"是：挂靠，<br/>增发链接"| shared
    q2 -->|"否：起自己的模型<br/>并登记"| shared
    q1 -->|否| snap("一次性只读快照服务<br/>（服务被禁用时<br/>pair 改为拒绝）")
    fe(["<b>浏览器与安卓 App</b><br/>用一次性链接配对"])
    shared --> fe
    own --> fe
    snap --> fe

    classDef cmd fill:#E4E0FF,stroke:#5A4BE8,color:#1E1470,stroke-width:1.5px
    classDef ask fill:#FFFFFF,stroke:#8E8BA8,color:#2B2840,stroke-width:1.5px
    classDef owner fill:#FFE9D0,stroke:#F28C1B,color:#3B2200,stroke-width:1.5px,rx:12,ry:12
    classDef alone fill:#ECEBF3,stroke:#8E8BA8,color:#2B2840,stroke-width:1.5px,rx:12,ry:12
    classDef client fill:#17123F,stroke:#17123F,color:#FFFFFF,stroke-width:1.5px
    class pair,srv,rem cmd
    class q1,q2 ask
    class d,s,l owner
    class own,snap alone
    class fe client
    style shared fill:#F4F2FF,stroke:#B9B3F5,stroke-width:1.5px,rx:16,ry:16
    linkStyle default stroke-width:2px
```

- **桌面**：在设置里打开远端访问（默认关闭）时运行时内嵌回环服务（`ServeEmbedded`），浏览器
  直接与桌面配对。原生界面不变：仍直连进程内模型，不走 HTTP。
- **`service`**：桌面没跑时 `service start` 起分离的回环 `serve` 做共享服务；
  `stop`、`status`、`set disabled true|false` 管理它。配对需要共享服务，禁用
  后不可用。
- **`serve`**：回环监听挂靠活 server（增发、打印、等待）；否则起自己的模型并
  登记。非回环监听一律起自己的服务（容器与远端用），HTTPS 与二维码如前——
  但远端访问默认关闭，无显式 `--allow-remote` 拒绝启动。
- **`pair`**：从活 server 增发；否则在服务未禁用时起一次性快照服务。

即使两个模型同机并存（如桌面旁边再跑 LAN `serve`），变更命令依然串行：每条
任务在 vfox 进程运行时持有跨进程锁（设置旁 `tasks.lock`，`core.ProcessLock`），
进程内队列之外再加一道。

配对服务（`mygo/pair` 包）是触达无显示器机器的方式：一次性配对链接，会话可过期、
链接可增发；发布 Docker 镜像
（[`mygo/Dockerfile`](../../../mygo/Dockerfile)）无需显示直接跑它。
非回环服务走 HTTPS：不传 `--tls-cert/--tls-key` 时首次启动在设置目录旁生成
自签名证书，在浏览器核对打印的指纹，会话 cookie 为 Secure。配对后的浏览器在
`/app` 得到完整 Web 界面：已安装 SDK 的安装/使用/卸载、插件仓库的添加/更新/
删除、带实时进度与日志的任务、以及设置。Web 界面使用的 HTTP 接口见
[远端协议](remote-protocol.md)。

Android 客户端（`android/`）连同一个服务：它是包住 `/app` 的 WebView，
配对流程是原生 Material 3 Expressive（粘贴配对链接、核对证书指纹），
手机以此驱动 PC 上的 vfox-plus 实例。Web 界面本身遵循
[设计系统](design-system.md)（种子色、形状、动效），资源全部内联（favicon 也是），
因为配对服务只提供单个无外部请求的嵌入页面。`/manifest.webmanifest` 的 Web 应用清单
让 Chromium 系浏览器能把该页面安装为应用，图标相同。
