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

import 'package:material_new_shapes/material_new_shapes.dart';
import 'package:material_ui/material_ui.dart';

import '../theme/motion.dart';

/// The shapes of the Material 3 Expressive shape library that look good as
/// decorative containers of small icons and initials.
///
/// See https://m3.material.io/styles/shape/overview-principles.
final List<RoundedPolygon> decorativeShapes = <RoundedPolygon>[
  MaterialShapes.cookie9Sided,
  MaterialShapes.clover4Leaf,
  MaterialShapes.sunny,
  MaterialShapes.softBurst,
  MaterialShapes.cookie6Sided,
  MaterialShapes.gem,
  MaterialShapes.pentagon,
  MaterialShapes.flower,
  MaterialShapes.puffyDiamond,
  MaterialShapes.cookie4Sided,
  MaterialShapes.arch,
  MaterialShapes.clover8Leaf,
];

/// Picks a stable decorative shape for [key], so that the same plugin always
/// gets the same shape.
RoundedPolygon decorativeShapeFor(String key) => decorativeShapes[stableHash(key) % decorativeShapes.length];

/// A deterministic, platform-independent string hash (FNV-1a, 32 bit).
///
/// [String.hashCode] is not guaranteed to be stable between runs.
int stableHash(String text) {
  var hash = 0x811C9DC5;
  for (final int unit in text.codeUnits) {
    hash ^= unit;
    hash = (hash * 0x01000193) & 0xFFFFFFFF;
  }
  return hash;
}

/// A [ShapeBorder] drawn from a normalized Material shape
/// ([RoundedPolygon] inside the unit square), stretched to the painted
/// rectangle and optionally rotated around its centre.
class PolygonBorder extends OutlinedBorder {
  /// Creates a border from [polygon], which must be normalized.
  const PolygonBorder({required this.polygon, this.rotation = 0, super.side});

  /// The shape, inside the unit square.
  final RoundedPolygon polygon;

  /// Rotation around the centre, in radians.
  final double rotation;

  Path _path(Rect rect) {
    final matrix = Matrix4.identity()
      ..translateByDouble(rect.center.dx, rect.center.dy, 0, 1)
      ..rotateZ(rotation)
      ..scaleByDouble(rect.width, rect.height, 1, 1)
      ..translateByDouble(-0.5, -0.5, 0, 1);
    return polygon.toPath().transform(matrix.storage);
  }

  @override
  Path getOuterPath(Rect rect, {TextDirection? textDirection}) => _path(rect);

  @override
  Path getInnerPath(Rect rect, {TextDirection? textDirection}) => _path(rect.deflate(side.strokeInset));

  @override
  PolygonBorder copyWith({BorderSide? side, RoundedPolygon? polygon, double? rotation}) {
    return PolygonBorder(
      polygon: polygon ?? this.polygon,
      rotation: rotation ?? this.rotation,
      side: side ?? this.side,
    );
  }

  @override
  ShapeBorder scale(double t) => copyWith(side: side.scale(t));

  @override
  void paint(Canvas canvas, Rect rect, {TextDirection? textDirection}) {
    if (side.style == BorderStyle.none || side.width == 0) {
      return;
    }
    canvas.drawPath(_path(rect.deflate(side.strokeInset / 2)), side.toPaint());
  }

  @override
  bool operator ==(Object other) =>
      other is PolygonBorder && other.polygon == polygon && other.rotation == rotation && other.side == side;

  @override
  int get hashCode => Object.hash(polygon, rotation, side);
}

/// A [ShapeBorder] that morphs between two normalized Material shapes.
///
/// [progress] 0 draws the start shape of [morph], 1 its end shape. Used to
/// animate shape changes with a spring, which is the signature motion of
/// Material 3 Expressive.
class MorphBorder extends OutlinedBorder {
  /// Creates a border for [morph] at [progress].
  const MorphBorder({required this.morph, required this.progress, this.rotation = 0, super.side});

  /// The morph between two normalized shapes.
  final Morph morph;

  /// How far the morph has progressed, from 0 to 1.
  final double progress;

  /// Rotation around the centre, in radians.
  final double rotation;

  Path _path(Rect rect) {
    final matrix = Matrix4.identity()
      ..translateByDouble(rect.center.dx, rect.center.dy, 0, 1)
      ..rotateZ(rotation)
      ..scaleByDouble(rect.width, rect.height, 1, 1)
      ..translateByDouble(-0.5, -0.5, 0, 1);
    return morph.toPath(progress: progress.clamp(0, 1)).transform(matrix.storage);
  }

  @override
  Path getOuterPath(Rect rect, {TextDirection? textDirection}) => _path(rect);

  @override
  Path getInnerPath(Rect rect, {TextDirection? textDirection}) => _path(rect.deflate(side.strokeInset));

  @override
  MorphBorder copyWith({BorderSide? side, double? progress, double? rotation}) {
    return MorphBorder(
      morph: morph,
      progress: progress ?? this.progress,
      rotation: rotation ?? this.rotation,
      side: side ?? this.side,
    );
  }

  @override
  ShapeBorder scale(double t) => copyWith(side: side.scale(t));

  @override
  void paint(Canvas canvas, Rect rect, {TextDirection? textDirection}) {
    if (side.style == BorderStyle.none || side.width == 0) {
      return;
    }
    canvas.drawPath(_path(rect.deflate(side.strokeInset / 2)), side.toPaint());
  }

  @override
  bool operator ==(Object other) =>
      other is MorphBorder &&
      other.morph == morph &&
      other.progress == progress &&
      other.rotation == rotation &&
      other.side == side;

  @override
  int get hashCode => Object.hash(morph, progress, rotation, side);
}

/// A plugin avatar: an initial on a decorative Material 3 Expressive shape.
///
/// The shape and the colour are derived from [name], so every plugin keeps
/// its look. Hovering spins the shape a little, a small moment of delight
/// that Material 3 Expressive encourages for interactive elements.
class ShapeAvatar extends StatefulWidget {
  /// Creates an avatar for [name].
  const ShapeAvatar({super.key, required this.name, this.size = 40, this.icon});

  /// The plugin name the avatar represents.
  final String name;

  /// Width and height of the avatar.
  final double size;

  /// An icon to show instead of the initial.
  final IconData? icon;

  @override
  State<ShapeAvatar> createState() => _ShapeAvatarState();
}

class _ShapeAvatarState extends State<ShapeAvatar> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    final int hash = stableHash(widget.name);
    // Selected list items use the secondary container colour, so avatars
    // avoid it and stay visible on them.
    final (Color background, Color foreground) = switch (hash % 3) {
      0 => (colors.primaryContainer, colors.onPrimaryContainer),
      1 => (colors.secondary, colors.onSecondary),
      _ => (colors.tertiaryContainer, colors.onTertiaryContainer),
    };
    final String initial = widget.name.isEmpty ? '?' : widget.name.characters.first.toUpperCase();
    final RoundedPolygon polygon = decorativeShapeFor(widget.name);
    final TextStyle? textStyle = Theme.of(context).textTheme.titleMedium
        ?.copyWith(color: foreground, fontWeight: FontWeight.w700, fontSize: widget.size * 0.42);

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: SpringBuilder(
        value: _hovered ? math.pi / 6 : 0,
        spring: MotionScheme.of(context).fastSpatial,
        builder: (BuildContext context, double rotation, Widget? child) {
          return DecoratedBox(
            decoration: ShapeDecoration(
              color: background,
              shape: PolygonBorder(polygon: polygon, rotation: rotation),
            ),
            child: child,
          );
        },
        child: SizedBox.square(
          dimension: widget.size,
          child: Center(
            child: widget.icon != null
                ? Icon(widget.icon, size: widget.size * 0.5, color: foreground)
                : Text(initial, style: textStyle),
          ),
        ),
      ),
    );
  }
}
