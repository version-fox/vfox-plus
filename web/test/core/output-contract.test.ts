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

import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

import { splitTerminalLines, stripAnsi } from '../../src/core/terminal-text';
import {
  parseAvailablePlugins,
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
import { readCapture, SHARED } from '../support/fakes';

interface Case {
  readonly name: string;
  readonly expect: Readonly<Record<string, unknown>>;
}

function indexed(items: unknown[]): unknown {
  return { count: items.length, items: Object.fromEntries(items.map((item, index) => [String(index), item])) };
}

/** Turns a parser result into the JSON shape used by `cases.json`. */
const checks: Readonly<Record<string, (stdout: string, stderr: string) => unknown>> = {
  version: (stdout) => parseVfoxVersion(stdout),
  current: (stdout) => Object.fromEntries(parseCurrent(stdout)),
  list: (stdout) => Object.fromEntries(parseInstalledTree(stdout)),
  search: (stdout) => indexed(parseSearch(stdout)),
  available: (stdout) => indexed(parseAvailablePlugins(stdout)),
  pluginInfo: (stdout) => parsePluginInfo(stdout),
  installPath: (stdout) => parseInstallPath(stdout),
  config: (stdout) => parseConfigList(stdout),
  finalProgress: (_, stderr) => {
    let last: number | null = null;
    for (const line of splitTerminalLines(stripAnsi(stderr))) {
      last = parseDownloadProgress(line) ?? last;
    }
    return last;
  },
  error: summarizeError,
};

/** For `search` and `available`, only the listed items are compared. */
function select(check: string, actual: unknown, expected: unknown): unknown {
  if (check !== 'search' && check !== 'available') {
    return actual;
  }
  const actualMap = actual as { count: number; items: Record<string, unknown> };
  const expectedMap = expected as { items: Record<string, unknown> };
  return {
    count: actualMap.count,
    items: Object.fromEntries(Object.keys(expectedMap.items).map((index) => [index, actualMap.items[index]])),
  };
}

const fixtures = `${SHARED}/fixtures`;
const versions = readdirSync(fixtures)
  .filter((entry) => existsSync(`${fixtures}/${entry}/cases.json`))
  .sort();

it('finds the fixtures', () => {
  expect(versions.length).toBeGreaterThan(0);
});

for (const version of versions) {
  const manifest = JSON.parse(readFileSync(`${fixtures}/${version}/cases.json`, 'utf8')) as {
    vfox: string;
    cases: Case[];
  };
  describe(`vfox ${manifest.vfox}`, () => {
    for (const testCase of manifest.cases) {
      const stdout = readCapture(`${fixtures}/${version}/${testCase.name}.stdout`);
      const stderr = readCapture(`${fixtures}/${version}/${testCase.name}.stderr`);
      for (const [check, expected] of Object.entries(testCase.expect)) {
        it(`${testCase.name}: ${check}`, () => {
          const parse = checks[check];
          expect(parse, `Unknown parser "${check}" in cases.json`).toBeDefined();
          const actual = select(check, parse?.(stdout, stderr), expected);
          // Compare through JSON so that maps and lists compare by value.
          expect(JSON.parse(JSON.stringify(actual)) as unknown).toEqual(expected);
        });
      }
    }
  });
}
