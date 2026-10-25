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

// Writes golden.json: the tonal spot scheme of the Material colour utilities
// for a set of seed colours, as @material/material-color-utilities computes
// it. The Go port in ../ is tested against it.
//
// This file intentionally stays JavaScript: it must execute the JavaScript
// reference implementation, because the value of the golden test is that an
// independent implementation agrees with the Go port bit for bit. A Go
// program could only shell out to node (saving nothing) or reimplement the
// reference (testing the port against itself, which proves nothing).
//
//   npm install --prefix "$TMP" @material/material-color-utilities@0.4.0
//   node generate.mjs "$TMP/node_modules/@material/material-color-utilities" > golden.json
//
// With no argument, the package is read from node_modules next to this script.
// The seeds are the swatches of the application, a few edge cases and
// pseudo-random colours, which come from a fixed generator so that the output
// does not change.

import { register } from 'node:module';
import { pathToFileURL } from 'node:url';

// The package imports its own files without the extension, which bundlers
// resolve but Node does not: add it.
register(
  'data:text/javascript,' +
    encodeURIComponent(`
      export async function resolve(specifier, context, next) {
        try {
          return await next(specifier, context);
        } catch (error) {
          if (error.code === 'ERR_MODULE_NOT_FOUND' && specifier.startsWith('.') && !specifier.endsWith('.js')) {
            return next(specifier + '.js', context);
          }
          throw error;
        }
      }`),
);
const packageRoot =
  process.argv[2] === undefined
    ? new URL('./node_modules/@material/material-color-utilities/', import.meta.url).href
    : pathToFileURL(process.argv[2].endsWith('/') ? process.argv[2] : `${process.argv[2]}/`).href;
const { Hct, hexFromArgb, MaterialDynamicColors, SchemeTonalSpot } = await import(
  new URL('index.js', packageRoot).href
);

const seeds = [
  0xfff28c1b, 0xffd9473b, 0xffc2407a, 0xff7e57c2, 0xff3f6fd8, 0xff0b8a9a, 0xff3e8e41, 0xff8d7b1e,
  // The corners of the gamut, greys and the hue range that tone 99 treats
  // specially (yellow).
  0xff000000, 0xffffffff, 0xff808080, 0xffff0000, 0xff00ff00, 0xff0000ff, 0xffffff00, 0xff00ffff,
  0xffff00ff, 0xfffff44f, 0xffe6e100, 0xff6750a4, 0xff1b6ef3, 0xff0061a4,
];
// A linear congruential generator (Numerical Recipes), for repeatable colours.
let state = 20260718;
for (let i = 0; i < 60; i++) {
  state = (Math.imul(state, 1664525) + 1013904223) >>> 0;
  seeds.push((0xff000000 | (state >>> 8)) >>> 0);
}

const colors = new MaterialDynamicColors();
const output = {};
for (const seed of seeds) {
  const entry = {};
  for (const dark of [false, true]) {
    const scheme = new SchemeTonalSpot(Hct.fromInt(seed), dark, 0);
    const tokens = {};
    for (const color of [...colors.allColors, colors.surfaceVariant(), colors.shadow(), colors.scrim()]) {
      tokens[color.name.replace(/_/g, '-').toLowerCase()] = hexFromArgb(color.getArgb(scheme));
    }
    entry[dark ? 'dark' : 'light'] = tokens;
  }
  output[`#${(seed & 0xffffff).toString(16).padStart(6, '0')}`] = entry;
}
// One seed per line keeps the file readable and small.
console.log(
  `{\n${Object.entries(output)
    .map(([seed, schemes]) => `${JSON.stringify(seed)}: ${JSON.stringify(schemes)}`)
    .join(',\n')}\n}`,
);
