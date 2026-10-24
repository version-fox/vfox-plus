# vfox CLI contract

This document specifies how vfox-plus drives the vfox command
line tool: which commands it runs, in which environment, how it reads their
output and how it reports errors. It describes **vfox 1.0.13** and notes where
1.0.12 behaves differently; the captured outputs of both versions are in
[`shared/fixtures`](../shared/fixtures), together with the expected parser
results in `cases.json`.

The minimum supported vfox version is **1.0.0** (the `.vfox.toml` layout and the
non-interactive behaviour used below exist since then; `info --format` since
0.8.0).

## Running vfox

### Process

- Spawn the executable directly, with an argument vector. Never go through a
  shell, so that names and versions cannot be interpreted by it.
- Close standard input immediately. vfox detects that it does not run in a
  terminal and then never prompts; an open stdin would make an unexpected
  prompt hang.
- Read standard output and standard error as UTF-8, replacing malformed bytes.
- Validate user-provided values before passing them: plugin names match
  `^[A-Za-z0-9][A-Za-z0-9._+-]*$`, versions must not start with `-` and must not
  contain whitespace or `@`, configuration keys are dot-separated identifiers.
  This prevents option injection such as a plugin called `--all`.

### Environment

Start from the environment of the GUI, then:

| Variable                                                  | Action      | Why                                                                                                                                  |
| --------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `__VFOX_SHELL`, `__VFOX_PID`, `__VFOX_CURTMPPATH`, `__VFOX_INITIALIZED` | remove      | Set by an activated shell. A GUI started from such a terminal would otherwise act on that terminal's session scope.                   |
| user-defined variables (`extraEnvironment` setting)      | add         | Download mirrors such as `VFOX_NODEJS_MIRROR` live in shell profiles, which desktop launchers do not read.                           |
| `NO_COLOR`                                                | set to `1`  | Disables colours in most output (https://no-color.org).                                                                             |
| `__VFOX_SHELL`                                            | set to `bash`, for `vfox use` only | See [Selecting a version](#selecting-a-version).                                                                  |

Everything else passes through, including `VFOX_HOME`: when it is set, vfox
keeps plugins, caches and its configuration in that shared directory rather than
in `~/.vfox`.

### Working directory

vfox reads the *project* configuration (`.vfox.toml`, and legacy files such as
`.nvmrc` when enabled) from its working directory, without looking at parent
directories. Run every command that is not about a project in a directory that
cannot contain such files: **the filesystem root `/`**. Do not use the home
directory: users of asdf often have a `~/.tool-versions`.

Commands for a project scope run in the project directory.

### Session directories

Every vfox invocation creates `~/.vfox/tmp/<yyyymmdd>-<pid>`, where `<pid>` is
the PID of its parent process. Because the GUI spawns vfox directly, all
invocations share one directory per day; vfox removes stale ones itself.

## Output conventions

- Errors returned by a command are printed with exit code 1: to **standard
  error** since vfox 1.0.13, to **standard output** (`fmt.Println`) up to
  1.0.12. What a command prints before it fails, such as the reason why an
  installation failed, stays on standard output.
- Usage errors (`cli.Exit`) are printed to **standard error**, with exit code 1.
- The download progress bar is printed to **standard error**. It ignores
  `NO_COLOR`, contains ANSI colour codes and redraws itself with carriage
  returns: `\rDownloading...  42% [====>     ] (1.2 MB/s) [3s:4s]\r      \r`.
- Tree and status characters are UTF-8 (`├ └ │ ─ ┬ ✓ ✗ —`).

Parsers therefore:

1. strip ANSI escape sequences (CSI, OSC and two-byte escapes) even with
   `NO_COLOR`;
2. split lines on `\n` **and** on a lone `\r`;
3. skip every line they do not recognise instead of failing.

### Errors

The message shown to the user for a failed command (exit code ≠ 0) is computed
from its output:

1. Take the meaningful lines of standard output followed by those of standard
   error: non-blank and not a download progress line.
2. Use the last line, unless it is the generic trailer `failed to install …`
   and there is a line before it, which is then more specific
   (`failed to download maven@3.9.16 file, err:unexpected EOF`). Since vfox
   1.0.13 the trailer is on standard error and that line on standard output.
3. If no line is left, report the exit code.

If the executable cannot be started at all, report vfox as missing.

## Queries

Queries are read-only and may run concurrently.

### Version

```console
$ vfox --version
vfox version 1.0.13
```

The version is the first `\d+.\d+.\d+…` token. Fixture: `version`.

### Installed SDKs: `vfox list` and `vfox current`

Both commands are run; their results are merged by plugin name.

```console
$ vfox list
All installed sdk versions
├─┬maven
│ ├──3.9.16
│ └──3.8.9
├──mvn2
└─┬nodejs
  ├──24.11.1
  └──22.21.0
```

Lines matching `^([│ ]*)[├└]─[─┬](.+)$` are items; the length of the first
group divided by two is the depth. Depth 0 is a plugin (a leaf `├──name` is a
plugin without versions), depth 1 a version of the last plugin. Order is
preserved. When no plugin is installed, `vfox list` fails with
`you don't have any sdk installed yet` while `vfox current` prints nothing; this
is an empty result, not an error. Fixtures: `list`, `list-empty`.

```console
$ vfox current
maven -> v3.9.16
mvn2 -> N/A
nodejs -> v24.11.1
```

Each line is `<name> -> v<version>` or `<name> -> N/A ` (note the trailing
space). Strip **one** leading `v`. `vfox current` reflects the scope chain
global < session < project, which is why it must run in `/`. Fixtures:
`current`, `current-unset`, `current-empty`.

### Plugin information

```console
$ vfox info nodejs --format '{{.Name}}{{"\x1f"}}{{.Version}}{{"\x1f"}}{{.Homepage}}{{"\x1f"}}{{.InstallPath}}{{"\x1f"}}{{.Description}}'
```

The Go template separates the fields with the ASCII unit separator `0x1F`, which
cannot occur in them. The description is the last field and may contain line
breaks; join any further fields back into it. Fixtures: `info`, `info-unknown`.

`vfox info <name>@<version>` prints the installation directory of a version, or
`notfound`. Fixtures: `info-version-path`, `info-version-not-installed`.

### Installable versions

```console
$ vfox search nodejs
Available versions:
 - 26.12.0 [npm 11.20.0]
 - 24.22.0 (LTS) [npm 11.19.1]
 - 3.9.16 (installed)
```

Because standard output is not a terminal, vfox prints the plain list instead of
the interactive picker. Each entry starts with ` - ` and has the form
`<version>[ (<note>)][ [<tool> <version>,…]][ (installed)]`, parsed from the
right: the `(installed)` suffix, then the bracketed additions (comma-separated,
each `<tool> <version>`, the version may be empty), then a parenthesised note.
The version is the first token. Fixtures: `search-maven`,
`search-maven-installed`, `search-nodejs`, `search-unknown`.

The installed state of a version should be taken from `vfox list` rather than
from `(installed)`, so that it stays current without searching again. A search
may take seconds, because plugins fetch the list from the network unless vfox
has it cached (`cache.availableHookDuration`).

### Plugin registry

```console
$ vfox available
AVAILABLE PLUGINS

  bun                 ✗  https://github.com/arfaWong/vfox-bun
  nodejs              ✓  https://github.com/version-fox/vfox-nodejs

  Use 'vfox add <plugin>' to install
```

Entries match `^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$`: name, official (✓) and
homepage. Fixture: `available`.

### Configuration

```console
$ vfox config --list
plugin.http.timeout = 0s
proxy.url =
proxy.enable = false
…
```

Each line is `<key> = <value>`; the value may be empty. Unknown keys must be
shown and editable as text, so that the GUI keeps working when vfox adds
settings. The keys of vfox 1.0.13, the same as in 1.0.12:

| Key                           | Type                                  | Notes                                             |
| ----------------------------- | ------------------------------------- | ------------------------------------------------- |
| `proxy.enable`                | `true` / `false`                      |                                                   |
| `proxy.url`                   | URL                                   |                                                   |
| `registry.address`            | URL                                   | Empty means the official registry.               |
| `storage.sdkPath`             | directory                             | Must exist and be writable.                      |
| `legacyVersionFile.enable`    | `true` / `false`                      | `.nvmrc`, `.node-version`, `.sdkmanrc`, …        |
| `legacyVersionFile.strategy`  | `specified` / `latest_installed` / `latest_available` |                                  |
| `cache.availableHookDuration` | duration (`12h`), `-1` forever, `0` off |                                                |
| `plugin.http.timeout`         | duration, `0s` = default (30 s)       |                                                   |
| `plugin.http.downloadTimeout` | duration, `0s` = default (30 min)     |                                                   |
| `gitignore.enable`            | `true` / `false`                      | Adds `.vfox/` to `.gitignore` for project scopes. |

`vfox config <key> <value>` sets a value and `vfox config --unset <key>` resets
it. Invalid values fail with vfox's message, for example
`time: invalid duration "notaduration"` (fixture
`config-set-invalid-duration`). An unknown key is **silently accepted** (exit
code 0), so only offer keys that `--list` printed. Fixtures: `config-list`,
`config-set`, `config-get`.

## Commands

State-changing commands run through the task queue, one at a time, with their
output streamed to the user.

| Operation        | Command line                                         | Working directory | Notes |
| ---------------- | ---------------------------------------------------- | ----------------- | ----- |
| Install          | `vfox install --yes <name>@<version>`                | `/`               | Progress on standard error. Without a version vfox fails (`install-without-version`). |
| Uninstall        | `vfox uninstall <name>@<version>`                    | `/`               | Uninstalling the current version makes vfox switch to another installed one. |
| Use (global)     | `vfox use --global <name>@<version>`                 | `/`               | Needs `__VFOX_SHELL`, see below. |
| Use (project)    | `vfox use --project <name>@<version>`                | project directory | Writes `.vfox.toml`, links `.vfox/sdks`, may edit `.gitignore`. |
| Stop using       | `vfox unuse --global <name>`                         | `/` or project    | |
| Add plugin       | `vfox add <name>`                                    | `/`               | |
| Add from source  | `vfox add --source <url-or-path> [--alias <alias>] [<name>]` | `/`       | The name defaults to the plugin's own. Pass a copy of a local `.lua` file, see [the quirks](#known-quirks). |
| Remove plugin    | `vfox remove --yes <name>`                           | `/`               | Also deletes every installed version. Without `--yes` it fails (`remove-without-yes`). |
| Update plugin    | `vfox update <name>`                                 | `/`               | |
| Update all       | `vfox update --all`                                  | `/`               | |

Options always precede the positional arguments.

### Commands a GUI does not run

| Command                                      | Why |
| -------------------------------------------- | --- |
| `vfox activate`, `vfox env`                  | Shell integration: they print shell code or a session environment. The GUI shows the line to add to `~/.bashrc` instead. |
| `vfox use --session`, `vfox exec`, `vfox cd` | Only meaningful inside a terminal session. |
| `vfox upgrade`                               | Replaces the vfox executable, which a package manager or an installer usually owns; vfox is upgraded the way it was installed. |

### Selecting a version

On Linux and macOS, `vfox use` refuses to run outside an activated shell:

```console
$ vfox use --global maven@3.9.16
vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'
```

vfox only checks that `__VFOX_SHELL` is non-empty. With `--global` or
`--project`, `use` then updates the corresponding `.vfox.toml` and the symlinks
in `~/.vfox/sdks` or `<project>/.vfox/sdks` — exactly what a GUI needs — and
prints `Now using maven@3.9.16.` The GUI therefore sets `__VFOX_SHELL=bash` for
`vfox use` (and only for it). It never uses the session scope, which has no
meaning without a shell. Fixtures: `use-global`, `use-without-hook`.

Shells started *afterwards* pick up the global versions once vfox is activated
in them (`eval "$(vfox activate bash)"`); the GUI tells the user so.

### Download progress

While installing, read standard error line by line (splitting on `\r`) and match
`Downloading\.\.\.\s+(\d{1,3})%`. Show the last value as determinate progress,
and do not add these lines to the visible log. Fixture: `install`.

### Cancelling

A running command is cancelled by sending `SIGTERM` to vfox; the task then
counts as cancelled whatever the exit code. What an interrupted `vfox install`
leaves behind depends on the version:

- up to 1.0.12, vfox catches the signal, deletes the partial version directory
  and exits with code 0;
- since 1.0.13, the signal ends vfox. The partial directory stays, next to a
  `.v-<version>.installing` marker; `vfox list` does not show that version, and
  the next `vfox install` of it removes the leftovers first.

Since 1.0.13, vfox also refuses to install a version while another vfox
process, for example in a terminal, installs it:
`cannot install <name>@<version>: installation is in progress or cannot be locked: …`.

## Known quirks

These hold for vfox 1.0.12 and 1.0.13.

- `vfox add a b` reports per-plugin failures but exits with 0. The GUI adds one
  plugin per command.
- If a plugin's pre-install hook cannot reach the network (for example through a
  broken proxy), `vfox install` may report `Install maven@ success!` with an
  empty version. Treat the task output, not only the exit code, as what the
  user sees.
- `vfox config` with an unknown key succeeds without doing anything.
- `vfox add --source <path>.lua` *moves* the file: it renames it into a new
  directory in its temporary directory (`~/.vfox/tmp`, or `~/.version-fox/tmp`
  when that legacy directory exists). The file disappears from where the user
  keeps it, and the rename fails when the file is on another filesystem, such as
  a `/tmp` on tmpfs. Local `.zip` archives and URLs are only read. The GUI
  copies a local `.lua` file into a new directory in vfox's temporary directory
  and passes the copy; vfox never cleans up directories there whose names do not
  start with a date.
- Output is split over both streams: `vfox --version` goes to standard output,
  progress and usage errors to standard error, and the errors that commands
  return to standard output up to 1.0.12 but to standard error since 1.0.13
  (see [Output conventions](#output-conventions)).

## Testing against the real vfox

Tests that run the real vfox must not touch the vfox installation of whoever
runs them. Run vfox with `HOME` set to a new, empty directory and without
`VFOX_HOME`. The offline plugin in [`shared/test-plugin`](../shared/test-plugin)
together with a local HTTP server covers the registry, adding plugins,
searching, installing with download progress, using, uninstalling and removing,
without network access.

## Updating this contract

When vfox changes, capture its output again with
[`shared/fixtures/capture.sh`](../shared/fixtures/capture.sh), update
`cases.json` and this document, and run the parser tests. See [`shared/fixtures/README.md`](../shared/fixtures/README.md).
