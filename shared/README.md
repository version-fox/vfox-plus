# Shared resources

> [简体中文](README.zh-CN.md)

Resources used by vfox-plus. The application reads them from here and never
copies them, so each one exists once.

| Path | Content | Specification |
| --- | --- | --- |
| [`i18n/`](i18n) | Translation catalogues in ARB (JSON with ICU messages): `app_en.arb` (template) and `app_zh.arb`. | [docs/i18n.md](../docs/i18n.md) |
| [`icons.json`](icons.json) | The Material Symbols Rounded icons of the user interface, by meaning. | [docs/design-system.md](../docs/design-system.md#icons) |
| [`fixtures/`](fixtures) | Captured vfox output with the expected parser results, captured by `vfxtool capture`. | [fixtures/README.md](fixtures/README.md), [docs/vfox-cli-contract.md](../docs/vfox-cli-contract.md) |
| [`test-plugin/`](test-plugin) | An offline vfox plugin, and the protocol of the local server it downloads from, for end-to-end tests against the real vfox. | [test-plugin/README.md](test-plugin/README.md), [docs/vfox-cli-contract.md](../docs/vfox-cli-contract.md#testing-against-the-real-vfox) |
| [`packaging/linux/`](packaging/linux) | The desktop entry, the AppStream metadata and the icon of the application, packaged as a `.deb` and a `.tar.gz` by `vfxtool build-deb`. | [mygo/README.md](../mygo/README.md#packaging) |
| [`packaging/windows/`](packaging/windows) | The MSIX manifest, logos and bundle layout of the application, packaged as unsigned `.msix` and `.msixbundle` for Windows 11 by `vfxtool build-msix`. | [packaging/windows/README.md](packaging/windows/README.md), [mygo/README.md](../mygo/README.md#packaging) |
| [`tooling/`](tooling) | `vfxtool`, the single Go tool that replaces the former Python and shell scripts: the icon font cutter, the license collector, the end-to-end runner, the memory measurement, the screenshots, the offline test server, the Debian packager and the MSIX packager. | [docs/design-system.md](../docs/design-system.md#icons), [docs/memory.md](../docs/memory.md#how-it-is-measured) |
| [`sandbox/`](sandbox) | The base image of the Docker sandbox of the implementation (driven by `vfxtool sandbox`). | [sandbox/README.md](sandbox/README.md) |

Changing anything here affects the application: run its tests.
