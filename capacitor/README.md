# vfox+ — Capacitor implementation

vfox-plus built with [Ionic](https://ionicframework.com) 9 (Ionic Vue) and
[Capacitor](https://capacitorjs.com) 8, in an [Electron](https://www.electronjs.org)
shell for the Linux desktop. The pages are Vue components on Ionic's
components — buttons, lists and items, cards, segments, toggles, text fields,
search bars, modals, popovers, toasts and the ripple — restyled from Ionic's
Material Design mode to Material 3 Expressive through the custom properties
and shadow parts that Ionic documents ([`src/ui/styles/ionic.css`](src/ui/styles/ionic.css)).
What Ionic has no counterpart of comes from [@m3e/web](https://github.com/matraic/m3e),
the library of the [web front end](../web): the shape library, the loading
indicator, the wavy progress indicator and tooltips. The navigation rail and
bar are buttons with Ionic's ripple. Everything moves on the springs of the
expressive motion scheme.

Capacitor has no Linux platform, and its community Electron platform supports
only Capacitor 5 and has not been updated since 2023. The page therefore runs
in an Electron shell of its own ([`electron`](electron)), where Capacitor sees
the web platform. The native side of the page is a Capacitor plugin,
`VfoxHost` ([`src/capacitor/vfox-host.ts`](src/capacitor/vfox-host.ts)): its
web implementation carries the host protocol of the web front end
([`../web/src/hosts/protocol.ts`](../web/src/hosts/protocol.ts)) over the IPC
bridge of the preload script to the main process, which runs vfox and reads
and writes files with the Node.js host of the web front end
([`../web/dev-host/node-host.ts`](../web/dev-host/node-host.ts)). An Android or
iOS app would implement the same plugin natively and keep the rest.

The code that does not depend on the user interface is the one of the web
front end, imported as `@web/…`:

| Part                                                                         | What it holds                                                                                                                                                                                                        |
| ---------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`../web/src/core`](../web/src/core), [`../web/src/state`](../web/src/state) | The vfox client and its parsers, the settings, the translations, the task queue, the controllers and `AppModel`                                                                                                      |
| [`../web/src/ui`](../web/src/ui) (`context`, `theme`, `motion`)              | The running application, the colour scheme and the springs                                                                                                                                                           |
| [`src/capacitor`](src/capacitor)                                             | The `VfoxHost` plugin and the host on top of it                                                                                                                                                                      |
| [`src/ui`](src/ui)                                                           | The Vue components: the shell, the pages, their parts, the dialogs and menus, snack bars, transitions, and [`reactivity.ts`](src/ui/reactivity.ts), which makes renders depend on the objects of the model they read |
| [`electron`](electron)                                                       | The main process (window, `app:` protocol, IPC), the preload script and the desktop operations: dialogs, the clipboard, links and the accent colour of the XDG desktop portal                                        |

## Requirements

- Node.js 24 (pinned in [`../web/versions.env`](../web/versions.env)).
  Electron is pinned in [`package.json`](package.json); `npm run electron`
  downloads it into `node_modules`, checked against the checksums of the
  package.
- The libraries of Electron, gdbus for the accent colour and xdg-open for
  links:

  ```sh
  sudo apt install libgtk-3-0t64 libnss3 libgbm1 libasound2t64 libxkbcommon0 libglib2.0-bin xdg-utils
  ```

- Python 3 with [fontTools](https://github.com/fonttools/fonttools) and brotli
  (`sudo apt install fonttools python3-brotli`): the icons of
  [`shared/icons.json`](../shared/icons.json) are cut out of Material Symbols
  Rounded with [`shared/tools/icon_font.py`](../shared/tools/icon_font.py),
  which downloads the pinned font once, or reads it from `VFOX_PLUS_SYMBOLS`.
- vfox 1.0.0 or later at run time.

## Everyday commands

| Task                                                                                                   | Command                                                                 |
| ------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------- |
| Install the packages and the Electron runtime                                                          | `npm ci && npm run electron`                                            |
| Build the icon font and collect the licenses                                                           | `npm run assets`                                                        |
| Run in Electron                                                                                        | `npm start`                                                             |
| Develop in a browser at http://127.0.0.1:5174, with the development host of `../web` and the fake vfox | `npm run dev`                                                           |
| Format (CI fails on a difference)                                                                      | `npm run format`                                                        |
| Type check                                                                                             | `npm run typecheck`                                                     |
| ESLint (CI fails on warnings)                                                                          | `npm run lint`                                                          |
| Unit tests                                                                                             | `npm test`                                                              |
| Unit tests with the coverage gate (85 % of the lines of the TypeScript modules)                        | `npm run test:coverage`                                                 |
| Release build (`dist/app`) and the application with its Electron runtime (`dist/stage`)                | `npm run build && npm run stage`                                        |
| `.deb` and `.tar.gz`                                                                                   | `bash tools/package.sh <output directory>`, see [Packaging](#packaging) |

The licenses of the npm packages, Electron and the font must be compatible
with the Apache License 2.0; `npm run assets` fails otherwise (see
[`docs/architecture.md`](../docs/architecture.md#dependencies)).

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI, the
end-to-end tests and the packaging in throw-away Docker containers. It needs
nothing but Docker and changes nothing on your machine: the images bring
Node.js, the libraries of Electron and vfox (the image extends the one of
[`../web/sandbox`](../web/sandbox)), see the repository read-only and keep the
npm cache and the Electron runtime in Docker volumes. Chromium needs more
shared memory than Docker gives containers by default, so the containers get
1 GB of `/dev/shm`.

| Command                          | What it does                                                                                                                       |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `sandbox/sandbox.sh check`       | Formatting, type checks, ESLint, the licenses, and the unit tests with the coverage gate                                           |
| `sandbox/sandbox.sh e2e`         | The end-to-end scenarios, against the fixtures and against the real vfox                                                           |
| `sandbox/sandbox.sh package`     | Release build with a smoke test, `.deb` and `.tar.gz` checked by lintian, then an installation into a clean Ubuntu 26.04 container |
| `sandbox/sandbox.sh measure`     | The memory use on the Home page and with the versions of Node.js listed                                                            |
| `sandbox/sandbox.sh screenshots` | Screenshots of every page                                                                                                          |
| `sandbox/sandbox.sh all`         | `check`, `e2e` and `package`                                                                                                       |
| `sandbox/sandbox.sh shell`       | A shell in the sandbox                                                                                                             |
| `sandbox/sandbox.sh clean`       | Deletes the image, the cache volumes and the results                                                                               |

Results are written to `dist/sandbox/capacitor/<command>/`.
`SANDBOX_ELECTRON_MIRROR` sets a mirror of the Electron releases, for example
`https://npmmirror.com/mirrors/electron/`.

## End-to-end tests

The scenarios of the web front end ([`../web/src/e2e/scenarios.ts`](../web/src/e2e/scenarios.ts))
run in a build made with `npm run build:e2e`: the page drives itself with the
driver of the web front end, checks the state of the model and exits with the
result. [`../web/e2e/run.sh`](../web/e2e/run.sh) prepares a new, empty home
directory, the fake vfox or the real vfox with the offline test plugin and its
server:

```sh
npm run build:e2e
xvfb-run -a dbus-run-session -- ../web/e2e/run.sh fake node_modules/electron/dist/electron dist/app-e2e
VFOX_PLUS_E2E_VFOX=/path/to/vfox xvfb-run -a dbus-run-session -- \
  ../web/e2e/run.sh real node_modules/electron/dist/electron dist/app-e2e
```

They exercise the whole stack: the components, the state, the plugin and the
IPC, process spawning, streaming, cancellation with `SIGTERM`, the settings
file and vfox itself. [`shared/tools/screenshots.sh`](../shared/tools/screenshots.sh)
runs the `screenshots` scenario of the same build. In containers, set
`ELECTRON_DISABLE_SANDBOX=1`: the sandbox of Chromium needs a setuid helper or
user namespaces, which containers do not have.

## Packaging

[`tools/stage.mjs`](tools/stage.mjs) puts the application into the Electron
runtime: the `electron` executable becomes `vfox-plus`, the application goes to
`resources/app`, and only the English and Chinese locales of Chromium stay.
[`tools/package.sh`](tools/package.sh) packages it with the shared script
[`shared/packaging/linux/build-deb.sh`](../shared/packaging/linux/build-deb.sh)
as `vfox-plus-capacitor_<version>_amd64.deb` and a `.tar.gz`, with the same
layout, desktop entry, AppStream metadata and icon as the other
implementations (`/usr/lib/vfox-plus/vfox-plus`, started as `/usr/bin/vfox-plus`).
The package provides, conflicts with and replaces `vfox-plus`, so the
implementations replace each other. The licenses of the npm packages, Electron
and the font are installed as `/usr/share/doc/vfox-plus-capacitor/THIRD-PARTY-LICENSES`
and shown in the application; those of Chromium are in
`/usr/lib/vfox-plus/LICENSES.chromium.html`.

The sandbox helper of Chromium, `chrome-sandbox`, is installed setuid root,
as in every Electron package; lintian is told so. The `.tar.gz` cannot carry
that: unpacked by a user, the application needs either
`sudo chown root:root chrome-sandbox && sudo chmod 4755 chrome-sandbox` or
`vfox-plus --no-sandbox`. The window class and the Wayland application ID are
`dev.vfox.VfoxPlus`, so that the window matches the desktop entry. Chromium
keeps its profile in `~/.cache/vfox-plus/electron`, apart from the settings
of vfox+.

## Memory

Measured with `sandbox/sandbox.sh measure`: Ubuntu 26.04 in Docker, Xvfb with
software rendering, a 1280 × 800 window, the Electron processes together (the
browser, the zygotes, the GPU, network and renderer processes).

| State                      |    RSS |    PSS |
| -------------------------- | -----: | -----: |
| Home page                  | 742 MB | 388 MB |
| Versions of Node.js listed | 762 MB | 400 MB |

Electron brings its own Chromium, so nothing is shared with the libraries of
the desktop; compare implementations measured the same way. Only the rows of
the available versions that are in view are rendered
([TanStack Virtual](https://tanstack.com/virtual)).

## Differences from the Flutter implementation

- The text fields of the dialogs, whose labels float, keep Ionic's outline,
  which the label rests on; the other text fields are filled like those of
  the Flutter implementation.
- `@ionic/vue` imports every component of Ionic, and their modules are not
  marked free of side effects; [`vite.config.ts`](vite.config.ts) tells the
  bundler that they are, so that the bundle leaves out the unused ones.
