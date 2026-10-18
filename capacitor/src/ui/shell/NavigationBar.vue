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
  The Material 3 navigation bar at the bottom of compact windows. Ionic's tab
  bar would select its first tab whenever it updates without a router, so the
  items are buttons with Ionic's ripple.
-->
<script setup lang="ts">
import { IonBadge, IonRippleEffect } from '@ionic/vue';

import { app, DESTINATIONS } from '@web/ui/context';
import MaterialIcon from '../components/MaterialIcon.vue';
import { NAVIGATION } from '../navigation';
import { t, watched } from '../reactivity';

defineProps<{ active: number }>();
const shell = app().shell;
/** The visible destination; the navigation follows it. */
const destination = (): string => watched(shell).destination;
</script>

<template>
  <nav class="bar">
    <button
      v-for="item in DESTINATIONS"
      :key="item"
      class="nav-item ion-activatable"
      :class="{ selected: destination() === item }"
      :aria-label="t(NAVIGATION[item].label)"
      :aria-current="destination() === item ? 'page' : undefined"
      @click="shell.go(item)"
    >
      <span class="indicator">
        <MaterialIcon :name="NAVIGATION[item].icon" :filled="destination() === item" />
        <IonBadge v-if="item === 'tasks' && active > 0">{{ active }}</IonBadge>
        <IonRippleEffect />
      </span>
      <span class="nav-label">{{ t(NAVIGATION[item].label) }}</span>
    </button>
  </nav>
</template>

<style scoped>
.bar {
  flex: none;
  display: flex;
  height: 80px;
  background: var(--md-sys-color-surface-container);
}
.nav-item {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--gap-xs);
  padding: var(--gap-m) 0 var(--gap-l);
  border: none;
  background: none;
  font: inherit;
  color: var(--md-sys-color-on-surface-variant);
  cursor: pointer;
}
.nav-item:focus-visible {
  outline: 3px solid var(--md-sys-color-secondary);
  outline-offset: -3px;
}
.indicator {
  position: relative;
  overflow: hidden;
  display: grid;
  place-items: center;
  width: 64px;
  height: 32px;
  border-radius: 16px;
  transition: background-color var(--md-sys-motion-spring-fast-effects);
}
.nav-item:hover .indicator {
  background: color-mix(in srgb, var(--md-sys-color-on-surface) 8%, transparent);
}
.selected .indicator {
  background: var(--md-sys-color-secondary-container);
  color: var(--md-sys-color-on-secondary-container);
}
.nav-label {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 500;
  line-height: 16px;
  letter-spacing: 0.5px;
}
.selected .nav-label {
  color: var(--md-sys-color-on-surface);
  font-weight: 700;
}
ion-badge {
  position: absolute;
  top: 2px;
  inset-inline-start: 34px;
}
</style>
