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
  A friendly placeholder for empty, unavailable or failed content, with an
  expressive shape behind its icon. Buttons go into the default slot.
-->
<script setup lang="ts">
import type { IconName } from '../icons';
import type { ShapeName } from '../presentation';
import MaterialIcon from './MaterialIcon.vue';

withDefaults(
  defineProps<{
    heading: string;
    body?: string;
    icon?: IconName;
    shape?: ShapeName;
    /** Whether to use error colours. */
    error?: boolean;
  }>(),
  { body: '', icon: 'info', shape: '9-sided-cookie', error: false },
);
</script>

<template>
  <div class="status-host" :class="{ error }">
    <div class="status" role="status">
      <div class="shape" aria-hidden="true">
        <m3e-shape :name="shape" />
        <MaterialIcon :name="icon" filled />
      </div>
      <h2 class="type-title-large-emphasized">{{ heading }}</h2>
      <p v-if="body !== ''" class="type-body-medium">{{ body }}</p>
      <div v-if="$slots.default" class="actions"><slot /></div>
    </div>
  </div>
</template>

<style scoped>
.status-host {
  display: grid;
  place-items: center;
  padding: var(--gap-xxl);
  min-height: 100%;
}
.status {
  max-width: 440px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}
.shape {
  position: relative;
  width: 112px;
  height: 112px;
  display: grid;
  place-items: center;
  margin-bottom: var(--gap-xl);
}
m3e-shape {
  position: absolute;
  inset: 0;
  --m3e-shape-size: 112px;
  --m3e-shape-container-color: var(--md-sys-color-secondary-container);
}
.material-icon {
  position: relative;
  --icon-size: 48px;
  color: var(--md-sys-color-on-secondary-container);
}
.error m3e-shape {
  --m3e-shape-container-color: var(--md-sys-color-error-container);
}
.error .material-icon {
  color: var(--md-sys-color-on-error-container);
}
h2 {
  color: var(--md-sys-color-on-surface);
}
p {
  margin-top: var(--gap-s);
  color: var(--md-sys-color-on-surface-variant);
  white-space: pre-line;
  overflow-wrap: anywhere;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--gap-s);
  margin-top: var(--gap-xl);
}
</style>
