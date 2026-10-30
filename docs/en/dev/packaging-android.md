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
| `app/src/main/` | Kotlin sources (`dev.vfox.vfoxplus`), layouts, the Material 3 Expressive theme with the colours of the [design system](design-system.md#colour) (`values/colors.xml`, `values-night/colors.xml`), strings in English and Simplified Chinese, the Material Symbols as vector drawables (`drawable/ic_*.xml`, written by `vfxtool android-icons`; see the [design system](design-system.md#icons)), and the launcher icon (adaptive, with a monochrome layer for themed icons; see the [design system](design-system.md#application-icon)). |

## What the app does

- **Pairing** (`PairingFragment`) is native: the fox on the 9-sided cookie of the
  shape library, the command that the PC has to run (with a button that copies
  it), a button that scans the terminal QR code, and a field for the one-time
  link (its end icon pastes while the field is empty and clears it when it is
  not). After a first pairing the screen offers the last server first, with a
  button that reconnects. Connecting hides the keyboard, puts the loading
  indicator in the place of the button, and opens one session however often Go
  or Connect is pressed.
- **Scanning** (`ScanFragment`) is a CameraX preview with a zxing decoder: the
  decoded link fills the pairing field and connects at once. The picture is
  dimmed except for a square with 32 dp corners and corner brackets
  (`ViewfinderView`), which settles in on the default spatial spring, and a
  floating toolbar offers the torch, where the phone has one, and "Enter
  manually". `CAMERA` is requested as not required — without a camera, or when
  permission is denied, paste the link instead.
- **Session** (`ViewerFragment`) is a WebView of `/app` with the session
  cookie persisted across launches, file upload enabled (for plugin `.zip`
  and `.lua` files), pull-to-refresh, and a wavy progress indicator while a
  page loads. After a one-time link sets the cookie, the client moves to
  `/app` itself and starts the history of the page there, so that back does
  not load the used link again. The two things that only the app can do,
  pairing another link and disconnecting, are on the **This phone** card of
  the page's Settings: the app gives the page the object `window.vfoxAndroid`
  ([the bridge](remote-protocol.md#android-client)), and the page shows the
  card when it is there. A server that cannot be reached shows a native state
  (the error shape, "Try again", "Pair a new link") in place of the error page
  of the WebView. The web download of a failed task's full log is
  hidden inside the WebView, which cannot download files; read it in a
  browser instead. The window stays on the paired server: the address of
  another site that the page links to (the homepage of a plugin, its
  documentation) opens in the browser of the phone (`shouldOverrideUrlLoading`;
  the page clicks a link for them instead of calling `window.open`, which the
  WebView does not report).
- **TLS** (`onReceivedSslError`) shows the SHA-256 fingerprint dialog before
  trusting a self-signed certificate — the same fingerprint the PC terminal
  prints, in rows of eight bytes so that the two can be compared at a glance.
  Bringing your own certificate on the server avoids the warning.
- **Back** walks the WebView history before leaving the session, and the app
  opts in to the predictive back animation (`enableOnBackInvokedCallback`).
- **A new Activity** (the dark theme, another language, the phone turned) gets
  the page and its history back: the session screen saves the state of the
  WebView.

## The design system in the app

The app follows the [design system](design-system.md) as the other interfaces
do. This is where it lives in `android/`.

- **Edge to edge.** `MainActivity` calls `enableEdgeToEdge()`, and the app draws
  under the system bars. A screen keeps clear of them with `padToSafeArea`
  (`Insets.kt`: the bars, the display cutout and the keyboard become padding); a
  scroll view also brings the field that has the focus above the keyboard,
  because the system does not resize an edge-to-edge window. The page in the
  WebView gets the same insets as the CSS variables `--sai-*` ([the
  bridge](remote-protocol.md#android-client)).
- **Theme.** `Theme.VfoxPlus` extends `Theme.Material3Expressive.DayNight.NoActionBar`.
  `values/colors.xml` and `values-night/colors.xml` hold the 48 roles of the
  colour scheme of the seed `#F28C1B` (the theme maps 46 of them to attributes;
  shadow and scrim have none), with the same literal values as the `--m3-*`
  variables of the web interface. A test in `mygo/pair` (`webapp_design_test.go`)
  computes the scheme with `mygo/ui/mcu` and fails when the page or these files
  differ from it. Neither the theme nor a layout names a baseline
  `Widget.Material3.*` style: the Expressive styles
  (`Widget.Material3Expressive.*`) are the defaults of the theme, a layout only
  sets the size of a button (`materialSizeOverlay`), and a baseline style in a
  layout would hide the shape morph and the sizes.
- **Icons.** Material Symbols Rounded, from the same npm package as the other
  interfaces, as the vector drawables that `vfxtool android-icons` writes
  ([Icons](design-system.md#icons)).
- **Motion.** The springs are the `motionSpring*` attributes of the theme
  (`Motion.kt`). `springIn` brings the parts of a screen in, one after the
  other, on the default spatial spring (they rise) and the default effects
  spring (they fade in); the transitions between screens are shared axes on the
  slow effects spring, sampled into a curve over the time it needs to settle
  (`SpringInterpolator`). The root of each screen is a transition group
  (`android:transitionGroup`), so that the screen slides as one piece: otherwise
  each part slides by itself, and the outlined text field, which places its
  label from where its input is when it is laid out, keeps a label that was
  placed half way across. When the user removed the animations, everything
  jumps to its end.
- **Shapes.** The cookie and the soft burst of the shape library
  (`ExpressiveShapes`, drawn by `PolygonDrawable`) are built from the public API
  of `androidx.graphics:graphics-shapes`: `MaterialShapes` of Material 1.14.0 is
  restricted to the library group, and its drawable draws nothing.

## Why Views and not Compose

The screens are Views with Material Components 1.14.0, not Compose. Memory did not
decide it: the pairing screen written both ways takes about the same, within the
noise of the measuring ([Views or Compose in the Android
app](memory.md#views-or-compose-in-the-android-app)). These did:

- **Material 3 Expressive works in Views with the pinned toolchain.** Material
  Components 1.14.0 is a stable release with the Expressive theme, the buttons
  with their shape morph, the floating toolbar and the loading and wavy progress
  indicators, and it builds with AGP 8.13.2 and compileSdk 36. In Compose, as
  checked on 2026-10-30, the stable Material 3 1.4.0 keeps the Expressive theme,
  the motion schemes and the emphasized text styles `internal`, and its shape
  library cannot be used; Material 3 1.5.0-beta02 is a beta that needs AGP 9.1.0,
  compileSdk 37 and Compose 1.12.0, and the latest Compose, 1.12.1, needs the same
  AGP and compileSdk. Compose would mean a pre-release dependency and a new
  toolchain.
- **The APK is smaller.** Adding Compose makes it 1.2 MB (37 %) larger.
- **The native part is small.** The app is three screens around a WebView: what the
  user works in is the web interface, which no toolkit changes.

When Material 3 for Compose offers Expressive in a stable release that builds with
the pinned toolchain, measure again as [the memory
page](memory.md#views-or-compose-in-the-android-app) describes. The question will
then be the size of the APK and the cost of the migration, not memory.

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

The build needs no KVM; running the APK on an emulator does (see [The emulator
and KVM](environment.md#the-emulator-and-kvm)). The first run compiles
`GradleWrapperNeo.java` with the JDK and downloads Gradle and the dependencies,
which takes minutes. Where `services.gradle.org`
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
