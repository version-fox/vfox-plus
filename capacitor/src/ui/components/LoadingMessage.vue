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

<!-- A centred loading indicator with an optional label. -->
<script setup lang="ts">
import { t } from '../reactivity';

withDefaults(defineProps<{ label?: string }>(), { label: '' });
</script>

<template>
  <div class="loading-host">
    <div class="loading" role="progressbar" :aria-label="label === '' ? t('loading') : label">
      <m3e-loading-indicator variant="contained" />
      <span v-if="label !== ''" class="type-body-medium">{{ label }}</span>
    </div>
  </div>
</template>

<style scoped>
.loading-host {
  display: grid;
  place-items: center;
  padding: var(--gap-xxl);
  min-height: 100%;
}
.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--gap-l);
  text-align: center;
  color: var(--md-sys-color-on-surface-variant);
}
m3e-loading-indicator {
  --m3e-loading-indicator-size: 64px;
}
</style>
