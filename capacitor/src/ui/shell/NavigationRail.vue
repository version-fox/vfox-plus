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
  The Material 3 Expressive navigation rail of wider windows: collapsed, with
  the labels below the icons, or expanded, with the labels beside them and an
  extended floating action button. Ionic has no navigation rail; the items
  are buttons with Ionic's ripple.
-->
<script setup lang="ts">
import { IonBadge, IonRippleEffect } from '@ionic/vue';

import { app, DESTINATIONS } from '@web/ui/context';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import { NAVIGATION } from '../navigation';
import { t, watched } from '../reactivity';

defineProps<{ expanded: boolean; ready: boolean; active: number }>();
defineEmits<{ toggle: [] }>();
const shell = app().shell;
/** The visible destination; the navigation follows it. */
const destination = (): string => watched(shell).destination;
</script>

<template>
  <nav class="rail" :class="{ expanded }">
    <IconButton
      class="menu-toggle"
      :icon="expanded ? 'menuOpen' : 'menu'"
      :label="expanded ? t('navCollapse') : t('navExpand')"
      @click="$emit('toggle')"
    />
    <button v-if="ready" class="fab ion-activatable" :aria-label="t('navAddPlugin')" @click="shell.go('plugins')">
      <MaterialIcon name="add" />
      <span class="fab-label type-label-large">{{ t('navAddPlugin') }}</span>
      <IonRippleEffect />
    </button>
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
.rail {
  flex: none;
  width: 96px;
  height: 100%;
  overflow: hidden auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0 var(--gap-l);
  transition: width var(--md-sys-motion-spring-default-spatial);
}
.rail.expanded {
  width: 240px;
  align-items: stretch;
  padding-inline: var(--gap-m);
}
.menu-toggle {
  flex: none;
  margin-bottom: var(--gap-m);
}
.expanded .menu-toggle {
  align-self: flex-start;
  margin-inline-start: var(--gap-s);
}
button {
  font: inherit;
  color: inherit;
  border: none;
  background: none;
  padding: 0;
  cursor: pointer;
}
button:focus-visible {
  outline: 3px solid var(--md-sys-color-secondary);
  outline-offset: 2px;
}
.fab {
  position: relative;
  overflow: hidden;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--gap-m);
  width: 56px;
  height: 56px;
  margin-bottom: 28px;
  border-radius: var(--corner-large);
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
  box-shadow:
    0 1px 3px 1px rgb(0 0 0 / 15%),
    0 1px 2px rgb(0 0 0 / 30%);
  transition:
    width var(--md-sys-motion-spring-default-spatial),
    border-radius var(--md-sys-motion-spring-fast-spatial);
}
.fab:active {
  border-radius: var(--corner-medium);
}
.fab-label {
  display: none;
  white-space: nowrap;
}
.expanded .fab {
  margin-bottom: var(--gap-xl);
  align-self: flex-start;
  justify-content: flex-start;
  width: auto;
  padding-inline: var(--gap-l) var(--gap-xl);
}
.expanded .fab-label {
  display: inline;
}
.nav-item {
  position: relative;
  flex: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--gap-xs);
  width: 100%;
  min-height: 64px;
  margin-bottom: var(--gap-xs);
  color: var(--md-sys-color-on-surface-variant);
}
.indicator {
  position: relative;
  overflow: hidden;
  display: grid;
  place-items: center;
  width: 56px;
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
  font-size: 12px;
  font-weight: 500;
  line-height: 16px;
  letter-spacing: 0.5px;
}
.selected .nav-label {
  color: var(--md-sys-color-secondary);
  font-weight: 700;
}
.expanded .nav-item {
  min-height: 56px;
  flex-direction: row;
  margin-bottom: var(--gap-xs);
}
.expanded .indicator {
  position: absolute;
  inset: 0;
  width: auto;
  height: auto;
  border-radius: 28px;
  justify-items: start;
  padding-inline-start: var(--gap-l);
}
.expanded .nav-label {
  position: relative;
  margin-inline-start: 52px;
  font-size: 14px;
  line-height: 20px;
  letter-spacing: 0.1px;
  pointer-events: none;
}
.expanded .selected .nav-label {
  color: var(--md-sys-color-on-secondary-container);
}
ion-badge {
  position: absolute;
  top: 2px;
  inset-inline-start: 30px;
}
.expanded ion-badge {
  top: 14px;
  inset-inline-start: 30px;
}
</style>
