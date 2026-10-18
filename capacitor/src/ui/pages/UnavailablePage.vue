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

<!-- Shown instead of the pages while vfox is missing or broken. -->
<script setup lang="ts">
import { IonButton, IonContent } from '@ionic/vue';

import { app, LINKS } from '@web/ui/context';
import LoadingMessage from '../components/LoadingMessage.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import StatusMessage from '../components/StatusMessage.vue';
import { t, watched } from '../reactivity';
import { locateVfox } from '../vfox';

const { model, host } = app();
const vfox = () => watched(model.vfox);

function checking(): boolean {
  const availability = vfox().availability;
  return availability === 'unknown' || availability === 'checking' || availability === 'ready';
}
</script>

<template>
  <IonContent class="pane">
    <LoadingMessage v-if="checking()" :label="t('vfoxCheckingTitle')" />
    <StatusMessage
      v-else
      :icon="vfox().availability === 'notFound' ? 'notFound' : 'warning'"
      :shape="vfox().availability === 'notFound' ? '4-leaf-clover' : 'soft-burst'"
      :error="vfox().availability !== 'notFound'"
      :heading="vfox().availability === 'notFound' ? t('vfoxMissingTitle') : t('vfoxBrokenTitle')"
      :body="
        vfox().availability === 'notFound'
          ? t('vfoxMissingBody')
          : t('vfoxBrokenBody', { path: vfox().executable ?? 'vfox' })
      "
    >
      <IonButton @click="host.openUrl(LINKS.vfoxInstall)">
        <MaterialIcon slot="start" name="openInNew" />{{ t('vfoxInstallGuide') }}
      </IonButton>
      <IonButton color="tonal" @click="locateVfox()">
        <MaterialIcon slot="start" name="folder" />{{ t('vfoxLocate') }}
      </IonButton>
      <IonButton fill="outline" @click="model.vfox.apply(model.settings.settings)">
        <MaterialIcon slot="start" name="retry" />{{ t('actionRetry') }}
      </IonButton>
    </StatusMessage>
  </IonContent>
</template>
