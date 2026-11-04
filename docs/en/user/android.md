# The Android app

> [简体中文](../../zh-CN/user/android.md)

The Android app drives the vfox-plus on your PC from your phone: it shows the
same interface as the paired browser (see [The web interface and other
devices](remote.md)). Links to other sites, such as the homepage of a plugin or its
documentation, open in the browser of the phone.

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

The app opens on the pairing screen, which says where to find the code on the PC
(*Settings → Web interface → Pair a phone*): scan the QR code of the PC (Android
asks for the camera once) or paste the link. A self-signed certificate shows its
fingerprint first, to compare with the one on the PC.

Once paired, the app shows the interface of the paired browser: the home page
with the plugins in use, the SDKs with their versions, and the settings. The
first card of the settings, *This phone*, pairs another link or disconnects the
phone. The app follows the dark theme of the phone. When it cannot reach the PC
(vfox-plus stopped, other devices not allowed, another network, a firewall in the
way), it says so and offers to try again or to pair a new link.

| Home | An SDK | The settings, with the card of this phone | Home in the dark theme |
| --- | --- | --- | --- |
| ![The home page in the Android app](https://iili.io/nSx9JVf.png) | ![An SDK in the Android app: flutter, with its installed versions](https://iili.io/nSx9BR9.png) | ![The settings in the Android app, with the card of this phone](https://iili.io/nSx91WJ.png) | ![The home page in the Android app in the dark theme](https://iili.io/nSx94JS.png) |

The app needs your PC reachable from the phone (same Wi-Fi or another network
route) and the port of the web interface (8369 by default) allowed through the PC
firewall. Browsers can only set the global version; pinning versions to a
project stays in the desktop application.
