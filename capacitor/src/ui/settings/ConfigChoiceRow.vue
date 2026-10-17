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

<!-- A key of the vfox configuration with a few possible values, as a menu. -->
<script setup lang="ts">
import { IonItem, IonLabel, IonSelect, IonSelectOption } from '@ionic/vue';

import { app } from '@web/ui/context';
import { watched } from '../reactivity';
import { saveConfig } from '../vfox';

const props = defineProps<{ configKey: string; title: string; choices: readonly (readonly [string, string])[] }>();
const model = app().model;
const config = () => watched(model.vfoxConfig);

function change(selection: unknown): void {
  if (typeof selection === 'string' && selection !== '' && selection !== model.vfoxConfig.valueFor(props.configKey)) {
    void saveConfig(() => model.vfoxConfig.set(props.configKey, selection));
  }
}
</script>

<template>
  <IonItem class="setting">
    <span slot="start" class="icon-space"></span>
    <IonLabel>
      <div class="title type-body-large">{{ title }}</div>
    </IonLabel>
    <IonSelect
      slot="end"
      class="config-field"
      fill="outline"
      interface="popover"
      :interface-options="{ cssClass: 'menu' }"
      :aria-label="title"
      :value="config().valueFor(configKey) ?? ''"
      :disabled="config().isSaving(configKey)"
      @ion-change="change($event.detail.value)"
    >
      <IonSelectOption v-for="[choice, label] in choices" :key="choice" :value="choice">{{ label }}</IonSelectOption>
    </IonSelect>
  </IonItem>
</template>
