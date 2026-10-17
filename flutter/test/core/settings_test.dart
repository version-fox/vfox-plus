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

import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/settings/app_settings.dart';
import 'package:vfox_plus/src/core/settings/settings_store.dart';

void main() {
  group('AppSettings', () {
    test('round-trips through JSON', () {
      final settings = AppSettings(
        theme: ThemePreference.dark,
        useSystemColor: false,
        seedColor: 0xFF123456,
        language: LanguagePreference.simplifiedChinese,
        vfoxExecutable: '/opt/vfox',
        extraEnvironment: const <String, String>{'A': '1'},
        railExpanded: true,
      );
      expect(AppSettings.fromJson(jsonDecode(jsonEncode(settings.toJson())) as Map<String, Object?>), settings);
    });

    test('ignores unknown and malformed values', () {
      final settings = AppSettings.fromJson(<String, Object?>{
        'theme': 'purple',
        'useSystemColor': 'yes',
        'seedColor': '#fff',
        'language': 42,
        'vfoxExecutable': '',
        'extraEnvironment': <String, Object?>{'A': '1', 'B': 2},
        'future': true,
      });
      expect(settings.theme, ThemePreference.system);
      expect(settings.useSystemColor, isTrue);
      expect(settings.seedColor, AppSettings.defaultSeedColor);
      expect(settings.language, LanguagePreference.system);
      expect(settings.vfoxExecutable, isNull);
      expect(settings.extraEnvironment, <String, String>{'A': '1'});
    });

    test('copyWith can clear the executable', () {
      final settings = AppSettings(vfoxExecutable: '/opt/vfox');
      expect(settings.copyWith(clearVfoxExecutable: true).vfoxExecutable, isNull);
      expect(settings.copyWith(theme: ThemePreference.light).vfoxExecutable, '/opt/vfox');
    });
  });

  group('environment lines', () {
    test('parse KEY=VALUE, skipping comments and blank lines', () {
      final result = parseEnvironmentLines('# mirrors\nA=1\n\n B = x=y \nnot valid\n1X=2\n');
      expect(result.variables, <String, String>{'A': '1', 'B': 'x=y'});
      expect(result.invalidLines, <int>[5, 6]);
    });

    test('format sorted by key', () {
      expect(formatEnvironmentLines(const <String, String>{'B': '2', 'A': '1'}), 'A=1\nB=2');
    });
  });

  group('JsonFileSettingsStore', () {
    late Directory directory;

    setUp(() => directory = Directory.systemTemp.createTempSync('vfox-settings'));
    tearDown(() => directory.deleteSync(recursive: true));

    test('uses XDG_CONFIG_HOME and falls back to ~/.config', () {
      expect(
        JsonFileSettingsStore.standard(environment: const <String, String>{'XDG_CONFIG_HOME': '/x', 'HOME': '/h'})
            .file
            .path,
        '/x/vfox-plus/settings.json',
      );
      expect(
        JsonFileSettingsStore.standard(environment: const <String, String>{'XDG_CONFIG_HOME': 'relative', 'HOME': '/h'})
            .file
            .path,
        '/h/.config/vfox-plus/settings.json',
      );
    });

    test('returns defaults without a file, and what was saved afterwards', () async {
      final store = JsonFileSettingsStore(File('${directory.path}/nested/settings.json'));
      expect(await store.load(), AppSettings());
      final settings = AppSettings(theme: ThemePreference.light);
      await store.save(settings);
      expect(await store.load(), settings);
      expect(File('${directory.path}/nested/settings.json.tmp').existsSync(), isFalse);
    });

    test('returns defaults for a corrupted file', () async {
      final file = File('${directory.path}/settings.json')..writeAsStringSync('{not json');
      expect(await JsonFileSettingsStore(file).load(), AppSettings());
    });
  });
}
