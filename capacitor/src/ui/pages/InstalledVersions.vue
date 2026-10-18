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
  The installed versions of an SDK, with a split button to use one globally
  or in a project, and a menu with the other actions.
-->
<script setup lang="ts">
import { IonButton, IonItem, IonLabel, IonList } from '@ionic/vue';

import type { InstalledSdk } from '@web/core/models';
import type { VfoxTask } from '@web/state/task-queue';
import { app } from '@web/ui/context';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import StatusMessage from '../components/StatusMessage.vue';
import { askProjectDirectory, confirmAction, showMenu, type MenuItem } from '../overlays';
import { t, taskTitle, watched } from '../reactivity';
import { snackbar } from '../snackbar';

const props = defineProps<{ sdk: InstalledSdk }>();
defineEmits<{ browse: [] }>();

const { model, host } = app();

/** The operation that runs on `version`, if any. */
function task(version: string): VfoxTask | null {
  return watched(model.tasks).activeTask({ sdk: props.sdk.name, version });
}

async function useInProject(version: string): Promise<void> {
  const directory = await askProjectDirectory(props.sdk.name, version);
  if (directory !== null) {
    model.run(model.cli.use(props.sdk.name, version, 'project', directory));
  }
}

async function copyPath(version: string): Promise<void> {
  const path = await model.cli.installPath(props.sdk.name, version);
  if (path !== null) {
    await host.copyText(path);
    snackbar(t('actionCopied'));
  }
}

async function uninstall(version: string, current: boolean): Promise<void> {
  const confirmed = await confirmAction({
    title: t('versionUninstallTitle', { name: props.sdk.name, version }),
    body: current ? t('versionUninstallCurrentBody') : t('versionUninstallBody'),
    confirmLabel: t('versionUninstall'),
    icon: 'uninstall',
    destructive: true,
  });
  if (confirmed) {
    model.run(model.cli.uninstall(props.sdk.name, version));
  }
}

async function useOptions(event: Event, version: string): Promise<void> {
  const choice = await showMenu(event, [
    { id: 'global', label: t('versionUseGlobally'), icon: 'globalScope' },
    { id: 'project', label: t('versionUseInProject'), icon: 'projectScope' },
  ]);
  if (choice === 'global') {
    model.run(model.cli.use(props.sdk.name, version));
  } else if (choice === 'project') {
    await useInProject(version);
  }
}

async function more(event: Event, version: string, current: boolean): Promise<void> {
  const items: MenuItem[] = [
    ...(current
      ? ([
          { id: 'project', label: t('versionUseInProject'), icon: 'projectScope' },
          { id: 'unuse', label: t('versionUnuse'), icon: 'cancel' },
        ] as const)
      : []),
    { id: 'copy', label: t('versionCopyPath'), icon: 'copy' },
    { id: 'uninstall', label: t('versionUninstall'), icon: 'uninstall', danger: true },
  ];
  switch (await showMenu(event, items)) {
    case 'project':
      await useInProject(version);
      break;
    case 'unuse':
      model.run(model.cli.unuse(props.sdk.name));
      break;
    case 'copy':
      await copyPath(version);
      break;
    case 'uninstall':
      await uninstall(version, current);
      break;
  }
}
</script>

<template>
  <StatusMessage
    v-if="sdk.versions.length === 0"
    icon="install"
    shape="pill"
    :heading="t('sdkInstalledEmpty', { name: sdk.name })"
  >
    <IonButton @click="$emit('browse')"
      ><MaterialIcon slot="start" name="install" />{{ t('sdkBrowseVersions') }}</IonButton
    >
  </StatusMessage>
  <IonList v-else class="segmented versions" lines="none">
    <IonItem v-for="version in sdk.versions" :key="version" class="version">
      <MaterialIcon
        slot="start"
        :class="{ current: sdk.current === version }"
        :name="sdk.current === version ? 'current' : 'versions'"
        :filled="sdk.current === version"
      />
      <IonLabel :class="{ 'type-body-large-emphasized': sdk.current === version }">{{ version }}</IonLabel>
      <div slot="end" class="version-actions">
        <m3e-loading-indicator
          v-if="task(version) !== null"
          class="small"
          :aria-label="taskTitle(task(version)!.command)"
        />
        <template v-else>
          <span v-if="sdk.current === version" class="tag emphasized">
            <MaterialIcon name="current" filled />{{ t('versionInUse') }}
          </span>
          <div v-else class="split-button">
            <IonButton color="tonal" class="leading" @click="model.run(model.cli.use(sdk.name, version))">
              <MaterialIcon slot="start" name="check" />{{ t('versionUse') }}
            </IonButton>
            <IonButton
              color="tonal"
              class="trailing"
              :aria-label="t('versionUseOptions')"
              @click="useOptions($event, version)"
            >
              <MaterialIcon slot="icon-only" name="expandMore" />
            </IonButton>
          </div>
          <IconButton
            icon="more"
            :label="t('versionMoreActions', { version })"
            @click="more($event, version, sdk.current === version)"
          />
        </template>
      </div>
    </IonItem>
  </IonList>
</template>

<style scoped>
.versions {
  padding: 0 var(--gap-xl) var(--gap-xl);
}
.version {
  --min-height: 64px;
}
.version > .material-icon {
  color: var(--md-sys-color-on-surface-variant);
}
.version > .material-icon.current {
  color: var(--md-sys-color-primary);
}
.version-actions {
  display: flex;
  align-items: center;
  gap: var(--gap-xs);
}
m3e-loading-indicator.small {
  --m3e-loading-indicator-size: 40px;
}
</style>
