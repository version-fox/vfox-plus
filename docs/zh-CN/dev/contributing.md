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
    ```

    只有 Docker、又不想动本机 vfox 安装时，用沙箱跑全部检查、端到端测试和打包，
    `vfxtool sandbox all`（见
    [sandbox](sandbox.md)）。

4. 在 Pull Request 中说明改了什么、怎么测试的。界面变化附浅色和深色主题截图。

## 持续集成

| 工作流 | 触发 | 检查 |
| --- | --- | --- |
| [`repository.yml`](../../../.github/workflows/repository.yml) | 每次变更 | 许可证头（SkyWalking Eyes）、工作流（actionlint）、vfxtool（`gofmt`、`go vet`、`go test`）、文档语言与链接（`vfxtool docs-check`）、Markdown（markdownlint）、共享 JSON、无二进制文件 |
| [`memory.yml`](../../../.github/workflows/memory.yml) | 构建工作流结束、Pull Request | 在 Pull Request 下评论应用内存及与 `main` 的差异（见[内存评论](memory.md#pull-request-中的内存)） |
| [`mygo.yml`](../../../.github/workflows/mygo.yml) | `mygo/`、`shared/` | 格式化与 lint（golangci-lint，含端到端构建标签）、带竞态检测的单元测试与覆盖率门禁、Go 模块许可证、驱动构建好的应用的端到端场景（夹具与真实 vfox）、构建并对着应用运行的网关 Docker 镜像、内存测量、发布构建与冒烟测试、`.deb` 和 `.tar.gz`（lintian 校验并装入干净 Ubuntu 容器），以及 Windows 11 未签名 `.msix` 和 `.msixbundle`（x64 与 arm64，Linux 上 `vfxtool validate-msix` 校验，Windows Server 2025（x64）与 ARM 版 Windows 11（arm64）上校验并安装/卸载） |
| [`sandboxes.yml`](../../../.github/workflows/sandboxes.yml) | `mygo/sandbox/`、`mygo/versions.env`、`shared/sandbox/`、`shared/tooling/` | 构建 Docker 沙箱并在其中跑 `vfxtool sandbox all` |
| [`android.yml`](../../../.github/workflows/android.yml) | `android/`、`mygo/pair/`、`mygo/VERSION` | 构建 release APK（debug 密钥签名）并作为 `vfox-plus-android` 上传 |
| [`devcontainer.yml`](../../../.github/workflows/devcontainer.yml) | `.devcontainer/`、`mygo/versions.env`、`android/versions.env` | 构建[开发容器](environment.md#开发容器)，像编辑器那样启动并使用它：对照固定版本的工具、lint、虚拟显示里的端到端场景、Android 构建，以及用到 Docker 的检查 |
| [`release.yml`](../../../.github/workflows/release.yml) | 手动或 `v*` 标签 | 打 `v<版本>` 标签，打包并发布 GitHub release，自更新检查读取的就是它 |

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

与 [version-fox/vfox-java](https://github.com/version-fox/vfox-java) 的发布一样，
发布手动启动：

1. 先在 Pull Request 里定版本：`mygo/VERSION` 与 `mygo/ui/app.go` 的 `Version`
   常量（有单元测试保证两者一致）。
2. Actions、Release、Run workflow，输入不带 `v` 的版本号（`0.2.0`）。推送
   `v*` 标签跑同一个任务。
3. 任务只接受 `major.minor.patch`、只接受 `mygo/VERSION` 写明的版本，已存在的标签会被拒绝。任务打
   `v<版本>` 标签，按[持续集成](#持续集成)打包外加 `snap/snapcraft.yaml` 的 classic
   snap，以 `--generate-notes` 发布 GitHub release，其附件就是[自更新检查](architecture.md#自身更新)读取的那些。

## 文档语言

全部文档放在 `docs/` 下，先按语言、再按受众分类，英文为准、简体中文译文同一
Pull Request 提交。中文逐节对照，代码块、命令行、文件路径及
[i18n.md](i18n.md) 术语表保持不变。

| 英文（canonical） | 简体中文 |
| --- | --- |
| `docs/en/user/*.md` | `docs/zh-CN/user/*.md`（同名） |
| `docs/en/dev/*.md` | `docs/zh-CN/dev/*.md`（同名） |

用户指南（`user/`）给运行 vfox-plus 的人看：安装、使用桌面应用、浏览器与
Android 客户端，绝不涉及源码构建、仓库布局、vfox CLI 契约等开发者话题。
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

Android 应用不用这个包里的图标：它的构建导入 google/material-design-icons 发布的 Android XML
（做法见[图标](design-system.md#图标)）。要添加一个，在 `android/app/build.gradle.kts` 的 `materialSymbols` 列表里加一行：
drawable 的名字、符号，要填充变体就再写 `"fill1"`。要换到该仓库更新的提交，改那里的 `materialSymbolsCommit`。
不用复制，也不用提交任何文件。

要换新版本的包，改该文件 `source` 里的 `version` 与 `sha512`。注册表给的校验和是 base64，文件里要十六进制：

```sh
version=1.2.98
curl -fsS "https://registry.npmjs.org/@iconify-json%2fmaterial-symbols/$version" |
  grep -o '"integrity":"sha512-[^"]*"' | cut -d- -f2 | tr -d '"' | base64 -d | od -An -v -tx1 | tr -d ' \n'
```

### Web 界面

Web 界面是 TypeScript 加 Vite，源码在 [`mygo/web`](../../../mygo/web)。
构建把它打成 `mygo/pair/webapp.html` 这单个文件供 Go 服务嵌入：改源码，在那里跑
`npm run build`，页面与源码一起提交。`npm test` 跑页面的单元测试；`npm run dev`
对着运行中的应用以热重载供页面（见[Node 与 Web 界面](environment.md#node-与-web-界面)）。
`mygo/pair` 的 Go 测试继续守构建产物的契约：设计 token 与弹簧（主题动了它们会失败），
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
| `VFOX_PLUS_E2E_WINDOW=412x800` | 窄布局（窗口窄于 600 dp）：`compact-home`、`compact-detail` |
| 上面两个一起 | 窄布局下 Java 的发行版：`compact-java` |
| 设置 `webInterface`，以及 `VFOX_PLUS_E2E_ADDRESSES=192.168.1.20` | 网页界面的设置，像用户操作那样一步步走：已开启的开关与它的地址、允许其他设备连接前的询问、随之出现的设置，以及给手机配对的二维码：`web-interface`、`web-allow`、`web-remote`、`web-pair` |
| 同上，先选好语言 | 同样四张的简体中文版：`zh-web-interface`、`zh-web-allow`、`zh-web-remote`、`zh-web-pair` |

端到端构建可以用 `VFOX_PLUS_E2E_ADDRESSES`（逗号分隔的列表）被告知这台电脑的地址，而不是从网卡上读。图里
显示的是 `192.168.1.20`：家用网络常见的那种地址，每次运行都一样，也不属于任何一台机器；没有网络、因而没有
自己地址的沙箱，也就有了一个可以给手机配对的地址。

要多拍一张，就在 `shared/tooling/internal/screenshots/screenshots.go` 里该运行的函数
（`pages`、`installing`、`profiles`、`compactPages`、`compactProfiles` 或 `webInterface`）加一句
`c.shot(s, "名称")`。它之前的步骤像用屏幕阅读器的人一样点击、等待，靠用户看到的名字
（按钮的标签，或组件给它的 key）找控件，所以改了名的按钮会让场景失败，而不是拍出一张错的图。

README 与[用户指南](../user/index.md)引用 `home`、`sdks`、`dark-available`、`installing`、
`tasks`、`plugins`、`zh-settings`、`compact-home`、`compact-detail`、`java-choose`、
`java-temurin`、`flutter-available` 与 `options`，以及手机上的网页界面 `webapp-en` 与
`web-phone-java-temurin`（简体中文是 `webapp-zh` 与 `web-phone-zh-java-temurin`，见下）。
[使用说明](../user/using.md)引用 `java-install-name` 与 `options-mirror`，中文版引用
`zh-java-available` 与 `zh-options`。[网页界面指南](../user/remote.md)引用 `web-interface`、`web-allow`、
`web-remote` 与 `web-pair`（简体中文加前缀 `zh-`），以及手机上的网页界面：`webapp-en`、
`web-phone-java-temurin`、`web-connections` 与 `web-connections-pair`（简体中文是 `webapp-zh`、
`web-phone-zh-java-temurin`、`web-zh-connections` 与 `web-zh-connections-pair`）。
[安卓 App 指南](../user/android.md)引用 App 的图，[网关指南](../user/gateway.md)引用网关的图（见下）。其余的图
是改界面时用来看的，没有文档引用它们。

**Web 界面。** 用应用的端到端构建（在虚拟显示里）与 headless Chrome 渲染，语言为截图固定。`vfxtool webheap`
可以拍 `--tab` 打开的页面（`home`、`sdks`、`plugins`、`tasks`、`settings`），或 `--scene` 演出的场景：
`connections`、`connections-pair`、`flutter-chips`、`java-available`、`java-choose`、`java-temurin`、
`java-install-name`、`options`、`options-mirror`、`options-aliases`、`alias-set` 与 `link-manual`（见 `shared/tooling/internal/webheap/scenes.go`，它的测试会对照
`mygo/pair/webapp.html` 检查场景点击的元素）。每张图都用自己的应用与 HOME，因为所选发行版保存在应用的
设置里，选了发行版的场景会改变它之后的每一张图。应用因为设置而提供它的网页界面，并允许其他设备（连接卡片与给
手机配对的二维码需要它：`{"webInterface":true,"webRemote":true}`）；它被告知文档用的地址（见上）；端到端构建没有
命令行可打印链接，所以把配对链接写进 `VFOX_PLUS_E2E_LINK` 指定的文件。Java 的场景需要一台装着 Java 和 Flutter 的
机器，主页需要一台没装的，所以假 vfox 的库存是调用时给的变量：

```sh
go -C shared/tooling build -o /tmp/vfxtool ./cmd/vfxtool
/tmp/vfxtool prepare --root mygo && /tmp/vfxtool build --e2e --root mygo /tmp/vfox-plus-e2e

# shoot 名称 语言 宽,高 [webheap 的其他选项]
# INVENTORY=profiles shoot ... 让假 vfox 成为装着 Java 与 Flutter 的机器。
shoot() {
  name=$1 lang=$2 window=$3; shift 3
  home=/tmp/shot-home
  rm -rf $home && mkdir -p $home/bin $home/run $home/.config/vfox-plus && chmod 700 $home/run
  cp /tmp/vfxtool $home/bin/vfox
  echo '{"webInterface":true,"webRemote":true}' > $home/.config/vfox-plus/settings.json
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
  /tmp/vfxtool webheap --server "${LINK%%/auth/connect/*}" \
    --pairing "$LINK" --output /tmp/shot.txt --screenshot "$name.png" \
    --lang "$lang" --window "$window" --idle 0 --chrome "$(which google-chrome)" "$@"
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

**Android。** CI 拍一张图 `screenshot-android`：`android.yml` 的 `measure` 任务在模拟器跑 release APK，
**Android。** CI 拍一张图 `screenshot-android`：`android.yml` 的 `measure` 任务在模拟器跑 release APK，
`vfxtool measure-apk` 截冷启动，任务把它上传为构件（`gh run download`）。[安卓 App 指南](../user/android.md)里的
九张图是在你自己的模拟器上拍的（[自己的模拟器](environment.md#自己的模拟器)：API 36，1080×2400，420 dpi），
用 CI 构建出来的 release APK（它的构件 `vfox-plus-android`），以及[运行 App](environment.md#运行-app)里描述的 PC 端应用：
对着 `VFOX_PLUS_FAKE_INVENTORY=profiles` 的假 vfox 的端到端构建，开启网页界面和其他设备，端口为 8080，并设
`VFOX_PLUS_E2E_ADDRESSES=10.0.2.2`——模拟器访问这台电脑用的地址。配对屏、链接和证书就都写着实际使用的地址，
也不必自己生成证书。这九张是 `android-pairing`、`android-scan`、`android-trust`、`android-home`、`android-sdk`、
`android-settings`、`android-dark-pairing`、`android-dark-home` 与 `android-unreachable`；指南全部引用，两种语言的指南
都用英文版。让图保持一致的做法以及容易出错的地方：

- 状态栏的演示模式固定时钟，显示满电与 Wi-Fi、没有通知：`adb shell settings put global sysui_demo_allowed 1`，
  再 `adb shell am broadcast -a com.android.systemui.demo -e command`，依次接 `enter`、`clock -e hhmm 1000`、
  `battery -e level 100 -e plugged false`、`network -e wifi show -e level 4 -e fully true`、
  `network -e mobile hide`（它的图标在浅色主题下是白的）与 `notifications -e visible false`。
- 相机打开时，Android 会在状态栏里显示一个绿色指示，顶掉 Wi-Fi 和电池图标，扫码界面的图上就有它。用 `adb root`，再
  `adb shell device_config put privacy camera_mic_icons_enabled false` 关掉（shell 用户不能写这个开关，root 可以）。
- 语言与主题是 App 与系统的：`adb shell cmd locale set-app-locales dev.vfox.vfoxplus --locales zh-CN` 与
  `adb shell cmd uimode night yes`。每轮之间清数据（`pm clear`），并重新授予相机权限（`pm clear` 会收回它），
  否则系统对话框会盖住扫码界面。
- App 里的控件用 `adb exec-out uiautomator dump /dev/tty` 里的 id 找（`scanButton`、`manualButton`、`linkField`、
  `connectButton`、`reconnectButton`）。WebView 里的网页界面不在导出里：底部的栏按位置点，
  `adb shell input tap X 2205`，X 依次取 125、332、540、749、956（主页、SDK、插件、任务、设置），
  flutter 是插件列表里的第一个，在 `540 682`。
- 配对链接要在 App 为英文时输入（`adb shell input text`）。App 为中文时 Gboard 会转成拼音，把链接写成全角标点，还把 `auth` 联想成
  `author`；在点*连接*之前把语言改回来，输入框的文字会保留。键盘会盖住按钮：输入后先按返回键
  （`adb shell input keyevent 4`）收起它。
- 指纹对话框与设置要在应用的同一次运行里拍：应用每次运行都重新生成证书，两处的指纹只在同一次运行里一致。选*信任一次*，
  拍主页之前要等页面加载完。
- 想拍“连不上 PC”的页面，就结束 PC 端的应用，再打开 App：它先打开带“上次的服务器”的配对界面，选*重新连接*就出现那个页面。
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
