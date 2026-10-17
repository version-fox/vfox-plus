# vfox output fixtures

vfox prints human-oriented text, and every GUI in this repository has to parse
it. This directory holds that text, captured from real vfox runs, together
with the results a correct parser must produce. Each implementation runs its
parsers against these files in its unit tests, so all implementations agree on
what vfox output means.

## Layout

```text
fixtures/
├── capture.sh                 # Recreates the captures for the installed vfox
├── fake-vfox.sh               # A stand-in for vfox that replays the captures
└── vfox-<version>/
    ├── cases.json             # Commands, exit codes and expected results
    ├── <case>.stdout          # Standard output of the command
    └── <case>.stderr          # Standard error of the command
```

## How the output was captured

`capture.sh` runs every command

- with `NO_COLOR=1`, without a terminal on stdin or stdout, which is how a GUI
  runs vfox (see [the CLI contract](../../docs/vfox-cli-contract.md));
- with `HOME=/tmp/vfox-sb/home`, a throw-away sandbox, and without
  `VFOX_HOME`, so the real vfox installation is never touched and the captured
  paths are stable;
- from the working directory `/`.

Some versions are faked by creating their directories, so that the tree printed
by `vfox list` has several versions per plugin without downloading them.

Some captures contain carriage returns, escape sequences or the field separator
`0x1F`. So that every file stays plain text, `capture.sh` writes each control
character other than tab and line feed as `\x` followed by two lowercase
hexadecimal digits (a carriage return becomes `\x0d`), and each backslash as
`\\`. Decode a capture before passing it to a parser, by replacing both escapes
in a single pass from left to right; this gives back the exact bytes that vfox
printed.

## `cases.json`

| Field         | Meaning                                                                   |
| ------------- | ------------------------------------------------------------------------- |
| `name`        | Base name of the `.stdout` and `.stderr` files.                           |
| `command`     | Arguments passed to `vfox`.                                               |
| `environment` | Variables set in addition to the ones listed above, if any.               |
| `exitCode`    | Exit code of the command.                                                 |
| `expect`      | Expected parser results, keyed by parser. An empty object means the case is kept as a reference only. |

Expected results by parser:

| Key             | Input            | Expected value                                                             |
| --------------- | ---------------- | -------------------------------------------------------------------------- |
| `version`       | stdout           | The version number.                                                        |
| `current`       | stdout           | Plugin name to current version; `null` when no version is in effect.      |
| `list`          | stdout           | Plugin name to installed versions, in output order.                       |
| `search`        | stdout           | `count` of versions and selected `items` by index.                        |
| `available`     | stdout           | `count` of plugins and selected `items` by index.                         |
| `pluginInfo`    | stdout           | The fields of the `--format` template.                                    |
| `installPath`   | stdout           | The directory of a version, or `null` for `notfound`.                     |
| `config`        | stdout           | Every `key = value` entry, in output order.                               |
| `finalProgress` | stderr lines     | The last download progress (0–1) found, or `null` if there is none.       |
| `error`         | stdout + stderr  | The one-line error summary shown to users.                                |

Parsers must skip lines they do not understand instead of failing, and the
error summary follows the rules in the
[CLI contract](../../docs/vfox-cli-contract.md#errors).

## Updating for a new vfox release

1. Install the new vfox release and run `shared/fixtures/capture.sh`. It needs
   network access, takes about a minute and writes `vfox-<version>/`.
   Alternatively, put the new version and the checksum of its Linux archive in
   [`flutter/versions.env`](../../flutter/versions.env) and run
   `flutter/sandbox/sandbox.sh fixtures`: it captures in a container, where
   nothing has to be installed, and writes to `dist/sandbox/fixtures/`.
2. Copy `cases.json` from the previous version, then compare the new output
   with the old output (`diff -r`) and adjust the expected values. Keep the
   directory of the previous version when its output differs, so that the
   parsers stay tested against both (vfox 1.0.13 moved errors from standard
   output to standard error).
3. Run the parser tests of every implementation, for example
   `cd flutter && flutter test test/core`.
4. Document any change of the output format in
   [the CLI contract](../../docs/vfox-cli-contract.md).

## The fake vfox

[`fake-vfox.sh`](fake-vfox.sh) behaves like vfox 1.0.13 towards a GUI: it
answers the read-only commands with the captures of `vfox-1.0.13/` (or of
`VFOX_PLUS_FIXTURES`), pretends to install versions with a progress bar on
standard error, and fails like vfox when `vfox use` runs without the hook
environment. It also checks what the CLI contract requires of the caller:
`NO_COLOR=1`, no `__VFOX_PID` of an activated shell and the working directory
`/` for commands that are not about a project. It keeps no state.
Implementations use it to try the application without touching a real vfox
installation, in end-to-end tests and in
[`../tools/measure.sh`](../tools/measure.sh).
