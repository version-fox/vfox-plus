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

// Checks that keep the project consistent: versions, and the messages and
// icons that the user interface uses.

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import { APP_VERSION } from '@web/version';

const root = fileURLToPath(new URL('..', import.meta.url));
const shared = fileURLToPath(new URL('../../shared', import.meta.url));

function sources(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const path = join(directory, entry);
    return statSync(path).isDirectory() ? sources(path) : /\.(ts|vue)$/.test(path) ? [path] : [];
  });
}

describe('project', () => {
  const manifest = JSON.parse(readFileSync(`${root}package.json`, 'utf8')) as {
    version: string;
    devDependencies: Record<string, string>;
  };

  it('has the version of the other implementations', () => {
    expect(manifest.version).toBe(APP_VERSION);
  });

  it('ships the Electron that it pins', () => {
    // The runtime is packaged as it is, so its version never floats.
    expect(manifest.devDependencies['electron']).toMatch(/^\d+\.\d+\.\d+$/);
  });

  it('uses only messages that the catalogue has', () => {
    const catalogue = JSON.parse(readFileSync(`${shared}/i18n/app_en.arb`, 'utf8')) as Record<string, unknown>;
    const used = new Set<string>();
    for (const file of sources(`${root}src/ui`)) {
      for (const match of readFileSync(file, 'utf8').matchAll(/\bt\('([A-Za-z]+)'/g)) {
        used.add(match[1] ?? '');
      }
    }
    expect(used.size).toBeGreaterThan(100);
    expect([...used].filter((key) => !(key in catalogue))).toEqual([]);
  });

  it('uses only icons of shared/icons.json', () => {
    const icons = (JSON.parse(readFileSync(`${shared}/icons.json`, 'utf8')) as { icons: Record<string, string> }).icons;
    const used = new Set<string>();
    for (const file of sources(`${root}src/ui`)) {
      for (const match of readFileSync(file, 'utf8').matchAll(/(?<=\s)(?:name|icon)="([A-Za-z]+)"/g)) {
        used.add(match[1] ?? '');
      }
    }
    expect(used.size).toBeGreaterThan(30);
    expect([...used].filter((name) => !(name in icons))).toEqual([]);
  });
});
