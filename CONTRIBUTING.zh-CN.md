# 参与 vfox-plus 贡献

> [English](CONTRIBUTING.md)

感谢你的帮助！本仓库存放 vfox-plus 应用程序（见
[README.zh-CN.md](README.zh-CN.md)）；下文大多适用于它的各个部分。

## 基本规则

- 代码注释、提交信息和 Pull Request 使用**英文**。用户可见文本走
  [`shared/i18n`](shared/i18n) 中的翻译目录，提供英文和简体中文。
- 文档使用**英文并附简体中文译文**：每个 Markdown 文档都有中文对照版本（见
  [文档语言](#文档语言)）。
- 每个源文件开头都带有本仓库使用的 Apache-2.0 许可证头。用
  `docker run --rm -v "$PWD:/github/workspace" apache/skywalking-eyes:0.9.0 header check`
  检查，用 `header fix` 补齐。
- 运行时依赖必须采用与 Apache License 2.0 兼容的许可证。
- 各层共享的行为——vfox CLI 契约、界面、设计标记、翻译、图标、设置——在
  [`docs`](docs) 和 [`shared`](shared) 中规定。改代码时同步改规约。
- 文件使用 UTF-8、LF 换行并以换行结尾（见
  [`.editorconfig`](.editorconfig) 和 [`.gitattributes`](.gitattributes)）。
- 不提交二进制文件。截图等图片上传到图床后引用（见
  [截图](#截图)）；图标字体等二进制构建输入在构建时生成。
  CI 会拒绝二进制文件。

## 工作流

1. 大改动先开 issue 或在已有 issue 下留言。
2. 从 `main` 建分支，Pull Request 保持聚焦。
3. 跑检查；与 CI 相同，列表见
  [mygo README](mygo/README.zh-CN.md#日常命令)：

    ```sh
    cd mygo
    golangci-lint fmt --diff
    golangci-lint run && golangci-lint run --build-tags e2e
    CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./... && vfxtool coverage coverage.out 85
    ```

    只有 Docker、又不想动本机 vfox 安装时，用沙箱跑全部检查、端到端测试和打包，
    `vfxtool sandbox all`（见
    [shared/sandbox](shared/sandbox/README.zh-CN.md)）。

4. 在 Pull Request 中说明改了什么、怎么测试的。界面变化附浅色和深色主题截图。

## 持续集成

| 工作流 | 触发 | 检查 |
| --- | --- | --- |
| [`repository.yml`](.github/workflows/repository.yml) | 每次变更 | 许可证头（SkyWalking Eyes）、工作流（actionlint）、vfxtool（`gofmt`、`go vet`、`go test`）、共享 JSON、无二进制文件 |
| [`memory.yml`](.github/workflows/memory.yml) | 构建工作流结束、Pull Request | 在 Pull Request 下评论应用内存及与 `main` 的差异（见[内存评论](docs/zh-CN/memory.md#pull-request-中的内存)） |
| [`mygo.yml`](.github/workflows/mygo.yml) | `mygo/`、`shared/` | 格式化与 lint（golangci-lint，含端到端构建标签）、带竞态检测的单元测试与覆盖率门禁、Go 模块许可证、应用内端到端场景（夹具与真实 vfox）、内存测量、发布构建与冒烟测试、`.deb` 和 `.tar.gz`（lintian 校验并装入干净 Ubuntu 容器），以及 Windows 11 未签名 `.msix` 和 `.msixbundle`（x64 与 arm64，Linux 上 `vfxtool validate-msix` 校验，Windows Server 2025（x64）与 ARM 版 Windows 11（arm64）上校验并安装/卸载） |
| [`sandboxes.yml`](.github/workflows/sandboxes.yml) | `mygo/sandbox/`、`mygo/versions.env`、`shared/sandbox/`、`shared/tooling/` | 构建 Docker 沙箱并在其中跑 `vfxtool sandbox all` |

Dependabot 负责 Go 模块和 GitHub Actions 的更新；action 按提交 SHA 锁定。

工作流用
[`install-linux-packages`](.github/actions/install-linux-packages/action.yml) 安装
Ubuntu 包：下载的压缩包按包列表和周做缓存，每次 8 个文件并行下载，断点续传，
慢镜像自动切换，15 分钟内完成。部分 runners 的 Ubuntu 镜像有时很慢：
2026-10-23 某次连接只有 40 KB/s，而别的连接几秒下载 90 MB。apt 不校验已在目录中的
压缩包，所以 action 按哈希逐个校验。失败的运行会保存已下载部分，下次继续。

重建应用的 Pull Request 会收到一条内存评论及与 `main` 的差异，新提交会替换它
而不是追加。超过 CI 噪声的变化会加粗；小数字先读
[Pull Request 中的内存](docs/zh-CN/memory.md#pull-request-中的内存)再下结论。

## 文档语言

每个 Markdown 文档都有英文 canonical 版和简体中文译文，同一 Pull Request 提交。
英文为准；中文逐节对照，代码块、命令行、文件路径及
[docs/zh-CN/i18n.md](docs/zh-CN/i18n.md) 术语表保持不变。

| 英文（canonical） | 简体中文 |
| --- | --- |
| `README.md` | `README.zh-CN.md`（同目录） |
| `CONTRIBUTING.md` | `CONTRIBUTING.zh-CN.md` |
| `mygo/README.md` | `mygo/README.zh-CN.md` |
| `shared/README.md`、`shared/*/README.md` | 旁边的 `README.zh-CN.md` |
| `docs/*.md` | `docs/zh-CN/*.md`（同名） |

每个文档顶部互链（英文用 `[简体中文](…)`，中文用 `[English](…)`），
`vfxtool docs-check` 校验映射；仓库工作流在缺对照时失败。新增文档同样成对添加。

## 常见改动

### 文本

在 [`shared/i18n/app_en.arb`](shared/i18n/app_en.arb) 中添加或修改消息
（含 `description` 与占位符元数据），再译到
[`app_zh.arb`](shared/i18n/app_zh.arb)。见
[docs/zh-CN/i18n.md](docs/zh-CN/i18n.md) 及术语表。

### 图标

在 [`shared/icons.json`](shared/icons.json) 中按含义添加 Material Symbol，
再重新生成图标字体（构建时 `vfxtool prepare` 会经 `vfxtool iconfont` 裁剪）。
字体构建时生成，不提交。

### vfox 输出

vfox 变化时用 `vfxtool capture` 重抓夹具，按
[`shared/fixtures/README.zh-CN.md`](shared/fixtures/README.zh-CN.md) 更新，并在
[docs/zh-CN/vfox-cli-contract.md](docs/zh-CN/vfox-cli-contract.md) 中记录。

### 截图

文档图片不提交：托管在 [freeimage.host](https://freeimage.host)（免账号免费图床）后
引用。`vfxtool sandbox screenshots` 可复现地渲染截图到
`dist/sandbox/mygo/screenshots/`：每个页面 × 双主题 × 双语言、窄布局与安装中状态，
xfwm4 Greybird 标题栏（现代 GNOME 应用观感），Ubuntu 默认壁纸而非裸 X 的黑色。
[README.zh-CN.md](README.zh-CN.md) 引用 `home`、`sdks`、`dark-available`、
`installing`、`tasks`、`plugins`、`zh-settings`、
`compact-home` 与 `compact-detail`。上传新图：

```sh
curl -s -F key=6d207e02198a847aa98d0a2a901485a5 -F action=upload -F format=json \
  -F source=@dist/sandbox/mygo/screenshots/home.png https://freeimage.host/api/1/upload |
  jq -r '.image.url'
```

输出 URL（key 是 freeimage.host 在 [API 页](https://freeimage.host/api) 给所有人的公开
key），替换 README 中的链接。curl 以 56 退出是连接断了：重试。每次上传都有新 URL，
旧图缓存不会串。匿名图无法删除；不可公开的东西不要传。
