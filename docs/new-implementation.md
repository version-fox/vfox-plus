# Adding an implementation

[Issue #1](https://github.com/version-fox/vfox-plus/issues/1) compares several
GUI technologies (for example Gio, Tauri, Dioxus, Flutter, Capacitor with Ionic,
Uno, Flet, Compose Multiplatform and Wails). This checklist keeps a new
implementation consistent with the existing ones, so the comparison stays fair
and users get the same application whatever it is built with.

## 1. Set up the directory

- Create one top-level directory named after the technology, for example
  `tauri/`. Everything specific to it stays inside: sources, build files, tests,
  packaging and a `README.md`.
- Every runtime dependency must be licensed under terms compatible with the
  Apache License 2.0. Record why each dependency is needed.
- Do not commit binary files: link screenshots from an image host and generate
  binary build inputs, such as an icon font subset, at build time.
- Every source file carries the Apache-2.0 header (see
  [`.licenserc.yaml`](../.licenserc.yaml)); generated files are listed in its
  `paths-ignore`.

## 2. Port the core

Follow [the architecture](architecture.md): a core without UI framework, a
state layer, and the UI on top.

- Implement the models, the process runner and the vfox client exactly as
  [the CLI contract](vfox-cli-contract.md) describes: environment, working
  directory, argument validation, error summaries.
- Parse vfox's output with pure functions and test them against **every** case
  in [`shared/fixtures/*/cases.json`](../shared/fixtures/README.md). This is the
  most valuable test of all: it is what guarantees that all implementations read
  vfox the same way.
- Implement the task queue: one state-changing command at a time, streamed
  output, download progress, bounded log, cancellation, retry.
- Read and write the settings file described in
  [the architecture](architecture.md#settings), with the same keys.

## 3. Build the interface

- Implement every screen, state and flow of [the UI specification](ui-spec.md).
- Use the tokens, springs and components of [the design system](design-system.md).
  When the technology lacks a Material 3 Expressive component, implement it
  following the specification rather than substituting a Material 2 or 3 one.
- Use the Material Symbols Rounded icons listed in
  [`shared/icons.json`](../shared/icons.json), and ship only those glyphs:
  [`shared/tools/icon_font.py`](../shared/tools/icon_font.py) cuts them out of
  a pinned version of the font at build time.
- Read the translations from [`shared/i18n`](../shared/i18n); do not copy them.
  See [the i18n guide](i18n.md).
- Support English and Simplified Chinese, light and dark themes, the desktop
  accent colour, reduced motion and keyboard navigation.

## 4. Test and automate

- Unit tests for the core, including the shared fixtures; tests for the state
  layer with a fake process runner; UI tests for the main flows.
- End-to-end tests of the built application: against a fake vfox that replays
  the fixtures, and against the real vfox with the offline plugin of
  [`shared/test-plugin`](../shared/test-plugin), which needs no network and
  touches no vfox installation (see
  [the CLI contract](vfox-cli-contract.md#testing-against-the-real-vfox)).
- A way to run all checks without changing the developer's machine: a Docker
  sandbox built on [`shared/sandbox`](../shared/sandbox) (see
  [`tauri/sandbox`](../tauri/sandbox) for a small example).
- A formatter and a linter, both enforced in CI.
- A workflow `.github/workflows/<directory>.yml` that only runs for changes in
  the directory, `shared/` and the workflow itself, and that builds the
  installable packages as artifacts. Use [`flutter.yml`](../.github/workflows/flutter.yml)
  as a model.

## 5. Package

The issue asks for:

- a `.deb` for Ubuntu 26.04 (application ID `dev.vfox.VfoxPlus`, with the
  desktop file, AppStream metadata and icon of
  [`shared/packaging/linux`](../shared/packaging/linux));
  [`build-deb.sh`](../shared/packaging/linux/build-deb.sh) packages a release
  build with the common layout, names the package `vfox-plus-<directory>` and
  makes it replace the other implementations;
- an `.msixbundle` for Windows 11;
- an `.apk` for remote use from Android.

Install the `.deb` into a clean Ubuntu 26.04 container and start the application
there, as `install_deb_test` of
[`shared/sandbox/task-lib.sh`](../shared/sandbox/task-lib.sh) does: GUI
toolkits load some libraries at run time, which `ldd` does not show (Flutter
needs `libegl1` and `libgles2`), so only a clean system proves that the declared
dependencies are complete. Ship the licenses of bundled third-party assets, such
as icon fonts, with the application.

Each implementation should also report its memory use: the resident set size
after start-up on the Home page and after opening the version list of Node.js,
measured on the same machine as the other implementations. The `measure` task
of the sandboxes does this in Docker, with Xvfb and software rendering, and
adds the memory of child processes such as the processes of a web view.

## 6. Document

- `<directory>/README.md`: prerequisites, how to run, test, build and package,
  and the project layout.
- Add the implementation to the table in the [root README](../README.md).
