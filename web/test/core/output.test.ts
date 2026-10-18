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

import { describe, expect, it } from 'vitest';

import { formatAddition } from '../../src/core/models';
import { splitTerminalLines, stripAnsi, TerminalLineSplitter } from '../../src/core/terminal-text';
import {
  mergeInstalledSdks,
  parseConfigList,
  parseCurrent,
  parseDownloadProgress,
  parseInstallPath,
  parseInstalledTree,
  parsePluginInfo,
  parseSearch,
  parseVfoxVersion,
  summarizeError,
} from '../../src/core/vfox/output';

describe('parseVfoxVersion', () => {
  it('reads release and pre-release versions', () => {
    expect(parseVfoxVersion('vfox version 1.0.12\n')).toBe('1.0.12');
    expect(parseVfoxVersion('vfox version 2.0.0-beta.1')).toBe('2.0.0-beta.1');
  });

  it('returns null without a version', () => {
    expect(parseVfoxVersion('command not found')).toBeNull();
  });
});

describe('parseCurrent', () => {
  it('strips the v prefix only once', () => {
    expect(Object.fromEntries(parseCurrent('go -> vv1.2\n'))).toEqual({ go: 'v1.2' });
  });

  it('ignores colour codes and unrelated lines', () => {
    const output = 'Warning: something\njava -> \x1B[92mv21\x1B[0m\nmaven -> N/A \n';
    expect(Object.fromEntries(parseCurrent(output))).toEqual({ java: '21', maven: null });
  });
});

describe('parseInstalledTree', () => {
  it('handles a single plugin without versions', () => {
    expect(Object.fromEntries(parseInstalledTree('All installed sdk versions\n└──maven\n'))).toEqual({ maven: [] });
  });

  it('ignores the header and blank lines', () => {
    const output = 'All installed sdk versions\n\n└─┬nodejs\n  └──24.11.1\n\n';
    expect(Object.fromEntries(parseInstalledTree(output))).toEqual({ nodejs: ['24.11.1'] });
  });
});

describe('mergeInstalledSdks', () => {
  it('keeps plugins that only one command reported', () => {
    const sdks = mergeInstalledSdks(
      new Map([['java', ['21']]]),
      new Map<string, string | null>([
        ['java', '21'],
        ['go', null],
      ]),
    );
    expect(sdks).toEqual([
      { name: 'java', versions: ['21'], current: '21' },
      { name: 'go', versions: [], current: null },
    ]);
  });
});

describe('parseSearch', () => {
  it('reads notes, additions and the installed marker', () => {
    const versions = parseSearch(
      'Available versions:\n' +
        ' - 24.22.0 (LTS) [npm 11.19.1,corepack 0.34.0] (installed)\n' +
        ' - 1.0 (release candidate)\n' +
        ' - 0.1.14 [npm ]\n' +
        ' - 2.0\n',
    );
    expect(versions).toEqual([
      {
        version: '24.22.0',
        note: 'LTS',
        additions: [
          { name: 'npm', version: '11.19.1' },
          { name: 'corepack', version: '0.34.0' },
        ],
        installed: true,
      },
      { version: '1.0', note: 'release candidate', additions: [], installed: false },
      { version: '0.1.14', note: null, additions: [{ name: 'npm', version: '' }], installed: false },
      { version: '2.0', note: null, additions: [], installed: false },
    ]);
    expect(versions[0]?.additions.map(formatAddition)).toEqual(['npm 11.19.1', 'corepack 0.34.0']);
    expect(versions[2]?.additions.map(formatAddition)).toEqual(['npm']);
  });

  it('keeps a note that is not in parentheses', () => {
    expect(parseSearch(' - 1.0 preview\n')[0]?.note).toBe('preview');
  });

  it('ignores lines that are not list entries', () => {
    expect(parseSearch('Please select a version\n -\n')).toEqual([]);
  });
});

describe('parsePluginInfo', () => {
  it('keeps separators that appear in the description', () => {
    const info = parsePluginInfo('a\x1F1.0\x1Fhttps://x\x1F/p\x1Fone\x1Ftwo');
    expect(info?.description).toBe('one\x1Ftwo');
    expect(info?.homepage).toBe('https://x');
  });

  it('rejects incomplete output', () => {
    expect(parsePluginInfo('nodejs not supported')).toBeNull();
  });
});

describe('parseInstallPath', () => {
  it('maps notfound to null', () => {
    expect(parseInstallPath('notfound\n')).toBeNull();
    expect(parseInstallPath('/opt/sdk/v-1\n')).toBe('/opt/sdk/v-1');
  });
});

describe('parseConfigList', () => {
  it('keeps empty values and values containing spaces', () => {
    expect(parseConfigList('proxy.url = \nfoo.bar = a = b\nnot a setting\n')).toEqual([
      { key: 'proxy.url', value: '' },
      { key: 'foo.bar', value: 'a = b' },
    ]);
  });

  it('accepts a key whose trailing space was trimmed', () => {
    expect(parseConfigList('registry.address =\n')).toEqual([{ key: 'registry.address', value: '' }]);
  });
});

describe('parseDownloadProgress', () => {
  it('reads the percentage of a progress bar redraw', () => {
    expect(parseDownloadProgress('Downloading...  42% [===>   ] (1 MB/s) [1s:2s]')).toBe(0.42);
    expect(parseDownloadProgress('Downloading... 100% [======]')).toBe(1);
  });

  it('returns null for other lines', () => {
    expect(parseDownloadProgress('Preinstalling maven@3.9.16...')).toBeNull();
  });
});

describe('summarizeError', () => {
  it('prefers standard error', () => {
    expect(summarizeError('Removing...\n', 'Use the -y flag\n')).toBe('Use the -y flag');
  });

  it('skips progress bars and the generic install trailer', () => {
    expect(
      summarizeError(
        'Preinstalling x@1...\nfailed to download x@1 file, err:EOF\nfailed to install x\n',
        '\rDownloading...  3% [>   ]\r      \r',
      ),
    ).toBe('failed to download x@1 file, err:EOF');
  });

  it('finds the reason on standard output when the trailer is on standard error', () => {
    // vfox 1.0.13 prints the error a command returns to standard error.
    expect(
      summarizeError(
        'Preinstalling x@1...\nfailed to download x@1 file, err:EOF\n',
        '\rDownloading...  3% [>   ]\r      \rfailed to install x\n',
      ),
    ).toBe('failed to download x@1 file, err:EOF');
  });

  it('returns an empty string without output', () => {
    expect(summarizeError('', '  \n')).toBe('');
  });
});

describe('terminal text', () => {
  it('strips CSI, OSC and two-byte escapes', () => {
    expect(stripAnsi('\x1B[1;32mok\x1B[0m \x1B]8;;https://x\x07link\x1B]8;;\x07 \x1BMx')).toBe('ok link x');
  });

  it('splits on new lines and lone carriage returns', () => {
    expect(splitTerminalLines('a\r\nb\rc\nd')).toEqual(['a', 'b', 'c', 'd']);
  });

  it('splits streamed output into clean lines', () => {
    const splitter = new TerminalLineSplitter();
    const encoder = new TextEncoder();
    const bytes = encoder.encode('one\r\x1B[32mtw');
    expect(splitter.push(bytes)).toEqual(['one']);
    // A multi-byte character split between two chunks stays intact.
    const euro = encoder.encode('€');
    expect(splitter.push(new Uint8Array([...encoder.encode('o '), euro[0] ?? 0]))).toEqual([]);
    expect(splitter.push(new Uint8Array([euro[1] ?? 0, euro[2] ?? 0, 0x0a, 0x20, 0x0a]))).toEqual(['two €']);
    expect(splitter.push(encoder.encode('last'))).toEqual([]);
    expect(splitter.close()).toEqual(['last']);
  });

  it('replaces malformed bytes', () => {
    const splitter = new TerminalLineSplitter();
    expect(splitter.push(new Uint8Array([0x61, 0xff, 0x62, 0x0a]))).toEqual(['a�b']);
  });
});
