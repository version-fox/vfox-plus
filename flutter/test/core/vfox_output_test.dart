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

// Edge cases of the parsers that the captured fixtures do not cover.

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/models.dart';
import 'package:vfox_plus/src/core/vfox/vfox_output.dart';

void main() {
  group('parseVfoxVersion', () {
    test('reads release and pre-release versions', () {
      expect(parseVfoxVersion('vfox version 1.0.12\n'), '1.0.12');
      expect(parseVfoxVersion('vfox version 2.0.0-beta.1'), '2.0.0-beta.1');
    });

    test('returns null without a version', () {
      expect(parseVfoxVersion('command not found'), isNull);
    });
  });

  group('parseCurrent', () {
    test('strips the v prefix only once', () {
      expect(parseCurrent('go -> vv1.2\n'), <String, String?>{'go': 'v1.2'});
    });

    test('ignores colour codes and unrelated lines', () {
      const output = 'Warning: something\njava -> \x1B[92mv21\x1B[0m\nmaven -> N/A \n';
      expect(parseCurrent(output), <String, String?>{'java': '21', 'maven': null});
    });
  });

  group('parseInstalledTree', () {
    test('handles a single plugin without versions', () {
      expect(parseInstalledTree('All installed sdk versions\n└──maven\n'), <String, List<String>>{'maven': <String>[]});
    });

    test('ignores the header and blank lines', () {
      const output = 'All installed sdk versions\n\n└─┬nodejs\n  └──24.11.1\n\n';
      expect(parseInstalledTree(output), <String, List<String>>{
        'nodejs': <String>['24.11.1'],
      });
    });
  });

  group('mergeInstalledSdks', () {
    test('keeps plugins that only one command reported', () {
      final List<InstalledSdk> sdks = mergeInstalledSdks(
        <String, List<String>>{
          'java': <String>['21'],
        },
        <String, String?>{'java': '21', 'go': null},
      );
      expect(sdks, <InstalledSdk>[
        InstalledSdk(name: 'java', versions: const <String>['21'], current: '21'),
        InstalledSdk(name: 'go', versions: const <String>[]),
      ]);
    });
  });

  group('parseSearch', () {
    test('reads notes, additions and the installed marker', () {
      final List<AvailableVersion> versions = parseSearch(
        'Available versions:\n'
        ' - 24.22.0 (LTS) [npm 11.19.1,corepack 0.34.0] (installed)\n'
        ' - 1.0 (release candidate)\n'
        ' - 0.1.14 [npm ]\n'
        ' - 2.0\n',
      );
      expect(versions, <AvailableVersion>[
        AvailableVersion(
          version: '24.22.0',
          note: 'LTS',
          additions: const <VersionAddition>[
            VersionAddition(name: 'npm', version: '11.19.1'),
            VersionAddition(name: 'corepack', version: '0.34.0'),
          ],
          installed: true,
        ),
        AvailableVersion(version: '1.0', note: 'release candidate'),
        AvailableVersion(
          version: '0.1.14',
          additions: const <VersionAddition>[VersionAddition(name: 'npm', version: '')],
        ),
        AvailableVersion(version: '2.0'),
      ]);
    });

    test('ignores lines that are not list entries', () {
      expect(parseSearch('Please select a version\n -\n'), isEmpty);
    });
  });

  group('parsePluginInfo', () {
    test('keeps separators that appear in the description', () {
      final PluginInfo? info = parsePluginInfo('a\x1F1.0\x1Fhttps://x\x1F/p\x1Fone\x1Ftwo');
      expect(info?.description, 'one\x1Ftwo');
      expect(info?.homepage, 'https://x');
    });

    test('rejects incomplete output', () {
      expect(parsePluginInfo('nodejs not supported'), isNull);
    });
  });

  group('parseInstallPath', () {
    test('maps notfound to null', () {
      expect(parseInstallPath('notfound\n'), isNull);
      expect(parseInstallPath('/opt/sdk/v-1\n'), '/opt/sdk/v-1');
    });
  });

  group('parseConfigList', () {
    test('keeps empty values and values containing spaces', () {
      expect(parseConfigList('proxy.url = \nfoo.bar = a = b\nnot a setting\n'), const <ConfigEntry>[
        ConfigEntry(key: 'proxy.url', value: ''),
        ConfigEntry(key: 'foo.bar', value: 'a = b'),
      ]);
    });

    test('accepts a key whose trailing space was trimmed', () {
      expect(parseConfigList('registry.address =\n'), const <ConfigEntry>[
        ConfigEntry(key: 'registry.address', value: ''),
      ]);
    });
  });

  group('parseDownloadProgress', () {
    test('reads the percentage of a progress bar redraw', () {
      expect(parseDownloadProgress('Downloading...  42% [===>   ] (1 MB/s) [1s:2s]'), 0.42);
      expect(parseDownloadProgress('Downloading... 100% [======]'), 1.0);
    });

    test('returns null for other lines', () {
      expect(parseDownloadProgress('Preinstalling maven@3.9.16...'), isNull);
    });
  });

  group('summarizeError', () {
    test('prefers standard error', () {
      expect(summarizeError('Removing...\n', 'Use the -y flag\n'), 'Use the -y flag');
    });

    test('skips progress bars and the generic install trailer', () {
      expect(
        summarizeError(
          'Preinstalling x@1...\nfailed to download x@1 file, err:EOF\nfailed to install x\n',
          '\rDownloading...  3% [>   ]\r      \r',
        ),
        'failed to download x@1 file, err:EOF',
      );
    });

    test('finds the reason on standard output when the trailer is on standard error', () {
      // vfox 1.0.13 prints the error a command returns to standard error.
      expect(
        summarizeError(
          'Preinstalling x@1...\nfailed to download x@1 file, err:EOF\n',
          '\rDownloading...  3% [>   ]\r      \rfailed to install x\n',
        ),
        'failed to download x@1 file, err:EOF',
      );
    });

    test('returns an empty string without output', () {
      expect(summarizeError('', '  \n'), '');
    });
  });
}
