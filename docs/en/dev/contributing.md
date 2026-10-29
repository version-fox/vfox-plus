# Contributing to vfox-plus

> [简体中文](../../zh-CN/dev/contributing.md)

Thank you for helping! This repository holds the vfox-plus application (see
the [user guide](../user/index.md)); most of what follows applies to all of its parts.

## Ground rules

- Code comments, commit messages and pull requests are written in
  **English**. User-visible texts go through the translation catalogues in
  [`shared/i18n`(../../../shared/i18n) and are available in English and Simplified
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
  [`.editorconfig`(../../../.editorconfig) and [`.gitattributes`(../../../.gitattributes)).
- Binary files are not committed. Upload screenshots and other images to an
  image host and link them (see [Screenshots](#screenshots)); generate binary
  build inputs, such as the logos of the Windows package, at build time.
  CI rejects binary files.

## Workflow

To set up a machine (Ubuntu 26.04, on its own or in WSL2), see
[Development environment](environment.md).

1. Open an issue or comment on an existing one before large changes.
2. Create a branch from `main` and keep pull requests focused.
3. Run the checks; they are the same as in CI and listed in the
   [mygo README(implementation-mygo.md#everyday-commands):

   ```sh
   cd mygo
   golangci-lint fmt --diff
   golangci-lint run && golangci-lint run --build-tags e2e
   CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./... && vfxtool coverage coverage.out 85
   ```

   To run every check, the end-to-end tests and the packaging without installing
   anything but Docker, and without any risk for the vfox installation on your
   machine, use the sandbox, `vfxtool sandbox all` (see
   [shared/sandbox(sandbox.md)).

4. Describe what changed and how you tested it in the pull request. Include
   screenshots for visual changes, in light and dark themes.

## Continuous integration

| Workflow | Runs on | Checks |
| --- | --- | --- |
| [`repository.yml`](../../../.github/workflows/repository.yml) | every change | License headers (SkyWalking Eyes), workflows (actionlint), vfxtool (`gofmt`, `go vet`, `go test`), shared JSON, no committed binary files |
| [`memory.yml`](../../../.github/workflows/memory.yml) | the end of the build workflow, on pull requests | One comment on the pull request with the memory of the application, and the difference from `main` (see [Memory in pull requests(memory.md#memory-in-pull-requests)) |
| [`mygo.yml`](../../../.github/workflows/mygo.yml) | `mygo/`, `shared/` | Formatting and lint (golangci-lint, with and without the end-to-end build tag), unit tests with the race detector and the coverage gates, the licenses of the Go modules, end-to-end scenarios in the application against captured output and against the real vfox, the memory measurement, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container, unsigned `.msix` and `.msixbundle` for Windows 11 (x64 and arm64) validated with `vfxtool validate-msix` on Linux, and validated plus installed/uninstalled on Windows Server 2025 (x64) and Windows 11 on ARM (arm64) |
| [`sandboxes.yml`](../../../.github/workflows/sandboxes.yml) | `mygo/sandbox/`, `mygo/versions.env`, `shared/sandbox/`, `shared/tooling/` | Builds the Docker sandbox and runs `vfxtool sandbox all` in it |
| [`android.yml`](../../../.github/workflows/android.yml) | `android/`, `mygo/pair/`, `mygo/VERSION` | Builds the release APK (signed with the debug key) and uploads it as `vfox-plus-android` |

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
[Memory in pull requests(memory.md#memory-in-pull-requests) before you trust a small
one.

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

## Common changes

### Texts

Add or change messages in [`shared/i18n/app_en.arb`(../../../shared/i18n/app_en.arb)
with a `description` (and placeholder metadata), then translate them in
[`app_zh.arb`(../../../shared/i18n/app_zh.arb). See [docs/i18n.md(i18n.md),
including the glossary.

### Icons

Add the Material Symbol to [`shared/icons.json`](../../../shared/icons.json) under a
name that describes its meaning, with the name of the symbol as at
<https://fonts.google.com/icons> (`history`, `folder_open`). Nothing else is to be
added: `vfxtool prepare` cuts the SVGs out of the pinned npm package when it builds,
and fails, naming the icons, for a symbol that the package does not have.

To move to a newer version of the package, change `version` and `sha512` in the
`source` of that file. The registry says the checksum in base64; the file wants it
in hexadecimal:

```sh
version=1.2.98
curl -fsS "https://registry.npmjs.org/@iconify-json%2fmaterial-symbols/$version" |
  grep -o '"integrity":"sha512-[^"]*"' | cut -d- -f2 | tr -d '"' | base64 -d | od -An -v -tx1 | tr -d ' \n'
```

### vfox output

When vfox changes, recapture the fixtures with
`vfxtool capture` and follow
[`shared/fixtures/README.md`(fixtures.md). Document the change in
[docs/vfox-cli-contract.md(vfox-cli-contract.md).

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
into `dist/sandbox/mygo/screenshots/`, reproducibly: with the Greybird title bar of
xfwm4, the way modern GNOME applications look, over the default wallpaper of
Ubuntu instead of the black of a bare X server. It runs the `screenshots`
scenario of the end-to-end build three times against the fake vfox, and the
application prints `e2e: SHOT <name>` whenever the screen is to be captured:

| Run | What it shows |
| --- | --- |
| plain | every page, in both themes and languages, and the compact layout: `home`, `sdks`, `available`, `tasks`, `plugins`, `dialog`, `settings`, `dark-expanded`, `dark-sdks`, `dark-available`, `zh-home`, `zh-settings`, `compact-home`, `compact-detail` |
| `VFOX_PLUS_FAKE_DELAY=4` | an installation in progress: `installing` |
| `VFOX_PLUS_FAKE_INVENTORY=profiles` | a machine that has Java and Flutter installed next to Node.js, and what vfox-plus offers for these plugins: `java-installed`, `java-available`, `java-choose`, `java-temurin`, `java-install-name`, `java-all`, `flutter-available`, `options`, `options-mirror`, `zh-options`, `zh-java-available`, `compact-java` |

To take another picture, add a `s.shot("name")` to `screenshotsScenario` or to
`profilesScreenshots` in `mygo/ui/e2e_scenarios.go`. The steps before it click and
wait the way a person does, through the names that the user sees, so a renamed
button fails the scenario instead of taking a wrong picture.

The README and the [user guide](../user/index.md) link `home`, `sdks`,
`dark-available`, `installing`, `tasks`, `plugins`, `zh-settings`, `compact-home`,
`compact-detail`, `java-choose`, `java-temurin`, `flutter-available` and `options`.
The [usage guide](../user/using.md) links `java-install-name` and `options-mirror`,
and `zh-java-available` and `zh-options` in Simplified Chinese. The
[remote guide](../user/remote.md) links `webapp-en` and `webapp-zh`, the paired web
interface in English and Simplified Chinese, and the scene `java-temurin` of the web
interface on a phone, in both languages (`web-phone-java-temurin` and
`web-phone-zh-java-temurin`, taken with `--window 412,915`). The other pictures are
there to look at when the interface changes; nothing links them.

**The web interface.** Render it with the headless server and headless Chrome, in a
language pinned for the picture. `vfxtool webheap` takes the screenshot of the page
that `--tab` opens (`home`, `sdks`, `plugins`, `tasks`, `settings`) or of a scene
that `--scene` plays: `java-available`, `java-choose`, `java-temurin`,
`java-install-name`, `flutter-chips`, `options` and `options-mirror` (see
`shared/tooling/internal/webheap/scenes.go`, whose test checks the elements that the
scenes click against `mygo/pair/webapp.html`). Every picture gets a server and a home
of its own, because choosing a distribution is kept in the settings of the server, and
a scene that chooses one would change every picture after it:

```sh
go -C mygo build -o /tmp/vfox-plus-serve ./cmd/vfox-plus-serve
go -C shared/tooling build -o /tmp/vfxtool ./cmd/vfxtool

# shoot NAME LANGUAGE WIDTH,HEIGHT [options of webheap]
shoot() {
  name=$1 lang=$2 window=$3; shift 3
  rm -rf /tmp/shot-home && mkdir -p /tmp/shot-home/bin && cp /tmp/vfxtool /tmp/shot-home/bin/vfox
  HOME=/tmp/shot-home PATH=/tmp/shot-home/bin:$PATH \
    VFOX_PLUS_FIXTURES=shared/fixtures/vfox-1.0.13 VFOX_PLUS_FAKE_INVENTORY=profiles NO_COLOR=1 \
    /tmp/vfox-plus-serve serve --bind 127.0.0.1 --port 0 >serve.log 2>&1 &
  server=$!
  for i in $(seq 1 30); do
    LINK="$(grep -o 'http://127\.0\.0\.1:[0-9]*/auth/connect/[0-9a-f]*' serve.log | head -1)"
    [ -n "$LINK" ] && break
    sleep 1
  done
  /tmp/vfxtool webheap --server "${LINK%%/auth/connect/*}" \
    --pairing "$LINK" --output /tmp/shot.txt --screenshot "$name.png" \
    --lang "$lang" --window "$window" --idle 0 --chrome "$(which google-chrome)" "$@"
  kill "$server"
}
shoot webapp-en en 412,915
shoot web-java-temurin en 1280,800 --scene java-temurin
```

(`PATH` first carries a `vfox` executable — a copy of `vfxtool` acts as the
fake vfox with the fixtures — so that the SDK list has content, and
`VFOX_PLUS_FAKE_INVENTORY=profiles` makes it a machine that has Java and Flutter;
`--lang` pins the interface language; `--window 412,915` shoots the phone portrait
and defaults to the 1280 by 800 desktop window that CI measures; `--idle 0` skips
the twenty seconds on the open page that the measurement of the heap needs; `--dark`
takes the picture in the dark theme; `--chrome` takes any Chrome that speaks CDP, and
the tool looks for one in `$CHROME_BIN` and on `PATH` when it is left out.) Look at the
size of the picture before you upload it (`file webapp-en.png`): not every
Chrome keeps a window that small. Chrome for Testing 156 answered
`--window 412,915` with a picture of 500 by 715, and `chrome-headless-shell` of
the same version gives exactly 412 by 915, so pass that one with `--chrome` when
the size is wrong.

**Android.** The Android screenshot, `screenshot-android`,
comes from CI instead: the `measure` job of `android.yml` runs the release
APK on an emulator, `vfxtool measure-apk` captures the cold start, and
the job uploads it as the `screenshot-android` artifact. Download it with
`gh run download` and upload it the same way.

**Uploading.** Upload a new image with

```sh
curl -s -F key=6d207e02198a847aa98d0a2a901485a5 -F action=upload -F format=json \
  -F source=@dist/sandbox/mygo/screenshots/home.png https://freeimage.host/api/1/upload |
  jq -r '.image.url'
```

which prints its URL (the key is the public one that freeimage.host gives to
everyone on its [API page](https://freeimage.host/api)), and replace the link in
the README. If curl stops with error 56, the connection broke: run it again.
Every upload gets a new URL, so cached copies of an old image never show. The
uploader of an anonymous image cannot delete it; upload nothing that must not
stay public.
