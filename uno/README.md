# vfox+ — Uno Platform implementation

vfox-plus built with [Uno Platform](https://platform.uno), which brings the
WinUI API (`Microsoft.UI.Xaml`) to .NET on other platforms. On Linux the
`net10.0-desktop` target draws with Skia into an X11 window. Uno's themes do not
look like Material 3 Expressive and this implementation does not use their
controls: it uses what WinUI provides underneath — the window, the input, the
layout panels, the text and the shapes — and everything above that is written
for this project, in C#, in one process without a web view:

| Directory | What it does |
| --- | --- |
| [`src/VfoxPlus.Core/`](src/VfoxPlus.Core) | The models, the process runner, the vfox client and the parsers of its output, the settings store, the translation catalogues with their ICU messages — no UI |
| [`src/VfoxPlus.State/`](src/VfoxPlus.State) | The observable controllers, the task queue (one state-changing command at a time, streamed output, download progress, cancellation, retry) |
| [`src/VfoxPlus.Design/`](src/VfoxPlus.Design) | What the interface is made of, without a UI type: the colour scheme (a C# port of the Material colour utilities: HCT, CAM16, tonal spot), the shapes (a port of the rounded polygons and their morphs of `androidx.graphics.shapes`), the springs and the motion scheme, the type scale, the window size classes, the placement of menus |
| [`src/VfoxPlus.Platform/`](src/VfoxPlus.Platform) | The XDG desktop portal over D-Bus: accent colour, dark mode and reduced motion, and the file chooser |
| [`src/VfoxPlus.App/`](src/VfoxPlus.App) | The application: the toolkit (`Controls/`, `Controls/Overlays/`), the theme, the motion, the shell with its navigation, the pages, the dialogs and the end-to-end agent (`E2E/`) |
| [`tests/VfoxPlus.Tests/`](tests/VfoxPlus.Tests) | The unit tests of the four libraries |
| [`tools/`](tools) | `prepare.sh` (the generated assets), `publish.sh` (the release build), `coverage.py` (the coverage gate) |
| [`sandbox/`](sandbox) | The Docker sandbox |

The layers follow [the architecture](../docs/architecture.md): `Core` knows
nothing of the others, `State` only knows `Core`, `Design` and `Platform` know
no UI, and `App` sits on top. `Core` and `State` are ports of the other
implementations and pass the same [fixtures](../shared/fixtures/README.md).

## Requirements

- The .NET SDK, pinned in [`global.json`](global.json) and
  [`versions.env`](versions.env) (the release archive from Microsoft, with its
  checksum), and the Uno SDK, whose version is the one of `Uno.Sdk` in
  `global.json`. NuGet restores everything else.
- Python 3 with fontTools and brotli, for the icon fonts, and the libraries that
  Uno and Skia load on Linux:

  ```sh
  sudo apt install libx11-6 libxext6 libxfixes3 libxi6 libxrandr2 libfontconfig1 \
    fontconfig libfreetype6 libexpat1 zlib1g libbz2-1.0 libpng16-16t64 libbrotli1 \
    libicu78 libssl3t64 fonts-ubuntu fonts-noto-cjk fonttools python3-brotli
  ```

- vfox 1.0.0 or later at run time.

## Everyday commands

Run them in this directory.

| Task | Command |
| --- | --- |
| Generate the icon fonts, the translations and the licenses the application ships | `tools/prepare.sh` |
| Restore | `dotnet restore VfoxPlus.slnx` |
| Build (warnings are errors, also those of the style rules) | `dotnet build VfoxPlus.slnx` |
| Run | `dotnet run --project src/VfoxPlus.App` |
| Format check (CI fails on a difference) | `dotnet format VfoxPlus.slnx --verify-no-changes` |
| Unit tests | `dotnet test tests/VfoxPlus.Tests` |
| Coverage gate | `dotnet test tests/VfoxPlus.Tests --collect:"XPlat Code Coverage" --results-directory TestResults --settings tests/coverlet.runsettings`, then `python3 tools/coverage.py TestResults/*/coverage.cobertura.xml 90 VfoxPlus.Core VfoxPlus.State VfoxPlus.Design VfoxPlus.Platform` |
| Release build in `build/publish` | `dotnet restore src/VfoxPlus.App --runtime linux-x64 -p:Configuration=Release`, then `tools/publish.sh build/publish` |
| `.deb` and `.tar.gz` | see [Packaging](#packaging) |

[`tools/prepare.sh`](tools/prepare.sh) needs the network the first time: it
downloads the pinned Material Symbols font (or set `VFOX_PLUS_SYMBOLS` to a
directory that has it) and cuts out the 56 icons of
[`shared/icons.json`](../shared/icons.json) into two static fonts, outlined and
filled (WinUI has no way to set the axes of a variable font, such as the fill
axis); the user interface cross-fades them to animate the fill. The result is
not committed.

The Debug and the Release configurations restore different packages: the Uno
development server (hot reload) is only in Debug, and a release build refuses
it. Restore for the configuration you build.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the image brings the
.NET SDK, the libraries, Xvfb and vfox (it extends the shared base image, see
[`shared/sandbox`](../shared/sandbox)), sees the repository read-only, and keeps
the NuGet cache in Docker volumes.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | `dotnet format`, the build with warnings as errors (also with the end-to-end agent), the unit tests with the coverage gate |
| `sandbox/sandbox.sh e2e` | The end-to-end scenarios, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure` | The memory use on the Home page and with the versions of Node.js listed |
| `sandbox/sandbox.sh screenshots` | Screenshots of every page |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh shell` | A shell in the sandbox |
| `sandbox/sandbox.sh clean` | Deletes the image, the cache volumes and the results |

Results are written to `dist/sandbox/uno/<command>/`. Set
`SANDBOX_UBUNTU_MIRROR` if the default Ubuntu archive is slow from where you
are.

## Tests

- **Unit tests** cover the four libraries, with a gate of 90 % of the lines of
  `Core`, `State`, `Design` and `Platform` together (the tests bring them to
  about 95 %). The parsers run against **every** case in
  [`shared/fixtures`](../shared/fixtures/README.md).
- **Golden tests** compare the colour scheme and the shapes with those of the
  JavaScript libraries the web front end uses.
  [`Design/testdata/golden.json`](tests/VfoxPlus.Tests/Design/testdata) holds
  the schemes of 46 seed colours (22 named ones and 24 random ones) in both modes, 48 roles each, bit-exact;
  [`shapes.json`](tests/VfoxPlus.Tests/Design/testdata) holds the outlines of
  the 16 shapes, which agree to within 2 % of their size. Both are generated by
  a script (`node generate-colors.mjs > golden.json` and
  `node generate-shapes.mjs > shapes.json`, after `npm ci` in `../web`).
- **Portal tests** start a private session bus (`dbus-daemon`) with a service
  that answers as the desktop portal does, and check what goes over the wire:
  the settings, the change signals and the file chooser, with its options.
- **End-to-end scenarios** run in the built application when
  `VFOX_PLUS_E2E_SCENARIO` is set: `fake` against the fake vfox of the
  fixtures, `real` against the real vfox with the offline test plugin and its
  server, `measure` for the memory measurement and `screenshots` for the
  screenshots. They are the scenarios of the other implementations; the code
  is in [`src/VfoxPlus.App/E2E`](src/VfoxPlus.App/E2E) and is only compiled
  with `-p:E2E=true`:

  ```sh
  tools/publish.sh --e2e build/vfox-plus-e2e
  xvfb-run -a dbus-run-session -- ../web/e2e/run.sh fake build/vfox-plus-e2e/VfoxPlus
  VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a dbus-run-session -- \
    ../web/e2e/run.sh real build/vfox-plus-e2e/VfoxPlus
  ```

  [`web/e2e/run.sh`](../web/e2e/run.sh) prepares a new, empty home directory,
  the fake vfox or the real vfox with the offline test plugin and its server,
  starts the application and reads its `e2e: …` report; the application ends
  with exit code 0 when the scenario passed. The scenarios click, fill and read
  what the visual tree shows (the names of the controls, their roles and their
  places), on the thread of the window, so they exercise the real controls, the
  state, process spawning, streaming, cancellation with `SIGTERM`, the settings
  file and vfox itself. [`shared/tools/screenshots.sh`](../shared/tools/screenshots.sh)
  runs the `screenshots` scenario. The agent works on the release build, which
  is compiled ahead of time and trimmed.

## How the user interface works

- **Everything is code.** The pages and the controls are C# classes built from
  WinUI's panels (`Grid`, `StackPanel`, `Border`, `TextBlock`, `TextBox`,
  `ItemsRepeater`, `Path`); XAML only holds the theme resources of
  [`App.xaml`](src/VfoxPlus.App/App.xaml). The controls are the Material 3
  Expressive ones: buttons, icon buttons, split buttons, button groups, the
  switch, the text field and the search bar, list groups and items, cards, the
  navigation rail and bar, the extended FAB, menus, dialogs, tooltips, snack
  bars, the avatars, the wavy progress and the loading indicator. Their names
  start with `M3` where WinUI has one of the same name.
- **Pages** derive from `PageBase`: a page shows the state of the application,
  and shows it again when the state it watches changes (changes that follow each
  other are shown at once); a page that is not on screen does it when it comes
  back. Pages are made when they are first visited and kept.
- **Colours** are generated at run time from the seed colour or the accent
  colour of the desktop with the same algorithm as the other implementations
  ([`Design/Color`](src/VfoxPlus.Design/Color), tonal spot 2021). The theme
  holds one shared brush per colour role and changes it in place, so a change of
  theme rebuilds nothing.
- **Motion** is a damped spring per animated value, with the spatial and the
  effects springs of the Material 3 Expressive motion scheme. The
  [`FrameClock`](src/VfoxPlus.App/Motion/FrameClock.cs) listens to the
  rendering of the window only while some value moves, so the application is
  idle (0 % CPU) when nothing changes. Reduced motion from the desktop makes
  every value jump. Page transitions (fade through, shared axis) are in
  [`ViewSwitcher`](src/VfoxPlus.App/Controls/ViewSwitcher.cs).
- **Overlays** — menus, tooltips, dialogs and snack bars — live in an
  [`OverlayHost`](src/VfoxPlus.App/Controls/Overlays/OverlayHost.cs) at the root
  of the window, in layers, so that they sit above everything without the page
  knowing; Escape closes menus and dialogs, and a menu opens above its anchor
  when there is no room below.
- **Lists** are virtual where they are long: the 872 versions of Node.js are
  rows that an `ItemsRepeater` makes for the part in view and recycles.
- **Shapes** — the avatars, the stat tiles, the swatches, the loading
  indicator — are the polygons and morphs of Material 3 Expressive, drawn as
  paths ([`Design/Shapes`](src/VfoxPlus.Design/Shapes)).
- **Text** uses the sans-serif font of the desktop, picked from the installed
  families through Skia's font manager: Ubuntu Sans on Ubuntu, and the system's
  fallback for Chinese (Noto Sans CJK). The icons are the two static fonts of
  `prepare.sh`; the fill of an icon animates by cross-fading them.
- **Shadows** are WinUI's `ThemeShadow` with a depth (`Translation`) per
  elevation level.
- **Rendering** is Skia on the processor (see [Memory](#memory)).

## Packaging

The release build is published self-contained (the .NET runtime is in it),
compiled ahead of time (ReadyToRun) and trimmed (`PublishTrimmed`), which
halves the start-up time (the window is up after 0.3 s and the Home page
painted after 0.5 s, instead of 0.55 s and 1 s) and takes 89 MB instead of
158 MB; the `.deb` is 27 MB. [`tools/publish.sh`](tools/publish.sh) also leaves out what a release
does not need: the debugging libraries of the runtime and the fonts of Uno's
themes, which the application does not use (the screenshots of every page are
the same pixel for pixel without them).

The shared script
[`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
packages it as `vfox-plus-uno_<version>_amd64.deb` and a `.tar.gz`, with the same
layout, desktop entry, AppStream metadata and icon as the other implementations;
the version is the one in [`VERSION`](VERSION). The window class is the
application ID `dev.vfox.VfoxPlus` (the project has a
[`Package.appxmanifest`](src/VfoxPlus.App/Package.appxmanifest), whose identity
the Uno SDK fills in), so it matches the desktop entry. The licenses of the
NuGet packages, of the .NET runtime, of the ports and of the icon font are
installed as `/usr/share/doc/vfox-plus-uno/THIRD-PARTY-LICENSES` and shown in the
application ([`NOTICE.txt`](NOTICE.txt) holds the notices of the ports; the rest
is collected by [`shared/tools/nuget_licenses.py`](../shared/tools/nuget_licenses.py),
which fails on a license that is not compatible with the Apache License 2.0).

The package depends on the libraries Uno and Skia load (X11, fontconfig,
FreeType, ICU, OpenSSL for the runtime) and a font (`fonts-ubuntu` or
`fonts-noto-core`), and recommends `fonts-noto-cjk` for Chinese, a desktop portal,
`zenity` or `kdialog` for the file choosers, and `xdg-utils` for links. The list
is in [`sandbox/run-task.sh`](sandbox/run-task.sh), and CI installs the package
into a clean Ubuntu container to prove that it is complete. (Uno needs ICU at
start-up, also with the invariant globalization of .NET that this application
uses: it cannot start without `libicu`.)

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb, a
1280 × 800 window, one process, the release build.

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 138 MB | 126 MB |
| Versions of Node.js listed | 169 MB | 157 MB |

Skia draws with the processor. Uno would otherwise draw with OpenGL, and the
graphics stack that this loads takes more memory than the application does:
with Mesa's software GL (LLVM) it was 295 MB on the Home page instead of 165 MB
for the same build, with every other setting equal. An interface this size does
not need the GPU, and the memory does not depend on the driver of the machine.
The compiled-ahead-of-time, trimmed release build takes about 20 MB less than the
one that compiles when it starts.

## Limits

- **Wayland.** Uno has no Wayland back end: the window is an X11 window, which
  runs through XWayland on a Wayland desktop. The scale of the display comes from
  the `Xft.dpi` X resource (the window is 1280 × 820 dp, and 2560 × 1640 pixels
  with `Xft.dpi: 192`) or from `UNO_DISPLAY_SCALE_OVERRIDE`; where neither is set the
  window is drawn at scale 1, and whether the desktop enlarges it depends on the
  compositor. Scales 1 and 2 have been tried.
- **Accessibility.** Uno's X11 host does not talk to AT-SPI, so screen readers do
  not see the window. Keyboard navigation works — Tab moves between controls,
  Enter and Space activate them, Escape closes menus and dialogs — and every
  control has an automation name, ready for the day Uno gets an accessibility
  back end on Linux.
- **Window decorations** are those of the window manager (the compositor, for
  an X11 window on Wayland), so the window has the title bar of the desktop and
  none of its own.
- **Fonts.** The interface uses the fonts of the desktop: Chinese text needs a
  CJK font on the system, which the package recommends (`fonts-noto-cjk`).
- **Input methods** are those that Uno's X11 host supports, IBus and Fcitx; they
  have not been tried here.
