# 安装 vfox-plus

> [English](../../en/user/install.md)

先装 [vfox](https://vfox.dev/zh-hans/guides/quick-start.html)：vfox-plus 靠它干活，
没有它不能工作。

## Linux

需要 GTK 3、fontconfig 与字体，以及 vfox 1.1.0 或更高：

```sh
sudo apt install libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk
```

另有一个可选的包：`libayatana-appindicator3-1`，它画出 vfox-plus 无窗口运行时（[轻量模式](remote.md#轻量模式)）
在 GNOME 的托盘里显示的图标。没有它 vfox-plus 就没有托盘图标，其他照常。vfox-plus 的软件包推荐安装它。

从[最新的 GitHub release](https://github.com/version-fox/vfox-plus/releases/latest) 下载适合你电脑（amd64 或 arm64）的 `.deb` 或 `.tar.gz`，二选一（Ubuntu 26.04 装包，示例为 amd64）：

```sh
sudo apt install ./vfox-plus*_amd64.deb
vfox-plus
```

或把 `.tar.gz` 解到任意位置，运行 `./vfox-plus`。

GitHub release 的 classic snap（amd64）则手动安装（strict 限制会挡住 vfox-plus 所驱动的工具链，
所以不上架商店）：

```sh
sudo snap install --dangerous --classic vfox-plus_*.snap
```

旁加载的 snap 不会自动更新：新版本下载新的 `.snap`，同样命令再装一次。

vfox-plus 是桌面应用：不接受任何参数，需要桌面会话来显示窗口。想同时在浏览器或手机上用，
在它的设置里开启网页界面（见[网页界面与其他设备](remote.md)）。

## Windows 11

从[最新的 GitHub release](https://github.com/version-fox/vfox-plus/releases/latest) 下载未签名的 `.msixbundle`，用管理员 PowerShell 安装：

```powershell
Add-AppxPackage -Path .\vfox-plus_<version>.msixbundle -AllowUnsigned
```

安装包暂未签名，正式签名版后续提供。卸载：

```powershell
Remove-AppxPackage -Package (Get-AppxPackage -Name dev.vfox.VfoxPlus).PackageFullName
```

## 安卓

从最新的 GitHub release 下载 `.apk` 到手机并安装。新版本直接装在旧版本上，与 PC 的配对保留；只有用别的密钥签名的 APK（例如开发构建）需要先卸载，这会忘记与 PC 的配对。然后再与 PC 上的 vfox-plus 配对。
见[安卓 App](android.md)。

## Docker

Docker 镜像是[网关](gateway.md)：它把某台电脑上运行着的 vfox-plus 的网页界面，显示给
与它配对的浏览器。镜像里没有 vfox，也没有窗口，所以它不能代替上面的安装。从仓库构建（暂未发布）并
运行：

```sh
docker build -f mygo/Dockerfile -t vfox-plus-gateway mygo
docker run --rm -p 8080:8080 vfox-plus-gateway
```
