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

// Collects the licenses of everything the front end ships into
// public/licenses.txt, which the about section shows: the packages that
// package-lock.json lists as runtime dependencies, and the Material Symbols
// font. Fails when a package has no license file or a license that is not
// compatible with the Apache License 2.0 (see docs/architecture.md).
//
//   npm run assets

import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('..', import.meta.url));
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
const font = `${root}public/fonts/MaterialSymbolsRounded-LICENSE.txt`;
if (!existsSync(font)) {
  throw new Error('Run `npm run icons` first.');
}
sections.push(`Material Symbols Rounded, Google LLC (Apache-2.0)\n\n${readFileSync(font, 'utf8').trim()}`);
sections.sort((a, b) => a.localeCompare(b));
writeFileSync(`${root}public/licenses.txt`, `${sections.join(`\n\n${'-'.repeat(72)}\n\n`)}\n`);
console.log(`Wrote public/licenses.txt (${sections.length} licenses)`);
