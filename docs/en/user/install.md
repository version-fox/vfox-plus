# Install vfox-plus

> [简体中文](../../zh-CN/user/install.md)

Install [vfox](https://vfox.dev/guides/quick-start.html) first: vfox-plus
drives it and cannot work without it.

## Linux

Requirements: GTK 3, fontconfig and a font, plus vfox 1.1.0 or later:

```sh
sudo apt install libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk
```

The tray icon that vfox-plus shows while it runs without its window (the
[lightweight mode](remote.md#lightweight-mode)) needs no package of its own. On GNOME the AppIndicator
extension of the shell draws it, which Ubuntu enables. Where the desktop shows no tray icons, vfox-plus has
no tray icon and works as ever.

Download the `.deb` or the `.tar.gz` for your computer, amd64 or arm64, of the
[latest GitHub release](https://github.com/version-fox/vfox-plus/releases/latest),
then either install the package (Ubuntu 26.04; the example is the amd64 one):

```sh
sudo apt install ./vfox-plus*_amd64.deb
vfox-plus
```

or extract the `.tar.gz` anywhere and run `./vfox-plus`.

Each `.deb` and `.tar.gz` of a release has a build attestation, which names the
release workflow of this repository as the one that built it. To check a
download, with the [GitHub CLI](https://cli.github.com/):

```sh
gh attestation verify vfox-plus_<version>_amd64.deb --repo version-fox/vfox-plus
```

apt and dpkg do not check it: the attestation is what tells that the file is the
one the workflow built.

A classic snap of the GitHub release (amd64) installs by hand instead (strict
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

Download the `.msixbundle` of the
[latest GitHub release](https://github.com/version-fox/vfox-plus/releases/latest),
and install it in an elevated PowerShell:

```powershell
Add-AppxPackage -Path .\vfox-plus_<version>.msixbundle
```

The bundle is signed with a certificate that Windows trusts, so it installs
without `-AllowUnsigned`. A version from before the signed release has another
publisher, so it stays installed beside the new one: remove it first. To see the
installed packages, and to remove one of them:

```powershell
Get-AppxPackage -Name dev.vfox.VfoxPlus | Select-Object PackageFullName, Publisher
Remove-AppxPackage -Package <PackageFullName>
```

## Android

Download the `.apk` of the latest GitHub release to your phone and install it. A newer release installs over the one on the phone and keeps the pairing with the PC; only an APK signed with another key, such as a development build, must be uninstalled first. Then pair it with the vfox-plus on your PC. See [The Android app](android.md).

## Docker

The Docker image is the [gateway](gateway.md): it shows the web interface of a
vfox-plus that runs on a computer, in the browsers that you pair with it. It contains no vfox
and no window, so it does not replace the installation above. Build it from the repository
(it is not published) and run it:

```sh
docker build -f mygo/Dockerfile -t vfox-plus-gateway mygo
docker run --rm -p 8080:8080 vfox-plus-gateway
```
