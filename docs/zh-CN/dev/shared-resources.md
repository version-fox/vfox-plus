# 共享资源

> [English](../../en/dev/shared-resources.md)

vfox-plus 使用的资源。应用从这里直接读取，从不复制，保证每份只存在一处。

| 路径 | 内容 | 规约 |
| --- | --- | --- |
| [`i18n/`](../../../shared/i18n) | ARB（带 ICU 消息的 JSON）翻译目录：`app_en.arb`（模板）与 `app_zh.arb`。 | [docs/zh-CN/i18n.md](i18n.md) |
| [`icons.json`](../../../shared/icons.json) | 界面使用的 Material Symbols Rounded 图标，按含义命名，以及 `vfxtool prepare` 从中切出 SVG 的 npm 包（版本与校验和）。 | [docs/zh-CN/design-system.md](design-system.md#图标) |
| [`fixtures/`](../../../shared/fixtures) | 捕获的 vfox 输出与期望解析结果，`vfxtool capture` 捕获。 | [fixtures/README.zh-CN.md](fixtures.md)、[docs/zh-CN/vfox-cli-contract.md](vfox-cli-contract.md) |
| [`test-plugin/`](../../../shared/test-plugin) | 对真实 vfox 做端到端测试的离线 vfox 插件，及下载它的本地服务器协议。 | [test-plugin/README.zh-CN.md](test-plugin.md)、[docs/zh-CN/vfox-cli-contract.md](vfox-cli-contract.md#对真实-vfox-测试) |
| [`packaging/linux/`](../../../shared/packaging/linux) | 应用的桌面入口、AppStream 元数据与图标，`vfxtool build-deb` 打包为 `.deb` 与 `.tar.gz`。 | [mygo/README.zh-CN.md](implementation-mygo.md#打包) |
| [`packaging/windows/`](../../../shared/packaging/windows) | 应用的 MSIX 清单、徽标与 bundle 布局，`vfxtool build-msix` 为 Windows 11 打包为未签名 `.msix` 与 `.msixbundle`。 | [packaging/windows/README.zh-CN.md](packaging-windows.md)、[mygo/README.zh-CN.md](implementation-mygo.md#打包) |
| [`tooling/`](../../../shared/tooling) | `vfxtool`：取代原 Python 与 shell 脚本的单一 Go 工具：许可证收集、端到端运行器、内存测量、截图、离线测试服务器、Debian 打包器与 MSIX 打包器。 | [docs/zh-CN/design-system.md](design-system.md#图标)、[docs/zh-CN/memory.md](memory.md#如何测量) |
| [`sandbox/`](../../../shared/sandbox) | 实现的 Docker 沙箱的基础镜像（`vfxtool sandbox` 驱动）。 | [sandbox/README.zh-CN.md](sandbox.md) |

改动这里影响应用：跑其测试。
