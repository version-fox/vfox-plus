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
import vue from '@vitejs/plugin-vue';
import { defineConfig, type Plugin } from 'vite';

import { devHost } from '../web/dev-host/plugin.ts';

const shared = fileURLToPath(new URL('../shared', import.meta.url));
const web = fileURLToPath(new URL('../web/src', import.meta.url));

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

// The renderer of the Electron shell. `development` runs it in a browser
// against the development host of ../web (dev-host/) instead. VFOX_PLUS_E2E=1
// adds the end-to-end test runner of ../web/src/e2e to a build.
export default defineConfig(({ mode }) => ({
  base: './',
  plugins: [
    // The elements of @m3e/web are web components, not Vue components.
    vue({ template: { compilerOptions: { isCustomElement: (tag) => tag.startsWith('m3e-') } } }),
    arb(),
    ...(mode === 'development' ? [devHost()] : []),
  ],
  define: {
    __VFOX_PLUS_E2E__: JSON.stringify(process.env['VFOX_PLUS_E2E'] === '1'),
  },
  resolve: {
    alias: { '@shared': shared, '@web': web },
    // The modules of ../web import their packages from here.
    dedupe: ['@material/material-color-utilities'],
  },
  server: {
    host: '127.0.0.1',
    port: 5174,
    strictPort: true,
    fs: { allow: ['.', shared, web] },
  },
  build: {
    outDir: 'dist/app/renderer',
    target: 'es2023',
    // Electron loads the files from disk; nothing is gained from splitting them.
    cssCodeSplit: false,
    chunkSizeWarningLimit: 4096,
    reportCompressedSize: false,
    rolldownOptions: {
      treeshake: {
        // @ionic/vue imports every component of Ionic. Their modules only
        // create the classes of the elements, which defineCustomElement()
        // registers when a component is used, but they are not marked free
        // of side effects, so the unused ones would be bundled too.
        moduleSideEffects: (id) => !/[\\/]@ionic[\\/]core[\\/]components[\\/]ion-[a-z-]+\.js$/.test(id),
      },
    },
  },
  test: {
    include: ['test/**/*.test.ts'],
    environment: 'node',
    // Its modules import each other without file extensions, which Node.js does not resolve.
    server: { deps: { inline: ['@material/material-color-utilities'] } },
    coverage: {
      provider: 'v8',
      include: ['electron/**/*.ts', 'src/**/*.ts'],
      // The entry points only wire the parts together; the end-to-end tests run them.
      exclude: ['electron/main.ts', 'electron/preload.ts', 'src/main.ts', '**/*.d.ts'],
      reporter: ['text-summary', 'lcov'],
      thresholds: { lines: 85, functions: 85, statements: 85, branches: 80 },
    },
  },
}));
