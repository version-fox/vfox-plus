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
  An icon button with a tooltip, which also serves as its accessible name.
  Every icon-only button of the application is one.
-->
<script setup lang="ts">
import { IonButton } from '@ionic/vue';
import { useId } from 'vue';

import type { IconName } from '../icons';
import MaterialIcon from './MaterialIcon.vue';

withDefaults(
  defineProps<{
    icon: IconName;
    label: string;
    variant?: 'standard' | 'tonal';
    disabled?: boolean;
    filled?: boolean;
  }>(),
  { variant: 'standard', disabled: false, filled: false },
);

// The tooltip finds the button by its identifier.
const id = useId();
</script>

<template>
  <span class="icon-button-host">
    <IonButton
      :id="id"
      class="icon-button"
      :class="variant"
      shape="round"
      :fill="variant === 'tonal' ? 'solid' : 'clear'"
      :color="variant === 'tonal' ? 'tonal' : undefined"
      :aria-label="label"
      :disabled="disabled"
    >
      <MaterialIcon slot="icon-only" :name="icon" :filled="filled" />
    </IonButton>
    <m3e-tooltip :for="id" show-delay="500">{{ label }}</m3e-tooltip>
  </span>
</template>
