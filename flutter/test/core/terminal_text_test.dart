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

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/terminal_text.dart';

import '../support/fakes.dart';

void main() {
  group('stripAnsi', () {
    test('removes colours, cursor movement and hyperlinks', () {
      expect(stripAnsi('\x1B[92mv1\x1B[0m \x1B[K\x1B]8;;https://x\x07link\x1B]8;;\x07'), 'v1 link');
    });

    test('leaves plain text alone', () {
      expect(stripAnsi('├──maven ✓'), '├──maven ✓');
    });
  });

  group('splitTerminalLines', () {
    test('treats a lone carriage return as a line break', () {
      expect(splitTerminalLines('a\r\nb\rc\nd'), <String>['a', 'b', 'c', 'd']);
    });
  });

  group('terminalLines', () {
    test('joins lines and characters split across chunks', () async {
      final List<int> bytes = utf8.encode('第一行\nsecond\rthird');
      // Split inside the first, three-byte character and inside "second".
      final chunks = <List<int>>[bytes.sublist(0, 1), bytes.sublist(1, 12), bytes.sublist(12)];
      expect(await terminalLines(Stream<List<int>>.fromIterable(chunks)).toList(), <String>['第一行', 'second', 'third']);
    });

    test('drops blank redraws and colours of the real install output', () async {
      final List<String> lines = await terminalLines(Stream<List<int>>.value(utf8.encode(fixtureStderr('install'))))
          .toList();
      expect(lines, isNotEmpty);
      expect(lines.every((String line) => line.startsWith('Downloading...')), isTrue);
      expect(lines.any((String line) => line.contains('\x1B')), isFalse);
    });
  });
}
