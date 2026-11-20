# Packaging the phone app

> [简体中文](../../zh-CN/dev/packaging-android.md)

The phone app is [`mygo/phone`](../../../mygo/phone): a native application of
mygo for Android. Its screens are the native user interface of mygo, not Android
Views and not a WebView. It pairs with a computer that runs vfox-plus, and then
shows the SDKs, plugins and tasks of that computer, over
[the remote protocol](remote-protocol.md). Users install the APK and pair it as
described in [Install](../user/install.md#android) and
[The Android app](../user/android.md).

mygo builds the app: `mygo build -platform android` writes the Android project
from `mygo/phone`, runs Gradle on it, and writes the APK and the AAB to
`mygo/phone/dist/android/`. The repository holds no Android project of its own.

## Layout

| Path | Content |
| --- | --- |
| `main.go` | The entry point. On Linux, `go run ./phone` (from `mygo/`) opens the app in a window of a phone's size, for development. |
| `app/` | The model of the app: the pages, the tabs and the backend that the screens show. |
| `screens/` | The screens, drawn by mygo: the pairing screen and the five tabs, Home, SDKs, Plugins, Tasks and Settings. |
| `pairing/` | The pairing link, the fingerprint of the certificate of the computer, and the reading of a QR code that carries a link, from a photo. |
| `remote/` | The client of [the remote protocol](remote-protocol.md). |
| `store/` | What the app keeps between launches. |
| `i18n/` | The texts, in English and Simplified Chinese. |
| `resources/` | The sources of the icons. `vfxtool appicon` writes them before a build ([Icons](#icons)). |
| `mygo.json` | The application: its name, the identifier `dev.vfox.vfoxplus`, the slug, the version, the URL scheme `vfoxplus`, and the adaptive icon of Android. |
| `versions.env` | The pinned versions ([Pinned versions](#pinned-versions)). |

## Pairing

A pairing link starts the pairing. The address `vfoxplus://pair?link=<link>`, with
the scheme of `mygo.json`, opens the pairing screen with the link filled in. The
person checks the fingerprint of the certificate and taps *Trust and pair* (信任并配对
in Chinese). That is the one button that trusts the certificate and pairs the phone.
When the pairing succeeds, the app logs one line: `vfoxplus: paired host=<host>
fingerprint=<fingerprint>`.

`vfxtool measure-apk` drives this path, and it looks for that line and for the labels
of the button and of the tabs ([Notes for maintainers](#notes-for-maintainers)).

## Building

The build needs the tools that [`mygo/phone/versions.env`](../../../mygo/phone/versions.env)
pins ([Pinned versions](#pinned-versions)): a JDK 17, the Android SDK with platform 36
and build-tools 36.0.0, the NDK 28.2.13676358 (its clang compiles the Go code of the app
for Android), and the mygo CLI. [Android](environment.md#android) says how to install
them, and [The dev container](environment.md#the-dev-container) has all of them. Gradle,
which mygo runs on the Android project, downloads its distribution and the libraries on
the first build, which takes minutes. They stay in its user home: `~/.gradle`, or
`/home/vscode/.gradle` in the container.

From the root of the repository, with `vfxtool` built as [The sandbox](environment.md#the-sandbox)
says:

```sh
vfxtool appicon mygo/phone/resources
cd mygo/phone
mygo build -platform android
```

`mygo build -platform android` makes the APK and the AAB that the release ships.
`-platform android/amd64` makes the same pair for x86_64 only, which is the one that the
emulator runs; CI builds that one. Write the icons first, from the root of the repository:
the path of `vfxtool appicon` is relative to it, and from `mygo/phone` the icons would land
under `mygo/phone/mygo/phone`.

The configuration names no keystore, so `mygo build` signs the APK with the debug key of
the machine, which Gradle creates at the first build. Such an APK is for testing: the
release replaces its signature ([Signing status](#signing-status)). The output is
git-ignored, in `mygo/phone/dist/`. To install the APK on an emulator or a phone, use
`adb install -r`, as [Run the app](environment.md#run-the-app) describes.

On Linux, `go run ./phone` (from `mygo/`) opens the same app in a window of a phone's
size, which is quicker than an emulator for the screens.

## Pinned versions

| Variable | Pins |
| --- | --- |
| `MYGO_VERSION` | The mygo CLI, installed with `go install github.com/egoist/mygo/cmd/mygo@v<version>`. It is the version of the `mygo` module in `mygo/go.mod`, which must match. |
| `JAVA_VERSION` | The JDK, 17: Gradle 8.14.3, which mygo runs, does not run on JDK 25. CI uses Temurin 17, and the dev container the OpenJDK 17 of the Ubuntu archive. |
| `ANDROID_CMDLINE_TOOLS_BUILD`, `ANDROID_CMDLINE_TOOLS_SHA256` | The command-line tools of the SDK that the dev container installs: the build in the name of the zip that dl.google.com publishes, and its checksum. |
| `ANDROID_SDK_PLATFORM`, `ANDROID_SDK_BUILD_TOOLS` | The platform `android-36` and the build tools 36.0.0, which hold `aapt2` and `apksigner`. |
| `ANDROID_GRADLE_BUILD_TOOLS` | The build tools 35.0.0, which the Android Gradle plugin asks for besides the ones above, and which it would download in the middle of the build: that fails where the SDK is not writable, as in the dev container. |
| `ANDROID_NDK_VERSION` | The NDK, 28.2.13676358. |
| `ANDROID_EMULATOR_API`, `ANDROID_SYSTEM_IMAGE_TARGET` | The emulator of [CI](#continuous-integration): API 36 with `google_apis`, for x86_64. |

CI, the dev container and its check ([`devcontainer.yml`](../../../.github/workflows/devcontainer.yml))
all read this file, so a change reaches every one of them.

## Icons

The repository holds no binary files. `vfxtool appicon mygo/phone/resources` draws
`icon.png` and `icon-foreground.png` from the paths of the application icon
([Application icon](design-system.md#application-icon)), and git ignores both. The
background of the adaptive icon is `#1B1268`, which `mygo.json` names. CI and the
[dev container](environment.md#the-dev-container) write the icons before they build.

## Signing status

Android installs an update only over an app that is signed with the same key, and it
refuses another one (`INSTALL_FAILED_UPDATE_INCOMPATIBLE`). So every release of
vfox-plus is signed with the **release key**: one key, kept as secrets of the `release`
environment of the repository, the same for every release. A release installs over the
one before it, and the data and the pairing with the computer stay.

The APK is signed in two steps, in [`release.yml`](../../../.github/workflows/release.yml):

- The `build` job makes the APK with mygo, with the debug key of its runner, and uploads
  it as `vfox-plus-android-unsigned`. That job holds no secrets, so the release key never
  reaches it.
- The `sign-windows` job, after the approval, signs the APK with
  [`shared/scripts/sign-apk.ps1`](../../../shared/scripts/sign-apk.ps1). `zipalign -f -P 16 4`
  aligns it again, with 16 KB pages for its native libraries, and `apksigner sign` replaces
  the debug signature with the release key. The script checks the signature, that no
  `CN=Android Debug` certificate is left, and with `zipalign -c` that the alignment is still
  there after the signing. The job writes the APK to `dist-android/vfox-plus_<version>.apk`.
  `vfxtool check-assets` requires exactly one such file before anything is published.

**The rehearsal.** The release cannot be tried before it is merged: it is started by hand
from `main`, and its key is in an environment that only `main` and the tags can use. So the
`signing` job of [`phone.yml`](../../../.github/workflows/phone.yml) runs the same script on
a Windows runner, as the release does, whenever the phone app or the script changes. The key
is made by the job (a PKCS12 keystore, with a password that nothing else knows), and the APK
is the one that the `build` job makes with the debug key. The job checks that the APK of the
build is signed with the debug key of mygo, signs it, and checks that the result has the
certificate of the rehearsal and no debug one. What the rehearsal cannot reach is what is
the release's alone: where the key comes from (the four secrets below), the steps of Azure
that sign the Windows packages, the approval, and the version. A dry run of the Release
workflow on `main` tries them: Actions, Release, Run workflow, with `dry-run` (it builds and
signs everything, and pushes nothing). Run one after a change of the signing, and once
after the environment is set up.

The job refuses to run without the four secrets: `ANDROID_RELEASE_KEYSTORE_BASE64` (the
keystore, in base64), `ANDROID_RELEASE_STORE_PASSWORD`, `ANDROID_RELEASE_KEY_ALIAS` and
`ANDROID_RELEASE_KEY_PASSWORD`. They are in the `release` environment and nowhere else in
the repository, and only `sign-windows` reads them. The keystore goes to a temporary file
of the runner, which is removed whatever the outcome.

A run waits for a member of `@version-fox/vfox-maintainers` or `@version-fox/vfox-committers`
to approve it once, before `sign-windows` signs the Windows packages and the APK; the
publishing then follows without a second approval. An administrator of the repository may
start a run without the approval. Only a run from `main` or from a tag `v*` can use the
environment. GitHub does not show the values of the secrets, and the maintainers do not
share them. If you need the actual values of these four secrets, contact
<https://github.com/linghengqian>.

To set the release key up, once. Create the `release` environment in the repository
settings (Environments): the required reviewers are the teams `vfox-maintainers` and
`vfox-committers`, "Allow administrators to bypass configured protection rules" is on, and
the deployment branches and tags are `main` and `v*`. Keep the keystore and its passwords in
a password manager, and a copy somewhere else: without them no later release can update an
installed app.

```sh
keytool -genkeypair -v -keystore vfox-plus-release.jks -alias vfox-plus \
  -keyalg RSA -keysize 4096 -validity 10000
base64 -w0 vfox-plus-release.jks | gh secret set ANDROID_RELEASE_KEYSTORE_BASE64 --env release -R version-fox/vfox-plus
gh secret set ANDROID_RELEASE_STORE_PASSWORD --env release -R version-fox/vfox-plus
gh secret set ANDROID_RELEASE_KEY_ALIAS --env release -R version-fox/vfox-plus
gh secret set ANDROID_RELEASE_KEY_PASSWORD --env release -R version-fox/vfox-plus
```

Keep the `.jks` file out of the repository: `*.jks` and `*.keystore` are ignored.

## Versions and updates

Android accepts an update only from the same signing key, and only with a higher
`versionCode`. The release sets the version in the files that carry it, `mygo/VERSION` and
the `version` of `mygo/phone/mygo.json` among them (`vfxtool set-version`), so the phone
shows the version of the desktop application. The `versionCode` is mygo's default,
`major × 10000 + minor × 100 + patch`: 1.2.3 is 10203. The minor and the patch numbers must
therefore stay below 100, or the codes stop following the versions.

An APK that the debug key signs, such as the one of CI or of `mygo build` on your machine,
is not a release. It does not update to a release, and a release does not update it. Uninstall
it first, which also forgets the pairing with the computer.

## Continuous integration

[`phone.yml`](../../../.github/workflows/phone.yml) builds the app and drives it on an
emulator. It runs for changes to the app, to the code that it pairs with, to the shared
tooling and to the pinned versions:

- The `build` job makes the APK and the AAB for x86_64, with the debug key, and uploads
  them as `vfox-plus-android-amd64`.
- The `emulator` job starts the end-to-end build of the desktop application with
  `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2`, the address at which the emulator reaches the runner,
  in a virtual display, against the fake vfox: the tabs hold what a person's do, SDKs that are
  installed and plugins to offer. It starts an emulator (API 36, `google_apis`, x86_64, with a software
  GPU), then runs `vfxtool phone-link`, which pairs with the desktop application and writes
  a pairing link for `10.0.2.2`, and `vfxtool measure-apk`. The tool installs the APK, starts
  it cold five times and measures it after each start (the line is the median, see
  [the phone app](memory.md#the-phone-app)), presses the key back with the dialog of the pairing open (it
  must close the dialog and leave the app in front, `--check-back-key`), then pairs it through the deep link, visits the five tabs with a
  screenshot of each, takes the pictures of the [guide to the phone app](../user/android.md) on the way
  (`--pictures`, see [Screenshots](contributing.md#screenshots)), and fails when the log of the app has an
  error. When the dialog of the
  pairing says that the computer did not answer, which an emulator whose network is not up yet
  can cause, the tool closes it and opens the link again, up to three times, and says so in its
  output. A run that fails keeps the screen, the log of the app and what the device says of its
  network. The cold measurement is the artifact `memory-android`. The screenshots, the log of
  the app, the evidence of a failure, the measurement of the paired session and the five readings
  of the cold starts, with the whole `dumpsys meminfo` report of each, are the artifact
  `phone-e2e-android`, and the pictures of the guide are the artifact `phone-pictures`.
- The `signing` job rehearses the signing of the release on a Windows runner, with the script
  of the release and a key that it makes ([Signing status](#signing-status)). It runs when
  `shared/scripts/sign-apk.ps1` changes too.

[`release.yml`](../../../.github/workflows/release.yml) makes the APK of the release and
signs it ([Signing status](#signing-status)). [`memory.yml`](../../../.github/workflows/memory.yml)
reports the memory of a pull request when `mygo` or `phone` ends
([Memory in pull requests](memory.md#memory-in-pull-requests)).

## Notes for maintainers

- `vfxtool measure-apk` looks for what the app shows and logs, in English and in Simplified
  Chinese: the label of the confirmation, the message that the computer did not answer and
  the label of the button that closes it, the labels of the five tabs, and the line that the
  pairing logs (see `shared/tooling/internal/measureapk/measureapk.go`). The run that takes the
  pictures looks for the row of the SDK that it opens (`nodejs`) and for the buttons of the dialog
  that adds a plugin, *Add from source* and *Cancel* (`pictures.go`, beside it). Change them in the
  app and in those files in one commit, or the measurement fails.
- `mygo/phone/versions.env` and `mygo/go.mod` pin the same mygo version. `JAVA_VERSION` follows
  the Gradle that mygo runs for the project: raise it only together with a mygo that runs a
  Gradle which accepts the newer JDK.
- Do not edit the `version` of `mygo.json` by hand: `vfxtool set-version` writes it together with
  `mygo/VERSION`.
