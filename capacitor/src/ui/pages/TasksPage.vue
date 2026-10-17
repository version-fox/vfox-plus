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

<!-- Running, queued and finished operations with their output. -->
<script setup lang="ts">
import { IonButton, IonContent } from '@ionic/vue';

import { app } from '@web/ui/context';
import MaterialIcon from '../components/MaterialIcon.vue';
import StatusMessage from '../components/StatusMessage.vue';
import { t, watched } from '../reactivity';
import TaskItem from './TaskItem.vue';

const { model } = app();
const tasks = () => watched(model.tasks);
</script>

<template>
  <IonContent class="pane">
    <header class="page-header">
      <div class="titles">
        <h1 class="type-headline-medium-emphasized">{{ t('navTasks') }}</h1>
      </div>
      <div class="actions">
        <IonButton fill="clear" :disabled="!tasks().hasFinished" @click="model.tasks.clearFinished()">
          <MaterialIcon slot="start" name="succeeded" />{{ t('tasksClearFinished') }}
        </IonButton>
      </div>
    </header>
    <StatusMessage
      v-if="tasks().tasks.length === 0"
      icon="tasks"
      shape="sunny"
      :heading="t('tasksEmptyTitle')"
      :body="t('tasksEmptyBody')"
    />
    <div v-else class="list">
      <TaskItem
        v-for="(task, index) in tasks().tasks"
        :key="task.id"
        :task="task"
        :first="index === 0"
        :last="index === tasks().tasks.length - 1"
      />
    </div>
  </IonContent>
</template>

<style scoped>
.list {
  display: flex;
  flex-direction: column;
  gap: var(--gap-hairline);
  padding: 0 var(--gap-xl) var(--gap-xl);
}
</style>
