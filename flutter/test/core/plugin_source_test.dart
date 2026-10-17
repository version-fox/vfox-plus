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
import 'package:vfox_plus/src/core/vfox/plugin_source.dart';

void main() {
  late Directory directory;

  setUp(() => directory = Directory.systemTemp.createTempSync('vfox-plugin-source'));
  tearDown(() => directory.deleteSync(recursive: true));

  test('only local .lua files are moved by vfox', () {
    expect(vfoxMovesPluginSource('/home/u/plugin.lua'), isTrue);
    expect(vfoxMovesPluginSource('/home/u/PLUGIN.LUA'), isTrue);
    expect(vfoxMovesPluginSource('/home/u/plugin.zip'), isFalse);
    expect(vfoxMovesPluginSource('https://example.com/plugin.lua'), isFalse);
    expect(isRemotePluginSource('http://127.0.0.1:8080/p.zip'), isTrue);
  });

  test('the temporary directory of vfox is in its home directory', () {
    expect(vfoxTemporaryDirectory(<String, String>{'HOME': directory.path}).path, '${directory.path}/.vfox/tmp');
    Directory('${directory.path}/.version-fox').createSync();
    expect(vfoxTemporaryDirectory(<String, String>{'HOME': directory.path}).path, '${directory.path}/.version-fox/tmp');
    expect(vfoxTemporaryDirectory(const <String, String>{}).path, Directory.systemTemp.path);
  });

  test('URLs and archives are passed through', () async {
    for (final source in <String>['https://example.com/p.lua', '${directory.path}/p.zip']) {
      final PreparedPluginSource prepared = await preparePluginSource(source, temporaryRoot: directory);
      expect(prepared.path, source);
      expect(prepared.isCopy, isFalse);
      await prepared.dispose();
    }
  });

  test('a local .lua file is copied and the original is kept', () async {
    final original = File('${directory.path}/mine.lua')..writeAsStringSync('PLUGIN = {}');
    // Created when missing, as vfox would.
    final temporaryRoot = Directory('${directory.path}/.vfox/tmp');
    final PreparedPluginSource prepared = await preparePluginSource(original.path, temporaryRoot: temporaryRoot);
    expect(prepared.isCopy, isTrue);
    expect(prepared.path, isNot(original.path));
    expect(prepared.path, endsWith('/mine.lua'));
    expect(File(prepared.path).readAsStringSync(), 'PLUGIN = {}');

    // vfox moves the copy away; the original stays.
    File(prepared.path).deleteSync();
    expect(original.existsSync(), isTrue);

    await prepared.dispose();
    expect(temporaryRoot.listSync(), isEmpty);
  });

  test('a missing file is left for vfox to report', () async {
    final Directory temporaryRoot = Directory('${directory.path}/tmp')..createSync();
    final PreparedPluginSource prepared = await preparePluginSource(
      '${directory.path}/missing.lua',
      temporaryRoot: temporaryRoot,
    );
    expect(prepared.path, '${directory.path}/missing.lua');
    expect(prepared.isCopy, isFalse);
    expect(temporaryRoot.listSync(), isEmpty);
  });
}
