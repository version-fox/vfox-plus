# Architecture

vfox-plus is a graphical front end for the [vfox](https://vfox.dev) command
line tool. It is built with [mygo](https://github.com/egoist/mygo), the
desktop application framework for Go, and its native user interface: no web
view, no HTML, no JavaScript. mygo draws the window itself, from elements that
are laid out with flexbox and painted with its own renderer, and opens it
through GTK 3 without a C compiler.

## Repository layout

```text
vfox-plus/
├── docs/                  Specification: architecture, vfox CLI contract, UI, design system, i18n
│   ├── architecture.md    This document
│   ├── vfox-cli-contract.md  How the GUI drives vfox: commands, environment, parsing
│   ├── ui-spec.md         Screens, flows and states
│   ├── design-system.md   Material 3 Expressive tokens, components and motion
│   ├── i18n.md            Translations and terminology
│   └── memory.md          The memory use of the application, measured the same way
├── shared/                Machine-readable resources used by the application
│   ├── i18n/              Translation catalogues (ARB: JSON with ICU messages)
│   ├── icons.json         Material Symbols used by the user interface
│   ├── fixtures/          Captured vfox output and the expected parser results
│   ├── test-plugin/       Offline vfox plugin for end-to-end tests against the real vfox
│   ├── packaging/linux/   Desktop entry, AppStream metadata and icon, packaged by `vfxtool build-deb`
│   ├── tooling/           `vfxtool`, the Go tool that builds, tests and packages everything
│   └── sandbox/           Shared parts of the Docker sandbox
├── mygo/                  The application (Go)
└── .github/               CI workflows: repository checks, the build and the memory report
```

The application reads the shared resources from `../shared` and never copies
them, so that a translation or a fixture is changed in one place only.

## Layers

The application is organised in three layers. Dependencies only point
downwards.

```text
┌────────────────────────────────────────────────────────────────────┐
│ UI         pages, components, theme, motion, platform integration  │
├────────────────────────────────────────────────────────────────────┤
│ State      observable controllers: settings, vfox availability,    │
│            installed SDKs, registry, configuration, task queue     │
├────────────────────────────────────────────────────────────────────┤
│ Core       models, process runner, vfox client and output parsers,  │
│            settings store — no UI framework                        │
└────────────────────────────────────────────────────────────────────┘
                                 │ child processes
                                 ▼
                          vfox command line
```

**Core** is plain Go without a UI framework. It contains:

- the domain models (`InstalledSdk`, `PluginInfo`, `AvailablePlugin`,
  `AvailableVersion`, `ConfigEntry`, `UseScope`);
- a *process runner* interface with one implementation based on the platform's
  process API; tests use a scripted fake, and a future remote back end (see
  below) can provide another one;
- the *vfox client*, which builds command lines, runs read-only queries and
  parses their output, and describes state-changing operations as *commands*
  (values) instead of running them;
- the *settings store*;
- the *translation catalogues* with their ICU messages (`core/i18n`).

It is tested against `shared/fixtures`.

**State** turns core results into observable state for the UI. The most
important piece is the *task queue*: every state-changing command (install,
uninstall, use, unuse, add, remove, update) is queued and executed **one at a
time**, because vfox does not lock its data directory. Each task streams the
output of its process, extracts the download progress, keeps a bounded log and
summarises failures. When a task finishes, the installed SDKs are reloaded.

**UI** renders the state, following [the UI specification](ui-spec.md) and
[the design system](design-system.md). Desktop integration (browser, file
choosers, clipboard, accent colour) sits behind a small *platform services*
interface so that UI tests can replace it. The Material 3 Expressive
components live in `ui/kit`, built on mygo's elements and headless widgets;
the colour scheme (`ui/mcu`, a Go port of the Material colour utilities) and
the shapes (`ui/shapes`, a Go port of `androidx.graphics.shapes`) are plain Go
without a UI framework, like `core` and `state`.

In the sources the layers are `mygo/core`, `mygo/state` and `mygo/ui`
(`ui/kit`, `ui/mcu` and `ui/shapes` are part of the UI layer); `core` knows
nothing of the others, and `state` only knows `core`.

## Talking to vfox

The GUI never reads or writes vfox's files directly; everything goes through the
command line, as described in [the CLI contract](vfox-cli-contract.md). This
keeps the GUI correct across vfox releases and lets vfox keep its invariants.
The contract also lists the quirks of the current vfox output that parsers must
cope with.

Read-only queries run concurrently and are cheap (about 10 ms each); only
`vfox search` and `vfox available` may touch the network.

## Settings

Settings of the GUI itself are stored as JSON in
`$XDG_CONFIG_HOME/vfox-plus/settings.json` (by default
`~/.config/vfox-plus/settings.json`).

| Key                | Type            | Default      | Meaning                                                    |
| ------------------ | --------------- | ------------ | ---------------------------------------------------------- |
| `theme`            | string          | `"system"`   | `system`, `light` or `dark`.                               |
| `useSystemColor`   | boolean         | `true`       | Derive the colours from the desktop accent colour.         |
| `seedColor`        | integer (ARGB)  | `0xFFF28C1B` | Seed colour when the accent colour is not used.            |
| `language`         | string          | `"system"`   | `system`, `english` or `simplifiedChinese`.                |
| `vfoxExecutable`   | string          | absent       | Path of vfox; absent means "detect".                       |
| `extraEnvironment` | object          | `{}`         | Extra environment variables for vfox, e.g. mirrors.        |
| `railExpanded`     | boolean         | `false`      | Whether the navigation rail shows its labels.              |

Unknown keys must be preserved or ignored, never rejected.

The vfox configuration (proxy, registry, storage directory, …) is **not** part
of these settings: it belongs to vfox and is edited through `vfox config`.

## Locating vfox

If no executable is configured, the GUI searches `PATH`, then the usual install
locations that a desktop launcher's `PATH` often lacks: `/usr/local/bin`,
`/usr/bin`, `~/.local/bin`, `~/bin`, `~/.linuxbrew/bin`,
`/home/linuxbrew/.linuxbrew/bin` and `/snap/bin`. The user can also pick the
executable. If vfox is missing or broken, the GUI explains how to install it
and keeps the settings reachable.

## Dependencies

Every runtime dependency must fall in Category A or B of the license
classification of the Apache Software Foundation
(see https://www.apache.org/legal/resolved): what the ASF allows in its
products may be used here — permissive licenses (for example Apache-2.0,
MIT, BSD, Unlicense) or weak-copyleft licenses (for example MPL-2.0,
EPL-2.0). Category X (strong copyleft such as GPL and AGPL) is excluded.
Category B dependencies ship under their own license and its conditions
(their notices stay intact and their source stays obtainable); the
application links them unmodified. mygo, the framework of the application, is under the MIT license, and
the Go modules it links are purego under the Apache License 2.0, the text
shaper `go-text/typesetting` under the Unlicense or a BSD license, and the
`golang.org/x` modules under BSD licenses, as is the Go standard library; the
ports in `mygo/ui/mcu` and `mygo/ui/shapes` are of libraries under the Apache
License 2.0 and the MIT license, and their notices ship with the application
([`mygo/NOTICE.txt`](../mygo/NOTICE.txt)). The application does not link GTK 3,
GLib, Pango and Cairo, which are under the LGPL: mygo loads them from the system
when it starts, and the package depends on them as declared in
`vfxtool runtask` (`shared/tooling/internal/runtask`). The application
ships no strong-copyleft (Category X) code; weak-copyleft (Category B)
dependencies, if any, ship under their own license terms. The build tooling
in `shared/tooling` (`vfxtool`, never shipped in the application) additionally
uses `go-github` (the GitHub API for the memory report),
`nfpm` (Debian packaging as a library), `andybalholm/brotli` (WOFF2
compression for the icon font), `go-licenses` (license classification for
the license collector) and `go-getter` (resumable downloads for the package
prefetch), all Category A.
Prefer the platform and the official libraries of a technology over third-party
packages, and document why each dependency is needed next to its declaration.

## Memory

Low memory use is a goal of the project. The application:

- keeps lists lazy (the version list of Node.js alone has close to 900
  entries);
- ships only the icons it uses (the two icon font subsets total about 36 KB);
- stops animations that are not visible, and honours the platform's reduced
  motion setting;
- avoids caching data that vfox can produce again cheaply.

[The memory page](memory.md) records what the application uses.

## Future: remote front ends

A web UI and an Android client that connect to a machine running vfox remain
possible. The process runner and the vfox client are the seams for that: a
remote implementation of the process runner (or of the client) can forward the
same commands to an agent on the machine, and the state and UI layers stay
unchanged. The CLI contract then becomes the protocol.
