# vfox+ — Flutter implementation

The Flutter implementation of vfox-plus: a Material 3 Expressive desktop
application for Linux that drives the [vfox](https://vfox.dev) command line
tool. It follows the shared specification in [`../docs`](../docs) and uses the
shared resources in [`../shared`](../shared).

## Requirements

- Flutter **3.47.5** (stable), with Dart 3.13. CI and the sandbox use exactly
  this version, pinned in [`versions.env`](versions.env).
- For Linux desktop builds: `clang cmake ninja-build pkg-config libgtk-3-dev
  liblzma-dev`; for packaging also `patchelf`, `lintian` and
  `desktop-file-utils`.
- vfox 1.0.0 or later at run time.

```sh
sudo apt-get install clang cmake ninja-build pkg-config libgtk-3-dev liblzma-dev
```

## Everyday commands

Run them from this directory. After cloning, get the dependencies and generate
the icon font, which is a build output and not committed, before anything else.

| Task | Command |
| --- | --- |
| Get dependencies | `flutter pub get` |
| Generate the icon font (after cloning and after changing `shared/icons.json`) | `dart run tool/generate_icon_font.dart` |
| Run the app | `flutter run -d linux` |
| Format | `dart format lib test tool integration_test` |
| Analyze (CI fails on infos) | `flutter analyze --fatal-infos --fatal-warnings` |
| Unit and widget tests | `flutter test` |
| Coverage (CI requires 85 %) | `flutter test --coverage && dart run tool/coverage.dart --min=85` |
| End-to-end tests on Linux | `xvfb-run -a flutter test integration_test/fake_vfox_test.dart -d linux`, one file per run (see [below](#end-to-end-tests)) |
| Release bundle | `flutter build linux --release` |
| `.deb` and `.tar.gz` | `packaging/linux/build_deb.sh` (writes to `../dist`) |
| Regenerate translations | `flutter gen-l10n` (also runs on `flutter pub get`) |

If you use a pub mirror through `PUB_HOSTED_URL`, run `flutter pub get` once
without it before committing a changed `pubspec.lock`: the lock file must only
reference `https://pub.dev`, and CI checks that.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker (rootless Docker works) and changes nothing on your machine:
the containers bring their own Flutter SDK and vfox, in the versions of
[`versions.env`](versions.env), see the repository read-only, work on a copy of
it and keep the pub packages in a Docker volume. Your Flutter SDK, pub cache,
vfox installation and home directory are neither used nor modified.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | Formatting, generated translations, analysis, tests and the coverage gate |
| `sandbox/sandbox.sh e2e` | The end-to-end tests, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container that checks the declared dependencies |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh measure` | The memory use of the release build on the Home page and with the versions of Node.js listed, measured like the other implementations, with [`tool/measure.py`](tool/measure.py) driving the application with the mouse (see [the memory comparison](../docs/memory.md#measuring-flutter)) |
| `sandbox/sandbox.sh screenshots` | Renders the screenshots of the README, which are kept on an image host (see [CONTRIBUTING](../CONTRIBUTING.md#screenshots)) |
| `sandbox/sandbox.sh fixtures` | Captures the vfox output of [`shared/fixtures`](../shared/fixtures) with the pinned vfox |
| `sandbox/sandbox.sh shell` | A shell in the sandbox, for debugging |
| `sandbox/sandbox.sh clean` | Deletes the sandbox images, the pub cache volume and the results |

Results are written to `dist/sandbox/<command>/` at the root of the repository.
The tests run with networking disabled; only downloading the pub packages,
building the image, the installation test and `fixtures` use the network. The
first run builds the image, which downloads about 2 GB. Behind a slow
connection, set `SANDBOX_UBUNTU_MIRROR` (for example
`http://mirrors.aliyun.com/ubuntu/`) and `SANDBOX_FLUTTER_STORAGE_BASE_URL`
(for example `https://storage.flutter-io.cn`).

## End-to-end tests

The integration tests run the Linux application, so they need a display; use
`xvfb-run -a` without one. `flutter test` starts a desktop application only
once per run, so run one file at a time:

| File | What it tests |
| --- | --- |
| `integration_test/fake_vfox_test.dart` | The application against a script that replays [`shared/fixtures`](../shared/fixtures) and checks the environment required by the CLI contract |
| `integration_test/real_vfox_test.dart` | Every operation against the real vfox named by `VFOX_PLUS_E2E_VFOX`, with the offline plugin of [`shared/test-plugin`](../shared/test-plugin); skipped when the variable is not set |
| `integration_test/screenshots_test.dart` | Renders the screenshots of the documentation into `VFOX_PLUS_SCREENSHOTS_DIR`; skipped when the variable is not set |

The real-vfox test cannot harm a vfox installation: vfox runs with a new
temporary `HOME` and without `VFOX_HOME`, and downloads only from a server that
the test starts on the loopback interface.

## Project layout

```text
flutter/
├── lib/
│   ├── main.dart                 Wires the real process runner and settings file
│   └── src/
│       ├── core/                 Pure Dart: models, process runner, vfox client,
│       │                         output parsers, settings — no Flutter imports
│       ├── state/                Controllers (ChangeNotifier): AppModel, task queue, …
│       ├── ui/
│       │   ├── app.dart          MaterialApp, theme, locale, task snack bars
│       │   ├── shell/            Adaptive navigation (rail or bar) and page transitions
│       │   ├── pages/            Home, SDKs (list and detail), Plugins, Tasks, Settings
│       │   ├── widgets/          Material 3 Expressive components
│       │   ├── theme/            Colour, shape and motion tokens, emphasized type
│       │   └── icons/            Generated AppIcons (Material Symbols Rounded)
│       ├── l10n/                 Generated by `flutter gen-l10n` from ../shared/i18n
│       └── version.dart          Application version (checked against pubspec.yaml)
├── test/                         Unit and widget tests
│   ├── core/                     Parsers (incl. the shared fixtures), CLI client, settings
│   ├── state/                    Task queue and controllers with a fake vfox
│   ├── ui/                       Components and whole-app flows
│   ├── support/                  Fakes and the app harness
│   └── project_test.dart         Layering, translations, version and icon checks
├── integration_test/             End-to-end tests of the Linux app, and the documentation screenshots
├── tool/                         Icon font generator, coverage gate, memory measurement
├── assets/                       Application icon; the generated icon font subset
├── linux/                        GTK runner (application ID dev.vfox.VfoxPlus)
├── packaging/linux/              Desktop entry, AppStream metadata, .deb script
├── sandbox/                      Docker sandbox: image, host script and in-container tasks
└── versions.env                  Flutter and vfox versions used by CI and the sandbox
```

The three layers and their rules are described in
[`../docs/architecture.md`](../docs/architecture.md).

## Material 3 Expressive

The app uses [`material_ui`](https://pub.dev/packages/material_ui), Flutter's
Material library maintained outside the SDK, and never imports
`package:flutter/material.dart` (a test enforces this). Icon buttons use its
Material 3 Expressive variant. The other expressive pieces are implemented in
`lib/src/ui`, following [`../docs/design-system.md`](../docs/design-system.md):

- spring motion schemes and a spring-driven builder (`theme/motion.dart`);
- the shape-morphing loading indicator and the wavy progress indicator;
- the collapsible navigation rail with its floating action button;
- connected button groups, split buttons and segmented lists that change shape
  when pressed, hovered or selected;
- shape avatars and morphing shapes based on
  [`material_new_shapes`](https://pub.dev/packages/material_new_shapes);
- emphasized typography.

## Icons

The icons listed in [`../shared/icons.json`](../shared/icons.json) are cut out of
the Material Symbols Rounded variable font, keeping its fill, weight, grade and
optical size axes, into `assets/fonts/MaterialSymbolsRounded.ttf` (about 155 KB
instead of 15 MB). To use a new icon, add it to `shared/icons.json`, run
`dart run tool/generate_icon_font.dart`, commit the regenerated
`lib/src/ui/icons/app_icons.g.dart` and use its `AppIcons` constant. The
generator uses the `font-subset` tool of the Flutter SDK (`flutter precache`
downloads it) and the font of the `material_symbols_icons` development
dependency.

The font is a build output: binary files are not committed, so CI and the
sandbox generate it before every build, and the generator produces the same
file from the same inputs. Making `material_symbols_icons` a regular dependency
instead would avoid the step, but its two unused 9 and 11 MB styles would then
be bundled.

Flutter only collects the licenses of the Dart packages the application depends
on, so the license of the font (Apache-2.0, Google) is bundled as
`assets/fonts/MaterialSymbolsRounded-LICENSE.txt` and added to the license page
by `lib/src/ui/licenses.dart`. The `.deb` lists it in its copyright file.

## Packaging

`packaging/linux/build_deb.sh` turns the release bundle into
`vfox-plus_<version>_<arch>.deb` and `vfox-plus-<version>-linux-<arch>.tar.gz`.
The `.deb` installs the bundle to `/usr/lib/vfox-plus`, the launcher
`/usr/bin/vfox-plus`, the desktop entry, the AppStream metadata and the icon, and
recommends the `vfox` package. Set `DEB_MAINTAINER="Name <email>"` for packages
that are published. CI builds both files for every change, validates them
with `desktop-file-validate` and `lintian`, and installs the `.deb` into a clean
Ubuntu 26.04 container, where the application must start with nothing but the
declared dependencies (the Flutter engine loads EGL and OpenGL ES at run time,
which `ldd` does not show).

## Memory

`sandbox/sandbox.sh measure` measures the release build in Xvfb, without a GPU,
with the fake vfox, while [`tool/measure.py`](tool/measure.py) opens the pages with
the mouse. [The memory comparison](../docs/memory.md#measuring-flutter) describes
the steps and puts the result next to the other implementations:

| Page | RSS | PSS |
| --- | ---: | ---: |
| Home | 212 MB | 172 MB |
| Available versions of Node.js (872) | 223 MB | 183 MB |

The application runs in two processes: the second, `glycin-svg`, is the helper that
gdk-pixbuf starts to load the SVG icon of the window, and has 18 MB of the RSS.

`linux/runner/main.cc` sets `FLUTTER_LINUX_RENDERER=software` and `GDK_GL=disable`,
and `my_application.cc` turns Impeller off (the engine of Flutter 3.47 stops with
"Impeller DlText cannot be drawn to a Skia canvas" otherwise): Skia draws on the
processor instead of OpenGL, which maps Mesa's software OpenGL and its compiler when
there is no real GPU. The same build took 339 MB of PSS on the Home page without
them, and 4 to 5 seconds of processor time until the page was shown, against 0.6.
The engine creates its OpenGL contexts when it starts, whatever it draws with
(`fl_opengl_manager_new` in `fl_engine.cc`), so Mesa's compiler is still part of the
172 MB. Without the SVG icon of the window, which `set_window_icon` of
`my_application.cc` loads through gdk-pixbuf and its helper, the application takes
about 18 MB less, in one process; the desktop entry gives the window its icon when
the package is installed. In six runs on an earlier day, with the sandbox images of
that day and OpenGL, the PSS was 332 to 342 MB on the Home page and 344 to 369 MB
with the versions.

Measure on the target hardware when comparing implementations, with
`awk '/^Rss:|^Pss:/' /proc/$(pidof vfox-plus)/smaps_rollup`.

Before `measure` existed, the application was measured under WSLg, which renders in
software with Mesa llvmpipe too, 10 seconds after start-up with a 1280 × 820
window, for the process of the application alone, with OpenGL: 336 MB of RSS and
320 MB of PSS on the Home page, against 254 MB and 247 MB for the counter app that
`flutter create` makes, which is what Flutter itself needed with OpenGL.
