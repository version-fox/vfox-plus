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

/** Actions on the vfox installation that more than one page offers. */

import { app, describeError, t } from '@web/ui/context';
import { snackbar } from './snackbar';

/** Saves a vfox setting with `save` and reports the result in a snack bar. */
export async function saveConfig(save: () => Promise<void>): Promise<void> {
  try {
    await save();
    snackbar(t('settingsSaved'));
  } catch (error) {
    snackbar(describeError(error));
  }
}

/** Lets the user choose the vfox executable and stores the choice. */
export async function locateVfox(): Promise<void> {
  const { host, model } = app();
  const path = await host.pickFile('/usr/local/bin');
  if (path !== null) {
    await model.settings.update((settings) => ({ ...settings, vfoxExecutable: path }));
  }
}
