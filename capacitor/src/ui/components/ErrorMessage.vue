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

<!-- A status message for an error, with the error text and a retry button. -->
<script setup lang="ts">
import { IonButton } from '@ionic/vue';

import { describeError } from '@web/ui/context';
import { t } from '../reactivity';
import MaterialIcon from './MaterialIcon.vue';
import StatusMessage from './StatusMessage.vue';

defineProps<{
  heading: string;
  error: unknown;
  retry?: () => void;
}>();
</script>

<template>
  <StatusMessage icon="error" shape="soft-burst" error :heading="heading" :body="describeError(error)">
    <IonButton v-if="retry !== undefined" color="tonal" @click="retry">
      <MaterialIcon slot="start" name="retry" />{{ t('actionRetry') }}
    </IonButton>
  </StatusMessage>
</template>
