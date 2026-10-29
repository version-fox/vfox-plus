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
  [截图](#截图)）；图标字体等二进制构建输入在构建时生成。
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
| [`repository.yml`](../../../.github/workflows/repository.yml) | 每次变更 | 许可证头（SkyWalking Eyes）、工作流（actionlint）、vfxtool（`gofmt`、`go vet`、`go test`）、共享 JSON、无二进制文件 |
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

## 常见改动

### 文本

在 [`shared/i18n/app_en.arb`(../../../shared/i18n/app_en.arb) 中添加或修改消息
（含 `description` 与占位符元数据），再译到
[`app_zh.arb`(../../../shared/i18n/app_zh.arb)。见
[i18n.md](i18n.md) 及术语表。

### 图标

在 [`shared/icons.json`(../../../shared/icons.json) 中按含义添加 Material Symbol，
再重新生成图标字体（构建时 `vfxtool prepare` 会经 `vfxtool iconfont` 裁剪）。
字体构建时生成，不提交。

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
引用。`vfxtool sandbox screenshots` 可复现地渲染截图到
`dist/sandbox/mygo/screenshots/`：每个页面 × 双主题 × 双语言、窄布局与安装中状态，
xfwm4 Greybird 标题栏（现代 GNOME 应用观感），Ubuntu 默认壁纸而非裸 X 的黑色。
[用户指南](../user/index.md) 引用 `home`、`sdks`、`dark-available`、
`installing`、`tasks`、`plugins`、`zh-settings`、
`compact-home` 与 `compact-detail`。[远端指南](../user/remote.md)引用
`webapp-en` 与 `webapp-zh`：英文与简体中文的配对后 Web 界面。用 headless
服务与 headless Chrome 渲染，语言为截图固定：

```sh
go -C mygo build -o /tmp/vfox-plus-serve ./cmd/vfox-plus-serve
go -C shared/tooling build -o /tmp/vfxtool ./cmd/vfxtool
mkdir -p /tmp/shot-home/bin && cp /tmp/vfxtool /tmp/shot-home/bin/vfox
HOME=/tmp/shot-home PATH=/tmp/shot-home/bin:$PATH \
  VFOX_PLUS_FIXTURES=shared/fixtures/vfox-1.0.13 NO_COLOR=1 \
  /tmp/vfox-plus-serve serve --bind 127.0.0.1 --port 0 >serve.log 2>&1 &
for i in $(seq 1 30); do
  LINK="$(grep -o 'http://127\.0\.0\.1:[0-9]*/auth/connect/[0-9a-f]*' serve.log | head -1)"
  [ -n "$LINK" ] && break
  sleep 1
done
/tmp/vfxtool webheap --server "${LINK%%/auth/connect/*}" \
  --pairing "$LINK" --output /tmp/shot.txt --screenshot webapp-en.png \
  --lang en --window 412,915 --idle 0 --chrome "$(which google-chrome)"
```

（`PATH` 首位放一个 `vfox` 可执行文件——`vfxtool` 的复制品配夹具即假
vfox——SDK 列表才有内容；`--lang` 固定界面语言；`--window 412,915` 拍文档用的
手机竖屏，默认是 CI 测量的 1280×800 桌面窗口；`--idle 0` 跳过测量堆所需的、
停在打开页面上的 20 秒；`--tab` 在拍照前打开某个页面，`--dark` 用深色主题拍；
`--chrome` 接受任何讲 CDP 的 Chrome，不传则工具在 `$CHROME_BIN` 和 `PATH`
上自动找。）上传前看一眼图的尺寸
（`file webapp-en.png`）：不是每个 Chrome 都保持这么小的窗口。Chrome for
Testing 156 把 `--window 412,915` 拍成了 500×715 的图，同版本的
`chrome-headless-shell` 恰好是 412×915，尺寸不对时用 `--chrome` 传它。
Android 截图 `screenshot-android` 来自 CI：`android.yml` 的 `measure` 任务在模拟器跑
release APK，`vfxtool measure-apk` 截冷启动，任务上传为
`screenshot-android` 构件。用 `gh run download` 取回后同样上传。上传新图：

```sh
curl -s -F key=6d207e02198a847aa98d0a2a901485a5 -F action=upload -F format=json \
  -F source=@dist/sandbox/mygo/screenshots/home.png https://freeimage.host/api/1/upload |
  jq -r '.image.url'
```

输出 URL（key 是 freeimage.host 在 [API 页](https://freeimage.host/api) 给所有人的公开
key），替换 README 中的链接。curl 以 56 退出是连接断了：重试。每次上传都有新 URL，
旧图缓存不会串。匿名图无法删除；不可公开的东西不要传。
