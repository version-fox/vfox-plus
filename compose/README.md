# vfox+ — Compose Multiplatform implementation

vfox-plus built with [Compose Multiplatform](https://www.jetbrains.com/compose-multiplatform/)
for the desktop: Kotlin on the JVM, drawn with Skia, with the Material 3
Expressive components of `org.jetbrains.compose.material3` — the wide
navigation rail, connected button groups, split buttons, segmented list items,
the loading indicator, the wavy progress indicator and the shape library —
themed with the expressive motion scheme.

The code is organised like the [Flutter implementation](../flutter):

| Package | What it holds |
| --- | --- |
| [`core`](src/main/kotlin/dev/vfox/plus/core) | Pure Kotlin: the models, the parsers of the output of vfox, the vfox client, the process runner, the settings and the translations (ICU messages of [`shared/i18n`](../shared/i18n)) |
| [`state`](src/main/kotlin/dev/vfox/plus/state) | Snapshot state with coroutines: asynchronous resources, the task queue, the controllers and `AppModel`, which wires them together |
| [`ui`](src/main/kotlin/dev/vfox/plus/ui) | The theme, the icons, the transitions, the shell, the pages and their components |

The state changes on the event dispatch thread, where Compose runs; processes
are read on background threads and hand their output over in order.

## Requirements

- A JDK and Gradle, pinned in [`versions.env`](versions.env). The build has no
  Gradle wrapper, whose JAR would be a binary in the repository.
- Python 3 with [fontTools](https://github.com/fonttools/fonttools)
  (`sudo apt install fonttools`): the build cuts the icons of
  [`shared/icons.json`](../shared/icons.json) out of Material Symbols Rounded
  with [`shared/tools/icon_font.py`](../shared/tools/icon_font.py), which
  downloads the pinned font into `.gradle/material-symbols` once, or reads it
  from `VFOX_PLUS_SYMBOLS`.
- vfox 1.0.0 or later at run time.

## Everyday commands

| Task | Command |
| --- | --- |
| Run | `gradle run` |
| Format (ktlint) | `gradle spotlessApply` |
| Check the formatting (CI fails on a difference) | `gradle spotlessCheck` |
| Unit tests with the coverage gate (85 % of the lines of `core` and `state`) | `gradle test koverVerify` |
| End-to-end tests | `gradle e2eTest`, see [End-to-end tests](#end-to-end-tests) |
| Screenshots of every page (`build/screenshots`) | `gradle screenshots` |
| Licenses of the bundled libraries | `gradle checkLicense` |
| Application image (`build/compose/binaries/main/app/vfox-plus`) | `gradle createDistributable` |
| `.deb` and `.tar.gz` | see [Packaging](#packaging) |

The compiler is the linter: every warning fails the build. The libraries are
pinned in [`gradle/libs.versions.toml`](gradle/libs.versions.toml);
[`allowed-licenses.json`](allowed-licenses.json) lists the licenses that
`checkLicense` accepts, all of them compatible with the Apache License 2.0.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the image brings the
JDK, Gradle and vfox, sees the repository read-only and keeps the Gradle cache
in a Docker volume.

| Command | What it does |
| --- | --- |
| `sandbox/sandbox.sh check` | Formatting, compilation, the license check and the unit tests with the coverage gate |
| `sandbox/sandbox.sh e2e` | The end-to-end tests, against the fixtures and against the real vfox |
| `sandbox/sandbox.sh package` | The application image with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure` | The memory use on the Home page and with the versions of Node.js listed |
| `sandbox/sandbox.sh screenshots` | Screenshots of every page |
| `sandbox/sandbox.sh all` | `check`, `e2e` and `package` |
| `sandbox/sandbox.sh shell` | A shell in the sandbox |
| `sandbox/sandbox.sh clean` | Deletes the image, the cache volume and the results |

Results are written to `dist/sandbox/compose/<command>/`.

## End-to-end tests

The end-to-end tests ([`src/test/kotlin/dev/vfox/plus/e2e`](src/test/kotlin/dev/vfox/plus/e2e))
run the whole application — the user interface, the state, the vfox client
and real vfox processes — in the test window of Compose, with a new home
directory, and drive it as a user would, by the texts and names on the
screen:

- `FakeVfoxTest` against [`shared/fixtures/fake-vfox.sh`](../shared/fixtures/fake-vfox.sh),
  which replays the captured output of vfox and fails when the application
  breaks [the CLI contract](../docs/vfox-cli-contract.md);
- `RealVfoxTest` against the vfox that `VFOX_PLUS_E2E_VFOX` names, with the
  offline plugin of [`shared/test-plugin`](../shared/test-plugin) and its
  local server (`TestServer.kt`): the registry, adding plugins from the
  registry and from a local file, installing with progress, cancelling,
  using, uninstalling and removing. Without `VFOX_PLUS_E2E_VFOX` it is
  skipped.

```sh
VFOX_PLUS_E2E_VFOX=/path/to/vfox gradle e2eTest
```

The application image itself is started by the smoke tests of the sandbox
and CI, and by the memory measurement.

## Packaging

`gradle createDistributable` builds an application image with jpackage: the
launcher `bin/vfox-plus`, the application JARs and a Java runtime reduced
with jlink to the modules the application needs. The shared script
[`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
packages the image as `vfox-plus-compose_<version>_amd64.deb` and a `.tar.gz`,
with the same layout, desktop entry, AppStream metadata and icon as the other
implementations. `Main.kt` names the window class `dev.vfox.VfoxPlus`, so that
the window matches the desktop entry; the JVM runs with a small heap and the
serial collector.

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb with
software rendering, a 1280 × 800 window.

| State | RSS | PSS |
| --- | ---: | ---: |
| Home page | 290 MB | 266 MB |
| Versions of Node.js listed | 301 MB | 277 MB |

The JVM runs with a heap of at most 256 MB and the serial collector; Skia
draws the window itself, so there are no processes of a web view. Compare
implementations measured the same way.
