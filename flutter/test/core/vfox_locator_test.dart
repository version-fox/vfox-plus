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

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/vfox/vfox_locator.dart';

void main() {
  test('prefers PATH, in order', () {
    final locator = VfoxLocator(
      environment: const <String, String>{'PATH': '/a:/b', 'HOME': '/home/u'},
      isExecutable: (String path) => path == '/b/vfox' || path == '/usr/local/bin/vfox',
    );
    expect(locator.locate(), '/b/vfox');
  });

  test('falls back to well-known directories missing from a launcher PATH', () {
    final locator = VfoxLocator(
      environment: const <String, String>{'PATH': '/usr/bin', 'HOME': '/home/u'},
      isExecutable: (String path) => path == '/home/u/.local/bin/vfox',
    );
    expect(locator.locate(), '/home/u/.local/bin/vfox');
    expect(locator.searchDirectories.toSet(), hasLength(locator.searchDirectories.length));
  });

  test('returns null when nothing is found', () {
    final locator = VfoxLocator(environment: const <String, String>{}, isExecutable: (_) => false);
    expect(locator.locate(), isNull);
  });

  test('checks the execute permission of real files', () {
    final Directory directory = Directory.systemTemp.createTempSync('vfox-locator');
    addTearDown(() => directory.deleteSync(recursive: true));
    final file = File('${directory.path}/vfox')..writeAsStringSync('#!/bin/sh\n');
    final locator = VfoxLocator(environment: const <String, String>{});
    expect(locator.isUsable(file.path), isFalse);
    Process.runSync('chmod', <String>['+x', file.path]);
    expect(locator.isUsable(file.path), isTrue);
    expect(locator.isUsable(directory.path), isFalse);
    expect(locator.isUsable('${directory.path}/missing'), isFalse);
  });
}
