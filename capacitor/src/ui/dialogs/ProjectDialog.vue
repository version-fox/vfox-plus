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

<!-- Asks for the project directory in which to pin a version; confirmed with its path. -->
<script setup lang="ts">
import { IonButton, IonInput } from '@ionic/vue';
import { onMounted, ref } from 'vue';

import { app } from '@web/ui/context';
import { t } from '../reactivity';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import { closeDialog, whenPresented } from './dialog';

defineProps<{ name: string; version: string }>();

const path = ref('');
const error = ref('');
const root = ref<HTMLElement | null>(null);
const field = ref<{ $el: HTMLIonInputElement } | null>(null);

async function browse(): Promise<void> {
  const { host, model } = app();
  const current = path.value.trim();
  const directory = await host.pickDirectory(current === '' ? model.cli.launchConfig.baseEnvironment['HOME'] : current);
  if (directory !== null) {
    path.value = directory;
    error.value = '';
  }
}

async function submit(): Promise<void> {
  const directory = path.value.trim();
  if (directory === '' || !(await app().host.files.isDirectory(directory))) {
    error.value = t('useProjectInvalid');
    return;
  }
  closeDialog(directory);
}

onMounted(() => {
  whenPresented(root.value, () => {
    void field.value?.$el.setFocus();
  });
});
</script>

<template>
  <div ref="root" class="dialog-content">
    <div class="dialog-header">
      <MaterialIcon name="projectScope" />
      <h2 class="type-headline-small">{{ t('useProjectTitle', { name, version }) }}</h2>
    </div>
    <p class="dialog-body type-body-medium">{{ t('useProjectBody') }}</p>
    <div class="field-row">
      <IonInput
        ref="field"
        v-model="path"
        fill="outline"
        label-placement="floating"
        :label="t('useProjectDirectory')"
        :error-text="error"
        :class="{ 'ion-invalid': error !== '', 'ion-touched': error !== '' }"
        @ion-input="error = ''"
        @keydown.enter="submit"
      />
      <IconButton icon="folder" variant="tonal" :label="t('actionBrowse')" @click="browse" />
    </div>
    <div class="dialog-actions">
      <IonButton fill="clear" @click="closeDialog()">{{ t('actionCancel') }}</IonButton>
      <IonButton @click="submit">{{ t('useProjectConfirm') }}</IonButton>
    </div>
  </div>
</template>
