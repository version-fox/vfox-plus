# vfox+ — Tauri implementation

vfox-plus built with [Tauri 2](https://tauri.app): a Rust application that shows
the [web front end](../web) in the web view of the system (WebKitGTK on
Linux). The front end holds the vfox client, the task queue and the user
interface; the Rust side, in [`src-tauri`](src-tauri), answers its
[host protocol](../web/src/hosts/protocol.ts):

| Module | What it does |
| --- | --- |
| [`process.rs`](src-tauri/src/process.rs) | Spawns vfox with an argument vector, a complete environment and a closed stdin; streams its output in order and cancels it with `SIGTERM` |
| [`files.rs`](src-tauri/src/files.rs) | The settings file (written atomically), the locator of vfox, copies of local plugins |
| [`desktop.rs`](src-tauri/src/desktop.rs) | The accent colour from the XDG desktop portal, file choosers, links, the clipboard |
| [`host.rs`](src-tauri/src/host.rs) | The commands `host_call` and `host_start`, which dispatch the requests |

## Requirements

- Rust (pinned in [`versions.env`](versions.env)) and Node.js 24 (pinned in
  [`../web/versions.env`](../web/versions.env)).
- The WebKitGTK development files:

  ```sh
  sudo apt install build-essential pkg-config libwebkit2gtk-4.1-dev libgtk-3-dev librsvg2-dev libssl-dev
  ```

- Python 3 with fontTools and brotli for the icon font of the front end (see
  [`../web/README.md`](../web/README.md#requirements)).
- vfox 1.0.0 or later at run time.

## Everyday commands

| Task | Command |
| --- | --- |
| Install the packages (here and in `../web`) | `npm ci && npm --prefix ../web ci && npm --prefix ../web run assets` |
| Generate the icons from the shared SVG | `npm run icons` |
| Run in development | `npx tauri dev` |
| Format the Rust code | `cargo fmt` in `src-tauri` |
| Clippy (CI fails on warnings) | `cargo clippy --all-targets -- -D warnings` in `src-tauri` |
| Unit tests | `cargo test` in `src-tauri` |
| Release binary | `npx tauri build --no-bundle` (`src-tauri/target/release/vfox-plus`) |
| `.deb` and `.tar.gz` | see [Packaging](#packaging) |

`npx tauri build` builds the front end first (`npm run build:tauri` in
`../web`), because the binary embeds it. The icons in `src-tauri/icons` are
generated from [`shared/packaging/linux/dev.vfox.VfoxPlus.svg`](../shared/packaging/linux)
and not committed.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the images bring Rust,
Node.js, WebKitGTK and vfox, see the repository read-only and keep the cargo
and npm caches in Docker volumes.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | Formatting, Clippy and the unit tests of the Rust side |
| `sandbox/sandbox.sh e2e` | The end-to-end scenarios of the front end in the application, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure` | The memory use on the Home page and with the versions of Node.js listed |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh shell` | A shell in the sandbox |
| `sandbox/sandbox.sh clean` | Deletes the image, the cache volumes and the results |

Results are written to `dist/sandbox/tauri/<command>/`. The front end itself is
checked by [`../web/sandbox/sandbox.sh`](../web/sandbox/sandbox.sh).

## End-to-end tests

The scenarios of the front end ([`../web/README.md`](../web/README.md#end-to-end-tests))
run in a build of the application whose front end was built with
`npm run build:e2e:tauri`:

```sh
npx tauri build --no-bundle --config '{"build": {"frontendDist": "../../web/dist/tauri-e2e", "beforeBuildCommand": ""}}'
xvfb-run -a dbus-run-session -- ../web/e2e/run.sh fake src-tauri/target/release/vfox-plus
VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a dbus-run-session -- ../web/e2e/run.sh real src-tauri/target/release/vfox-plus
```

They exercise the whole stack, including the Rust side: process spawning,
streaming, cancellation with `SIGTERM`, file operations and the settings file.

## Packaging

The release binary contains everything, front end included. The shared script
[`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
packages it as `vfox-plus-tauri_<version>_amd64.deb` and a `.tar.gz`, with the
same layout, desktop entry, AppStream metadata and icon as the other
implementations (`/usr/lib/vfox-plus/vfox-plus`, started as `/usr/bin/vfox-plus`).
The package provides, conflicts with and replaces `vfox-plus`, so the
implementations replace each other. Tauri's own bundler is not used because it
names the desktop entry after the product and cannot produce this layout. The
licenses of the bundled npm packages and of the icon font are installed as
`/usr/share/doc/vfox-plus-tauri/THIRD-PARTY-LICENSES` and shown in the
application.

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb with
software rendering, a 1280 × 800 window, the application and its WebKit
processes together.

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 598 MB | 397 MB |
| Versions of Node.js listed | 683 MB | 482 MB |

Software rendering dominates these numbers; compare implementations measured
the same way.
