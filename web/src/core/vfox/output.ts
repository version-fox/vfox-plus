// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/**
 * Parsers for the human-oriented output of the vfox CLI.
 *
 * vfox has no machine-readable output mode for most commands, so every format
 * handled here is documented in `docs/vfox-cli-contract.md` and covered by the
 * captured outputs in `shared/fixtures/`. All parsers are pure functions: they
 * receive the decoded standard output of a command and never throw on
 * unexpected lines, which are skipped instead.
 */

import type {
  AvailablePlugin,
  AvailableVersion,
  ConfigEntry,
  InstalledSdk,
  PluginInfo,
  VersionAddition,
} from '../models';
import { splitTerminalLines, stripAnsi } from '../terminal-text';

/** The ASCII unit separator used between the fields of `vfox info --format`. */
export const FIELD_SEPARATOR = '\u001f';

/**
 * The Go template passed to `vfox info <name> --format`.
 *
 * The fields are joined with {@link FIELD_SEPARATOR}, which cannot appear in
 * any of them, so descriptions may contain new lines and any punctuation.
 */
export const PLUGIN_INFO_TEMPLATE =
  '{{.Name}}{{"\\x1f"}}{{.Version}}{{"\\x1f"}}{{.Homepage}}' + '{{"\\x1f"}}{{.InstallPath}}{{"\\x1f"}}{{.Description}}';

/**
 * Parses `vfox --version`, for example `vfox version 1.0.12`.
 *
 * Returns `null` if no version number is found.
 */
export function parseVfoxVersion(output: string): string | null {
  const match = /(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)/.exec(stripAnsi(output));
  return match?.[1] ?? null;
}

/**
 * Parses `vfox current` without arguments.
 *
 * Each line has the form `<name> -> v<version>`, or `<name> -> N/A` when no
 * version is in effect. vfox prefixes the version with a `v`, which is removed
 * again. The map preserves the order of the output.
 */
export function parseCurrent(output: string): Map<string, string | null> {
  const result = new Map<string, string | null>();
  for (const line of lines(output)) {
    const arrow = line.indexOf(' -> ');
    if (arrow <= 0) {
      continue;
    }
    const name = line.slice(0, arrow).trim();
    const value = line.slice(arrow + 4).trim();
    if (name === '' || name.includes(' ')) {
      continue;
    }
    result.set(name, value === '' || value === 'N/A' ? null : stripVPrefix(value));
  }
  return result;
}

/**
 * Parses the tree printed by `vfox list` without arguments.
 *
 * ```text
 * All installed sdk versions
 * ├─┬java
 * │ ├──25.4.4.1-graalce
 * │ └──25.3.4.1-graalce
 * └──maven
 * ```
 *
 * A plugin without installed versions is drawn as a leaf (`└──maven`). The map
 * preserves the order of the output; versions keep their order too.
 */
export function parseInstalledTree(output: string): Map<string, string[]> {
  const result = new Map<string, string[]>();
  const itemPattern = /^([│ ]*)[├└]─[─┬](.+)$/u;
  let currentSdk: string | null = null;
  for (const line of lines(output)) {
    const match = itemPattern.exec(line);
    if (match === null) {
      continue;
    }
    // The tree characters are single UTF-16 code units.
    const level = Math.floor((match[1] ?? '').length / 2);
    const text = (match[2] ?? '').trim();
    if (text === '') {
      continue;
    }
    if (level === 0) {
      currentSdk = text;
      if (!result.has(text)) {
        result.set(text, []);
      }
    } else if (currentSdk !== null) {
      result.get(currentSdk)?.push(text);
    }
  }
  return result;
}

/**
 * Combines the outputs of `vfox list` and `vfox current` into SDKs.
 *
 * Plugins that only appear in one of the outputs are still included, so that
 * the result degrades gracefully if one command fails.
 */
export function mergeInstalledSdks(
  tree: ReadonlyMap<string, readonly string[]>,
  current: ReadonlyMap<string, string | null>,
): InstalledSdk[] {
  const names = new Set<string>([...tree.keys(), ...current.keys()]);
  return [...names].map((name) => ({
    name,
    versions: [...(tree.get(name) ?? [])],
    current: current.get(name) ?? null,
  }));
}

/**
 * Parses `vfox search <name>` in a non-interactive terminal.
 *
 * ```text
 * Available versions:
 *  - 24.22.0 (LTS) [npm 11.19.1]
 *  - 3.9.16 (installed)
 * ```
 *
 * Every entry is `<version>[ (<note>)][ [<tool> <version>,...]][ (installed)]`.
 */
export function parseSearch(output: string): AvailableVersion[] {
  const result: AvailableVersion[] = [];
  for (const rawLine of lines(output)) {
    const line = rawLine.trimEnd();
    if (!line.startsWith(' - ')) {
      continue;
    }
    const version = parseSearchEntry(line.slice(3).trim());
    if (version !== null) {
      result.push(version);
    }
  }
  return result;
}

function parseSearchEntry(entry: string): AvailableVersion | null {
  if (entry === '') {
    return null;
  }
  const firstSpace = entry.indexOf(' ');
  if (firstSpace < 0) {
    return { version: entry, note: null, additions: [], installed: false };
  }
  const version = entry.slice(0, firstSpace);
  let rest = entry.slice(firstSpace + 1).trim();

  let installed = false;
  const installedSuffix = '(installed)';
  if (rest.endsWith(installedSuffix)) {
    installed = true;
    rest = rest.slice(0, rest.length - installedSuffix.length).trim();
  }

  let additions: VersionAddition[] = [];
  if (rest.endsWith(']')) {
    const open = rest.lastIndexOf('[');
    if (open >= 0) {
      additions = parseAdditions(rest.slice(open + 1, rest.length - 1));
      rest = rest.slice(0, open).trim();
    }
  }

  let note: string | null = null;
  if (rest.startsWith('(') && rest.endsWith(')')) {
    const inner = rest.slice(1, rest.length - 1).trim();
    note = inner === '' ? null : inner;
  } else if (rest !== '') {
    note = rest;
  }
  return { version, note, additions, installed };
}

function parseAdditions(text: string): VersionAddition[] {
  const result: VersionAddition[] = [];
  for (const part of text.split(',')) {
    const item = part.trim();
    if (item === '') {
      continue;
    }
    const space = item.indexOf(' ');
    result.push(
      space < 0 ? { name: item, version: '' } : { name: item.slice(0, space), version: item.slice(space + 1).trim() },
    );
  }
  return result;
}

/**
 * Parses `vfox available`.
 *
 * ```text
 * AVAILABLE PLUGINS
 *
 *   nodejs              ✓  https://github.com/version-fox/vfox-nodejs
 *   bun                 ✗  https://github.com/arfaWong/vfox-bun
 *
 *   Use 'vfox add <plugin>' to install
 * ```
 */
export function parseAvailablePlugins(output: string): AvailablePlugin[] {
  const pattern = /^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$/u;
  const result: AvailablePlugin[] = [];
  for (const line of lines(output)) {
    const match = pattern.exec(line);
    if (match === null) {
      continue;
    }
    result.push({ name: match[1] ?? '', official: match[2] === '✓', homepage: match[3] ?? '' });
  }
  return result;
}

/**
 * Parses the output of `vfox info <name> --format` with
 * {@link PLUGIN_INFO_TEMPLATE}.
 *
 * Returns `null` if the output does not have the expected number of fields.
 */
export function parsePluginInfo(output: string): PluginInfo | null {
  const fields = stripAnsi(output).split(FIELD_SEPARATOR);
  if (fields.length < 5) {
    return null;
  }
  return {
    name: (fields[0] ?? '').trim(),
    version: (fields[1] ?? '').trim(),
    homepage: (fields[2] ?? '').trim(),
    installPath: (fields[3] ?? '').trim(),
    // The description is the last field and may itself contain separators.
    description: fields.slice(4).join(FIELD_SEPARATOR).trim(),
  };
}

/**
 * Parses `vfox info <name>@<version>`, which prints the directory of an
 * installed version, or `notfound`. Returns `null` for `notfound`.
 */
export function parseInstallPath(output: string): string | null {
  const path = stripAnsi(output).trim();
  return path === '' || path === 'notfound' ? null : path;
}

/** Parses `vfox config --list`, which prints one `<key> = <value>` per line. */
export function parseConfigList(output: string): ConfigEntry[] {
  const result: ConfigEntry[] = [];
  for (const line of lines(output)) {
    const equals = line.indexOf(' = ');
    if (equals <= 0) {
      // `key =` with an empty value loses its trailing space when trimmed.
      const trimmed = line.trimEnd();
      if (trimmed.endsWith(' =')) {
        result.push({ key: trimmed.slice(0, trimmed.length - 2).trim(), value: '' });
      }
      continue;
    }
    const key = line.slice(0, equals).trim();
    if (key === '' || key.includes(' ')) {
      continue;
    }
    result.push({ key, value: line.slice(equals + 3).trim() });
  }
  return result;
}

/**
 * Extracts the download progress from a line of `vfox install` output.
 *
 * The progress bar is written to standard error and redrawn with carriage
 * returns: `Downloading...  42% [=====>    ] (1.2 MB/s) [3s:4s]`. Returns a
 * fraction between 0 and 1, or `null` if `line` is not a progress line.
 */
export function parseDownloadProgress(line: string): number | null {
  const match = /Downloading\.\.\.\s+(\d{1,3})%/.exec(line);
  if (match === null) {
    return null;
  }
  const percent = Number.parseInt(match[1] ?? '0', 10);
  return Math.min(Math.max(percent, 0), 100) / 100;
}

/**
 * Picks a short, human readable error message from a failed command.
 *
 * Usage errors (`cli.Exit`) go to standard error. The error a command returns
 * goes to standard output up to vfox 1.0.12 (`fmt.Println(err)`) and to
 * standard error from 1.0.13, while what a command prints before failing, such
 * as the reason of a failed installation, stays on standard output. So the
 * lines of standard output are read first and those of standard error last.
 * Progress bar redraws and blank lines are ignored, and the last meaningful
 * line wins, unless it is the generic `failed to install <name>` trailer, in
 * which case the more specific line printed before it is used.
 */
export function summarizeError(stdout: string, stderr: string): string {
  const all = [...meaningfulLines(stdout), ...meaningfulLines(stderr)];
  const last = all.at(-1);
  if (last === undefined) {
    return '';
  }
  if (all.length > 1 && /^failed to install\b/.test(last)) {
    return all[all.length - 2] ?? last;
  }
  return last;
}

function meaningfulLines(text: string): string[] {
  return lines(text)
    .map((line) => line.trim())
    .filter((line) => line !== '' && parseDownloadProgress(line) === null);
}

function stripVPrefix(version: string): string {
  return version.startsWith('v') && version.length > 1 ? version.slice(1) : version;
}

function lines(output: string): string[] {
  return splitTerminalLines(stripAnsi(output));
}
