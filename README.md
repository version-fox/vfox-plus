# vfox-plus

Graphical front ends for [vfox](https://vfox.dev), the cross-platform version
manager, built with the [Material 3 Expressive](https://m3.material.io/) design
language.

vfox-plus explores several GUI technologies side by side
([issue #1](https://github.com/version-fox/vfox-plus/issues/1)). Every
implementation follows the same specification and uses the same translations,
icons and test fixtures, so they look and behave alike and can be compared on
equal terms.

![vfox+ home page](https://files.catbox.moe/yjwjll.png)

| SDK versions, dark theme | Settings in Simplified Chinese |
| --- | --- |
| ![SDK detail](https://files.catbox.moe/et01kh.png) | ![Settings](https://files.catbox.moe/ijq2tt.png) |

## Implementations

| Directory | Technology | Platforms | Package | Status |
| --- | --- | --- | --- | --- |
| [`flutter/`](flutter) | Flutter 3.47 with `material_ui` | Linux desktop | `vfox-plus` | Working |
| [`tauri/`](tauri) | Tauri 2 (Rust) with the [web front end](web) | Linux desktop | `vfox-plus-tauri` | Working |
| [`wails/`](wails) | Wails 3 (Go) with the [web front end](web) | Linux desktop | `vfox-plus-wails` | Working |

[`web/`](web) is not an implementation of its own: it is the front end that
the Tauri and Wails implementations share (TypeScript, Lit and the Material 3
Expressive web components of [@m3e/web](https://github.com/matraic/m3e)). Every
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
[Flutter](https://github.com/version-fox/vfox-plus/actions/workflows/flutter.yml)
[Tauri](https://github.com/version-fox/vfox-plus/actions/workflows/tauri.yml) or
[Wails](https://github.com/version-fox/vfox-plus/actions/workflows/wails.yml),
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
web/        The web front end of the Tauri and Wails implementations
tauri/      The Tauri implementation
wails/      The Wails implementation
.github/    CI workflows: repository checks and one workflow per implementation
```

| Document | Content |
| --- | --- |
| [docs/architecture.md](docs/architecture.md) | Layers, settings file, dependencies, memory |
| [docs/vfox-cli-contract.md](docs/vfox-cli-contract.md) | How a GUI runs vfox and parses its output |
| [docs/ui-spec.md](docs/ui-spec.md) | Screens, flows and states |
| [docs/design-system.md](docs/design-system.md) | Material 3 Expressive colour, shape, type, motion and components |
| [docs/i18n.md](docs/i18n.md) | Translation catalogues and glossary |
| [docs/new-implementation.md](docs/new-implementation.md) | Checklist for another GUI technology |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

vfox-plus is licensed under the [Apache License 2.0](LICENSE). Material Symbols
are licensed under the Apache License 2.0 by Google.
