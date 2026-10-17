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
  A Material 3 Expressive connected button group with single selection, on
  Ionic's segment (see styles/ionic.css). The selected button cannot be
  unselected by clicking it again.
-->
<script setup lang="ts" generic="T extends string">
import { IonLabel, IonSegment, IonSegmentButton, type SegmentCustomEvent } from '@ionic/vue';

import type { Choice } from './choice';
import MaterialIcon from './MaterialIcon.vue';

const props = defineProps<{
  choices: readonly Choice<T>[];
  selected: T;
}>();
const emit = defineEmits<{ choose: [value: T] }>();

function change(event: SegmentCustomEvent): void {
  const value = props.choices.find(([choice]) => choice === event.detail.value)?.[0];
  if (value !== undefined && value !== props.selected) {
    emit('choose', value);
  }
}
</script>

<template>
  <IonSegment class="choice-group" :value="selected" @ion-change="change">
    <IonSegmentButton v-for="[value, label, icon] in choices" :key="value" :value="value" layout="icon-start">
      <MaterialIcon v-if="icon !== null" :name="icon" />
      <IonLabel>{{ label }}</IonLabel>
    </IonSegmentButton>
  </IonSegment>
</template>
