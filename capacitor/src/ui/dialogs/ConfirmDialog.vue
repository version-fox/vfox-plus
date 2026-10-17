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

<!-- Asks the user to confirm an action; confirmed with `true`. -->
<script setup lang="ts">
import { IonButton } from '@ionic/vue';
import { onMounted, ref } from 'vue';

import { t } from '../reactivity';
import MaterialIcon from '../components/MaterialIcon.vue';
import type { IconName } from '../icons';
import { closeDialog, whenPresented } from './dialog';

withDefaults(
  defineProps<{
    title: string;
    body: string;
    confirmLabel: string;
    icon?: IconName;
    destructive?: boolean;
  }>(),
  { icon: undefined, destructive: false },
);

const root = ref<HTMLElement | null>(null);
const confirm = ref<{ $el: HTMLElement } | null>(null);
onMounted(() => {
  whenPresented(root.value, () => {
    confirm.value?.$el.focus();
  });
});
</script>

<template>
  <div ref="root" class="dialog-content">
    <div class="dialog-header" :class="{ destructive }">
      <MaterialIcon v-if="icon !== undefined" :name="icon" />
      <h2 class="type-headline-small">{{ title }}</h2>
    </div>
    <p class="dialog-body type-body-medium">{{ body }}</p>
    <div class="dialog-actions">
      <IonButton fill="clear" @click="closeDialog()">{{ t('actionCancel') }}</IonButton>
      <IonButton ref="confirm" :color="destructive ? 'danger' : 'primary'" @click="closeDialog(true)">
        {{ confirmLabel }}
      </IonButton>
    </div>
  </div>
</template>
