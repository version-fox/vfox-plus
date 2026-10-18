# Copyright 2026 Han Li and contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Parsers for the human-oriented output of the vfox CLI.

vfox has no machine-readable output mode for most commands, so every format
handled here is documented in `docs/vfox-cli-contract.md` and covered by the
captured outputs in `shared/fixtures/`. All parsers are pure functions: they
receive the decoded standard output of a command and never raise on
unexpected lines, which are skipped instead.
"""

import re
from collections.abc import Mapping, Sequence

from vfox_plus.core.models import (
    AvailablePlugin,
    AvailableVersion,
    ConfigEntry,
    InstalledSdk,
    PluginInfo,
    VersionAddition,
)
from vfox_plus.core.terminal_text import split_terminal_lines, strip_ansi

FIELD_SEPARATOR = "\x1f"
"""The ASCII unit separator used between the fields of `vfox info --format`."""

PLUGIN_INFO_TEMPLATE = (
    '{{.Name}}{{"\\x1f"}}{{.Version}}{{"\\x1f"}}{{.Homepage}}{{"\\x1f"}}{{.InstallPath}}{{"\\x1f"}}{{.Description}}'
)
"""The Go template passed to `vfox info <name> --format`.

The fields are joined with :data:`FIELD_SEPARATOR`, which cannot appear in any
of them, so descriptions may contain new lines and any punctuation.
"""

_VERSION = re.compile(r"(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)")
_TREE_ITEM = re.compile(r"^([│ ]*)[├└]─[─┬](.+)$")
_AVAILABLE_PLUGIN = re.compile(r"^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$")
_DOWNLOAD_PROGRESS = re.compile(r"Downloading\.\.\.\s+(\d{1,3})%")
_FAILED_TO_INSTALL = re.compile(r"^failed to install\b")


def parse_vfox_version(output: str) -> str | None:
    """Parses `vfox --version`, for example `vfox version 1.0.12`.

    Returns `None` if no version number is found.
    """
    match = _VERSION.search(strip_ansi(output))
    return match.group(1) if match else None


def parse_current(output: str) -> dict[str, str | None]:
    """Parses `vfox current` without arguments.

    Each line has the form `<name> -> v<version>`, or `<name> -> N/A` when no
    version is in effect. vfox prefixes the version with a `v`, which is
    removed again. The result preserves the order of the output.
    """
    result: dict[str, str | None] = {}
    for line in _lines(output):
        arrow = line.find(" -> ")
        if arrow <= 0:
            continue
        name = line[:arrow].strip()
        value = line[arrow + 4 :].strip()
        if name == "" or " " in name:
            continue
        result[name] = None if value in ("", "N/A") else _strip_v_prefix(value)
    return result


def parse_installed_tree(output: str) -> dict[str, list[str]]:
    """Parses the tree printed by `vfox list` without arguments.

    ```text
    All installed sdk versions
    ├─┬java
    │ ├──25.4.4.1-graalce
    │ └──25.3.4.1-graalce
    └──maven
    ```

    A plugin without installed versions is drawn as a leaf (`└──maven`). The
    result preserves the order of the output; versions keep their order too.
    """
    result: dict[str, list[str]] = {}
    current_sdk: str | None = None
    for line in _lines(output):
        match = _TREE_ITEM.match(line)
        if match is None:
            continue
        level = len(match.group(1)) // 2
        text = match.group(2).strip()
        if text == "":
            continue
        if level == 0:
            current_sdk = text
            result.setdefault(text, [])
        elif current_sdk is not None:
            result[current_sdk].append(text)
    return result


def merge_installed_sdks(tree: Mapping[str, Sequence[str]], current: Mapping[str, str | None]) -> list[InstalledSdk]:
    """Combines the outputs of `vfox list` and `vfox current` into SDKs.

    Plugins that only appear in one of the outputs are still included, so that
    the result degrades gracefully if one command fails.
    """
    names = dict.fromkeys([*tree, *current])
    return [InstalledSdk(name, tuple(tree.get(name, ())), current.get(name)) for name in names]


def parse_search(output: str) -> list[AvailableVersion]:
    """Parses `vfox search <name>` in a non-interactive terminal.

    ```text
    Available versions:
     - 24.22.0 (LTS) [npm 11.19.1]
     - 3.9.16 (installed)
    ```

    Every entry is
    `<version>[ (<note>)][ [<tool> <version>,...]][ (installed)]`.
    """
    result: list[AvailableVersion] = []
    for raw_line in _lines(output):
        line = raw_line.rstrip()
        if not line.startswith(" - "):
            continue
        version = _parse_search_entry(line[3:].strip())
        if version is not None:
            result.append(version)
    return result


def _parse_search_entry(entry: str) -> AvailableVersion | None:
    if entry == "":
        return None
    first_space = entry.find(" ")
    if first_space < 0:
        return AvailableVersion(entry)
    version = entry[:first_space]
    rest = entry[first_space + 1 :].strip()

    installed = False
    installed_suffix = "(installed)"
    if rest.endswith(installed_suffix):
        installed = True
        rest = rest[: -len(installed_suffix)].strip()

    additions: tuple[VersionAddition, ...] = ()
    if rest.endswith("]"):
        opening = rest.rfind("[")
        if opening >= 0:
            additions = _parse_additions(rest[opening + 1 : -1])
            rest = rest[:opening].strip()

    note: str | None = None
    if rest.startswith("(") and rest.endswith(")"):
        inner = rest[1:-1].strip()
        note = inner or None
    elif rest != "":
        note = rest
    return AvailableVersion(version, note, additions, installed)


def _parse_additions(text: str) -> tuple[VersionAddition, ...]:
    result: list[VersionAddition] = []
    for part in text.split(","):
        item = part.strip()
        if item == "":
            continue
        name, _, version = item.partition(" ")
        result.append(VersionAddition(name, version.strip()))
    return tuple(result)


def parse_available_plugins(output: str) -> list[AvailablePlugin]:
    """Parses `vfox available`.

    ```text
    AVAILABLE PLUGINS

      nodejs              ✓  https://github.com/version-fox/vfox-nodejs
      bun                 ✗  https://github.com/arfaWong/vfox-bun

      Use 'vfox add <plugin>' to install
    ```
    """
    result: list[AvailablePlugin] = []
    for line in _lines(output):
        match = _AVAILABLE_PLUGIN.match(line)
        if match is not None:
            result.append(AvailablePlugin(name=match.group(1), homepage=match.group(3), official=match.group(2) == "✓"))
    return result


def parse_plugin_info(output: str) -> PluginInfo | None:
    """Parses the output of `vfox info <name> --format` with
    :data:`PLUGIN_INFO_TEMPLATE`.

    Returns `None` if the output does not have the expected number of fields.
    """
    fields = strip_ansi(output).split(FIELD_SEPARATOR)
    if len(fields) < 5:
        return None
    return PluginInfo(
        name=fields[0].strip(),
        version=fields[1].strip(),
        homepage=fields[2].strip(),
        install_path=fields[3].strip(),
        # The description is the last field and may itself contain separators.
        description=FIELD_SEPARATOR.join(fields[4:]).strip(),
    )


def parse_install_path(output: str) -> str | None:
    """Parses `vfox info <name>@<version>`, which prints the directory of an
    installed version, or `notfound`. Returns `None` for `notfound`.
    """
    path = strip_ansi(output).strip()
    return None if path in ("", "notfound") else path


def parse_config_list(output: str) -> list[ConfigEntry]:
    """Parses `vfox config --list`, which prints one `<key> = <value>` per
    line.
    """
    result: list[ConfigEntry] = []
    for line in _lines(output):
        equals = line.find(" = ")
        if equals <= 0:
            # `key =` with an empty value loses its trailing space when trimmed.
            trimmed = line.rstrip()
            if trimmed.endswith(" ="):
                result.append(ConfigEntry(trimmed[:-2].strip(), ""))
            continue
        key = line[:equals].strip()
        if key == "" or " " in key:
            continue
        result.append(ConfigEntry(key, line[equals + 3 :].strip()))
    return result


def parse_download_progress(line: str) -> float | None:
    """Extracts the download progress from a line of `vfox install` output.

    The progress bar is written to standard error and redrawn with carriage
    returns: `Downloading...  42% [=====>    ] (1.2 MB/s) [3s:4s]`. Returns a
    fraction between 0 and 1, or `None` if `line` is not a progress line.
    """
    match = _DOWNLOAD_PROGRESS.search(line)
    if match is None:
        return None
    return min(max(int(match.group(1)), 0), 100) / 100


def summarize_error(stdout: str, stderr: str) -> str:
    """Picks a short, human readable error message from a failed command.

    Usage errors (`cli.Exit`) go to standard error. The error a command
    returns goes to standard output up to vfox 1.0.12 (`fmt.Println(err)`)
    and to standard error from 1.0.13, while what a command prints before
    failing, such as the reason of a failed installation, stays on standard
    output. So the lines of standard output are read first and those of
    standard error last. Progress bar redraws and blank lines are ignored, and
    the last meaningful line wins, unless it is the generic
    `failed to install <name>` trailer, in which case the more specific line
    printed before it is used.
    """
    lines = [*_meaningful_lines(stdout), *_meaningful_lines(stderr)]
    if not lines:
        return ""
    if len(lines) > 1 and _FAILED_TO_INSTALL.match(lines[-1]):
        return lines[-2]
    return lines[-1]


def _meaningful_lines(text: str) -> list[str]:
    return [line for line in (raw.strip() for raw in _lines(text)) if line and parse_download_progress(line) is None]


def _strip_v_prefix(version: str) -> str:
    return version[1:] if version.startswith("v") and len(version) > 1 else version


def _lines(output: str) -> list[str]:
    return split_terminal_lines(strip_ansi(output))
