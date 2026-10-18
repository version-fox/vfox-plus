# vfox+ — web front end

The web front end of vfox-plus, shared by the [Tauri](../tauri) and the
[Wails](../wails) implementations: the vfox client, the task queue and the
Material 3 Expressive user interface, in TypeScript. The two shells only add
what a web view cannot do — spawning vfox, a few file operations and the
desktop integration — through the small [host protocol](src/hosts/protocol.ts).
Sharing the front end keeps the comparison of the two shells about what
differs between them: the language of the native side, packaging and memory.
The [Capacitor implementation](../capacitor) shares everything here but the
Lit elements: the core, the state, `src/ui/context.ts`, `theme.ts` and
`motion.ts`, the Node.js host of [`dev-host/`](dev-host) and the end-to-end
scenarios. Keep those free of Lit.

It follows the shared specification in [`../docs`](../docs) and uses the shared
resources in [`../shared`](../shared).

## Technology

| Part       | Choice                                                                                                      | License                |
| ---------- | ----------------------------------------------------------------------------------------------------------- | ---------------------- |
| Language   | TypeScript 6, built with Vite 8                                                                             | Apache-2.0, MIT        |
| Components | [Lit](https://lit.dev) and [@m3e/web](https://github.com/matraic/m3e), Material 3 Expressive web components | BSD-3-Clause, MIT      |
| Long lists | [@lit-labs/virtualizer](https://github.com/lit/lit/tree/main/packages/labs/virtualizer)                     | BSD-3-Clause           |
| Colour     | [Material Color Utilities](https://github.com/material-foundation/material-color-utilities)                 | Apache-2.0             |
| Shells     | [@tauri-apps/api](https://tauri.app), [@wailsio/runtime](https://wails.io)                                  | Apache-2.0 or MIT, MIT |

`@m3e/web` provides the expressive components: the navigation rail and bar,
connected button groups, split buttons, segmented lists, the shape-morphing
loading indicator, the wavy progress indicator, the shape library, menus,
dialogs and snack bars. The application overrides its colours and springs to
follow [the design system](../docs/design-system.md) exactly: the colour scheme
is the tonal spot scheme of the 2021 specification, as in the other
implementations, and every motion token is a CSS `linear()` easing sampled from
the springs of the expressive motion scheme (`src/ui/motion.ts`).

## Requirements

- Node.js 24 (the version is pinned in [`versions.env`](versions.env)).
- Python 3 with fontTools and brotli, to cut the icon font out of Material
  Symbols: `sudo apt install fonttools python3-brotli` or
  `pip install fonttools brotli`.

## Everyday commands

Run them from this directory.

| Task                                                                                               | Command                                        |
| -------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| Install the packages                                                                               | `npm ci`                                       |
| Generate the icon font and the license list (after cloning and after changing `shared/icons.json`) | `npm run assets`                               |
| Run in a browser against a fake vfox                                                               | `npm run dev`, then open http://127.0.0.1:5173 |
| Format                                                                                             | `npm run format`                               |
| Check formatting                                                                                   | `npm run format:check`                         |
| Lint (CI fails on warnings)                                                                        | `npm run lint`                                 |
| Type check                                                                                         | `npm run typecheck`                            |
| Unit tests                                                                                         | `npm test`                                     |
| Coverage (CI requires 85 % of the core and the state)                                              | `npm run test:coverage`                        |
| Build for Tauri and Wails                                                                          | `npm run build` (`dist/tauri`, `dist/wails`)   |

`npm run dev` serves the front end with a _development host_
([`dev-host/`](dev-host)): Node.js does what Tauri or Wails would do, over a
WebSocket. vfox runs with a new temporary `HOME`, so your vfox installation is
never touched; by default it is the fake vfox of
[`shared/fixtures/fake-vfox.sh`](../shared/fixtures/fake-vfox.sh), which
replays the captured output of vfox. Set `VFOX_PLUS_DEV_VFOX` to use another
executable. Never expose the development server beyond your machine.

The npm registry is set to `https://registry.npmjs.org/` in
[`.npmrc`](.npmrc), so that `package-lock.json` never records a mirror; set the
`npm_config_registry` environment variable to use one.

## Sandbox

[`sandbox/sandbox.sh`](sandbox/sandbox.sh) runs the checks of CI in throw-away
Docker containers, without anything but Docker on the host:

| Command                    | What it does                                                                                 |
| -------------------------- | -------------------------------------------------------------------------------------------- |
| `sandbox/sandbox.sh check` | Formatting, lint, type checking, unit tests with the coverage gate and the production builds |
| `sandbox/sandbox.sh dev`   | Serves the front end on http://127.0.0.1:5173 against the fake vfox                          |
| `sandbox/sandbox.sh shell` | A shell in the sandbox                                                                       |
| `sandbox/sandbox.sh clean` | Deletes the image, the npm cache volume and the results                                      |

The shared parts of the sandboxes are described in
[`../shared/sandbox`](../shared/sandbox).

## End-to-end tests

The scenarios of [`src/e2e/scenarios.ts`](src/e2e/scenarios.ts) are the same as
those of the Flutter implementation. They run _inside_ the application: a build
made with `VFOX_PLUS_E2E=1` contains a runner that drives the user interface
like a user would — finding elements by their text and accessible name,
clicking and typing — and ends the application with the result.
[`e2e/run.sh`](e2e/run.sh) prepares a temporary home directory and starts the
application:

| Scenario      | What it tests                                                                                                                                                                                  |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `fake`        | The application against the fake vfox, which also checks the environment the CLI contract requires                                                                                             |
| `real`        | Every operation against the real vfox named by `VFOX_PLUS_E2E_VFOX`, with the offline plugin of [`shared/test-plugin`](../shared/test-plugin) and [`e2e/test-server.mjs`](e2e/test-server.mjs) |
| `measure`     | Prints the memory use of the application on the Home page and with the versions of Node.js listed                                                                                              |
| `screenshots` | Shows every page for [`shared/tools/screenshots.sh`](../shared/tools/screenshots.sh), which captures the screen at each of them; the compact pages need a shell that can resize its window     |

The Tauri and Wails workflows and sandboxes run them in their applications.
The [Capacitor implementation](../capacitor) runs them in its Ionic user
interface, so the driver ([`src/e2e/driver.ts`](src/e2e/driver.ts)) knows
Ionic's components and the scenarios pick elements of both libraries. The
[Dioxus implementation](../dioxus) ports the scenarios to Rust and drives its
page with the same driver, so `e2e/run.sh` runs its end-to-end build too;
changes to either reach all of them.

## Project layout

```text
web/
├── index.html                The page; <m3e-theme> right below the body
├── src/
│   ├── main.ts               Connects to the host and starts the application
│   ├── core/                 Models, process runner, vfox client and parsers,
│   │                         settings, translations — no UI
│   ├── state/                Controllers: AppModel, task queue, …
│   ├── ui/                   Lit elements: shell, pages, components, theme, motion
│   ├── hosts/                Host protocol and its adapters: Tauri, Wails, development
│   └── e2e/                  End-to-end runner and scenarios
├── test/                     Unit tests, including the shared fixtures
├── dev-host/                 Node.js host of `npm run dev`
├── e2e/                      Harness, fake vfox and test server of the end-to-end tests
├── tools/                    Icon font and license list generators
└── sandbox/                  Docker sandbox
```

The three layers and their rules are described in
[`../docs/architecture.md`](../docs/architecture.md); ESLint enforces that the
core and the state do not import the user interface.

## Icons

`npm run assets` cuts the icons of [`shared/icons.json`](../shared/icons.json)
out of Material Symbols Rounded with
[`shared/tools/icon_font.py`](../shared/tools/icon_font.py) into
`public/fonts/MaterialSymbolsRounded.woff2`: 9 KB, keeping the ligatures of the
icon names and the fill axis, which animates the selected destination. The font
is a build output and not committed. The variable font is taken from
`VFOX_PLUS_SYMBOLS` (the sandbox images contain it) or downloaded once into
`node_modules/.cache`.
