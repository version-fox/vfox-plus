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

<!-- The list of installed SDKs, with a search bar, on the SDKs page. -->
<script setup lang="ts">
import { IonButton, IonItem, IonLabel, IonList, IonSearchbar } from '@ionic/vue';

import type { InstalledSdk } from '@web/core/models';
import { app } from '@web/ui/context';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import PluginAvatar from '../components/PluginAvatar.vue';
import StatusMessage from '../components/StatusMessage.vue';
import { t, watched } from '../reactivity';

defineProps<{
  sdks: readonly InstalledSdk[];
  selected: string | null;
  /** Whether the items lead to a page of their own, with a chevron. */
  chevrons: boolean;
}>();
const query = defineModel<string>('query', { required: true });

const { model, shell } = app();

function visible(sdks: readonly InstalledSdk[]): InstalledSdk[] {
  const search = query.value.trim().toLowerCase();
  return sdks.filter((sdk) => search === '' || sdk.name.toLowerCase().includes(search));
}
</script>

<template>
  <StatusMessage
    v-if="sdks.length === 0"
    icon="empty"
    shape="flower"
    :heading="t('sdksEmptyTitle')"
    :body="t('sdksEmptyBody')"
  >
    <IonButton @click="shell.go('plugins')"><MaterialIcon slot="start" name="add" />{{ t('navAddPlugin') }}</IonButton>
  </StatusMessage>
  <template v-else>
    <header class="page-header">
      <div class="titles">
        <h1 class="type-headline-medium-emphasized">{{ t('navSdks') }}</h1>
      </div>
      <div class="actions">
        <IconButton
          icon="refresh"
          :label="t('actionRefresh')"
          :disabled="watched(model.installed).isLoading"
          @click="model.installed.load()"
        />
      </div>
    </header>
    <div class="search">
      <IonSearchbar v-model="query" class="search-bar" :placeholder="t('sdksSearchHint')" show-clear-button="always" />
    </div>
    <StatusMessage
      v-if="visible(sdks).length === 0"
      icon="notFound"
      :heading="t('sdksNoMatch', { query: query.trim() })"
    />
    <IonList v-else class="segmented items" lines="none">
      <IonItem
        v-for="sdk in visible(sdks)"
        :key="sdk.name"
        button
        :detail="false"
        class="list-action"
        :class="{ selected: sdk.name === selected }"
        :aria-current="sdk.name === selected ? 'true' : 'false'"
        @click="shell.showSdk(sdk.name)"
      >
        <PluginAvatar slot="start" :name="sdk.name" />
        <IonLabel>
          <div class="type-body-large">{{ sdk.name }}</div>
          <div class="supporting type-body-medium">
            {{ sdk.current === null ? t('sdksNoCurrent') : t('sdksCurrent', { version: sdk.current }) }}
          </div>
          <div class="supporting type-body-medium">{{ t('sdksVersionCount', { count: sdk.versions.length }) }}</div>
        </IonLabel>
        <MaterialIcon v-if="chevrons" slot="end" name="chevronRight" />
      </IonItem>
    </IonList>
  </template>
</template>

<style scoped>
.search {
  padding: 0 var(--gap-l) var(--gap-m);
}
.items {
  padding: 0 var(--gap-l) var(--gap-l);
}
</style>
