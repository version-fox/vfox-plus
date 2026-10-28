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
| `gradlew`, `gradlew.bat`, `gradlew.ps1`, `gradle/wrapper/` | The Gradle wrapper ([Gradle Wrapper Neo](https://github.com/Glavo/gradle-wrapper-neo)); `gradle-wrapper.properties` pins Gradle with its SHA-256. |
| `versions.env` | Pinned tool versions CI reads (Java, AGP, Kotlin, Material). |
| `app/build.gradle.kts` | Application ID `dev.vfox.vfoxplus`, `minSdk 26`, `targetSdk 36`; `versionName` is read from `mygo/VERSION`, so the phone shows the same version as the desktop. CameraX 1.6.2 and zxing core 3.5.3 (both Apache-2.0) scan the QR code. |
| `app/src/main/` | Kotlin sources (`dev.vfox.vfoxplus`), layouts, Material 3 Expressive theme, strings in English and Simplified Chinese, the launcher icon (adaptive, with a monochrome layer for themed icons; see the [design system](design-system.md#application-icon)). |

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

JDK 17 and the Android SDK (platform 36 and build-tools 35.0.0, the default of
the pinned AGP) are required. CI installs them; locally point `ANDROID_HOME` at
an SDK (or write `sdk.dir` into the git-ignored `local.properties`); Gradle
installs a missing package only after its license is accepted
(`sdkmanager --licenses`). The wrapper downloads the Gradle that
`gradle/wrapper/gradle-wrapper.properties` pins, so no Gradle install is needed:

```sh
cd android
./gradlew :app:assembleRelease
```

The first run compiles `GradleWrapperNeo.java` with the JDK and downloads
Gradle and the dependencies, which takes minutes. Where `services.gradle.org`
is slow, configure a download mirror for your user, not for the project; see
"Configure download mirrors" in the Neo README.

The release APK lands at `app/build/outputs/apk/release/app-release.apk`
(about 3.2 MB), signed with the debug key (see [Signing
status](#signing-status)). Rename it with the version for distribution:

```sh
cp app/build/outputs/apk/release/app-release.apk \
  vfox-plus_$(tr -d '[:space:]' < ../mygo/VERSION).apk
```

Install it on the phone (allow installing unknown apps when asked) and pair
it as described in [Remote and Android](../user/remote.md).

## Signing status

Android installs no APK that has no signature (an unsigned one fails with
`INSTALL_PARSE_FAILED_NO_CERTIFICATES`), so the release build is signed with
the **debug key**, which Gradle creates on the first build in
`~/.android/debug.keystore` (or `$ANDROID_USER_HOME`). Every build machine has
its own key, and CI discards it with the runner: an APK from another build
cannot update an installed one (uninstall it first; the pairing is forgotten),
and the signature does not say who built it. An officially signed APK will be
provided later; it changes only the `signingConfig` of the release build in
`app/build.gradle.kts`.

## Notes for maintainers

- `usesCleartextTraffic` is on: the server may be plain HTTP on a trusted
  local network (the PC side only speaks HTTPS on non-loopback binds, but the
  user may terminate TLS elsewhere). The session cookie is the credential
  either way.
- The wrapper is [Gradle Wrapper Neo](https://github.com/Glavo/gradle-wrapper-neo)
  0.2.0 (Apache-2.0): the stock `gradle-wrapper.jar` is a binary, which the
  repository rejects, and Neo does the same job from one Java source file that
  `gradlew` compiles on first use. `gradlew`, `gradlew.bat`, `gradlew.ps1` and
  `GradleWrapperNeo.java` stay as the release publishes them (so
  `.licenserc.yaml` skips their headers); to update, download those four files
  from the new release as its README shows. `.gitattributes` keeps `*.bat` as
  CRLF, which `gradlew.bat` needs on Windows.
- To change Gradle, edit `distributionUrl` and `distributionSha256Sum` in
  `gradle/wrapper/gradle-wrapper.properties` (checksums:
  <https://gradle.org/release-checksums/>). It is 9.5.1 because AGP 8.13.2
  fails on Gradle 9.8.0; raise AGP and Gradle together.
