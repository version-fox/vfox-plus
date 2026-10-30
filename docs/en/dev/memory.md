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
the pairing server, so its memory is the system WebView's plus the page, and it
is measured on the device, not with this method. What its toolkit costs is in
[Views or Compose in the Android app](#views-or-compose-in-the-android-app).

## The pairing server and the background service

`vfox-plus serve` with a model and the headless `vfox-plus-serve serve`
idle at about the same order of magnitude, measured the same way — one
process, smaps_rollup, a fresh home directory without vfox, settled after
start:

| Server | Binary | RSS | PSS |
| --- | --- | ---: | ---: |
| `vfox-plus serve` | desktop binary, 18 MB | 16 MB | 16 MB |
| `vfox-plus-serve serve` | headless binary, 12 MB | 11 MB | 11 MB |

The headless binary (`mygo/cmd/vfox-plus-serve`: the same `pair`, `serve`
and `service` commands without the mygo user interface) sheds about a third
of the binary and of the resident set; the idle model itself accounts for
only about a megabyte either way, and the rest is the Go runtime and the
touched code pages of the binary. There is no single stable number: a reading
right after start comes out lower, before all code pages are faulted in, and
garbage collection timing moves the heap by megabytes. Treat these as one
snapshot, like the table above, and re-measure rather than compare across
machines. The desktop embeds the same server in its own process when remote
access is switched on, so it adds no process and only a listener with idle
goroutines; with remote access off (the default) it adds nothing at all.

## Beyond the desktop

The pull-request comment also tracks what the desktop table does not: the
headless server, the web client heap, and the Android APK. Each has its own
method, units and noise; compare the rows of one comment, as with the table
above.

- **Headless server.** `vfxtool measure-serve` starts `vfox-plus-serve serve`
  on loopback with a fresh home directory and the fake vfox, and prints the
  RSS and PSS of the server process tree at two settled points: `serve-idle`
  with no browser, and `serve-loaded` with a paired browser past its first
  inventory over server-sent events. A change counts only past 5 MB and 5%,
  like the desktop.
- **Web client heap.** `vfxtool webheap` (which needs a Chrome and nothing else)
  drives the paired `/app` page in headless Chrome through the Chrome DevTools
  Protocol and writes the JavaScript heap in bytes at two points: `app` after
  the SDK list settles, and `app-settled` after twenty seconds idle on the open
  page with live events, which catches leaks in the event stream. The report
  shows KB; a change counts only past 100 KB and 5%.
- **Android client.** The `measure` job of `android.yml` installs the release
  APK on an emulator (API 36, x86_64), cold-starts it through the launcher,
  settles, and reads `dumpsys meminfo`: `apk-cold`, in MB. Emulator boots vary
  more than GitHub runners, so a change counts only past 5 MB and 10%.

In CI the `measure-remote` job of `mygo.yml` uploads `memory-serve` and
`memory-web`, the `measure` job of `android.yml` uploads `memory-android`,
and `memory.yml` reports when either workflow ends, waiting for a still
running Android run like for the rest. The sections render below the desktop
table with the numbers of the pull request next to `main`.

Coverage gaps: there is no paired-WebView APK state (the emulator has no
server to pair with, so `apk-cold` is the unpaired first screen), and the heap
is measured in desktop Chrome, not in the system WebView that the APK embeds,
whose collector may behave differently.

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
  workflow's `measure-remote` job uploads `memory-serve` and `memory-web`, and the
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
