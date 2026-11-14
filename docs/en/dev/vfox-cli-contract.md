# vfox CLI contract

> [简体中文](../../zh-CN/dev/vfox-cli-contract.md)

This document specifies how vfox-plus drives the vfox command
line tool: which commands it runs, in which environment, how it reads their
output and how it reports errors. It describes **vfox 1.1.0**; the captured
outputs are in [`shared/fixtures`](../../../shared/fixtures), together with the
expected parser results in `cases.json`.

The minimum supported vfox version is **1.1.0**: 1.1.0 adds the link,
doctor, alias, path.mode, install --use commands and the plugin Detect and
Fingerprint hooks the application relies on, and earlier versions are not
supported.

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
| `NO_COLOR`                                                | set to `1`  | Disables colours in most output (<https://no-color.org>).                                                                             |
| `__VFOX_SHELL`                                            | set to `bash`, for `vfox use` only | See [Selecting a version](#selecting-a-version).                                                                  |

Everything else passes through, including `VFOX_HOME`: when it is set, vfox
keeps plugins, caches and its configuration in that shared directory rather than
in `~/.vfox`.

### Working directory

vfox reads the *project* configuration (`.vfox.toml`, and legacy files such as
`.nvmrc` when enabled) from its working directory, without looking at parent
directories. Run every command that is not about a project in a directory that
cannot contain such files: **the filesystem root** (`/` on Unix, the system
drive root such as `C:\` on Windows). Do not use the home
directory: users of asdf often have a `~/.tool-versions`.

Commands for a project scope run in the project directory.

### Session directories

Every vfox invocation creates `<home>/.vfox/tmp/<yyyymmdd>-<pid>` (`~/.vfox/tmp`
on Unix, `%USERPROFILE%/.vfox/tmp` on Windows), where `<pid>` is
the PID of its parent process. Because the GUI spawns vfox directly, all
invocations share one directory per day; vfox removes stale ones itself.

## Output conventions

- Errors returned by a command are printed with exit code 1 to **standard
  error**. What a command prints before it fails, such as the reason why an
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

An error that a hook of a plugin raises is printed with a Lua traceback:

```text
plugin [PreInstall] method error: ~/.vfox/plugin/java/hooks/pre_install.lua:42: No JDK found for 99-tem on linux/amd64. …
stack traceback:
	[G]: in function 'error'
	~/.vfox/plugin/java/hooks/pre_install.lua:42: in main chunk
	[G]: ?
```

so that rule 1 also leaves out the `stack traceback:` line and the tab-indented lines
after it, and the file and the line number of `plugin [<hook>] method error:` are
dropped: what remains is the sentence of the plugin, `No JDK found for 99-tem …`,
instead of `[G]: ?`. This matters for installing by name, where a typo is the
plugin's to report. Fixture: `install-plugin-error`.

If the executable cannot be started at all, report vfox as missing.

## Queries

Queries are read-only and may run concurrently.

### Version

```console
$ vfox --version
vfox version 1.1.0
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
plugin without versions), depth 1 a version of the last plugin. Since vfox
1.1.0 a version carries its link and alias annotations, which are left out:
`17-system (linked → /usr/lib/jvm/java-17) [alias: j17]` parses as
`17-system`. Order is preserved. When no plugin is installed, `vfox list` fails with
`you don't have any sdk installed yet` while `vfox current` prints nothing; this
is an empty result, not an error. Fixtures: `list`, `list-empty`.

```console
$ vfox current
maven -> 3.9.16
mvn2 -> N/A
nodejs -> 24.11.1
```

Each line is `<name> -> <version>` or `<name> -> N/A`. vfox 1.0.x prefixes the
version with a `v`, which is stripped again; vfox 1.1.0 prints the stored name
verbatim. `vfox current <sdk>` prints `-> <version>` for one plugin.
`vfox current` reflects the scope chain global < session < project, which is
why it must run in `/`. Fixtures: `current`, `current-unset`, `current-empty`.

### Plugin information

```console
$ vfox info nodejs --format '{{.Name}}{{"\x1f"}}{{.Version}}{{"\x1f"}}{{.Homepage}}{{"\x1f"}}{{.InstallPath}}{{"\x1f"}}{{.Description}}'
```

The Go template separates the fields with the ASCII unit separator `0x1F`, which
cannot occur in them. The description is the last field and may contain line
breaks; join any further fields back into it. Fixtures: `info`, `info-unknown`.

`vfox info <name>@<version>` prints the installation directory of a version, or
`notfound`. Fixtures: `info-version-path`, `info-version-not-installed`.

`vfox info --json <name>[@<version>]` answers a JSON object with at least
`name`, `version`, `installed`, `origin` (`installed` or `linked`), `target`
(the linked directory, empty for installs) and `home` (the runtime directory,
for example `JAVA_HOME`). The GUI reads `origin` and `target` to tell linked
versions apart. Fixture: `info-json`.

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

#### Search arguments

```console
$ vfox search java
$ vfox search java tem
$ vfox search java all
```

Whatever follows the plugin name is handed to the plugin's `Available` hook as
`ctx.args`; most plugins ignore it, but vfox-java takes the short or the long name
of a distribution (`tem`, `temurin`), or `all`, and lists that distribution
instead of OpenJDK from jdk.java.net, which is what it lists without an argument
(on Linux for loong64 `loongnix`, `lnxabi1` and `lnxabi2`). vfox caches the answer
once per list of arguments. The application passes one argument, taken from the
choices of the [profile](architecture.md#plugin-profiles) of the plugin, and none for the
default. Each argument is checked like a plugin name is: not empty, not starting
with `-`, without whitespace or control characters. The list of `java all` has 2,683
entries, the notes have the forms `LTS`, `JavaFX` and `LTS, JavaFX`, and the
version names carry the distribution: `21.0.12.1+1-tem`, `25.0.4+9-librca-fx`.
Fixtures: `search-java`, `search-java-tem`, `search-java-all`, `search-flutter`
(whose notes are channels, with `[dart <version>]` as the bundled tool, an empty
`[dart ]` for the oldest releases, and `(OpenHarmony)` for the builds of the fork).

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
settings. The keys of vfox 1.1.0:

| Key                           | Type                                  | Notes                                             |
| ----------------------------- | ------------------------------------- | ------------------------------------------------- |
| `proxy.enable`                | `true` / `false`                      |                                                   |
| `proxy.url`                   | URL                                   |                                                   |
| `registry.address`            | URL                                   | Empty means the official registry.               |
| `storage.sdkPath`             | directory                             | Must exist and be writable.                      |
| `legacyVersionFile.enable`    | `true` / `false`                      | `.nvmrc`, `.node-version`, `.sdkmanrc`, `gradle/wrapper/gradle-wrapper.properties`, … |
| `legacyVersionFile.strategy`  | `specified` / `latest_installed` / `latest_available` |                                  |
| `cache.availableHookDuration` | duration (`12h`), `-1` forever, `0` off |                                                |
| `plugin.http.timeout`         | duration, `0s` = default (30 s)       |                                                   |
| `plugin.http.downloadTimeout` | duration, `0s` = default (30 min)     |                                                   |
| `gitignore.enable`            | `true` / `false`                      | Adds `.vfox/` to `.gitignore` for project scopes. |
| `diagnostics.onUse`           | `true` / `false`                      | Warn about shadowed and drifted SDKs on `vfox use`. |
| `diagnostics.ignore`          | list, for example `[]`                | Finding IDs not to report.                        |
| `path.mode`                   | `legacy` / `baseline` / `aggressive`  | `baseline` is the default since 1.1.0; see [PATH order](#path-order). |
| `path.preserve`               | list of directories                   | Directories that stay in front with `baseline` and `aggressive`. |
| `aliases`                     | map, for example `map[]`              | Version aliases; edited through `vfox alias`, not by hand. |

`vfox config <key> <value>` sets a value and `vfox config --unset <key>` resets
it. Since 1.1.0 vfox writes only the key it sets, keeping comments and the
other keys untouched. Invalid values fail with vfox's message, for example
`cache.availableHookDuration: time: invalid duration "notaduration"` (fixture
`config-set-invalid-duration`). Setting a section, a list or a table as a key
fails (`aliases holds a list or a table…: edit …`); unsetting a missing key
succeeds without rewriting the file. An unknown key is **silently accepted**
(exit code 0), so only offer keys that `--list` printed. Fixtures:
`config-list`, `config-set`, `config-get`.

### Health, aliases and linked versions

```console
$ vfox doctor --json
{"ok": true, "findings": []}
$ vfox alias list
java@j17 -> java@17.0.9-tem
```

`vfox doctor [--json] [<sdk>[@<version>]]` checks vfox and the installed SDKs:
`path.shadowed`, `sdk.drift`, `sdk.link-dangling`, `sdk.payload-empty`,
`sdk.no-receipt`, `sdk.plugin-missing`, `alias.dangling` and others. The human
output says `No problems found.` or `Found … errors, … warnings and …
notices.` with a `fix:` per finding; `--json` answers
`{"ok": bool, "findings": [{id, severity, sdk, version, message, hints, fix,
ignored}]}` and exits 1 only on unignored errors. `--adopt` records
pre-receipt installs so drift can be detected later. Fixtures: `doctor`,
`doctor-json`.

`vfox alias list [<sdk>]` prints `<sdk>@<alias> -> <sdk>@<version>` per line,
or `No aliases defined. …` when there are none. `vfox alias set <sdk> <alias>
<version>` names an installed version and `vfox alias unset <sdk> <alias>`
removes it; aliases work wherever a version does (`use`, `uninstall`, `info`)
and are never written to `.vfox.toml`. Fixture: `alias-list`.

`vfox list` marks linked versions (`17-system (linked → /opt/jdk)`) and their
aliases (`[alias: j17]`); the parser keeps only the version name.
`vfox info --json` tells them apart with `origin` and `target`.

### PATH order

`path.mode` decides what stays in front of the vfox directories: `legacy` keeps
everything (the behaviour before 1.1.0), `baseline` (the default since 1.1.0)
keeps only what was added after activation, and `aggressive` keeps only
`VIRTUAL_ENV`, `CONDA_PREFIX` and `path.preserve`. When `node` or `java` is not
the one just selected, `vfox doctor` reports `path.shadowed` with the plugin
`shadowHints` explaining the other manager.

## Commands

State-changing commands run through the task queue, one at a time, with their
output streamed to the user.

| Operation        | Command line                                         | Working directory | Notes |
| ---------------- | ---------------------------------------------------- | ----------------- | ----- |
| Install          | `vfox install --yes <name>@<version>`                | `/`               | Progress on standard error. Without a version vfox fails (`install-without-version`). The version may be any name the plugin resolves, see [Names that the plugin resolves](#names-that-the-plugin-resolves). |
| Install and use  | `vfox install --yes --use <name>@<version>`          | `/`               | Like Install, then switches to it in the session scope like `vfox use`. Needs `__VFOX_SHELL` for the switch; without one the install succeeds and the switch fails. |
| Reinstall        | `vfox install --yes --reinstall <name>@<version>`    | `/`               | Replaces an installed version with a fresh copy, for drifted SDKs (`flutter upgrade` moved it in place). Refuses linked versions. |
| Uninstall        | `vfox uninstall <name>@<version>`                    | `/`               | Accepts a leading `v` (`v24.11.1`). Uninstalling the current version makes vfox switch to another installed one, removing aliases that point at it. |
| Use (global)     | `vfox use --global <name>@<version>`                 | `/`               | Needs `__VFOX_SHELL`, see below. Aliases work as versions (`java@j17`). Prints shadow and drift warnings (disable with `diagnostics.onUse false`). |
| Use (project)    | `vfox use --project <name>@<version>`                | project directory | Writes `.vfox.toml`, links `.vfox/sdks`, may edit `.gitignore`. |
| Stop using       | `vfox unuse --global <name>`                         | `/` or project    | |
| Link             | `vfox link <name>@<version> <path>`                  | `/`               | Uses the SDK at `path` (its `bin` directory) as `version`; `--force` repoints a linked version. Uninstalling removes the link only. |
| Link detect      | `vfox link --detect <name> [--yes]`                  | `/`               | Lets the `Detect` hook of the plugin find SDKs (java, gradle, flutter, golang, python) and links them; `-y` links everything without asking. |
| Alias set        | `vfox alias set <sdk> <alias> <version>`             | `/`               | Names an installed version, for example `java@j17`. |
| Alias unset      | `vfox alias unset <sdk> <alias>`                     | `/`               | Removes the alias. |
| Doctor adopt     | `vfox doctor --adopt [<sdk>[@<version>]]`            | `/`               | Records pre-receipt installs so drift can be detected later. |
| Add plugin       | `vfox add <name>`                                    | `/`               | |
| Add from source  | `vfox add --source <url-or-path> [--alias <alias>] [<name>]` | `/`       | The name defaults to the plugin's own. Pass a copy of a local `.lua` file, see [the quirks](#known-quirks). |
| Remove plugin    | `vfox remove --yes <name>`                           | `/`               | Also deletes every installed version. Without `--yes` it fails (`remove-without-yes`). |
| Update plugin    | `vfox update <name>`                                 | `/`               | Falls back to the plugin `manifestUrl` when the registry misses it. |
| Update all       | `vfox update --all`                                  | `/`               | |

Options always precede the positional arguments.

### Names that the plugin resolves

`<version>` is not only a name from the list of a search. The plugin's `PreInstall`
hook turns what it gets into a package, and the plugins of the version-fox
organisation understand more than a listed name: vfox-nodejs takes the start of a
release (`24`, `24.21`) and `latest`; vfox-flutter takes the channels `stable` and
`beta`, with an architecture suffix, and installs the release a channel points to
under the name of that release; vfox-java takes a version and a distribution
(`21-tem`, `temurin-21`, `21.0.8-tem`, `21-zulu-fx`) or an early-access build
(`28.ea-open`); vfox-gradle takes `latest` besides the listed releases;
vfox-golang takes `latest` and release candidates such as `1.27rc3`; vfox-python takes `latest`, a partial release such as `3.13` or `3.13t`, and the full name of a GraalPy release such as `graalpy-24.2.0`. The
application therefore installs whatever name the user types,
checked like any other version (not empty, not starting with `-`, no whitespace, no
`@`), and takes the installed names from `vfox list` afterwards, since a channel
is not an installed name. The [profile](architecture.md#plugin-profiles) of each
plugin lists the names worth suggesting.

### Environment of the plugins

The plugins read environment variables from the environment of the vfox process,
which is why the application passes the extra environment of its settings to every
vfox command (see [Environment](#environment)): `VFOX_JAVA_GITHUB_MIRROR` and
`VFOX_FLUTTER_GITHUB_MIRROR` (a prefix that goes in front of a GitHub address, so it
has to start with `http://` or `https://`; vfox takes anything else for a file on
disk), `VFOX_NODEJS_MIRROR`, `FLUTTER_STORAGE_BASE_URL` and `VFOX_NODEJS_LIBC`
(`musl` or `glibc`), `VFOX_GRADLE_MIRRORS`, `VFOX_GOLANG_MIRROR` (with a trailing
`/`), `VFOX_PYTHON_MIRROR` and `VFOX_PYTHON_UV_BUILD_MIRROR` (the python.org and uv-build mirrors), `VFOX_PYTHON_USE_UV_BUILD` (`0` or `1`), `VFOX_PYTHON_UV_LIBC` (`gnu` or `musl`), `VFOX_PYTHON_REGISTER_PY_LAUNCHER` (`0` turns the Windows launcher registration off) and `VFOX_JAVA_JDKS_LINKS` (exposes JDKs to Gradle and IDEA). vfox keeps the
answer of a search per list of arguments for
`cache.availableHookDuration`, whatever the environment says, so a changed mirror
shows in the list after the cache expires.

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

A running command is cancelled by terminating vfox (`SIGTERM` on Unix, kill on
Windows); the task then counts as cancelled whatever the exit code. What an
interrupted `vfox install` leaves behind depends on the version:

- the signal ends vfox. The partial directory stays, next to a
  `.v-<version>.installing` marker; `vfox list` does not show that version, and
  the next `vfox install` of it removes the leftovers first.

vfox also refuses to install a version while another vfox
process, for example in a terminal, installs it:
`cannot install <name>@<version>: installation is in progress or cannot be locked: …`.

## Known quirks

These hold for vfox 1.1.0.

- `vfox add a b` reports per-plugin failures but exits with 0. The GUI adds one
  plugin per command.
- If a plugin's pre-install hook cannot reach the network (for example through a
  broken proxy), `vfox install` may report `Install maven@ success!` with an
  empty version. Treat the task output, not only the exit code, as what the
  user sees.
- `vfox config` with an unknown key succeeds without doing anything. Since 1.1.0
  it writes only the key it sets and rejects sections, lists and tables as
  keys, bad booleans and bad durations with the messages above.
- Download failures now name the HTTP status (`unexpected HTTP status 403
  Forbidden for <url>`) and redact URL credentials (`user:xxxxx@host`,
  `?token=****`).
- `vfox current` prints the stored name verbatim (no leading `v`);
  `uninstall`, `info`, `doctor` and `use` accept a conventional leading `v`
  when it names nothing else.
- `vfox add --source <path>.lua` *moves* the file: it renames it into a new
  directory in its temporary directory (`~/.vfox/tmp`, or `~/.version-fox/tmp`
  when that legacy directory exists). The file disappears from where the user
  keeps it, and the rename fails when the file is on another filesystem, such as
  a `/tmp` on tmpfs. Local `.zip` archives and URLs are only read. The GUI
  copies a local `.lua` file into a new directory in vfox's temporary directory
  and passes the copy; vfox never cleans up directories there whose names do not
  start with a date.
- Output is split over both streams: `vfox --version` goes to standard output,
  progress, usage errors and the errors that commands return to standard error
  (see [Output conventions](#output-conventions)).

## Testing against the real vfox

Tests that run the real vfox must not touch the vfox installation of whoever
runs them. Run vfox with `HOME` set to a new, empty directory and without
`VFOX_HOME`. The offline plugin in [`shared/test-plugin`](../../../shared/test-plugin)
together with a local HTTP server covers the registry, adding plugins,
searching, installing with download progress, using, uninstalling and removing,
without network access.

## Updating this contract

When vfox changes, capture its output again with
`vfxtool capture`, update
`cases.json` and this document, and run the parser tests. See [`shared/fixtures/README.md`](../../../shared/fixtures/README.md).
