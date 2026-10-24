# Offline test plugin

[`testsdk.lua`](testsdk.lua) is a vfox plugin that lets the end-to-end tests run
against the **real** vfox without the network and without touching anybody's
vfox installation: `shared/e2e/run.sh` starts the test server of
[`shared/e2e/test-server.mjs`](../e2e/test-server.mjs), which implements the
protocol below, and drives the application through the `real` scenario.

## How a test uses it

1. Create an empty home directory and run vfox with `HOME` pointing to it, so
   that vfox keeps all of its state (`~/.vfox`) there.
2. Start an HTTP server on the loopback interface that implements the protocol
   below, and pass its base URL to vfox in `VFOX_PLUS_TEST_SERVER`, for example
   `http://127.0.0.1:43210`.
3. Point vfox to the test registry:
   `vfox config registry.address <base URL>/registry`.
4. Drive the application: list the registry, add `testsdk` from the registry
   (or from a copy of `testsdk.lua` with `vfox add --source`), search, install
   both versions, use one, uninstall the other, remove the plugin.

Nothing leaves the machine, and every result is deterministic.

## Server protocol

| Path | Content |
| --- | --- |
| `/registry/index.json` | `[{"name": "testsdk", "desc": "…", "homepage": "https://github.com/version-fox/vfox-plus"}]` — the homepage makes vfox list the plugin as official. |
| `/registry/testsdk.json` | The plugin manifest: `{"name": "testsdk", "version": "0.1.0", "license": "Apache-2.0", "author": "vfox-plus", "downloadUrl": "<base URL>/plugin/testsdk.lua", "minRuntimeVersion": "1.0.0"}`. |
| `/plugin/testsdk.lua` | This plugin. |
| `/sdk/testsdk-<version>.tar.gz` | The SDK: a gzip-compressed tar archive with a single directory `testsdk-<version>/` containing `bin/testsdk`. Serve it with a `Content-Length` header and in several chunks, so that vfox prints download progress. |

## What the plugin offers

| Version | Note | Bundled tools |
| --- | --- | --- |
| `2.0.0` | `LTS` | `testtool 2.0` |
| `1.0.0` | — | — |

`vfox search testsdk` therefore prints:

```text
Available versions:
 - 2.0.0 (LTS) [testtool 2.0]
 - 1.0.0
```
