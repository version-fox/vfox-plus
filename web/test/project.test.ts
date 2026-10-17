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

// Checks that keep the project consistent: versions, translations in use,
// icons and the motion tokens.

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import { EXPRESSIVE, motionTokens, settleDuration, springEasing, springState } from '../src/ui/motion';
import { APP_VERSION } from '../src/version';
import { SHARED } from './support/fakes';

const root = fileURLToPath(new URL('..', import.meta.url));

function sources(directory: string): string[] {
  return readdirSync(directory).flatMap((entry) => {
    const path = join(directory, entry);
    return statSync(path).isDirectory() ? sources(path) : path.endsWith('.ts') ? [path] : [];
  });
}

describe('project', () => {
  it('has one version', () => {
    const manifest = JSON.parse(readFileSync(`${root}package.json`, 'utf8')) as { version: string };
    expect(APP_VERSION).toBe(manifest.version);
  });

  it('uses only messages that the catalogue has', () => {
    const catalogue = JSON.parse(readFileSync(`${SHARED}/i18n/app_en.arb`, 'utf8')) as Record<string, unknown>;
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
    const icons = (JSON.parse(readFileSync(`${SHARED}/icons.json`, 'utf8')) as { icons: Record<string, string> }).icons;
    for (const file of sources(`${root}src/ui`)) {
      for (const match of readFileSync(file, 'utf8').matchAll(/iconTemplate\('([A-Za-z]+)'/g)) {
        expect(icons, file).toHaveProperty(match[1] ?? '');
      }
    }
  });
});

describe('motion', () => {
  it('settles springs the way the Flutter implementation measures them', () => {
    expect(settleDuration(EXPRESSIVE.slowEffects)).toBeGreaterThan(300);
    expect(settleDuration(EXPRESSIVE.slowEffects)).toBeLessThan(380);
    expect(settleDuration(EXPRESSIVE.fastSpatial)).toBeLessThan(settleDuration(EXPRESSIVE.slowSpatial));
  });

  it('lets spatial springs overshoot and effects springs not', () => {
    const peak = (spring: typeof EXPRESSIVE.fastSpatial): number =>
      Math.max(...Array.from({ length: 200 }, (_, i) => springState(spring, 0, 1, 0, i / 400).position));
    expect(peak(EXPRESSIVE.fastSpatial)).toBeGreaterThan(1.05);
    expect(peak(EXPRESSIVE.fastEffects)).toBeLessThanOrEqual(1.0001);
  });

  it('turns springs into CSS easing and transition tokens', () => {
    expect(springEasing(EXPRESSIVE.defaultEffects, 4)).toMatch(/^linear\(0, [\d.]+, [\d.]+, [\d.]+, 1\)$/);
    const tokens = motionTokens(EXPRESSIVE, false);
    expect(Object.keys(tokens)).toContain('--md-sys-motion-spring-fast-spatial');
    expect(tokens['--md-sys-motion-spring-slow-effects']).toMatch(/^\d+ms linear\(/);
    expect(motionTokens(EXPRESSIVE, true)['--md-sys-motion-spring-fast-spatial']).toBe('0ms linear');
  });
});
