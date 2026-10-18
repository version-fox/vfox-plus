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

/** Snack bars, as Ionic toasts in the style of Material 3 (see styles/ionic.css). */

import { toastController, type ToastButton } from '@ionic/vue';

import { t } from '@web/ui/context';
import { symbolOf } from './icons';

/** How a snack bar behaves. */
export interface SnackbarOptions {
  /** The label of the action button. */
  readonly action?: string;
  /** Called when the action is chosen. */
  readonly onAction?: () => void;
  /** Whether the snack bar stays until it is dismissed. */
  readonly persistent?: boolean;
}

let current: HTMLIonToastElement | null = null;

/** The buttons of a snack bar with `options`. */
export function snackbarButtons(options: SnackbarOptions): ToastButton[] {
  const buttons: ToastButton[] = [];
  if (options.action !== undefined) {
    buttons.push({
      text: options.action,
      handler: () => {
        options.onAction?.();
      },
    });
  }
  if (options.persistent === true) {
    // The text is the ligature of the close icon, in the icon font.
    buttons.push({ text: symbolOf('close'), role: 'cancel', htmlAttributes: { 'aria-label': t('actionClose') } });
  }
  return buttons;
}

async function show(message: string, options: SnackbarOptions): Promise<void> {
  const previous = current;
  const toast = await toastController.create({
    message,
    // A duration of zero keeps the snack bar open.
    duration: options.persistent === true ? 0 : 4000,
    position: 'bottom',
    cssClass: 'snackbar',
    buttons: snackbarButtons(options),
  });
  current = toast;
  void toast.onDidDismiss().then(() => {
    if (current === toast) {
      current = null;
    }
  });
  await previous?.dismiss();
  await toast.present();
}

/** Shows `message` at the bottom of the window, replacing the current snack bar. */
export function snackbar(message: string, options: SnackbarOptions = {}): void {
  void show(message, options);
}
