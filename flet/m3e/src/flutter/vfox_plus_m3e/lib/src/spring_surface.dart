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
import 'dart:ui' show lerpDouble;

import 'package:flet/flet.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import 'motion.dart';

/// The `SpringSurface` control: an interactive surface whose corners morph
/// on a spring as it is hovered, selected and pressed — the shape morphing
/// of Material 3 Expressive.
///
/// The Python side composes connected button groups, segmented list items
/// and split buttons from it; they are the `ConnectedButtonGroup`,
/// `ListGroupItem` and `SplitButton` of
/// `flutter/lib/src/ui/widgets/`. A corner of 9999 is fully rounded: half of
/// the shorter side of the surface.
class SpringSurfaceControl extends StatefulWidget {
  const SpringSurfaceControl({super.key, required this.control});

  final Control control;

  @override
  State<SpringSurfaceControl> createState() => _SpringSurfaceControlState();
}

class _SpringSurfaceControlState extends State<SpringSurfaceControl> {
  bool _hovered = false;
  bool _pressed = false;

  @override
  Widget build(BuildContext context) {
    final Control control = widget.control;
    final ColorScheme colors = Theme.of(context).colorScheme;
    final bool interactive = !control.disabled && control.getBool('interactive', true)!;
    final bool selected = control.getBool('selected', false)!;
    final Spring spring = Springs.byName(control.getString('spring'), Springs.fastSpatial);
    final BorderRadius rest = control.getBorderRadius('radius') ?? BorderRadius.zero;
    final BorderRadius hovered = control.getBorderRadius('hover_radius') ?? rest;
    final BorderRadius chosen = control.getBorderRadius('selected_radius') ?? hovered;
    final BorderRadius pressed = control.getBorderRadius('pressed_radius') ?? chosen;
    final Color background = control.getColor('bgcolor', context, colors.surfaceContainer)!;
    final Duration colorDuration = reduceMotion(context) ? Duration.zero : Springs.fastEffects.settleDuration;

    final Widget content = control.buildWidget('content') ?? const SizedBox.shrink();
    Widget surface = SpringBuilder(
      value: interactive && _hovered ? 1 : 0,
      spring: spring,
      builder: (BuildContext context, double hover, Widget? child) => SpringBuilder(
        value: selected ? 1 : 0,
        spring: spring,
        builder: (BuildContext context, double selection, Widget? child) => SpringBuilder(
          value: interactive && _pressed ? 1 : 0,
          spring: spring,
          builder: (BuildContext context, double press, Widget? child) {
            final shape = _StateShapeBorder(
              radii: <BorderRadius>[rest, hovered, chosen, pressed],
              progress: <double>[hover, selection, press],
            );
            return TweenAnimationBuilder<Color?>(
              tween: ColorTween(end: background),
              duration: colorDuration,
              curve: Springs.fastEffects.curve,
              builder: (BuildContext context, Color? color, Widget? child) => Material(
                color: color,
                shape: shape,
                clipBehavior: Clip.antiAlias,
                child: interactive
                    ? InkWell(
                        customBorder: shape,
                        onTap: () => control.triggerEvent('click'),
                        onHover: (bool value) => setState(() => _hovered = value),
                        onHighlightChanged: (bool value) => setState(() => _pressed = value),
                        child: child,
                      )
                    : child,
              ),
              child: child,
            );
          },
          child: child,
        ),
        child: child,
      ),
      child: content,
    );
    surface = Semantics(
      button: interactive && control.getBool('semantics_button', true)!,
      selected: control.getBool('semantics_selected') ?? (selected ? true : null),
      inMutuallyExclusiveGroup: control.getBool('mutually_exclusive'),
      child: surface,
    );
    return LayoutControl(control: control, child: surface);
  }
}

/// A rounded rectangle whose corners are the [radii] of four states — rest,
/// hovered, selected and pressed — blended by the [progress] of the last
/// three, each from 0 to 1. Corners are limited to half of the shorter side
/// before they are blended, so that fully rounded corners morph smoothly.
class _StateShapeBorder extends OutlinedBorder {
  const _StateShapeBorder({required this.radii, required this.progress});

  final List<BorderRadius> radii;
  final List<double> progress;

  RRect _rrect(Rect rect) {
    final double full = rect.shortestSide / 2;
    Radius limit(Radius radius) => Radius.circular(math.min(radius.x, full));
    Radius corner(Radius Function(BorderRadius) of) {
      var value = limit(of(radii[0])).x;
      for (var i = 0; i < progress.length; i++) {
        value = lerpDouble(value, limit(of(radii[i + 1])).x, progress[i])!;
      }
      return Radius.circular(value.clamp(0, full));
    }

    return RRect.fromRectAndCorners(
      rect,
      topLeft: corner((BorderRadius r) => r.topLeft),
      topRight: corner((BorderRadius r) => r.topRight),
      bottomLeft: corner((BorderRadius r) => r.bottomLeft),
      bottomRight: corner((BorderRadius r) => r.bottomRight),
    );
  }

  @override
  Path getOuterPath(Rect rect, {TextDirection? textDirection}) => Path()..addRRect(_rrect(rect));

  @override
  Path getInnerPath(Rect rect, {TextDirection? textDirection}) => Path()..addRRect(_rrect(rect));

  @override
  OutlinedBorder copyWith({BorderSide? side}) => this;

  @override
  ShapeBorder scale(double t) => this;

  @override
  void paint(Canvas canvas, Rect rect, {TextDirection? textDirection}) {}

  @override
  bool operator ==(Object other) =>
      other is _StateShapeBorder && listEquals(other.radii, radii) && listEquals(other.progress, progress);

  @override
  int get hashCode => Object.hash(Object.hashAll(radii), Object.hashAll(progress));
}
