# Contributing to vfox-plus

Thank you for helping! This repository holds several implementations of the
same application (see the [README](README.md)); most of what follows applies to
all of them.

## Ground rules

- Code comments, documentation, commit messages and pull requests are written in
  **English**. User-visible texts go through the translation catalogues in
  [`shared/i18n`](shared/i18n) and are available in English and Simplified
  Chinese.
- Every source file starts with the Apache-2.0 license header used in this
  repository. Check with
  `docker run --rm -v "$PWD:/github/workspace" apache/skywalking-eyes:0.9.0 header check`
  and add missing headers with `header fix`.
- Runtime dependencies must be licensed under terms compatible with the Apache
  License 2.0.
- Behaviour that every implementation shares — the vfox CLI contract, screens,
  design tokens, translations, icons, settings — is specified in [`docs`](docs)
  and [`shared`](shared). Change the specification together with the code.
- Files use UTF-8, LF line endings and a final newline (see
  [`.editorconfig`](.editorconfig) and [`.gitattributes`](.gitattributes)).
- Binary files are not committed. Upload screenshots and other images to an
  image host and link them (see [Screenshots](#screenshots)); generate binary
  build inputs, such as the Flutter icon font, from their sources at build
  time. CI rejects binary files.

## Workflow

1. Open an issue or comment on an existing one before large changes.
2. Create a branch from `main` and keep pull requests focused.
3. Run the checks of the implementation you changed; they are the same as in CI
   and listed in the README of each implementation. For Flutter, see
   [`flutter/README.md`](flutter/README.md#everyday-commands):

   ```sh
   cd flutter
   dart format lib test tool integration_test
   flutter analyze --fatal-infos --fatal-warnings
   flutter test --coverage && dart run tool/coverage.dart --min=85
   ```

   To run every check, the end-to-end tests and the packaging without installing
   anything but Docker, and without any risk for the vfox installation on your
   machine, use the sandbox of the implementation, for example
   `flutter/sandbox/sandbox.sh all` or `tauri/sandbox/sandbox.sh all` (see
   [shared/sandbox](shared/sandbox/README.md)).

4. Describe what changed and how you tested it in the pull request. Include
   screenshots for visual changes, in light and dark themes.

## Continuous integration

| Workflow | Runs on | Checks |
| --- | --- | --- |
| [`repository.yml`](.github/workflows/repository.yml) | every change | License headers (SkyWalking Eyes), workflows (actionlint), shell scripts (ShellCheck), shared JSON, no committed binary files |
| [`flutter.yml`](.github/workflows/flutter.yml) | `flutter/`, `shared/` | Lock file, formatting, generated translations, analysis, tests with an 85 % coverage gate, end-to-end tests on Linux under Xvfb against captured output and against the real vfox, release build with a smoke test and a memory measurement, `.deb` and `.tar.gz` packages validated with lintian |
| [`flutter-sandbox.yml`](.github/workflows/flutter-sandbox.yml) | `flutter/sandbox/`, `flutter/versions.env` | Builds the Docker sandbox and runs `sandbox.sh all` in it |
| [`web.yml`](.github/workflows/web.yml) | `web/`, `shared/` | Formatting (Prettier), lint (ESLint), type checking, unit tests with an 85 % coverage gate, the production builds |
| [`tauri.yml`](.github/workflows/tauri.yml) | `tauri/`, `web/`, `shared/` | Formatting (rustfmt), Clippy, tests, the licenses of the Rust crates, the end-to-end scenarios of the web front end in the application against captured output and against the real vfox, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`wails.yml`](.github/workflows/wails.yml) | `wails/`, `web/`, `shared/` | Formatting and lint (golangci-lint), tests, the licenses of the Go modules, the end-to-end scenarios of the web front end in the application against captured output and against the real vfox, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`compose.yml`](.github/workflows/compose.yml) | `compose/`, `shared/` | Formatting (ktlint), compilation with warnings as errors, the license check, unit tests with the coverage gate, end-to-end tests against captured output and against the real vfox, the application image with a smoke test and the memory measurement, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`dioxus.yml`](.github/workflows/dioxus.yml) | `dioxus/`, `web/e2e/`, `web/src/e2e/`, `shared/` | Formatting (rustfmt, Prettier), Clippy (pedantic), the licenses of the npm packages and the Rust crates, unit tests with the coverage gate, end-to-end tests against captured output and against the real vfox, the memory measurement, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`capacitor.yml`](.github/workflows/capacitor.yml) | `capacitor/`, `web/src/`, `web/dev-host/`, `web/e2e/`, `shared/` | Formatting (Prettier), type checking (vue-tsc), lint (ESLint), the licenses of the npm packages and of Electron, unit tests with the coverage gate, end-to-end tests against captured output and against the real vfox, the memory measurement, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`gio.yml`](.github/workflows/gio.yml) | `gio/`, `web/e2e/`, `web/src/e2e/`, `shared/` | Formatting and lint (golangci-lint, with and without the end-to-end build tag), unit tests with the coverage gates, the licenses of the Go modules, end-to-end scenarios in the application against captured output and against the real vfox, the memory measurement, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`uno.yml`](.github/workflows/uno.yml) | `uno/`, `web/e2e/`, `web/src/e2e/`, `shared/` | Formatting (`dotnet format`), the build with warnings as errors (also with the end-to-end agent), unit tests with the coverage gate, the licenses of the NuGet packages, end-to-end scenarios in the application against captured output and against the real vfox, the memory measurement, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`sandboxes.yml`](.github/workflows/sandboxes.yml) | `*/sandbox/`, `*/versions.env`, `shared/sandbox/` | Builds the Docker sandboxes of the affected implementations and runs `sandbox.sh all` in them |

Dependabot keeps the pub, npm, cargo, Go, Gradle and NuGet dependencies and the
GitHub Actions up to date; actions are pinned to commit SHAs.

## Common changes

### Texts

Add or change messages in [`shared/i18n/app_en.arb`](shared/i18n/app_en.arb)
with a `description` (and placeholder metadata), then translate them in
[`app_zh.arb`](shared/i18n/app_zh.arb). See [docs/i18n.md](docs/i18n.md),
including the glossary. Flutter regenerates its accessors with
`flutter gen-l10n`; commit the generated files.

### Icons

Add the Material Symbol to [`shared/icons.json`](shared/icons.json) under a
name that describes its meaning, then regenerate the icons of each
implementation (Flutter: `dart run tool/generate_icon_font.dart`, then commit
`lib/src/ui/icons/app_icons.g.dart`; the other implementations cut their fonts
with [`shared/tools/icon_font.py`](shared/tools/icon_font.py) when they build,
for example `npm run assets` in `web/`). Fonts are generated at build time and
not committed.

### vfox output

When vfox changes its output, recapture the fixtures with
[`shared/fixtures/capture.sh`](shared/fixtures/capture.sh) (or in a container
with `flutter/sandbox/sandbox.sh fixtures`) and follow
[`shared/fixtures/README.md`](shared/fixtures/README.md). Document the change in
[docs/vfox-cli-contract.md](docs/vfox-cli-contract.md).

### Screenshots

The images of the documentation are not committed: they are hosted on
[catbox.moe](https://catbox.moe), a free image host that needs no account, and
linked. `flutter/sandbox/sandbox.sh screenshots` renders the screenshots into
`dist/sandbox/screenshots/`, reproducibly: every page, in both themes and
languages, the compact layout and an installation in progress
([`screenshots_test.dart`](flutter/integration_test/screenshots_test.dart)). The
[README](README.md) links `home-light`, `sdks-light`, `sdks-dark`,
`installing-light`, `tasks-light`, `plugins-light`, `settings-zh`,
`compact-home` and `compact-sdks`. Upload a new image with

```sh
curl -F reqtype=fileupload -F fileToUpload=@dist/sandbox/screenshots/home-light.png https://catbox.moe/user/api.php
```

which prints its URL, and replace the link in the README. Every upload gets a
new URL, so cached copies of an old image never show. Anonymous uploads cannot
be deleted; upload nothing that must not stay public.

### A new implementation

Follow [docs/new-implementation.md](docs/new-implementation.md).
