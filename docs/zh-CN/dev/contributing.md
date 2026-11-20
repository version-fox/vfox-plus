# 参与 vfox-plus 贡献

> [English](../../en/dev/contributing.md)

感谢你的帮助！本仓库存放 vfox-plus 应用程序（见
[用户指南](../user/index.md)）；下文大多适用于它的各个部分。

## 基本规则

- 提交只署编写与评审的人。不要为 AI agent 或工具添加 `Co-authored-by`
  trailer；仓库工作流会拒绝它们。
- 代码注释、提交信息和 Pull Request 使用**英文**。用户可见文本走
  [`shared/i18n`](../../../shared/i18n) 中的翻译目录，提供英文和简体中文。
- 文档使用**英文并附简体中文译文**：每个 Markdown 文档都有中文对照版本（见
  [文档语言](#文档语言)）。
- 每个源文件开头都带有本仓库使用的 Apache-2.0 许可证头。用
  `docker run --rm -v "$PWD:/github/workspace" apache/skywalking-eyes:0.9.0 header check`
  检查，用 `header fix` 补齐。
- 运行时依赖必须采用与 Apache License 2.0 兼容的许可证。
- 各层共享的行为——vfox CLI 契约、界面、设计标记、翻译、图标、设置——在
  本 `dev/` 目录和 [`shared/`](../../../shared/) 中规定。改代码时同步改规约。
- 文件使用 UTF-8、LF 换行并以换行结尾（见
  [`.editorconfig`](../../../.editorconfig) 和 [`.gitattributes`](../../../.gitattributes)）。
- 不提交二进制文件。截图等图片上传到图床后引用（见
  [截图](#截图)）；Windows 包的徽标等二进制构建输入在构建时生成。
  CI 会拒绝二进制文件。

## 工作流

配置机器（Ubuntu 26.04，单独使用或在 WSL2 中），或使用开发容器，见[开发环境](environment.md)。

1. 大改动先开 issue 或在已有 issue 下留言。
2. 从 `main` 建分支，Pull Request 保持聚焦。
3. 跑检查；与 CI 相同，列表见
  [mygo 实现](implementation-mygo.md#日常命令)：

    ```sh
    cd mygo
    golangci-lint fmt --diff
    golangci-lint run && golangci-lint run --build-tags e2e
    CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./... && vfxtool coverage coverage.out 85
    go test ./ui -run TestTheScreensAreWithinTheBudgets
    ```

    只有 Docker、又不想动本机 vfox 安装时，用沙箱跑全部检查、端到端测试和打包，
    `vfxtool sandbox all`（见
    [sandbox](sandbox.md)）。

4. 在 Pull Request 中说明改了什么、怎么测试的。界面变化附浅色和深色主题截图。

## 手机端应用

手机端应用是 [`mygo/phone`](../../../mygo/phone)，由 mygo CLI 为安卓构建（见[手机端打包](packaging-android.md)）。在装好了 [`mygo/phone/versions.env`](../../../mygo/phone/versions.env) 所钉住工具的机器上，或在已装好它们的[开发容器](environment.md#开发容器)里，于仓库根目录构建：

```sh
vfxtool appicon mygo/phone/resources
cd mygo/phone && mygo build -platform android/amd64
```

`vfxtool appicon` 写出图标（被 git 忽略）。APK 与 AAB 在 `mygo/phone/dist/android/` 里，用本机的 debug 密钥签名。要在 Linux 上不用模拟器看屏幕，于 `mygo/` 运行 `go run ./phone`，应用会在手机大小的窗口里打开。要在模拟器上与桌面应用配对，见[运行 App](environment.md#运行-app)。

## 持续集成

| 工作流 | 触发 | 检查 |
| --- | --- | --- |
| [`repository.yml`](../../../.github/workflows/repository.yml) | 每次变更 | 许可证头（SkyWalking Eyes）、工作流（actionlint）、vfxtool（`gofmt`、`go vet`、`go test`）、文档语言与链接（`vfxtool docs-check`）、Markdown（markdownlint）、共享 JSON、无二进制文件 |
| [`memory.yml`](../../../.github/workflows/memory.yml) | `mygo` 或 `phone` 工作流结束、Pull Request | 在 Pull Request 下评论应用内存及与 `main` 的差异（见[内存评论](memory.md#pull-request-中的内存)） |
| [`mygo.yml`](../../../.github/workflows/mygo.yml) | `mygo/`、`shared/` | 格式化与 lint（golangci-lint，含端到端构建标签）、带竞态检测的单元测试与覆盖率门禁、Go 模块许可证、驱动构建好的应用的端到端场景（夹具与真实 vfox）、构建并对着应用运行的网关 Docker 镜像、内存测量、在 Linux、Windows（Server 2025 与 2022，以及 ARM 版 11）和 macOS（26 与 15，以及 Intel 上的 15）上取桌面应用各屏幕的普查、发布构建与冒烟测试、amd64 的 `.deb` 与 `.tar.gz`（lintian 校验并装入干净 Ubuntu 容器），由这个 `.deb` 打包的 core26 classic snap（装好并启动），arm64 的 `.deb` 与 `.tar.gz`（lintian 校验，并核对程序的机器类型；没有 arm64 机器装它们），以及 Windows 11 未签名 `.msix` 和 `.msixbundle`（x64 与 arm64，Linux 上 `vfxtool validate-msix` 校验，Windows Server 2025（x64）与 ARM 版 Windows 11（arm64）上校验并安装/卸载） |
| [`sandboxes.yml`](../../../.github/workflows/sandboxes.yml) | `mygo/sandbox/`、`mygo/versions.env`、`shared/sandbox/`、`shared/tooling/` | 构建 Docker 沙箱并在其中分别跑各部分（`vfxtool sandbox check`、`e2e`、`package` 与普查的四组），每部分一个任务 |
| [`phone.yml`](../../../.github/workflows/phone.yml) | `mygo/phone/`、`mygo/pair/`、`mygo/ui/kit/`、`mygo/VERSION`、`mygo/go.mod`、`mygo/go.sum`、`mygo/versions.env`、`shared/tooling/`、`shared/icons.json` | 为 x86_64 模拟器构建[手机端应用](#手机端应用)的 APK 与 AAB（用 debug 密钥签名），并在模拟器上驱动它：经深链与桌面应用配对，逐个访问五个标签页，冷启动与配对后各测一次（见[手机端应用](memory.md#手机端应用)） |
| [`devcontainer.yml`](../../../.github/workflows/devcontainer.yml) | `.devcontainer/`、`mygo/versions.env`、`mygo/phone/versions.env` | 构建[开发容器](environment.md#开发容器)，像编辑器那样启动并使用它：对照固定版本的工具、lint、虚拟显示里的端到端场景、手机端应用的构建，以及用到 Docker 的检查 |
| [`release.yml`](../../../.github/workflows/release.yml) | 在 `main` 上手动，或 `v*` 标签 | 设定版本并提交到 `main`，打 `v<版本>` 标签，打包（含 Linux arm64 与手机端 APK）并发布 GitHub release，自更新检查读取的就是它 |

Dependabot 负责 Go 模块和 GitHub Actions 的更新；action 按提交 SHA 锁定。

工作流用
[`install-linux-packages`](../../../.github/actions/install-linux-packages/action.yml) 安装
Ubuntu 包：下载的压缩包按包列表和周做缓存，每次 8 个文件并行下载，断点续传，
慢镜像自动切换，15 分钟内完成。部分 runners 的 Ubuntu 镜像有时很慢：
2026-10-23 某次连接只有 40 KB/s，而别的连接几秒下载 90 MB。apt 不校验已在目录中的
压缩包，所以 action 按哈希逐个校验。失败的运行会保存已下载部分，下次继续。

重建应用的 Pull Request 会收到一条内存评论及与 `main` 的差异，新提交会替换它
而不是追加。超过 CI 噪声的变化会加粗；小数字先读
[Pull Request 中的内存](memory.md#pull-request-中的内存)再下结论。

## 发布

一次发布就是手动运行一次 Release 工作流：

1. Actions、Release、Run workflow。分支保持为 `main`，任务会拒绝其他分支。输入不带 `v` 的版本号，例如 `1.0.0`。
2. 任务在改动任何东西之前先检查请求：版本须是 `major.minor.patch` 的形式，比最新的标签新，且还没有对应的标签。`main` 上最近一次 `mygo` 运行必须已通过：发布只做 `go vet` 和冒烟测试，不跑测试。
3. 构建任务把版本写入 `mygo/VERSION`、`mygo/ui/app.go` 的 `Version` 常量、`mygo/phone/mygo.json` 的 `version`，以及 [metainfo](../../../shared/packaging/linux/dev.vfox.VfoxPlus.metainfo.xml) 的 `<releases>` 列表，并以发起运行的人的名义提交。它构建 [`mygo.yml`](#持续集成) 里那些打包的产物（amd64 与 arm64），Windows 的 `.msix` 包以其证书的发布者构建，手机端的 APK 用 debug 密钥构建（发布时由下一步替换），外加 `snap/snapcraft.yaml` 的 classic snap。它不推送任何东西。
4. `sign-windows` 任务等待批准，然后用 Artifact Signing 给 Windows 包签名，并安装 bundle 来检查（见[发布的签名](packaging-windows.md#发布的签名)），再用发布密钥签名 APK（见[签名状态](packaging-android.md#签名状态)）。
5. 发布任务给 Linux 的包做构建证明，并检查发布的文件。只有这之后，它才把提交推到 `main`、打 `v<版本>` 标签，并以 `--generate-notes` 发布 GitHub release。release 含 Linux 的 `.deb` 与 `.tar.gz`（amd64 与 arm64）、APK、snap（amd64），以及每个架构已签名的 `.msix` 与已签名的 `.msixbundle`。自更新检查只读当前平台的 `.deb` 或 `.tar.gz`，以及 `.msix` 或 `.msixbundle`；其余供手动使用。

APK 用发布密钥签名，每个 release 都是同一把（见[签名状态](packaging-android.md#签名状态)），所以新版本直接装在旧版本上；任务没有这把密钥就拒绝运行。发布的 secret 与 Windows 签名的身份都在 `release` 环境里。运行要在 `sign-windows` 签名 Windows 包与 APK 之前，等 `@version-fox/vfox-maintainers` 或 `@version-fox/vfox-committers` 的成员批准一次；发布紧接着进行，不再等第二次。仓库管理员发起的运行可以跳过审批。

勾选 `dry-run` 则构建、签名并安装全部东西，但不推送、不打标签、不发布。签名或 secret 有改动时，先这样运行一次。在那之前，APK 的签名每次有改动都已由 [`phone.yml`](../../../.github/workflows/phone.yml) 的 `signing` 任务用发布的脚本和任务自己生成的密钥演练过：dry run 多出来的，是只有发布才有的部分，即它的 secret、Azure 的步骤与批准（见[签名状态](packaging-android.md#签名状态)）。

版本提交不会在 `main` 上启动 CI 运行：工作流自己的令牌推送的提交，GitHub 不会启动任何工作流。所以那个提交上不跑测试：它之前最近一次 `mygo` 运行才算数，这就是任务要检查它的原因。

推送 `v*` 标签会跑同样的几个任务，但不设置版本：标签指向的提交必须已经写明该版本，否则任务拒绝。

任务在推送之前失败，GitHub 上什么都没有改变：修好原因后重新运行。若在推送之后失败，版本提交可能已经在 `main` 上；下一次运行会发现文件已写明版本，不会再提交一次。删除失败运行留下的东西：标签（`git push origin --delete v<版本>`），若已发布了 release 再删除它（`gh release delete v<版本>`），然后重新运行。

## 文档语言

全部文档放在 `docs/` 下，先按语言、再按受众分类，英文为准、简体中文译文同一
Pull Request 提交。中文逐节对照，代码块、命令行、文件路径及
[i18n.md](i18n.md) 术语表保持不变。

| 英文（canonical） | 简体中文 |
| --- | --- |
| `docs/en/user/*.md` | `docs/zh-CN/user/*.md`（同名） |
| `docs/en/dev/*.md` | `docs/zh-CN/dev/*.md`（同名） |

用户指南（`user/`）给运行 vfox-plus 的人看：安装、使用桌面应用、浏览器与
手机端应用，绝不涉及源码构建、仓库布局、vfox CLI 契约等开发者话题。
开发者规约（`dev/`）给贡献者看：架构、契约、界面与设计规约、打包与工具链。

仓库根与源码目录只保留薄入口
（`README.md`、`mygo/README.md`、`shared/*/README.md`），由它们链入 `docs/`，
实质内容都在 `docs/` 里。

每个文档顶部互链（英文用 `[简体中文](…)`，中文用 `[English](…)`），
`vfxtool docs-check` 校验映射；仓库工作流在缺对照时失败。新增文档同样成对添加。

## Markdown

CI 用 [markdownlint](https://github.com/DavidAnson/markdownlint) 检查每个文档，配置在
[`.markdownlint.yaml`](../../../.markdownlint.yaml)。推送前用 CI 的同一个镜像（按 digest 固定）检查：

```sh
docker run --rm --volume "$PWD:/workdir" \
  ghcr.io/igorshubovych/markdownlint-cli:v0.50.0@sha256:db7dcb73cd3163cc45854b60fc21eefe1ebe7fa52aad758706010b64a4778900 \
  --ignore-path .gitignore "**/*.md"
```

最要紧的两个习惯：

- **URL 写在尖括号里**，`<https://vfox.dev/>`，或作链接目标 `[vfox](https://vfox.dev/)`。
  GitHub 把裸 URL 一直识别到下一个空白，所以它后面的 `)`，以及中文里的全角 `）`、`，`、`。`，
  都会算进地址，预览里就是错误的 URL。
- **每个链接都用 `]` 闭合**：`[文字](目标)`。`[文字(目标)` 不是链接，会原样显示；
  `vfxtool docs-check` 对它报错。

表格单元格里的 `|` 哪怕在代码里也会结束单元格，要写成 `\|`。

## 常见改动

### 文本

在 [`shared/i18n/app_en.arb`](../../../shared/i18n/app_en.arb) 中添加或修改消息
（含 `description` 与占位符元数据），再译到
[`app_zh.arb`](../../../shared/i18n/app_zh.arb)。见
[i18n.md](i18n.md) 及术语表。

### 图标

在 [`shared/icons.json`](../../../shared/icons.json) 中按含义添加 Material Symbol，符号名同
<https://fonts.google.com/icons>（`history`、`folder_open`），其余不用加：构建时 `vfxtool prepare`
从锁定的 npm 包切出 SVG；包里没有的符号会失败并列出这些图标。

手机端应用用同一个包里的符号，经同样内嵌的图标使用它们，所以不需要为它另外添加任何东西。它的启动器图标另有来源：
由 `vfxtool appicon mygo/phone/resources` 写出 `icon.png` 与 `icon-foreground.png`，由 [`mygo/phone/mygo.json`](../../../mygo/phone/mygo.json) 指定（见[应用图标](design-system.md#应用图标)）。

要换新版本的包，改该文件 `source` 里的 `version` 与 `sha512`。注册表给的校验和是 base64，文件里要十六进制：

```sh
version=1.2.98
curl -fsS "https://registry.npmjs.org/@iconify-json%2fmaterial-symbols/$version" |
  grep -o '"integrity":"sha512-[^"]*"' | cut -d- -f2 | tr -d '"' | base64 -d | od -An -v -tx1 | tr -d ' \n'
```

### Web 界面

Web 界面是 TypeScript 加 Vite，源码在 [`mygo/web`](../../../mygo/web)。
构建把它打成 `mygo/gateway/webapp.html` 这单个文件供网关嵌入：改源码，在那里跑
`npm run build`，页面与源码一起提交。`npm test` 跑页面的单元测试；`npm run dev`
对着运行中的应用以热重载供页面（见[Node 与 Web 界面](environment.md#node-与-web-界面)）。
`mygo/gateway` 的 Go 测试继续守构建产物的契约：设计 token 与弹簧（主题动了它们会失败），
以及 favicon 占位符。页面的文案放在它自己的 i18n catalog（`mygo/web/src/i18n-*`），按语言做类型检查，
不在 `shared/i18n` 里。

### vfox 输出

vfox 变化时用 `vfxtool capture` 重抓夹具，按
[`fixtures` 夹具](fixtures.md) 更新，并在
[vfox-cli-contract.md](vfox-cli-contract.md) 中记录。

### 图表

图表用 [Mermaid](https://mermaid.js.org/) 画在 `mermaid` 围栏代码块里，GitHub 会渲染
它；不要用字符画，也不要提交图片。[架构](architecture.md)、
[夹具](fixtures.md)与[远端协议](remote-protocol.md#任务)里的图定下了风格：复制其中
一个的 front matter（带主题），这样每张图在 GitHub 的浅色与深色主题下都一致；再给
图写 `accTitle` 与 `accDescr`，供读屏软件使用。

颜色代表角色：靛蓝（`#E4E0FF`）界面与等待，橙色（`#FFE9D0`）状态、资源与运行中，
青绿（`#D9F2EA`）核心与成功，红色（`#FFDAD6`）失败，灰色（`#ECEBF3`）自动化与已取消，
墨色（`#17123F`）应用之外的东西。长标签自己用 `<br/>` 断行（`wrappingWidth` 只限定
最宽的一行），占位符的尖括号写成 `#lt;版本#gt;`，要让子图里的节点排成一行，就用
`~~~` 把它们串起来。提交前用 Mermaid 11 和 12（[在线编辑器](https://mermaid.live)
或 `mmdc`）看一眼结果：GitHub 遇到解析不了的图，显示的是错误而不是图。

### 截图

文档图片不提交：托管在 [freeimage.host](https://freeimage.host)（免账号免费图床）后
引用。

**桌面应用。** `vfxtool sandbox screenshots` 可复现地把图渲染到
`dist/sandbox/mygo/screenshots/`：Metacity 加 Yaru 主题画出 Ubuntu 26.04 GNOME
X11 会话的标题栏，
Ubuntu 默认壁纸而非裸 X 的黑色。它对着假 vfox 把端到端构建启动七遍，经 mygo 的自动化插件
把它带到每个页面，页面沉降后用 ImageMagick 截取屏幕：

| 运行 | 拍什么 |
| --- | --- |
| 普通 | 每个页面 × 双主题 × 双语言：`home`、`sdks`、`available`、`tasks`、`plugins`、`dialog`、`settings`、`dark-expanded`、`dark-sdks`、`dark-available`、`zh-home`、`zh-settings` |
| `VFOX_PLUS_FAKE_DELAY=4` | 安装中：`installing` |
| `VFOX_PLUS_FAKE_INVENTORY=profiles` | 一台同时装着 Java、Flutter 与 Node.js 的机器，以及 vfox-plus 为这些插件提供的功能：`java-installed`、`java-available`、`java-choose`、`java-temurin`、`java-install-name`、`java-all`、`flutter-available`、`options`、`options-mirror`、`options-aliases`、`alias-set`、`link-manual`、`zh-options`、`zh-java-available` |
| `VFOX_PLUS_FAKE_INVENTORY=python` | 一台装着 Python 与其他插件的机器，以及 Python 的选项与 GraalPy 构建的筛选：`python-options`、`python-install-method` 与 `python-graalpy-community`，深色主题下为 `dark-python-options` |
| `VFOX_PLUS_E2E_WINDOW=412x800` | 窄布局（窗口窄于 600 dp）：`compact-home`、`compact-detail` |
| 上面两个一起 | 窄布局下 Java 的发行版：`compact-java` |
| `VFOX_PLUS_E2E_ADDRESSES=192.168.1.20` | 其他设备的设置，像用户操作那样一步步走：尚未允许时的分区、允许其他设备连接前的询问、随之出现的设置，以及给手机配对的二维码：`web-interface`、`web-allow`、`web-remote`、`web-pair` |
| 同上，先选好语言 | 同样四张的简体中文版：`zh-web-interface`、`zh-web-allow`、`zh-web-remote`、`zh-web-pair` |

端到端构建可以用 `VFOX_PLUS_E2E_ADDRESSES`（逗号分隔的列表）被告知这台电脑的地址，而不是从网卡上读。图里
显示的是 `192.168.1.20`：家用网络常见的那种地址，每次运行都一样，也不属于任何一台机器；没有网络、因而没有
自己地址的沙箱，也就有了一个可以给手机配对的地址。

要多拍一张，就在 `shared/tooling/internal/screenshots/screenshots.go` 里该运行的函数
（`pages`、`installing`、`profiles`、`compactPages` 或 `compactProfiles`）加一句
`c.shot(s, "名称")`。它之前的步骤像用屏幕阅读器的人一样点击、等待，靠用户看到的名字
（按钮的标签，或组件给它的 key）找控件，所以改了名的按钮会让场景失败，而不是拍出一张错的图。

README 与[用户指南](../user/index.md)引用 `home`、`sdks`、`dark-available`、`installing`、
`tasks`、`plugins`、`zh-settings`、`compact-home`、`compact-detail`、`java-choose`、
`java-temurin`、`flutter-available` 与 `options`，以及手机上的网页界面 `webapp-en` 与
`web-phone-java-temurin`（简体中文是 `webapp-zh` 与 `web-phone-zh-java-temurin`，见下）。
[使用说明](../user/using.md)引用 `java-install-name` 与 `options-mirror`，中文版引用
`zh-java-available` 与 `zh-options`。[其他设备指南](../user/remote.md)引用 `web-interface`、`web-allow`、
`web-remote` 与 `web-pair`（简体中文加前缀 `zh-`），以及手机上的网页界面：`webapp-en`、
`web-phone-java-temurin`、`web-connections` 与 `web-connections-pair`（简体中文是 `webapp-zh`、
`web-phone-zh-java-temurin`、`web-zh-connections` 与 `web-zh-connections-pair`）。
[安卓 App 指南](../user/android.md)引用 App 的图，[网关指南](../user/gateway.md)引用网关的图（见下）。其余的图
是改界面时用来看的，没有文档引用它们。

**Web 界面。** 用应用的端到端构建（在虚拟显示里）与 headless Chrome 渲染，语言为截图固定。`vfxtool webheap`
可以拍 `--tab` 打开的页面（`home`、`sdks`、`plugins`、`tasks`、`settings`），或 `--scene` 演出的场景：
`home`、`connections`、`connections-pair`、`flutter-chips`、`java-available`、`java-choose`、`java-temurin`、
`java-install-name`、`options`、`options-mirror`、`options-aliases`、`options-legacy`、`settings-languages`、
`alias-set`、`link-manual`、`python-options` 与 `python-available`（见 `shared/tooling/internal/webheap/scenes.go`，它的测试会对照
`mygo/gateway/webapp.html` 检查场景点击的元素）。场景的最后一步会等页面不比窗口宽，所以溢出的布局会让截图失败，而不是
拍出一张图。每张图都用自己的应用与 HOME，因为所选发行版保存在应用的
设置里，选了发行版的场景会改变它之后的每一张图。应用允许其他设备（连接卡片与给
手机配对的二维码需要它：`{"webRemote":true}`）；它被告知文档用的地址（见上）；端到端构建没有
命令行可打印链接，所以把配对链接写进 `VFOX_PLUS_E2E_LINK` 指定的文件。页面是网关提供的，所以 `shoot`
也启动网关：它用这个链接，像人在网关页面上那样与应用配对（`--application`）。Java 的场景需要一台装着 Java 和 Flutter 的
机器，Python 的场景需要一台装着 Python 的机器，主页需要一台没装的，所以假 vfox 的库存是调用时给的变量：

```sh
go -C shared/tooling build -o /tmp/vfxtool ./cmd/vfxtool
/tmp/vfxtool prepare --root mygo && /tmp/vfxtool build --e2e --root mygo /tmp/vfox-plus-e2e
go -C mygo build -o /tmp/vfox-plus-gateway ./cmd/vfox-plus-gateway

# shoot 名称 语言 宽,高 [webheap 的其他选项]
# INVENTORY=profiles shoot ... 让假 vfox 成为装着 Java 与 Flutter 的机器。
shoot() {
  name=$1 lang=$2 window=$3; shift 3
  home=/tmp/shot-home
  rm -rf $home && mkdir -p $home/bin $home/run $home/.config/vfox-plus && chmod 700 $home/run
  cp /tmp/vfxtool $home/bin/vfox
  echo '{"webRemote":true}' > $home/.config/vfox-plus/settings.json
  (
    export HOME=$home XDG_CONFIG_HOME=$home/.config XDG_RUNTIME_DIR=$home/run PATH=$home/bin:$PATH NO_COLOR=1 \
      VFOX_PLUS_FIXTURES=$PWD/shared/fixtures/vfox-1.1.0 VFOX_PLUS_E2E_LINK=$home/link.txt \
      VFOX_PLUS_E2E_ADDRESSES=192.168.1.20
    [ -n "${INVENTORY:-}" ] && export VFOX_PLUS_FAKE_INVENTORY=$INVENTORY
    exec setsid xvfb-run -a dbus-run-session -- /tmp/vfox-plus-e2e >$home/app.log 2>&1
  ) &
  app=$!
  for i in $(seq 1 30); do
    LINK="$(grep -o 'http://127\.0\.0\.1:[0-9]*/auth/connect/[0-9a-f]*' $home/link.txt 2>/dev/null | head -1)"
    [ -n "$LINK" ] && break
    sleep 1
  done
  # 网关提供页面：它用自己的 HOME 运行，并打印把浏览器与它配对的链接。
  gw=/tmp/shot-gateway
  rm -rf $gw && mkdir -p $gw/config
  (export HOME=$gw XDG_CONFIG_HOME=$gw/config; exec /tmp/vfox-plus-gateway >$gw/log 2>&1) &
  gateway=$!
  for i in $(seq 1 30); do
    PAIR="$(grep -o 'https://localhost:8080/auth/connect/[A-Za-z0-9_-]*' $gw/log 2>/dev/null | head -1)"
    [ -n "$PAIR" ] && break
    sleep 1
  done
  /tmp/vfxtool webheap --server https://localhost:8080 --pairing "$PAIR" --application "$LINK" \
    --output /tmp/shot.txt --screenshot "$name.png" \
    --lang "$lang" --window "$window" --idle 0 --chrome "$(which google-chrome)" "$@"
  kill $gateway; wait $gateway 2>/dev/null
  kill -- -$app
}
shoot webapp-en en 412,915
shoot webapp-zh zh 412,915
INVENTORY=profiles shoot web-phone-java-temurin en 412,915 --scene java-temurin
INVENTORY=profiles shoot web-phone-zh-java-temurin zh 412,915 --scene java-temurin
shoot web-connections en 412,915 --scene connections
shoot web-connections-pair en 412,915 --scene connections-pair
shoot web-zh-connections zh 412,915 --scene connections
shoot web-zh-connections-pair zh 412,915 --scene connections-pair
INVENTORY=python shoot web-python-options en 412,915 --scene python-options
INVENTORY=python shoot web-python-available en 412,915 --scene python-available
INVENTORY=python shoot web-zh-python-options zh 412,915 --scene python-options
INVENTORY=python shoot web-zh-python-available zh 412,915 --scene python-available
```

（`PATH` 首位放一个 `vfox` 可执行文件——`vfxtool` 的复制品配夹具即假
vfox——SDK 列表才有内容，`INVENTORY=profiles` 让它成为装着 Java 与 Flutter 的机器；
`setsid` 让应用成为自己进程组的组长，`kill -- -$app` 把它连同虚拟显示一起结束；
`--lang` 固定界面语言；`--window 412,915` 拍手机竖屏，默认是 CI 测量的 1280×800 桌面窗口；
`--idle 0` 跳过堆诊断用的、在打开页面上停 20 秒的等待（堆数字只做本地诊断，不进 CI 报告）；
`--dark` 用深色主题拍；
`--chrome` 接受任何讲 CDP 的 Chrome，不传则工具在 `$CHROME_BIN` 和 `PATH`
上自动找。）上传前看一眼图的尺寸
（`file webapp-en.png`）：不是每个 Chrome 都保持这么小的窗口。Chrome for
Testing 156 把 `--window 412,915` 拍成了 500×715 的图，同版本的
`chrome-headless-shell` 恰好是 412×915，尺寸不对时用 `--chrome` 传它。

**网关。** `vfxtool gateway-shots` 拍它各页的图：它做 `vfxtool gateway-check` 做的事（见[实现指南](implementation-mygo.md)），
只是用无头 Chrome 代替人，并把 Chrome 每一步显示的画面留下来。它写出 `gateway-intro`（还没有配对的浏览器）、
`gateway-home`（浏览器已与网关配对，网关还没有配对任何东西）、`gateway-link`（粘贴了应用的链接）、
`gateway-trust`（要核对的指纹）、`gateway-app` 与 `gateway-zh-app`（通过网关看到的应用界面，英文与简体中文）
和 `gateway-paired`，每张的高度都恰好是那一页的高度。网关的页面本来就是中英文同时出现，所以一张图两份指南通用。
图里显示的是应用的地址，网关真的会连它，所以这台电脑得有这个地址：`unshare -rn` 在自己的网络命名空间里运行命令，
你在里面是 root，里面只有一个回环接口，地址就加到那个接口上，不碰你的网络。电脑没有这个地址时，命令在启动任何东西
之前就会停下并说明。

```sh
go -C mygo build -o /tmp/vfox-plus-gateway ./cmd/vfox-plus-gateway
unshare -rn sh -c 'ip link set lo up && ip addr add 192.168.1.20/32 dev lo &&
  xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- \
  /tmp/vfxtool gateway-shots --binary /tmp/vfox-plus-gateway --output /tmp/gateway-shots /tmp/vfox-plus-e2e'
```

（应用的构建就是上面那个；要在仓库根目录运行，命令靠它找假 vfox 的夹具；`--chrome` 和 `webheap` 一样指定 Chrome。）

**Android。** CI 拍一张图 `screenshot-android`，并留下配对后会话的证据。[`phone.yml`](../../../.github/workflows/phone.yml)
的 `emulator` 任务以 `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2` 启动桌面应用的端到端构建，运行 `vfxtool phone-link`（允许其他设备，
并为 `10.0.2.2` 写出配对链接），然后运行 `vfxtool measure-apk`。后者安装 APK，冷启动应用，再经深链 `vfoxplus://pair?link=…`
与它配对：点按*信任并配对*，逐个访问五个标签页，并为每个标签页保存一张截图与一份无障碍树的导出。配对成功后，应用在日志里写一行 `vfoxplus: paired host=… fingerprint=…`，`measure-apk` 等的就是这一行。首屏是构件
`screenshot-android`；标签页的截图与导出、应用的日志、桌面应用的日志，以及配对后的内存，是构件 `phone-e2e-android`。

[安卓 App 指南](../user/android.md)里的九张图是在你自己的模拟器上拍的（[自己的模拟器](environment.md#自己的模拟器)：API 36，
1080×2400，420 dpi），用 CI 构建出来的 APK（它的构件 `vfox-plus-android-amd64`），以及[运行 App](environment.md#运行-app)里描述的
PC 端应用：对着 `VFOX_PLUS_FAKE_INVENTORY=profiles` 的假 vfox 的端到端构建，设 `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2`——模拟器访问
这台电脑用的地址。配对链接是*显示二维码*给出的那个，地址为 `10.0.2.2`，并按[运行 App](environment.md#运行-app)那样作为深链打开。
这九张是 `android-pairing`、`android-scan`、`android-trust`、`android-home`、`android-sdk`、`android-settings`、
`android-dark-pairing`、`android-dark-home` 与 `android-unreachable`；指南全部引用，两种语言的指南都用英文版。让图保持一致的做法以及容易出错的地方：

- 状态栏的演示模式固定时钟，显示满电与 Wi-Fi、没有通知：`adb shell settings put global sysui_demo_allowed 1`，
  再 `adb shell am broadcast -a com.android.systemui.demo -e command`，依次接 `enter`、`clock -e hhmm 1000`、
  `battery -e level 100 -e plugged false`、`network -e wifi show -e level 4 -e fully true`、
  `network -e mobile hide`（它的图标在浅色主题下是白的）与 `notifications -e visible false`。
- 相机打开时，Android 会在状态栏里显示一个绿色指示，顶掉 Wi-Fi 和电池图标，扫码界面的图上就有它。用 `adb root`，再
  `adb shell device_config put privacy camera_mic_icons_enabled false` 关掉（shell 用户不能写这个开关，root 可以）。
- 语言与主题是 App 与系统的：`adb shell cmd locale set-app-locales dev.vfox.vfoxplus --locales zh-CN` 与
  `adb shell cmd uimode night yes`。每轮之间清数据（`pm clear`），并重新授予相机权限（`pm clear` 会收回它），
  否则系统对话框会盖住扫码界面。
- App 里的控件在无障碍树里：`adb exec-out uiautomator dump /dev/tty` 给出每个控件的文字与边界，
  `adb shell input tap X Y` 点按边界的中心。控件的文字是*信任并配对*（英文界面为 Trust and pair）、*连接*（Connect），
  以及五个标签页 *主页*、*SDK*、*插件*、*任务* 与 *设置*（英文界面为 Home、SDKs、Plugins、Tasks、Settings），
  `vfxtool measure-apk` 也按这些文字找。
- 配对链接要在 App 为英文时输入（`adb shell input text`）。App 为中文时 Gboard 会转成拼音，把链接写成全角标点，还把 `auth` 联想成
  `author`；在点*连接*之前把语言改回来，输入框的文字会保留。键盘会盖住按钮：输入后先按返回键
  （`adb shell input keyevent 4`）收起它。
- 显示指纹的对话框与设置要在应用的同一次运行里拍：应用每次运行都重新生成证书，两处的指纹只在同一次运行里一致。点按*信任并配对*，
  拍主页之前要等标签页加载完。
- 想拍“电脑没有响应”的页面，就在 App 打开时结束 PC 端的应用：标签页上方的条会显示它，并有*重试*按钮。
- 界面稳定后用 `adb exec-out screencap -p > NAME.png` 截图。扫码界面拍到的是模拟器的客厅
  （`hw.camera.back=virtualscene`），因为没有二维码可对。

图保持截下来的 1080×2400。`vfxtool frame-phone NAME.png OUT.png` 给它们加上手机屏幕的圆角（半径 96 px，圆角外透明）与 5 px
的灰色边框（`#7A7A84`）：浅色主题的米色屏幕在 GitHub 的白色页面上没有边缘。

**上传。** 上传新图：

```sh
curl -s -F key=6d207e02198a847aa98d0a2a901485a5 -F action=upload -F format=json \
  -F source=@dist/sandbox/mygo/screenshots/home.png https://freeimage.host/api/1/upload |
  jq -r '.image.url'
```

输出 URL（key 是 freeimage.host 在 [API 页](https://freeimage.host/api) 给所有人的公开
key），替换文档中的链接。curl 以 56 退出是连接断了：重试。每次上传都有新 URL，
旧图缓存不会串。匿名图无法删除；不可公开的东西不要传。

### 审查界面

`vfxtool webheap` 可以给它截图的那个页面做普查：`-screenshot` 旁边加 `-audit census.json`，会以 JSON
写出人读到什么、什么读不到：词数、8 个词以上的句子和标题下的说明小字、对比度低于 WCAG AA 的文字、
一行里有两个带文字按钮的列表行、三行以同一图标开头的列表、不在字号阶梯上的字号、小于 24 px 的控件，
被省略号截断的标题和控件文字、画到行内边距之外的控件和文字，以及页面是否比窗口宽
（表达式在 `shared/tooling/internal/webheap/audit.js`）。一个屏幕要满足的预算在
[设计系统](design-system.md#预算)里。

要审查整个界面，就按上文把每个页签（`-tab`）和每个场景（`-scene`）都拍一遍，浅色和深色（`-dark`）、
手机（`-window 412,915`）和宽窗口（`-window 1280,800`）各一遍，每张带自己的 `-audit` 文件，再把这些文件
放在一起看：总数显示字都在哪里，超预算的屏幕显示该看哪里。图也要看，并排放在一张图上：有的布局差，
差在还没有人量过的地方，普查里没有对应的数字。

桌面应用的普查在它的测试里（`mygo/ui/census_test.go`、`mygo/ui/kit/census.go`）。它不开窗口地画每个屏幕，
窗口就是[设计系统](design-system.md#预算)里写的那几种；从节点里读每段文字、每个图标和每个控件登记的位置，
文字怎么排的（字号、字重、颜色），以及单行文字想要的宽度；从窗口的图里读文字落在什么底色上。所以它能查到
网页普查查不到的几件事：图标表里没有的图标，以及一段文字画在另一段上面、或画出了它的控件之外，这些图上看得出、数字里说不出。
开着对话框或菜单的屏幕，就按对话框或菜单算。
比窗口还高的页面（设置、插件的选项、插件和 Java 发行版的列表：`censusWindow.heightFor`）画在和它一样高的窗口里，
折叠线以下的内容也算在内；页面仍然比窗口高时，普查就超预算（`Census.Taller`；带着一个比任何窗口能显示的行数都多的列表的页面，
Node.js 的版本，按它显示出来的行算）。其中六种窗口用系统文字缩放的 130 %、150 % 和 200 %，三种显示中文（场景用英文走到它的屏幕，
再在那里切换语言）。十四种窗口共 546 个屏幕。以对话框或菜单结尾的场景（`modal`）还要多画在三种窗口里，那是应用能打开的最矮窗口：场景先在够高的窗口里走到
它的屏幕，再把窗口缩矮（`censusWindow.shrunk`），场景要打开的菜单（`open`）则是从它滚动到了的按钮上打开的，就像人做的那样。会滚动的列表
（`toEnd`）还要在末尾再画一遍，比它矮的窗口里的对话框内容也滚到末尾、再滚回一个窗口的高度，各画一遍。一共 600 个屏幕、79 个末尾和 48 个中间位置，约两分钟：每个屏幕的每一帧都由软件渲染器绘制，开竞态检测要慢三倍，而普查并不需要它，
所以单元测试在竞态下跳过普查，CI 和沙箱单独用一步来跑。新增一个场景要多画十四次，新增一种窗口要多画二十五次，
加之前先掂量；也别指望再开线程，渲染器已经在用多个核了。这台电脑有哪些地址，由测试告诉网页界面
（`pair.HostOptions.Addresses`）：一个 IPv4 和一个 IPv6，都取自专门留作文档示例的地址段，所以配对的屏幕在每台机器上都一样。

```sh
cd mygo
go test ./ui -run TestTheScreensAreWithinTheBudgets
```

超预算时会失败，并列出超了哪几项，以及造成超标的文字、行或控件。`VFOX_PLUS_CENSUS=目录` 把每个屏幕的普查
写到该目录：每屏一个 JSON 文件，外加每屏一行的 `summary.txt`，并把超出预算的屏幕的图
（`census.<屏幕>.<窗口>.png`）也存进去，好看看数字说的是什么；`VFOX_PLUS_SNAPSHOTS=目录` 再把每个屏幕的图都写出来。
新增一个屏幕，就在 `censusScenes` 里加一项。

另有五个测试用别的窗口、别的文字取这些场景。`TestTheScreensAreWithinTheBudgetsInTheFontsThatTheSystemChooses` 用系统给 `sans-serif` 和
`monospace` 的字体画它们（日志里写着那是哪种字体），取三种窗口，约 20 秒；
`TestTheScreensAreWithinTheBudgetsInTheWidestFontsThatTheSystemHas` 用系统和桌面拿来排界面的字体族里最宽的三个
（Verdana、DejaVu Sans、Geneva、Inter 等，由文字引擎量出来）画它们，同样三种窗口，其中最宽的一个再用中文画一遍，约一分半钟；
`TestTheScreensAreWithinTheBudgetsInATextThatIsWider` 用字距（`kit.Tracking`）把所有文字变宽，每种字号都宽五分之二，
十种窗口，390 个屏幕，约一分半钟；
`TestTheScreensAreWithinTheBudgetsAtEverySizeOfText` 取普查没取的文字大小 115、130、150 和 175 %，四种宽度，英文和中文各一遍，
1248 个屏幕，约七分钟。这三个只在要求时才跑，用
`VFOX_PLUS_CENSUS_SWEEP=1 go test ./ui -run 'TestTheScreensAreWithinTheBudgets(AtEverySizeOfText|InTheWidestFonts|InATextThatIsWider)'`。
`TestTheScreensAreWithinTheBudgetsMeasureWhatTheyDraw` 很快，每次都跑：它向文字引擎问底栏标签的宽度，再把标签画进这个宽度的盒子里，
普查说一个标签被截断，靠的就是这一点。

CI 会要求它们。`mygo.yml` 的 `census` 在每台机器上是四个任务（Intel 上是五个）——屏幕（普查和系统字体；Intel 上的加宽文字另成一个任务）、最宽字体，以及字号扫描的两个分片（1280 dp 的窗口一片，其余的窗口一片）——同时运行，所以一次普查的时间是其中最长的那个，而不是四者相加。它在 Windows Server 2025 和 Windows Server 2022（x64）、ARM 版 Windows 11（arm64）、macOS 26 和 macOS 15（arm64）以及 Intel 上的 macOS 15 上取这些测试，每个平台用它自己的文字引擎和字体（Linux 是 Pango，这几个是 DirectWrite 和 Core Text）：GitHub 提供的最新和最旧的 Windows 与 macOS，以及各自的另一种架构。Linux 上装着桌面用来排文字的字体，与应用自己的字体并排；Linux 在运行器自己的字体里的普查是 `build` 任务的一步，与端到端场景排在一起。普查不等 `build`；每个平台上也跑 `kit` 的测试；打印它所在系统的版本；某一组一个屏幕也没跑时它失败；图则作为产物保存：一次运行失败时保存超出预算的屏幕的图，人工触发工作流（`workflow_dispatch`）时保存每个屏幕的图。沙箱也按同样的四组取普查，每组一个任务，用容器自己的字体。改了布局之后自己也要跑一遍，好在推送前就知道。
