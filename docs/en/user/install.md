# Install vfox-plus

> [简体中文](../../zh-CN/user/install.md)

Install [vfox](https://vfox.dev/guides/quick-start.html) first: vfox-plus
drives it and cannot work without it.

## Linux

Requirements: GTK 3, fontconfig and a font, plus vfox 1.0.0 or later:

```sh
sudo apt install libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk
```

Download the `vfox-plus-linux-x64` artifact of the latest successful build,
then either install the package (Ubuntu 26.04):

```sh
sudo apt install ./vfox-plus*_amd64.deb
vfox-plus
```

or extract the `.tar.gz` anywhere and run `./vfox-plus`.

To run only the pairing server, without the desktop interface or its GTK
dependency — on a server or a container host, for example — download the
`vfox-plus-serve-linux` artifact instead and run
`./vfox-plus-serve serve --allow-remote --port 8080`. It accepts the same
`pair`, `serve` and `service` commands; only the desktop application is gone.

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

## Docker (no display needed)

The published image runs the pairing server with no display. See
[Remote and Android](remote.md).

```sh
docker build -f mygo/Dockerfile -t vfox-plus .
docker run --rm -p 8080:8080 vfox-plus
```

## Android

Install the `vfox-plus-android` artifact's `.apk` on your phone and pair it
with the vfox-plus on your PC. See [Remote and Android](remote.md).
