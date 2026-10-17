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

//! The Material 3 Expressive motion system: physics-based springs.
//!
//! The springs of `docs/design-system.md#motion` are simulated here and
//! sampled into CSS `linear()` easing functions over the time they need to
//! settle, exactly as the web front end does. They replace the cubic-bézier
//! approximations of `@m3e/web`, so every component of the library moves on
//! the springs of the specification, and the transitions of the application
//! use them through the same custom properties.

/// A damped spring with unit mass.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Spring {
    /// 1 is critically damped (no overshoot); lower values bounce.
    pub damping_ratio: f64,
    /// Higher values settle faster.
    pub stiffness: f64,
}

/// The six springs of a Material 3 motion scheme.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct MotionScheme {
    /// Small components that move, resize or reshape.
    pub fast_spatial: Spring,
    /// Partial-screen spatial changes.
    pub default_spatial: Spring,
    /// Full-screen spatial changes.
    pub slow_spatial: Spring,
    /// Colour and opacity of small components.
    pub fast_effects: Spring,
    /// Colour and opacity of partial-screen changes.
    pub default_effects: Spring,
    /// Colour and opacity of full-screen changes.
    pub slow_effects: Spring,
}

/// The expressive scheme, which vfox-plus uses.
pub const EXPRESSIVE: MotionScheme = MotionScheme {
    fast_spatial: Spring { damping_ratio: 0.6, stiffness: 800.0 },
    default_spatial: Spring { damping_ratio: 0.8, stiffness: 380.0 },
    slow_spatial: Spring { damping_ratio: 0.8, stiffness: 200.0 },
    fast_effects: Spring { damping_ratio: 1.0, stiffness: 3800.0 },
    default_effects: Spring { damping_ratio: 1.0, stiffness: 1600.0 },
    slow_effects: Spring { damping_ratio: 1.0, stiffness: 800.0 },
};

const DISTANCE_TOLERANCE: f64 = 0.001;
const VELOCITY_TOLERANCE: f64 = 0.01;

/// The position and velocity of a spring released at `start` with `velocity`
/// towards `end`, after `time` seconds.
pub fn spring_state(spring: Spring, start: f64, end: f64, velocity: f64, time: f64) -> (f64, f64) {
    let omega = spring.stiffness.sqrt();
    let zeta = spring.damping_ratio;
    let x0 = start - end;
    if zeta < 1.0 {
        let omega_d = omega * (1.0 - zeta * zeta).sqrt();
        let decay = (-zeta * omega * time).exp();
        let a = x0;
        let b = (velocity + zeta * omega * x0) / omega_d;
        let (sin, cos) = (omega_d * time).sin_cos();
        let x = decay * (a * cos + b * sin);
        let v = decay * ((b * omega_d - zeta * omega * a) * cos - (a * omega_d + zeta * omega * b) * sin);
        return (end + x, v);
    }
    // Critically damped.
    let decay = (-omega * time).exp();
    let b = velocity + omega * x0;
    let x = (x0 + b * time) * decay;
    let v = (b - omega * (x0 + b * time)) * decay;
    (end + x, v)
}

fn at_rest((position, velocity): (f64, f64), end: f64) -> bool {
    (position - end).abs() < DISTANCE_TOLERANCE && velocity.abs() < VELOCITY_TOLERANCE
}

/// The time, in milliseconds, a spring needs to travel from 0 to 1 and settle.
pub fn settle_duration(spring: Spring) -> u32 {
    let mut time = 0.0;
    while time < 5.0 && !at_rest(spring_state(spring, 0.0, 1.0, 0.0, time), 1.0) {
        time += 1.0 / 240.0;
    }
    // At most 5000.
    #[allow(clippy::cast_possible_truncation, clippy::cast_sign_loss)]
    let milliseconds = (time * 1000.0).round() as u32;
    milliseconds
}

/// `value` as JavaScript prints `Number(value.toFixed(4))`.
fn format_sample(value: f64) -> String {
    let rounded = (value * 10_000.0).round() / 10_000.0;
    // Negative zero prints as 0.
    if rounded == 0.0 { "0".to_owned() } else { rounded.to_string() }
}

/// The spring as a CSS easing function over [`settle_duration`], sampled at
/// `samples` points.
pub fn spring_easing(spring: Spring, samples: u32) -> String {
    let seconds = f64::from(settle_duration(spring)) / 1000.0;
    let points: Vec<String> = (0..=samples)
        .map(|i| {
            let t = f64::from(i) / f64::from(samples);
            let value = if i == samples { 1.0 } else { spring_state(spring, 0.0, 1.0, 0.0, t * seconds).0 };
            format_sample(value)
        })
        .collect();
    format!("linear({})", points.join(", "))
}

/// A spring as a CSS transition timing: `<duration> <easing>`.
pub fn spring_transition(spring: Spring) -> String {
    format!("{}ms {}", settle_duration(spring), spring_easing(spring, 40))
}

/// The CSS custom properties of `@m3e/web` for the springs of `scheme`, and
/// those of the transitions of the application (`--vfox-*`). With reduced
/// motion, transitions take no time.
pub fn motion_tokens(scheme: &MotionScheme, reduced_motion: bool) -> Vec<(String, String)> {
    let springs = [
        ("fast-spatial", scheme.fast_spatial),
        ("default-spatial", scheme.default_spatial),
        ("slow-spatial", scheme.slow_spatial),
        ("fast-effects", scheme.fast_effects),
        ("default-effects", scheme.default_effects),
        ("slow-effects", scheme.slow_effects),
    ];
    let mut tokens = Vec::new();
    for (name, spring) in springs {
        let transition = if reduced_motion { "0ms linear".to_owned() } else { spring_transition(spring) };
        tokens.push((format!("--md-sys-motion-spring-{name}"), transition));
        let (duration, easing) = if reduced_motion {
            (0, "linear".to_owned())
        } else {
            (settle_duration(spring), spring_easing(spring, 40))
        };
        tokens.push((format!("--vfox-{name}-duration"), format!("{duration}ms")));
        tokens.push((format!("--vfox-{name}-easing"), easing));
    }
    tokens
}

#[cfg(test)]
mod tests {
    use super::*;

    /// The values of `web/src/ui/motion.ts` for the same springs.
    #[test]
    fn springs_settle_and_ease_like_the_web_front_end() {
        let durations: Vec<u32> = [
            EXPRESSIVE.fast_spatial,
            EXPRESSIVE.default_spatial,
            EXPRESSIVE.slow_spatial,
            EXPRESSIVE.fast_effects,
            EXPRESSIVE.default_effects,
            EXPRESSIVE.slow_effects,
        ]
        .into_iter()
        .map(settle_duration)
        .collect();
        assert_eq!(durations, WEB_DURATIONS);
        assert_eq!(spring_easing(EXPRESSIVE.fast_spatial, 40), WEB_FAST_SPATIAL_EASING);
        assert_eq!(format_sample(-0.000_01), "0");
        assert_eq!(format_sample(1.0), "1");
    }

    #[test]
    fn reduced_motion_takes_no_time() {
        let tokens = motion_tokens(&EXPRESSIVE, true);
        assert!(
            tokens.iter().any(|(name, value)| name == "--md-sys-motion-spring-fast-spatial" && value == "0ms linear")
        );
        assert!(tokens.iter().any(|(name, value)| name == "--vfox-slow-effects-duration" && value == "0ms"));
        let tokens = motion_tokens(&EXPRESSIVE, false);
        assert_eq!(tokens.len(), 18);
        let (_, value) = tokens.iter().find(|(name, _)| name == "--md-sys-motion-spring-fast-spatial").unwrap();
        assert!(value.starts_with(&format!("{}ms linear(0, ", WEB_DURATIONS[0])));
    }

    const WEB_DURATIONS: [u32; 6] = [408, 487, 654, 183, 267, 367];
    const WEB_FAST_SPATIAL_EASING: &str = "linear(0, 0.037, 0.1304, 0.2572, 0.3988, 0.541, 0.6737, 0.7903, 0.8874, \
        0.964, 1.0207, 1.0594, 1.0826, 1.0932, 1.0942, 1.0883, 1.078, 1.0653, 1.0517, 1.0385, 1.0264, 1.0161, 1.0076, \
        1.0011, 0.9964, 0.9934, 0.9917, 0.991, 0.9912, 0.992, 0.9931, 0.9943, 0.9956, 0.9968, 0.9979, 0.9988, 0.9995, \
        1.0001, 1.0005, 1.0007, 1)";
}
