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

<!-- A version that can be installed: its note, its additions and how to install it. -->
<script setup lang="ts">
import { IonButton, IonItem, IonLabel } from '@ionic/vue';

import { formatAddition, type AvailableVersion } from '@web/core/models';
import { app } from '@web/ui/context';
import MaterialIcon from '../components/MaterialIcon.vue';
import { t, watched } from '../reactivity';
import InstallProgress from './InstallProgress.vue';

const props = defineProps<{
  name: string;
  version: AvailableVersion;
  installed: boolean;
  /** Whether the row is the first or the last of the list, with the larger corners. */
  first: boolean;
  last: boolean;
}>();

const { model } = app();

function task(): ReturnType<typeof model.tasks.activeTask> {
  return watched(model.tasks).activeTask({ operation: 'install', sdk: props.name, version: props.version.version });
}
</script>

<template>
  <IonItem class="version-row" :class="{ first, last }" lines="none">
    <IonLabel>
      <div class="title">
        <span class="type-body-large">{{ version.version }}</span>
        <span v-if="version.note !== null" class="tag emphasized">{{ version.note }}</span>
      </div>
      <div v-if="version.additions.length > 0" class="additions type-body-medium">
        {{ version.additions.map(formatAddition).join(' · ') }}
      </div>
    </IonLabel>
    <div slot="end">
      <InstallProgress v-if="task() !== null" :task="task()!" />
      <IonButton v-else-if="installed" fill="clear" disabled>
        <MaterialIcon slot="start" name="check" />{{ t('versionInstalled') }}
      </IonButton>
      <IonButton v-else color="tonal" @click="model.run(model.cli.install(name, version.version))">
        <MaterialIcon slot="start" name="install" />{{ t('versionInstall') }}
      </IonButton>
    </div>
  </IonItem>
</template>

<style scoped>
.version-row {
  --min-height: 60px;
  margin-bottom: var(--gap-hairline);
}
.version-row ion-label {
  margin: var(--gap-s) 0;
}
.version-row.last {
  margin-bottom: 0;
}
.title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--gap-s);
}
.additions {
  color: var(--md-sys-color-on-surface-variant);
}
</style>
