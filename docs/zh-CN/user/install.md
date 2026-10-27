# 安装 vfox-plus

> [English](../../en/user/install.md)

先装 [vfox](https://vfox.dev/zh-hans/guides/quick-start.html)：vfox-plus 靠它干活，
没有它不能工作。

## Linux

需要 GTK 3、fontconfig 与字体，以及 vfox 1.0.0 或更高：

```sh
sudo apt install libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk
```

下载最新成功构建的 `vfox-plus-linux-x64` 产物，二选一（Ubuntu 26.04 装包）：

```sh
sudo apt install ./vfox-plus*_amd64.deb
vfox-plus
```

或把 `.tar.gz` 解到任意位置，运行 `./vfox-plus`。

只跑配对服务、不要桌面界面及其 GTK 依赖时（比如服务器或容器宿主机），下载
`vfox-plus-serve-linux` 产物，运行
`./vfox-plus-serve serve --allow-remote --port 8080`。它接受同样的 `pair`、
`serve` 和 `service` 命令，只是没有桌面应用。

## Windows 11

下载 `vfox-plus-windows` 产物并解压，用管理员 PowerShell 安装未签名的
`.msixbundle`：

```powershell
Add-AppxPackage -Path .\vfox-plus_<version>.msixbundle -AllowUnsigned
```

安装包暂未签名，正式签名版后续提供。卸载：

```powershell
Remove-AppxPackage -Package (Get-AppxPackage -Name dev.vfox.VfoxPlus).PackageFullName
```

## Docker（无需显示器）

发布镜像直接跑配对服务，无需显示器。见[远端与安卓](remote.md)。

```sh
docker build -f mygo/Dockerfile -t vfox-plus .
docker run --rm -p 8080:8080 vfox-plus
```

## 安卓

在手机上安装 `vfox-plus-android` 产物的 `.apk`，再与 PC 上的 vfox-plus 配对。
见[远端与安卓](remote.md)。
