// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue';

/**
 * The width of the element in `target`, in pixels, which pages use to adapt
 * their layout to the space they have; `initial` until it is laid out.
 */
export function useWidth(target: Ref<{ $el?: Element } | Element | null>, initial: number): Ref<number> {
  const width = ref(initial);
  const observer = new ResizeObserver((entries) => {
    const next = entries[0]?.contentRect.width ?? width.value;
    // Scroll bars that come and go must not make the layout flicker.
    if (Math.abs(next - width.value) > 1) {
      width.value = next;
    }
  });
  onMounted(() => {
    const value = target.value;
    const element = value instanceof Element ? value : value?.$el;
    if (element instanceof Element) {
      observer.observe(element);
    }
  });
  onBeforeUnmount(() => {
    observer.disconnect();
  });
  return width;
}
