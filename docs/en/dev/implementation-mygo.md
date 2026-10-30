# vfox+

> [简体中文](../../zh-CN/dev/implementation-mygo.md)

vfox-plus built with [mygo](https://github.com/egoist/mygo), the desktop
application framework for Go, and its **native user interface** (package `ui`):
no web view, no HTML, no JavaScript. mygo draws the window itself, from elements
that are laid out with flexbox and painted with its own renderer, and opens it
through GTK 3 without a C compiler: it calls GTK, GLib, Pango and Cairo through
[purego](https://github.com/ebitengine/purego), so that the application is one
Go executable of about 18 MB that is linked to the C library alone and loads
GTK 3 when it starts. mygo has no widgets that look like Material 3 Expressive;
everything above its elements and its headless widgets is written for this
project, in Go:

| Directory | What it does |
| --- | --- |
| [`core/`](../../../mygo/core) | The models, the process runner, the vfox client and the parsers of its output, the settings store, the translation catalogues with their ICU messages, the plugin profiles (`core/profile`: what is particular to vfox-java, vfox-nodejs and vfox-flutter) — no UI |
| [`state/`](../../../mygo/state) | The observable controllers, the task queue (one state-changing command at a time, streamed output, download progress, cancellation, retry) |
| [`ui/`](../../../mygo/ui) | The application: the window, the shell with its navigation, the pages and the dialogs |
| [`ui/kit/`](../../../mygo/ui/kit) | The toolkit: the Material 3 Expressive components, built on mygo's elements and headless widgets |
| [`ui/mcu/`](../../../mygo/ui/mcu) | The colour scheme: a Go port of the Material colour utilities (HCT, CAM16, tonal spot) |
| [`ui/shapes/`](../../../mygo/ui/shapes) | The shapes: a Go port of the rounded polygons and their morphs of `androidx.graphics.shapes` |
| [`assets/`](../../../mygo/assets) | What the binary embeds: the icons, the translations and the licenses (all collected at build time) |
| [`pair/`](../../../mygo/pair) | `vfox-plus pair` / `vfox-plus serve`: pair a browser with this machine (one-time links) or keep the server up for containers and remote instances; the paired browser gets the full web interface (`/app`: SDKs, registry, tasks with live logs, settings) |
| [`main.go`](../../../mygo/main.go) | Assembles the model and the window; dispatches the `pair` subcommand |
| [`automation_e2e.go`](../../../mygo/automation_e2e.go) | Links the automation plugin of mygo, only with the build tag `e2e`: the end-to-end tests, the memory measurement and the screenshots drive the application through it |

The layers follow [the architecture](architecture.md): `core` knows
nothing of the others, `state` only knows `core`, and `ui` sits on top. `core`,
`state`, `ui/mcu` and `ui/shapes` are plain Go without a user interface
toolkit, and the parsers pass the [fixtures](../../../shared/fixtures/README.md).

## Requirements

- Go (pinned in [`versions.env`](../../../mygo/versions.env)); mygo needs Go 1.27.

- At run time GTK 3, fontconfig and a font, and vfox 1.0.0 or later:

  ```sh
  sudo apt install libgtk-3-0t64 fontconfig fonts-ubuntu fonts-noto-cjk
  ```

- A C compiler only for the race detector of the unit tests (`go test -race`
  needs cgo); the application is built without one.

[Development environment](environment.md) says how to install all of this on
Ubuntu 26.04, also in WSL2.

## Everyday commands

`vfxtool` is the single Go tool that builds, tests and packages the
application (`../shared/tooling/cmd/vfxtool`). Build it once and put it on
`PATH`, or prefix every command below with
`go -C ../shared/tooling run ./cmd/vfxtool`:

```sh
go -C ../shared/tooling build -o ~/.local/bin/vfxtool ./cmd/vfxtool
```

| Task | Command |
| --- | --- |
| Collect the icons, the translations and the licenses the binary embeds | `vfxtool prepare` (set `MYGO_SKIP_LICENSES=1` to skip the licenses of the Go modules, which need `go mod download`) |
| Build (`build/vfox-plus`) | `vfxtool build build/vfox-plus` |
| Build the headless server (`build/vfox-plus-serve`) | `vfxtool build --package ./cmd/vfox-plus-serve build/vfox-plus-serve` (same `pair`, `serve` and `service` commands, no user interface; about a third smaller) |
| Cross-compile for Windows | `vfxtool build --goos windows --goarch amd64\|arm64 build/vfox-plus.exe` |
| Run | `vfxtool build build/vfox-plus && build/vfox-plus` |
| Format check (CI fails on a difference) | `golangci-lint fmt --diff` |
| Lint (CI fails on any finding) | `golangci-lint run` and `golangci-lint run --build-tags e2e` |
| Unit tests | `CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./...` |
| Coverage gates | `vfxtool coverage coverage.out 85` |
| `.deb`, `.tar.gz`, `.msix`, `.msixbundle` | see [Packaging](#packaging) |

`vfxtool prepare` cuts the SVGs of the icons of [`shared/icons.json`](../../../shared/icons.json)
(outlined and filled, see [Icons](#how-the-user-interface-works)) out of the npm package that the
file pins, copies the catalogues of [`shared/i18n`](../../../shared/i18n), and writes the licenses, all
to `mygo/assets`. It needs the network twice, the first time only: for the package (1.3 MB, checked
against its SHA-512 and kept in `vfox-plus/icons` of the cache directory of the user, or in the
directory that `VFOX_PLUS_ICONS` names, where it can be put beforehand; `VFOX_PLUS_NPM_REGISTRY` names a
mirror of the registry), and for `go mod download`, which the licenses of the Go modules need.
`vfxtool prepare --fetch` only downloads the package. The result is not committed;
`//go:embed` in [`assets/assets.go`](../../../mygo/assets/assets.go) fails the build if it is missing.

The release binary is built with `CGO_ENABLED=0`, position independent
(`-buildmode=pie`), stripped, without the inspector of mygo's native user
interface, and about 18 MB.

## Sandbox

`vfxtool sandbox` runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the image brings Go,
golangci-lint, GTK 3, Xvfb and vfox (it extends the shared base image, see
[`shared/sandbox`](../../../shared/sandbox)), sees the repository read-only, and keeps
the Go module and build caches in Docker volumes.

| Command | What it does |
| --- | --- |
| `vfxtool sandbox check` | Formatting, golangci-lint (also with the end-to-end tag), `go vet`, the unit tests with the race detector and the coverage gates |
| `vfxtool sandbox e2e` | The end-to-end scenarios, against the fixtures and against the real vfox |
| `vfxtool sandbox package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `vfxtool sandbox measure` | The memory use on the Home page and with the versions of Node.js listed |
| `vfxtool sandbox screenshots` | Screenshots of every page, and of Java, Flutter and the options of Node.js |
| `vfxtool sandbox all` | `check`, `e2e` and `package` |
| `vfxtool sandbox shell` | A shell in the sandbox |
| `vfxtool sandbox clean` | Deletes the image, the cache volumes and the results |

Results are written to `dist/sandbox/mygo/<command>/`. Set `SANDBOX_GOPROXY`
(for example `https://goproxy.cn`) if the default Go module proxy is slow from
where you are, `SANDBOX_UBUNTU_MIRROR` for the Ubuntu archive and
`SANDBOX_NPM_REGISTRY` (for example `https://registry.npmmirror.com`) for the npm package of the icons.

## Tests

- **Unit tests** cover `core`, `state`, the colour scheme and the shapes. The
  gates are 85 % of the statements of `core` and `state` together and 75 % of
  those of `ui`, which the tests of its components and pages bring to about 90 %.
  The parsers run against **every** case in
  [`shared/fixtures`](../../../shared/fixtures/README.md).
- **Component and page tests** run mygo's views without a window, with its
  `ui.Tester`, which lays out and draws frames in memory, clicks, types and
  advances a clock of its own ([`ui/kit`](../../../mygo/ui/kit/behaviour_test.go)).
  [`ui/app_test.go`](../../../mygo/ui/app_test.go) runs the whole application against the
  captured output of vfox, one flow per test: Home, the navigation, the SDK
  list and detail, installing, using, uninstalling, the registry, tasks,
  settings, the compact layout and the missing vfox. Set
  `VFOX_PLUS_SNAPSHOTS=<directory>` to have the tests write PNGs of the pages
  there.
- **Golden tests** compare the Go colour scheme and shapes with the JavaScript
  libraries they are ported from. [`ui/mcu/testdata`](../../../mygo/ui/mcu/testdata)
  holds the schemes of 82 seed colours in both modes, bit-exact;
  [`ui/shapes/testdata`](../../../mygo/ui/shapes/testdata) holds the outlines of the shapes,
  which agree to within 2 % of their size. Each `testdata` directory explains
  how to regenerate its file: the colour schemes need Node.js to run the
  JavaScript reference implementation, while the shapes are regenerated with
  plain `go run generate.go`, no Node.js required.
- **End-to-end scenarios** drive the built application from outside, the way
  assistive technology does. `vfxtool build --e2e` builds it with the tag `e2e`, which links the
  [automation plugin](https://github.com/egoist/mygo/blob/v0.7.3/docs/plugins/automation.md) of
  mygo ([`automation_e2e.go`](../../../mygo/automation_e2e.go)): when `MYGO_AUTOMATION` names a Unix
  socket, the application listens on it and answers requests to list the elements of its window
  (the ones it gives screen readers, with the keys that the components set), to press them, to
  fill the fields, to send keys and to capture the window. The scenarios are in
  [`shared/tooling`](../../../shared/tooling) (`internal/e2erun`); the application has no code for
  them, except the profile of the offline test plugin and the size of the window of the compact
  layout (`VFOX_PLUS_E2E_WINDOW`, as `WIDTHxHEIGHT`):

  ```sh
  vfxtool build --e2e build/vfox-plus-e2e
  xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- vfxtool e2e-run fake build/vfox-plus-e2e
  VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- \
    vfxtool e2e-run real build/vfox-plus-e2e
  ```

  `fake` runs against the fake vfox of the fixtures, `real` against the real vfox with the offline
  test plugin and its server. `vfxtool e2e-run` prepares a new, empty home directory, starts the
  application, waits for its window, runs the scenario and ends the application. A wait that times
  out fails the scenario, after the elements that the window showed are printed. The scenarios
  exercise the real components, the state, process spawning, streaming, cancellation with `SIGTERM`,
  the settings file and vfox itself. `vfxtool measure` and `vfxtool screenshots` start the application
  the same way (see [How it is measured](memory.md#how-it-is-measured) and
  [Screenshots](contributing.md#screenshots)). The build keeps the inspector of the native user
  interface, which the plugin needs and the release build leaves out; it is about 0.4 MB larger.

## How the user interface works

mygo calls a function of the state, the view, for every frame that something
changed in, and the view builds the elements of the window again: containers
(`ui.Row`, `ui.Column`, `ui.Scroll`, `ui.List`), text, and the widgets without a
look that handle the pointer, the keyboard and the focus (`ui.ButtonBase`,
`ui.SwitchBase`, `ui.TextInputBase`, `ui.PopoverBase`, `ui.DialogBase`,
`ui.TooltipBase`, `ui.ToastViewportBase`). What persists between frames is kept
by mygo under the identity of an element (its place among its siblings, or a
key). [`ui/kit`](../../../mygo/ui/kit) gives these the look of Material 3 Expressive, and the
pages are built from its components.

- **Components** are functions of a `kit.Ctx` (mygo's context and the
  environment of the window) that build their elements in place and return the
  outermost one, which the caller places with the element methods
  (`Grow`, `Margin`, `Width`, …): `kit.Button(k, kit.ButtonSpec{…})`. Their
  actions are callbacks that mygo runs after the frame was built.
- **Motion** is a damped spring per animated value, with the spatial and effects
  springs of the Material 3 Expressive motion scheme
  ([`motion.go`](../../../mygo/ui/kit/motion.go)), given to mygo's `AnimateTo`: exact
  solutions that keep their velocity when the target changes, drawn only while
  they move, so the window is idle when nothing changes. Reduced motion from the
  desktop makes every value jump. Dialogs and menus pop in with a pose spring,
  and the pages give way on mygo's element transitions (fade through, shared
  axis). The loading indicator and the wavy progress animate in the painter,
  without building the view again.
- **Overlays** — menus, tooltips, dialogs and snack bars — are mygo's popovers,
  tooltips, dialogs and toasts, which take care of the focus, Escape and the press
  outside; the kit styles them.
- **Lists** are virtual: only the rows in view are built, their heights are
  remembered, and a jump of thousands of pixels costs the same as a step. The
  version list of Node.js has 872 entries.
- **The registry** ([`registry.go`](../../../mygo/ui/kit/registry.go)) is a flat list of the
  nodes the components of a build registered: role, name, state, the list item or
  dialog they are in, and what a click or an input does. The tests of the components
  and the pages use it to find and operate widgets; it is not an accessibility tree
  (mygo gives screen readers its own, which the end-to-end scenarios read, see
  [Limits](#limits)).
- **Colours** are generated at run time from the seed colour or the accent
  colour of the desktop with the Material colour utilities
  ([`ui/mcu`](../../../mygo/ui/mcu), tonal spot 2021). mygo's
  own widgets (text fields, scroll bars, tooltips) follow them through its theme.
- **Shapes** — the avatars, the stat tiles, the swatches, the loading
  indicator — are the polygons and morphs of Material 3 Expressive, drawn as
  paths ([`ui/shapes`](../../../mygo/ui/shapes)).
- **Text** is laid out and drawn by Pango, in the sans-serif font of the desktop
  found with fontconfig: Ubuntu Sans on Ubuntu, with the weights of the type
  scale, and the fonts that the system chooses for Chinese (Noto Sans CJK).
- **Icons** are the SVGs of Material Symbols Rounded that `vfxtool prepare` cuts out of an npm
  package, drawn by mygo's `ui.Icon` in the colour of the text, the outlined and the filled one of
  each symbol; the fill that the navigation and the buttons animate is the cross-fade of the two
  ([`ui/kit/icon.go`](../../../mygo/ui/kit/icon.go)). An SVG is parsed when its icon is first drawn.

## Packaging

The release binary contains everything.

On Linux, the shared tool `vfxtool build-deb` packages it as
`vfox-plus_<version>_amd64.deb` and a `.tar.gz`, with the desktop entry,
AppStream metadata and icon of [`shared/packaging/linux`](../../../shared/packaging/linux);
the version is the one in [`VERSION`](../../../mygo/VERSION), which a test keeps equal to the
one the application shows. The application names its window `dev.vfox.VfoxPlus`
(GTK takes the application ID of Wayland and the window class of X11 from the
program name, which the application sets with `g_set_prgname` before mygo starts
GTK), so that it matches the desktop entry. The licenses of the Go modules, of
the Go standard library, of the ports and of the icons are installed as
`/usr/share/doc/vfox-plus/THIRD-PARTY-LICENSES` and shown in the
application ([`NOTICE.txt`](../../../mygo/NOTICE.txt) holds the notices of the ports; the rest
is collected by `vfxtool golicenses`, which fails on a license that is not
compatible with the Apache License 2.0).

The package depends on GTK 3 (which brings GLib, Pango, Cairo and the rest of
what mygo loads), fontconfig and a font (`fonts-ubuntu` or `fonts-noto-core`),
and recommends `fonts-noto-cjk` for Chinese, a desktop portal and `xdg-utils` for
links. The list is the `Depends` of `vfxtool runtask` (`shared/tooling/internal/runtask`), and CI
installs the package into a clean Ubuntu container to prove that it is complete.

On Windows 11, `vfxtool build-msix` packages cross-compiled executables
(`vfxtool build --goos windows --goarch amd64|arm64`) as unsigned
`vfox-plus_<version>_<arch>.msix` and a `vfox-plus_<version>.msixbundle`,
with the manifest, logos and bundle layout of
[`shared/packaging/windows`](../../../shared/packaging/windows). Users install the
unsigned bundle in an elevated PowerShell with
`Add-AppxPackage -AllowUnsigned`; the bundle is not signed yet and an
officially signed bundle will be provided later. CI cross-compiles and
packages on Linux, validates with
`vfxtool validate-msix`, and installs/uninstalls the bundle on Windows
Server 2025 (x64) and Windows 11 on ARM (arm64).

### Docker image

[`Dockerfile`](../../../mygo/Dockerfile) builds the published image from the repository
root: a minimal Ubuntu runtime with vfox and the headless server
(`vfox-plus-serve serve`, the image entry point), needing no display server.
The toolchain versions default to [`versions.env`](../../../mygo/versions.env).
The toolchain versions default to [`versions.env`](../../../mygo/versions.env).

```sh
docker build -f mygo/Dockerfile -t vfox-plus .
docker run --rm -p 8080:8080 vfox-plus
```

Open the pairing link it prints, with `0.0.0.0` replaced by the reachable
address. The link is `https`: the container generates a self-signed
certificate on first start and prints its fingerprint for comparison in
the browser. A `-v vfox-plus-data:/home/vfoxplus` volume keeps the installed
SDKs between runs. A browser on another machine pairs the same way, over
the network the container is reachable on.

## Memory

Measured with `vfxtool sandbox measure`: Ubuntu 26.04 in Docker, Xvfb, a
1280 × 800 window, one process, the end-to-end build (the release build with the
inspector and the automation plugin that the measuring drives it through, which
take up to 1 MB of PSS).

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 63 MB | 51 MB |
| Versions of Node.js listed | 73 MB | 59 MB |

mygo draws with the processor, and the application keeps GTK from loading
OpenGL: a GDK that opens its OpenGL connection maps Mesa, and with the software
driver of Mesa its compiler LLVM, which is more than the application takes (the
same build took about 78 MB of PSS on the Home page when GDK did, and about
43 MB when it did not, with every other setting equal). An interface this size
does not need the GPU, and the memory does not depend on the driver of the
machine. What is left is mostly the executable with the locale data it embeds,
the heap of Go, and GTK with the C libraries it loads. The list of versions
adds 8 MB because only the rows in view are built.

## Limits

- **OpenGL is off on purpose.** mygo draws on the GPU with OpenGL where it runs
  on one, which loads Mesa, and with the software driver of Mesa the compiler
  LLVM: GDK opens its OpenGL connection when the display opens, and that alone
  maps some 35 MB of PSS. The pages of vfox+ draw on the CPU in a fraction of a
  millisecond, so the application sets `MYGO_GPU=0` and `GDK_GL=disable` before it
  starts GTK, unless `MYGO_GPU` is set (`MYGO_GPU=1` draws with OpenGL). Dialogs
  that scale in and a window that is resized are the frames that cost most on the
  CPU; a machine of today draws them within a frame.
- **The title bar is the desktop's.** GTK draws the title bar (client-side on
  GNOME's Wayland session, the window manager's on X11) in the style of the
  desktop; mygo can hand the title bar to the application, which this implementation
  does not use yet.
- **Scaling** is the one of GTK 3: whole numbers, from `GDK_SCALE` or the
  desktop, and the text scale of the desktop.
- **mygo is young** (v0.7.3 at the time of writing, with a changing API): the
  version is pinned in [`go.mod`](../../../mygo/go.mod), and the pages follow what that
  version offers.
- **Accessibility** goes through mygo's ATK objects, which GTK bridges to AT-SPI,
  so that Orca reads the window. The end-to-end scenarios read the same elements
  through the automation plugin, which is why the components name theirs (the
  tooltip of an icon button is its name; the row of a task says its status). mygo's
  screen reader support has not been tried with Orca here.
- **Input methods** are the ones of GTK 3 (IBus, Fcitx, the Wayland text input).
- **Only Linux on x86-64 is run and tested end to end.** mygo has its own backends
  for Windows and macOS, and the application compiles for them and for Linux on
  arm64 (`vfxtool build --goos windows --goarch amd64` and the like). Windows 11
  is packaged as an unsigned `.msixbundle` (x64 and arm64) and validated in CI;
  installing it is sideloading with `Add-AppxPackage -AllowUnsigned`. `core` is
  portable: process termination (`SIGTERM` on Unix, kill on Windows), `PATH`
  splitting (`:` vs `;`), vfox lookup (`vfox` vs `vfox.exe` with Windows fallback
  directories) and the settings location (`~/.config` vs `%APPDATA%`) follow the
  platform.
- **`.apk` lives in `android/`, not here.** mygo has no Android backend, so the
  interface cannot be a native Android app built from this code. The Android
  client is a WebView around the pairing server (`vfox-plus serve`): the phone
  drives the vfox-plus instance on the PC over the
  [remote protocol](remote-protocol.md), and the web interface it shows follows
  the [design system](design-system.md) (see
  [Remote front ends](architecture.md#remote-front-ends)).
