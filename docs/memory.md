# Memory use

Low memory use is a goal of the project (see [the architecture](architecture.md#memory)).
This page compares what the implementations use on Linux, measured the same way,
so that the choice between them can take it into account. The numbers are a
snapshot: every row is the result of the `measure` task of the sandbox of that
implementation, run once, and the same numbers are in the Memory section of its
README (the links in the first column).

## Results

In MB, for the application and all of its child processes together, in Docker on
Ubuntu 26.04 with Xvfb (1280 × 800, no GPU). The table is sorted by the PSS of the
Home page; "Versions" is the SDKs page with the 872 available versions of Node.js
listed.

| Implementation                             | Technology                                      | Processes | Home RSS | Home PSS | Versions RSS | Versions PSS | The list adds (PSS) |
| ------------------------------------------ | ----------------------------------------------- | --------: | -------: | -------: | -----------: | -----------: | ------------------: |
| [Uno](../uno/README.md#memory)             | Uno Platform 6, C# on .NET 10, Skia on the CPU  |         1 |      146 |      133 |          181 |          168 |                 +35 |
| [Gio](../gio/README.md#memory)             | Gio, Go, OpenGL ES through EGL                  |         1 |      240 |      209 |          257 |          226 |                 +17 |
| [Compose](../compose/README.md#memory)     | Compose Multiplatform, Kotlin on the JVM, Skia  |         1 |      290 |      263 |          301 |          275 |                 +12 |
| [Flutter](../flutter/README.md#memory)     | Flutter 3.47, Dart, Impeller (OpenGL ES), GTK 3 |         2 |      379 |      340 |          394 |          355 |                 +15 |
| [Flet](../flet/README.md#memory)           | Flet, Python 3.14 in a Flutter application      |         2 |      374 |      341 |          394 |          360 |                 +19 |
| [Capacitor](../capacitor/README.md#memory) | Ionic Vue and Capacitor in Electron             |         6 |      729 |      385 |          750 |          398 |                 +13 |
| [Tauri](../tauri/README.md#memory)         | Tauri 2, Rust, WebKitGTK 4.1 (GTK 3)            |         3 |      598 |      397 |          683 |          482 |                 +85 |
| [Dioxus](../dioxus/README.md#memory)       | Dioxus 0.7, Rust, WebKitGTK                     |         3 |      618 |      418 |          698 |          497 |                 +79 |
| [Wails](../wails/README.md#memory)         | Wails 3, Go, WebKitGTK 6.0 (GTK 4)              |         3 |      686 |      480 |          758 |          554 |                 +74 |

- The three implementations that run in one process and draw themselves (Uno, Gio,
  Compose) are the three smallest, with 133 to 263 MB of PSS on the Home page. Flutter
  and Flet, whose client is a Flutter application too, follow with 340 and 341 MB, each
  in two processes. The four that bring a web view or a browser, in three to six
  processes (Capacitor, Tauri, Dioxus, Wails), have 385 to 480 MB.
- Showing the list of versions adds 12 to 35 MB of PSS to Compose, Capacitor, Gio,
  Flutter, Flet and Uno, and 74 to 85 MB to Wails, Dioxus and Tauri, which show their
  pages in WebKitGTK.
- WebKitGTK 6.0 with GTK 4 (Wails) needs about 80 MB more than WebKitGTK 4.1 with
  GTK 3 (Tauri) for the same web front end.

## How it is measured

[`shared/tools/measure.sh`](../shared/tools/measure.sh) measures every implementation
but Flutter the same way (Flutter has its own steps, [below](#measuring-flutter)):

1. It starts the application on the current display, with a new, empty home
   directory and the fake vfox of [`shared/fixtures`](../shared/fixtures), which
   replays captured output (so no real vfox runs), in the `measure` scenario of the
   application.
2. The application shows the Home page, waits 3 seconds so that it has settled and
   reports `e2e: MEASURE home`. The script, which reads the report, adds up `Rss`
   and `Pss` from `/proc/<pid>/smaps_rollup` for the application and every process
   below it; the application waits 5 more seconds, so that the page does not change
   during the measurement.
3. The application opens the available versions of Node.js, waits 3 seconds and
   reports `e2e: MEASURE nodejs`, and the script measures again.

The sizes are in MiB, which the script calls MB. The `measure` task of a sandbox runs
all of this in Docker (`<implementation>/sandbox/sandbox.sh measure`, which needs
nothing but Docker); the numbers above come from a WSL2 host with 16 logical CPUs.

## Reading the numbers

- **RSS and PSS.** RSS counts every page that a process has in memory, including
  those it shares with other processes, so the RSS of several processes counts a
  shared page several times: 729 MB of RSS against 385 MB of PSS for Capacitor, in six
  processes. PSS divides each shared page among the processes that map it, so the
  sum is what the application uses. Compare PSS.
- **No GPU.** The virtual display has no GPU, so everything that draws with OpenGL
  uses the software rasterizer of Mesa (llvmpipe), and its memory is in the numbers
  (the Gio README says that it is most of what Gio uses). Uno was set to draw with Skia on the
  processor, and the same build of it took 295 MB on the Home page with Mesa's
  software GL and 165 MB without it, with every other setting equal. On a machine
  with a GPU the numbers, and perhaps the order, differ; measure there when it
  matters.
- **One measurement per row.** Nothing is averaged over runs and the implementations
  were not measured at the same time, so small differences mean little. Flutter, which
  was measured six times, had 332 to 342 MB of PSS on the Home page and 344 to
  369 MB with the list: the list added 4 to 30 MB.
- **What is not in them.** Only the process tree of the application is measured: the
  X server, the session bus and the desktop portal are not, and neither is vfox,
  which is the fake one here, so a real command adds the process of vfox while it
  runs.

## Measuring Flutter

The Flutter implementation has no `measure` scenario: its pages keep their state in
widgets, where only a test can reach it, and the integration tests run a debug build.
[`flutter/tool/measure.py`](../flutter/tool/measure.py) therefore measures the release
build from the outside, with the same sums as `measure.sh`, and opens the pages the way
a user does:

1. It starts the release bundle with the environment of `measure.sh`: a new, empty home
   directory, the fake vfox and the same settings file.
2. It waits for the window, and then for the screen to show something and not to change
   for 3 seconds (the other implementations wait 3 seconds after they show the Home
   page). That is the Home page, which it measures.
3. It clicks, with the mouse events of the XTEST extension of the X server, the SDKs
   entry of the navigation rail, `nodejs` in the list and the Available tab, and waits
   for the screen to settle after each click. With the 872 versions listed, it measures
   again. A click has to change a tenth of the screen at least; if a change of the
   layout has moved a button, the script stops with a message instead of measuring the
   wrong page.

`flutter/sandbox/sandbox.sh measure` builds the release bundle and runs the script in
Xvfb with a session bus, as the other sandboxes do. It writes `memory.txt`, a screenshot
of each of the four pages that the script opened (`home.png`, `sdks.png`,
`nodejs-installed.png` and `nodejs.png`, so that what was measured can be looked at) and
`app.log` to `dist/sandbox/measure/`. The `package` job of the Flutter workflow runs the
script on the bundle that it builds. Without the sandbox, run
`xvfb-run -a -s "-screen 0 1280x800x24" dbus-run-session -- python3 tool/measure.py <bundle>/vfox-plus`
in `flutter/`; the script needs nothing but Python 3 and an X11 display.

What is different from the other implementations:

- The application runs in two processes. The second, `glycin-svg`, is the helper that
  gdk-pixbuf starts to load the SVG icon of the window (with the icon file removed, the
  process is not there). It has 18 MB of the RSS, and counts like the processes of a web
  view.
- The window is 1280 × 820 and the screen 1280 × 800: the window fills the screen and the
  last 20 lines are cut off. The positions that the script clicks are for this window and
  this layout; they are constants at the top of the script.
- The memory is read when the screen has been still for 3 seconds, not when the
  application reports a page.

## Updating this page

When an implementation is added, or its memory changes noticeably, run
`<implementation>/sandbox/sandbox.sh measure`, copy the lines of
`dist/sandbox/<implementation>/measure/memory.txt` (`dist/sandbox/measure/memory.txt`
for Flutter) into the table and into the Memory section of the README of the
implementation, and keep the table sorted by the PSS of the Home page.
