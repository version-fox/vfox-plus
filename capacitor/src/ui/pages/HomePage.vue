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

<!-- The overview: what is installed, what is in use and where to go next. -->
<script setup lang="ts">
import { IonCard, IonContent, IonItem, IonLabel, IonList } from '@ionic/vue';
import { ref } from 'vue';

import type { InstalledSdk } from '@web/core/models';
import { app, LINKS } from '@web/ui/context';
import ErrorMessage from '../components/ErrorMessage.vue';
import IconButton from '../components/IconButton.vue';
import LoadingMessage from '../components/LoadingMessage.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import PluginAvatar from '../components/PluginAvatar.vue';
import type { IconName } from '../icons';
import { ACTIVATION_COMMAND, type ShapeName } from '../presentation';
import { t, watched } from '../reactivity';
import { snackbar } from '../snackbar';
import { useWidth } from '../use-width';

interface Tile {
  readonly value: number;
  readonly label: string;
  readonly icon: IconName;
  readonly shape: ShapeName;
  readonly role: 'primary' | 'secondary' | 'tertiary';
}

const { model, shell, host } = app();
const installed = () => watched(model.installed);
const content = ref<{ $el: Element } | null>(null);
const width = useWidth(content, 1080);

function active(sdks: readonly InstalledSdk[]): InstalledSdk[] {
  return sdks.filter((sdk) => sdk.current !== null);
}

function tiles(sdks: readonly InstalledSdk[]): Tile[] {
  const versions = sdks.reduce((sum, sdk) => sum + sdk.versions.length, 0);
  return [
    {
      value: sdks.length,
      label: t('homeStatPlugins', { count: sdks.length }),
      icon: 'plugins',
      shape: '9-sided-cookie',
      role: 'primary',
    },
    {
      value: versions,
      label: t('homeStatVersions', { count: versions }),
      icon: 'versions',
      shape: '4-leaf-clover',
      role: 'secondary',
    },
    { value: active(sdks).length, label: t('homeStatActive'), icon: 'current', shape: 'sunny', role: 'tertiary' },
  ];
}

function subtitle(): string {
  const vfox = watched(model.vfox);
  return [t('homeSubtitle', { version: vfox.version ?? '?' }), vfox.executable]
    .filter((part) => part !== null)
    .join(' · ');
}

async function copyCommand(): Promise<void> {
  await host.copyText(ACTIVATION_COMMAND);
  snackbar(t('actionCopied'));
}
</script>

<template>
  <IonContent ref="content" class="pane">
    <template v-if="installed().value === null">
      <LoadingMessage v-if="installed().error === null" />
      <ErrorMessage
        v-else
        :heading="t('sdksLoadError')"
        :error="installed().error"
        :retry="() => void model.installed.load()"
      />
    </template>
    <template v-else>
      <header class="page-header">
        <div class="titles">
          <h1 class="type-headline-medium-emphasized">{{ t('appTitle') }}</h1>
          <p class="subtitle type-body-medium">{{ subtitle() }}</p>
        </div>
        <div class="actions">
          <IconButton
            icon="refresh"
            :label="t('actionRefresh')"
            :disabled="installed().isLoading"
            @click="model.installed.load()"
          />
        </div>
      </header>
      <!-- Narrow windows keep the three tiles in a row, in a compact size. -->
      <div class="content" :class="{ compact: width < 560 + 48, narrow: width < 760 + 48 }">
        <div class="stats">
          <IonCard
            v-for="tile in tiles(installed().value ?? [])"
            :key="tile.icon"
            button
            class="tile"
            :style="{
              '--tile-container': `var(--md-sys-color-${tile.role}-container)`,
              '--tile-on-container': `var(--md-sys-color-on-${tile.role}-container)`,
            }"
            @click="shell.showSdk(null)"
          >
            <m3e-shape :name="tile.shape" aria-hidden="true" />
            <MaterialIcon :name="tile.icon" filled />
            <span
              class="value"
              :class="width < 560 + 48 ? 'type-headline-large-emphasized' : 'type-display-small-emphasized'"
              >{{ tile.value }}</span
            >
            <span class="label type-label-large-emphasized">{{ tile.label }}</span>
          </IonCard>
        </div>
        <h2 class="group-header">{{ t('homeActiveTitle') }}</h2>
        <IonList class="segmented" lines="none">
          <IonItem
            v-if="active(installed().value ?? []).length === 0"
            button
            :detail="false"
            class="list-action"
            @click="shell.showSdk(null)"
          >
            <MaterialIcon slot="start" name="info" />
            <IonLabel>{{ t('homeActiveEmpty') }}</IonLabel>
          </IonItem>
          <IonItem
            v-for="sdk in active(installed().value ?? [])"
            :key="sdk.name"
            button
            :detail="false"
            class="list-action"
            @click="shell.showSdk(sdk.name)"
          >
            <PluginAvatar slot="start" :name="sdk.name" />
            <IonLabel>
              <div class="type-body-large">{{ sdk.name }}</div>
              <div class="supporting type-body-medium">{{ sdk.current }}</div>
            </IonLabel>
            <MaterialIcon slot="end" name="chevronRight" />
          </IonItem>
        </IonList>
        <div class="shortcuts">
          <IonCard class="shell-card">
            <div class="heading">
              <MaterialIcon name="terminal" />
              <h2 class="type-title-medium-emphasized">{{ t('homeShellTitle') }}</h2>
            </div>
            <p class="type-body-medium muted">{{ t('homeShellBody') }}</p>
            <div class="command">
              <code class="mono type-body-medium">{{ ACTIVATION_COMMAND }}</code>
              <IconButton icon="copy" :label="t('actionCopy')" @click="copyCommand" />
            </div>
          </IonCard>
          <IonCard button class="shortcut" @click="shell.go('plugins')">
            <span class="shape" aria-hidden="true"
              ><m3e-shape name="6-sided-cookie" /><MaterialIcon name="explore"
            /></span>
            <span class="text">
              <span class="type-title-small-emphasized">{{ t('homeExploreTitle') }}</span>
              <span class="type-body-small muted">{{ t('homeExploreBody') }}</span>
            </span>
            <MaterialIcon class="trailing" name="chevronRight" />
          </IonCard>
          <IonCard button class="shortcut" @click="host.openUrl(LINKS.vfoxDocs)">
            <span class="shape" aria-hidden="true"
              ><m3e-shape name="6-sided-cookie" /><MaterialIcon name="docs"
            /></span>
            <span class="text">
              <span class="type-title-small-emphasized">{{ t('homeDocsTitle') }}</span>
              <span class="type-body-small muted">{{ t('homeDocsBody') }}</span>
            </span>
            <MaterialIcon class="trailing" name="openInNew" />
          </IonCard>
        </div>
      </div>
    </template>
  </IonContent>
</template>

<style scoped>
.content {
  max-width: 1080px;
}
.stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--gap-m);
}
.tile {
  --background: var(--tile-container);
  --color: var(--tile-on-container);
  height: 136px;
  border-radius: var(--corner-extra-large);
  transition: border-radius var(--md-sys-motion-spring-fast-spatial);
}
.tile::part(native) {
  position: relative;
  height: 100%;
  padding: var(--gap-xl);
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  text-align: start;
}
.tile.ion-activated {
  border-radius: var(--corner-large);
}
.tile m3e-shape {
  position: absolute;
  inset-inline-end: -28px;
  bottom: -36px;
  --m3e-shape-size: 140px;
  --m3e-shape-container-color: color-mix(in srgb, currentColor 8%, transparent);
}
.tile .value {
  position: relative;
  margin-top: auto;
  line-height: 1;
}
.tile .label {
  position: relative;
  margin-top: var(--gap-xs);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.compact .stats {
  gap: var(--gap-s);
}
.compact .tile::part(native) {
  padding: var(--gap-l);
}
.shortcuts {
  display: grid;
  grid-template-columns: 3fr 2fr;
  grid-template-rows: 1fr 1fr;
  gap: var(--gap-m);
  margin-top: var(--gap-xl);
}
.shell-card {
  grid-row: span 2;
  padding: var(--gap-xl);
  --background: var(--md-sys-color-surface-container-highest);
  --color: var(--md-sys-color-on-surface);
}
.narrow .shortcuts {
  grid-template-columns: 1fr;
  grid-template-rows: none;
}
.narrow .shell-card {
  grid-row: auto;
}
.shell-card .heading {
  display: flex;
  align-items: center;
  gap: var(--gap-m);
}
.shell-card .heading .material-icon {
  color: var(--md-sys-color-primary);
}
.shell-card p {
  margin: var(--gap-s) 0 var(--gap-m);
}
.command {
  display: flex;
  align-items: center;
  gap: var(--gap-s);
  padding-inline: var(--gap-l) var(--gap-xs);
  min-height: 48px;
  border-radius: var(--corner-medium);
  background: var(--md-sys-color-surface-container-lowest);
}
.command code {
  flex: 1 1 auto;
  overflow-x: auto;
  white-space: nowrap;
  user-select: text;
  cursor: text;
}
.shortcut {
  --background: var(--md-sys-color-surface-container);
  --color: var(--md-sys-color-on-surface);
  transition: border-radius var(--md-sys-motion-spring-fast-spatial);
}
.shortcut.ion-activated {
  border-radius: var(--corner-small);
}
.shortcut::part(native) {
  height: 100%;
  display: flex;
  align-items: center;
  gap: var(--gap-l);
  padding: var(--gap-l);
  text-align: start;
}
.shortcut .shape {
  position: relative;
  flex: none;
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  color: var(--md-sys-color-on-secondary-container);
}
.shortcut .shape m3e-shape {
  position: absolute;
  inset: 0;
  --m3e-shape-size: 48px;
  --m3e-shape-container-color: var(--md-sys-color-secondary-container);
}
.shortcut .shape .material-icon {
  position: relative;
}
.shortcut .text {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--gap-xs);
}
.shortcut .trailing {
  color: var(--md-sys-color-on-surface-variant);
}
</style>
