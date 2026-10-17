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

/// <reference types="vitest/config" />

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { defineConfig, type Plugin } from 'vite';

import { devHost } from './dev-host/plugin.ts';

const shared = fileURLToPath(new URL('../shared', import.meta.url));

/** Imports the ARB catalogues of shared/i18n as JSON modules, unchanged. */
function arb(): Plugin {
  return {
    name: 'vfox-plus-arb',
    load(id) {
      if (!id.endsWith('.arb')) {
        return null;
      }
      const catalogue = JSON.parse(readFileSync(id, 'utf8')) as unknown;
      return `export default ${JSON.stringify(catalogue)};`;
    },
  };
}

// Modes: `development` runs in a browser against the development host
// (dev-host/), `tauri` and `wails` build the front end of those shells.
// VFOX_PLUS_E2E=1 adds the end-to-end test runner (e2e/) to a build.
export default defineConfig(({ mode }) => ({
  base: './',
  plugins: [arb(), ...(mode === 'development' ? [devHost()] : [])],
  define: {
    __VFOX_PLUS_E2E__: JSON.stringify(process.env['VFOX_PLUS_E2E'] === '1'),
  },
  resolve: {
    alias: { '@shared': shared },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    fs: { allow: ['.', shared] },
  },
  build: {
    target: 'es2023',
    // The web views of Tauri and Wails load the files from disk; nothing is
    // gained from splitting them.
    cssCodeSplit: false,
    chunkSizeWarningLimit: 2048,
    reportCompressedSize: false,
  },
  test: {
    include: ['test/**/*.test.ts'],
    environment: 'node',
    coverage: {
      provider: 'v8',
      include: ['src/core/**', 'src/state/**'],
      reporter: ['text-summary', 'lcov'],
      thresholds: { lines: 85, functions: 85, statements: 85, branches: 80 },
    },
  },
}));
