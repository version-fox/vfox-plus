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
  Everything about one installed plugin: its metadata, its installed versions
  and the versions that can be installed.
-->
<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue';

import type { InstalledSdk } from '@web/core/models';
import { app } from '@web/ui/context';
import ChoiceGroup from '../components/ChoiceGroup.vue';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import PluginAvatar from '../components/PluginAvatar.vue';
import { confirmAction, showMenu, type MenuItem } from '../overlays';
import { t, watched } from '../reactivity';
import { useWidth } from '../use-width';
import AvailableVersions from './AvailableVersions.vue';
import InstalledVersions from './InstalledVersions.vue';

type Tab = 'installed' | 'available';

const props = withDefaults(
  defineProps<{
    name: string;
    /** Whether to show a back button, which emits `back`. */
    back?: boolean;
  }>(),
  { back: false },
);
defineEmits<{ back: [] }>();

const { model, host, shell } = app();
const controller = model.sdkDetail(props.name);
void controller.info.load();
const stopTasks = model.tasks.onFinished((task) => {
  if (task.command.sdk === props.name && task.command.operation === 'updatePlugin') {
    void controller.info.load();
  }
});
onBeforeUnmount(() => {
  stopTasks();
  controller.dispose();
});
/** The metadata of the plugin, once loaded. */
const info = () => watched(controller.info).value;
/** Whether the plugin is being updated. */
const updating = (): boolean =>
  watched(model.tasks).activeTask({ operation: 'updatePlugin', sdk: props.name }) !== null;

const root = ref<HTMLElement | null>(null);
const width = useWidth(root, 1000);
const tab = ref<Tab>('installed');

function sdk(): InstalledSdk {
  watched(model.installed);
  return model.installedSdk(props.name) ?? { name: props.name, versions: [], current: null };
}

function showTab(next: Tab): void {
  tab.value = next;
  if (next === 'available') {
    void controller.versions.ensureLoaded();
  }
}

async function removePlugin(): Promise<void> {
  const confirmed = await confirmAction({
    title: t('sdkRemoveTitle', { name: props.name }),
    body: t('sdkRemoveBody', { count: sdk().versions.length }),
    confirmLabel: t('sdkRemoveConfirm'),
    icon: 'removePlugin',
    destructive: true,
  });
  if (confirmed) {
    model.run(model.cli.removePlugin(props.name));
    shell.showSdk(null);
  }
}

async function more(event: Event): Promise<void> {
  const homepage = controller.info.value?.homepage ?? '';
  const items: MenuItem[] = [
    ...(homepage === '' ? [] : [{ id: 'homepage', label: t('actionOpenHomepage'), icon: 'openInNew' } as const]),
    { id: 'remove', label: t('sdkRemovePlugin'), icon: 'removePlugin', danger: true },
  ];
  switch (await showMenu(event, items)) {
    case 'homepage':
      await host.openUrl(homepage);
      break;
    case 'remove':
      await removePlugin();
      break;
  }
}
</script>

<template>
  <div ref="root" class="sdk-detail">
    <header class="header" :class="{ narrow: width < 520 }">
      <div v-if="width < 520" class="actions">
        <IconButton v-if="back" icon="back" :label="t('actionClose')" @click="$emit('back')" />
        <span class="spacer"></span>
        <IconButton
          v-if="!updating()"
          icon="update"
          :label="t('sdkUpdatePlugin')"
          @click="model.run(model.cli.updatePlugin(name))"
        />
        <m3e-loading-indicator v-else class="small" :aria-label="t('loading')" />
        <IconButton icon="more" :label="t('sdkMoreActions')" @click="more" />
      </div>
      <div class="identity">
        <IconButton v-if="back && width >= 520" icon="back" :label="t('actionClose')" @click="$emit('back')" />
        <PluginAvatar :name="name" :size="width < 520 ? 56 : 72" />
        <div class="text">
          <h1 class="type-headline-medium-emphasized">{{ name }}</h1>
          <div class="tags">
            <span class="tag" :class="{ emphasized: sdk().current !== null }">
              <MaterialIcon v-if="sdk().current !== null" name="current" filled />
              {{ sdk().current === null ? t('sdksNoCurrent') : t('sdksCurrent', { version: sdk().current ?? '' }) }}
            </span>
            <span v-if="(info()?.version ?? '') !== ''" class="tag">
              {{ t('sdkPluginVersion', { version: info()?.version ?? '' }) }}
            </span>
          </div>
          <p v-if="(info()?.description ?? '') !== ''" class="description type-body-medium">
            {{ info()?.description }}
          </p>
        </div>
      </div>
      <div v-if="width >= 520" class="actions">
        <IconButton
          v-if="!updating()"
          icon="update"
          :label="t('sdkUpdatePlugin')"
          @click="model.run(model.cli.updatePlugin(name))"
        />
        <m3e-loading-indicator v-else class="small" :aria-label="t('loading')" />
        <IconButton icon="more" :label="t('sdkMoreActions')" @click="more" />
      </div>
    </header>
    <div class="tabs">
      <ChoiceGroup
        :choices="[
          ['installed', `${t('sdkTabInstalled')} (${sdk().versions.length})`, 'versions'],
          ['available', t('sdkTabAvailable'), 'install'],
        ]"
        :selected="tab"
        @choose="showTab"
      />
    </div>
    <InstalledVersions v-if="tab === 'installed'" :sdk="sdk()" @browse="showTab('available')" />
    <AvailableVersions v-else :sdk="sdk()" :controller="controller" />
  </div>
</template>

<style scoped>
.header {
  display: flex;
  align-items: flex-start;
  gap: var(--gap-l);
  padding: var(--gap-xl) var(--gap-l) var(--gap-xl) var(--gap-s);
}
.header.narrow {
  flex-direction: column;
  align-items: stretch;
  gap: var(--gap-s);
}
.identity {
  display: flex;
  align-items: flex-start;
  gap: var(--gap-l);
  flex: 1 1 auto;
  min-width: 0;
}
.header:not(.narrow) .identity {
  padding-inline-start: var(--gap-l);
}
.text {
  flex: 1 1 auto;
  min-width: 0;
}
h1 {
  color: var(--md-sys-color-on-surface);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--gap-xs) var(--gap-s);
  margin-top: var(--gap-xs);
}
.description {
  margin-top: var(--gap-s);
  color: var(--md-sys-color-on-surface-variant);
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.actions {
  display: flex;
  align-items: center;
  gap: var(--gap-xs);
  flex: none;
}
.narrow .actions .spacer {
  flex: 1 1 auto;
}
/* The tabs keep their own width, as in the other implementations. */
.tabs {
  display: flex;
  padding: 0 var(--gap-xl) var(--gap-l);
}
m3e-loading-indicator.small {
  --m3e-loading-indicator-size: 40px;
}
</style>
