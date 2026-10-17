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
  The adaptive frame around the pages: a navigation bar at the bottom of
  compact windows, the expressive navigation rail at the start of wider ones.
  Switching destinations fades through. The frame is the view container of
  Ionic, which hides it from assistive technologies while a dialog is open.
-->
<script setup lang="ts">
import { onBeforeUnmount, ref, type Component } from 'vue';

import { app, type Destination } from '@web/ui/context';
import { COMPACT_QUERY } from '../presentation';
import { watched } from '../reactivity';
import { fadeThroughEnter, fadeThroughLeave } from '../transitions';
import HomePage from '../pages/HomePage.vue';
import PluginsPage from '../pages/PluginsPage.vue';
import SdksPage from '../pages/SdksPage.vue';
import SettingsPage from '../pages/SettingsPage.vue';
import TasksPage from '../pages/TasksPage.vue';
import UnavailablePage from '../pages/UnavailablePage.vue';
import NavigationBar from './NavigationBar.vue';
import NavigationRail from './NavigationRail.vue';

const PAGES: Readonly<Record<Destination | 'unavailable', Component>> = {
  home: HomePage,
  sdks: SdksPage,
  plugins: PluginsPage,
  tasks: TasksPage,
  settings: SettingsPage,
  unavailable: UnavailablePage,
};

const { model, shell } = app();

const media = matchMedia(COMPACT_QUERY);
const compact = ref(media.matches);
const onMedia = (): void => {
  compact.value = media.matches;
};
media.addEventListener('change', onMedia);
onBeforeUnmount(() => {
  media.removeEventListener('change', onMedia);
});

/** The page shown for the destination; Settings stay reachable without vfox. */
function page(): Destination | 'unavailable' {
  const destination = watched(shell).destination;
  return !ready() && destination !== 'settings' ? 'unavailable' : destination;
}

function ready(): boolean {
  return watched(model.vfox).availability === 'ready';
}

function toggleRail(): void {
  void model.settings.update((settings) => ({ ...settings, railExpanded: !settings.railExpanded }));
}
</script>

<template>
  <div id="ion-view-container-root" class="shell" :class="{ compact }">
    <NavigationRail
      v-if="!compact"
      :expanded="watched(model.settings).settings.railExpanded"
      :ready="ready()"
      :active="watched(model.tasks).activeCount"
      @toggle="toggleRail"
    />
    <div class="stage">
      <Transition :css="false" @enter="fadeThroughEnter" @leave="fadeThroughLeave">
        <component :is="PAGES[page()]" :key="page()" class="page" />
      </Transition>
    </div>
    <NavigationBar v-if="compact" :active="watched(model.tasks).activeCount" />
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  height: 100%;
  background: var(--md-sys-color-surface);
  color: var(--md-sys-color-on-surface);
}
.shell.compact {
  flex-direction: column;
}
.stage {
  position: relative;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
}
.shell:not(.compact) .stage {
  margin: var(--gap-l) var(--gap-l) var(--gap-l) 0;
}
.page {
  position: absolute;
  inset: 0;
}
</style>
