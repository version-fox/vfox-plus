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
 * The transitions of the user interface, on the springs of the expressive
 * motion scheme (../web/src/ui/motion.ts): fade through between pages, the
 * horizontal shared axis between list and detail, and the dialogs, as
 * animations of Ionic.
 */

import { createAnimation, type Animation } from '@ionic/vue';

import { EXPRESSIVE, prefersReducedMotion, settleDuration, springEasing, type Spring } from '@web/ui/motion';

/** How long `spring` takes, or no time with reduced motion. */
export function durationOf(spring: Spring): number {
  return prefersReducedMotion() ? 0 : settleDuration(spring);
}

/** Calls `done` once `animation` has ended, however it ended. */
function whenDone(animation: globalThis.Animation, done: () => void): void {
  animation.onfinish = done;
  animation.oncancel = done;
}

/**
 * The page that fades out of a fade through: it fades out during the first
 * 30 % of the transition and can no longer be used.
 */
export function fadeThroughLeave(element: Element, done: () => void): void {
  const duration = durationOf(EXPRESSIVE.slowEffects);
  (element as HTMLElement).inert = true;
  element.setAttribute('aria-hidden', 'true');
  whenDone(element.animate([{ opacity: 1 }, { opacity: 0 }], { duration: duration * 0.3, fill: 'forwards' }), done);
}

/** The page that fades in afterwards, growing from 92 %. */
export function fadeThroughEnter(element: Element, done: () => void): void {
  const duration = durationOf(EXPRESSIVE.slowEffects);
  whenDone(
    element.animate(
      [
        { opacity: 0, transform: 'scale(0.92)' },
        { opacity: 1, transform: 'none' },
      ],
      {
        duration: duration * 0.7,
        delay: duration * 0.3,
        easing: springEasing(EXPRESSIVE.slowEffects),
        fill: 'backwards',
      },
    ),
    done,
  );
}

/** The incoming side of the horizontal shared axis: forwards (1) or backwards (-1). */
export function sharedAxis(element: Element, direction: 1 | -1): void {
  element.animate(
    [
      { opacity: 0, transform: `translateX(${30 * direction}px)` },
      { opacity: 1, transform: 'none' },
    ],
    { duration: durationOf(EXPRESSIVE.slowEffects), easing: springEasing(EXPRESSIVE.slowEffects) },
  );
}

/** Grows `element` from no height to its height, as an expanding section does. */
export function expand(element: HTMLElement): void {
  element.animate([{ height: '0px' }, { height: `${element.scrollHeight}px` }], {
    duration: durationOf(EXPRESSIVE.defaultSpatial),
    easing: springEasing(EXPRESSIVE.defaultSpatial),
  });
}

/** The parts of an `ion-modal` that its animations move. */
function modalParts(base: HTMLElement): { backdrop: Element; wrapper: Element } {
  const root = base.shadowRoot ?? base;
  const backdrop = root.querySelector('ion-backdrop');
  const wrapper = root.querySelector('.modal-wrapper');
  if (backdrop === null || wrapper === null) {
    throw new Error('The modal has no backdrop or wrapper.');
  }
  return { backdrop, wrapper };
}

/** The scrim fades in and the dialog grows into place, as Material 3 dialogs do. */
export function dialogEnter(base: HTMLElement): Animation {
  const { backdrop, wrapper } = modalParts(base);
  return createAnimation()
    .addElement(base)
    .addAnimation([
      createAnimation()
        .addElement(backdrop)
        .duration(durationOf(EXPRESSIVE.defaultEffects))
        .easing(springEasing(EXPRESSIVE.defaultEffects))
        .fromTo('opacity', '0.01', 'var(--backdrop-opacity)')
        .beforeStyles({ 'pointer-events': 'none' })
        .afterClearStyles(['pointer-events']),
      createAnimation()
        .addElement(wrapper)
        .duration(durationOf(EXPRESSIVE.defaultSpatial))
        .easing(springEasing(EXPRESSIVE.defaultSpatial))
        .keyframes([
          { offset: 0, opacity: '0', transform: 'translateY(-16px) scale(0.92)' },
          { offset: 0.3, opacity: '1' },
          { offset: 1, opacity: '1', transform: 'translateY(0) scale(1)' },
        ]),
    ]);
}

/** The dialog and the scrim fade out. */
export function dialogLeave(base: HTMLElement): Animation {
  const { backdrop, wrapper } = modalParts(base);
  return createAnimation()
    .addElement(base)
    .duration(durationOf(EXPRESSIVE.fastEffects))
    .easing(springEasing(EXPRESSIVE.fastEffects))
    .addAnimation([
      createAnimation().addElement(backdrop).fromTo('opacity', 'var(--backdrop-opacity)', '0'),
      createAnimation()
        .addElement(wrapper)
        .keyframes([
          { offset: 0, opacity: '1', transform: 'scale(1)' },
          { offset: 1, opacity: '0', transform: 'scale(0.96)' },
        ]),
    ]);
}
