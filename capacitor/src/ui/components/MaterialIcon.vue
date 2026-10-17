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
  A Material Symbol of the icon font, by meaning. The ligature is generated
  content, so that the name of the symbol is not part of the text of the
  element that contains the icon.
-->
<script setup lang="ts">
import { symbolOf, type IconName } from '../icons';

defineProps<{
  name: IconName;
  /** Whether to draw the filled variant of the symbol. */
  filled?: boolean;
}>();
</script>

<template>
  <span class="material-icon" :class="{ filled }" :data-symbol="symbolOf(name)" aria-hidden="true"></span>
</template>
