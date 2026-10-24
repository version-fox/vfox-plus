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

// Writes shapes.json: the outlines of the Material 3 shapes that vfox-plus
// draws, as SVG paths, from the shape component of @m3e/web (MIT, matraic).
// The Go shapes are tested against them (shapes_test.go).
//
//   npm install --prefix "$TMP" @m3e/web@2.10.1
//   node generate.mjs "$TMP/node_modules/@m3e/web/dist/shape.js" > shapes.json
//
// With no argument, the module is read from node_modules next to this script.

import { readFileSync } from 'node:fs';

const source = readFileSync(
  process.argv[2] ?? new URL('./node_modules/@m3e/web/dist/shape.js', import.meta.url),
  'utf8',
);
const match = source.match(/const SHAPE_PATHS = (\{[\s\S]*?\n\});/);
if (match === null) {
  throw new Error('SHAPE_PATHS not found');
}
const paths = new Function(`return (${match[1]})`)();
const names = [
  '4-leaf-clover', '4-sided-cookie', '6-sided-cookie', '8-leaf-clover', '9-sided-cookie', '12-sided-cookie',
  'oval', 'pentagon', 'pill', 'soft-burst', 'sunny', 'flower', 'gem', 'arch', 'puffy-diamond', 'circle',
];
console.log(`{\n${names.map((name) => `${JSON.stringify(name)}: ${JSON.stringify(paths[name])}`).join(',\n')}\n}`);
