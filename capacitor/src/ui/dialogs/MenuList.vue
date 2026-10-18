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

<!-- The items of a menu, in a popover that closes with the chosen item. -->
<script setup lang="ts">
import { IonItem, IonLabel, IonList, popoverController } from '@ionic/vue';

import MaterialIcon from '../components/MaterialIcon.vue';
import type { MenuItem } from '../overlays';

defineProps<{ items: readonly MenuItem[] }>();

function choose(id: string): void {
  void popoverController.dismiss(id);
}
</script>

<template>
  <IonList class="menu-list" lines="none">
    <IonItem
      v-for="item in items"
      :key="item.id"
      button
      :detail="false"
      :class="{ danger: item.danger === true }"
      @click="choose(item.id)"
    >
      <MaterialIcon slot="start" :name="item.icon" />
      <IonLabel>{{ item.label }}</IonLabel>
    </IonItem>
  </IonList>
</template>
