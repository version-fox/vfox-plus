# The gateway (Docker)

> [简体中文](../../zh-CN/user/gateway.md)

The Docker image is not vfox-plus in a container: it has no vfox, no window and no SDKs of
its own, and it takes no arguments. It is a *gateway*. Like the [Android app](android.md), it
connects to one vfox-plus that runs on a computer (with the web interface and *Other devices*
turned on, see [The web interface and other devices](remote.md)), and shows the interface of
that vfox-plus to the browsers that you pair with the gateway. Use it to put the interface at
an address of your choice, for example on a container host that your browsers reach and your
phone does not.

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

   A browser that is not paired sees the page of the gateway, which shows the fingerprint
   too:

   ![The page of the gateway for a browser that is not paired, with the fingerprint of its certificate](https://iili.io/niyLhua.png)

2. **Pair the gateway with vfox-plus.** The page of the gateway has a field for the pairing
   link of a vfox-plus. On the PC, choose *Settings → Web interface → Pair a phone → Show
   code → Copy link*, and paste the link there. The gateway then shows the fingerprint of the
   certificate of vfox-plus: compare it with the one in the settings of vfox-plus
   (*Certificate*), and choose *They are the same: trust and pair*. From then on the gateway
   trusts that certificate and no other, and asks again, with a warning, if the one at the
   address changes. A certificate that an authority signed needs no comparison.

   | The page of a gateway that is paired with nothing | The link of vfox-plus, pasted |
   | --- | --- |
   | ![The page of a gateway that is paired with no vfox-plus yet, with the field for its link](https://iili.io/niyLeGp.png) | ![The field with the pairing link of vfox-plus pasted in](https://iili.io/niyLbS9.png) |

   ![The page that asks to compare the fingerprint of the certificate of vfox-plus with the one in its settings](https://iili.io/niyQWf2.png)

3. **Use it.** The browser is taken to the interface of vfox-plus: the one a paired browser
   has, served by that vfox-plus itself, so the page always fits the version that runs. Pair
   more browsers with *Pair another browser* on the page of the gateway, and pair the gateway
   with another vfox-plus by pasting its link there (*Forget this vfox+* lets go of the one it
   has).

   | The interface of vfox-plus, through the gateway | The page of a gateway that is paired |
   | --- | --- |
   | ![The interface of vfox-plus in a browser, served through the gateway](https://iili.io/nbUaB2t.png) | ![The page of the gateway once it is paired: the address of vfox-plus and the certificate that it trusts](https://iili.io/niyZIee.png) |

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
