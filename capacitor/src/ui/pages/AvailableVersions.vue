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
  The versions of an SDK that can be installed, with a filter. Hundreds of
  versions are common, so the list is virtualised: only the rows near the
  visible part of the page are rendered, in the scroll element of the page.
-->
<script setup lang="ts">
import { IonInput } from '@ionic/vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { computed, onMounted, onUpdated, ref, shallowRef } from 'vue';

import type { InstalledSdk } from '@web/core/models';
import type { SdkDetailController, VersionFilter } from '@web/state/controllers';
import { app } from '@web/ui/context';
import type { Choice } from '../components/choice';
import ChoiceGroup from '../components/ChoiceGroup.vue';
import ErrorMessage from '../components/ErrorMessage.vue';
import LoadingMessage from '../components/LoadingMessage.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import StatusMessage from '../components/StatusMessage.vue';
import { t, watched } from '../reactivity';
import AvailableVersionRow from './AvailableVersionRow.vue';

const props = defineProps<{ sdk: InstalledSdk; controller: SdkDetailController }>();

const { model } = app();
/** The controller of the page, which follows the query, the filter and the versions. */
const controller = () => watched(props.controller);

const installed = computed(() => {
  watched(model.installed);
  return new Set(props.sdk.versions);
});
const visible = computed(() => controller().visibleVersions(installed.value));

const list = ref<HTMLElement | null>(null);
const scroller = shallowRef<HTMLElement | null>(null);
/** Where the list starts in the scroll element. */
const offset = ref(0);

const virtualizer = useVirtualizer(
  computed(() => ({
    count: visible.value.length,
    getScrollElement: () => scroller.value,
    estimateSize: () => 66,
    overscan: 8,
    scrollMargin: offset.value,
    getItemKey: (index: number) => visible.value[index]?.version ?? index,
  })),
);

function measure(element: unknown): void {
  if (element instanceof Element) {
    virtualizer.value.measureElement(element);
  }
}

function updateOffset(): void {
  const scroll = scroller.value;
  if (scroll !== null && list.value !== null) {
    const next = list.value.getBoundingClientRect().top - scroll.getBoundingClientRect().top + scroll.scrollTop;
    if (Math.abs(next - offset.value) > 0.5) {
      offset.value = next;
    }
  }
}

onMounted(async () => {
  const content = list.value?.closest('ion-content') ?? document.querySelector('ion-content');
  scroller.value = (await content?.getScrollElement()) ?? null;
  updateOffset();
});
onUpdated(updateOffset);

function filters(): Choice<VersionFilter>[] {
  return [
    ['all', t('availableFilterAll'), null],
    ['withNote', t('availableFilterTagged'), null],
    ['installed', t('availableFilterInstalled'), null],
  ];
}
</script>

<template>
  <template v-if="!watched(controller().versions).hasValue">
    <LoadingMessage v-if="watched(controller().versions).error === null" :label="t('availableLoading')" />
    <ErrorMessage
      v-else
      :heading="t('availableError', { name: sdk.name })"
      :error="watched(controller().versions).error"
      :retry="() => void props.controller.versions.load()"
    />
  </template>
  <template v-else>
    <div class="toolbar">
      <IonInput
        class="filter"
        fill="outline"
        :placeholder="t('availableSearchHint')"
        :aria-label="t('availableSearchHint')"
        :value="controller().query"
        @ion-input="props.controller.query = String($event.detail.value ?? '')"
      >
        <MaterialIcon slot="start" name="search" />
      </IonInput>
      <ChoiceGroup
        :choices="filters()"
        :selected="controller().filter"
        @choose="(filter) => (props.controller.filter = filter)"
      />
      <span class="count type-label-large">{{ t('availableCount', { count: visible.length }) }}</span>
    </div>
    <StatusMessage v-if="visible.length === 0" icon="notFound" :heading="t('availableEmpty')" />
    <div v-else ref="list" class="list" :style="{ height: `${virtualizer.getTotalSize()}px` }">
      <div
        v-for="row in virtualizer.getVirtualItems()"
        :key="String(row.key)"
        :ref="measure"
        class="row"
        :data-index="row.index"
        :style="{ transform: `translateY(${row.start - offset}px)` }"
      >
        <AvailableVersionRow
          v-if="visible[row.index] !== undefined"
          :name="sdk.name"
          :version="visible[row.index]!"
          :installed="installed.has(visible[row.index]!.version)"
          :first="row.index === 0"
          :last="row.index === visible.length - 1"
        />
      </div>
    </div>
  </template>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--gap-s) var(--gap-m);
  padding: 0 var(--gap-xl) var(--gap-m);
}
.filter {
  width: 280px;
  min-height: 48px;
}
.count {
  color: var(--md-sys-color-on-surface-variant);
}
.list {
  position: relative;
  margin: 0 var(--gap-xl) var(--gap-xl);
}
.row {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
}
</style>
