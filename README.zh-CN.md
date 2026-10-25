# vfox-plus

> [English](README.md)

[vfox](https://vfox.dev) 的图形化前端。vfox 是跨平台版本管理器。本应用采用
[Material 3 Expressive](https://m3.material.io/) 设计语言。

它使用 [mygo](https://github.com/egoist/mygo) 0.6.3（Go）构建，原生用户界面走 GTK 3，
Material 3 Expressive 组件均为自研——无 Web 视图、无 HTML、无 JavaScript——单个 Go
可执行文件约 12 MB。在 Home 页约占用 45 MB PSS（见
[docs/zh-CN/memory.md](docs/zh-CN/memory.md)）。截图如下。

![vfox+ home page](https://iili.io/nkxZJOG.png)

| 已安装的 SDK 版本 | 可用版本（深色主题） |
| --- | --- |
| ![Installed versions of nodejs](https://iili.io/nkxB73u.png) | ![Available versions of nodejs, dark theme](https://iili.io/nkxBcyx.png) |

| 正在安装 | 可跟进、可取消的任务 |
| --- | --- |
| ![Installing a version of nodejs](https://iili.io/nkxBW6F.png) | ![Tasks](https://iili.io/nkxBNyv.png) |

| 插件仓库 | 简体中文设置 |
| --- | --- |
| ![Plugins](https://iili.io/nkxBvZN.png) | ![Settings](https://iili.io/nkxZEmJ.png) |

| 窄窗口 Home | 窄窗口 SDK |
| --- | --- |
| ![Home in a compact window](https://iili.io/nkxZhqN.png) | ![An SDK in a compact window](https://iili.io/nkxBZt2.png) |

## 实现

| 目录 | 技术 | 平台 | 包 | 状态 |
| --- | --- | --- | --- | --- |
| [`mygo/`](mygo) | mygo 0.6.3（Go），Linux 桌面走 GTK 3 原生界面、Windows 走 Windows 后端，自研 Material 3 Expressive 组件 | Linux 桌面、Windows 11 桌面 | `vfox-plus` | Linux 上可用；Windows 可构建、可打包、可安装（未签名侧加载），已手工冒烟测试 |

## 功能

- 一览已安装的 SDK、正在使用的版本和统计信息。
- 安装、卸载 SDK 版本，全局或按项目选择版本，带实时下载进度。
- 浏览 vfox 插件仓库，添加插件（也支持 URL 或文件）、更新和移除插件。
- 每次变更都作为可跟进、可取消、可重试的任务运行。
- 编辑 vfox 配置（代理、仓库、存储、旧版版本文件、缓存、超时）以及下载镜像等额外环境变量。
- 支持英文和简体中文；跟随桌面主题和强调色。

vfox+ 驱动 vfox 命令行工具，需另行安装 vfox（见
[vfox 安装指南](https://vfox.dev/guides/quick-start.html)）。

## 获取 vfox+

[`mygo`](.github/workflows/mygo.yml) 工作流每次变更都会构建安装包。下载最新一次
成功运行的 `vfox-plus-linux-x64` 构件，然后：

```sh
sudo apt install ./vfox-plus*_amd64.deb   # Ubuntu 26.04
vfox-plus
```

或解压 `.tar.gz` 到任意位置并运行 `./vfox-plus`。从源码构建见
[mygo README](mygo/README.zh-CN.md)；只有 Docker 时，
`vfxtool sandbox package` 会把两个文件构建到 `dist/sandbox/`，
不改变本机任何东西。

Windows 11 用户请下载 `vfox-plus-windows` 构件，解压后在 PowerShell 跑一键脚本
（本地自签名证书并安装，见
[Windows 打包](shared/packaging/windows/README.zh-CN.md)）：

```powershell
powershell -ExecutionPolicy Bypass -File Sign-Msix.ps1
```

## 仓库布局

```text
docs/       共享规约：架构、vfox CLI 契约、界面、设计系统、国际化
docs/zh-CN/ 上述文档的简体中文译文
shared/     共享资源：翻译、图标、捕获的 vfox 输出（测试夹具）、离线
            测试插件、端到端运行器、打包文件、vfxtool（Go 开发者
            工具）和沙箱的共享部分
mygo/       应用程序
.github/    CI 工作流：仓库检查、构建、内存报告和沙箱
```

| 文档 | 内容 |
| --- | --- |
| [docs/architecture.md](docs/zh-CN/architecture.md) | 分层、设置文件、依赖、内存 |
| [docs/vfox-cli-contract.md](docs/zh-CN/vfox-cli-contract.md) | GUI 如何运行 vfox 并解析其输出 |
| [docs/ui-spec.md](docs/zh-CN/ui-spec.md) | 界面、流程和状态 |
| [docs/design-system.md](docs/zh-CN/design-system.md) | Material 3 Expressive 颜色、形状、字体、动效和组件 |
| [docs/i18n.md](docs/zh-CN/i18n.md) | 翻译目录和术语表 |
| [docs/memory.md](docs/zh-CN/memory.md) | 应用程序的内存占用，同一方法测量 |

## 参与贡献

见 [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md)。

## 许可证

vfox-plus 采用 [Apache License 2.0](LICENSE)。Material Symbols 由 Google 采用
Apache License 2.0 授权。
