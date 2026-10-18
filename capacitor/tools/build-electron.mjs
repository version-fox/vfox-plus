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

// Bundles the main process and the preload script of electron/ into
// <app>/main.cjs and <app>/preload.cjs, next to the page that `vite build`
// wrote to <app>/renderer, and writes the package.json that Electron reads.
// <app> is then what `electron <app>` runs and tools/stage.mjs packages.
//
//   node tools/build-electron.mjs dist/app

import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { build } from 'esbuild';

const root = fileURLToPath(new URL('..', import.meta.url));
const output = process.argv[2] ?? 'dist/app';
const manifest = JSON.parse(readFileSync(`${root}package.json`, 'utf8'));

await build({
  absWorkingDir: root,
  entryPoints: { main: 'electron/main.ts', preload: 'electron/preload.ts' },
  outdir: output,
  outExtension: { '.js': '.cjs' },
  bundle: true,
  platform: 'node',
  format: 'cjs',
  target: 'node24',
  external: ['electron'],
  legalComments: 'none',
  logLevel: 'warning',
});
writeFileSync(
  `${output}/package.json`,
  `${JSON.stringify(
    {
      name: 'vfox-plus',
      version: manifest.version,
      description: manifest.description,
      license: manifest.license,
      main: 'main.cjs',
      // Wayland takes the application ID from the desktop entry named here.
      desktopName: 'dev.vfox.VfoxPlus.desktop',
    },
    null,
    2,
  )}\n`,
);
console.log(`Built the Electron shell in ${output}`);
