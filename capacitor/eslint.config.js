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

// Static analysis of the Capacitor implementation. CI fails on any warning:
//
//   npm run lint

import js from '@eslint/js';
import vue from 'eslint-plugin-vue';
import globals from 'globals';
import tseslint from 'typescript-eslint';
import vueParser from 'vue-eslint-parser';

/** The rules of eslint-plugin-vue about layout, which Prettier takes care of. */
const LAYOUT = [
  'vue/first-attribute-linebreak',
  'vue/html-closing-bracket-newline',
  'vue/html-closing-bracket-spacing',
  'vue/html-end-tags',
  'vue/html-indent',
  'vue/html-quotes',
  'vue/html-self-closing',
  'vue/max-attributes-per-line',
  'vue/multiline-html-element-content-newline',
  'vue/mustache-interpolation-spacing',
  'vue/no-multi-spaces',
  'vue/no-spaces-around-equal-signs-in-attribute',
  'vue/singleline-html-element-content-newline',
];

export default tseslint.config(
  { ignores: ['node_modules/', 'dist/', 'coverage/', 'public/'] },
  js.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  ...vue.configs['flat/recommended'],
  {
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: tseslint.parser,
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
        extraFileExtensions: ['.vue'],
        sourceType: 'module',
      },
      globals: { ...globals.browser, ...globals.node },
    },
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/restrict-template-expressions': ['error', { allowNumber: true }],
      '@typescript-eslint/no-confusing-void-expression': ['error', { ignoreArrowShorthand: true }],
      // Environment variables read better as env['PATH'].
      '@typescript-eslint/dot-notation': 'off',
      // TypeScript knows the globals, the types of Ionic's elements among them.
      'no-undef': 'off',
      eqeqeq: 'error',
      'no-console': ['error', { allow: ['error', 'warn'] }],
      curly: 'error',
      ...Object.fromEntries(LAYOUT.map((rule) => [rule, 'off'])),
      // Elements are put into the named slots of Ionic's web components with
      // the slot attribute, which is not the deprecated slot syntax of Vue 2.
      'vue/no-deprecated-slot-attribute': 'off',
      // The objects of the model that components get are meant to be changed.
      'vue/no-mutating-props': ['error', { shallowOnly: true }],
    },
  },
  {
    // The user interface knows the model only through the reactivity of src/ui/reactivity.ts.
    files: ['src/ui/**/*.vue'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@web/ui/context',
              importNames: ['t', 'taskTitle'],
              message: 'Use t() and taskTitle() of src/ui/reactivity.ts, which follow the language.',
            },
          ],
        },
      ],
    },
  },
  {
    files: ['**/*.js', '**/*.mjs'],
    ...tseslint.configs.disableTypeChecked,
  },
  {
    files: ['tools/**'],
    rules: { 'no-console': 'off' },
  },
);
