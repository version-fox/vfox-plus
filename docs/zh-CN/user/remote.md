# 远端与安卓

> [English](../../en/user/remote.md)

`vfox-plus pair` 把浏览器连到本机的 vfox-plus。Ubuntu 的 bash 与 Windows
的 PowerShell 用法相同：

```sh
vfox-plus pair
```

打开它打印的链接。链接一次性有效，几分钟后过期。配对后的浏览器得到完整界面：
已安装 SDK 的安装/卸载/使用、插件仓库的添加/更新/移除、带实时进度与日志的任务、
以及设置。vfox-java、vfox-nodejs 与 vfox-flutter 这三个插件提供的功能一应俱全，和桌面应用一样：
Java 要列出的发行版、筛选 Chip、按名称安装，以及带下载镜像的选项页（见
[Java、Node.js 与 Flutter](using.md#javanodejs-与-flutter)）。安卓应用显示的是同一个界面。

| 配对后的 Web 界面 | 其中的 Java：Temurin，长期支持版 |
| --- | --- |
| ![配对后的 Web 界面](https://iili.io/n87KTiu.png) | ![配对后的 Web 界面里 Java 的发行版与 Chip](https://iili.io/n8eIQMG.png) |

Chrome 和 Edge 可以把配对后的界面安装为应用，有独立的窗口和 vfox+ 图标（地址栏里的安装
按钮，或菜单里的“安装”）。浏览器在本机上、以及在它信任其证书的 HTTPS 下才会提供，`serve`
生成的自签名证书不行。已安装的应用显示配对页时，再运行一次 `vfox-plus pair`，
打开新的链接即可。

一个 server 拥有应用状态：桌面在设置里打开远端访问（默认关闭）并运行时，
`pair` 连的就是桌面自己；否则连后台服务或回环 `serve`（有的话），都没有才
起自己的一次性服务。桌面版 `vfox-plus` 与 headless 版 `vfox-plus-serve`
接受同样的 `pair`、`serve` 和 `service` 命令，后者只是没有桌面应用。

## 后台服务

不用桌面时，用后台服务常驻共享 server。回环上的 `service start` 与 `pair`
无需额外确认：它们不出本机。

```sh
vfox-plus service start
vfox-plus service status
vfox-plus service stop
```

`pair` 与浏览器同样挂靠它。`service set disabled true` 关掉后台共享（之后
`pair` 需要桌面或 `serve` 在跑；`unset disabled` 恢复）。

## 另一台机器或容器

要从另一台机器连接，或在容器里跑 vfox-plus，常驻配对服务。远端访问默认关闭：
把本机开放给网络需要显式的 `--allow-remote`：

```sh
vfox-plus-serve serve --allow-remote --port 8080
```

把打印链接的地址换成可达地址再打开。发布 Docker 镜像无需显示器直接跑它：

```sh
docker build -f mygo/Dockerfile -t vfox-plus .
docker run --rm -p 8080:8080 vfox-plus
```

`-v vfox-plus-data:/home/vfoxplus` 卷在多次运行之间保留已装 SDK。配对后的浏览器
可在界面里为更多浏览器增发链接。

非回环服务走 HTTPS：不带自己的证书时首次启动生成自签名证书，并在浏览器核对打印
的指纹。用 `--tls-cert/--tls-key` 带自己的证书可免去警告。

## 安卓 App

安卓 App 用手机操控 PC 上的 vfox-plus：界面与上述配对浏览器相同。界面里指向其他网站的链接
（插件主页、插件文档）会在手机的浏览器里打开。

1. PC 上运行 `vfox-plus-serve serve --allow-remote --port 8080`（或 Docker 镜像，
   镜像自行 opt-in）并保持运行。终端打印配对链接，HTTPS 还有证书指纹；非回环
   监听还会打印该链接的二维码。
2. 在手机上安装 `vfox-plus-android` 产物的 `.apk` 并打开。
3. 用 App 扫描二维码，或把配对链接粘贴进 App 并连接。自签名证书会显示指纹：
   接受前先与 PC 终端上的指纹核对，和浏览器里一样。
4. App 保持配对，同一网络下自动重连。配对过期后扫码或粘贴新链接：从已配对浏览器
   增发，或重启 `vfox-plus-serve serve` 拿新的。

手机须能连到 PC（同一 Wi-Fi 或其它可达路由），PC 防火墙须放行服务端口（如 8080）。
浏览器只能设全局版本；固定版本到项目仍在桌面应用里做。
