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

// The web components of @m3e/web that the user interface uses, and the
// helpers of window.vfoxPlus that its Rust code calls (src/ui/bridge.rs).
// build.mjs bundles them into assets/components.js.

import '@m3e/web/button';
import '@m3e/web/button-group';
import '@m3e/web/dialog';
import '@m3e/web/fab';
import '@m3e/web/form-field';
import '@m3e/web/icon';
import '@m3e/web/icon-button';
import '@m3e/web/list';
import '@m3e/web/loading-indicator';
import '@m3e/web/menu';
import '@m3e/web/nav-bar';
import '@m3e/web/nav-rail';
import '@m3e/web/option';
import '@m3e/web/progress-indicator';
import '@m3e/web/search';
import '@m3e/web/select';
import '@m3e/web/shape';
import '@m3e/web/split-button';
import '@m3e/web/switch';
import '@m3e/web/theme';
import '@m3e/web/tooltip';
import { M3eSnackbar } from '@m3e/web/snackbar';

// Elements marked data-controlled, such as the buttons of a choice group and
// switches, never change their own state: the application renders it. They
// announce a change with a cancelable beforeinput event.
document.addEventListener(
  'beforeinput',
  (event) => {
    if (event.target instanceof HTMLElement && event.target.hasAttribute('data-controlled')) {
      event.preventDefault();
    }
  },
  true,
);

/** The closest ancestor of `element` that scrolls. */
function scrollParent(element) {
  for (let current = element.parentElement; current !== null; current = current.parentElement) {
    const overflow = getComputedStyle(current).overflowY;
    if (overflow === 'auto' || overflow === 'scroll') {
      return current;
    }
  }
  return document.scrollingElement ?? document.documentElement;
}

window.vfoxPlus = {
  /**
   * Shows a snack bar, replacing the current one. Resolves with whether the
   * user chose its action, once it has closed.
   */
  snackbar(message, { action = null, persistent = false, closeLabel = 'Close' } = {}) {
    return new Promise((resolve) => {
      const options = { duration: persistent ? 0 : 4000, closeLabel, actionCallback: () => resolve(true) };
      if (action === null) {
        M3eSnackbar.open(message, persistent, options);
      } else {
        M3eSnackbar.open(message, action, persistent, options);
      }
      customElements.get('m3e-snackbar')?.current?.addEventListener('toggle', (event) => {
        if (event.newState === 'closed') {
          // After the action callback, which settles the promise first.
          setTimeout(() => resolve(false));
        }
      });
    });
  },

  /** Resolves once the dialog with the identifier `id` has closed. */
  closed(id) {
    return new Promise((resolve) => {
      const dialog = document.getElementById(id);
      if (dialog === null) {
        resolve(true);
      } else {
        dialog.addEventListener('closed', () => resolve(true), { once: true });
      }
    });
  },

  /** Writes the colour and motion tokens, the colour scheme, the language and the title. */
  applyTheme(properties, dark, language, title) {
    const root = document.documentElement;
    for (const [name, value] of Object.entries(properties)) {
      root.style.setProperty(name, value);
    }
    root.style.colorScheme = dark ? 'dark' : 'light';
    root.lang = language;
    document.title = title;
  },

  /** Calls `callback` with whether the media `query` matches, now and on every change. */
  watchMedia(query, callback) {
    const list = matchMedia(query);
    callback(list.matches);
    list.addEventListener('change', () => callback(list.matches));
  },

  /**
   * Calls `callback` with the range [start, end) of the rows of the list with
   * the identifier `id` that are in view of its scrolling ancestor, with
   * `overscan` rows on either side, whenever it changes. The rows are `pitch`
   * pixels apart. Stops once the list has left the document.
   */
  visibleRows(id, pitch, overscan, callback) {
    const list = document.getElementById(id);
    if (list === null) {
      return;
    }
    const scroller = scrollParent(list);
    let last = '';
    let frame = 0;
    const update = () => {
      frame = 0;
      if (!list.isConnected) {
        stop();
        return;
      }
      const top = list.getBoundingClientRect().top - scroller.getBoundingClientRect().top;
      const start = Math.max(0, Math.floor(-top / pitch) - overscan);
      const end = Math.max(start, Math.ceil((scroller.clientHeight - top) / pitch) + overscan);
      const key = `${start}:${end}`;
      if (key !== last) {
        last = key;
        callback([start, end]);
      }
    };
    const schedule = () => {
      if (frame === 0) {
        frame = requestAnimationFrame(update);
      }
    };
    const observer = new ResizeObserver(schedule);
    const stop = () => {
      scroller.removeEventListener('scroll', schedule);
      observer.disconnect();
    };
    scroller.addEventListener('scroll', schedule, { passive: true });
    observer.observe(scroller);
    observer.observe(list);
    update();
  },

  /** Scrolls the element with the identifier `id` to its end. */
  scrollToEnd(id) {
    const element = document.getElementById(id);
    if (element !== null) {
      element.scrollTop = element.scrollHeight;
    }
  },

  /** The value of the element with the identifier `id`, such as an m3e-select. */
  valueOf(id) {
    return String(document.getElementById(id)?.value ?? '');
  },
};
