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

/// Parsers for the human-oriented output of the vfox CLI.
///
/// vfox has no machine-readable output mode for most commands, so every
/// format handled here is documented in `docs/vfox-cli-contract.md` and
/// covered by the captured outputs in `shared/fixtures/`. All parsers are
/// pure functions: they receive the decoded standard output of a command
/// and never throw on unexpected lines, which are skipped instead.
library;

import '../models.dart';
import '../terminal_text.dart';

/// The ASCII unit separator used between the fields of `vfox info --format`.
const String fieldSeparator = '\u001F';

/// The Go template passed to `vfox info <name> --format`.
///
/// The fields are joined with [fieldSeparator], which cannot appear in any of
/// them, so descriptions may contain new lines and any punctuation.
const String pluginInfoTemplate =
    '{{.Name}}{{"\\x1f"}}{{.Version}}{{"\\x1f"}}{{.Homepage}}'
    '{{"\\x1f"}}{{.InstallPath}}{{"\\x1f"}}{{.Description}}';

/// Parses `vfox --version`, for example `vfox version 1.0.12`.
///
/// Returns `null` if no version number is found.
String? parseVfoxVersion(String output) {
  final RegExpMatch? match = RegExp(r'(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)').firstMatch(stripAnsi(output));
  return match?.group(1);
}

/// Parses `vfox current` without arguments.
///
/// Each line has the form `<name> -> v<version>`, or `<name> -> N/A` when no
/// version is in effect. vfox prefixes the version with a `v`, which is
/// removed again. The map preserves the order of the output.
Map<String, String?> parseCurrent(String output) {
  final result = <String, String?>{};
  for (final String rawLine in _lines(output)) {
    final int arrow = rawLine.indexOf(' -> ');
    if (arrow <= 0) {
      continue;
    }
    final String name = rawLine.substring(0, arrow).trim();
    final String value = rawLine.substring(arrow + 4).trim();
    if (name.isEmpty || name.contains(' ')) {
      continue;
    }
    result[name] = value.isEmpty || value == 'N/A' ? null : _stripVPrefix(value);
  }
  return result;
}

/// Parses the tree printed by `vfox list` without arguments.
///
/// ```text
/// All installed sdk versions
/// ├─┬java
/// │ ├──25.4.4.1-graalce
/// │ └──25.3.4.1-graalce
/// └──maven
/// ```
///
/// A plugin without installed versions is drawn as a leaf (`└──maven`).
/// The map preserves the order of the output; versions keep their order too.
Map<String, List<String>> parseInstalledTree(String output) {
  final result = <String, List<String>>{};
  final itemPattern = RegExp(r'^([│ ]*)[├└]─[─┬](.+)$');
  String? currentSdk;
  for (final String rawLine in _lines(output)) {
    final RegExpMatch? match = itemPattern.firstMatch(rawLine);
    if (match == null) {
      continue;
    }
    final int level = match.group(1)!.length ~/ 2;
    final String text = match.group(2)!.trim();
    if (text.isEmpty) {
      continue;
    }
    if (level == 0) {
      currentSdk = text;
      result.putIfAbsent(text, () => <String>[]);
    } else if (currentSdk != null) {
      result[currentSdk]!.add(text);
    }
  }
  return result;
}

/// Combines the outputs of `vfox list` and `vfox current` into SDKs.
///
/// Plugins that only appear in one of the outputs are still included, so
/// that the result degrades gracefully if one command fails.
List<InstalledSdk> mergeInstalledSdks(Map<String, List<String>> tree, Map<String, String?> current) {
  final names = <String>{...tree.keys, ...current.keys};
  return <InstalledSdk>[
    for (final String name in names)
      InstalledSdk(name: name, versions: tree[name] ?? const <String>[], current: current[name]),
  ];
}

/// Parses `vfox search <name>` in a non-interactive terminal.
///
/// ```text
/// Available versions:
///  - 24.22.0 (LTS) [npm 11.19.1]
///  - 3.9.16 (installed)
/// ```
///
/// Every entry is `<version>[ (<note>)][ [<tool> <version>,...]][ (installed)]`.
List<AvailableVersion> parseSearch(String output) {
  final result = <AvailableVersion>[];
  for (final String rawLine in _lines(output)) {
    final String line = rawLine.trimRight();
    if (!line.startsWith(' - ')) {
      continue;
    }
    final AvailableVersion? version = _parseSearchEntry(line.substring(3).trim());
    if (version != null) {
      result.add(version);
    }
  }
  return result;
}

AvailableVersion? _parseSearchEntry(String entry) {
  if (entry.isEmpty) {
    return null;
  }
  final int firstSpace = entry.indexOf(' ');
  if (firstSpace < 0) {
    return AvailableVersion(version: entry);
  }
  final String version = entry.substring(0, firstSpace);
  var rest = entry.substring(firstSpace + 1).trim();

  var installed = false;
  const installedSuffix = '(installed)';
  if (rest.endsWith(installedSuffix)) {
    installed = true;
    rest = rest.substring(0, rest.length - installedSuffix.length).trim();
  }

  var additions = const <VersionAddition>[];
  if (rest.endsWith(']')) {
    final int open = rest.lastIndexOf('[');
    if (open >= 0) {
      additions = _parseAdditions(rest.substring(open + 1, rest.length - 1));
      rest = rest.substring(0, open).trim();
    }
  }

  String? note;
  if (rest.startsWith('(') && rest.endsWith(')')) {
    final String inner = rest.substring(1, rest.length - 1).trim();
    note = inner.isEmpty ? null : inner;
  } else if (rest.isNotEmpty) {
    note = rest;
  }
  return AvailableVersion(version: version, note: note, additions: additions, installed: installed);
}

List<VersionAddition> _parseAdditions(String text) {
  final result = <VersionAddition>[];
  for (final String part in text.split(',')) {
    final String item = part.trim();
    if (item.isEmpty) {
      continue;
    }
    final int space = item.indexOf(' ');
    result.add(
      space < 0
          ? VersionAddition(name: item, version: '')
          : VersionAddition(name: item.substring(0, space), version: item.substring(space + 1).trim()),
    );
  }
  return result;
}

/// Parses `vfox available`.
///
/// ```text
/// AVAILABLE PLUGINS
///
///   nodejs              ✓  https://github.com/version-fox/vfox-nodejs
///   bun                 ✗  https://github.com/arfaWong/vfox-bun
///
///   Use 'vfox add <plugin>' to install
/// ```
List<AvailablePlugin> parseAvailablePlugins(String output) {
  final pattern = RegExp(r'^\s+(\S+)\s+([✓✗])\s+(\S+)\s*$');
  final result = <AvailablePlugin>[];
  for (final String rawLine in _lines(output)) {
    final RegExpMatch? match = pattern.firstMatch(rawLine);
    if (match == null) {
      continue;
    }
    result.add(AvailablePlugin(name: match.group(1)!, official: match.group(2) == '✓', homepage: match.group(3)!));
  }
  return result;
}

/// Parses the output of `vfox info <name> --format` with [pluginInfoTemplate].
///
/// Returns `null` if the output does not have the expected number of fields.
PluginInfo? parsePluginInfo(String output) {
  final List<String> fields = stripAnsi(output).split(fieldSeparator);
  if (fields.length < 5) {
    return null;
  }
  return PluginInfo(
    name: fields[0].trim(),
    version: fields[1].trim(),
    homepage: fields[2].trim(),
    installPath: fields[3].trim(),
    // The description is the last field and may itself contain separators.
    description: fields.sublist(4).join(fieldSeparator).trim(),
  );
}

/// Parses `vfox info <name>@<version>`, which prints the directory of an
/// installed version, or `notfound`. Returns `null` for `notfound`.
String? parseInstallPath(String output) {
  final String path = stripAnsi(output).trim();
  return path.isEmpty || path == 'notfound' ? null : path;
}

/// Parses `vfox config --list`, which prints one `<key> = <value>` per line.
List<ConfigEntry> parseConfigList(String output) {
  final result = <ConfigEntry>[];
  for (final String rawLine in _lines(output)) {
    final int equals = rawLine.indexOf(' = ');
    if (equals <= 0) {
      // `key =` with an empty value loses its trailing space when trimmed.
      if (rawLine.trimRight().endsWith(' =')) {
        final String key = rawLine.trimRight();
        result.add(ConfigEntry(key: key.substring(0, key.length - 2).trim(), value: ''));
      }
      continue;
    }
    final String key = rawLine.substring(0, equals).trim();
    if (key.isEmpty || key.contains(' ')) {
      continue;
    }
    result.add(ConfigEntry(key: key, value: rawLine.substring(equals + 3).trim()));
  }
  return result;
}

/// Extracts the download progress from a line of `vfox install` output.
///
/// The progress bar is written to standard error and redrawn with carriage
/// returns: `Downloading...  42% [=====>    ] (1.2 MB/s) [3s:4s]`.
/// Returns a fraction between 0 and 1, or `null` if [line] is not a progress
/// line.
double? parseDownloadProgress(String line) {
  final RegExpMatch? match = RegExp(r'Downloading\.\.\.\s+(\d{1,3})%').firstMatch(line);
  if (match == null) {
    return null;
  }
  final int percent = int.parse(match.group(1)!);
  return percent.clamp(0, 100) / 100;
}

/// Picks a short, human readable error message from a failed command.
///
/// Usage errors (`cli.Exit`) go to standard error. The error a command returns
/// goes to standard output up to vfox 1.0.12 (`fmt.Println(err)`) and to
/// standard error from 1.0.13, while what a command prints before failing,
/// such as the reason of a failed installation, stays on standard output. So
/// the lines of standard output are read first and those of standard error
/// last. Progress bar redraws and blank lines are ignored, and the last
/// meaningful line wins, unless it is the generic `failed to install <name>`
/// trailer, in which case the more specific line printed before it is used.
String summarizeError(String stdout, String stderr) {
  final List<String> lines = <String>[..._meaningfulLines(stdout), ..._meaningfulLines(stderr)];
  if (lines.isEmpty) {
    return '';
  }
  if (lines.length > 1 && _genericInstallFailure.hasMatch(lines.last)) {
    return lines[lines.length - 2];
  }
  return lines.last;
}

final RegExp _genericInstallFailure = RegExp(r'^failed to install\b');

List<String> _meaningfulLines(String text) =>
    _lines(text)
        .map((String line) => line.trim())
        .where((String line) => line.isNotEmpty && parseDownloadProgress(line) == null)
        .toList();

String _stripVPrefix(String version) => version.startsWith('v') && version.length > 1 ? version.substring(1) : version;

Iterable<String> _lines(String output) => splitTerminalLines(stripAnsi(output));
