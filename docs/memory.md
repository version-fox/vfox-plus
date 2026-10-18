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

| Implementation                             | Technology                                     | Processes | Home RSS | Home PSS | Versions RSS | Versions PSS | The list adds (PSS) |
| ------------------------------------------ | ---------------------------------------------- | --------: | -------: | -------: | -----------: | -----------: | ------------------: |
| [Uno](../uno/README.md#memory)             | Uno Platform 6, C# on .NET 10, Skia on the CPU |         1 |      146 |      133 |          181 |          168 |                 +35 |
| [Gio](../gio/README.md#memory)             | Gio, Go, OpenGL ES through EGL                 |         1 |      240 |      209 |          257 |          226 |                 +17 |
| [Compose](../compose/README.md#memory)     | Compose Multiplatform, Kotlin on the JVM, Skia |         1 |      290 |      263 |          301 |          275 |                 +12 |
| [Flet](../flet/README.md#memory)           | Flet, Python 3.14 in a Flutter application     |         2 |      374 |      341 |          394 |          360 |                 +19 |
| [Capacitor](../capacitor/README.md#memory) | Ionic Vue and Capacitor in Electron            |         6 |      729 |      385 |          750 |          398 |                 +13 |
| [Tauri](../tauri/README.md#memory)         | Tauri 2, Rust, WebKitGTK 4.1 (GTK 3)           |         3 |      598 |      397 |          683 |          482 |                 +85 |
| [Dioxus](../dioxus/README.md#memory)       | Dioxus 0.7, Rust, WebKitGTK                    |         3 |      618 |      418 |          698 |          497 |                 +79 |
| [Wails](../wails/README.md#memory)         | Wails 3, Go, WebKitGTK 6.0 (GTK 4)             |         3 |      686 |      480 |          758 |          554 |                 +74 |

- The three implementations that run in one process and draw themselves (Uno, Gio,
  Compose) are the three smallest, with 133 to 263 MB of PSS on the Home page. Flet,
  which runs the Flutter engine and an embedded Python interpreter, has 341 MB. The
  four that bring a web view or a browser, in three to six processes (Capacitor,
  Tauri, Dioxus, Wails), have 385 to 480 MB.
- Showing the list of versions adds 12 to 35 MB of PSS to Compose, Capacitor, Gio, Flet
  and Uno, and 74 to 85 MB to Wails, Dioxus and Tauri, which show their pages in
  WebKitGTK.
- WebKitGTK 6.0 with GTK 4 (Wails) needs about 80 MB more than WebKitGTK 4.1 with
  GTK 3 (Tauri) for the same web front end.

## How it is measured

[`shared/tools/measure.sh`](../shared/tools/measure.sh) measures every implementation
the same way:

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
  were not measured at the same time, so small differences mean little.
- **What is not in them.** Only the process tree of the application is measured: the
  X server, the session bus and the desktop portal are not, and neither is vfox,
  which is the fake one here, so a real command adds the process of vfox while it
  runs.

## Flutter

The Flutter implementation is not in the table: its sandbox has no `measure` task
(the application has no `measure` scenario), and the numbers in its README were
measured another way — on Ubuntu 26.04 under WSLg, which renders in software with
Mesa llvmpipe, 10 seconds after start-up, in a 1280 × 820 window: 336 MB of RSS and
320 MB of PSS on the Home page, against 254 MB and 247 MB for the counter app that
`flutter create` makes. They cannot be compared with the table. To put Flutter in it,
give it the `measure` scenario and measure it in the same container as the others.

## Updating this page

When an implementation is added, or its memory changes noticeably, run
`<implementation>/sandbox/sandbox.sh measure`, copy the lines of
`dist/sandbox/<implementation>/measure/memory.txt` into the table and into the
Memory section of the README of the implementation, and keep the table sorted by the
PSS of the Home page.
