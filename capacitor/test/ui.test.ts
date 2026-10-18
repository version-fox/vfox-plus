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

// @vitest-environment happy-dom

// The modules of the user interface that do not render: the reactivity of the
// model, snack bars, dialogs and menus, transitions and the actions on vfox.
// Ionic's controllers are replaced by test doubles.

import { mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, effectScope, h, nextTick, ref } from 'vue';

const ionic = vi.hoisted(() => {
  const overlay = (data: unknown, role: string | undefined) => ({
    present: vi.fn(() => Promise.resolve()),
    dismiss: vi.fn(() => Promise.resolve(true)),
    onDidDismiss: vi.fn(() => Promise.resolve({ data, role })),
    onWillDismiss: vi.fn(() => Promise.resolve({ data, role })),
  });
  return {
    overlay,
    toastController: { create: vi.fn((_options: unknown) => Promise.resolve(overlay(undefined, undefined))) },
    modalController: {
      create: vi.fn((_options: unknown) => Promise.resolve(overlay(undefined, 'cancel'))),
      dismiss: vi.fn((_data?: unknown, _role?: string) => Promise.resolve(true)),
    },
    popoverController: {
      create: vi.fn((_options: unknown) => Promise.resolve(overlay(undefined, undefined))),
      dismiss: vi.fn((_data?: unknown) => Promise.resolve(true)),
    },
  };
});
vi.mock('@ionic/vue', async (original) => ({ ...(await original<object>()), ...ionic }));

import { VfoxError } from '@web/core/vfox/command';
import { ChangeNotifier } from '@web/state/notifier';
import { DESTINATIONS } from '@web/ui/context';
import { closeDialog, CONFIRM, whenPresented } from '../src/ui/dialogs/dialog';
import { symbolOf } from '../src/ui/icons';
import { NAVIGATION } from '../src/ui/navigation';
import { askCustomPlugin, askProjectDirectory, confirmAction, showMenu, showText } from '../src/ui/overlays';
import { t, taskTitle, watched } from '../src/ui/reactivity';
import { snackbar, snackbarButtons } from '../src/ui/snackbar';
import { applyTheme } from '../src/ui/theme';
import {
  dialogEnter,
  dialogLeave,
  durationOf,
  expand,
  fadeThroughEnter,
  fadeThroughLeave,
  sharedAxis,
} from '../src/ui/transitions';
import { useWidth } from '../src/ui/use-width';
import { locateVfox, saveConfig } from '../src/ui/vfox';
import { startFakeApp } from './support';

class Counter extends ChangeNotifier {
  count = 0;

  increment(): void {
    this.count++;
    this.notifyListeners();
  }
}

/** The options that `create` of a controller double got last. */
function lastOptions(create: { mock: { calls: unknown[][] } }): Record<string, unknown> {
  return create.mock.calls.at(-1)?.[0] as Record<string, unknown>;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('reactivity', () => {
  it('makes computed values follow the model', () => {
    const counter = new Counter();
    const scope = effectScope();
    const doubled = scope.run(() => computed(() => watched(counter).count * 2));
    expect(doubled?.value).toBe(0);
    counter.increment();
    expect(doubled?.value).toBe(2);
    expect(watched(counter)).toBe(counter);
    scope.stop();
  });

  it('translates in the current language and follows it', async () => {
    const app = startFakeApp();
    const scope = effectScope();
    const title = scope.run(() => computed(() => t('navSdks')));
    const task = scope.run(() =>
      computed(() =>
        taskTitle({
          operation: 'install',
          sdk: 'nodejs',
          version: '24.11.0',
          arguments: ['install', '--yes', 'nodejs@24.11.0'],
          workingDirectory: '/',
          requiresHookEnvironment: false,
        }),
      ),
    );
    expect(title?.value).toBe('SDKs');
    expect(task?.value).toBe('Install nodejs@24.11.0');
    app.locale.apply('simplifiedChinese');
    await nextTick();
    expect(task?.value).toBe('安装 nodejs@24.11.0');
    scope.stop();
  });
});

describe('snack bars', () => {
  it('offer an action and, when they stay, a close button', () => {
    startFakeApp();
    const onAction = vi.fn();
    const buttons = snackbarButtons({ action: 'View', onAction, persistent: true });
    expect(buttons.map((button) => button.role ?? button.text)).toEqual(['View', 'cancel']);
    expect(buttons[1]).toMatchObject({ text: 'close', htmlAttributes: { 'aria-label': 'Close' } });
    void buttons[0]?.handler?.();
    expect(onAction).toHaveBeenCalled();
    expect(snackbarButtons({})).toEqual([]);
  });

  it('replace each other', async () => {
    startFakeApp();
    snackbar('Saved');
    await vi.waitFor(() => {
      expect(ionic.toastController.create).toHaveBeenCalledTimes(1);
    });
    snackbar('Copied to the clipboard', { persistent: true });
    await vi.waitFor(() => {
      expect(ionic.toastController.create).toHaveBeenCalledTimes(2);
    });
    expect(lastOptions(ionic.toastController.create)).toMatchObject({
      message: 'Copied to the clipboard',
      duration: 0,
      cssClass: 'snackbar',
    });
    const first = (await ionic.toastController.create.mock.results[0]?.value) as
      ReturnType<typeof ionic.overlay> | undefined;
    await vi.waitFor(() => {
      expect(first?.dismiss).toHaveBeenCalled();
    });
  });
});

describe('dialogs and menus', () => {
  it('resolve with what the user confirmed', async () => {
    ionic.modalController.create.mockResolvedValueOnce(ionic.overlay(true, CONFIRM));
    expect(await confirmAction({ title: 'Remove?', body: '', confirmLabel: 'Remove', destructive: true })).toBe(true);
    expect(lastOptions(ionic.modalController.create)).toMatchObject({
      componentProps: { title: 'Remove?', destructive: true },
      cssClass: 'dialog',
    });

    ionic.modalController.create.mockResolvedValueOnce(ionic.overlay('/home/me/project', CONFIRM));
    expect(await askProjectDirectory('nodejs', '24.11.0')).toBe('/home/me/project');

    ionic.modalController.create.mockResolvedValueOnce(ionic.overlay({ source: 'a.lua', alias: null }, CONFIRM));
    expect(await askCustomPlugin()).toEqual({ source: 'a.lua', alias: null });
  });

  it('resolve with nothing when dismissed', async () => {
    expect(await confirmAction({ title: 'Remove?', body: '', confirmLabel: 'Remove' })).toBe(false);
    expect(await askCustomPlugin()).toBeNull();
    await showText('Licenses', 'MIT');
    expect(lastOptions(ionic.modalController.create)).toMatchObject({ componentProps: { heading: 'Licenses' } });
  });

  it('close with a value or without one', () => {
    closeDialog('/home/me');
    closeDialog();
    expect(ionic.modalController.dismiss.mock.calls).toEqual([
      ['/home/me', CONFIRM],
      [undefined, 'cancel'],
    ]);
  });

  it('focus their fields once presented', () => {
    const modal = document.createElement('ion-modal');
    const content = document.createElement('div');
    modal.append(content);
    const focus = vi.fn();
    whenPresented(content, focus);
    whenPresented(null, focus);
    modal.dispatchEvent(new Event('ionModalDidPresent'));
    modal.dispatchEvent(new Event('ionModalDidPresent'));
    expect(focus).toHaveBeenCalledTimes(1);
  });

  it('show menus at the element that was clicked', async () => {
    const event = new MouseEvent('click');
    ionic.popoverController.create.mockResolvedValueOnce(ionic.overlay('copy', undefined));
    expect(await showMenu(event, [{ id: 'copy', label: 'Copy installation path', icon: 'copy' }])).toBe('copy');
    expect(lastOptions(ionic.popoverController.create)).toMatchObject({
      event,
      side: 'bottom',
      alignment: 'end',
      showBackdrop: false,
    });
    expect(await showMenu(event, [])).toBeNull();
  });
});

describe('transitions', () => {
  let reduced = false;

  const animate = vi.fn((..._: unknown[]) => ({ onfinish: null, oncancel: null }) as unknown as Animation);

  beforeEach(() => {
    vi.stubGlobal('matchMedia', (query: string) => ({ matches: reduced && query.includes('reduce') }));
    Object.defineProperty(Element.prototype, 'animate', { value: animate, configurable: true, writable: true });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('take the time of their springs, or none with reduced motion', () => {
    expect(durationOf({ dampingRatio: 1, stiffness: 800 })).toBeGreaterThan(200);
    reduced = true;
    expect(durationOf({ dampingRatio: 1, stiffness: 800 })).toBe(0);
    reduced = false;
  });

  it('fade pages through and make the old one inert', () => {
    const leaving = document.createElement('div');
    const entering = document.createElement('div');
    const done = vi.fn();
    fadeThroughLeave(leaving, done);
    fadeThroughEnter(entering, done);
    expect(leaving.inert).toBe(true);
    expect(leaving.getAttribute('aria-hidden')).toBe('true');
    const animation = animate.mock.results[0]?.value as Animation;
    animation.onfinish?.(new Event('finish') as AnimationPlaybackEvent);
    expect(done).toHaveBeenCalledTimes(1);
    sharedAxis(entering, -1);
    expand(entering);
    expect(animate).toHaveBeenCalledTimes(4);
  });
});

describe('vfox', () => {
  it('reports how saving the configuration went', async () => {
    startFakeApp();
    await saveConfig(() => Promise.resolve());
    expect(lastOptions(ionic.toastController.create)).toMatchObject({ message: 'Saved' });
    await saveConfig(() =>
      Promise.reject(
        new VfoxError('vfox rejected the value', { arguments: ['config'], exitCode: 1, stdout: '', stderr: '' }),
      ),
    );
    await vi.waitFor(() => {
      expect(lastOptions(ionic.toastController.create)).toMatchObject({ message: 'vfox rejected the value' });
    });
  });

  it('stores the executable that the user chose', async () => {
    const pickFile = vi.fn().mockResolvedValueOnce('/opt/vfox/vfox').mockResolvedValueOnce(null);
    const app = startFakeApp({ pickFile });
    await locateVfox();
    expect(pickFile).toHaveBeenCalledWith('/usr/local/bin');
    expect(app.settings.vfoxExecutable).toBe('/opt/vfox/vfox');
    app.settings = {};
    await locateVfox();
    expect(app.settings).toEqual({});
  });
});

describe('dialog animations', () => {
  /** An element with the parts of an ion-modal that its animations move. */
  function modal(): HTMLElement {
    const element = document.createElement('div');
    element.attachShadow({ mode: 'open' }).innerHTML = '<ion-backdrop></ion-backdrop><div class="modal-wrapper"></div>';
    return element;
  }

  it('move the scrim and the dialog', () => {
    const entering = dialogEnter(modal());
    expect(entering.childAnimations).toHaveLength(2);
    expect(dialogLeave(modal()).childAnimations).toHaveLength(2);
  });

  it('need the parts of a modal', () => {
    expect(() => dialogEnter(document.createElement('div'))).toThrow('no backdrop');
  });
});

describe('theme', () => {
  it('applies the colours of Material 3 and Ionic and the springs', () => {
    applyTheme(0xfff28c1b, true, false);
    const style = document.documentElement.style;
    expect(style.colorScheme).toBe('dark');
    expect(style.getPropertyValue('--md-sys-color-primary')).toMatch(/^#[0-9a-f]{6}$/);
    expect(style.getPropertyValue('--ion-color-primary')).toBe(style.getPropertyValue('--md-sys-color-primary'));
    expect(style.getPropertyValue('--md-sys-motion-spring-fast-spatial')).toContain('linear(');
    applyTheme(0xfff28c1b, false, true);
    expect(style.colorScheme).toBe('light');
    expect(style.getPropertyValue('--md-sys-motion-spring-fast-spatial')).toBe('0ms linear');
  });
});

describe('navigation', () => {
  it('has an icon and a label for every destination', () => {
    startFakeApp();
    for (const destination of DESTINATIONS) {
      expect(symbolOf(NAVIGATION[destination].icon)).not.toBe('');
      expect(t(NAVIGATION[destination].label)).not.toBe(NAVIGATION[destination].label);
    }
  });
});

describe('useWidth', () => {
  it('follows the width of an element', async () => {
    let observe: ResizeObserverCallback = () => undefined;
    const disconnect = vi.fn();
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(callback: ResizeObserverCallback) {
          observe = callback;
        }
        observe = vi.fn();
        disconnect = disconnect;
      },
    );
    const Probe = defineComponent({
      setup() {
        const element = ref<HTMLElement | null>(null);
        const width = useWidth(element, 100);
        return () => h('div', { ref: element }, String(width.value));
      },
    });
    const wrapper = mount(Probe);
    expect(wrapper.text()).toBe('100');
    const resize = (width: number): void => {
      observe([{ contentRect: { width } } as ResizeObserverEntry], {} as ResizeObserver);
    };
    resize(640);
    await nextTick();
    expect(wrapper.text()).toBe('640');
    // Scroll bars that come and go do not count.
    resize(640.5);
    resize(Number.NaN);
    await nextTick();
    expect(wrapper.text()).toBe('640');
    wrapper.unmount();
    expect(disconnect).toHaveBeenCalled();
    vi.unstubAllGlobals();
  });
});
