# The web interface, other devices and Android

> [简体中文](../../zh-CN/user/remote.md)

vfox-plus is a desktop application, and it has no command line: you start it like any
other application, and what else it does is a matter of its settings. This page is about
three of them, in *Settings → Web interface*: the web interface, the lightweight mode and
the other devices that may connect. It also tells how to use that interface from a phone
(the [Android app](#android-app)) and from a container host (the [gateway](#the-gateway-docker)).

## The web interface

Turn on *Web interface* and vfox-plus serves its interface to the browsers of this
computer, for as long as it runs. The item says where: an address on this computer only,
like `http://127.0.0.1:41233`, which no other computer can reach. Choose *Open in browser*
and your browser opens it, paired by a one-time link. The link works once and expires after
five minutes, and *Copy link* makes another one for the clipboard.

The paired browser gets the full interface: installed SDKs with
install/uninstall/use, the plugin registry with add/update/remove, tasks with
live progress and logs, and the settings. The plugins vfox-java, vfox-nodejs and
vfox-flutter get everything they offer, as in the desktop application: the
distribution of Java to list, filter chips, installation by name and the options
with the download mirrors (see [Java, Node.js and
Flutter](using.md#java-nodejs-and-flutter)). The Android app shows the same
interface.

| The paired web interface | Java in it: Temurin, long-term support |
| --- | --- |
| ![The paired web interface](https://iili.io/nv67lFs.png) | ![The distribution and the chips of Java in the paired web interface](https://iili.io/n8eIitn.png) |

Chrome and Edge can install the paired interface as an application, with a window
of its own and the vfox+ icon (the install button in the address bar, or *Install*
in the menu). They offer it on the machine itself and over HTTPS with a certificate
that they trust, not over the one that vfox-plus makes for itself. Pairing does not
survive a restart of vfox-plus: when the installed application shows the pairing page,
choose *Open in browser* in the settings again.

Only the window can turn the web interface on or off. A paired browser or phone
cannot, because that would end the connection it is using.

## Lightweight mode

Closing the window quits vfox-plus, and the web interface with it. Turn on *Lightweight
mode* (it needs the web interface) and closing the window leaves vfox-plus running
without it. The web interface stays up for the browsers and the phones that use it, and
the tasks that run go on.

- **Bring the window back** from the icon that vfox-plus puts in the tray while it has no
  window: choose *Open vfox+* in its menu. Starting vfox-plus again, from the launcher as
  always, does the same, and works without a tray: no second vfox-plus starts, the one that
  runs opens its window and the new one ends.
- **Open the web interface** in the browser of this computer from the same menu, with
  *Open the web interface*.
- **End it for good** with *Quit vfox+*, which the menu of the icon has, and which the window
  shows in the settings while the lightweight mode is on, or from a paired browser or phone
  (*Settings → Connections → Quit vfox+*). Quitting cancels the tasks that run. Ending the
  process from the system quits it, too.

Closing the window frees what the window held, but vfox-plus stays a desktop application:
it keeps the toolkit of its window loaded, so it uses a few MB less, not a few tens.

### The tray icon

The icon shows only while there is no window, and goes when the window opens, so a
vfox-plus with a window has none. It is for the two desktops that vfox-plus is made and
tested for:

- **Windows 11.** In the notification area, with the other icons of the taskbar. Click it to
  bring the window back, or right-click it for the menu.
- **GNOME**, the desktop of Ubuntu. In the panel at the top, which shows such icons through
  its AppIndicator extension: Ubuntu has it on, and other distributions have it as the package
  `gnome-shell-extension-appindicator`. Click the icon for the menu. It also needs the library
  `libayatana-appindicator3`: the package of vfox-plus recommends it, so it comes with it
  unless you install without recommended packages (`sudo apt install libayatana-appindicator3-1`
  adds it later).

Other desktops are not supported. KDE Plasma and the desktops whose tray is of the old kind get
no icon that vfox-plus promises: use GNOME. Where there is no icon, nothing else changes: vfox-plus
runs without its window as before, and you bring the window back by starting vfox-plus again, or use
the browser.

## Other devices

A phone, or another computer, can use the web interface too. Turn on *Other devices*
(it needs the web interface), and confirm. It is off by default: it opens vfox-plus to the
network, so allow it only in a network that you trust.

- vfox-plus listens on the **port** that the settings show, 8369 unless you change it. The
  firewall of this computer has to allow it.
- Other devices connect over **HTTPS**. Without a certificate of your own, vfox-plus makes
  one for itself and signs it itself, and the settings show its SHA-256 fingerprint: compare
  it with the one that the browser or the Android app shows before you trust it. To use
  your own, choose *Certificate → Browse…*, then the certificate and its private key
  (PEM files).
- *Pair a phone* shows the **QR code** of a pairing link, and the link. A computer that is
  on more than one network has more than one address; choose the one that the phone
  reaches. Scan the code with the [Android app](#android-app), or open the link in a
  browser of another computer. It works once and expires after five minutes.
- A paired device keeps its session for 24 hours. Turning *Other devices* off, or
  closing vfox-plus, ends the connections of all of them at once, and they have to pair
  again.

The settings of the web interface show the same switches: from a paired browser or phone
you can allow other devices, change the port and turn the lightweight mode on or off, and
*Pair a phone* shows the code there, too, which is how a computer without a window
(lightweight mode) pairs a phone.

## Android app

The Android app drives the vfox-plus on your PC from your phone: it shows the
same interface as the paired browser above. Links to other sites, such as the
homepage of a plugin or its documentation, open in the browser of the phone.

1. On the PC, open vfox-plus, go to *Settings → Web interface* and turn on the web
   interface and *Other devices*. Allow the port in the firewall of the PC.
2. Install the `vfox-plus-android` artifact's `.apk` on your phone and open
   it.
3. Choose *Show code* next to *Pair a phone* on the PC, and scan the QR code with the
   app, or copy the link and paste it into the app, and connect. If the PC uses a
   certificate that it made itself, the app shows its fingerprint: compare it with the one
   in the settings on the PC before accepting, exactly as you would in a browser.
4. The app stays paired and reconnects automatically on the same network. If
   pairing expires, or vfox-plus on the PC was restarted, scan a fresh code.

The app opens on the pairing screen: scan the QR code of the PC (Android
asks for the camera once) or paste the link. A self-signed certificate shows its
fingerprint first, to compare with the one on the PC.

| The pairing screen | Scanning the QR code | The fingerprint of a self-signed certificate |
| --- | --- | --- |
| ![The pairing screen of the Android app](https://iili.io/nSoyuyX.png) | ![The scanner of the Android app, with the viewfinder and the toolbar](https://iili.io/nSoyU91.png) | ![The dialog that shows the fingerprint of a self-signed certificate](https://iili.io/nSoyiMv.png) |

Once paired, the app shows the interface of the paired browser: the home page
with the plugins in use, the SDKs with their versions, and the settings. The
first card of the settings, *This phone*, pairs another link or disconnects the
phone.

| Home | An SDK | The settings, with the card of this phone |
| --- | --- | --- |
| ![The home page in the Android app](https://iili.io/nSx9JVf.png) | ![An SDK in the Android app: flutter, with its installed versions](https://iili.io/nSx9BR9.png) | ![The settings in the Android app, with the card of this phone](https://iili.io/nSx91WJ.png) |

The app follows the dark theme of the phone. When it cannot reach the PC (vfox-plus
stopped, another network, a firewall in the way), it says so and offers to
try again or to pair a new link.

| The pairing screen in the dark theme | Home in the dark theme | A PC that cannot be reached |
| --- | --- | --- |
| ![The pairing screen of the Android app in the dark theme](https://iili.io/nSx9kWG.png) | ![The home page in the Android app in the dark theme](https://iili.io/nSx94JS.png) | ![The Android app when the PC cannot be reached](https://iili.io/nSx9QWb.png) |

The app needs your PC reachable from the phone (same Wi-Fi or another network
route) and the port of the web interface (8369 by default) allowed through the PC
firewall. Browsers can only set the global version; pinning versions to a
project stays in the desktop application.

## The gateway (Docker)

The Docker image is not vfox-plus in a container: it has no vfox, no window and no SDKs of
its own, and it takes no arguments. It is a *gateway*. Like the Android app, it connects to
one vfox-plus that runs on a computer (with the web interface and *Other devices* turned
on), and shows the interface of that vfox-plus to the browsers that you pair with the
gateway. Use it to put the interface at an address of your choice, for example on a
container host that your browsers reach and your phone does not.

There is no published image yet. Build it from the repository, and run it with the port
mapped:

```sh
docker build -f mygo/Dockerfile -t vfox-plus-gateway mygo
docker run --rm -p 8080:8080 -v vfox-plus-gateway:/home/vfoxplus/.config/vfox-plus-gateway vfox-plus-gateway
```

The volume keeps what the gateway remembers between runs: the vfox-plus it is paired with,
its session there, and the certificate of the gateway itself. Without it, the gateway starts
from nothing each time the container is made.

1. **Pair a browser with the gateway.** The gateway prints (see `docker logs`) the SHA-256
   fingerprint of its certificate, and a one-time link such as
   `https://localhost:8080/auth/connect/…`. Replace `localhost` with the address that reaches
   the container and open the link in a browser. The gateway serves HTTPS with a certificate
   that it makes for itself, so the browser warns about it: compare the fingerprint that the
   browser shows with the printed one before you go on. The link works once and expires after
   five minutes; restart the container for a new one, which is the last in the log.
2. **Pair the gateway with vfox-plus.** The page of the gateway has a field for the pairing
   link of a vfox-plus. On the PC, choose *Settings → Web interface → Pair a phone → Show
   code → Copy link*, and paste the link there. The gateway then shows the fingerprint of the
   certificate of vfox-plus: compare it with the one in the settings of vfox-plus
   (*Certificate*), and choose *They are the same: trust and pair*. From then on the gateway
   trusts that certificate and no other, and asks again, with a warning, if the one at the
   address changes. A certificate that an authority signed needs no comparison.
3. **Use it.** The browser is taken to the interface of vfox-plus: the one a paired browser
   has, served by that vfox-plus itself, so the page always fits the version that runs. Pair
   more browsers with *Pair another browser* on the page of the gateway, and pair the gateway
   with another vfox-plus by pasting its link there (*Forget this vfox+* lets go of the one it
   has).

The session of the gateway at vfox-plus lasts a day, like that of a phone, and the gateway
renews it every few hours for as long as it runs. When vfox-plus quits or starts again its
sessions end: the page of the gateway says so, and you paste a new link. While vfox-plus is
not running, the interface says that it cannot be reached.

Good to know:

- **A vfox-plus on the same computer.** The link for other devices works from the container:
  it is an address of the PC on the network. The *Open in browser* link
  (`http://127.0.0.1:…`) names the container itself when it is pasted into the gateway. On
  Linux, run the container with `--network host` instead of `-p` to reach the loopback address
  of the computer; the gateway then listens on port 8080 of the computer.
- **A certificate of your own.** Put a certificate that browsers trust in the volume, as
  `cert.pem` and `key.pem` (PEM files), and the gateway serves with it and makes none of its
  own.
- **Who may use it.** A browser that is paired with the gateway has the whole interface of the
  vfox-plus it is paired with, as a paired phone does, and it can have the gateway connect to any
  address that the container can reach (that is how the gateway is paired with a vfox-plus). The
  gateway keeps its session at vfox-plus in the volume, in a file that only its user can read:
  keep the volume as you keep a password. The log of the container shows the link that pairs a
  browser, so whoever can read the log can pair one.
- **The port.** The port inside the container is 8080; map it as you like (`-p 9443:8080`),
  and allow it in the firewall of the host.
