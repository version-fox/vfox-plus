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

import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter/physics.dart';
import 'package:flutter/scheduler.dart';
import 'package:material_new_shapes/material_new_shapes.dart';
import 'package:material_ui/material_ui.dart';

import '../theme/motion.dart';

/// The Material 3 Expressive loading indicator: a shape that morphs through
/// a sequence of Material shapes while it rotates.
///
/// Use it for waits shorter than a few seconds whose progress is unknown.
/// See https://m3.material.io/components/loading-indicator.
class LoadingIndicator extends StatefulWidget {
  /// Creates a loading indicator.
  const LoadingIndicator({super.key, this.size = 48, this.contained = false, this.color, this.semanticsLabel});

  /// Width and height of the indicator, including its container.
  final double size;

  /// Whether to draw the indicator on a circular container, for use on top
  /// of content.
  final bool contained;

  /// Colour of the morphing shape. Defaults to the primary colour, or to
  /// on-primary-container when [contained].
  final Color? color;

  /// Accessibility label.
  final String? semanticsLabel;

  /// The shape sequence of the Material 3 Expressive indeterminate indicator.
  static final List<RoundedPolygon> shapes = <RoundedPolygon>[
    MaterialShapes.softBurst,
    MaterialShapes.cookie9Sided,
    MaterialShapes.pentagon,
    MaterialShapes.pill,
    MaterialShapes.sunny,
    MaterialShapes.cookie4Sided,
    MaterialShapes.oval,
  ];

  /// Time between the start of two morphs.
  static const Duration morphInterval = Duration(milliseconds: 650);

  /// Time of one full turn of the constant rotation.
  static const Duration rotationPeriod = Duration(milliseconds: 4666);

  /// The spring that drives each morph.
  static const Spring morphSpring = Spring(dampingRatio: 0.6, stiffness: 200);

  @override
  State<LoadingIndicator> createState() => _LoadingIndicatorState();
}

class _LoadingIndicatorState extends State<LoadingIndicator> with SingleTickerProviderStateMixin {
  late final Ticker _ticker = createTicker((Duration elapsed) => _elapsed.value = elapsed);
  final ValueNotifier<Duration> _elapsed = ValueNotifier<Duration>(Duration.zero);

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final bool animate = !reduceMotion(context);
    if (animate && !_ticker.isActive) {
      _ticker.start();
    } else if (!animate && _ticker.isActive) {
      _ticker.stop();
    }
  }

  @override
  void dispose() {
    _ticker.dispose();
    _elapsed.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    final Color color = widget.color ?? (widget.contained ? colors.onPrimaryContainer : colors.primary);
    return Semantics(
      label: widget.semanticsLabel,
      child: RepaintBoundary(
        child: SizedBox.square(
          dimension: widget.size,
          child: CustomPaint(
            painter: _LoadingIndicatorPainter(
              elapsed: _elapsed,
              color: color,
              containerColor: widget.contained ? colors.primaryContainer : null,
            ),
          ),
        ),
      ),
    );
  }
}

class _LoadingIndicatorPainter extends CustomPainter {
  _LoadingIndicatorPainter({required this.elapsed, required this.color, required this.containerColor})
    : super(repaint: elapsed);

  final ValueListenable<Duration> elapsed;
  final Color color;
  final Color? containerColor;

  /// The active indicator is 38 dp inside a 48 dp container.
  static const double _activeScale = 38 / 48;

  static final List<Morph> _morphs = _createMorphs();
  static final SpringSimulation _morphProgress = LoadingIndicator.morphSpring.simulate(0, 1);

  static List<Morph> _createMorphs() {
    // Scale every shape so that it fits in the unit circle in any rotation,
    // which keeps the indicator inside its bounds while it spins.
    final List<RoundedPolygon> shapes = <RoundedPolygon>[
      for (final RoundedPolygon shape in LoadingIndicator.shapes) _fitToCircle(shape),
    ];
    return <Morph>[for (var i = 0; i < shapes.length; i++) Morph(shapes[i], shapes[(i + 1) % shapes.length])];
  }

  static RoundedPolygon _fitToCircle(RoundedPolygon shape) {
    final List<double> bounds = shape.calculateMaxBounds();
    final double radius = (bounds[2] - bounds[0]) / 2;
    final double scale = 0.5 / radius;
    final double cx = shape.centerX;
    final double cy = shape.centerY;
    return shape.transformed((double x, double y) => ((x - cx) * scale + 0.5, (y - cy) * scale + 0.5));
  }

  @override
  void paint(Canvas canvas, Size size) {
    final double side = math.min(size.width, size.height);
    final Offset center = size.center(Offset.zero);
    if (containerColor != null) {
      canvas.drawCircle(center, side / 2, Paint()..color = containerColor!);
    }

    final double millis = elapsed.value.inMicroseconds / 1000;
    final double interval = LoadingIndicator.morphInterval.inMilliseconds.toDouble();
    final int step = millis ~/ interval;
    final double progress = _morphProgress.x((millis - step * interval) / 1000);
    final Morph morph = _morphs[step % _morphs.length];

    // A constant rotation plus a quarter turn per morph that follows the
    // (overshooting) morph spring.
    final double rotation =
        2 * math.pi * (millis / LoadingIndicator.rotationPeriod.inMilliseconds) + (step + progress) * math.pi / 2;

    final double indicator = side * _activeScale;
    final matrix = Matrix4.identity()
      ..translateByDouble(center.dx, center.dy, 0, 1)
      ..rotateZ(rotation)
      ..scaleByDouble(indicator, indicator, 1, 1)
      ..translateByDouble(-0.5, -0.5, 0, 1);
    final Path path = morph.toPath(progress: progress.clamp(0, 1)).transform(matrix.storage);
    canvas.drawPath(path, Paint()..color = color);
  }

  @override
  bool shouldRepaint(_LoadingIndicatorPainter oldDelegate) =>
      oldDelegate.color != color || oldDelegate.containerColor != containerColor;
}
