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

/** Snack bars, through the snack bar service of `@m3e/web`. */

import { M3eSnackbar } from '@m3e/web/snackbar';

import { t } from './context';

/** How a snack bar behaves. */
export interface SnackbarOptions {
  /** The label of the action button. */
  readonly action?: string;
  /** Called when the action is chosen. */
  readonly onAction?: () => void;
  /** Whether the snack bar stays until it is dismissed. */
  readonly persistent?: boolean;
}

/** Shows `message` at the bottom of the window, replacing the current snack bar. */
export function snackbar(message: string, options: SnackbarOptions = {}): void {
  const settings = {
    // A duration of zero keeps the snack bar open.
    duration: options.persistent === true ? 0 : 4000,
    closeLabel: t('actionClose'),
    ...(options.onAction === undefined ? {} : { actionCallback: options.onAction }),
  };
  if (options.action === undefined) {
    M3eSnackbar.open(message, options.persistent === true, settings);
  } else {
    M3eSnackbar.open(message, options.action, options.persistent === true, settings);
  }
}
