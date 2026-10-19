# vfox-plus

Graphical front ends for [vfox](https://vfox.dev), the cross-platform version
manager, built with the [Material 3 Expressive](https://m3.material.io/) design
language.

vfox-plus explores several GUI technologies side by side
([issue #1](https://github.com/version-fox/vfox-plus/issues/1)). Every
implementation follows the same specification and uses the same translations,
icons and test fixtures, so they look and behave alike and can be compared on
equal terms. The screenshots show the Flutter implementation.

![vfox+ home page](https://iili.io/nw2Xggj.png)

| Installed versions of an SDK | Available versions, dark theme |
| --- | --- |
| ![Installed versions of nodejs](https://iili.io/nw2Wo6G.png) | ![Available versions of nodejs, dark theme](https://iili.io/nw2XmrJ.png) |

| An installation in progress | Tasks, which can be followed and cancelled |
| --- | --- |
| ![Installing a version of nodejs](https://iili.io/nw2WRuS.png) | ![Tasks](https://iili.io/nw2WG8Q.png) |

| The plugin registry | Settings in Simplified Chinese |
| --- | --- |
| ![Plugins](https://iili.io/nw2XEsn.png) | ![Settings](https://iili.io/nw2hn2f.png) |

| Home in a compact window | An SDK in a compact window |
| --- | --- |
| ![Home in a compact window](https://iili.io/nw2Xj5l.png) | ![An SDK in a compact window](https://iili.io/nw2Ghtn.png) |

## Implementations

| Directory | Technology | Platforms | Package | Status |
| --- | --- | --- | --- | --- |
| [`flutter/`](flutter) | Flutter 3.47 with `material_ui` | Linux desktop | `vfox-plus` | Working |
| [`tauri/`](tauri) | Tauri 2 (Rust) with the [web front end](web) | Linux desktop | `vfox-plus-tauri` | Working |
| [`wails/`](wails) | Wails 3 (Go) with the [web front end](web) | Linux desktop | `vfox-plus-wails` | Working |
| [`compose/`](compose) | Compose Multiplatform (Kotlin) | Linux desktop | `vfox-plus-compose` | Working |
| [`dioxus/`](dioxus) | Dioxus 0.7 (Rust) with the web components of @m3e/web | Linux desktop | `vfox-plus-dioxus` | Working |
| [`capacitor/`](capacitor) | Ionic 9 (Vue) and Capacitor 8 in an Electron shell | Linux desktop | `vfox-plus-capacitor` | Working |
| [`flet/`](flet) | Flet 1.0.4 (Python) with a Dart extension | Linux desktop | `vfox-plus-flet` | Working |
| [`gio/`](gio) | Gio 0.10 (Go), drawn with its own Material 3 Expressive toolkit | Linux desktop | `vfox-plus-gio` | Working |
| [`uno/`](uno) | Uno Platform 6 (C#, .NET 10), built with its own Material 3 Expressive controls | Linux desktop | `vfox-plus-uno` | Working |
| [`mygo/`](mygo) | mygo 0.5 (Go), with its native user interface through GTK 3 and its own Material 3 Expressive components | Linux desktop | `vfox-plus-mygo` | Working |

[`web/`](web) is not an implementation of its own: it is the front end that
the Tauri and Wails implementations share (TypeScript, Lit and the Material 3
Expressive web components of [@m3e/web](https://github.com/matraic/m3e)); the
Capacitor implementation has its own user interface on its core and state. Every
implementation ships a `.deb` for Ubuntu 26.04 and a `.tar.gz`; the packages
share the application ID `dev.vfox.VfoxPlus` and replace each other.

New implementations follow [docs/new-implementation.md](docs/new-implementation.md).

## What it does

- Shows the installed SDKs, the versions in use and statistics at a glance.
- Installs, uninstalls and selects SDK versions, globally or for a project,
  with live download progress.
- Browses the vfox plugin registry, adds plugins (also from a URL or a file),
  updates and removes them.
- Runs every change as a task that can be followed, cancelled and retried.
- Edits the vfox configuration (proxy, registry, storage, legacy version files,
  caches, timeouts) and extra environment variables such as download mirrors.
- Speaks English and Simplified Chinese; follows the desktop theme and accent
  colour.

vfox+ drives the vfox command line tool, which must be installed separately
(see the [vfox installation guide](https://vfox.dev/guides/quick-start.html)).

## Getting vfox+

Every implementation's workflow builds its packages for every change. Download
the `vfox-plus-<implementation>-linux-x64` artifact (`vfox-plus-linux-x64` for
Flutter) of the latest successful run of its workflow, for example
[Flutter](https://github.com/version-fox/vfox-plus/actions/workflows/flutter.yml),
[Tauri](https://github.com/version-fox/vfox-plus/actions/workflows/tauri.yml),
[Wails](https://github.com/version-fox/vfox-plus/actions/workflows/wails.yml),
[Compose](https://github.com/version-fox/vfox-plus/actions/workflows/compose.yml),
[Dioxus](https://github.com/version-fox/vfox-plus/actions/workflows/dioxus.yml),
[Capacitor](https://github.com/version-fox/vfox-plus/actions/workflows/capacitor.yml),
[Flet](https://github.com/version-fox/vfox-plus/actions/workflows/flet.yml),
[Gio](https://github.com/version-fox/vfox-plus/actions/workflows/gio.yml),
[Uno](https://github.com/version-fox/vfox-plus/actions/workflows/uno.yml) or
[mygo](https://github.com/version-fox/vfox-plus/actions/workflows/mygo.yml),
then:

```sh
sudo apt install ./vfox-plus*_amd64.deb   # Ubuntu 26.04
vfox-plus
```

or extract the `.tar.gz` anywhere and run `./vfox-plus`. To build from source,
see the README of the implementation; with nothing but Docker,
`<implementation>/sandbox/sandbox.sh package` builds both files into
`dist/sandbox/` without changing anything on your machine.

## Repository layout

```text
docs/       Shared specification: architecture, vfox CLI contract, UI, design system, i18n
shared/     Shared resources: translations, icons, captured vfox output (test fixtures), an offline
            test plugin, packaging files, the icon font tool and the shared parts of the sandboxes
flutter/    The Flutter implementation
web/        The web front end of the Tauri and Wails implementations, whose core
            and state the Capacitor implementation shares
tauri/      The Tauri implementation
wails/      The Wails implementation
compose/    The Compose Multiplatform implementation
dioxus/     The Dioxus implementation
capacitor/  The Capacitor implementation
flet/       The Flet implementation
gio/        The Gio implementation
uno/        The Uno implementation
mygo/       The mygo implementation
.github/    CI workflows: repository checks and one workflow per implementation
```

| Document | Content |
| --- | --- |
| [docs/architecture.md](docs/architecture.md) | Layers, settings file, dependencies, memory |
| [docs/vfox-cli-contract.md](docs/vfox-cli-contract.md) | How a GUI runs vfox and parses its output |
| [docs/ui-spec.md](docs/ui-spec.md) | Screens, flows and states |
| [docs/design-system.md](docs/design-system.md) | Material 3 Expressive colour, shape, type, motion and components |
| [docs/i18n.md](docs/i18n.md) | Translation catalogues and glossary |
| [docs/memory.md](docs/memory.md) | The memory use of the implementations, measured the same way |
| [docs/new-implementation.md](docs/new-implementation.md) | Checklist for another GUI technology |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

vfox-plus is licensed under the [Apache License 2.0](LICENSE). Material Symbols
are licensed under the Apache License 2.0 by Google.
