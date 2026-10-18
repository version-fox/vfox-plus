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
 * The Material 3 Expressive motion system: physics-based springs.
 *
 * The springs of `docs/design-system.md#motion` are simulated here and sampled
 * into CSS `linear()` easing functions over the time they need to settle.
 * They replace the cubic-bézier approximations of `@m3e/web`, so every
 * component of the library moves on the springs of the specification. CSS
 * transitions that are interrupted restart from where they are, without the
 * velocity a spring would keep; animations driven by script use
 * {@link SpringAnimation}, which keeps it.
 */

/** A damped spring with unit mass. */
export interface Spring {
  /** 1 is critically damped (no overshoot); lower values bounce. */
  readonly dampingRatio: number;
  /** Higher values settle faster. */
  readonly stiffness: number;
}

/** The six springs of a Material 3 motion scheme. */
export interface MotionScheme {
  readonly fastSpatial: Spring;
  readonly defaultSpatial: Spring;
  readonly slowSpatial: Spring;
  readonly fastEffects: Spring;
  readonly defaultEffects: Spring;
  readonly slowEffects: Spring;
}

/** The expressive scheme, which vfox-plus uses. */
export const EXPRESSIVE: MotionScheme = {
  fastSpatial: { dampingRatio: 0.6, stiffness: 800 },
  defaultSpatial: { dampingRatio: 0.8, stiffness: 380 },
  slowSpatial: { dampingRatio: 0.8, stiffness: 200 },
  fastEffects: { dampingRatio: 1, stiffness: 3800 },
  defaultEffects: { dampingRatio: 1, stiffness: 1600 },
  slowEffects: { dampingRatio: 1, stiffness: 800 },
};

/** The spring of each morph of the loading indicator. */
export const LOADING_MORPH: Spring = { dampingRatio: 0.6, stiffness: 200 };

const DISTANCE_TOLERANCE = 0.001;
const VELOCITY_TOLERANCE = 0.01;

/**
 * The position and velocity of a spring released at `start` with `velocity`
 * towards `end`, after `time` seconds.
 */
export function springState(
  spring: Spring,
  start: number,
  end: number,
  velocity: number,
  time: number,
): { position: number; velocity: number } {
  const omega = Math.sqrt(spring.stiffness);
  const zeta = spring.dampingRatio;
  const x0 = start - end;
  if (zeta < 1) {
    const omegaD = omega * Math.sqrt(1 - zeta * zeta);
    const decay = Math.exp(-zeta * omega * time);
    const a = x0;
    const b = (velocity + zeta * omega * x0) / omegaD;
    const cos = Math.cos(omegaD * time);
    const sin = Math.sin(omegaD * time);
    const x = decay * (a * cos + b * sin);
    const v = decay * ((b * omegaD - zeta * omega * a) * cos - (a * omegaD + zeta * omega * b) * sin);
    return { position: end + x, velocity: v };
  }
  // Critically damped.
  const decay = Math.exp(-omega * time);
  const b = velocity + omega * x0;
  const x = (x0 + b * time) * decay;
  const v = (b - omega * (x0 + b * time)) * decay;
  return { position: end + x, velocity: v };
}

/** Whether a spring is at rest at `state`, moving towards `end`. */
function atRest(state: { position: number; velocity: number }, end: number): boolean {
  return Math.abs(state.position - end) < DISTANCE_TOLERANCE && Math.abs(state.velocity) < VELOCITY_TOLERANCE;
}

const settleCache = new Map<string, number>();

/** The time, in milliseconds, a spring needs to travel from 0 to 1 and settle. */
export function settleDuration(spring: Spring): number {
  const key = `${spring.dampingRatio}/${spring.stiffness}`;
  let duration = settleCache.get(key);
  if (duration === undefined) {
    let time = 0;
    while (time < 5 && !atRest(springState(spring, 0, 1, 0, time), 1)) {
      time += 1 / 240;
    }
    duration = Math.round(time * 1000);
    settleCache.set(key, duration);
  }
  return duration;
}

/**
 * The spring as a CSS easing function over {@link settleDuration}, sampled
 * at `samples` points.
 */
export function springEasing(spring: Spring, samples = 40): string {
  const seconds = settleDuration(spring) / 1000;
  const points: string[] = [];
  for (let i = 0; i <= samples; i++) {
    const t = i / samples;
    const value = i === samples ? 1 : springState(spring, 0, 1, 0, t * seconds).position;
    points.push(Number(value.toFixed(4)).toString());
  }
  return `linear(${points.join(', ')})`;
}

/** A spring as a CSS transition timing: `<duration> <easing>`. */
export function springTransition(spring: Spring): string {
  return `${settleDuration(spring)}ms ${springEasing(spring)}`;
}

/**
 * The CSS custom properties of `@m3e/web` for the springs of `scheme`. With
 * reduced motion, transitions take no time.
 */
export function motionTokens(scheme: MotionScheme, reducedMotion: boolean): Record<string, string> {
  const tokens: Record<string, string> = {};
  const names: (keyof MotionScheme)[] = [
    'fastSpatial',
    'defaultSpatial',
    'slowSpatial',
    'fastEffects',
    'defaultEffects',
    'slowEffects',
  ];
  for (const name of names) {
    const property = `--md-sys-motion-spring-${name.replace(/[A-Z]/g, (c) => `-${c.toLowerCase()}`)}`;
    tokens[property] = reducedMotion ? '0ms linear' : springTransition(scheme[name]);
  }
  return tokens;
}

/**
 * Animates a number towards a target on a spring, keeping its velocity when
 * the target changes mid-flight.
 */
export class SpringAnimation {
  private position: number;
  private velocity = 0;
  private start = 0;
  private from = 0;
  private target: number;
  private frame = 0;

  constructor(
    initial: number,
    private readonly spring: Spring,
    private readonly apply: (value: number) => void,
    private readonly reducedMotion: () => boolean = () => false,
  ) {
    this.position = initial;
    this.target = initial;
    apply(initial);
  }

  /** The current value. */
  get value(): number {
    return this.position;
  }

  /** Moves towards `target`. */
  animateTo(target: number): void {
    if (target === this.target && this.frame === 0) {
      return;
    }
    if (this.reducedMotion()) {
      this.stop();
      this.target = target;
      this.position = target;
      this.velocity = 0;
      this.apply(target);
      return;
    }
    this.target = target;
    this.from = this.position;
    this.start = performance.now();
    if (this.frame === 0) {
      this.frame = requestAnimationFrame((now) => {
        this.tick(now);
      });
    }
  }

  /** Stops where it is. */
  stop(): void {
    if (this.frame !== 0) {
      cancelAnimationFrame(this.frame);
      this.frame = 0;
    }
  }

  private tick(now: number): void {
    const state = springState(this.spring, this.from, this.target, this.velocity, (now - this.start) / 1000);
    if (atRest(state, this.target)) {
      this.position = this.target;
      this.velocity = 0;
      this.frame = 0;
      this.apply(this.position);
      return;
    }
    this.position = state.position;
    this.apply(this.position);
    // Restart the simulation from the current state on every frame, so that a
    // new target continues with the current velocity.
    this.velocity = state.velocity;
    this.from = state.position;
    this.start = now;
    this.frame = requestAnimationFrame((next) => {
      this.tick(next);
    });
  }
}

/** Whether the platform asks for reduced motion. */
export function prefersReducedMotion(): boolean {
  return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
}
