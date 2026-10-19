# vfox+ — Dioxus implementation

vfox-plus built with [Dioxus 0.7](https://dioxuslabs.com) for the desktop: the
whole application is Rust — the vfox client, the state in signals and the
components — and Dioxus renders the components into the web view of the
system (WebKitGTK on Linux). The components are the Material 3 Expressive web
components of [@m3e/web](https://github.com/matraic/m3e), the same as those of
the [web front end](../web) of the Tauri and Wails implementations: the
navigation rail and bar, connected button groups, split buttons, segmented
lists, menus, dialogs, the loading indicator, the wavy progress indicator and
the shape library, moving on the springs of the expressive motion scheme.

The code is organised like the [Flutter implementation](../flutter):

| Module | What it holds |
| --- | --- |
| [`core`](src/core) | Plain Rust: the models, the parsers of the output of vfox, the vfox client, the process runner (tokio), the settings and the translations (ICU messages of [`shared/i18n`](../shared/i18n)) |
| [`state`](src/state) | Signals: asynchronous resources, the task queue, the controllers and `AppModel`, which wires them together |
| [`ui`](src/ui) | The theme and the motion, the icons, the desktop integration, the bridge to the page, the shell, the pages and their components |

[`components`](components) bundles the web components, a few helpers of the
page (`window.vfoxPlus`, for snack bars, the end of dialogs, the theme and
long lists) and the icon font; [`assets/app.css`](assets/app.css) styles the
pages. The executable carries all of it, so it needs no files at run time.

## Requirements

- Rust (pinned in [`../tauri/versions.env`](../tauri/versions.env)) and
  Node.js 24 (pinned in [`../web/versions.env`](../web/versions.env)).
- The WebKitGTK development files and libxdo, which the menus of Dioxus link:

  ```sh
  sudo apt install build-essential pkg-config libwebkit2gtk-4.1-dev libgtk-3-dev libxdo-dev libssl-dev
  ```

- Python 3 with [fontTools](https://github.com/fonttools/fonttools) and brotli
  (`sudo apt install fonttools python3-brotli`): the icons of
  [`shared/icons.json`](../shared/icons.json) are cut out of Material Symbols
  Rounded with [`shared/tools/icon_font.py`](../shared/tools/icon_font.py),
  which downloads the pinned font once, or reads it from `VFOX_PLUS_SYMBOLS`.
- vfox 1.0.0 or later at run time.

## Everyday commands

| Task | Command |
| --- | --- |
| Build the assets: components, icon font, npm licenses | `npm ci && npm run build` in `components` |
| Collect the licenses of the Rust crates | `python3 ../shared/tools/cargo_licenses.py --manifest-path Cargo.toml --output assets/crate-licenses.txt` |
| Run | `cargo run` |
| Format (CI fails on a difference) | `cargo fmt` and `npm run format` in `components` |
| Clippy (pedantic; CI fails on warnings) | `cargo clippy --all-targets --all-features -- -D warnings` |
| Unit tests | `cargo test --all-features` |
| Unit tests with the coverage gate (85 % of the lines of `core` and `state`) | see [`sandbox/run-task.sh`](sandbox/run-task.sh) (`cargo llvm-cov`) |
| Release binary | `cargo build --release` (`target/release/vfox-plus`) |
| `.deb` and `.tar.gz` | see [Packaging](#packaging) |

The build fails until the assets exist: the executable includes them. The
licenses of the npm packages, the font and the Rust crates must be compatible
with the Apache License 2.0; `npm run build` and `cargo_licenses.py` fail
otherwise (see [`docs/architecture.md`](../docs/architecture.md#dependencies)).

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the images bring Rust,
Node.js, WebKitGTK and vfox (the image extends the one of
[`../tauri/sandbox`](../tauri/sandbox)), see the repository read-only and keep
the cargo and npm caches in Docker volumes.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | Formatting, Clippy, the licenses, and the unit tests with the coverage gate |
| `sandbox/sandbox.sh e2e` | The end-to-end scenarios, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure` | The memory use on the Home page and with the versions of Node.js listed |
| `sandbox/sandbox.sh screenshots` | Screenshots of every page |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh shell` | A shell in the sandbox |
| `sandbox/sandbox.sh clean` | Deletes the image, the cache volumes and the results |

Results are written to `dist/sandbox/dioxus/<command>/`.

## End-to-end tests

The scenarios of the web front end ([`../web/src/e2e/scenarios.ts`](../web/src/e2e/scenarios.ts)),
ported to Rust in [`src/ui/e2e.rs`](src/ui/e2e.rs), run in a build with the
`e2e` feature: the application drives its own page with the driver of the web
front end, checks its state and exits with the result.
[`../web/e2e/run.sh`](../web/e2e/run.sh) prepares a new, empty home directory,
the fake vfox or the real vfox with the offline test plugin and its server:

```sh
cargo build --release --features e2e
xvfb-run -a dbus-run-session -- ../web/e2e/run.sh fake target/release/vfox-plus
VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a dbus-run-session -- ../web/e2e/run.sh real target/release/vfox-plus
```

They exercise the whole stack: the components, the state, process spawning,
streaming, cancellation with `SIGTERM`, the settings file and vfox itself.
[`shared/tools/screenshots.sh`](../shared/tools/screenshots.sh) runs the `screenshots` scenario
of the same build.

## Packaging

The shared script [`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
packages the release binary as `vfox-plus-dioxus_<version>_amd64.deb` and a
`.tar.gz`, with the same layout, desktop entry, AppStream metadata and icon as
the other implementations (`/usr/lib/vfox-plus/vfox-plus`, started as
`/usr/bin/vfox-plus`). The package provides, conflicts with and replaces
`vfox-plus`, so the implementations replace each other. The licenses of the
npm packages, the font and the Rust crates are installed as
`/usr/share/doc/vfox-plus-dioxus/THIRD-PARTY-LICENSES` and shown in the
application. The program name `dev.vfox.VfoxPlus` names the window class, so
that the window matches the desktop entry.

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb with
software rendering, a 1280 × 800 window, the application and its WebKit
processes together.

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 434 MB | 258 MB |
| Versions of Node.js listed | 473 MB | 296 MB |

`main.rs` sets `WEBKIT_DISABLE_COMPOSITING_MODE=1` before the webview starts,
unless it is already set: without it, accelerated compositing maps Mesa's
software OpenGL and its compiler, which the pages of vfox+ do not need, and
the same build took 427 MB of PSS on the Home page instead of 258. Software
rendering dominates what is left; compare implementations measured the same
way. Only the rows of the available versions that are in view are rendered.
