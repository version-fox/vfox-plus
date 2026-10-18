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

import 'package:flet/flet.dart';
import 'package:flutter/material.dart';

import 'motion.dart';

/// The `WavyProgressIndicator` control: see [WavyProgressIndicator].
class WavyProgressIndicatorControl extends StatelessWidget {
  const WavyProgressIndicatorControl({super.key, required this.control});

  final Control control;

  @override
  Widget build(BuildContext context) {
    return LayoutControl(
      control: control,
      child: WavyProgressIndicator(
        value: control.getDouble('value'),
        color: control.getColor('color', context),
        trackColor: control.getColor('track_color', context),
        semanticsLabel: control.getString('semantics_label'),
      ),
    );
  }
}

/// The Material 3 Expressive linear progress indicator with a wavy active
/// track.
///
/// With a [value] it is determinate: the wave flattens near the start and the
/// end, and a stop indicator marks the end of the track. Without a [value] it
/// is indeterminate. A port of
/// `flutter/lib/src/ui/widgets/wavy_progress_indicator.dart`; see
/// https://m3.material.io/components/progress-indicators.
class WavyProgressIndicator extends StatefulWidget {
  /// Creates a wavy linear progress indicator.
  const WavyProgressIndicator({
    super.key,
    this.value,
    this.color,
    this.trackColor,
    this.strokeWidth = 4,
    this.amplitude = 3,
    this.wavelength,
    this.semanticsLabel,
  });

  /// Progress between 0 and 1, or `null` for an indeterminate indicator.
  final double? value;

  /// Colour of the active indicator. Defaults to the primary colour.
  final Color? color;

  /// Colour of the track. Defaults to the secondary container colour.
  final Color? trackColor;

  /// Thickness of the active indicator and of the track.
  final double strokeWidth;

  /// Height of the wave crests.
  final double amplitude;

  /// Length of one wave. Defaults to 40 dp when determinate and 20 dp when
  /// indeterminate.
  final double? wavelength;

  /// Accessibility label.
  final String? semanticsLabel;

  @override
  State<WavyProgressIndicator> createState() => _WavyProgressIndicatorState();
}

class _WavyProgressIndicatorState extends State<WavyProgressIndicator> with SingleTickerProviderStateMixin {
  // Drives the travelling wave and the indeterminate cycle.
  late final AnimationController _clock = AnimationController(vsync: this, duration: const Duration(seconds: 2));

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (reduceMotion(context)) {
      _clock.stop();
    } else if (!_clock.isAnimating) {
      _clock.repeat();
    }
  }

  @override
  void dispose() {
    _clock.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    final double? value = widget.value?.clamp(0.0, 1.0);
    final double height = widget.strokeWidth + widget.amplitude * 2;
    // The wave flattens near the start and the end of the progress.
    final double targetAmplitude = value == null || (value > 0.1 && value < 0.95) ? 1 : 0;

    return Semantics(
      label: widget.semanticsLabel,
      value: value == null ? null : '${(value * 100).round()}%',
      child: SpringBuilder(
        value: targetAmplitude,
        spring: Springs.defaultEffects,
        builder: (BuildContext context, double amplitudeFactor, Widget? _) {
          return SpringBuilder(
            value: value ?? 0,
            spring: Springs.defaultSpatial,
            builder: (BuildContext context, double animatedValue, Widget? _) {
              return RepaintBoundary(
                child: SizedBox(
                  height: height,
                  width: double.infinity,
                  child: CustomPaint(
                    painter: _WavyPainter(
                      clock: _clock,
                      value: value == null ? null : animatedValue.clamp(0.0, 1.0),
                      color: widget.color ?? colors.primary,
                      trackColor: widget.trackColor ?? colors.secondaryContainer,
                      strokeWidth: widget.strokeWidth,
                      amplitude: widget.amplitude * amplitudeFactor.clamp(0.0, 1.0),
                      wavelength: widget.wavelength ?? (value == null ? 20 : 40),
                    ),
                  ),
                ),
              );
            },
          );
        },
      ),
    );
  }
}

class _WavyPainter extends CustomPainter {
  _WavyPainter({
    required this.clock,
    required this.value,
    required this.color,
    required this.trackColor,
    required this.strokeWidth,
    required this.amplitude,
    required this.wavelength,
  }) : super(repaint: clock);

  final Animation<double> clock;
  final double? value;
  final Color color;
  final Color trackColor;
  final double strokeWidth;
  final double amplitude;
  final double wavelength;

  /// Space between the active indicator and the track.
  static const double _gap = 4;

  // The timing of the Material 3 indeterminate linear indicator, as fractions
  // of a 2 second cycle.
  static const Curve _line1Head = Interval(0, 0.375, curve: Cubic(0.2, 0, 0.8, 1));
  static const Curve _line1Tail = Interval(0.1665, 0.5415, curve: Cubic(0.4, 0, 1, 1));
  static const Curve _line2Head = Interval(0.5, 0.7835, curve: Cubic(0, 0, 0.65, 1));
  static const Curve _line2Tail = Interval(0.6335, 0.9, curve: Cubic(0.1, 0, 0.45, 1));

  @override
  void paint(Canvas canvas, Size size) {
    final double width = size.width;
    final double centerY = size.height / 2;
    final double cap = strokeWidth / 2;
    final activePaint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth
      ..strokeCap = StrokeCap.round;
    final trackPaint = Paint()
      ..color = trackColor
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth
      ..strokeCap = StrokeCap.round;
    // The wave travels one wavelength per second.
    final double phase = clock.value * 2 * 2 * math.pi;

    void drawTrack(double from, double to) {
      if (to - from > strokeWidth) {
        canvas.drawLine(Offset(from + cap, centerY), Offset(to - cap, centerY), trackPaint);
      }
    }

    void drawActive(double from, double to) {
      if (to - from <= 0) {
        return;
      }
      final path = Path();
      final double start = from + cap;
      final double end = math.max(start, to - cap);
      for (double x = start; x <= end; x += 1) {
        final double y = centerY + amplitude * math.sin(2 * math.pi * x / wavelength - phase);
        if (x == start) {
          path.moveTo(x, y);
        } else {
          path.lineTo(x, y);
        }
      }
      if (end == start) {
        path.lineTo(end + 0.01, centerY);
      }
      canvas.drawPath(path, activePaint);
    }

    final double? progress = value;
    if (progress != null) {
      final double activeEnd = width * progress;
      drawActive(0, activeEnd);
      drawTrack(progress > 0 ? activeEnd + _gap : 0, width);
      // The stop indicator marks the end of the track.
      if (progress < 1) {
        canvas.drawCircle(Offset(width - cap, centerY), cap / 2 + 0.5, Paint()..color = color);
      }
      return;
    }

    final double t = clock.value;
    final double head1 = width * _line1Head.transform(t);
    final double tail1 = width * _line1Tail.transform(t);
    final double head2 = width * _line2Head.transform(t);
    final double tail2 = width * _line2Tail.transform(t);
    // Track, line 2, track, line 1, track — from right to left.
    var cursor = 0.0;
    if (head2 > tail2) {
      drawTrack(cursor, tail2 - _gap);
      drawActive(tail2, head2);
      cursor = head2 + _gap;
    }
    if (head1 > tail1) {
      drawTrack(cursor, tail1 - _gap);
      drawActive(tail1, head1);
      cursor = head1 + _gap;
    }
    drawTrack(cursor, width);
  }

  @override
  bool shouldRepaint(_WavyPainter oldDelegate) =>
      oldDelegate.value != value ||
      oldDelegate.color != color ||
      oldDelegate.trackColor != trackColor ||
      oldDelegate.amplitude != amplitude ||
      oldDelegate.wavelength != wavelength ||
      oldDelegate.strokeWidth != strokeWidth;
}
