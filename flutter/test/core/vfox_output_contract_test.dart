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

// Runs the parsers against the shared vfox output fixtures.
//
// The expected results live in shared/fixtures/vfox-*/cases.json and are
// shared by every implementation of the GUI; see shared/fixtures/README.md.

import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/models.dart';
import 'package:vfox_plus/src/core/terminal_text.dart';
import 'package:vfox_plus/src/core/vfox/vfox_output.dart';

import '../support/fakes.dart';

/// Turns a parser result into the JSON shape used by `cases.json`.
typedef _Check = Object? Function(String stdout, String stderr);

final Map<String, _Check> _checks = <String, _Check>{
  'version': (String stdout, String _) => parseVfoxVersion(stdout),
  'current': (String stdout, String _) => parseCurrent(stdout),
  'list': (String stdout, String _) => parseInstalledTree(stdout),
  'search': (String stdout, String _) => _indexed(parseSearch(stdout).map(_versionJson).toList()),
  'available': (String stdout, String _) => _indexed(parseAvailablePlugins(stdout).map(_pluginJson).toList()),
  'pluginInfo': (String stdout, String _) => _pluginInfoJson(parsePluginInfo(stdout)),
  'installPath': (String stdout, String _) => parseInstallPath(stdout),
  'config': (String stdout, String _) => <Object?>[
    for (final ConfigEntry entry in parseConfigList(stdout)) <String, Object?>{'key': entry.key, 'value': entry.value},
  ],
  'finalProgress': (String _, String stderr) {
    double? last;
    for (final String line in splitTerminalLines(stripAnsi(stderr))) {
      last = parseDownloadProgress(line) ?? last;
    }
    return last;
  },
  'error': summarizeError,
};

Map<String, Object?> _indexed(List<Map<String, Object?>> items) => <String, Object?>{
  'count': items.length,
  'items': <String, Object?>{for (var i = 0; i < items.length; i++) '$i': items[i]},
};

Map<String, Object?> _versionJson(AvailableVersion version) => <String, Object?>{
  'version': version.version,
  'note': version.note,
  'additions': <Object?>[
    for (final VersionAddition addition in version.additions)
      <String, Object?>{'name': addition.name, 'version': addition.version},
  ],
  'installed': version.installed,
};

Map<String, Object?> _pluginJson(AvailablePlugin plugin) => <String, Object?>{
  'name': plugin.name,
  'official': plugin.official,
  'homepage': plugin.homepage,
};

Map<String, Object?>? _pluginInfoJson(PluginInfo? info) => info == null
    ? null
    : <String, Object?>{
        'name': info.name,
        'version': info.version,
        'homepage': info.homepage,
        'installPath': info.installPath,
        'description': info.description,
      };

/// For `search` and `available`, only the listed items are compared.
Object? _select(String check, Object? actual, Object? expected) {
  if ((check == 'search' || check == 'available') && actual is Map && expected is Map) {
    final Map<Object?, Object?> actualItems = actual['items']! as Map<Object?, Object?>;
    final Map<Object?, Object?> expectedItems = expected['items']! as Map<Object?, Object?>;
    return <String, Object?>{
      'count': actual['count'],
      'items': <Object?, Object?>{for (final Object? index in expectedItems.keys) index: actualItems[index]},
    };
  }
  return actual;
}

void main() {
  final fixtures = Directory('../shared/fixtures');
  final List<Directory> versions =
      fixtures
          .listSync()
          .whereType<Directory>()
          .where((Directory dir) => File('${dir.path}/cases.json').existsSync())
          .toList()
        ..sort((Directory a, Directory b) => a.path.compareTo(b.path));

  test('fixtures are present', () {
    expect(versions, isNotEmpty, reason: 'Run the tests from the flutter/ directory.');
  });

  for (final directory in versions) {
    final manifest = jsonDecode(File('${directory.path}/cases.json').readAsStringSync()) as Map<String, Object?>;
    final cases = (manifest['cases']! as List<Object?>).cast<Map<String, Object?>>();

    group('vfox ${manifest['vfox']}', () {
      for (final testCase in cases) {
        final name = testCase['name']! as String;
        final String stdout = readCapture('${directory.path}/$name.stdout');
        final String stderr = readCapture('${directory.path}/$name.stderr');
        final expectations = testCase['expect']! as Map<String, Object?>;

        for (final MapEntry<String, Object?> expectation in expectations.entries) {
          test('$name: ${expectation.key}', () {
            final _Check? check = _checks[expectation.key];
            expect(check, isNotNull, reason: 'Unknown parser "${expectation.key}" in cases.json');
            final Object? actual = _select(expectation.key, check!(stdout, stderr), expectation.value);
            // Compare through JSON so that maps and lists compare by value.
            expect(jsonDecode(jsonEncode(actual)), expectation.value);
          });
        }
      }
    });
  }
}
