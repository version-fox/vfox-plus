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
  Installed SDKs, in the list-detail canonical layout: wide pages show the
  list and the selected SDK side by side, and the detail fades through when
  the selection changes; narrow pages show one of them at a time and move
  between them along the horizontal shared axis.
-->
<script setup lang="ts">
import { IonContent } from '@ionic/vue';
import { onUpdated, ref } from 'vue';

import type { InstalledSdk } from '@web/core/models';
import { app } from '@web/ui/context';
import ErrorMessage from '../components/ErrorMessage.vue';
import LoadingMessage from '../components/LoadingMessage.vue';
import StatusMessage from '../components/StatusMessage.vue';
import { TWO_PANE_WIDTH } from '../presentation';
import { t, watched } from '../reactivity';
import { fadeThroughEnter, sharedAxis } from '../transitions';
import { useWidth } from '../use-width';
import SdkDetail from './SdkDetail.vue';
import SdkList from './SdkList.vue';

const { model, shell } = app();
const installed = () => watched(model.installed);
const root = ref<HTMLElement | null>(null);
const width = useWidth(root, 0);
const query = ref('');
const detailHost = ref<HTMLElement | null>(null);
const single = ref<HTMLElement | null>(null);

/** The selected SDK, if it is installed. */
function selected(sdks: readonly InstalledSdk[]): string | null {
  const name = watched(shell).selectedSdk;
  return sdks.some((sdk) => sdk.name === name) ? name : null;
}

function twoPanes(): boolean {
  return width.value >= TWO_PANE_WIDTH || width.value === 0;
}

let shownSelection: string | null = null;
let narrowShown: 'list' | 'detail' | null = null;
onUpdated(() => {
  const sdks = installed().value;
  if (sdks === null || width.value === 0) {
    return;
  }
  const selection = selected(sdks);
  if (twoPanes()) {
    narrowShown = null;
    if (selection !== shownSelection && detailHost.value !== null) {
      // Fade through between SDKs in the detail pane.
      fadeThroughEnter(detailHost.value, () => undefined);
    }
  } else {
    const shown = selection === null ? 'list' : 'detail';
    if (narrowShown !== null && shown !== narrowShown && single.value !== null) {
      // Forwards into the detail, backwards out of it.
      sharedAxis(single.value, shown === 'detail' ? 1 : -1);
    }
    narrowShown = shown;
  }
  shownSelection = selection;
});
</script>

<template>
  <div ref="root" class="sdks-page">
    <IonContent v-if="installed().value === null" class="pane">
      <LoadingMessage v-if="installed().error === null" />
      <ErrorMessage
        v-else
        :heading="t('sdksLoadError')"
        :error="installed().error"
        :retry="() => void model.installed.load()"
      />
    </IonContent>
    <div v-else-if="twoPanes()" class="layout">
      <IonContent class="pane list-pane">
        <SdkList
          v-model:query="query"
          :sdks="installed().value!"
          :selected="selected(installed().value!)"
          :chevrons="false"
        />
      </IonContent>
      <IonContent class="pane detail-pane">
        <div ref="detailHost" class="detail-host">
          <StatusMessage
            v-if="selected(installed().value!) === null"
            icon="sdks"
            shape="gem"
            :heading="t('sdksSelectPrompt')"
          />
          <SdkDetail v-else :key="selected(installed().value!) ?? ''" :name="selected(installed().value!) ?? ''" />
        </div>
      </IonContent>
    </div>
    <div v-else ref="single" class="single">
      <IonContent v-if="selected(installed().value!) === null" class="pane">
        <SdkList v-model:query="query" :sdks="installed().value!" :selected="null" :chevrons="true" />
      </IonContent>
      <IonContent v-else :key="selected(installed().value!) ?? ''" class="pane">
        <SdkDetail :name="selected(installed().value!) ?? ''" back @back="shell.showSdk(null)" />
      </IonContent>
    </div>
  </div>
</template>

<style scoped>
.sdks-page {
  height: 100%;
}
.layout {
  display: flex;
  gap: var(--gap-xl);
  height: 100%;
}
.list-pane {
  flex: none;
  width: 360px;
}
.detail-pane {
  flex: 1 1 auto;
  min-width: 0;
}
.detail-host {
  min-height: 100%;
}
.single {
  height: 100%;
}
</style>
