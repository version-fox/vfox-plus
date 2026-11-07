# Install vfox-plus

> [简体中文](../../zh-CN/user/install.md)

Install [vfox](https://vfox.dev/guides/quick-start.html) first: vfox-plus
drives it and cannot work without it.

## Linux

Requirements: GTK 3, fontconfig and a font, plus vfox 1.0.0 or later:

```sh
sudo apt install libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk
```

One more package is optional: `libayatana-appindicator3-1`, which draws the tray icon that
vfox-plus shows while it runs without its window (the
[lightweight mode](remote.md#lightweight-mode)), on GNOME. Without it vfox-plus has no tray icon and
works as ever. The package of vfox-plus recommends it.

Download the `vfox-plus-linux-x64` artifact of the latest successful build,
then either install the package (Ubuntu 26.04):

```sh
sudo apt install ./vfox-plus*_amd64.deb
vfox-plus
```

or extract the `.tar.gz` anywhere and run `./vfox-plus`.

A classic snap of the GitHub release installs by hand instead (strict
confinement would fence off the toolchains that vfox-plus drives, so the
snap is not in the store):

```sh
sudo snap install --dangerous --classic vfox-plus_*.snap
```

A sideloaded snap does not refresh itself: download the `.snap` of the new
release and install it again the same way.

vfox-plus is a desktop application: it takes no arguments, and needs a desktop session
to show its window. To use it from a browser or a phone as well, turn on its web
interface in its settings (see [The web interface and other devices](remote.md)).

## Windows 11

Download the `vfox-plus-windows` artifact, extract it and install the
unsigned `.msixbundle` in an elevated PowerShell:

```powershell
Add-AppxPackage -Path .\vfox-plus_<version>.msixbundle -AllowUnsigned
```

The bundle is not signed yet; an officially signed bundle will be provided
later. To remove it again:

```powershell
Remove-AppxPackage -Package (Get-AppxPackage -Name dev.vfox.VfoxPlus).PackageFullName
```

## Android

Install the `vfox-plus-android` artifact's `.apk` on your phone and pair it
with the vfox-plus on your PC. See [The Android app](android.md).

## Docker

The Docker image is the [gateway](gateway.md): it shows the web interface of a
vfox-plus that runs on a computer, in the browsers that you pair with it. It contains no vfox
and no window, so it does not replace the installation above. Build it from the repository
(it is not published) and run it:

```sh
docker build -f mygo/Dockerfile -t vfox-plus-gateway mygo
docker run --rm -p 8080:8080 vfox-plus-gateway
```
