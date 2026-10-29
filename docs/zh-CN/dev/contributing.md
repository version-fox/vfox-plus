# 参与 vfox-plus 贡献

> [English](../../en/dev/contributing.md)

感谢你的帮助！本仓库存放 vfox-plus 应用程序（见
[用户指南](../user/index.md)）；下文大多适用于它的各个部分。

## 基本规则

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

配置机器（Ubuntu 26.04，单独使用或在 WSL2 中）见[开发环境](environment.md)。

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
| [`mygo.yml`](../../../.github/workflows/mygo.yml) | `mygo/`、`shared/` | 格式化与 lint（golangci-lint，含端到端构建标签）、带竞态检测的单元测试与覆盖率门禁、Go 模块许可证、应用内端到端场景（夹具与真实 vfox）、内存测量、发布构建与冒烟测试、`.deb` 和 `.tar.gz`（lintian 校验并装入干净 Ubuntu 容器），以及 Windows 11 未签名 `.msix` 和 `.msixbundle`（x64 与 arm64，Linux 上 `vfxtool validate-msix` 校验，Windows Server 2025（x64）与 ARM 版 Windows 11（arm64）上校验并安装/卸载） |
| [`sandboxes.yml`](../../../.github/workflows/sandboxes.yml) | `mygo/sandbox/`、`mygo/versions.env`、`shared/sandbox/`、`shared/tooling/` | 构建 Docker 沙箱并在其中跑 `vfxtool sandbox all` |
| [`android.yml`](../../../.github/workflows/android.yml) | `android/`、`mygo/pair/`、`mygo/VERSION` | 构建 release APK（debug 密钥签名）并作为 `vfox-plus-android` 上传 |

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

要换新版本的包，改该文件 `source` 里的 `version` 与 `sha512`。注册表给的校验和是 base64，文件里要十六进制：

```sh
version=1.2.98
curl -fsS "https://registry.npmjs.org/@iconify-json%2fmaterial-symbols/$version" |
  grep -o '"integrity":"sha512-[^"]*"' | cut -d- -f2 | tr -d '"' | base64 -d | od -An -v -tx1 | tr -d ' \n'
```

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
`dist/sandbox/mygo/screenshots/`：xfwm4 Greybird 标题栏（现代 GNOME 应用观感），
Ubuntu 默认壁纸而非裸 X 的黑色。它让端到端构建的 `screenshots` 场景对着假 vfox
跑三遍，应用每到要截屏的时刻就打印 `e2e: SHOT <名称>`：

| 运行 | 拍什么 |
| --- | --- |
| 普通 | 每个页面 × 双主题 × 双语言与窄布局：`home`、`sdks`、`available`、`tasks`、`plugins`、`dialog`、`settings`、`dark-expanded`、`dark-sdks`、`dark-available`、`zh-home`、`zh-settings`、`compact-home`、`compact-detail` |
| `VFOX_PLUS_FAKE_DELAY=4` | 安装中：`installing` |
| `VFOX_PLUS_FAKE_INVENTORY=profiles` | 一台同时装着 Java、Flutter 与 Node.js 的机器，以及 vfox-plus 为这些插件提供的功能：`java-installed`、`java-available`、`java-choose`、`java-temurin`、`java-install-name`、`java-all`、`flutter-available`、`options`、`options-mirror`、`zh-options`、`zh-java-available`、`compact-java` |

要多拍一张，就在 `mygo/ui/e2e_scenarios.go` 的 `screenshotsScenario` 或 `profilesScreenshots` 里加一句
`s.shot("名称")`。它之前的步骤像人一样点击、等待，靠用户看到的名字找控件，所以改了名的按钮
会让场景失败，而不是拍出一张错的图。

README 与[用户指南](../user/index.md)引用 `home`、`sdks`、`dark-available`、`installing`、
`tasks`、`plugins`、`zh-settings`、`compact-home`、`compact-detail`、`java-choose`、
`java-temurin`、`flutter-available` 与 `options`。[使用说明](../user/using.md)引用
`java-install-name` 与 `options-mirror`，中文版引用 `zh-java-available` 与 `zh-options`。
[远端指南](../user/remote.md)引用 `webapp-en` 与 `webapp-zh`（英文与简体中文的配对后 Web 界面），
以及手机上 Web 界面的 `java-temurin` 场景的两种语言版本（`web-phone-java-temurin` 与
`web-phone-zh-java-temurin`，用 `--window 412,915` 拍）。其余的图是改界面时用来看的，没有文档引用它们。

**Web 界面。** 用 headless 服务与 headless Chrome 渲染，语言为截图固定。`vfxtool webheap`
可以拍 `--tab` 打开的页面（`home`、`sdks`、`plugins`、`tasks`、`settings`），或 `--scene` 演出的场景：
`java-available`、`java-choose`、`java-temurin`、`java-install-name`、`flutter-chips`、`options` 与
`options-mirror`（见 `shared/tooling/internal/webheap/scenes.go`，它的测试会对照
`mygo/pair/webapp.html` 检查场景点击的元素）。每张图都用自己的服务与 HOME，因为所选发行版保存在服务的
设置里，选了发行版的场景会改变它之后的每一张图：

```sh
go -C mygo build -o /tmp/vfox-plus-serve ./cmd/vfox-plus-serve
go -C shared/tooling build -o /tmp/vfxtool ./cmd/vfxtool

# shoot 名称 语言 宽,高 [webheap 的其他选项]
shoot() {
  name=$1 lang=$2 window=$3; shift 3
  rm -rf /tmp/shot-home && mkdir -p /tmp/shot-home/bin && cp /tmp/vfxtool /tmp/shot-home/bin/vfox
  HOME=/tmp/shot-home PATH=/tmp/shot-home/bin:$PATH \
    VFOX_PLUS_FIXTURES=shared/fixtures/vfox-1.0.13 VFOX_PLUS_FAKE_INVENTORY=profiles NO_COLOR=1 \
    /tmp/vfox-plus-serve serve --bind 127.0.0.1 --port 0 >serve.log 2>&1 &
  server=$!
  for i in $(seq 1 30); do
    LINK="$(grep -o 'http://127\.0\.0\.1:[0-9]*/auth/connect/[0-9a-f]*' serve.log | head -1)"
    [ -n "$LINK" ] && break
    sleep 1
  done
  /tmp/vfxtool webheap --server "${LINK%%/auth/connect/*}" \
    --pairing "$LINK" --output /tmp/shot.txt --screenshot "$name.png" \
    --lang "$lang" --window "$window" --idle 0 --chrome "$(which google-chrome)" "$@"
  kill "$server"
}
shoot webapp-en en 412,915
shoot web-java-temurin en 1280,800 --scene java-temurin
```

（`PATH` 首位放一个 `vfox` 可执行文件——`vfxtool` 的复制品配夹具即假
vfox——SDK 列表才有内容，`VFOX_PLUS_FAKE_INVENTORY=profiles` 让它成为装着 Java 与 Flutter 的机器；
`--lang` 固定界面语言；`--window 412,915` 拍手机竖屏，默认是 CI 测量的 1280×800 桌面窗口；
`--idle 0` 跳过测量堆所需的、停在打开页面上的 20 秒；`--dark` 用深色主题拍；
`--chrome` 接受任何讲 CDP 的 Chrome，不传则工具在 `$CHROME_BIN` 和 `PATH`
上自动找。）上传前看一眼图的尺寸
（`file webapp-en.png`）：不是每个 Chrome 都保持这么小的窗口。Chrome for
Testing 156 把 `--window 412,915` 拍成了 500×715 的图，同版本的
`chrome-headless-shell` 恰好是 412×915，尺寸不对时用 `--chrome` 传它。

**Android。** Android 截图 `screenshot-android` 来自 CI：`android.yml` 的 `measure` 任务在模拟器跑
release APK，`vfxtool measure-apk` 截冷启动，任务上传为
`screenshot-android` 构件。用 `gh run download` 取回后同样上传。

**上传。** 上传新图：

```sh
curl -s -F key=6d207e02198a847aa98d0a2a901485a5 -F action=upload -F format=json \
  -F source=@dist/sandbox/mygo/screenshots/home.png https://freeimage.host/api/1/upload |
  jq -r '.image.url'
```

输出 URL（key 是 freeimage.host 在 [API 页](https://freeimage.host/api) 给所有人的公开
key），替换 README 中的链接。curl 以 56 退出是连接断了：重试。每次上传都有新 URL，
旧图缓存不会串。匿名图无法删除；不可公开的东西不要传。
