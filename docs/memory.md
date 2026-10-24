# Memory use

Low memory use is a goal of the project (see [the architecture](architecture.md#memory)).
This page records what the application uses on Linux, measured the same way,
so that regressions show up. The numbers are a snapshot: the row is the result
of the `measure` task of the sandbox, run once, and the same numbers are in
the Memory section of the [mygo README](../mygo/README.md#memory).

## Results

In MB, for the application and all of its child processes together, in Docker on
Ubuntu 26.04 with Xvfb (1280 × 800, no GPU). "Versions" is
the SDKs page with the 872 available versions of Node.js listed.

| Implementation                   | Technology                                                   | Processes | Home PSS | Home RSS | Versions PSS | Versions RSS | The list adds (PSS) |
| -------------------------------- | ------------------------------------------------------------ | --------: | -------: | -------: | -----------: | -----------: | ------------------: |
| [mygo](../mygo/README.md#memory) | mygo 0.6, Go 1.27.3, GTK 3 (purego), CPU renderer            |         1 |       41 |       54 |           47 |           60 |                  +6 |

mygo (41 MB of PSS on the Home page, in one process) draws on the processor from a
Go executable of 12 MB and loads GTK 3 from the system. Showing the list of versions
adds 6 MB of PSS because only the rows in view are built.

## How it is measured

[`shared/tools/measure.sh`](../shared/tools/measure.sh) measures the application:

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

The sizes are in MiB, which the script calls MB. The `measure` task of the sandbox runs
all of this in Docker (`mygo/sandbox/sandbox.sh measure`, which needs
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

- The workflow of the implementation ([`mygo.yml`](../.github/workflows/mygo.yml))
  runs the `measure` scenario of [How it is measured](#how-it-is-measured) and
  uploads the lines that it prints as the artifact `memory-mygo`. `measure.sh`
  writes them to the file named in `VFOX_PLUS_MEASURE_OUTPUT` as well.
- [`memory.yml`](../.github/workflows/memory.yml) starts when that workflow ends
  and runs [`memory_report.py`](../.github/scripts/memory_report.py). It compares
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
and adjust `TOLERANCES` in the script.

### Using it

- The artifacts expire after 90 days. Without a run on `main` for that long,
  the report has "no baseline" until the workflow runs on `main` again.
- `gh workflow run memory.yml -f pull_request=<number>` writes the report of an open pull
  request again. To only see it, run the script with `--dry-run`:

  ```sh
  GH_TOKEN="$(gh auth token)" python3 .github/scripts/memory_report.py --dry-run \
    --repository version-fox/vfox-plus --pull-request <number> \
    --implementations "mygo"
  ```

- The tests of the script run in the `Repository` workflow:
  `python3 -m unittest discover -s .github/scripts`.

## Updating this page

When the memory changes noticeably, run
`mygo/sandbox/sandbox.sh measure`, copy the lines of
`dist/sandbox/mygo/measure/memory.txt` into the table, where the PSS comes before
the RSS, and into the Memory section of the README of the implementation. When the
sandbox images are built again, measure again, so that the table stays one
snapshot (see [Reading the numbers](#reading-the-numbers)).
