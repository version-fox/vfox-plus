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
  One task of the list: status, title, progress, actions and its output,
  which expands below it.
-->
<script setup lang="ts">
import { IonItem, IonLabel } from '@ionic/vue';
import { nextTick, onUpdated, ref } from 'vue';

import { commandLine } from '@web/core/vfox/command';
import { isActive, type VfoxTask } from '@web/state/task-queue';
import { app } from '@web/ui/context';
import IconButton from '../components/IconButton.vue';
import MaterialIcon from '../components/MaterialIcon.vue';
import { snackbar } from '../snackbar';
import { expand } from '../transitions';
import { t, taskTitle, watched } from '../reactivity';

const props = defineProps<{ task: VfoxTask; first: boolean; last: boolean }>();
/** The task, which follows its status, progress and output. */
const task = () => watched(props.task);

const expanded = ref(false);
const frame = ref<HTMLElement | null>(null);
const output = ref<HTMLElement | null>(null);

function toggle(): void {
  expanded.value = !expanded.value;
  if (expanded.value) {
    void nextTick(() => {
      if (frame.value !== null) {
        expand(frame.value);
      }
    });
  }
}

function detail(): string {
  const current = task();
  switch (current.status) {
    case 'queued':
      return t('taskStatusQueued');
    case 'running':
      return current.progress === null
        ? t('taskStatusRunning')
        : t('taskDownloading', { percent: Math.round(current.progress * 100) });
    case 'succeeded':
      return t('taskStatusSucceeded');
    case 'failed':
      return `${t('taskStatusFailed')} · ${current.errorMessage ?? ''}`;
    case 'cancelled':
      return t('taskStatusCancelled');
  }
}

function command(): string {
  return `$ ${commandLine(task().command)}`;
}

function text(): string {
  return [command(), ...task().log].join('\n');
}

async function copyOutput(): Promise<void> {
  await app().host.copyText(text());
  snackbar(t('actionCopied'));
}

// Show the end of the output, where vfox writes.
onUpdated(() => {
  if (output.value !== null) {
    output.value.scrollTop = output.value.scrollHeight;
  }
});
</script>

<template>
  <div class="task" :class="{ first, last, expanded }">
    <IonItem lines="none" class="summary" @click="toggle">
      <div slot="start" class="status" :class="task().status">
        <MaterialIcon v-if="task().status === 'queued'" name="queued" />
        <m3e-loading-indicator v-else-if="task().status === 'running'" :aria-label="t('taskStatusRunning')" />
        <MaterialIcon v-else-if="task().status === 'succeeded'" name="succeeded" filled />
        <MaterialIcon v-else-if="task().status === 'failed'" name="error" filled />
        <MaterialIcon v-else name="cancel" />
      </div>
      <IonLabel>
        <div class="title type-body-large-emphasized">{{ taskTitle(task().command) }}</div>
        <div class="detail type-body-medium" :class="{ failed: task().status === 'failed' }">{{ detail() }}</div>
        <m3e-linear-progress-indicator
          v-if="task().status === 'running' && task().command.operation === 'install'"
          variant="wavy"
          :mode="task().progress === null ? 'indeterminate' : 'determinate'"
          :value="Math.round((task().progress ?? 0) * 100)"
        />
      </IonLabel>
      <div slot="end" class="actions" @click.stop>
        <IconButton
          v-if="isActive(task().status)"
          icon="cancel"
          :label="t('actionCancel')"
          @click="app().model.tasks.cancel(props.task)"
        />
        <IconButton
          v-else-if="task().status === 'failed' || task().status === 'cancelled'"
          icon="retry"
          :label="t('actionRetry')"
          @click="app().model.tasks.retry(props.task)"
        />
        <IconButton
          class="toggle"
          icon="expandMore"
          :label="expanded ? t('taskHideOutput') : t('taskShowOutput')"
          @click="toggle"
        />
      </div>
    </IonItem>
    <div v-if="expanded" ref="frame" class="output-frame">
      <div class="output">
        <pre ref="output" class="mono type-body-small">{{
          task().log.length === 0 ? `${command()}\n${t('taskNoOutput')}` : text()
        }}</pre>
        <IconButton icon="copy" :label="t('taskCopyOutput')" @click="copyOutput" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.task {
  --outer: var(--corner-large-increased);
  --inner: var(--corner-extra-small);
}
.summary {
  --background: var(--md-sys-color-surface-container);
  --border-radius: var(--inner);
  --padding-start: var(--gap-l);
  --inner-padding-end: var(--gap-l);
  --min-height: 72px;
}
.summary {
  cursor: pointer;
}
.summary:hover {
  --inner: var(--corner-medium);
}
.summary:hover::part(native) {
  background: color-mix(in srgb, var(--md-sys-color-on-surface) 8%, var(--md-sys-color-surface-container));
}
.first .summary {
  --border-radius: var(--outer) var(--outer) var(--inner) var(--inner);
}
.last:not(.expanded) .summary {
  --border-radius: var(--inner) var(--inner) var(--outer) var(--outer);
}
.first.last:not(.expanded) .summary {
  --border-radius: var(--outer);
}
.status {
  flex: none;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  margin-inline-end: var(--gap-l);
  color: var(--md-sys-color-on-surface-variant);
}
.status.succeeded {
  color: var(--md-sys-color-primary);
}
.status.failed {
  color: var(--md-sys-color-error);
}
.status m3e-loading-indicator {
  --m3e-loading-indicator-size: 32px;
}
.title {
  color: var(--md-sys-color-on-surface);
}
.detail {
  color: var(--md-sys-color-on-surface-variant);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  overflow-wrap: anywhere;
  white-space: normal;
}
.detail.failed {
  color: var(--md-sys-color-error);
}
m3e-linear-progress-indicator {
  display: block;
  max-width: 480px;
  margin-top: var(--gap-s);
}
.actions {
  display: flex;
  align-items: center;
}
.toggle {
  transition: transform var(--md-sys-motion-spring-fast-spatial);
}
.expanded .toggle {
  transform: rotate(180deg);
}
.output-frame {
  overflow: hidden;
}
.output {
  display: flex;
  align-items: flex-start;
  gap: var(--gap-xs);
  margin-top: var(--gap-hairline);
  padding: var(--gap-s) var(--gap-xs) var(--gap-m) var(--gap-l);
  background: var(--md-sys-color-surface-container-highest);
  border-radius: var(--inner);
}
.last .output {
  border-bottom-left-radius: var(--outer);
  border-bottom-right-radius: var(--outer);
}
pre {
  flex: 1 1 auto;
  margin: 0;
  max-height: 320px;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.5;
  user-select: text;
  cursor: text;
  color: var(--md-sys-color-on-surface);
}
</style>
