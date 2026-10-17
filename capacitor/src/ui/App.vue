<!--
  Copyright 2026 Han Li and contributors

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
-->

<!-- The root: theme, language, the shell and the task snack bars. -->
<script setup lang="ts">
import { IonApp } from '@ionic/vue';
import { onBeforeUnmount } from 'vue';

import type { VfoxTask } from '@web/state/task-queue';
import { app } from '@web/ui/context';
import { prefersReducedMotion } from '@web/ui/motion';
import { t, taskTitle } from './reactivity';
import AppShell from './shell/AppShell.vue';
import { snackbar } from './snackbar';
import { applyTheme } from './theme';

const DARK_QUERY = '(prefers-color-scheme: dark)';
const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';

const { model, host, locale, shell } = app();
const dark = matchMedia(DARK_QUERY);
const reducedMotion = matchMedia(REDUCED_MOTION_QUERY);
let accent: number | null = null;

function applySettings(): void {
  const settings = model.settings.settings;
  const seed = settings.useSystemColor && accent !== null ? accent : settings.seedColor;
  const isDark = settings.theme === 'dark' || (settings.theme === 'system' && dark.matches);
  applyTheme(seed, isDark, prefersReducedMotion());
  locale.apply(settings.language);
  document.documentElement.lang = locale.l10n.languageTag;
  document.title = t('appTitle');
}

/** Tells the user how a task ended, unless they are watching the task list. */
function announce(task: VfoxTask): void {
  if (task.status === 'cancelled') {
    return;
  }
  const title = taskTitle(task.command);
  if (task.status === 'failed') {
    // Failures stay until dismissed, so that they can be read in full.
    snackbar(`${t('taskFailed', { title })}\n${task.errorMessage ?? ''}`.trim(), {
      action: t('taskView'),
      onAction: () => {
        shell.go('tasks');
      },
      persistent: true,
    });
  } else if (shell.destination !== 'tasks') {
    snackbar(t('taskSucceeded', { title }));
  }
}

const stops = [model.settings.addListener(applySettings), model.tasks.onFinished(announce)];
dark.addEventListener('change', applySettings);
reducedMotion.addEventListener('change', applySettings);
applySettings();
void host.accentColor().then(
  (color) => {
    accent = color;
    applySettings();
  },
  () => undefined,
);

onBeforeUnmount(() => {
  stops.forEach((stop) => {
    stop();
  });
  dark.removeEventListener('change', applySettings);
  reducedMotion.removeEventListener('change', applySettings);
});
</script>

<template>
  <IonApp>
    <AppShell />
  </IonApp>
</template>
