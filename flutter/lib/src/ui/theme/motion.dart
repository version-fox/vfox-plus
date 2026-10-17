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

/// The Material 3 Expressive motion system: physics-based springs.
///
/// Material 3 Expressive replaces easing curves and durations with springs.
/// Spatial springs move, resize and reshape things and may overshoot;
/// effects springs change colour and opacity and never overshoot. See
/// https://m3.material.io/styles/motion/overview/how-it-works and
/// `docs/design-system.md#motion`.
library;

import 'package:flutter/physics.dart';
import 'package:material_ui/material_ui.dart';

/// A critically or under-damped spring with unit mass.
@immutable
final class Spring {
  /// Creates a spring from its damping ratio and stiffness.
  const Spring({required this.dampingRatio, required this.stiffness});

  /// 1 is critically damped (no overshoot); lower values bounce.
  final double dampingRatio;

  /// Higher values settle faster.
  final double stiffness;

  /// The spring as a Flutter physics description.
  SpringDescription get description =>
      SpringDescription.withDampingRatio(mass: 1, stiffness: stiffness, ratio: dampingRatio);

  /// A simulation of this spring moving from [start] to [end].
  ///
  /// The simulation lands exactly on [end] once it is within tolerance.
  SpringSimulation simulate(double start, double end, {double velocity = 0}) =>
      SpringSimulation(description, start, end, velocity, tolerance: _tolerance, snapToEnd: true);

  /// The time this spring needs to travel from 0 to 1 and settle.
  Duration get settleDuration => _settleDurations.putIfAbsent(this, () {
    final SpringSimulation simulation = simulate(0, 1);
    var time = 0.0;
    while (!simulation.isDone(time) && time < 5) {
      time += 1 / 240;
    }
    return Duration(microseconds: (time * Duration.microsecondsPerSecond).round());
  });

  /// The spring as a [Curve] over [settleDuration], for APIs that only accept
  /// a duration and a curve, such as page transitions.
  Curve get curve => _SpringCurve(this);

  static final Map<Spring, Duration> _settleDurations = <Spring, Duration>{};
  static const Tolerance _tolerance = Tolerance(distance: 0.001, velocity: 0.01);

  @override
  bool operator ==(Object other) =>
      other is Spring && other.dampingRatio == dampingRatio && other.stiffness == stiffness;

  @override
  int get hashCode => Object.hash(dampingRatio, stiffness);

  @override
  String toString() => 'Spring(dampingRatio: $dampingRatio, stiffness: $stiffness)';
}

class _SpringCurve extends Curve {
  _SpringCurve(this.spring)
    : _simulation = spring.simulate(0, 1),
      _seconds = spring.settleDuration.inMicroseconds / Duration.microsecondsPerSecond;

  final Spring spring;
  final SpringSimulation _simulation;
  final double _seconds;

  @override
  double transformInternal(double t) => _simulation.x(t * _seconds);
}

/// The six springs of a Material 3 motion scheme.
@immutable
class MotionScheme extends ThemeExtension<MotionScheme> {
  /// Creates a motion scheme.
  const MotionScheme({
    required this.fastSpatial,
    required this.defaultSpatial,
    required this.slowSpatial,
    required this.fastEffects,
    required this.defaultEffects,
    required this.slowEffects,
  });

  /// The expressive scheme: bouncy spatial springs. Used for hero moments
  /// and interactive components.
  static const MotionScheme expressive = MotionScheme(
    fastSpatial: Spring(dampingRatio: 0.6, stiffness: 800),
    defaultSpatial: Spring(dampingRatio: 0.8, stiffness: 380),
    slowSpatial: Spring(dampingRatio: 0.8, stiffness: 200),
    fastEffects: Spring(dampingRatio: 1, stiffness: 3800),
    defaultEffects: Spring(dampingRatio: 1, stiffness: 1600),
    slowEffects: Spring(dampingRatio: 1, stiffness: 800),
  );

  /// The standard scheme: barely any overshoot. Used for utilitarian motion.
  static const MotionScheme standard = MotionScheme(
    fastSpatial: Spring(dampingRatio: 0.9, stiffness: 1400),
    defaultSpatial: Spring(dampingRatio: 0.9, stiffness: 700),
    slowSpatial: Spring(dampingRatio: 0.9, stiffness: 300),
    fastEffects: Spring(dampingRatio: 1, stiffness: 3800),
    defaultEffects: Spring(dampingRatio: 1, stiffness: 1600),
    slowEffects: Spring(dampingRatio: 1, stiffness: 800),
  );

  /// Small components such as buttons and switches.
  final Spring fastSpatial;

  /// Partial-screen changes such as sheets and the navigation rail.
  final Spring defaultSpatial;

  /// Full-screen transitions.
  final Spring slowSpatial;

  /// Colour and opacity of small components.
  final Spring fastEffects;

  /// Colour and opacity of partial-screen changes.
  final Spring defaultEffects;

  /// Colour and opacity of full-screen transitions.
  final Spring slowEffects;

  /// The scheme of the closest [Theme], or [expressive] by default.
  static MotionScheme of(BuildContext context) => Theme.of(context).extension<MotionScheme>() ?? expressive;

  @override
  MotionScheme copyWith({
    Spring? fastSpatial,
    Spring? defaultSpatial,
    Spring? slowSpatial,
    Spring? fastEffects,
    Spring? defaultEffects,
    Spring? slowEffects,
  }) {
    return MotionScheme(
      fastSpatial: fastSpatial ?? this.fastSpatial,
      defaultSpatial: defaultSpatial ?? this.defaultSpatial,
      slowSpatial: slowSpatial ?? this.slowSpatial,
      fastEffects: fastEffects ?? this.fastEffects,
      defaultEffects: defaultEffects ?? this.defaultEffects,
      slowEffects: slowEffects ?? this.slowEffects,
    );
  }

  @override
  MotionScheme lerp(MotionScheme? other, double t) => t < 0.5 || other == null ? this : other;
}

/// Whether the platform asks for reduced motion.
bool reduceMotion(BuildContext context) => MediaQuery.maybeDisableAnimationsOf(context) ?? false;

/// Animates a value towards [value] with [spring] whenever [value] changes.
///
/// Unlike implicitly animated widgets driven by a duration and a curve, a
/// spring keeps its velocity when the target changes mid-flight, which is
/// what makes Material 3 Expressive motion feel continuous and physical.
class SpringBuilder extends StatefulWidget {
  /// Creates a spring-driven builder.
  const SpringBuilder({super.key, required this.value, required this.spring, required this.builder, this.child});

  /// The target value.
  final double value;

  /// The spring that drives the value.
  final Spring spring;

  /// Builds the widget for the current, animated value.
  final ValueWidgetBuilder<double> builder;

  /// A child that does not depend on the value, passed to [builder].
  final Widget? child;

  @override
  State<SpringBuilder> createState() => _SpringBuilderState();
}

class _SpringBuilderState extends State<SpringBuilder> with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController.unbounded(vsync: this, value: widget.value);

  @override
  void didUpdateWidget(SpringBuilder oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.value == oldWidget.value) {
      return;
    }
    if (reduceMotion(context)) {
      _controller.value = widget.value;
    } else {
      _controller.animateWith(widget.spring.simulate(_controller.value, widget.value, velocity: _controller.velocity));
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      child: widget.child,
      builder: (BuildContext context, Widget? child) => widget.builder(context, _controller.value, child),
    );
  }
}
