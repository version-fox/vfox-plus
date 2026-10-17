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
  A plugin avatar: an initial on a decorative Material 3 Expressive shape.
  The shape and the colour are derived from the name, so every plugin keeps
  its look. Hovering spins the shape by 30° on the fast spatial spring. The
  initial is generated content, which keeps it out of the text of list items.
-->
<script setup lang="ts">
import { computed } from 'vue';

import type { IconName } from '../icons';
import { avatarLook } from '../presentation';
import MaterialIcon from './MaterialIcon.vue';

const props = withDefaults(
  defineProps<{
    name: string;
    /** Width and height, in pixels. */
    size?: number;
    /** An icon to show instead of the initial. */
    icon?: IconName;
  }>(),
  { size: 40, icon: undefined },
);

const look = computed(() => avatarLook(props.name));
</script>

<template>
  <span
    class="avatar"
    :style="{ '--size': `${size}px`, color: `var(--md-sys-color-${look.content})` }"
    aria-hidden="true"
  >
    <m3e-shape :name="look.shape" :style="{ '--m3e-shape-container-color': `var(--md-sys-color-${look.container})` }" />
    <MaterialIcon v-if="icon !== undefined" :name="icon" />
    <span v-else class="initial" :data-initial="look.initial"></span>
  </span>
</template>

<style scoped>
.avatar {
  position: relative;
  flex: none;
  width: var(--size);
  height: var(--size);
  display: inline-grid;
  place-items: center;
}
m3e-shape {
  position: absolute;
  inset: 0;
  --m3e-shape-size: var(--size);
  transition: transform var(--md-sys-motion-spring-fast-spatial);
}
.avatar:hover m3e-shape {
  transform: rotate(30deg);
}
.initial::before {
  content: attr(data-initial);
  position: relative;
  font-weight: 700;
  font-size: calc(var(--size) * 0.42);
  line-height: 1;
}
.material-icon {
  position: relative;
  --icon-size: calc(var(--size) * 0.5);
}
</style>
