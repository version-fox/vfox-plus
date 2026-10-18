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

<!-- The download progress of an installation, with a cancel button. -->
<script setup lang="ts">
import type { VfoxTask } from '@web/state/task-queue';
import { app } from '@web/ui/context';
import IconButton from '../components/IconButton.vue';
import { t, taskTitle, watched } from '../reactivity';

const props = defineProps<{ task: VfoxTask }>();
/** The task, which follows its progress. */
const task = () => watched(props.task);

function label(): string {
  const progress = task().progress;
  if (progress === null) {
    return task().status === 'queued' ? t('taskStatusQueued') : t('taskStatusRunning');
  }
  return t('taskDownloading', { percent: Math.round(progress * 100) });
}
</script>

<template>
  <div class="progress">
    <div class="bar">
      <m3e-linear-progress-indicator
        variant="wavy"
        :mode="task().progress === null ? 'indeterminate' : 'determinate'"
        :value="Math.round((task().progress ?? 0) * 100)"
        :aria-label="taskTitle(task().command)"
      />
      <span class="type-label-small">{{ label() }}</span>
    </div>
    <IconButton icon="cancel" :label="t('actionCancel')" @click="app().model.tasks.cancel(props.task)" />
  </div>
</template>

<style scoped>
.progress {
  display: flex;
  align-items: center;
  gap: var(--gap-xs);
  width: 220px;
}
.bar {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  gap: var(--gap-xs);
  color: var(--md-sys-color-on-surface-variant);
}
m3e-linear-progress-indicator {
  width: 100%;
}
</style>
