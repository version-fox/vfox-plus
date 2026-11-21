# Other devices and the lightweight mode

> [简体中文](../../zh-CN/user/remote.md)

vfox-plus is a desktop application, and it has no command line: you start it like any
other application, and what else it does is a matter of its settings. This page is about
three of them, in *Settings → Other devices*: whether other devices may connect, the
lightweight mode, and how they reach vfox-plus.

vfox-plus shows no web interface on this computer. Other devices see it through the
[gateway](gateway.md), a Docker image that serves the interface to the browsers that pair
with it, and a phone uses the [phone app](android.md).

## Other devices

Turn on *Other devices* and confirm. A gateway or a phone can then pair with vfox-plus.
It is off by default: it opens vfox-plus to the network, so allow it only in a network
that you trust.

| The question that comes first | What the settings show once it is allowed |
| --- | --- |
| ![The question that asks whether other devices may connect](https://iili.io/oFDBQQp.png) | ![The settings with other devices allowed: the port, the code that pairs a phone and the certificate](https://iili.io/oFDCMtR.png) |

- vfox-plus listens on the **port** that the settings show, 8369 unless you change it. The
  firewall of this computer has to allow it.
- Beyond your own network, vfox-plus is only reachable where something else publishes it: a
  reverse proxy, a tunnel, or the [gateway](gateway.md). Put that address — such as
  `https://vfox.example.com` — in **Public address** and the pairing links and the code use
  it instead of the address of this computer, so a phone that is not at home can pair.
  Without it the links name the address of this computer on the network, which only a phone
  on the same network reaches.
- Other devices connect over **HTTPS**. Without a certificate of your own, vfox-plus makes
  one for itself and signs it itself, and the settings show its SHA-256 fingerprint: compare
  it with the one that the browser, the gateway or the phone app shows before you trust it.
  To use your own, choose *Certificate → Browse…*, then the certificate and its private key
  (PEM files).
- *Pair a phone* shows the **QR code** of a pairing link. Read it with the
  [phone app](android.md) (a photo of the code, or a picture of it), or paste the link into
  the [phone app](android.md) or the [gateway](gateway.md). The link is shown under the
  code, and *Copy link* copies it; the fingerprint of the certificate, for whoever wants to
  check it, is in the *Certificate* row of the settings. A computer that is on more than one
  network, or that has a public address, offers each address under the code: choose the one
  that the phone reaches. The link works once and expires after five minutes.
- A paired device keeps its session for 24 hours. Turning *Other devices* off, or closing
  vfox-plus, ends the connections of all of them at once, and they have to pair again.

| The code and the link of *Pair a phone*, on the computer | The same code, in the settings of the interface that the gateway shows |
| --- | --- |
| ![The code that pairs a phone, with its link, in the window of vfox-plus](https://iili.io/oFDnBxR.png) | ![The code that pairs a phone, in the settings of the interface](https://iili.io/oFDo8Cl.png) |

The settings of the phone app and of the interface that the gateway shows have the same
switches: from a paired device you can allow
other devices, change the port and turn the lightweight mode on or off, and *Pair a phone*
shows the code there, too, which is how a computer without a window (lightweight mode)
pairs a phone.

![The connections in the settings of a paired interface, on a phone](https://iili.io/nmDNrvI.png)

## Publishing to the internet

At home, other devices reach vfox-plus at the address of this computer on the network,
which is enough for a phone on the same Wi-Fi. From anywhere else it needs an address on
the internet, and vfox-plus does not run a relay of its own: the address comes from
something that publishes it.

- **A tunnel of your own.** Put the address that a reverse proxy, a tunnel or the
  [gateway](gateway.md) publishes in **Public address**; pairing links and the code use it.
- **The built-in switch.** *Public tunnel* publishes vfox-plus through a free
  [Cloudflare quick tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/trycloudflare/):
  no account, no configuration, one switch. When the program is not installed yet, turning
  it on downloads it (about 50 MB) from Cloudflare's GitHub release and checks its SHA-256.
  The address is random, such as
  `https://one-two-three.trycloudflare.com`, and changes at every start. It ends when you
  turn the switch off or quit vfox+.

The tunnel carries the traffic of the gateways, phones and other devices that connect
through it past Cloudflare; quick tunnels are meant for short-lived use, without a promise
of uptime. What protects vfox-plus is the pairing: a device still needs a one-time link, and
the session lasts 24 hours.

## Lightweight mode

Closing the window quits vfox-plus, and the connections of other devices with it. Turn on
*Lightweight mode* (it needs other devices to be allowed) and closing the window leaves
vfox-plus running without it. The gateway and the phones that use it stay connected, and
the tasks that run go on.

- **Bring the window back** from the icon that vfox-plus puts in the tray while it has no
  window: choose *Open vfox+* in its menu. Starting vfox-plus again, from the launcher as
  always, does the same, and works without a tray: no second vfox-plus starts, the one that
  runs opens its window and the new one ends.
- **End it for good** with *Quit vfox+*, which the menu of the icon has, and which the
  settings show while the lightweight mode is on (*Settings → Other devices → Quit vfox+*).
  Quitting cancels the tasks that run. Ending the process from the system quits it, too.

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
  `gnome-shell-extension-appindicator`. Click the icon for the menu.

Other desktops are not supported. KDE Plasma and the desktops whose tray is of the old kind get
no icon that vfox-plus promises: use GNOME. Where there is no icon, nothing else changes:
vfox-plus runs without its window as before, and you bring the window back by starting
vfox-plus again.
