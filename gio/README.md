# vfox+ — Gio implementation

vfox-plus built with [Gio](https://gioui.org), the immediate-mode GUI library
for Go. Gio provides the window, the input, the text shaping and the GPU
renderer (OpenGL ES through EGL on X11 and Wayland); it has no widgets that
look like Material 3 Expressive. Everything above it is written for this
project, in Go, in one process without a web view or a runtime:

| Directory | What it does |
| --- | --- |
| [`core/`](core) | The models, the process runner, the vfox client and the parsers of its output, the settings store, the translation catalogues with their ICU messages — no UI |
| [`state/`](state) | The observable controllers, the task queue (one state-changing command at a time, streamed output, download progress, cancellation, retry) |
| [`ui/`](ui) | The window, the shell with its navigation, the pages, the dialogs and the end-to-end scenarios |
| [`ui/kit/`](ui/kit) | The toolkit: layout, springs, and the Material 3 Expressive components |
| [`ui/mcu/`](ui/mcu) | The colour scheme: a Go port of the Material colour utilities (HCT, CAM16, tonal spot) |
| [`ui/shapes/`](ui/shapes) | The shapes: a Go port of the rounded polygons and their morphs of `androidx.graphics.shapes` |
| [`ui/platform/`](ui/platform) | The desktop: accent colour, dark mode and reduced motion from the XDG desktop portal, file choosers, links |
| [`assets/`](assets) | What the binary embeds: the icon font, the translations and the licenses (all generated at build time) |
| [`main.go`](main.go) | Assembles the model and the window |

The layers follow [the architecture](../docs/architecture.md): `core` knows
nothing of the others, `state` only knows `core`, and `ui` sits on top.
`core` and `state` are ports of the Flutter implementation and pass the same
[fixtures](../shared/fixtures/README.md).

## Requirements

- Go (pinned in [`versions.env`](versions.env)).
- The development files of Gio's native parts, a C compiler and Python 3 with
  fontTools and brotli for the icon font:

  ```sh
  sudo apt install build-essential pkg-config libwayland-dev libx11-dev \
    libx11-xcb-dev libxkbcommon-dev libxkbcommon-x11-dev libgles2-mesa-dev \
    libegl1-mesa-dev libffi-dev libxcursor-dev libxfixes-dev libvulkan-dev \
    fontconfig fonts-ubuntu fonts-noto-cjk fonttools python3-brotli
  ```

- vfox 1.0.0 or later at run time.

## Everyday commands

| Task | Command |
| --- | --- |
| Generate the icon font, the translations and the licenses the binary embeds | `tools/prepare.sh` (set `GIO_SKIP_LICENSES=1` to skip the licenses of the Go modules, which need `go mod download`) |
| Build (`build/vfox-plus`) | `tools/build.sh build/vfox-plus` |
| Run | `tools/build.sh build/vfox-plus && build/vfox-plus` |
| Format check (CI fails on a difference) | `golangci-lint fmt --diff` |
| Lint (CI fails on any finding) | `golangci-lint run` and `golangci-lint run --build-tags e2e` |
| Unit tests | `go test -race -coverprofile=coverage.out ./...` |
| Coverage gates | `tools/coverage.sh coverage.out 85` |
| `.deb` and `.tar.gz` | see [Packaging](#packaging) |

[`tools/prepare.sh`](tools/prepare.sh) needs the network the first time: it
downloads the pinned Material Symbols font (or set `VFOX_PLUS_SYMBOLS` to a
directory that has it) and cuts out the 56 icons of
[`shared/icons.json`](../shared/icons.json), keeping the fill axis, which the
user interface animates. The result is not committed; `//go:embed` in
[`assets/assets.go`](assets/assets.go) fails the build if it is missing.

The release binary is position independent (`-buildmode=pie`), stripped, and
about 12 MB.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the image brings Go,
golangci-lint, the development files, Xvfb and vfox (it extends the shared
base image, see [`shared/sandbox`](../shared/sandbox)), sees the repository
read-only, and keeps the Go module and build caches in Docker volumes.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | Formatting, golangci-lint (also with the end-to-end tag), `go vet`, the unit tests with the race detector and the coverage gates |
| `sandbox/sandbox.sh e2e` | The end-to-end scenarios, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure` | The memory use on the Home page and with the versions of Node.js listed |
| `sandbox/sandbox.sh screenshots` | Screenshots of every page |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh shell` | A shell in the sandbox |
| `sandbox/sandbox.sh clean` | Deletes the image, the cache volumes and the results |

Results are written to `dist/sandbox/gio/<command>/`. Set `SANDBOX_GOPROXY`
(for example `https://goproxy.cn`) if the default Go module proxy is slow from
where you are, and `SANDBOX_UBUNTU_MIRROR` for the Ubuntu archive.

## Tests

- **Unit tests** cover `core`, `state`, the colour scheme and the shapes. The
  gates are 85 % of the statements of `core` and `state` together and 75 % of
  those of `ui`, which the tests of its widgets and pages bring to about 87 %.
  The parsers run against **every** case in
  [`shared/fixtures`](../shared/fixtures/README.md).
- **Component and page tests** lay widgets out frame after frame without a
  window and feed them pointer events ([`ui/kit`](ui/kit/harness_test.go));
  [`ui/app_test.go`](ui/app_test.go) runs the whole application against the
  captured output of vfox and drives it through the nodes the widgets
  register, one flow per test: Home, the navigation, the SDK list and detail,
  installing, using, uninstalling, the registry, tasks, settings, the compact
  layout and the missing vfox.
- **Golden tests** compare the Go colour scheme and shapes with those of the
  JavaScript libraries the web front end uses. [`ui/mcu/testdata`](ui/mcu/testdata)
  holds the schemes of 82 seed colours in both modes, bit-exact;
  [`ui/shapes/testdata`](ui/shapes/testdata) holds the outlines of the shapes,
  which agree to within 2 % of their size. Both are generated by a script
  (`node generate.mjs > golden.json`, after `npm ci` in `../web`).
- **End-to-end scenarios** run in the built application when
  `VFOX_PLUS_E2E_SCENARIO` is set: `fake` against the fake vfox of the
  fixtures, `real` against the real vfox with the offline test plugin and its
  server, `measure` for the memory measurement and `screenshots` for the
  screenshots. They are the scenarios of the other implementations; the code
  is in [`ui/e2e*.go`](ui) and is only compiled with the build tag `e2e`:

  ```sh
  tools/build.sh --e2e build/vfox-plus-e2e
  xvfb-run -a dbus-run-session -- ../web/e2e/run.sh fake build/vfox-plus-e2e
  VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a dbus-run-session -- \
    ../web/e2e/run.sh real build/vfox-plus-e2e
  ```

  [`web/e2e/run.sh`](../web/e2e/run.sh) prepares a new, empty home directory,
  the fake vfox or the real vfox with the offline test plugin and its server,
  starts the application and reads its `e2e: …` report; the application ends
  with exit code 0 when the scenario passed. The scenarios click, fill and
  read the nodes the widgets register, on the goroutine of the window between
  two frames, so they exercise the real widgets, the state, process spawning,
  streaming, cancellation with `SIGTERM`, the settings file and vfox itself.
  [`shared/tools/screenshots.sh`](../shared/tools/screenshots.sh) runs the
  `screenshots` scenario.

## How the user interface works

Gio redraws the whole window for every frame, and so does this application:
the pages are functions of the model, built and laid out again each frame, the
way Flutter and Compose describe them. What persists between frames is kept by
the toolkit.

- **Widgets** are functions `func(kit.Ctx) layout.Dimensions`. A `Ctx` is
  Gio's layout context plus the environment: theme, state store, registry and
  the services of the window. [`layout.go`](ui/kit/layout.go) has the boxes
  of Flutter — `Row`, `Column` and `Stack` with `Expanded`, `Flexible` and
  intrinsic stretch, `Pad`, `Align`, `Sized`, `Limit`, `Wrap` — so that the
  pages read like the Flutter ones.
- **State** such as the hover, press and focus of a button, the text of a field
  or the position of a spring is kept in a store, under the widget's ID inside
  a scope (`c.Scope("row:1.0")`), and dropped when a frame does not use it.
- **Motion** is a damped spring per animated value
  ([`motion.go`](ui/kit/motion.go)) with the spatial and effects springs of
  the Material 3 Expressive motion scheme, which ask for another frame only
  while they move, so the window is idle when nothing changes. Reduced motion
  from the desktop makes every value jump. Page transitions (fade through,
  shared axis) are in [`transition.go`](ui/kit/transition.go).
- **Overlays** — menus, tooltips, dialogs and snack bars — are laid out after
  the page, at the position their anchors registered, so they sit above
  everything without the page knowing.
- **Lists** are virtual: only the items in view are laid out, their heights are
  remembered, and a jump of thousands of pixels costs the same as a step. The
  version list of Node.js has 872 entries.
- **The registry** ([`registry.go`](ui/kit/registry.go)) is a flat list of the
  nodes the widgets of a frame registered: role, name, rectangle, state, and
  what a click or an input does. The tests and the end-to-end scenarios use it
  to find and operate widgets; it is not an accessibility tree (see
  [Limits](#limits)).
- **Colours** are generated at run time from the seed colour or the accent
  colour of the desktop with the same algorithm as Flutter and the web front
  end ([`ui/mcu`](ui/mcu), tonal spot 2021), so the palettes are identical.
- **Shapes** — the avatars, the stat tiles, the swatches, the loading
  indicator — are the polygons and morphs of Material 3 Expressive, drawn as
  paths ([`ui/shapes`](ui/shapes)).
- **Text** uses the sans-serif font of the desktop, found with fontconfig:
  Ubuntu Sans on Ubuntu, with its weight axis set to the weights of the type
  scale, and Gio's fallback to the system fonts for Chinese (Noto Sans CJK).
  Without fontconfig the interface falls back to the Go fonts that ship with
  Gio. The icons are the subset of Material Symbols Rounded, whose fill axis
  the buttons animate.

## Packaging

The release binary contains everything. The shared script
[`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
packages it as `vfox-plus-gio_<version>_amd64.deb` and a `.tar.gz`, with the
same layout, desktop entry, AppStream metadata and icon as the other
implementations; the version is the one in [`VERSION`](VERSION), which a test
keeps equal to the one the application shows. The window sets the application
ID `dev.vfox.VfoxPlus`, so it matches the desktop entry on X11 and Wayland.
The licenses of the Go modules, of the Go standard library, of the ports and
of the icon font are installed as
`/usr/share/doc/vfox-plus-gio/THIRD-PARTY-LICENSES` and shown in the
application ([`NOTICE.txt`](NOTICE.txt) holds the notices of the ports; the
rest is collected by [`shared/tools/go_licenses.py`](../shared/tools/go_licenses.py),
which fails on a license that is not compatible with the Apache License 2.0).

The package depends on the libraries Gio loads (X11, xkbcommon, Wayland, EGL
and GLES, the Mesa drivers for machines without a GPU), fontconfig and a font
(`fonts-ubuntu` or `fonts-noto-core`), and recommends `fonts-noto-cjk` for
Chinese, a desktop portal, `zenity` or `kdialog` for the file choosers, and
`xdg-utils` for links. The list is in [`sandbox/run-task.sh`](sandbox/run-task.sh),
and CI installs the package into a clean Ubuntu container to prove that it is
complete.

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb with
software rendering (Mesa llvmpipe, which is most of it), a 1280 × 800 window,
one process.

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 240 MB | 215 MB |
| Versions of Node.js listed | 253 MB | 228 MB |

## Limits

- **Accessibility.** Gio does not talk to AT-SPI, so screen readers do not see
  the window. Keyboard navigation works — Tab moves between controls, Enter and
  Space activate them, Escape closes menus and dialogs — and every control has a
  name in the registry, ready for the day Gio gets an accessibility backend.
- **Window decorations.** Where the system draws the title bar (X11 window
  managers, and KDE or Sway on Wayland) the window has that one. GNOME's Mutter
  does not, and Gio would then draw a title bar of its own, in a sample theme
  that has nothing to do with this interface; so on Wayland the application
  turns that off and draws its own, in the colours of the theme
  ([`ui/caption.go`](ui/caption.go)), when the compositor leaves it to the
  window. The title is dragged to move the window, the edges resize it, and the
  buttons are those of the desktop's `button-layout` setting, which Ubuntu sets to
  minimize, maximize and close. Double-clicking the title does not maximize the
  window: Gio hands the press to the compositor for the move, so the
  application never sees it; use the button.
- **Fonts.** Chinese text needs a CJK font on the system; Noto Sans CJK has
  only the regular and bold weights, so the other weights of the type scale
  are rendered with the nearest one.
- **Input methods** work through IBus on X11 and `text-input-v3` on Wayland,
  which are the ones Gio supports.
