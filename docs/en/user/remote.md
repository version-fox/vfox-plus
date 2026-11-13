# The web interface and other devices

> [简体中文](../../zh-CN/user/remote.md)

vfox-plus is a desktop application, and it has no command line: you start it like any
other application, and what else it does is a matter of its settings. This page is about
three of them, in *Settings → Web interface*: the web interface, the lightweight mode and
the other devices that may connect. To use that interface from a phone, see the
[Android app](android.md); to show it at an address of your choice, from a container,
see the [gateway](gateway.md).

## The web interface

Turn on *Web interface* and vfox-plus serves its interface to the browsers of this
computer, for as long as it runs. The item says where: an address on this computer only,
like `http://127.0.0.1:41233`, which no other computer can reach. Choose *Open in browser*
and your browser opens it, paired by a one-time link. The link works once and expires after
five minutes, and *Copy link* makes another one for the clipboard.

![The settings of the web interface: the switch and the address that serves, the button that opens it in the browser, the lightweight mode and the other devices](https://iili.io/nyiVJrF.png)

The paired browser gets the full interface: installed SDKs with
install/uninstall/use, the plugin registry with add/update/remove, tasks with
live progress and logs, and the settings. The plugins vfox-java, vfox-nodejs,
vfox-flutter, vfox-gradle and vfox-golang get everything they offer, as in the desktop
application: the distribution of Java to list, filter chips, installation by name,
the links and aliases of vfox 1.1.0 and the options with the download mirrors (see
[Java, Node.js, Flutter, Gradle and Go](using.md#java-nodejs-flutter-gradle-and-go)).
The [Android app](android.md) shows the same interface.

| The paired web interface | Java in it: Temurin, long-term support |
| --- | --- |
| ![The paired web interface](https://iili.io/nmidc0B.png) | ![The distribution and the chips of Java in the paired web interface](https://iili.io/nmiuiE7.png) |

Chrome and Edge can install the paired interface as an application, with a window
of its own and the vfox+ icon (the install button in the address bar, or *Install*
in the menu). They offer it on the machine itself and over HTTPS with a certificate
that they trust, not over the one that vfox-plus makes for itself. Pairing does not
survive a restart of vfox-plus: when the installed application shows the pairing page,
choose *Open in browser* in the settings again.

Only the window can turn the web interface on or off. A paired browser or phone
cannot, because that would end the connection it is using.

## Publishing to the internet

At home the interface is reachable at the address of this computer on the network, which
is enough for a phone on the same Wi-Fi. From anywhere else it needs an address on the
internet, and vfox-plus does not run a relay of its own: the address comes from something
that publishes the interface.

- **A tunnel of your own.** Put the address that a reverse proxy, a tunnel or the
  [gateway](gateway.md) publishes in **Public address**; pairing links and the code use it.
- **The built-in switch.** *Public tunnel* publishes the interface through a free
  [Cloudflare quick tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/trycloudflare/):
  no account, no configuration, one switch. When the program is not installed yet, turning
  it on downloads it (about 50 MB) from Cloudflare's GitHub release and checks its SHA-256.
  The address is random, such as
  `https://one-two-three.trycloudflare.com`, and changes at every start. It ends when you
  turn the switch off or quit vfox+.

The tunnel carries the traffic of the phones and browsers that connect through it past
Cloudflare; quick tunnels are meant for short-lived use, without a promise of uptime. What
protects the interface is the pairing: a device still needs a one-time link, and the
session lasts 24 hours.

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

| The question that comes first | What the settings show once it is allowed |
| --- | --- |
| ![The question that asks whether other devices may connect](https://iili.io/nyiVCLN.png) | ![The settings of the web interface with other devices allowed: the port, the code that pairs a phone and the certificate](https://iili.io/nyiVRpf.png) |

- vfox-plus listens on the **port** that the settings show, 8369 unless you change it. The
  firewall of this computer has to allow it.
- Beyond your own network, the interface is only reachable where something else publishes
  it: a reverse proxy, a tunnel, or the [gateway](gateway.md). Put that address — such as
  `https://vfox.example.com` — in **Public address** and the pairing links and the code use
  it instead of the address of this computer, so a phone that is not at home can pair.
  Without it the links name the address of this computer on the network, which only a phone
  on the same network reaches.
- Other devices connect over **HTTPS**. Without a certificate of your own, vfox-plus makes
  one for itself and signs it itself, and the settings show its SHA-256 fingerprint: compare
  it with the one that the browser or the Android app shows before you trust it. To use
  your own, choose *Certificate → Browse…*, then the certificate and its private key
  (PEM files).
- *Pair a phone* shows the **QR code** of a pairing link; nothing else is needed to pair.
  Scan it with the [Android app](android.md), or open the link in the browser of another
  computer. The link is under *Link and certificate*, with the fingerprint of the
  certificate for whoever wants to check them. A computer that is on more than one network,
  or that has a public address, offers each address above the code: choose the one that the
  phone reaches. The link works once and expires after five minutes.
- A paired device keeps its session for 24 hours. Turning *Other devices* off, or
  closing vfox-plus, ends the connections of all of them at once, and they have to pair
  again.

| The code and the link of *Pair a phone*, on the computer | The same, in a browser that is paired |
| --- | --- |
| ![The code that pairs a phone, with its link, in the window of vfox-plus](https://iili.io/nyiVG2e.png) | ![The code that pairs a phone, in the settings of the web interface](https://iili.io/nmiAZPI.png) |

The settings of the web interface show the same switches: from a paired browser or phone
you can allow other devices, change the port and turn the lightweight mode on or off, and
*Pair a phone* shows the code there, too, which is how a computer without a window
(lightweight mode) pairs a phone.

![The connections in the settings of a paired web interface, on a phone](https://iili.io/nmDNrvI.png)
