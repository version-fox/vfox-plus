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

<!-- The plugin registry: every plugin that `vfox add` can install. -->
<script setup lang="ts">
import { IonButton, IonCard, IonContent, IonSearchbar } from '@ionic/vue';
import { ref, useId } from 'vue';

import type { AvailablePlugin } from '@web/core/models';
import { app } from '@web/ui/context';
import type { Choice } from '../components/choice';
import ChoiceGroup from '../components/ChoiceGroup.vue';
import ErrorMessage from '../components/ErrorMessage.vue';
import IconButton from '../components/IconButton.vue';
import LoadingMessage from '../components/LoadingMessage.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import PluginAvatar from '../components/PluginAvatar.vue';
import StatusMessage from '../components/StatusMessage.vue';
import { askCustomPlugin } from '../overlays';
import { repositoryPath } from '../presentation';
import { t, watched } from '../reactivity';

type PluginFilter = 'all' | 'official' | 'community';

const { model, shell, host } = app();
const registry = () => watched(model.registry);
const tasks = () => watched(model.tasks);
void model.registry.ensureLoaded();

const query = ref('');
const filter = ref<PluginFilter>('all');
const ids = useId();

function installed(): ReadonlySet<string> {
  return new Set((watched(model.installed).value ?? []).map((sdk) => sdk.name));
}

function visible(plugins: readonly AvailablePlugin[]): AvailablePlugin[] {
  const search = query.value.trim().toLowerCase();
  return plugins.filter(
    (plugin) =>
      (filter.value === 'all' || (filter.value === 'official') === plugin.official) &&
      (search === '' || plugin.name.toLowerCase().includes(search) || plugin.homepage.includes(search)),
  );
}

function filters(): Choice<PluginFilter>[] {
  return [
    ['all', t('pluginsFilterAll'), null],
    ['official', t('pluginsFilterOfficial'), 'verified'],
    ['community', t('pluginsFilterCommunity'), null],
  ];
}

function adding(plugin: AvailablePlugin): boolean {
  return tasks().activeTask({ operation: 'addPlugin', sdk: plugin.name }) !== null;
}

async function addCustom(): Promise<void> {
  const source = await askCustomPlugin();
  if (source !== null) {
    await model.addPluginFromSource(source.source, source.alias);
  }
}
</script>

<template>
  <IonContent class="pane">
    <header class="page-header">
      <div class="titles">
        <h1 class="type-headline-medium-emphasized">{{ t('pluginsTitle') }}</h1>
        <p class="subtitle type-body-medium">{{ t('pluginsSubtitle') }}</p>
      </div>
      <div class="actions">
        <IconButton
          v-if="tasks().activeTask({ operation: 'updateAllPlugins' }) === null"
          icon="update"
          :label="t('pluginsUpdateAll')"
          :disabled="installed().size === 0"
          @click="model.run(model.cli.updateAllPlugins())"
        />
        <m3e-loading-indicator v-else :aria-label="t('pluginsUpdateAll')" />
        <IconButton
          icon="refresh"
          :label="t('actionRefresh')"
          :disabled="registry().isLoading"
          @click="model.registry.load()"
        />
        <IonButton color="tonal" @click="addCustom"
          ><MaterialIcon slot="start" name="add" />{{ t('pluginsAddCustom') }}</IonButton
        >
      </div>
    </header>
    <div class="filters">
      <IonSearchbar
        v-model="query"
        class="search-bar"
        :placeholder="t('pluginsSearchHint')"
        show-clear-button="always"
      />
      <ChoiceGroup :choices="filters()" :selected="filter" @choose="(choice) => (filter = choice)" />
    </div>
    <template v-if="registry().value === null">
      <LoadingMessage v-if="registry().error === null" />
      <ErrorMessage
        v-else
        :heading="t('pluginsLoadError')"
        :error="registry().error"
        :retry="() => void model.registry.load()"
      />
    </template>
    <StatusMessage
      v-else-if="visible(registry().value ?? []).length === 0"
      icon="notFound"
      :heading="t('pluginsNoMatch')"
    />
    <div v-else class="grid">
      <IonCard
        v-for="plugin in visible(registry().value ?? [])"
        :key="plugin.name"
        class="plugin"
        :class="{ installed: installed().has(plugin.name) }"
      >
        <div class="identity">
          <PluginAvatar :name="plugin.name" :size="44" />
          <div class="names">
            <div class="name">
              <span class="type-title-medium-emphasized ellipsis">{{ plugin.name }}</span>
              <template v-if="plugin.official">
                <MaterialIcon
                  :id="`${ids}-${plugin.name}`"
                  class="verified"
                  name="verified"
                  filled
                  role="img"
                  :aria-label="t('pluginsOfficial')"
                  :aria-hidden="false"
                />
                <m3e-tooltip :for="`${ids}-${plugin.name}`">{{ t('pluginsOfficial') }}</m3e-tooltip>
              </template>
            </div>
            <div class="source type-body-small ellipsis">{{ repositoryPath(plugin.homepage) }}</div>
          </div>
        </div>
        <div class="bottom">
          <IconButton icon="openInNew" :label="t('actionOpenHomepage')" @click="host.openUrl(plugin.homepage)" />
          <m3e-loading-indicator v-if="adding(plugin)" :aria-label="t('loading')" />
          <IonButton v-else-if="installed().has(plugin.name)" fill="clear" @click="shell.showSdk(plugin.name)">
            <MaterialIcon slot="start" name="check" />{{ t('pluginsAdded') }}
          </IonButton>
          <IonButton v-else color="tonal" @click="model.run(model.cli.addPlugin(plugin.name))">
            <MaterialIcon slot="start" name="add" />{{ t('pluginsAdd') }}
          </IonButton>
        </div>
      </IonCard>
    </div>
  </IonContent>
</template>

<style scoped>
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--gap-m);
  padding: 0 var(--gap-xl) var(--gap-l);
}
.filters .search-bar {
  flex: 0 1 420px;
  min-width: 240px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--gap-m);
  padding: 0 var(--gap-xl) var(--gap-xl);
}
.plugin {
  --background: var(--md-sys-color-surface-container);
  --color: var(--md-sys-color-on-surface);
  display: flex;
  flex-direction: column;
  height: 152px;
  padding: var(--gap-l);
  border-radius: var(--corner-large-increased);
}
.plugin.installed {
  --background: var(--md-sys-color-surface-container-highest);
}
.identity {
  display: flex;
  align-items: center;
  gap: var(--gap-m);
  min-width: 0;
}
.names {
  flex: 1 1 auto;
  min-width: 0;
}
.name {
  display: flex;
  align-items: center;
  gap: var(--gap-xs);
}
.verified {
  --icon-size: 18px;
  color: var(--md-sys-color-primary);
}
.source {
  color: var(--md-sys-color-on-surface-variant);
}
.bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
}
m3e-loading-indicator {
  --m3e-loading-indicator-size: 40px;
}
</style>
