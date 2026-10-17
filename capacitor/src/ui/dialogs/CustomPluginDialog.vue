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
  Asks for the source of a plugin that is not in the registry, and an
  optional name; confirmed with both.
-->
<script setup lang="ts">
import { IonButton, IonInput } from '@ionic/vue';
import { onMounted, ref } from 'vue';

import { isValidPluginName } from '@web/core/vfox/cli';
import { app } from '@web/ui/context';
import { t } from '../reactivity';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import type { CustomPluginSource } from '../overlays';
import { closeDialog, whenPresented } from './dialog';

const source = ref('');
const alias = ref('');
const sourceError = ref('');
const aliasError = ref('');
const root = ref<HTMLElement | null>(null);
const field = ref<{ $el: HTMLIonInputElement } | null>(null);

async function browse(): Promise<void> {
  const { host, model } = app();
  const file = await host.pickFile(model.cli.launchConfig.baseEnvironment['HOME'], {
    label: t('pluginsCustomFileType'),
    extensions: ['zip', 'lua'],
  });
  if (file !== null) {
    source.value = file;
    sourceError.value = '';
  }
}

function submit(): void {
  const chosen = source.value.trim();
  const name = alias.value.trim();
  sourceError.value = chosen === '' ? t('pluginsCustomRequired') : '';
  aliasError.value = name !== '' && !isValidPluginName(name) ? t('pluginsCustomInvalidName') : '';
  if (sourceError.value === '' && aliasError.value === '') {
    closeDialog({ source: chosen, alias: name === '' ? null : name } satisfies CustomPluginSource);
  }
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
      <MaterialIcon name="plugins" />
      <h2 class="type-headline-small">{{ t('pluginsCustomTitle') }}</h2>
    </div>
    <div class="field-row">
      <IonInput
        ref="field"
        v-model="source"
        fill="outline"
        label-placement="floating"
        :label="t('pluginsCustomSource')"
        :helper-text="t('pluginsCustomSourceHelp')"
        :error-text="sourceError"
        :class="{ 'ion-invalid': sourceError !== '', 'ion-touched': sourceError !== '' }"
        @ion-input="sourceError = ''"
        @keydown.enter="submit"
      />
      <IconButton icon="folder" variant="tonal" :label="t('actionBrowse')" @click="browse" />
    </div>
    <div class="field-row">
      <IonInput
        v-model="alias"
        fill="outline"
        label-placement="floating"
        :label="t('pluginsCustomAlias')"
        :helper-text="t('pluginsCustomAliasHelp')"
        :error-text="aliasError"
        :class="{ 'ion-invalid': aliasError !== '', 'ion-touched': aliasError !== '' }"
        @ion-input="aliasError = ''"
        @keydown.enter="submit"
      />
    </div>
    <div class="dialog-actions">
      <IonButton fill="clear" @click="closeDialog()">{{ t('actionCancel') }}</IonButton>
      <IonButton @click="submit">{{ t('pluginsCustomConfirm') }}</IonButton>
    </div>
  </div>
</template>
