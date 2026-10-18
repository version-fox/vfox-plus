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

<!-- A long, monospaced text such as the open-source licenses. -->
<script setup lang="ts">
import { IonButton } from '@ionic/vue';

import { t } from '../reactivity';
import { closeDialog } from './dialog';

defineProps<{ heading: string; text: string }>();
</script>

<template>
  <div class="dialog-content">
    <div class="dialog-header">
      <h2 class="type-headline-small">{{ heading }}</h2>
    </div>
    <pre class="text mono type-body-small">{{ text }}</pre>
    <div class="dialog-actions">
      <IonButton fill="clear" @click="closeDialog()">{{ t('actionClose') }}</IonButton>
    </div>
  </div>
</template>

<style scoped>
.text {
  margin: 0;
  max-width: min(720px, 80vw);
  max-height: 60vh;
  overflow: auto;
  white-space: pre-wrap;
  user-select: text;
  color: var(--md-sys-color-on-surface-variant);
}
</style>
