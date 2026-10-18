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

// Generates public/fonts/MaterialSymbolsRounded.woff2, the icons of
// shared/icons.json cut out of Material Symbols Rounded with
// shared/tools/icon_font.py (which needs Python 3 and fontTools), as for the
// web front end. Only the fill axis is kept: the user interface draws every
// icon at weight 400, grade 0 and optical size 24, and fills some of them.
//
// The variable font is taken from VFOX_PLUS_SYMBOLS (the sandbox images have
// it) or downloaded once into node_modules/.cache/material-symbols.
//
//   npm run assets

import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
const tool = fileURLToPath(new URL('../../shared/tools/icon_font.py', import.meta.url));

let fonts = process.env.VFOX_PLUS_SYMBOLS;
if (fonts === undefined || fonts === '') {
  fonts = `${root}node_modules/.cache/material-symbols`;
  execFileSync('python3', [tool, 'download', fonts], { stdio: 'inherit' });
}
mkdirSync(`${root}public/fonts`, { recursive: true });
execFileSync(
  'python3',
  [
    tool,
    'subset',
    '--fonts',
    fonts,
    '--output',
    `${root}public/fonts/MaterialSymbolsRounded.woff2`,
    '--pin',
    'wght=400',
    '--pin',
    'GRAD=0',
    '--pin',
    'opsz=24',
  ],
  { stdio: 'inherit' },
);
// The license of the font ships with the application.
copyFileSync(`${fonts}/LICENSE`, `${root}public/fonts/MaterialSymbolsRounded-LICENSE.txt`);
