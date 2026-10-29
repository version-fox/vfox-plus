# Remote and Android

> [简体中文](../../zh-CN/user/remote.md)

`vfox-plus pair` connects a browser to the vfox-plus on your machine. It works
the same in bash on Ubuntu and in PowerShell on Windows:

```sh
vfox-plus pair
```

Open the link it prints. Links work once and expire after a few minutes. The
paired browser gets the full interface: installed SDKs with
install/uninstall/use, the plugin registry with add/update/remove, tasks with
live progress and logs, and the settings.

![The paired web interface](https://iili.io/nv67lFs.png)

Chrome and Edge can install the paired interface as an application, with a window
of its own and the vfox+ icon (the install button in the address bar, or *Install*
in the menu). They offer it on the machine itself and over HTTPS with a certificate
that they trust, not over the self-signed one that `serve` generates. When the
installed application shows the pairing page, run `vfox-plus pair` again and open
the new link.

One server owns the application state: while the desktop application runs
with Remote access switched on in its Settings (off by default), `pair` links
to the desktop itself; otherwise it links to the background
service or a loopback `serve` if one runs, and starts its own one-shot server
if not. The desktop `vfox-plus` and the headless `vfox-plus-serve` accept the
same `pair`, `serve` and `service` commands; only the desktop application is
gone from the latter.

## Background service

Without the desktop, keep a shared server up with the background service.
`service start` and `pair` on loopback need no extra confirmation: they never
leave the machine.

```sh
vfox-plus service start
vfox-plus service status
vfox-plus service stop
```

`pair` and browsers attach to it the same way. `service set disabled
true` switches the background sharing off (`pair` then needs the desktop or a
`serve` running; `unset disabled` turns it back on).

## Another machine, or a container

To connect a browser on another machine, or to run vfox-plus in a container,
keep the pairing server up instead. Remote access is off by default: opening
the machine to the network needs an explicit `--allow-remote`:

```sh
vfox-plus-serve serve --allow-remote --port 8080
```

and open the link it prints, with its address replaced by the reachable one.
The published Docker image runs this server with no display needed:

```sh
docker build -f mygo/Dockerfile -t vfox-plus .
docker run --rm -p 8080:8080 vfox-plus
```

A `-v vfox-plus-data:/home/vfoxplus` volume keeps the installed SDKs between
runs. A paired browser mints links for more browsers from the interface.

Non-loopback servers speak HTTPS: without your own certificate a
self-signed one is generated on first start, and its fingerprint is printed
for comparison in the browser. Bring your own certificate with
`--tls-cert/--tls-key` to avoid the warning.

## Android app

The Android app drives the vfox-plus on your PC from your phone: it shows the
same interface as the paired browser above.

1. On the PC, run `vfox-plus-serve serve --allow-remote --port 8080` (or the Docker
   image, which opts in by itself) and keep it running. The terminal prints a
   pairing link and, for HTTPS, a certificate fingerprint. On a non-loopback
   bind it also prints a QR code of the link.
2. Install the `vfox-plus-android` artifact's `.apk` on your phone and open
   it.
3. Scan the QR code with the app, or paste the pairing link into it and
   connect. If the server uses a self-signed certificate, the app shows its
   fingerprint: compare it with the one on the PC terminal before accepting,
   exactly as you would in a browser.
4. The app stays paired and reconnects automatically on the same network. If
   pairing expires, scan or paste a fresh link: mint a new one from a paired
   browser, or restart `vfox-plus-serve serve`.

The app needs your PC reachable from the phone (same Wi-Fi or another network
route) and the server port (for example 8080) allowed through the PC
firewall. Browsers can only set the global version; pinning versions to a
project stays in the desktop application.
