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

// Checks that keep the project consistent with itself and with the shared
// resources in ../shared.

import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/ui/icons/app_icons.g.dart';
import 'package:vfox_plus/src/version.dart';

/// The Dart files under [directory], with paths relative to the package root.
List<File> _dartFiles(String directory) =>
    Directory(directory)
        .listSync(recursive: true)
        .whereType<File>()
        .where((File f) => f.path.endsWith('.dart'))
        .toList();

List<String> _imports(File file) => RegExp(
  r'''^\s*(?:import|export)\s+['"]([^'"]+)['"]''',
  multiLine: true,
).allMatches(file.readAsStringSync()).map((RegExpMatch m) => m.group(1)!).toList();

Map<String, Object?> _arb(String locale) =>
    jsonDecode(File('../shared/i18n/app_$locale.arb').readAsStringSync()) as Map<String, Object?>;

/// The placeholders that the English catalogue declares for [key].
Set<String> _declaredPlaceholders(Map<String, Object?> catalogue, String key) {
  final metadata = catalogue['@$key'] as Map<String, Object?>?;
  final placeholders = metadata?['placeholders'] as Map<String, Object?>?;
  return placeholders?.keys.toSet() ?? const <String>{};
}

void main() {
  group('architecture', () {
    // The layers are described in docs/architecture.md. Keeping core free of
    // Flutter makes it the reference for implementations in other stacks.
    test('core depends on Dart only', () {
      for (final File file in _dartFiles('lib/src/core')) {
        for (final String import in _imports(file)) {
          expect(
            import.startsWith('package:'),
            isFalse,
            reason: '${file.path} imports $import; lib/src/core must only use dart: libraries and other core files',
          );
        }
      }
    });

    test('state depends on core and flutter/foundation only', () {
      for (final File file in _dartFiles('lib/src/state')) {
        for (final String import in _imports(file)) {
          final bool allowed =
              !import.startsWith('package:') ||
              import == 'package:flutter/foundation.dart' ||
              import.startsWith('dart:');
          expect(allowed, isTrue, reason: '${file.path} imports $import');
          expect(import.contains('/ui/'), isFalse, reason: '${file.path} must not depend on the UI');
        }
      }
    });

    test('Material comes from material_ui, never from the Flutter SDK', () {
      for (final File file in <File>[..._dartFiles('lib'), ..._dartFiles('test')]) {
        expect(
          _imports(file),
          isNot(contains('package:flutter/material.dart')),
          reason: '${file.path} must import package:material_ui/material_ui.dart instead',
        );
      }
    });
  });

  group('translations', () {
    final Map<String, Object?> english = _arb('en');
    final Map<String, Object?> chinese = _arb('zh');
    final Set<String> messageKeys = english.keys.where((String key) => !key.startsWith('@')).toSet();

    test('every message has a description', () {
      for (final key in messageKeys) {
        final Object? metadata = english['@$key'];
        expect(metadata, isA<Map<String, Object?>>(), reason: '@$key is missing');
        expect((metadata! as Map<String, Object?>)['description'], isNotEmpty, reason: '@$key has no description');
      }
    });

    test('Simplified Chinese translates every message and nothing else', () {
      final Set<String> chineseKeys = chinese.keys.where((String key) => !key.startsWith('@')).toSet();
      expect(chineseKeys.difference(messageKeys), isEmpty, reason: 'Obsolete translations');
      expect(messageKeys.difference(chineseKeys), isEmpty, reason: 'Missing translations');
    });

    test('translations keep the placeholders and are not empty', () {
      for (final key in messageKeys) {
        final translated = chinese[key]! as String;
        expect(translated.trim(), isNotEmpty, reason: key);
        for (final String placeholder in _declaredPlaceholders(english, key)) {
          expect(
            translated.contains('{$placeholder}') || translated.contains('{$placeholder,'),
            isTrue,
            reason: 'The translation of $key does not use {$placeholder}',
          );
        }
      }
    });
  });

  test('the version matches pubspec.yaml', () {
    final RegExpMatch? match = RegExp(
      r'^version:\s*([^+\s]+)',
      multiLine: true,
    ).firstMatch(File('pubspec.yaml').readAsStringSync());
    expect(match?.group(1), appVersion);
  });

  test('AppIcons matches the shared icon manifest', () {
    final manifest = jsonDecode(File('../shared/icons.json').readAsStringSync()) as Map<String, Object?>;
    final icons = manifest['icons']! as Map<String, Object?>;
    expect(AppIcons.all.keys.toSet(), icons.keys.toSet(), reason: 'Run `dart run tool/generate_icon_font.dart`');
    final font = File('assets/fonts/MaterialSymbolsRounded.ttf');
    expect(font.existsSync(), isTrue, reason: 'Generate it with `dart run tool/generate_icon_font.dart`.');
    expect(font.lengthSync(), lessThan(512 * 1024));
  });
}
