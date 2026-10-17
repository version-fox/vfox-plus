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

// What the pages show apart from how they show it, and the theme for Ionic.

import { describe, expect, it } from 'vitest';

import { colorTokens } from '@web/ui/theme';
import { symbolOf } from '../src/ui/icons';
import {
  avatarLook,
  CONFIG_KEYS,
  DECORATIVE_SHAPES,
  KNOWN_CONFIG_KEYS,
  repositoryPath,
  stableHash,
} from '../src/ui/presentation';
import { ionicTokens, mix, rgbOf } from '../src/ui/theme';

describe('presentation', () => {
  it('hashes names as the other implementations do', () => {
    // FNV-1a over UTF-16 code units, as in the Flutter implementation.
    expect(stableHash('nodejs')).toBe(527731554);
    expect(stableHash('maven')).toBe(2799867016);
    expect(stableHash('')).toBe(0x811c9dc5);
    expect(stableHash('😀')).toBe(3409036472);
  });

  it('gives every plugin its own avatar', () => {
    expect(avatarLook('nodejs')).toEqual({
      shape: 'pentagon',
      container: 'primary-fixed-dim',
      content: 'on-primary-fixed',
      initial: 'N',
    });
    expect(avatarLook('maven')).toMatchObject({ shape: '6-sided-cookie', container: 'secondary', initial: 'M' });
    expect(avatarLook('vfox+')).toMatchObject({ shape: DECORATIVE_SHAPES[11], container: 'tertiary-container' });
    expect(avatarLook('').initial).toBe('?');
    expect(avatarLook('ñandú').initial).toBe('Ñ');
  });

  it('shows repositories by their path', () => {
    expect(repositoryPath('https://github.com/version-fox/vfox-nodejs')).toBe('version-fox/vfox-nodejs');
    expect(repositoryPath('https://example.com/')).toBe('https://example.com/');
    expect(repositoryPath('not a url')).toBe('not a url');
  });

  it('edits the known configuration keys with dedicated fields', () => {
    expect(KNOWN_CONFIG_KEYS.size).toBe(Object.keys(CONFIG_KEYS).length);
    expect(KNOWN_CONFIG_KEYS.has('registry.address')).toBe(true);
  });

  it('names icons by their Material Symbol', () => {
    expect(symbolOf('sdks')).toBe('deployed_code');
    expect(symbolOf('close')).toBe('close');
  });
});

describe('theme', () => {
  it('converts and mixes colours', () => {
    expect(rgbOf('#336699')).toBe('51, 102, 153');
    expect(mix('#000000', '#ffffff', 0.5)).toBe('#808080');
    expect(mix('#336699', '#336699', 0.3)).toBe('#336699');
  });

  it('derives the colours of Ionic from the Material scheme', () => {
    const colors = colorTokens(0xfff28c1b, false);
    const tokens = ionicTokens(colors);
    expect(tokens['--ion-color-primary']).toBe(colors['--md-sys-color-primary']);
    expect(tokens['--ion-color-primary-contrast']).toBe(colors['--md-sys-color-on-primary']);
    expect(tokens['--ion-color-tonal']).toBe(colors['--md-sys-color-secondary-container']);
    expect(tokens['--ion-color-danger']).toBe(colors['--md-sys-color-error']);
    expect(tokens['--ion-background-color']).toBe(colors['--md-sys-color-surface']);
    expect(tokens['--ion-text-color-rgb']).toBe(rgbOf(colors['--md-sys-color-on-surface'] ?? ''));
    expect(tokens['--ion-background-color-step-500']).toBe(
      mix(colors['--md-sys-color-surface'] ?? '', colors['--md-sys-color-on-surface'] ?? '', 0.5),
    );
    expect(Object.keys(tokens).filter((key) => key.includes('-step-'))).toHaveLength(38);
    // A scheme without a role falls back to black rather than failing.
    expect(ionicTokens({})['--ion-color-primary']).toBe('#000000');
  });
});
