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

// Static analysis of the web front end. CI fails on any warning:
//
//   npm run lint

import js from '@eslint/js';
import lit from 'eslint-plugin-lit';
import wc from 'eslint-plugin-wc';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['node_modules/', 'dist/', 'coverage/', 'public/'] },
  js.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  {
    languageOptions: {
      parserOptions: { projectService: true, tsconfigRootDir: import.meta.dirname },
      globals: { ...globals.browser, ...globals.node },
    },
    rules: {
      // The layers only depend downwards; see docs/architecture.md.
      'no-restricted-imports': 'off',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/restrict-template-expressions': ['error', { allowNumber: true }],
      '@typescript-eslint/no-confusing-void-expression': ['error', { ignoreArrowShorthand: true }],
      '@typescript-eslint/class-literal-property-style': 'off',
      // Environment variables read better as env['PATH'].
      '@typescript-eslint/dot-notation': 'off',
      eqeqeq: 'error',
      'no-console': ['error', { allow: ['error', 'warn'] }],
      curly: 'error',
    },
  },
  {
    files: ['src/core/**/*.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        { patterns: [{ group: ['**/state/**', '**/ui/**', 'lit', 'lit/*', '@m3e/*'], message: 'core has no UI.' }] },
      ],
    },
  },
  {
    files: ['src/state/**/*.ts'],
    rules: {
      'no-restricted-imports': [
        'error',
        { patterns: [{ group: ['**/ui/**', 'lit', 'lit/*', '@m3e/*'], message: 'state does not know the UI.' }] },
      ],
    },
  },
  {
    files: ['src/ui/**/*.ts'],
    ...lit.configs['flat/recommended'],
    ...wc.configs['flat/recommended'],
    plugins: { lit, wc },
    rules: {
      ...lit.configs['flat/recommended'].rules,
      ...wc.configs['flat/recommended'].rules,
      'wc/guard-super-call': 'off',
      // Lit calls the event listeners of templates with the element as `this`.
      '@typescript-eslint/unbound-method': 'off',
    },
  },
  {
    files: ['**/*.js', '**/*.mjs'],
    ...tseslint.configs.disableTypeChecked,
  },
  {
    files: ['tools/**', 'e2e/**'],
    rules: { 'no-console': 'off' },
  },
);
