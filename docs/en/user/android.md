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

| The pairing screen | Scanning the QR code | The fingerprint of a self-signed certificate |
| --- | --- | --- |
| ![The pairing screen of the Android app, which says where to find the code on the PC](https://iili.io/nDQhyZP.png) | ![The scanner of the Android app, with the viewfinder and the toolbar](https://iili.io/nDQjIPn.png) | ![The dialog that shows the fingerprint of a self-signed certificate](https://iili.io/nDQjwNV.png) |

Once paired, the app shows the interface of the paired browser: the home page
with the plugins in use, the SDKs with their versions, and the settings. The
first card of the settings, *This phone*, pairs another link or disconnects the
phone; the connections below it are the ones of the window of vfox-plus on the PC
(see [Other devices](remote.md#other-devices)).

| Home | An SDK | The settings, with the card of this phone and the connections |
| --- | --- | --- |
| ![The home page in the Android app](https://iili.io/nbVCV72.png) | ![An SDK in the Android app: flutter, with its installed versions](https://iili.io/nbVCeqb.png) | ![The settings in the Android app: the card of this phone, and the connections of the PC](https://iili.io/nbXbk8u.png) |

The app follows the dark theme of the phone. When it cannot reach the PC (vfox-plus
stopped, other devices not allowed, another network, a firewall in the way), it says
so and offers to try again or to pair a new link.

| The pairing screen in the dark theme | Home in the dark theme | A PC that cannot be reached |
| --- | --- | --- |
| ![The pairing screen of the Android app in the dark theme](https://iili.io/nDQwo5x.png) | ![The home page in the Android app in the dark theme](https://iili.io/nbVCD2R.png) | ![The Android app when the PC cannot be reached](https://iili.io/nDQwkml.png) |

The app needs your PC reachable from the phone (same Wi-Fi or another network
route) and the port of the web interface (8369 by default) allowed through the PC
firewall. Browsers can only set the global version; pinning versions to a
project stays in the desktop application.
