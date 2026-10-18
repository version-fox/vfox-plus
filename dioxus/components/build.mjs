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

// Builds the assets that the executable carries into ../assets (not
// committed; the Rust build fails without them):
//
//   components.js                 index.js: the web components and window.vfoxPlus
//   e2e-driver.js                 e2e.js: the end-to-end driver, for e2e builds
//   MaterialSymbolsRounded.woff2  the icons of shared/icons.json
//   npm-licenses.txt              the licenses of the bundled packages and of the font
//
// The icons are cut out of Material Symbols Rounded with
// shared/tools/icon_font.py (Python 3 and fontTools): only the fill axis is
// kept, at weight 400, grade 0 and optical size 24. The variable font is taken
// from VFOX_PLUS_SYMBOLS (the sandbox images have it) or downloaded once into
// node_modules/.cache/material-symbols. The license check fails on a package
// whose license is not compatible with the Apache License 2.0 (see
// docs/architecture.md#dependencies).
//
//   npm run build

import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

import { build } from 'esbuild';

const root = fileURLToPath(new URL('.', import.meta.url));
const assets = fileURLToPath(new URL('../assets/', import.meta.url));
mkdirSync(assets, { recursive: true });

// The scripts are inlined into the document, as classic scripts that run
// before the application renders.
for (const [entry, output] of [
  ['index.js', 'components.js'],
  ['e2e.js', 'e2e-driver.js'],
]) {
  await build({
    entryPoints: [`${root}${entry}`],
    bundle: true,
    minify: true,
    format: 'iife',
    target: 'safari16',
    outfile: `${assets}${output}`,
    legalComments: 'none',
    logLevel: 'warning',
  });
  if (readFileSync(`${assets}${output}`, 'utf8').includes('</script')) {
    throw new Error(`${output} contains </script and cannot be inlined.`);
  }
}

let fonts = process.env.VFOX_PLUS_SYMBOLS;
const tool = fileURLToPath(new URL('../../shared/tools/icon_font.py', import.meta.url));
if (fonts === undefined || fonts === '') {
  fonts = `${root}node_modules/.cache/material-symbols`;
  execFileSync('python3', [tool, 'download', fonts], { stdio: 'inherit' });
}
execFileSync(
  'python3',
  [
    tool,
    'subset',
    '--fonts',
    fonts,
    '--output',
    `${assets}MaterialSymbolsRounded.woff2`,
    '--pin',
    'wght=400',
    '--pin',
    'GRAD=0',
    '--pin',
    'opsz=24',
  ],
  { stdio: 'inherit' },
);

const ALLOWED = new Set([
  'Apache-2.0',
  'MIT',
  'BSD-2-Clause',
  'BSD-3-Clause',
  'ISC',
  '0BSD',
  'Apache-2.0 OR MIT',
  'MIT OR Apache-2.0',
]);
const lock = JSON.parse(readFileSync(`${root}package-lock.json`, 'utf8'));
const sections = [];
for (const [path, entry] of Object.entries(lock.packages)) {
  if (path === '' || entry.dev === true || entry.devOptional === true || entry.optional === true) {
    continue;
  }
  const directory = `${root}${path}`;
  const manifest = JSON.parse(readFileSync(`${directory}/package.json`, 'utf8'));
  const license = typeof manifest.license === 'string' ? manifest.license : String(manifest.license?.type);
  if (!ALLOWED.has(license)) {
    throw new Error(`${manifest.name} is licensed under ${license}, which is not on the allowed list.`);
  }
  const file = readdirSync(directory).find((name) => /^(licen[cs]e|copying)(\.(md|txt))?$/i.test(name));
  const repository = typeof manifest.repository === 'string' ? manifest.repository : manifest.repository?.url;
  // A few packages of monorepos ship without the license text of their repository.
  const text =
    file === undefined
      ? `The package does not include the text of its license; see ${repository ?? manifest.homepage ?? manifest.name}.`
      : readFileSync(`${directory}/${file}`, 'utf8').trim();
  sections.push(`${manifest.name} ${manifest.version} (${license})\n\n${text}`);
}
if (!existsSync(`${fonts}/LICENSE`)) {
  throw new Error(`${fonts}/LICENSE is missing.`);
}
sections.push(
  `Material Symbols Rounded, Google LLC (Apache-2.0)\n\n${readFileSync(`${fonts}/LICENSE`, 'utf8').trim()}`,
);
sections.sort((a, b) => a.localeCompare(b));
writeFileSync(`${assets}npm-licenses.txt`, `${sections.join(`\n\n${'-'.repeat(72)}\n\n`)}\n`);
console.log(`Wrote the assets (${sections.length} licenses of packages and fonts)`);
