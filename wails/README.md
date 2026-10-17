# vfox+ — Wails implementation

vfox-plus built with [Wails 3](https://v3.wails.io): a Go application that
shows the [web front end](../web) in the web view of the system (WebKitGTK 6.0
with GTK 4 on Linux). The front end holds the vfox client, the task queue and
the user interface; the Go side answers its
[host protocol](../web/src/hosts/protocol.ts) through one bound method,
`host.Service.Call`, and sends the output of processes as `vfox-plus:process`
events:

| File | What it does |
| --- | --- |
| [`internal/host/process.go`](internal/host/process.go) | Spawns vfox with an argument vector, a complete environment and a closed stdin; streams its output in order and cancels it with `SIGTERM` |
| [`internal/host/files.go`](internal/host/files.go) | The settings file (written atomically), the locator of vfox, copies of local plugins |
| [`internal/host/accent.go`](internal/host/accent.go) | The accent colour from the XDG desktop portal |
| [`internal/host/host.go`](internal/host/host.go) | The bound service, which dispatches the requests |
| [`main.go`](main.go) | The application and its window; file choosers, links and the clipboard through the managers of Wails |

Tauri and Wails share the front end on purpose (see
[`../docs/architecture.md`](../docs/architecture.md)): comparing them compares
their native sides, packaging and memory.

## Requirements

- Go (pinned in [`versions.env`](versions.env)) and Node.js 24 (pinned in
  [`../web/versions.env`](../web/versions.env)).
- The GTK 4 and WebKitGTK 6.0 development files:

  ```sh
  sudo apt install build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev
  ```

- Python 3 with fontTools and brotli for the icon font of the front end (see
  [`../web/README.md`](../web/README.md#requirements)).
- vfox 1.0.0 or later at run time.

## Everyday commands

| Task | Command |
| --- | --- |
| Prepare the front end | `npm --prefix ../web ci && npm --prefix ../web run assets` |
| Build (front end and binary, `build/vfox-plus`) | `./build.sh` |
| Run | `./build.sh && build/vfox-plus` |
| Format check (CI fails on a difference) | `golangci-lint fmt --diff` |
| Lint (CI fails on any finding) | `golangci-lint run` |
| Unit tests | `go test -race ./...` |
| `.deb` and `.tar.gz` | see [Packaging](#packaging) |

[`build.sh`](build.sh) builds the front end with `npm run build:wails`, copies
it into `frontend/dist` (not committed), where `main.go` embeds it, and builds
the binary with the `production` tag, which removes the developer tools of
Wails. The Wails CLI is not needed. To work on the user interface, run the
front end in a browser with `npm run dev` in `../web`, which is faster.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | Release build, formatting, golangci-lint and the unit tests of the Go side |
| `sandbox/sandbox.sh e2e` | The end-to-end scenarios of the front end in the application, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure` | The memory use on the Home page and with the versions of Node.js listed |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh shell` | A shell in the sandbox |
| `sandbox/sandbox.sh clean` | Deletes the image, the cache volumes and the results |

Results are written to `dist/sandbox/wails/<command>/`. Set `SANDBOX_GOPROXY`
(for example `https://goproxy.cn`) if the default Go module proxy is slow from
where you are.

WebKitGTK 6.0 confines its web processes with bubblewrap, which needs user
namespaces. Containers and some CI runners do not have them, so the sandbox
and CI set `WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1`; on a desktop the
confinement stays on.

## End-to-end tests

The scenarios of the front end ([`../web/README.md`](../web/README.md#end-to-end-tests))
run in a build whose front end contains the test runner:

```sh
./build.sh --e2e build/vfox-plus-e2e
xvfb-run -a dbus-run-session -- ../web/e2e/run.sh fake build/vfox-plus-e2e
VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a dbus-run-session -- ../web/e2e/run.sh real build/vfox-plus-e2e
```

They exercise the whole stack, including the Go side: process spawning,
streaming through events, cancellation with `SIGTERM`, file operations and the
settings file.

## Packaging

The release binary contains everything, front end included. The shared script
[`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
packages it as `vfox-plus-wails_<version>_amd64.deb` and a `.tar.gz`, with the
same layout, desktop entry, AppStream metadata and icon as the other
implementations; the version is the one of the front end. The window sets the
application ID `dev.vfox.VfoxPlus` as its program name, so it matches the
desktop entry on X11 and Wayland. The licenses of the bundled npm packages, of
the icon font, of the Go modules and of the Go standard library are installed
as `/usr/share/doc/vfox-plus-wails/THIRD-PARTY-LICENSES` and shown in the
application: [`build.sh`](build.sh) adds those of Go, checked by
[`shared/tools/go_licenses.py`](../shared/tools/go_licenses.py), to the licenses
of the front end.

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb with
software rendering, a 1280 × 800 window, the application and its WebKit
processes together.

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 686 MB | 480 MB |
| Versions of Node.js listed | 758 MB | 554 MB |

WebKitGTK 6.0 with GTK 4 needs about 80 MB more than the WebKitGTK 4.1 with
GTK 3 of the [Tauri implementation](../tauri/README.md#memory) for the same
front end. Software rendering dominates these numbers; compare implementations
measured the same way.
