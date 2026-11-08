# Memory use

> [简体中文](../../zh-CN/dev/memory.md)

Low memory use is a goal of the project (see [the architecture](architecture.md#memory)).
This page records what the application uses on Linux, measured the same way,
so that regressions show up. The numbers are a snapshot: the row is the result
of the `measure` task of the sandbox, run once, and the same numbers are in
the Memory section of the [mygo README](implementation-mygo.md#memory).

## Results

In MB, for the application and all of its child processes together, in Docker on
Ubuntu 26.04 with Xvfb (1280 × 800, no GPU). "Versions" is
the SDKs page with the 872 available versions of Node.js listed.

| Implementation                   | Technology                                                        | Processes | Home PSS | Home RSS | Versions PSS | Versions RSS | The list adds (PSS) |
| -------------------------------- | ----------------------------------------------------------------- | --------: | -------: | -------: | -----------: | -----------: | ------------------: |
| [mygo](../../../mygo/README.md) | mygo 0.7.3, Go 1.27.3, GTK 3 (purego), CPU renderer             |         1 |       51 |       63 |           59 |           73 |                  +8 |

mygo (51 MB of PSS on the Home page, in one process) draws on the processor from a
Go executable of 18 MB and loads GTK 3 from the system. Showing the list of versions
adds 8 MB of PSS because only the rows in view are built.

The build that is measured is the end-to-end build: the release build with the inspector
of the native user interface and the automation plugin of mygo, which the measuring drives
it through. They take up to 1 MB of PSS, which the release build does not.

The Android client (`android/`) is not in this table: it is a WebView around
the web interface, so its memory is the system WebView's plus the page, and it
is measured on the device, not with this method. What its toolkit costs is in
[Views or Compose in the Android app](#views-or-compose-in-the-android-app).

## The lightweight mode

With the web interface and the lightweight mode on, closing the window leaves the
application running without it (see [the architecture](architecture.md#the-window-and-the-lightweight-mode)).
`vfxtool measure-lightweight` measures it the same way as the table above — one process,
smaps_rollup, a fresh home directory with the fake vfox, settled after start — on the
same WSL2 host, on 2026-11-04:

| State | RSS | PSS |
| --- | ---: | ---: |
| Window open, Home page, web interface on | 62 MB | 48 MB |
| Window closed (`lightweight-idle`) | 46 MB | 33 MB |
| Window closed, a browser paired and past its first inventory (`lightweight-loaded`) | 50 MB | 37 MB |

Before the application told the system what it does not want, which is described below, the same two
rows were 57 and 58 MB of RSS and 44 and 45 MB of PSS on the same host, as three runs of each build
agree to the megabyte.

Closing the window frees what the window held: its widgets, the shared-memory buffer
that it drew into, the page, the lists and the listeners (about 5 MB). It does not unload
the toolkit, and most of what stays resident is not memory that the application holds. Of the
57 MB that the window closed left resident, 15.6 MB is private memory, the heap and the stacks.
The rest is clean pages of files, which the system can take back whenever it needs the memory: the code
of the executable that was touched (17 MB: a Go executable is mapped whole, and the system maps
sixteen pages around every one that runs), the fonts that the toolkit mapped (10 MB), GTK 3 and the
libraries it loads. A desktop application keeps them for as long as it runs, so the lightweight mode
saved a few MB, and its point is not memory: it is that the web interface, its sessions and the tasks that
run go on without a window, and that the window comes back from the icon in the tray or by starting the
application again.

### What the application does about the pages of files

When the window has stayed closed for two seconds, the application tells the system that it does not want
those pages for now (`App.scheduleTrim`, `pageOutFiles` of `ui/memory_linux.go`): `madvise` with
`MADV_PAGEOUT`, from Linux 5.4, on the mappings that are the code and the read-only data of files and have
no page that the process wrote. The system takes them back at once instead of when it needs the memory
of another program, and then the resident set is what the application runs. The pages that other programs
map too, as those of the libraries of the desktop, are left where they are by the system, which pages out
only what one process maps. A window that opens in the two seconds, which needs them, cancels it; and the
pages come back when something runs them: the first request that the web interface got after a minute of
idleness took 14 ms in one measurement and under 1 ms in the next, where it took 1 ms before, and a window
opened from the icon in the tray took 0.15 to 0.25 s, as it did (0.19 s). Sampled for three minutes the
resident set does not creep back.

The table above is what it comes to: 11 or 12 MB of RSS and PSS less at rest, 8 MB less after a browser has used
the application. Private memory does not change by it beyond what runs of the same build differ by: the reading of
`/proc/self/smaps` that finds the mappings leaves no garbage that the collection after it does not return
(it is read line by line and taken apart where it lies), and the measurements show 0.2 MB more of private
memory with it. Where the system does not have the call, or the program is not on Linux, nothing happens.

The scheduler of Go uses at most four processors (`limitProcessors` of `main.go`), unless the user sets
`GOMAXPROCS`: the work of the application is waiting, for vfox and for the window, and each processor
that the scheduler uses keeps caches of memory of its own. On a machine of 16 threads this is 0.5 MB less of
private memory, 15.2 against 15.7 MB; on a machine of four or fewer it changes nothing. Setting
`runtime.GOMAXPROCS` when the window closes did not help (0.2 MB worse): it helps only at the start, and
the call at the start of `main` gets most of what the variable does.

### What it does not change

- **The glyph atlas of the toolkit.** The text system of mygo keeps the masks of the glyphs that the window drew in
  an atlas that doubles when it is full, up to 4096 by 4096 bytes, 16 MB, and never shrinks again. A window
  that drew text at many scales (a spring that scales a dialog, or a first frame at an odd size) leaves it at its
  largest, and the application without its window holds those 16 MB. It is the toolkit's, a process-wide
  state that the application cannot reach. It happens in about one of five runs of the end-to-end scenarios,
  which read the window far more often than a person does, and it was never seen in the forty runs that did not
  use the automation plugin. That is why the cost of the icon in the tray is measured with
  `vfxtool measure-lightweight --tray`, which does not.
- **The reclaim of the system.** On this WSL2 host the resident set of an idle process falls by itself after two
  minutes, from 64 MB to 40 and then to 36, with the private memory where it was: the system takes back the
  clean pages that nothing runs. The numbers above are measured after ten seconds, before that, and on a
  machine that does not reclaim, the pages paged out are what the system takes back sooner.

The headless server that the application had before, a binary without the user interface that
was started from the command line, used about 11 MB. It is gone: the application is a
desktop application and has no command line. A process without the toolkit would be a second
program, or the same one started again without its window, and that would drop the paired
sessions and the tasks that run at the moment it takes over.

## The tray icon

The icon in the tray that stands for the application while its window is closed (see
[the architecture](architecture.md#the-tray)) is made when the window closes and removed when
it opens. The rows above are therefore the rows without it where nothing shows an icon, as on a CI
runner: with a window it holds nothing.

Where there is one, it costs little. Measured on the same host before the pages of files were paged
out, with a stand-in StatusNotifier host that registers as `org.kde.StatusNotifierWatcher` (package
`trayhost` of the tooling), the same build, the window closed, as the mean of five runs. That session reads about
6 MB above the table above for the same state, so only the differences between its rows count:

| State | RSS | Private memory |
| --- | ---: | ---: |
| No icon, with the library that draws it loaded (mygo does when the system has it) | 64.1 MB | 15.6 MB |
| The icon in the tray | 64.5 MB | 15.7 MB |
| Without the library, which also means no icon | 63.8 MB | 15.5 MB |
| An icon made where no host shows it, as `libayatana-appindicator3` falls back to `GtkStatusIcon` | 71.2 MB | 17.4 MB |

- **The icon shown** adds 0.4 MB of RSS and 80 KB of private memory. The 0.4 MB are clean pages of
  the libraries that draw it, and about what two runs of the same build differ by.
- **The library** is loaded by mygo when it starts, whenever the system has it and whether an
  icon is made or not, which costs about 0.3 MB of RSS and 70 KB of private memory in every
  state. The Debian package recommends it, so a default installation has it.
- **The fallback** is where memory would go: 7 MB of RSS, 1.8 MB of it private, for the image
  loaders of the desktop that GTK starts to decode the picture of an icon that no panel shows
  (on Ubuntu 26.04 `glycin` alone is 3 MB). That is why `trayHostAvailable` asks the session bus for a host before the
  application asks for an icon, and why there is no icon without one.

With the pages of files paged out and the processors limited, the icon is far inside what the application
saved. With the icon in the tray, in the session of the stand-in host, the application holds 15.2 MB of private
memory on this host where it held 15.6 MB without the icon before, and its resident set is 11 MB smaller.

CI keeps the cost where it is. `vfxtool measure-lightweight --tray` (the last step of the `measure-remote` job)
runs the application twice in the same way, where nothing shows an icon and against the stand-in host, and fails the job
when the icon costs more than 5 MB of resident or 1.5 MB of private memory: room for the noise of a runner, and
less than the fallback costs, so that an icon made the way the fallback makes it is caught. It does not use the
automation plugin, for the glyph atlas above. The fallback itself, which costs its memory where there is no host, is
guarded by the scenario of the icon (`vfxtool e2e-run tray`), which fails when the application makes an icon where
there is no host. Of the other ways to win the 0.4 MB back that were measured, none is in the code except the
two above: `malloc_trim` after the window closes changed nothing, `MALLOC_ARENA_MAX=1` saved 0.1 MB, and a
second `debug.FreeOSMemory` a few seconds after the first saved 0.2 MB of private memory and nothing of
RSS (the collection that follows the paging out is that call). What stays in the Go heap after the window closes
is, among other things, the text engine of mygo (its paragraph cache, its glyph atlas and its SVG paths), which
the application has no way to release.

The runner of CI, Ubuntu 26.04 on GitHub, agrees with these numbers, as two runs of the job on the same code
show: the lightweight mode settles at 45 MB of RSS and 31 to 32 MB of PSS, and at 47 to 48 MB and 34 to 35 MB
with a browser paired, where the table has 46 and 33, and 50 and 37; and the icon in the tray costs 0.5 MB of RSS and
0.06 to 0.18 MB of private memory there, where it cost 0.4 MB and 0.08 MB here.

## Beyond the desktop

The pull-request comment also tracks what the desktop table does not: the
lightweight mode and the Android APK. Each has its own
method, units and noise; compare the rows of one comment, as with the table
above.

- **Lightweight mode.** `vfxtool measure-lightweight` starts the end-to-end build of the
  application under a display (`xvfb-run`) with a fresh home directory, the fake vfox, and
  settings with the web interface and the lightweight mode on. The end-to-end build writes a
  pairing link to the file that `VFOX_PLUS_E2E_LINK` names and closes its window on `SIGUSR1`,
  which a release build has no code for. The tool prints the RSS and PSS of the process tree
  at two settled points: `lightweight-idle` with the window closed and no browser, and
  `lightweight-loaded` with a paired browser past its first inventory over server-sent
  events. A change counts only past 5 MB and 5%, like the desktop. With `--tray` it measures instead what the icon in the
  tray costs (see [The tray icon](#the-tray-icon)).
- **Android client.** The `measure` job of `android.yml` installs the release
  APK on an emulator (API 36, x86_64), cold-starts it through the launcher,
  settles, and reads `dumpsys meminfo`: `apk-cold`, in MB. Emulator boots vary
  more than GitHub runners, so a change counts only past 5 MB and 10%.

In CI the `measure-remote` job of `mygo.yml` uploads `memory-lightweight`,
the `measure` job of `android.yml` uploads `memory-android`,
and `memory.yml` reports when either workflow ends, waiting for a still
running Android run like for the rest. The sections render below the desktop
table with the numbers of the pull request next to `main`.

Coverage gaps: `apk-cold` never loads the system WebView (it is the unpaired
first screen). `vfxtool webheap` can still read the JavaScript heap of the paired `/app`
page locally (next to a screenshot), for diagnosis only, not for the CI
report: it never measures the browser process around the page, so the heap
numbers do not stand for the browser or the Android client.

#### The paired session and the no-WebView control

To see what the system WebView costs, measure the same APK paired, on both
flavors. The `webview` flavor is what users install; the `nowebview` flavor
never constructs a WebView and shows the whole session natively instead —
the same five tabs over the JSON API (in `dev.vfox.vfoxplus.session`), so the
comparison measures the WebView, not missing features. Same package and
entry, so uninstall between the two runs. `vfxtool measure-apk --pair-url`
opens the paired session through an explicit `VIEW` intent (the measurement
entry point of `MainActivity`, the same session as pasting the link and
tapping connect), settles again, and appends the `memory (apk-paired)` line.
Pair against the end-to-end build of the application with
`VFOX_PLUS_E2E_ADDRESSES=10.0.2.2` in its environment, which makes its
certificate and its pairing link name the address where the emulator reaches
the computer; read the link from the file that `VFOX_PLUS_E2E_LINK` names and
mint a fresh one per run by restarting the application. The paired line is
local diagnosis only: never upload a two-line file, and CI keeps measuring
the cold start of the webview flavor.

Measured 2026-11-08 on one emulator (API 36, x86_64, no GPU) against the fake
vfox with the Java and Flutter profiles, release builds with R8, `dumpsys
meminfo` after 20 seconds of settling, paired over plain HTTP through `adb
reverse` so that no trust tap is needed on either flavor:

| Flavor | `apk-cold` PSS | `apk-paired` PSS |
| --- | ---: | ---: |
| `webview` | 37 MB | 84 MB |
| `nowebview` | 38 MB | 39 MB |

The cold start never loads the WebView, so both flavors agree there. The
paired session costs the system WebView about 47 MB PSS here; the native
session adds about 1 MB over its own cold start. A fully native client more
than halves the paired memory, measured with the whole session, not a stub:
home, SDKs with their versions and options, plugins, tasks and settings, with
live events streaming. Whether that ships is a product call — every screen now
exists twice, and the two have to stay alike — but memory no longer argues
against it. (An earlier run the same day read 32/68 MB and 31/32 MB; compare rows measured back to back.)

### Views or Compose in the Android app

The Android app is written in Views, with Material Components. To see what Compose
would cost, its pairing screen was written a second time in Compose, with the same
strings, colours, icons and shapes, and the apps were run on one emulator (API 36,
x86_64, no GPU) as release builds with R8: installed, started through the launcher,
settled for 12 seconds, and the PSS read from `dumpsys meminfo`, three rounds of each
app in turn, on 2026-10-30. "Fresh" is right after the install, where the code is
interpreted and compiled just in time; "compiled" is after
`cmd package compile -m speed -f`, which compiles all of the app ahead of time.

| App | APK (bytes) | PSS fresh (MB) | PSS compiled (MB) |
| --- | ---: | ---: | ---: |
| Views, before the rebuild in Material 3 Expressive | 3,176,011 | 40.8 | 44.5 |
| Views, rebuilt in Material 3 Expressive | 3,228,463 | 43.5 | 46.9 |
| Compose, the pairing screen only | 3,589,067 | 44.4 | 47.4 |

- **Memory does not tell them apart.** The Compose screen took 0.9 MB (fresh) and
  0.5 MB (compiled) more than the Views app, which is the size of the noise: in six
  fresh runs it read between 43.1 and 45.1 MB (its native heap alternated between 9.8
  and 11.5 MB), where the Views app read between 43.4 and 43.6 MB. The rebuild of the
  Views app changed more (+2.4 to +2.7 MB). Memory is no reason to choose either
  toolkit.
- **The size of the APK does.** Adding Compose to the Views app makes it 4,434,858
  bytes, 1.2 MB (37 %) more, and the Compose screen alone, without the scanner, the
  session screen, CameraX and zxing, is already larger than the whole Views app.
- **Compare apps that carry only what they run.** The first Compose build still had
  the Views screens in it, which never ran, and read about 3 MB (fresh) and 6 MB
  (compiled) more than the Views app: code that is in the APK costs memory once the
  system has compiled it.
- **The Compose screen** used the Compose BOM 2026.06.01 (Compose 1.11.4 and
  Material 3 1.4.0, stable API only), activity-compose 1.13.0 and the Compose
  compiler plugin of the pinned Kotlin 2.1.20.
- **The absolute numbers belong to this session**: an earlier one read 30 to 34 MB
  for the Views apps. Compare the rows of the table.
- **Not measured:** Compose with the Expressive components, which the pinned
  toolchain cannot build (see [Why Views and not
  Compose](packaging-android.md#why-views-and-not-compose)), the session screen with
  its WebView, and a phone.

## How it is measured

`vfxtool measure` measures the application:

1. It starts the end-to-end build of the application (`vfxtool build --e2e`, see
   [above](#results)) on the current display, with a new, empty home directory and
   the fake vfox of [`shared/fixtures`](../../../shared/fixtures), which replays
   captured output (so no real vfox runs).
2. It waits for the Home page and for 3 seconds more, so that the application has
   settled, and adds up `Rss` and `Pss` from `/proc/<pid>/smaps_rollup` for the
   application and every process below it.
3. It opens the available versions of Node.js through the elements of the window
   (the automation plugin of mygo), waits for the 872 versions and for 3 seconds, and
   measures again.

The sizes are in MiB, which the tool calls MB. The `measure` task of the sandbox runs
all of this in Docker (`vfxtool sandbox measure`, which needs
nothing but Docker); the numbers above come from a WSL2 host with 16 logical CPUs.

## Reading the numbers

- **RSS and PSS.** RSS counts every page that a process has in memory, including
  those it shares with other processes, so the RSS of several processes counts a
  shared page several times. PSS divides each shared page among the processes that map it, so the
  sum is what the application uses. Compare PSS.
- **No GPU.** The virtual display has no GPU, so everything that draws with OpenGL
  uses the software rasterizer of Mesa (llvmpipe), and its memory, with that of its
  compiler (LLVM), ends up in the measured process. The application draws on the
  processor instead, and keeps GTK from loading OpenGL: a GDK that opens its OpenGL
  connection maps Mesa, and with the software driver of Mesa its compiler LLVM,
  which is more than the application takes (the same build took about 78 MB of PSS
  on the Home page when GDK did, and about 43 MB when it did not, with every other
  setting equal). An interface this size does not need the GPU, and the memory
  does not depend on the driver of the machine. On a machine with a GPU the numbers
  may differ; measure there when it matters.
- **One measurement per row.** Nothing is averaged over runs, so small differences
  mean little. mygo was measured repeatedly in CI (see below): identical code
  varied by at most 1 MB of PSS on either page.
- **One snapshot at a time.** The numbers of the same code change from one
  snapshot to the next: measured again in sandbox images built from the Ubuntu
  archive of a later day, the same build may take a few MB more or less. Compare
  the rows of one snapshot, as this table is.
- **What is not in them.** Only the process tree of the application is measured: the
  X server, the session bus and the desktop portal are not, and neither is vfox,
  which is the fake one here, so a real command adds the process of vfox while it
  runs.

## Memory in pull requests

A pull request that rebuilds the application gets one comment with its memory: the
table [above](#results) with the difference from `main` next to the PSS. A new
commit replaces that comment instead of adding another one.

- The workflow of the implementation ([`mygo.yml`](../../../.github/workflows/mygo.yml))
  runs `vfxtool measure` ([How it is measured](#how-it-is-measured)) and
  uploads the lines that it prints as the artifact `memory-mygo`. `vfxtool measure`
  writes them to the file named in `VFOX_PLUS_MEASURE_OUTPUT` as well. The same
  workflow's `measure-remote` job uploads `memory-lightweight`, and the
  `measure` job of [`android.yml`](../../../.github/workflows/android.yml) uploads
  `memory-android`; the comment renders them as the [Beyond the desktop](#beyond-the-desktop)
  sections below the table.
- [`memory.yml`](../../../.github/workflows/memory.yml) starts when that workflow ends
  and runs `vfxtool memory-report`. It compares
  the artifact of the pull request with the latest run of the same workflow on
  `main` and writes the comment. The workflow runs in the context of `main`, which
  is what lets it comment on pull requests of forks, and it reads nothing but two
  lines of numbers from the artifact. If its token cannot write the comment, the
  run only warns, and the report is in its summary.
- The numbers are not those of the table above. They come from the runners of GitHub and
  from one run, not from Docker on a WSL2 host. Compare the rows of one comment with each
  other and with `main`, not with this page.

### The noise

The same code does not use the same memory in every CI run. Comparing runs of
identical code, the Home and the Versions PSS never differed by more than 1 MB
(2.4 % of the lowest number on the Home page; 0 MB with the versions listed).

So the comment calls a difference a change, and prints it in bold, only when it is more
than 5 MB and more than 5% of the number on `main`. The number of processes never varied,
so a different number is a change too. The sample is small: if the comment reports a
change that the commit cannot have caused, measure how much identical code varies again
and adjust `Tolerances` in `vfxtool memory-report`.

### Using it

- The artifacts expire after 90 days. Without a run on `main` for that long,
  the report has "no baseline" until the workflow runs on `main` again.
- `gh workflow run memory.yml -f pull_request=<number>` writes the report of an open pull
  request again. To only see it, run it with `--dry-run`:

  ```sh
  GH_TOKEN="$(gh auth token)" vfxtool memory-report --dry-run \
    --repository version-fox/vfox-plus --pull-request <number> \
    --implementations "mygo"
  ```

- The tests of the tool run in the `Repository` workflow:
  `go -C shared/tooling test ./...`.

## Updating this page

When the memory changes noticeably, run
`vfxtool sandbox measure`, copy the lines of
`dist/sandbox/mygo/measure/memory.txt` into the table, where the PSS comes before
the RSS, and into the Memory section of the README of the implementation. When the
sandbox images are built again, measure again, so that the table stays one
snapshot (see [Reading the numbers](#reading-the-numbers)).
