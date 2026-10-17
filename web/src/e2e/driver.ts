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
 * Drives the user interface from inside the page, like a user: finds visible
 * elements by their text or their accessible name, through every shadow
 * root, clicks them and types into fields.
 */

/** Every element below `root`, including those in shadow roots. */
function* allElements(root: Document | ShadowRoot | Element): Generator<Element> {
  for (const child of Array.from(root.children)) {
    yield child;
    if (child.shadowRoot !== null) {
      yield* allElements(child.shadowRoot);
    }
    yield* allElements(child);
  }
}

/**
 * Where a user can interact: the open modal dialogs, if there are any, which
 * make everything else inert; otherwise the whole document.
 */
function* interactiveElements(): Generator<Element> {
  const dialogs = [...allElements(document)].filter(
    (element) => element.tagName === 'M3E-DIALOG' && element.hasAttribute('open'),
  );
  if (dialogs.length === 0) {
    yield* allElements(document);
    return;
  }
  for (const dialog of dialogs) {
    yield dialog;
    yield* allElements(dialog);
  }
}

/** The parent of `element`, crossing shadow root boundaries. */
function parentOf(element: Element): Element | null {
  if (element.parentElement !== null) {
    return element.parentElement;
  }
  const root = element.getRootNode();
  return root instanceof ShadowRoot ? root.host : null;
}

/** Whether `element` is shown and not part of a page that is fading out. */
export function isVisible(element: Element): boolean {
  for (let current: Element | null = element; current !== null; current = parentOf(current)) {
    if (current.hasAttribute('inert') || current.getAttribute('aria-hidden') === 'true') {
      // Icons hide themselves from assistive technologies, but are visible.
      if (current !== element || current.tagName !== 'M3E-ICON') {
        return false;
      }
    }
  }
  const rectangles = element.getClientRects();
  if (rectangles.length === 0) {
    return false;
  }
  const style = getComputedStyle(element);
  return style.visibility !== 'hidden' && style.display !== 'none' && Number(style.opacity) > 0;
}

/** The text of the light DOM of `element`, with collapsed white space. */
export function ownText(element: Element): string {
  return element.textContent.replace(/\s+/g, ' ').trim();
}

/** The accessible name of `element`: its label, or its text. */
function nameOf(element: Element): string {
  return element.getAttribute('aria-label') ?? element.getAttribute('label') ?? ownText(element);
}

const INTERACTIVE = new Set([
  'BUTTON',
  'A',
  'M3E-BUTTON',
  'M3E-ICON-BUTTON',
  'M3E-FAB',
  'M3E-NAV-ITEM',
  'M3E-LIST-ACTION',
  'M3E-MENU-ITEM',
  'M3E-SWITCH',
]);

/** Visible elements, optionally limited to those matching `selector`. */
export function visibleElements(selector?: string): Element[] {
  return [...interactiveElements()].filter(
    (element) => (selector === undefined || element.matches(selector)) && isVisible(element),
  );
}

/** The innermost visible elements whose text is exactly `text`. */
export function findText(text: string): Element[] {
  const matches = [...interactiveElements()].filter(
    (element) => element.tagName !== 'STYLE' && element.tagName !== 'SCRIPT' && ownText(element) === text,
  );
  // Prefer the innermost element: a parent has the same text as its only child.
  return matches
    .filter((element) => !matches.some((other) => other !== element && element.contains(other)))
    .filter(isVisible);
}

/** Visible interactive elements whose accessible name is exactly `name`. */
export function findButtons(name: string): Element[] {
  return [...interactiveElements()].filter(
    (element) => INTERACTIVE.has(element.tagName) && nameOf(element) === name && isVisible(element),
  );
}

/** Waits until `condition` returns a value other than `null`, `undefined` or `false`. */
export async function waitFor<T>(
  description: string,
  condition: () => T | null | undefined | false,
  timeout = 30_000,
): Promise<T> {
  const end = Date.now() + timeout;
  for (;;) {
    const value = condition();
    if (value !== null && value !== undefined && value !== false) {
      return value;
    }
    if (Date.now() > end) {
      throw new Error(`Timed out waiting for ${description}`);
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
}

/** Waits until `text` is shown, and returns its innermost element. */
export function waitForText(text: string, timeout?: number): Promise<Element> {
  return waitFor(`the text "${text}"`, () => findText(text)[0], timeout);
}

/** Waits until a button named `name` is shown and clicks it; `index` picks among several. */
export async function clickButton(name: string, index = 0): Promise<void> {
  const button = await waitFor(`a button "${name}"`, () => findButtons(name)[index]).catch((error: unknown) => {
    const shown = [...interactiveElements()]
      .filter((element) => INTERACTIVE.has(element.tagName) && isVisible(element))
      .map((element) => `"${nameOf(element)}"`);
    throw new Error(`${error instanceof Error ? error.message : String(error)}; the buttons are ${shown.join(', ')}`);
  });
  (button as HTMLElement).click();
}

/** Waits until `text` is shown and clicks it. */
export async function clickText(text: string): Promise<void> {
  const element = await waitForText(text);
  (element as HTMLElement).click();
}

/** Clicks the first visible element matching `selector` and `filter`. */
export async function clickElement(
  description: string,
  selector: string,
  filter: (element: Element) => boolean = () => true,
): Promise<void> {
  const element = await waitFor(description, () => visibleElements(selector).find(filter));
  (element as HTMLElement).click();
}

/** Types `value` into the visible input labelled or described by `label`. */
export async function fill(label: string, value: string, { submit = false } = {}): Promise<void> {
  const input = await waitFor(`a field "${label}"`, () =>
    visibleElements('input, textarea').find((element) => {
      const field = element as HTMLInputElement;
      if (field.getAttribute('aria-label') === label || field.placeholder === label) {
        return true;
      }
      const root = field.getRootNode() as Document | ShadowRoot;
      const labels = field.id === '' ? [] : Array.from(root.querySelectorAll(`label[for="${field.id}"]`));
      return labels.some((element) => ownText(element) === label);
    }),
  );
  const field = input as HTMLInputElement;
  field.focus();
  field.value = value;
  field.dispatchEvent(new Event('input', { bubbles: true, composed: true }));
  if (submit) {
    field.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, composed: true }));
  }
  field.dispatchEvent(new Event('change', { bubbles: true, composed: true }));
}

/** Fails with `message` unless `condition` holds. */
export function check(condition: boolean, message: string): void {
  if (!condition) {
    throw new Error(`Check failed: ${message}`);
  }
}

/** Waits `milliseconds`. */
export function pause(milliseconds: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
