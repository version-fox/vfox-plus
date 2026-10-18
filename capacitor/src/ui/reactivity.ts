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

/**
 * Connects the application model of ../web/src/state, which notifies its
 * listeners like Flutter's `ChangeNotifier`, to the reactivity of Vue. The
 * model is not made reactive itself: reading an object of the model through
 * {@link watched} makes whatever reads it, a render, a slot or a computed
 * value, depend on the object, which triggers it whenever it notifies.
 */

import { customRef, type Ref } from 'vue';

import type { VfoxCommand } from '@web/core/vfox/command';
import type { MessageArguments } from '@web/core/i18n/message-format';
import type { Listenable } from '@web/state/notifier';
import { app, t as translate, taskTitle as titleOf } from '@web/ui/context';

const trackers = new WeakMap<Listenable, Ref<Listenable>>();

/** Returns `listenable` and makes the current reader depend on it. */
export function watched<T extends Listenable>(listenable: T): T {
  let tracker = trackers.get(listenable);
  if (tracker === undefined) {
    tracker = customRef((track, trigger) => {
      // The listener lives as long as the object: disposing it removes its listeners.
      listenable.addListener(trigger);
      return {
        get: () => {
          track();
          return listenable;
        },
        set: () => undefined,
      };
    });
    trackers.set(listenable, tracker);
  }
  return tracker.value as T;
}

/** The message `key` of the current language, formatted with `args`; follows the language. */
export function t(key: string, args: MessageArguments = {}): string {
  watched(app().locale);
  return translate(key, args);
}

/** The localized title of a task that runs `command`; follows the language. */
export function taskTitle(command: VfoxCommand): string {
  watched(app().locale);
  return titleOf(command);
}
