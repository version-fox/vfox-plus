# Contributing to vfox-plus

> [简体中文](../../zh-CN/dev/contributing.md)

Thank you for helping! This repository holds the vfox-plus application (see
the [user guide](../user/index.md)); most of what follows applies to all of its parts.

## Ground rules

- A commit names the people who wrote and reviewed it. Do not add
  `Co-authored-by` trailers for AI agents or tools; the repository workflow
  rejects them.
- Code comments, commit messages and pull requests are written in
  **English**. User-visible texts go through the translation catalogues in
  [`shared/i18n`](../../../shared/i18n) and are available in English and Simplified
  Chinese.
- Documentation is written in **English with a Simplified Chinese translation**:
  every Markdown document has a Chinese counterpart (see
  [Documentation languages](#documentation-languages)).
- Every source file starts with the Apache-2.0 license header used in this
  repository. Check with
  `docker run --rm -v "$PWD:/github/workspace" apache/skywalking-eyes:0.9.0 header check`
  and add missing headers with `header fix`.
- Runtime dependencies must be licensed under terms compatible with the Apache
  License 2.0.
- Behaviour shared by the layers — the vfox CLI contract, screens,
  design tokens, translations, icons, settings — is specified in this `dev/`
  directory and in [`shared/`](../../../shared/). Change the specification together with the code.
- Files use UTF-8, LF line endings and a final newline (see
  [`.editorconfig`](../../../.editorconfig) and [`.gitattributes`](../../../.gitattributes)).
- Binary files are not committed. Upload screenshots and other images to an
  image host and link them (see [Screenshots](#screenshots)); generate binary
  build inputs, such as the logos of the Windows package, at build time.
  CI rejects binary files.

## Workflow

To set up a machine (Ubuntu 26.04, on its own or in WSL2), or to work in the dev
container, see [Development environment](environment.md).

1. Open an issue or comment on an existing one before large changes.
2. Create a branch from `main` and keep pull requests focused.
3. Run the checks; they are the same as in CI and listed in the
   [mygo README](implementation-mygo.md#everyday-commands):

   ```sh
   cd mygo
   golangci-lint fmt --diff
   golangci-lint run && golangci-lint run --build-tags e2e
   CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./... && vfxtool coverage coverage.out 85
   ```

   To run every check, the end-to-end tests and the packaging without installing
   anything but Docker, and without any risk for the vfox installation on your
   machine, use the sandbox, `vfxtool sandbox all` (see
   [shared/sandbox](sandbox.md)).

4. Describe what changed and how you tested it in the pull request. Include
   screenshots for visual changes, in light and dark themes.

## Continuous integration

| Workflow | Runs on | Checks |
| --- | --- | --- |
| [`repository.yml`](../../../.github/workflows/repository.yml) | every change | License headers (SkyWalking Eyes), workflows (actionlint), vfxtool (`gofmt`, `go vet`, `go test`), documentation languages and links (`vfxtool docs-check`), Markdown (markdownlint), shared JSON, no committed binary files |
| [`memory.yml`](../../../.github/workflows/memory.yml) | the end of the build workflow, on pull requests | One comment on the pull request with the memory of the application, and the difference from `main` (see [Memory in pull requests](memory.md#memory-in-pull-requests)) |
| [`mygo.yml`](../../../.github/workflows/mygo.yml) | `mygo/`, `shared/` | Formatting and lint (golangci-lint, with and without the end-to-end build tag), unit tests with the race detector and the coverage gates, the licenses of the Go modules, end-to-end scenarios that drive the built application against captured output and against the real vfox, the Docker image of the gateway built and run against the application, the memory measurement, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container, unsigned `.msix` and `.msixbundle` for Windows 11 (x64 and arm64) validated with `vfxtool validate-msix` on Linux, and validated plus installed/uninstalled on Windows Server 2025 (x64) and Windows 11 on ARM (arm64) |
| [`sandboxes.yml`](../../../.github/workflows/sandboxes.yml) | `mygo/sandbox/`, `mygo/versions.env`, `shared/sandbox/`, `shared/tooling/` | Builds the Docker sandbox and runs `vfxtool sandbox all` in it |
| [`android.yml`](../../../.github/workflows/android.yml) | `android/`, `mygo/pair/`, `mygo/VERSION` | Builds the release APK (signed with the debug key) and uploads it as `vfox-plus-android` |
| [`devcontainer.yml`](../../../.github/workflows/devcontainer.yml) | `.devcontainer/`, `mygo/versions.env`, `android/versions.env` | Builds the [dev container](environment.md#the-dev-container), starts it the way an editor does and uses it: the tools against the pinned versions, lint, the end-to-end scenarios in the virtual display, the Android build, and the checks that use Docker |
| [`release.yml`](../../../.github/workflows/release.yml) | by hand or a `v*` tag | Tags `v<version>`, packages and publishes the GitHub release that the self-update check reads |

Dependabot keeps the Go modules and the GitHub Actions up to date; actions are
pinned to commit SHAs.

The workflows install their Ubuntu packages with
[`install-linux-packages`](../../../.github/actions/install-linux-packages/action.yml), which
keeps the downloaded archives in a cache (one for each list of packages and each
week) and downloads what the cache lacks eight files at a time, continuing what was cut
and trying the next mirror when one is slow, in 15 minutes at most. The Ubuntu mirror of
the runners is slow for some of them at times: on 2026-10-23 a job got 40 KB/s for a
connection while others downloaded 90 MB in seconds. apt does not check an archive that is in the
directory, so the action checks every archive by its hash. A run that cannot finish
saves what it downloaded, so that the next one gets further.

A pull request that rebuilds the application gets a comment with its memory and
the difference from `main`, which the next commit replaces instead of adding
another one. A change that is more than the noise between CI runs is in bold; read
[Memory in pull requests](memory.md#memory-in-pull-requests) before you trust a small
one.

## Release

Like the releases of
[version-fox/vfox-java](https://github.com/version-fox/vfox-java), a release
is started by hand:

1. Set the version in a pull request: `mygo/VERSION` and the `Version`
   constant in `mygo/ui/app.go` (a unit test keeps them in sync).
2. Actions, Release, Run workflow, and type the version without the `v`
   (`0.2.0`). Pushing a `v*` tag runs the same job.
3. The job refuses anything but `major.minor.patch`, anything `mygo/VERSION`
   does not say, and an existing tag. It tags `v<version>`, packages what
   [`mygo.yml`](#continuous-integration) packages plus the classic snap from
   `snap/snapcraft.yaml`, and publishes the GitHub release with
   `--generate-notes`, whose assets are what
   [the self-update check](architecture.md#self-updates) reads.

## Documentation languages

Every Markdown document lives under `docs/`, first by language, then by
audience, with English canonical and Simplified Chinese translated in the same
pull request. The Chinese translation follows the English section by section,
keeping code blocks, command lines, file paths and the glossary of
[i18n.md](i18n.md) unchanged.

| English (canonical) | Simplified Chinese |
| --- | --- |
| `docs/en/user/*.md` | `docs/zh-CN/user/*.md` (same base name) |
| `docs/en/dev/*.md` | `docs/zh-CN/dev/*.md` (same base name) |

User guides (`user/`) are for people who run vfox-plus: installing, using the
desktop application, the browser and the Android client. They never mention
building from source, the repository layout, the vfox CLI contract or any
other developer topic. Developer specification (`dev/`) is for contributors:
architecture, contracts, UI and design specifications, packaging and tooling.

The repository root and the source directories keep only thin entry points
(`README.md`, `mygo/README.md`, `shared/*/README.md`) that link into `docs/`;
the substance lives in `docs/`.

Each document links to its counterpart at the top (`[简体中文](…)` in English,
`[English](…)` in Chinese) and `vfxtool docs-check` verifies the mapping; the
repository workflow fails when a document has no counterpart. Future documents
follow the same table: add both files together.

## Markdown

CI checks every document with [markdownlint](https://github.com/DavidAnson/markdownlint),
configured in [`.markdownlint.yaml`](../../../.markdownlint.yaml). Run the check
before you push, with the image that CI uses (pinned by its digest):

```sh
docker run --rm --volume "$PWD:/workdir" \
  ghcr.io/igorshubovych/markdownlint-cli:v0.50.0@sha256:db7dcb73cd3163cc45854b60fc21eefe1ebe7fa52aad758706010b64a4778900 \
  --ignore-path .gitignore "**/*.md"
```

Two habits matter most:

- **Write a URL in angle brackets**, `<https://vfox.dev/>`, or as the target of a
  link, `[vfox](https://vfox.dev/)`. GitHub turns a bare URL into a link up to
  the next space, so the `)` after it, and in Chinese text the full-width `）`, `，`
  and `。`, become part of the address and the preview shows a wrong URL.
- **Close every link with `]`**: `[text](target)`. A `[text(target)` is no link and
  shows as it is; `vfxtool docs-check` fails on it.

A `|` ends the cell of a table even inside code; write `\|` there.

## Common changes

### Texts

Add or change messages in [`shared/i18n/app_en.arb`](../../../shared/i18n/app_en.arb)
with a `description` (and placeholder metadata), then translate them in
[`app_zh.arb`](../../../shared/i18n/app_zh.arb). See [docs/i18n.md](i18n.md),
including the glossary.

### Icons

Add the Material Symbol to [`shared/icons.json`](../../../shared/icons.json) under a
name that describes its meaning, with the name of the symbol as at
<https://fonts.google.com/icons> (`history`, `folder_open`). Nothing else is to be
added: `vfxtool prepare` cuts the SVGs out of the pinned npm package when it builds,
and fails, naming the icons, for a symbol that the package does not have.

The Android app does not take its icons from the package: its build imports the
Android XML that google/material-design-icons publishes ([Icons](design-system.md#icons)
says how). To add one, add a line to the list `materialSymbols` in
`android/app/build.gradle.kts`: the name of the drawable, the symbol, and `"fill1"` if
the filled variant is wanted. To move to a newer commit of that repository, change
`materialSymbolsCommit` there. Nothing is copied or committed.

To move to a newer version of the package, change `version` and `sha512` in the
`source` of that file. The registry says the checksum in base64; the file wants it
in hexadecimal:

```sh
version=1.2.98
curl -fsS "https://registry.npmjs.org/@iconify-json%2fmaterial-symbols/$version" |
  grep -o '"integrity":"sha512-[^"]*"' | cut -d- -f2 | tr -d '"' | base64 -d | od -An -v -tx1 | tr -d ' \n'
```

### Web interface

The web interface is TypeScript built with Vite, in [`mygo/web`](../../../mygo/web).
Its build bundles the page into the single `mygo/pair/webapp.html` that the Go
server embeds: edit the sources, run `npm run build` there, and commit the page
with them. `npm test` runs the unit tests of the page; `npm run dev` serves it
with hot reload against a running application (see
[Node and the web interface](environment.md#node-and-the-web-interface)).
The Go tests in `mygo/pair` still guard the contract of the built page: the
design tokens and the springs, which fail when the theme moves without them,
and the favicon placeholders. Copy of the page lives in its i18n catalogs
(`mygo/web/src/i18n-*`), typed per language, not in `shared/i18n`.

### vfox output

When vfox changes, recapture the fixtures with
`vfxtool capture` and follow
[`shared/fixtures/README.md`](fixtures.md). Document the change in
[docs/vfox-cli-contract.md](vfox-cli-contract.md).

### Diagrams

Draw a diagram in [Mermaid](https://mermaid.js.org/), in a fenced `mermaid`
block that GitHub renders; do not draw it as text art and do not commit an
image. The diagrams of the [architecture](architecture.md),
the [fixtures](fixtures.md) and the [remote protocol](remote-protocol.md#tasks)
set the style: copy the front matter of one of them, which carries the theme, so
that every diagram looks alike in the light and the dark theme of GitHub, and give
it an `accTitle` and an `accDescr` for screen readers.

Colours stand for roles: indigo (`#E4E0FF`) the user interface and waiting,
orange (`#FFE9D0`) state, resources and running, teal (`#D9F2EA`) the core and
success, red (`#FFDAD6`) failure, grey (`#ECEBF3`) automation and what was
cancelled, ink (`#17123F`) what lives outside the application. Break long labels
yourself with `<br/>` (`wrappingWidth` only sets the widest line), write the angle
brackets of a placeholder as `#lt;version#gt;`, and give a subgraph a row of nodes
by chaining them with `~~~`. Look at the result with Mermaid 11 and 12 (the
[live editor](https://mermaid.live) or `mmdc`) before you commit: GitHub shows an
error instead of a diagram that it cannot parse.

### Screenshots

The images of the documentation are not committed: they are hosted on
[freeimage.host](https://freeimage.host), a free image host that needs no
account, and linked.

**The desktop application.** `vfxtool sandbox screenshots` renders the pictures
into `dist/sandbox/mygo/screenshots/`, reproducibly: with the title bar of an
Ubuntu 26.04 GNOME session on X11, drawn by Metacity with the Yaru theme, over the default wallpaper of
Ubuntu instead of the black of a bare X server. It starts the end-to-end build
seven times against the fake vfox, drives it to every page through the automation plugin
of mygo, and captures the screen with ImageMagick when the page has settled:

| Run | What it shows |
| --- | --- |
| plain | every page, in both themes and languages: `home`, `sdks`, `available`, `tasks`, `plugins`, `dialog`, `settings`, `dark-expanded`, `dark-sdks`, `dark-available`, `zh-home`, `zh-settings` |
| `VFOX_PLUS_FAKE_DELAY=4` | an installation in progress: `installing` |
| `VFOX_PLUS_FAKE_INVENTORY=profiles` | a machine that has Java and Flutter installed next to Node.js, and what vfox-plus offers for these plugins: `java-installed`, `java-available`, `java-choose`, `java-temurin`, `java-install-name`, `java-all`, `flutter-available`, `options`, `options-mirror`, `options-aliases`, `alias-set`, `link-manual`, `zh-options`, `zh-java-available` |
| `VFOX_PLUS_E2E_WINDOW=412x800` | the compact layout, where the window is narrower than 600 dp: `compact-home`, `compact-detail` |
| both of the above | the distributions of Java in the compact layout: `compact-java` |
| the setting `webInterface`, and `VFOX_PLUS_E2E_ADDRESSES=192.168.1.20` | the settings of the web interface, as a person goes through them: the switch that is on with its address, the question that asks before other devices may connect, the settings that come with them, and the code that pairs a phone: `web-interface`, `web-allow`, `web-remote`, `web-pair` |
| the same, with the language chosen first | the same four in Simplified Chinese: `zh-web-interface`, `zh-web-allow`, `zh-web-remote`, `zh-web-pair` |

The end-to-end build can be told the addresses of the computer with `VFOX_PLUS_E2E_ADDRESSES`, a list
separated by commas, instead of reading them from the network interfaces. The pictures show `192.168.1.20`,
an address of the kind that a home network has, the same at every run and the address of no machine, and
a sandbox that has no network, and so no address of its own, still has one to pair a phone with.

To take another picture, add a `c.shot(s, "name")` to the function of the run in
`shared/tooling/internal/screenshots/screenshots.go` (`pages`, `installing`, `profiles`,
`compactPages`, `compactProfiles` or `webInterface`). The steps before it click and wait the way a
person with a screen reader does, through the names that the user sees (the label
of a button, the key that a component gave it), so a renamed button fails the
scenario instead of taking a wrong picture.

The README and the [user guide](../user/index.md) link `home`, `sdks`,
`dark-available`, `installing`, `tasks`, `plugins`, `zh-settings`, `compact-home`,
`compact-detail`, `java-choose`, `java-temurin`, `flutter-available` and `options`, and
the web interface on a phone, `webapp-en` and `web-phone-java-temurin` (`webapp-zh` and
`web-phone-zh-java-temurin` in Simplified Chinese; see below). The
[usage guide](../user/using.md) links `java-install-name` and `options-mirror`,
and `zh-java-available` and `zh-options` in Simplified Chinese. The
[guide to the web interface](../user/remote.md) links `web-interface`, `web-allow`, `web-remote` and
`web-pair` (with the prefix `zh-` in Simplified Chinese), and the web interface on a phone:
`webapp-en`, `web-phone-java-temurin`, `web-connections` and `web-connections-pair` (`webapp-zh`,
`web-phone-zh-java-temurin`, `web-zh-connections` and `web-zh-connections-pair` in Simplified Chinese).
The [guide to the Android app](../user/android.md) links the pictures of the app, and the
[guide to the gateway](../user/gateway.md) the pictures of the gateway (see below). The other
pictures are there to look at when the interface changes; nothing links them.

**The web interface.** Render it with the end-to-end build of the application, in a virtual
display, and headless Chrome, in a language pinned for the picture. `vfxtool webheap` takes the screenshot of the page
that `--tab` opens (`home`, `sdks`, `plugins`, `tasks`, `settings`) or of a scene
that `--scene` plays: `home`, `connections`, `connections-pair`, `flutter-chips`, `java-available`,
`java-choose`, `java-temurin`, `java-install-name`, `options`, `options-mirror`, `options-aliases`,
`options-legacy`, `settings-languages`, `alias-set` and `link-manual` (see
`shared/tooling/internal/webheap/scenes.go`, whose test checks the elements that the
scenes click against `mygo/pair/webapp.html`). A scene ends by waiting for the page to fit its
window, so a layout that overflows fails the shot instead of making a picture. Every picture gets
an application and a home of its own, because choosing a distribution is kept in the settings of
the application, and a scene that chooses one would change every picture after it. The application
serves its web interface because its settings say so, and allows other devices, which the
connections and the code that pairs a phone need (`{"webInterface":true,"webRemote":true}`); it is
told the addresses of the documentation (see above); and the end-to-end build writes a pairing link
to the file that `VFOX_PLUS_E2E_LINK` names, because it has no command line to print it on. The
scenes of Java need a machine that has Java and Flutter installed, and the home page needs one that
has not, which is why the inventory of the fake vfox is a variable of the call:

```sh
go -C shared/tooling build -o /tmp/vfxtool ./cmd/vfxtool
/tmp/vfxtool prepare --root mygo && /tmp/vfxtool build --e2e --root mygo /tmp/vfox-plus-e2e

# shoot NAME LANGUAGE WIDTH,HEIGHT [options of webheap]
# INVENTORY=profiles shoot ... plays a machine that has Java and Flutter.
shoot() {
  name=$1 lang=$2 window=$3; shift 3
  home=/tmp/shot-home
  rm -rf $home && mkdir -p $home/bin $home/run $home/.config/vfox-plus && chmod 700 $home/run
  cp /tmp/vfxtool $home/bin/vfox
  echo '{"webInterface":true,"webRemote":true}' > $home/.config/vfox-plus/settings.json
  (
    export HOME=$home XDG_CONFIG_HOME=$home/.config XDG_RUNTIME_DIR=$home/run PATH=$home/bin:$PATH NO_COLOR=1 \
      VFOX_PLUS_FIXTURES=$PWD/shared/fixtures/vfox-1.1.0 VFOX_PLUS_E2E_LINK=$home/link.txt \
      VFOX_PLUS_E2E_ADDRESSES=192.168.1.20
    [ -n "${INVENTORY:-}" ] && export VFOX_PLUS_FAKE_INVENTORY=$INVENTORY
    exec setsid xvfb-run -a dbus-run-session -- /tmp/vfox-plus-e2e >$home/app.log 2>&1
  ) &
  app=$!
  for i in $(seq 1 30); do
    LINK="$(grep -o 'http://127\.0\.0\.1:[0-9]*/auth/connect/[0-9a-f]*' $home/link.txt 2>/dev/null | head -1)"
    [ -n "$LINK" ] && break
    sleep 1
  done
  /tmp/vfxtool webheap --server "${LINK%%/auth/connect/*}" \
    --pairing "$LINK" --output /tmp/shot.txt --screenshot "$name.png" \
    --lang "$lang" --window "$window" --idle 0 --chrome "$(which google-chrome)" "$@"
  kill -- -$app
}
shoot webapp-en en 412,915
shoot webapp-zh zh 412,915
INVENTORY=profiles shoot web-phone-java-temurin en 412,915 --scene java-temurin
INVENTORY=profiles shoot web-phone-zh-java-temurin zh 412,915 --scene java-temurin
shoot web-connections en 412,915 --scene connections
shoot web-connections-pair en 412,915 --scene connections-pair
shoot web-zh-connections zh 412,915 --scene connections
shoot web-zh-connections-pair zh 412,915 --scene connections-pair
```

(`PATH` first carries a `vfox` executable — a copy of `vfxtool` acts as the
fake vfox with the fixtures — so that the SDK list has content, and
`INVENTORY=profiles` makes it a machine that has Java and Flutter;
`setsid` makes the application the leader of a group of processes of its own, which `kill -- -$app`
ends together with the virtual display;
`--lang` pins the interface language; `--window 412,915` shoots the phone portrait
and defaults to the 1280 by 800 desktop window; `--idle 0` skips
the twenty seconds on the open page that the local heap diagnosis waits (the
heap numbers stay local, out of the CI report); `--dark`
takes the picture in the dark theme; `--chrome` takes any Chrome that speaks CDP, and
the tool looks for one in `$CHROME_BIN` and on `PATH` when it is left out.) Look at the
size of the picture before you upload it (`file webapp-en.png`): not every
Chrome keeps a window that small. Chrome for Testing 156 answered
`--window 412,915` with a picture of 500 by 715, and `chrome-headless-shell` of
the same version gives exactly 412 by 915, so pass that one with `--chrome` when
the size is wrong.

**The gateway.** `vfxtool gateway-shots` takes the pictures of its pages: it does what
`vfxtool gateway-check` does (see [the implementation guide](implementation-mygo.md)), with a headless
Chrome in the place of the person, and keeps what the Chrome shows at each step. It writes
`gateway-intro` (a browser that is not paired), `gateway-home` (paired with the gateway, and the gateway
paired with nothing), `gateway-link` (the link of the application pasted), `gateway-trust` (the
fingerprint to compare), `gateway-app` and `gateway-zh-app` (the interface of the application through the
gateway, in English and in Simplified Chinese) and `gateway-paired`, each as high as its page. The pages
of the gateway are in both languages at once, so one picture serves both guides. The pictures show
the address of the application, which the gateway really connects to, so the computer has to have it:
`unshare -rn` runs a command in a network namespace of its own, in which you are root, that has nothing
but a loopback interface, and the address goes to that interface, without touching your network. The
command stops before it starts anything, and says so, when the computer lacks the address.

```sh
go -C mygo build -o /tmp/vfox-plus-gateway ./cmd/vfox-plus-gateway
unshare -rn sh -c 'ip link set lo up && ip addr add 192.168.1.20/32 dev lo &&
  xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- \
  /tmp/vfxtool gateway-shots --binary /tmp/vfox-plus-gateway --output /tmp/gateway-shots /tmp/vfox-plus-e2e'
```

(The build of the application is the one above, run from the root of the repository, which the
command needs for the fixtures of the fake vfox; `--chrome` names the Chrome as it does for `webheap`.)

**Android.** CI takes one picture, `screenshot-android`: the `measure` job of
`android.yml` runs the release APK on an emulator, `vfxtool measure-apk` captures the
cold start, and the job uploads it as an artifact (`gh run download`). The nine
pictures of the [guide to the Android app](../user/android.md) are taken on an emulator of
your own ([An emulator of your own](environment.md#an-emulator-of-your-own): API 36, 1080 by 2400
at 420 dpi), with the release APK of the build that CI makes (its artifact `vfox-plus-android`),
and with the application of the PC that [Run the app](environment.md#run-the-app) describes: the
end-to-end build against the fake vfox of `VFOX_PLUS_FAKE_INVENTORY=profiles`, with the web interface and the
other devices on, the port 8080 and `VFOX_PLUS_E2E_ADDRESSES=10.0.2.2`, the address at which the emulator
reaches the computer. The pairing screen, the links and the certificate then name the address that runs,
and there is no certificate to make. The nine are `android-pairing`, `android-scan`, `android-trust`,
`android-home`, `android-sdk`, `android-settings`, `android-dark-pairing`, `android-dark-home` and
`android-unreachable`; the guide links all of them, in English for both languages. What makes the
pictures alike, and what goes wrong:

- The demo mode of the status bar fixes the clock and shows a full battery and
  wifi and no notifications: `adb shell settings put global sysui_demo_allowed 1`,
  then `adb shell am broadcast -a com.android.systemui.demo -e command` with `enter`,
  `clock -e hhmm 1000`, `battery -e level 100 -e plugged false`,
  `network -e wifi show -e level 4 -e fully true`, `network -e mobile hide` (its icons
  come out white on the light theme) and `notifications -e visible false`.
- Android shows a green indicator in the status bar while the camera is open, in place of the wifi and
  battery icons, in the picture of the scanner. Turn it off with `adb root`, then
  `adb shell device_config put privacy camera_mic_icons_enabled false` (the shell may not write
  that flag, root may).
- The language and the theme are the ones of the app and of the system:
  `adb shell cmd locale set-app-locales dev.vfox.vfoxplus --locales zh-CN` and
  `adb shell cmd uimode night yes`. Clear the data between runs (`pm clear`) and grant
  the camera again, which `pm clear` drops, or the system dialog covers the scanner.
- The views of the app are found by their ids in `adb exec-out uiautomator dump /dev/tty`
  (`scanButton`, `manualButton`, `linkField`, `connectButton`, `reconnectButton`). The web interface in
  the WebView is not in the dump: the bar at the bottom is tapped by its position, with
  `adb shell input tap X 2205` for X = 125, 332, 540, 749 and 956 (Home, SDKs, Plugins, Tasks, Settings),
  and flutter is the first of the plugins, at `540 682`.
- Type the pairing link with the app in English (`adb shell input text`). With the app in Chinese Gboard turns
  to Pinyin and writes the link with full-width punctuation and "author" for "auth";
  the field keeps its text when the language is changed back before *Connect*. The keyboard covers
  the button: close it with BACK (`adb shell input keyevent 4`) before.
- The dialog of the fingerprint and the settings are taken in one run of the application: it makes its
  certificate again at every run, and the two show the same fingerprint only within one. Choose
  *Trust once*, and wait for the page to load before the picture of the home page.
- To see the page that says the PC cannot be reached, end the application of the PC and open the app
  again: it opens on the pairing screen, with the last server, and *Reconnect* shows the page.
- Take the picture with `adb exec-out screencap -p > NAME.png` when the screen has
  settled. The scanner shows the living room of the emulator (`hw.camera.back=virtualscene`), as there
  is no code to point it at.

The pictures are 1080 by 2400 as they come. `vfxtool frame-phone NAME.png OUT.png` gives them the
rounded corners of a phone display (a radius of 96 px, transparent outside) and a grey border of 5 px
(`#7A7A84`) over the edge: the cream screens of the light theme have no edge against the white page of GitHub.

**Uploading.** Upload a new image with

```sh
curl -s -F key=6d207e02198a847aa98d0a2a901485a5 -F action=upload -F format=json \
  -F source=@dist/sandbox/mygo/screenshots/home.png https://freeimage.host/api/1/upload |
  jq -r '.image.url'
```

which prints its URL (the key is the public one that freeimage.host gives to
everyone on its [API page](https://freeimage.host/api)), and replace the link in
the documentation. If curl stops with error 56, the connection broke: run it again.
Every upload gets a new URL, so cached copies of an old image never show. The
uploader of an anonymous image cannot delete it; upload nothing that must not
stay public.

### Auditing the interface

`vfxtool webheap` can take the census of the page that it shoots: `-audit census.json`
next to a `-screenshot` writes, in JSON, what a person reads and what could not be read:
the words, the sentences of eight words or more and the small lines under a title, the
texts under the contrast of WCAG AA, the rows of a list that have two buttons with words
on them, the lists in which three rows start with the same icon, the sizes off the type
scale, the controls under 24 px and whether the page is wider than its window (the
expression is `shared/tooling/internal/webheap/audit.js`). The budgets that a screen is
held to are in [the design system](design-system.md#budgets).

To audit the whole interface, shoot every tab (`-tab`) and every scene (`-scene`) as
described above, in the two colour schemes (`-dark`), on a phone (`-window 412,915`) and
in a wide window (`-window 1280,800`), each with its own `-audit` file, and read the files
together: the totals show where the words are, and a screen over a budget shows where to
look. Look at the pictures too, side by side on one sheet: the census has no number for a
layout that is bad in a way that nobody has measured yet.

The desktop application has the census in its tests (`mygo/ui/census_test.go`,
`mygo/ui/kit/census.go`). It draws each screen without a window, in the windows that
[the design system](design-system.md#budgets) names, and reads from the nodes that every
text, icon and control registered where it is, how a text is set (size, weight, colour)
and, for a text of a single line, the width that it wants, and from the picture of the
window what a text stands on. That gives it what the web census has not: a title or the text of a
control that an ellipsis cuts short, and an icon that the table of the icons lacks. A
screen that has a dialog or a menu open is the dialog or the menu. The windows are as tall
as a page can be, so that what is below the fold counts as well.

```sh
cd mygo
go test ./ui -run TestTheScreensAreWithinTheBudgets
```

fails with the budgets that a screen is over, each with the texts, rows or controls that
make it so. `VFOX_PLUS_CENSUS=dir` writes the census of every screen to `dir`, one JSON
file for each and `summary.txt` with a row for each, and `VFOX_PLUS_SNAPSHOTS=dir` writes
the picture of every screen too (`census.<screen>.<window>.png`), to look at what the numbers
are about. A new screen is a new entry of `censusScenes`.
