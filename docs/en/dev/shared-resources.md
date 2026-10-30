# Shared resources

> [简体中文](../../zh-CN/dev/shared-resources.md)

Resources used by vfox-plus. The application reads them from here and never
copies them, so each one exists once.

| Path | Content | Specification |
| --- | --- | --- |
| [`i18n/`](../../../shared/i18n) | Translation catalogues in ARB (JSON with ICU messages): `app_en.arb` (template) and `app_zh.arb`. | [docs/i18n.md](i18n.md) |
| [`icons.json`](../../../shared/icons.json) | The Material Symbols Rounded icons of the user interface, by meaning, and the npm package (version and checksum) that `vfxtool prepare` cuts their SVGs out of. | [docs/design-system.md](design-system.md#icons) |
| [`fixtures/`](../../../shared/fixtures) | Captured vfox output with the expected parser results, captured by `vfxtool capture`. | [fixtures/README.md](fixtures.md), [docs/vfox-cli-contract.md](vfox-cli-contract.md) |
| [`test-plugin/`](../../../shared/test-plugin) | An offline vfox plugin, and the protocol of the local server it downloads from, for end-to-end tests against the real vfox. | [test-plugin/README.md](test-plugin.md), [docs/vfox-cli-contract.md](vfox-cli-contract.md#testing-against-the-real-vfox) |
| [`packaging/linux/`](../../../shared/packaging/linux) | The desktop entry, the AppStream metadata and the icon of the application, packaged as a `.deb` and a `.tar.gz` by `vfxtool build-deb`. | [mygo/README.md](implementation-mygo.md#packaging) |
| [`packaging/windows/`](../../../shared/packaging/windows) | The MSIX manifest, logos and bundle layout of the application, packaged as unsigned `.msix` and `.msixbundle` for Windows 11 by `vfxtool build-msix`. | [packaging/windows/README.md](packaging-windows.md), [mygo/README.md](implementation-mygo.md#packaging) |
| [`tooling/`](../../../shared/tooling) | `vfxtool`, the single Go tool that replaces the former Python and shell scripts: the license collector, the end-to-end runner, the memory measurement, the screenshots, the offline test server, the Debian packager and the MSIX packager. | [docs/design-system.md](design-system.md#icons), [docs/memory.md](memory.md#how-it-is-measured) |
| [`sandbox/`](../../../shared/sandbox) | The base image of the Docker sandbox of the implementation (driven by `vfxtool sandbox`). | [sandbox/README.md](sandbox.md) |

Changing anything here affects the application: run its tests.
