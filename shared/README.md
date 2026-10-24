# Shared resources

Resources used by vfox-plus. The application reads them from here and never
copies them, so each one exists once.

| Path | Content | Specification |
| --- | --- | --- |
| [`i18n/`](i18n) | Translation catalogues in ARB (JSON with ICU messages): `app_en.arb` (template) and `app_zh.arb`. | [docs/i18n.md](../docs/i18n.md) |
| [`icons.json`](icons.json) | The Material Symbols Rounded icons of the user interface, by meaning. | [docs/design-system.md](../docs/design-system.md#icons) |
| [`fixtures/`](fixtures) | Captured vfox output with the expected parser results, and the script that captures it. | [fixtures/README.md](fixtures/README.md), [docs/vfox-cli-contract.md](../docs/vfox-cli-contract.md) |
| [`test-plugin/`](test-plugin) | An offline vfox plugin, and the protocol of the local server it downloads from, for end-to-end tests against the real vfox. | [test-plugin/README.md](test-plugin/README.md), [docs/vfox-cli-contract.md](../docs/vfox-cli-contract.md#testing-against-the-real-vfox) |
| [`packaging/linux/`](packaging/linux) | The desktop entry, the AppStream metadata and the icon of the application, and `build-deb.sh`, which packages the application as a `.deb` and a `.tar.gz`. | [mygo/README.md](../mygo/README.md#packaging) |
| [`e2e/`](e2e) | The end-to-end runner (`run.sh`) and the offline test server (`test-server.mjs`) of the scenarios against the fake and the real vfox. | [mygo/README.md](../mygo/README.md#tests) |
| [`tools/icon_font.py`](tools/icon_font.py) | Cuts the icons of `icons.json` out of the pinned Material Symbols Rounded variable font. | [docs/design-system.md](../docs/design-system.md#icons) |
| [`tools/measure.sh`](tools/measure.sh) | Measures the memory of the application with the fake vfox of `fixtures/`. | [docs/memory.md](../docs/memory.md#how-it-is-measured) |
| [`tools/screenshots.sh`](tools/screenshots.sh) | Takes the screenshots of [the README](../README.md) with the fake vfox of `fixtures/`. | [CONTRIBUTING.md](../CONTRIBUTING.md#screenshots) |
| [`sandbox/`](sandbox) | The base image and the scripts of the Docker sandbox of the implementation. | [sandbox/README.md](sandbox/README.md) |

Changing anything here affects the application: run its tests.
