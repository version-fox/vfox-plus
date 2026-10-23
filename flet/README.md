# vfox+ — Flet implementation

vfox-plus built with [Flet](https://flet.dev) 1.0.4 (Python) on Flutter
3.44.8: the core and the state are plain Python without a user interface
framework, and the user interface is declarative Flet components. Flet runs
the Python program inside the Flutter application, so the release is one
bundle — the Flutter engine, an embedded Python 3.14 and the program — with
no system Python needed.

Flet renders Flutter's Material widgets, not Material 3 Expressive ones.
Everything the design system needs that Flet lacks lives in a Flet extension
of this directory, [`m3e`](m3e) (Python controls, Dart widgets): the springs,
the shape library and its morphing, the loading indicator, the wavy progress
indicator, the connected button groups and split buttons built from spring
surfaces, the segmented list items, the expressive navigation rail, the
dialogs that scale in on a spring, and the Material Symbols Rounded font cut
out of the variable font. The pages compose these with Flet's own controls —
buttons, lists, cards, switches, text fields, search bars, menus, snack bars
and tooltips — themed to Material 3 Expressive in
[`src/vfox_plus/ui/theme.py`](src/vfox_plus/ui/theme.py).

| Part | What it holds |
| --- | --- |
| [`src/vfox_plus/core`](src/vfox_plus/core) | The models, the process runner, the vfox client and its parsers, the settings and the translations — no user interface framework |
| [`src/vfox_plus/state`](src/vfox_plus/state) | The task queue, the controllers and `AppModel`, observed through `ChangeNotifier` |
| [`src/vfox_plus/ui`](src/vfox_plus/ui) | The Flet components: the application, the shell, the pages, their parts, the dialogs and the end-to-end driver |
| [`src/vfox_plus/e2e`](src/vfox_plus/e2e) | The end-to-end scenarios, which the built application runs on itself |
| [`m3e`](m3e) | The Flet extension: `src/vfox_plus_m3e` (the Python controls) and `src/flutter/vfox_plus_m3e` (the Dart widgets) |
| [`tools`](tools) | `prepare.py` (the icon font, the translations and the application icon), `patch_build_template.py` (the window size), `licenses.py` and `package.sh` |
| [`tests`](tests) | The unit tests of core and state, and the interface tests, which run the whole user interface in the test process |

## Requirements

- [uv](https://docs.astral.sh/uv/) 0.13.1 (pinned in
  [`versions.env`](versions.env)): it installs the pinned Python packages and
  the Python interpreter itself.
- Flutter 3.44.8 (pinned in [`versions.env`](versions.env)): `flet build`
  needs it. The Dart side of the extension needs the same SDK as Flet's.
- The build dependencies of a Flutter Linux application, rsvg-convert for the
  application icon, and the icon font tooling:

  ```sh
  sudo apt install clang cmake ninja-build pkg-config libgtk-3-dev lld llvm binutils \
    libsecret-1-dev librsvg2-bin fonttools
  ```

- Python 3 with [fontTools](https://github.com/fonttools/fonttools): the
  icons of [`shared/icons.json`](../shared/icons.json) are cut out of
  Material Symbols Rounded with
  [`shared/tools/icon_font.py`](../shared/tools/icon_font.py), which downloads
  the pinned font once, or reads it from `VFOX_PLUS_SYMBOLS`.
- vfox 1.0.0 or later at run time.

Every runtime dependency is licensed under terms compatible with the Apache
License 2.0 (see [`docs/architecture.md`](../docs/architecture.md#dependencies)):
Flet and its Python dependencies (Apache-2.0, MIT and BSD), the Dart packages
of the extension (`material_new_shapes` MIT, `animations` and the Flutter
team packages BSD), CPython (PSF license) and the icon font (Apache-2.0,
Google). [`tools/licenses.py`](tools/licenses.py) collects their texts into
the package.

## Everyday commands

| Task | Command |
| --- | --- |
| Install the pinned Python packages | `uv sync --frozen` |
| Generate the icon font, the translations and the application icon | `python3 tools/prepare.py` |
| Patch the cached Flet build template to open at 1280 × 820 | `python3 tools/patch_build_template.py` |
| Run the application with the development client | `uv run flet run` |
| Format (CI fails on a difference) | `uv run ruff format --check .` |
| Lint (CI fails on warnings) | `uv run ruff check .` |
| Type check | `uv run mypy` |
| Unit tests | `uv run pytest` |
| Unit tests with the coverage gate (85 % of the lines of core and state) | `uv run coverage run -m pytest tests/core tests/state -q && uv run coverage report --fail-under=85 --include='*/vfox_plus/core/*,*/vfox_plus/state/*'` |
| Check the Dart extension | `dart format --set-exit-if-changed lib && flutter analyze` in `m3e/src/flutter/vfox_plus_m3e` |
| Release build (`build/linux`) | `uv run flet build linux --yes` |
| `.deb` and `.tar.gz` | `bash tools/package.sh <output directory>`, see [Packaging](#packaging) |

`flet build` renders its Flutter shell from a template that hardcodes the
Linux window size to 1280 × 720; resizing the window at run time crashes the
renderer from time to time on an X server without a window manager, such as
the Xvfb of the tests. `patch_build_template.py` therefore sets 1280 × 820 —
the default size of [the UI specification](../docs/ui-spec.md) — in the cached
template before the first frame, so the application never resizes itself at
start-up. `prepare.py` also needs the network the first time: the pinned
Material Symbols font (or set `VFOX_PLUS_SYMBOLS` to a directory that has it)
and the application icon are generated, never committed.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the image brings uv,
the Flutter SDK, the build dependencies and vfox (it extends the shared base
image, see [`shared/sandbox`](../shared/sandbox)), sees the repository
read-only and keeps the uv, Flet and Dart package caches in Docker volumes.
The embedded Python and the Flutter engine need more shared memory than Docker
gives containers by default, so the containers get 1 GB of `/dev/shm`.
Unlike the other implementations, the tasks that build keep the network:
`flet build` offers no offline mode, so only `check` runs without it.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | Formatting, type checks, the linter, the unit tests with the coverage gate, and the Dart analysis of the extension |
| `sandbox/sandbox.sh e2e` | The end-to-end scenarios, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure` | The memory use on the Home page and with the versions of Node.js listed |
| `sandbox/sandbox.sh screenshots` | Screenshots of every page |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh shell` | A shell in the sandbox |
| `sandbox/sandbox.sh clean` | Deletes the image, the cache volumes and the results |

Results are written to `dist/sandbox/flet/<command>/`.
`SANDBOX_UV_MIRROR` sets a PyPI mirror for uv, for example
`https://mirrors.aliyun.com/pypi/simple/`.

## End-to-end tests

The application runs a scenario named by `VFOX_PLUS_E2E_SCENARIO` on itself
([`src/vfox_plus/e2e`](src/vfox_plus/e2e)): it drives its own controls with
the in-process driver, checks the state of the model, reports every step as
`e2e: …` on its standard output and exits with the result. The scenarios are
the same as those of the other implementations — `fake` against the fake vfox
of the fixtures, `real` against the real vfox with the offline test plugin
and its server, `measure` for the memory measurement and `screenshots` for the
screenshots.
[`web/e2e/run.sh`](../web/e2e/run.sh) prepares a new, empty home directory,
the fake vfox or the real vfox with the offline test plugin and its server:

```sh
uv run flet build linux --yes
xvfb-run -a dbus-run-session -- ../web/e2e/run.sh fake ./build/linux/vfox-plus
VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a dbus-run-session -- \
  ../web/e2e/run.sh real ./build/linux/vfox-plus
```

They exercise the whole stack: the components, the extension, the state,
process spawning, streaming, cancellation with `SIGTERM`, the settings file
and vfox itself. [`shared/tools/screenshots.sh`](../shared/tools/screenshots.sh)
runs the `screenshots` scenario of the same build. The end-to-end code is only
reached when `VFOX_PLUS_E2E_SCENARIO` names a scenario.

## Packaging

`flet build linux` assembles the release bundle: the Flutter engine and its
plugins, the embedded Python 3.14 with the program and its packages, the icon
font and the translations. [`tools/package.sh`](tools/package.sh) packages it
with the shared script
[`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
as `vfox-plus-flet_<version>_amd64.deb` and a `.tar.gz`, with the same layout,
desktop entry, AppStream metadata and icon as the other implementations
(`/usr/lib/vfox-plus/vfox-plus`, started as `/usr/bin/vfox-plus`). The
package provides, conflicts with and replaces `vfox-plus`, so the
implementations replace each other.
[`tools/licenses.py`](tools/licenses.py) collects the licenses of the bundled
Dart packages, Python packages, Python itself and the icon font as
`/usr/share/doc/vfox-plus-flet/THIRD-PARTY-LICENSES`; the application shows
the same texts in Settings, About, Open-source licenses. The window class is
`dev.vfox.VfoxPlus`, so that the window matches the desktop entry.

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb with
software rendering, a 1280 × 800 window.

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 257 MB | 216 MB |
| Versions of Node.js listed | 266 MB | 224 MB |

The Flutter engine and the embedded interpreter share nothing with the
libraries of the desktop; compare implementations measured the same way. The
patch of the Flet build template (`tools/patch_build_template.py`, which also
sets the size of the window) makes `main` set `FLUTTER_LINUX_RENDERER=software`
and `GDK_GL=disable`: Skia draws on the processor instead of OpenGL, which maps
Mesa's software OpenGL and its compiler when there is no real GPU, and the same
build took about 340 MB of PSS on the Home page instead of 216. `flet build`
keeps the project that it generated from the template, so a cache of an older
build (the `vfox-plus-sandbox-flet-build` volume of the sandbox) has to be
deleted for a changed template to take effect. Only
the rows of the available versions that are in view are built
([`src/vfox_plus/ui/widgets/virtual_list.py`](src/vfox_plus/ui/widgets/virtual_list.py)).

## Differences from the Flutter implementation

- Menus are Flet's context and popup menus, themed to surface container with
  large corners; the application icon comes from `assets/icon.png`, generated
  from the shared brand mark.
- The open-source licenses are listed by a dialog of the Settings page that
  reads the same sources a Debian package ships, instead of Flutter's license
  page.
- The end-to-end scenarios are part of the program and run when
  `VFOX_PLUS_E2E_SCENARIO` names one, instead of a separate test package.
