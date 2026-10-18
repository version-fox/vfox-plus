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

<!-- A boolean key of the vfox configuration, as a switch. -->
<script setup lang="ts">
import { IonItem, IonToggle } from '@ionic/vue';

import { app } from '@web/ui/context';
import MaterialIcon from '../components/MaterialIcon.vue';
import type { IconName } from '../icons';
import { watched } from '../reactivity';
import { saveConfig } from '../vfox';

const props = defineProps<{ configKey: string; title: string; icon: IconName; help?: string }>();
const model = app().model;
const config = () => watched(model.vfoxConfig);

function change(checked: boolean): void {
  if (!model.vfoxConfig.isSaving(props.configKey)) {
    void saveConfig(() => model.vfoxConfig.set(props.configKey, String(checked)));
  }
}
</script>

<template>
  <IonItem class="setting">
    <MaterialIcon slot="start" :name="icon" />
    <IonToggle
      justify="space-between"
      :checked="config().valueFor(configKey) === 'true'"
      :disabled="config().isSaving(configKey)"
      @ion-change="change($event.detail.checked)"
    >
      <span class="title type-body-large">{{ title }}</span>
      <span v-if="help !== undefined" class="supporting type-body-medium">{{ help }}</span>
    </IonToggle>
  </IonItem>
</template>
