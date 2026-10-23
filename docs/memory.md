# Memory use

Low memory use is a goal of the project (see [the architecture](architecture.md#memory)).
This page compares what the implementations use on Linux, measured the same way,
so that the choice between them can take it into account. The numbers are a
snapshot: every row is the result of the `measure` task of the sandbox of that
implementation, run once, and the same numbers are in the Memory section of its
README (the links in the first column).

## Results

In MB, for the application and all of its child processes together, in Docker on
Ubuntu 26.04 with Xvfb (1280 × 800, no GPU). The table is sorted by the first of the
memory columns, the PSS of the Home page, from the smallest to the largest (the RSS
is not in that order: see [Reading the numbers](#reading-the-numbers)); "Versions" is
the SDKs page with the 872 available versions of Node.js listed.

| Implementation                             | Technology                                      | Processes | Home PSS | Home RSS | Versions PSS | Versions RSS | The list adds (PSS) |
| ------------------------------------------ | ----------------------------------------------- | --------: | -------: | -------: | -----------: | -----------: | ------------------: |
| [mygo](../mygo/README.md#memory)           | mygo 0.5, Go, GTK 3 (purego), CPU renderer                |         1 |       41 |       54 |           47 |           60 |                  +6 |
| [Uno](../uno/README.md#memory)             | Uno Platform 6, C# on .NET 10, Skia on the CPU, Native AOT |         1 |       99 |      106 |          116 |          123 |                 +17 |
| [Gio](../gio/README.md#memory)             | Gio, Go, OpenGL ES through EGL                             |         1 |      215 |      240 |          228 |          253 |                 +13 |
| [Dioxus](../dioxus/README.md#memory)       | Dioxus 0.7, Rust, WebKitGTK, compositing off               |         3 |      258 |      434 |          296 |          473 |                 +38 |
| [Compose](../compose/README.md#memory)     | Compose Multiplatform, Kotlin on the JVM, Skia             |         1 |      266 |      290 |          277 |          301 |                 +11 |
| [Tauri](../tauri/README.md#memory)         | Tauri 2, Rust, WebKitGTK 4.1 (GTK 3), compositing off      |         3 |      275 |      442 |          285 |          452 |                 +10 |
| [Capacitor](../capacitor/README.md#memory) | Ionic Vue and Capacitor in Electron, no hardware GPU       |         6 |      303 |      655 |          313 |          672 |                 +10 |
| [Flutter](../flutter/README.md#memory)     | Flutter 3.47, Dart, Impeller (OpenGL ES), GTK 3            |         2 |      329 |      369 |          345 |          384 |                 +16 |
| [Flet](../flet/README.md#memory)           | Flet, Python 3.14 in a Flutter application                 |         2 |      346 |      378 |          383 |          415 |                 +37 |
| [Wails](../wails/README.md#memory)         | Wails 3, Go, WebKitGTK 6.0 (GTK 4), compositing off        |         3 |      346 |      525 |          358 |          538 |                 +12 |

- Turning off GPU-dependent rendering where the pages do not need it, and compiling
  ahead of time where that lets trimming do a better job, closes most of the gap
  between the four implementations that run in one process and draw themselves
  (mygo, Uno, Gio, Compose, 41 to 266 MB of PSS on the Home page) and the four that
  bring a web view or a browser: with WebKitGTK's accelerated compositing turned off,
  Dioxus and Tauri are 258 and 275 MB, in three processes, both now cheaper than
  Compose; Capacitor, with Electron's hardware acceleration off, is 303, in six
  processes. mygo still uses under half of what Uno does, drawing on the processor
  from a Go executable of 12 MB and loading GTK 3 from the system; Uno, compiled with
  Native AOT instead of ReadyToRun, is a single piece of machine code with no separate
  managed runtime, and keeping no XAML pages suits that well. Flutter and Flet, whose
  client is a Flutter application too, follow at 329 and 346, each in two processes.
  Wails, in the same three processes and with the same fix as Tauri and Dioxus, stays
  at 346, tied with Flet: it links the heavier WebKitGTK 6.0 with GTK 4 instead of their
  WebKitGTK 4.1 with GTK 3 (see below).
- Showing the list of versions now adds 6 to 38 MB of PSS to every implementation; before
  this page turned their accelerated compositing off, it added 73 to 84 MB to Tauri,
  Dioxus and Wails, which show the list in WebKitGTK.
- WebKitGTK 6.0 with GTK 4 (Wails) still needs 71 MB more than WebKitGTK 4.1 with GTK 3
  (Tauri) for the same web front end, with accelerated compositing off on both.

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
  shared page several times: 742 MB of RSS against 388 MB of PSS for Capacitor, in six
  processes. PSS divides each shared page among the processes that map it, so the
  sum is what the application uses. Compare PSS.
- **No GPU.** The virtual display has no GPU, so everything that draws with OpenGL
  uses the software rasterizer of Mesa (llvmpipe), and its memory is in the numbers
  (the Gio README says that it is most of what Gio uses). Uno was set to draw with Skia on the
  processor, and the same build of it took 295 MB on the Home page with Mesa's
  software GL and 165 MB without it, with every other setting equal. mygo draws on the
  processor too, and the application has GDK leave OpenGL alone: the same build took
  about 78 MB when GDK opened its OpenGL connection, and about 43 MB when it did not.
  Capacitor's Electron shell calls `app.disableHardwareAcceleration()` for the same
  reason: the same build took 388 MB of PSS on the Home page with it on and 303 MB with
  it off. Tauri, Dioxus and Wails set `WEBKIT_DISABLE_COMPOSITING_MODE=1` before their
  webview starts, which keeps WebKitGTK from entering accelerated compositing mode at
  all: the same builds took 419, 427 and 482 MB with it on, against 275, 258 and 346
  with it off. On a machine with a GPU the numbers of the implementations that draw
  with OpenGL, and perhaps the order, differ (Uno, mygo, Capacitor, Tauri, Dioxus and
  Wails no longer depend on the GPU); measure there when it matters.
- **One measurement per row.** Nothing is averaged over runs, so small differences
  mean little; the implementations were measured one after another, with nothing else
  running. Flutter, which was measured six times on an earlier day, had 332 to 342 MB
  of PSS on the Home page and 344 to 369 MB with the list: the list added 4 to 30 MB.
- **One snapshot at a time.** The numbers of the same code change from one snapshot
  to the next: measured again in sandbox images built from the Ubuntu archive of a
  later day, Tauri took 419 MB of PSS on the Home page instead of 397 MB, and Uno
  125 MB instead of 133 MB. Compare the rows of one snapshot, as this table is.
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

## Memory in pull requests

A pull request that rebuilds an implementation gets one comment with the memory of all
of them: a table like [the one above](#results), sorted by the PSS of the Home page,
with the difference from `main` next to the PSS. A new commit replaces that comment
instead of adding another one.

- The workflow of each implementation runs the `measure` scenario of [How it is
  measured](#how-it-is-measured) (Flutter: `tool/measure.py`) and uploads the lines that
  it prints as the artifact `memory-<implementation>`. `measure.sh` writes them to the file
  named in `VFOX_PLUS_MEASURE_OUTPUT` as well.
- [`memory.yml`](../.github/workflows/memory.yml) starts every time one of those workflows
  ends and runs [`memory_report.py`](../.github/scripts/memory_report.py). While another
  one is still running for the latest commit of the pull request, the script stops. The
  run after the last one compares the artifacts of the pull request with those of the
  latest run of the same workflow on `main` and writes the comment. The workflow runs in
  the context of `main`, which is what lets it comment on pull requests of forks, and it
  reads nothing but two lines of numbers from the artifacts. If its token cannot write the
  comment, the run only warns, and the report is in its summary.
- An implementation that the pull request did not rebuild, because its workflow only runs
  for changes in its own paths, shows the numbers of `main` and is marked "(main)".
- The numbers are not those of the table above. They come from the runners of GitHub and
  from one run, not from Docker on a WSL2 host. Compare the rows of one comment with each
  other and with `main`, not with this page.

### The noise

The same code does not use the same memory in every CI run. This is the largest
difference between runs of identical code, from the 107 successful runs of 2026-10-18 and
19 whose source was the same as that of at least one other run (30 groups of 2 to 8 runs),
in MB and as a part of the lowest number:

| Implementation |   Home PSS | Versions PSS |
| -------------- | ---------: | -----------: |
| mygo           |   1 (2.4%) |     0 (0.0%) |
| Uno            |   4 (3.3%) |     5 (3.0%) |
| Gio            | 20 (10.4%) |   23 (11.4%) |
| Dioxus         |   5 (1.8%) |    13 (4.3%) |
| Compose        |  15 (6.1%) |    14 (5.4%) |
| Tauri          |  14 (3.3%) |    14 (2.9%) |
| Capacitor      |   4 (1.0%) |     3 (0.7%) |
| Flutter        | 35 (10.8%) |   43 (13.4%) |
| Flet           |  16 (5.2%) |    25 (7.5%) |
| Wails          |  11 (2.3%) |    13 (2.4%) |

So the comment calls a difference a change, and prints it in bold, only when it is more
than 5 MB and more than 5% of the number on `main`; for Compose and Flet more than 8%, for
Gio more than 12% and for Flutter more than 15%, because they vary more. Of the 940
comparisons of two runs of the same code (the Home and the Versions PSS of every pair, in
both directions), none was a change by these limits. The number of processes never varied,
so a different number is a change too. The sample is small: if the comment reports a
change that the commit cannot have caused, measure how much identical code varies again
and adjust `TOLERANCES` in the script.

### Using it

- The artifacts expire after 90 days. An implementation without a run on `main` for that
  long has "no baseline" until its workflow runs on `main` again.
- `gh workflow run memory.yml -f pull_request=<number>` writes the report of an open pull
  request again. To only see it, run the script with `--dry-run`:

  ```sh
  GH_TOKEN="$(gh auth token)" python3 .github/scripts/memory_report.py --dry-run \
    --repository version-fox/vfox-plus --pull-request <number> \
    --implementations "capacitor compose dioxus flet flutter gio mygo tauri uno wails"
  ```

- The tests of the script run in the `Repository` workflow:
  `python3 -m unittest discover -s .github/scripts`.
- An implementation that is added uploads its numbers as `memory-<directory>` like the
  others and goes into the two lists of `memory.yml`; one that is removed leaves them.

## Updating this page

When an implementation is added, or its memory changes noticeably, run
`<implementation>/sandbox/sandbox.sh measure`, copy the lines of
`dist/sandbox/<implementation>/measure/memory.txt` (`dist/sandbox/measure/memory.txt`
for Flutter) into the table, where the PSS comes before the RSS, and into the Memory
section of the README of the implementation, and keep the table sorted by the PSS of
the Home page. When the sandbox images are built again, measure every implementation
again, one after another and with nothing else running, so that the table stays one
snapshot (see [Reading the numbers](#reading-the-numbers)).
