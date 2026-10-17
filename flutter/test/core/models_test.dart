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

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/models.dart';
import 'package:vfox_plus/src/core/vfox/vfox_command.dart';

void main() {
  test('models compare by value', () {
    expect(
      InstalledSdk(name: 'a', versions: const <String>['1'], current: '1'),
      InstalledSdk(name: 'a', versions: const <String>['1'], current: '1'),
    );
    expect(
      InstalledSdk(name: 'a', versions: const <String>['1']).hashCode,
      InstalledSdk(name: 'a', versions: const <String>['1']).hashCode,
    );
    expect(
      InstalledSdk(name: 'a', versions: const <String>['1']),
      isNot(InstalledSdk(name: 'a', versions: const <String>['2'])),
    );
    expect(
      const PluginInfo(name: 'a', version: '1', homepage: 'h', installPath: 'p', description: 'd'),
      const PluginInfo(name: 'a', version: '1', homepage: 'h', installPath: 'p', description: 'd'),
    );
    expect(
      const AvailablePlugin(name: 'a', homepage: 'h', official: true).hashCode,
      const AvailablePlugin(name: 'a', homepage: 'h', official: true).hashCode,
    );
    expect(AvailableVersion(version: '1', note: 'LTS'), AvailableVersion(version: '1', note: 'LTS'));
    expect(AvailableVersion(version: '1').hashCode, AvailableVersion(version: '1').hashCode);
    expect(const ConfigEntry(key: 'k', value: 'v').hashCode, const ConfigEntry(key: 'k', value: 'v').hashCode);
  });

  test('copyWith and helpers', () {
    final sdk = InstalledSdk(name: 'a', versions: const <String>['1'], current: '1');
    expect(sdk.hasVersions, isTrue);
    expect(sdk.copyWith(clearCurrent: true).current, isNull);
    expect(sdk.copyWith(versions: const <String>[]).hasVersions, isFalse);
    expect(() => sdk.versions.add('2'), throwsUnsupportedError);
  });

  test('descriptions are readable', () {
    expect('${InstalledSdk(name: 'a', versions: const <String>['1'])}', contains('a'));
    expect(
      '${const PluginInfo(name: 'a', version: '1', homepage: '', installPath: '', description: '')}',
      'PluginInfo(a 1)',
    );
    expect('${const AvailablePlugin(name: 'a', homepage: '', official: false)}', contains('official: false'));
    expect('${AvailableVersion(version: '1')}', contains('1'));
    expect('${const VersionAddition(name: 'npm', version: '')}', 'npm');
    expect('${const VersionAddition(name: 'npm', version: '11')}', 'npm 11');
    expect('${const ConfigEntry(key: 'k', value: 'v')}', 'k = v');
  });

  test('commands describe themselves', () {
    const command = VfoxCommand(
      operation: VfoxOperation.install,
      arguments: <String>['install', '--yes', 'a@1'],
      sdk: 'a',
      version: '1',
    );
    expect('$command', 'vfox install --yes a@1');
    expect(const VfoxCommand(operation: VfoxOperation.updateAllPlugins, arguments: <String>[]).subject, isNull);
    expect(
      '${const VfoxException('boom', arguments: <String>['list'], exitCode: 1)}',
      'VfoxException: boom (vfox list, exit code 1)',
    );
    expect('${const VfoxNotFoundException('vfox', 'missing')}', 'VfoxNotFoundException: vfox: missing');
  });
}
