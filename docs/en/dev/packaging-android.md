# Packaging Android

> [简体中文](../../zh-CN/dev/packaging-android.md)

This directory holds the Android remote client: a WebView around the pairing
server (`vfox-plus serve`), with a native Material 3 Expressive pairing flow.
It programs against [the remote protocol](remote-protocol.md), not against the
Go code. Users install the `.apk` and pair it with the PC; see
[Remote and Android](../user/remote.md).

## Layout

| Path | Content |
| --- | --- |
| `settings.gradle.kts`, `build.gradle.kts`, `gradle.properties` | Project setup; plugin versions are pinned here. |
| `versions.env` | Pinned tool versions CI reads (Gradle, Java, AGP, Kotlin, Material). |
| `app/build.gradle.kts` | Application ID `dev.vfox.vfoxplus`, `minSdk 26`, `targetSdk 36`; `versionName` is read from `mygo/VERSION`, so the phone shows the same version as the desktop. CameraX 1.6.2 and zxing core 3.5.3 (both Apache-2.0) scan the QR code. |
| `app/src/main/` | Kotlin sources (`dev.vfox.vfoxplus`), layouts, Material 3 Expressive theme, strings in English and Simplified Chinese, launcher icon. |

## What the app does

- **Pairing** (`PairingFragment`) is native: scan the terminal QR code, paste
  the one-time link the PC printed, or reconnect to the last server; compare
  the TLS fingerprint, and connect. Shared-axis transitions, wavy
  indeterminate progress while connecting.
- **Scanning** (`ScanFragment`) is a CameraX preview with a zxing decoder: the
  decoded link fills the pairing field and connects at once. `CAMERA` is
  requested as not required — without a camera, or when permission is denied,
  paste the link instead.
- **Session** (`ViewerFragment`) is a WebView of `/app` with the session
  cookie persisted across launches, file upload enabled (for plugin `.zip`
  and `.lua` files), pull-to-refresh, and an overflow menu (refresh, pair a
  new link, disconnect). After a one-time link sets the cookie, the client
  moves to `/app` itself. The web download of a failed task's full log is
  hidden inside the WebView, which cannot download files; read it in a
  browser instead.
- **TLS** (`onReceivedSslError`) shows the SHA-256 fingerprint dialog before
  trusting a self-signed certificate — the same fingerprint the PC terminal
  prints. Bringing your own certificate on the server avoids the warning.
- **Back** walks the WebView history before leaving the session.

## Building

JDK 17 and the Android SDK (platform 36, build-tools 36) are required. CI
installs them; locally point `ANDROID_HOME` at an SDK:

```sh
cd android
gradle :app:assembleRelease
```

The unsigned release APK lands at
`app/build/outputs/apk/release/app-release-unsigned.apk` (about 3.1 MB).
Rename it with the version for distribution:

```sh
cp app/build/outputs/apk/release/app-release-unsigned.apk \
  vfox-plus_$(tr -d '[:space:]' < ../mygo/VERSION).apk
```

Install it on the phone (allow installing unknown apps when asked) and pair
it as described in [Remote and Android](../user/remote.md).

## Signing status

The `.apk` is currently **unsigned** and installs with the unknown-sources
prompt. An officially signed bundle will be provided later. Signing changes
nothing in the build except the signing config; the unsigned APK is what CI
uploads as the `vfox-plus-android` artifact.

## Notes for maintainers

- `usesCleartextTraffic` is on: the server may be plain HTTP on a trusted
  local network (the PC side only speaks HTTPS on non-loopback binds, but the
  user may terminate TLS elsewhere). The session cookie is the credential
  either way.
- No Gradle wrapper jar is committed (the repository rejects binary files);
  CI installs Gradle `GRADLE_VERSION` from `versions.env`.
