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

// Assembles the release of vfox+ from the Electron runtime of the electron
// package (`npm run electron` downloads it) and the application that
// `npm run build` wrote to dist/app, for shared/packaging/linux/build-deb.sh:
// the electron executable becomes vfox-plus, the application goes to
// resources/app, and only the English and Chinese locales of Chromium stay.
//
//   node tools/stage.mjs [application] [stage]

import { chmodSync, cpSync, existsSync, readdirSync, renameSync, rmSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname } from 'node:path';

const application = process.argv[2] ?? 'dist/app';
const stage = process.argv[3] ?? 'dist/stage';
const electron = `${dirname(createRequire(import.meta.url).resolve('electron/package.json'))}/dist`;
const LOCALES = new Set(['en-US.pak', 'zh-CN.pak']);

if (!existsSync(`${electron}/electron`)) {
  throw new Error('The Electron runtime is missing; run `npm run electron` first.');
}
if (!existsSync(`${application}/main.cjs`)) {
  throw new Error(`${application} has no application; run \`npm run build\` first.`);
}
rmSync(stage, { recursive: true, force: true });
cpSync(electron, stage, { recursive: true, verbatimSymlinks: true });
renameSync(`${stage}/electron`, `${stage}/vfox-plus`);
// The application that Electron shows when it is started without one.
rmSync(`${stage}/resources/default_app.asar`);
for (const locale of readdirSync(`${stage}/locales`)) {
  if (!LOCALES.has(locale)) {
    rmSync(`${stage}/locales/${locale}`);
  }
}
cpSync(application, `${stage}/resources/app`, { recursive: true });
// Chromium loads its libraries; they need not be executable.
for (const file of readdirSync(stage)) {
  if (/\.so(\.\d+)*$/.test(file)) {
    chmodSync(`${stage}/${file}`, 0o644);
  }
}
console.log(`Staged vfox+ in ${stage}`);
