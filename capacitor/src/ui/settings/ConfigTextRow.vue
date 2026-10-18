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

<!--
  A key of the vfox configuration as a text field, saved with Enter or when
  the field loses the focus. Clearing the field resets the key.
-->
<script setup lang="ts">
import { IonInput, IonItem, IonLabel } from '@ionic/vue';
import { ref } from 'vue';

import { app } from '@web/ui/context';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import type { IconName } from '../icons';
import { t, watched } from '../reactivity';
import { saveConfig } from '../vfox';

const props = withDefaults(
  defineProps<{
    configKey: string;
    title: string;
    icon: IconName | null;
    hint: string;
    help?: string | null;
    /** Whether the value is a directory, with a button to choose one. */
    directory?: boolean;
  }>(),
  { help: null, directory: false },
);

const { model, host } = app();
const config = model.vfoxConfig;
/** What the user typed and has not saved yet. */
const draft = ref<string | null>(null);

function save(value: string): void {
  const typed = value.trim();
  const stored = config.valueFor(props.configKey) ?? '';
  draft.value = null;
  if (typed === stored) {
    return;
  }
  // If vfox rejects the value, the reload shows the stored value again.
  void saveConfig(() => (typed === '' ? config.reset(props.configKey) : config.set(props.configKey, typed)));
}

async function browse(): Promise<void> {
  const current = draft.value ?? config.valueFor(props.configKey) ?? '';
  const directory = await host.pickDirectory(current === '' ? undefined : current);
  if (directory !== null) {
    save(directory);
  }
}

function valueOf(event: Event): string {
  return (event as CustomEvent<{ value?: string | null }>).detail.value ?? '';
}
</script>

<template>
  <IonItem class="setting">
    <MaterialIcon v-if="icon !== null" slot="start" :name="icon" />
    <span v-else slot="start" class="icon-space"></span>
    <IonLabel>
      <div class="title type-body-large">{{ title }}</div>
      <div v-if="help !== null" class="supporting type-body-medium">{{ help }}</div>
    </IonLabel>
    <IonInput
      slot="end"
      class="config-field"
      fill="outline"
      :aria-label="title"
      :placeholder="hint"
      :value="draft ?? watched(config).valueFor(configKey) ?? ''"
      :disabled="watched(config).isSaving(configKey)"
      @ion-input="draft = valueOf($event)"
      @ion-change="save(valueOf($event))"
      @keydown.enter="save(($event.target as HTMLInputElement).value)"
    >
      <IconButton v-if="directory" slot="end" icon="folder" :label="t('actionBrowse')" @click="browse" />
    </IonInput>
  </IonItem>
</template>
