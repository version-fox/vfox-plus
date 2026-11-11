# vfox-plus 用户指南

> [English](../../en/user/index.md)

vfox-plus 是 [vfox](https://vfox.dev)（跨平台版本管理器）的图形化前端。遵循
[Material 3 Expressive](https://m3.material.io/) 设计语言，支持英文与简体中文，
跟随桌面主题与强调色。

![vfox+ 主页](https://iili.io/nm02B6v.png)

| 某 SDK 的已安装版本 | 可安装版本（深色主题） |
| --- | --- |
| ![已安装的 nodejs 版本](https://iili.io/nD3UCUQ.png) | ![可安装的 nodejs 版本，深色主题](https://iili.io/nm027MG.png) |

| 正在安装 | 可跟随、可取消的任务 |
| --- | --- |
| ![正在安装 nodejs 版本](https://iili.io/nm02Vte.png) | ![任务](https://iili.io/nD3UuzF.png) |

| 插件仓库 | 简体中文设置 |
| --- | --- |
| ![插件](https://iili.io/nD3UAWg.png) | ![设置](https://iili.io/nbhBVCF.png) |

| 窄窗口主页 | 窄窗口 SDK 页 |
| --- | --- |
| ![窄窗口主页](https://iili.io/nm02UDF.png) | ![窄窗口 SDK 页](https://iili.io/nD3UEbt.png) |

| 选择要列出的 Java 发行版 | Temurin，用 Chip 筛选长期支持版 |
| --- | --- |
| ![Java 的发行版](https://iili.io/nm02DNt.png) | ![Temurin 的长期支持版](https://iili.io/nm03nWu.png) |

| Flutter 的通道与架构 | Node.js 的选项 |
| --- | --- |
| ![稳定通道的 Flutter 版本](https://iili.io/nm03azF.png) | ![Node.js 的选项](https://iili.io/nD3gFpI.png) |

| 手机上的网页界面 | 网页界面里的 Java |
| --- | --- |
| ![手机上的网页界面主页](https://iili.io/nm0FB7p.png) | ![网页界面里的 Temurin 长期支持版](https://iili.io/nm0FW4j.png) |

## 它能做什么

- 一眼看到在用的版本与已安装 SDK 的健康状况。
- 安装、卸载、切换 SDK 版本，可全局或仅对当前项目，带实时下载进度。
- 浏览 vfox 插件仓库，添加插件（也支持 URL 或文件）、更新与移除。
- 每次变更都是一条任务，可跟随进度、可取消、可重试。
- 编辑 vfox 配置（代理、仓库源、存储目录、旧版版本文件、缓存、超时）与下载镜像
  等额外环境变量。
- 提供 vfox-java、vfox-nodejs、vfox-flutter、vfox-gradle 与 vfox-golang 在普通列表之外的功能：选择要列出的 Java
  发行版（共 31 个）、筛选 Chip（LTS、JavaFX、通道、架构）、按名称安装与下载镜像，见
  [Java、Node.js、Flutter、Gradle 与 Go](using.md#javanodejsfluttergradle-与-go)。
- 一步安装并使用某个版本、把机器上已有的 SDK 链接进来、给版本起别名、检查 vfox 与已安装 SDK 的健康状况，
  这些都是 vfox 1.1.0 才有的功能。
- 把同一个界面提供给浏览器；你允许的话，也提供给同一网络里的手机和其他电脑（走 HTTPS）；还可以
  不留窗口地在后台运行，见[网页界面与其他设备](remote.md)。

vfox-plus 通过 vfox 命令行工具干活，vfox 需另行安装（见
[vfox 安装指南](https://vfox.dev/zh-hans/guides/quick-start.html)）。

## 指南

| 指南 | 内容 |
| --- | --- |
| [安装](install.md) | 在 Linux、Windows 11 或安卓上装 vfox-plus。 |
| [使用](using.md) | 桌面应用：主页、SDK、插件、任务与设置。 |
| [网页界面与其他设备](remote.md) | 用浏览器使用 vfox-plus，不留窗口地后台运行，让手机和其他电脑连接。 |
| [安卓 App](android.md) | 用手机操控 PC 上的 vfox-plus。 |
| [网关（Docker）](gateway.md) | 把某个 vfox-plus 的界面显示给与容器配对的浏览器。 |

## 许可证

vfox-plus 采用 [Apache License 2.0](../../../LICENSE)。Material Symbols 由
Google 以 Apache License 2.0 授权。
