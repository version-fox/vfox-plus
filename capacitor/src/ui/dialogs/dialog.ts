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

/** What the components inside the dialogs of ../overlays.ts use. */

import { modalController } from '@ionic/vue';

/** The role with which a dialog closes when the user confirms it. */
export const CONFIRM = 'confirm';

/** Closes the topmost dialog with `value`, as confirmed, or as dismissed without one. */
export function closeDialog(value?: unknown): void {
  void modalController.dismiss(value, value === undefined ? 'cancel' : CONFIRM);
}

/**
 * Calls `focus` once the dialog around `element` has been presented, when its
 * fields can take the focus.
 */
export function whenPresented(element: Element | null, focus: () => void): void {
  element?.closest('ion-modal')?.addEventListener('ionModalDidPresent', focus, { once: true });
}
