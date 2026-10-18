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

/**
 * The dialogs and menus of the application, as Ionic modals and popovers
 * with Vue components inside. The functions resolve with what the user chose.
 */

import type { ComponentRef } from '@ionic/core';
import { modalController, popoverController } from '@ionic/vue';

import type { IconName } from './icons';
import { dialogEnter, dialogLeave } from './transitions';
import ConfirmDialog from './dialogs/ConfirmDialog.vue';
import CustomPluginDialog from './dialogs/CustomPluginDialog.vue';
import { CONFIRM } from './dialogs/dialog';
import MenuList from './dialogs/MenuList.vue';
import ProjectDialog from './dialogs/ProjectDialog.vue';
import TextDialog from './dialogs/TextDialog.vue';

/** Shows `component` in a dialog and resolves with the value it was confirmed with. */
async function showDialog<T>(component: ComponentRef, componentProps: Record<string, unknown> = {}): Promise<T | null> {
  const modal = await modalController.create({
    component,
    componentProps,
    cssClass: 'dialog',
    enterAnimation: dialogEnter,
    leaveAnimation: dialogLeave,
  });
  await modal.present();
  const { data, role } = await modal.onDidDismiss<T>();
  return role === CONFIRM && data !== undefined ? data : null;
}

/** Asks the user to confirm an action. Resolves with whether they did. */
export async function confirmAction(options: {
  title: string;
  body: string;
  confirmLabel: string;
  icon?: IconName;
  destructive?: boolean;
}): Promise<boolean> {
  return (await showDialog<boolean>(ConfirmDialog, options)) === true;
}

/** Asks for the project directory in which to pin `name` `version`. */
export function askProjectDirectory(name: string, version: string): Promise<string | null> {
  return showDialog<string>(ProjectDialog, { name, version });
}

/** The source and optional alias of a plugin added from a URL or a path. */
export interface CustomPluginSource {
  readonly source: string;
  readonly alias: string | null;
}

/** Asks for the source of a plugin that is not in the registry. */
export function askCustomPlugin(): Promise<CustomPluginSource | null> {
  return showDialog<CustomPluginSource>(CustomPluginDialog);
}

/** Shows a long, monospaced text such as the open-source licenses. */
export async function showText(title: string, text: string): Promise<void> {
  await showDialog(TextDialog, { heading: title, text });
}

/** An item of a menu. */
export interface MenuItem {
  readonly id: string;
  readonly label: string;
  readonly icon: IconName;
  /** Whether the item destroys something, in the error colour. */
  readonly danger?: boolean;
}

/**
 * Shows a menu with `items` below the element that `event` happened on,
 * aligned to its end, and resolves with the identifier of the chosen item.
 */
export async function showMenu(event: Event, items: readonly MenuItem[]): Promise<string | null> {
  const popover = await popoverController.create({
    component: MenuList,
    componentProps: { items },
    event,
    side: 'bottom',
    alignment: 'end',
    arrow: false,
    // Material 3 menus have no scrim; a click outside still closes them.
    showBackdrop: false,
    cssClass: 'menu',
  });
  await popover.present();
  const { data } = await popover.onWillDismiss<string>();
  return data ?? null;
}
