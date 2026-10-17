# Shared resources

Resources used by every implementation of vfox-plus. Implementations read them
from here and never copy them, so each one exists once.

| Path | Content | Specification |
| --- | --- | --- |
| [`i18n/`](i18n) | Translation catalogues in ARB (JSON with ICU messages): `app_en.arb` (template) and `app_zh.arb`. | [docs/i18n.md](../docs/i18n.md) |
| [`icons.json`](icons.json) | The Material Symbols Rounded icons of the user interface, by meaning. | [docs/design-system.md](../docs/design-system.md#icons) |
| [`fixtures/`](fixtures) | Captured vfox output with the expected parser results, and the script that captures it. | [fixtures/README.md](fixtures/README.md), [docs/vfox-cli-contract.md](../docs/vfox-cli-contract.md) |
| [`test-plugin/`](test-plugin) | An offline vfox plugin, and the protocol of the local server it downloads from, for end-to-end tests against the real vfox. | [test-plugin/README.md](test-plugin/README.md), [docs/vfox-cli-contract.md](../docs/vfox-cli-contract.md#testing-against-the-real-vfox) |

Changing anything here affects every implementation: run all of their tests.
