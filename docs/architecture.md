# Architecture

vfox-plus is a graphical front end for the [vfox](https://vfox.dev) command
line tool. Several implementations, in different GUI technologies, are meant to
live side by side in this repository. They differ in their technology, but they
share one architecture, one specification and one set of resources, so that
they can be compared fairly and kept consistent.

## Repository layout

```text
vfox-plus/
├── docs/                  Specification shared by every implementation
│   ├── architecture.md    This document
│   ├── vfox-cli-contract.md  How a GUI drives vfox: commands, environment, parsing
│   ├── ui-spec.md         Screens, flows and states
│   ├── design-system.md   Material 3 Expressive tokens, components and motion
│   ├── i18n.md            Translations and terminology
│   └── new-implementation.md  Checklist for a new technology stack
├── shared/                Machine-readable resources used by every implementation
│   ├── i18n/              Translation catalogues (ARB: JSON with ICU messages)
│   ├── icons.json         Material Symbols used by the user interface
│   ├── fixtures/          Captured vfox output and the expected parser results
│   ├── test-plugin/       Offline vfox plugin for end-to-end tests against the real vfox
│   ├── packaging/linux/   Desktop entry, AppStream metadata, icon, .deb script
│   ├── tools/             The icon font tool, the license collector of Rust crates and the memory measurement
│   └── sandbox/           Shared parts of the Docker sandboxes
├── flutter/               The Flutter implementation (Linux desktop)
├── web/                   The web front end shared by tauri/ and wails/, whose core and
│                          state capacitor/ shares
├── tauri/                 The Tauri implementation
├── wails/                 The Wails implementation
├── compose/               The Compose Multiplatform implementation
├── dioxus/                The Dioxus implementation
├── capacitor/             The Capacitor implementation
├── flet/                  The Flet implementation (Python, with a Dart extension)
├── gio/                   The Gio implementation (Go)
└── <stack>/               The other implementations, one directory each
```

An implementation owns exactly one top-level directory. It reads the shared
resources from `../shared` and must not copy them, so that a translation or a
fixture is changed in one place only.

The one exception is [`web/`](../web): Tauri and Wails are both native shells
around a web view, whose user interface is "bring your own web front end".
They share one front end — the vfox client, the task queue and the user
interface in TypeScript — and each adds only its native side, which answers a
small host protocol (spawning processes, a few file operations, dialogs, links,
the clipboard and the accent colour). Their comparison is then about what
actually differs between them: the language of the native side, packaging and
memory.

The Capacitor implementation shares the parts of `web/` that no user interface
framework depends on: the core, the state, the running application, the colour
scheme and the springs of `web/src/ui`, and the end-to-end scenarios. Its user
interface is its own, in Vue on Ionic's components, and its Electron main
process answers the host protocol with the Node.js host of `web/dev-host`.

The Flet implementation shares nothing but the shared resources: its core and
state are plain Python, and its user interface is declarative Flet components.
What Flet lacks of Material 3 Expressive lives in `flet/m3e`, a Flet extension
with the Python controls and the Dart widgets that render them, and the
application drives its own end-to-end scenarios in the built bundle.

The Gio implementation shares the shared resources and the end-to-end runner
`web/e2e/run.sh` with its test server, and nothing else: its core and state are
plain Go, and its user interface is drawn by an immediate-mode toolkit of its
own (`gio/ui/kit`) on top of Gio, which only provides the window, the input,
the text shaping and the GPU renderer. The colour scheme, the shapes and the
components of Material 3 Expressive are ports and implementations in Go
(`gio/ui/mcu`, `gio/ui/shapes`, `gio/ui/kit`), and the application drives its
own end-to-end scenarios.

## Layers

Every implementation is organised in three layers. Dependencies only point
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

**Core** is plain code without a UI framework. It is the part to port first when
starting a new implementation, and it is tested against `shared/fixtures`. It
contains:

- the domain models (`InstalledSdk`, `PluginInfo`, `AvailablePlugin`,
  `AvailableVersion`, `ConfigEntry`, `UseScope`);
- a *process runner* interface with one implementation based on the platform's
  process API; tests use a scripted fake, and a future remote back end (see
  below) can provide another one;
- the *vfox client*, which builds command lines, runs read-only queries and
  parses their output, and describes state-changing operations as *commands*
  (values) instead of running them;
- the *settings store*.

**State** turns core results into observable state for the UI. The most
important piece is the *task queue*: every state-changing command (install,
uninstall, use, unuse, add, remove, update) is queued and executed **one at a
time**, because vfox does not lock its data directory. Each task streams the
output of its process, extracts the download progress, keeps a bounded log and
summarises failures. When a task finishes, the installed SDKs are reloaded.

**UI** renders the state, following [the UI specification](ui-spec.md) and
[the design system](design-system.md). Desktop integration (browser, file
choosers, clipboard, accent colour) sits behind a small *platform services*
interface so that UI tests can replace it.

In the Flutter implementation the layers are `lib/src/core`, `lib/src/state`
and `lib/src/ui`, and `test/project_test.dart` fails the build if a layer
imports a layer above it. In the web front end they are `src/core`,
`src/state` and `src/ui`, and ESLint enforces the same rule.

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
`~/.config/vfox-plus/settings.json`). Every implementation uses the same file
and keys, so users can switch implementations without losing their settings.
Unknown keys must be preserved or ignored, never rejected.

| Key                | Type            | Default      | Meaning                                                    |
| ------------------ | --------------- | ------------ | ---------------------------------------------------------- |
| `theme`            | string          | `"system"`   | `system`, `light` or `dark`.                               |
| `useSystemColor`   | boolean         | `true`       | Derive the colours from the desktop accent colour.         |
| `seedColor`        | integer (ARGB)  | `0xFFF28C1B` | Seed colour when the accent colour is not used.            |
| `language`         | string          | `"system"`   | `system`, `english` or `simplifiedChinese`.                |
| `vfoxExecutable`   | string          | absent       | Path of vfox; absent means "detect".                       |
| `extraEnvironment` | object          | `{}`         | Extra environment variables for vfox, e.g. mirrors.        |
| `railExpanded`     | boolean         | `false`      | Whether the navigation rail shows its labels.              |

The web front end keeps unknown keys when it saves the file, so settings
written by a newer implementation survive.

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

Every runtime dependency must be licensed under terms compatible with, or no
more restrictive than, the Apache License 2.0 (for example Apache-2.0, MIT,
BSD), as required by [issue #1](https://github.com/version-fox/vfox-plus/issues/1).
The one exception is code under the file-based weak copyleft of the Mozilla
Public License 2.0 that a framework cannot do without, such as `option-ext`,
which the web view crate `wry` of the Rust implementations depends on: it is
only shipped unmodified and in binary form, as the Apache Software Foundation
allows for its own products, with a note on where its source code is
([`shared/tools/cargo_licenses.py`](../shared/tools/cargo_licenses.py)
checks and collects the licenses of Rust crates).
Electron, the runtime of the Capacitor implementation, is under the MIT
license, and so is most of what it bundles, but Chromium in it contains
third-party components under other licenses, among them FFmpeg under the LGPL
2.1, which Electron links dynamically as the separate library `libffmpeg.so`.
Like the MPL code above, it is weak copyleft that the framework cannot do
without, shipped unmodified and in binary form; Electron's list of those
licenses, `LICENSES.chromium.html`, ships next to the executable.
Flet, the framework of the Flet implementation, is under the Apache License
2.0, as are its Python and Dart dependencies (MIT and BSD); the CPython it
embeds is under the PSF license and the icon font under the Apache License
2.0, all of which the application lists with the licenses of the bundled
Flutter engine.
Gio, the toolkit of the Gio implementation, is dual-licensed under the Unlicense
and the MIT license, and the other Go modules it links (the text shaper
`go-text/typesetting`, `godbus/dbus` and the `golang.org/x` modules) are under
BSD licenses, as is the Go standard library; the ports in `gio/ui/mcu` and
`gio/ui/shapes` are of libraries under the Apache License 2.0 and the MIT
license, and their notices ship with the application
([`gio/NOTICE.txt`](../gio/NOTICE.txt)). The Gio implementation has no weak
copyleft dependency.
Prefer the platform and the official libraries of a technology over third-party
packages, and document why each dependency is needed next to its declaration.

## Memory

Low memory use is a goal of the project. Implementations should:

- keep lists lazy (the version list of Node.js alone has close to 900 entries);
- ship only the icons they use (the Flutter implementation subsets the Material
  Symbols font from 15 MB to about 155 KB);
- stop animations that are not visible, and honour the platform's reduced
  motion setting;
- avoid caching data that vfox can produce again cheaply.

## Future: remote and web front ends

The issue also asks for a web UI and an Android client that connect to a
machine running vfox. The process runner and the vfox client are the seams for
that: a remote implementation of the process runner (or of the client) can
forward the same commands to an agent on the machine, and the state and UI
layers stay unchanged. The CLI contract then becomes the protocol. The
Capacitor implementation already keeps its native side in one Capacitor
plugin, whose native implementation on Android would be such an agent's
client, with the Vue and Ionic user interface unchanged.
