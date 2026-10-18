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

/// The Material 3 Expressive shape library: borders drawn from Material
/// shapes, shape morphing, and the controls built on them.
///
/// A port of `flutter/lib/src/ui/widgets/polygon_border.dart` and of the
/// colour swatches of `flutter/lib/src/ui/pages/settings_page.dart`. See
/// https://m3.material.io/styles/shape/overview-principles.
library;

import 'dart:math' as math;

import 'package:flet/flet.dart';
import 'package:flutter/material.dart';
import 'package:material_new_shapes/material_new_shapes.dart';

import 'motion.dart';

/// The Material shape called [name], as named in `MaterialShapes`.
RoundedPolygon materialShape(String? name) => switch (name) {
  'circle' => MaterialShapes.circle,
  'square' => MaterialShapes.square,
  'oval' => MaterialShapes.oval,
  'pill' => MaterialShapes.pill,
  'arch' => MaterialShapes.arch,
  'pentagon' => MaterialShapes.pentagon,
  'gem' => MaterialShapes.gem,
  'sunny' => MaterialShapes.sunny,
  'verySunny' => MaterialShapes.verySunny,
  'cookie4Sided' => MaterialShapes.cookie4Sided,
  'cookie6Sided' => MaterialShapes.cookie6Sided,
  'cookie7Sided' => MaterialShapes.cookie7Sided,
  'cookie9Sided' => MaterialShapes.cookie9Sided,
  'cookie12Sided' => MaterialShapes.cookie12Sided,
  'clover4Leaf' => MaterialShapes.clover4Leaf,
  'clover8Leaf' => MaterialShapes.clover8Leaf,
  'burst' => MaterialShapes.burst,
  'softBurst' => MaterialShapes.softBurst,
  'flower' => MaterialShapes.flower,
  'puffy' => MaterialShapes.puffy,
  'puffyDiamond' => MaterialShapes.puffyDiamond,
  _ => MaterialShapes.circle,
};

/// The shapes of the shape library that look good as decorative containers
/// of small icons and initials.
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

/// A deterministic, platform-independent string hash (FNV-1a, 32 bit), over
/// the UTF-16 code units of [text] as in the other implementations.
int stableHash(String text) {
  var hash = 0x811C9DC5;
  for (final int unit in text.codeUnits) {
    hash ^= unit;
    hash = (hash * 0x01000193) & 0xFFFFFFFF;
  }
  return hash;
}

/// A [ShapeBorder] drawn from a normalized Material shape, stretched to the
/// painted rectangle and optionally rotated around its centre.
class PolygonBorder extends OutlinedBorder {
  /// Creates a border from [polygon], which must be normalized.
  const PolygonBorder({required this.polygon, this.rotation = 0, super.side});

  /// The shape, inside the unit square.
  final RoundedPolygon polygon;

  /// Rotation around the centre, in radians.
  final double rotation;

  Path _path(Rect rect) => polygon.toPath().transform(_unitToRect(rect, rotation).storage);

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
    if (side.style != BorderStyle.none && side.width != 0) {
      canvas.drawPath(_path(rect.deflate(side.strokeInset / 2)), side.toPaint());
    }
  }

  @override
  bool operator ==(Object other) =>
      other is PolygonBorder && other.polygon == polygon && other.rotation == rotation && other.side == side;

  @override
  int get hashCode => Object.hash(polygon, rotation, side);
}

/// A [ShapeBorder] that morphs between two normalized Material shapes:
/// [progress] 0 draws the start shape of [morph], 1 its end shape.
class MorphBorder extends OutlinedBorder {
  /// Creates a border for [morph] at [progress].
  const MorphBorder({required this.morph, required this.progress, this.rotation = 0, super.side});

  /// The morph between two normalized shapes.
  final Morph morph;

  /// How far the morph has progressed, from 0 to 1.
  final double progress;

  /// Rotation around the centre, in radians.
  final double rotation;

  Path _path(Rect rect) => morph.toPath(progress: progress.clamp(0, 1)).transform(_unitToRect(rect, rotation).storage);

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
    if (side.style != BorderStyle.none && side.width != 0) {
      canvas.drawPath(_path(rect.deflate(side.strokeInset / 2)), side.toPaint());
    }
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

Matrix4 _unitToRect(Rect rect, double rotation) => Matrix4.identity()
  ..translateByDouble(rect.center.dx, rect.center.dy, 0, 1)
  ..rotateZ(rotation)
  ..scaleByDouble(rect.width, rect.height, 1, 1)
  ..translateByDouble(-0.5, -0.5, 0, 1);

/// The `MaterialShape` control: [content] on a Material shape, which may
/// turn on a spring when hovered.
class MaterialShapeControl extends StatefulWidget {
  const MaterialShapeControl({super.key, required this.control});

  final Control control;

  @override
  State<MaterialShapeControl> createState() => _MaterialShapeControlState();
}

class _MaterialShapeControlState extends State<MaterialShapeControl> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final Control control = widget.control;
    final RoundedPolygon polygon = materialShape(control.getString('shape'));
    final double size = control.getDouble('size', 48)!;
    final double rotation = control.getDouble('rotation', 0)! * math.pi / 180;
    final double hoverRotation = control.getDouble('hover_rotation', 0)! * math.pi / 180;
    final Widget content = SizedBox.square(
      dimension: size,
      child: Center(child: control.buildWidget('content')),
    );
    Widget shape = SpringBuilder(
      value: _hovered ? hoverRotation : 0,
      spring: Springs.fastSpatial,
      builder: (BuildContext context, double turn, Widget? child) => DecoratedBox(
        decoration: ShapeDecoration(
          color: control.getColor('bgcolor', context, Theme.of(context).colorScheme.secondaryContainer),
          shape: PolygonBorder(polygon: polygon, rotation: rotation + turn),
        ),
        child: child,
      ),
      child: content,
    );
    if (hoverRotation != 0) {
      shape = MouseRegion(
        onEnter: (_) => setState(() => _hovered = true),
        onExit: (_) => setState(() => _hovered = false),
        child: shape,
      );
    }
    return LayoutControl(control: control, child: shape);
  }
}

/// The `ShapeAvatar` control: a plugin avatar, the initial of a name on a
/// decorative shape whose form and colours are derived from the name, so
/// that every plugin keeps its look. Hovering spins the shape a little.
class ShapeAvatarControl extends StatefulWidget {
  const ShapeAvatarControl({super.key, required this.control});

  final Control control;

  @override
  State<ShapeAvatarControl> createState() => _ShapeAvatarControlState();
}

class _ShapeAvatarControlState extends State<ShapeAvatarControl> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final Control control = widget.control;
    final String name = control.getString('name', '')!;
    final double size = control.getDouble('size', 40)!;
    final ColorScheme colors = Theme.of(context).colorScheme;
    final int hash = stableHash(name);
    // Selected list items use the secondary container colour. Avatars avoid
    // it, and the primary container too, which some seeds (the vfox orange
    // among them) make just as light, so that they stay visible on selected
    // items.
    final (Color background, Color foreground) = switch (hash % 3) {
      0 => (colors.primaryFixedDim, colors.onPrimaryFixed),
      1 => (colors.secondary, colors.onSecondary),
      _ => (colors.tertiaryContainer, colors.onTertiaryContainer),
    };
    final String initial = name.isEmpty ? '?' : String.fromCharCode(name.runes.first).toUpperCase();
    final IconData? icon = control.getIconData('icon');
    final RoundedPolygon polygon = decorativeShapes[hash % decorativeShapes.length];
    final TextStyle? textStyle = Theme.of(context).textTheme.titleMedium?.copyWith(
      color: foreground,
      fontWeight: FontWeight.w700,
      fontVariations: const <FontVariation>[FontVariation('wght', 700)],
      fontSize: size * 0.42,
    );
    return LayoutControl(
      control: control,
      child: ExcludeSemantics(
        child: MouseRegion(
          onEnter: (_) => setState(() => _hovered = true),
          onExit: (_) => setState(() => _hovered = false),
          child: SpringBuilder(
            value: _hovered ? math.pi / 6 : 0,
            spring: Springs.fastSpatial,
            builder: (BuildContext context, double rotation, Widget? child) => DecoratedBox(
              decoration: ShapeDecoration(
                color: background,
                shape: PolygonBorder(polygon: polygon, rotation: rotation),
              ),
              child: child,
            ),
            child: SizedBox.square(
              dimension: size,
              child: Center(
                child: icon != null ? Icon(icon, size: size * 0.5, color: foreground) : Text(initial, style: textStyle),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// The `ColorSwatch` control: a colour to choose as seed. The selected
/// swatch morphs from a circle into a flower while turning, and shows a
/// check mark.
class ColorSwatchControl extends StatelessWidget {
  const ColorSwatchControl({super.key, required this.control});

  final Control control;

  static final Morph _circleToFlower = Morph(MaterialShapes.circle, MaterialShapes.flower);

  @override
  Widget build(BuildContext context) {
    final Color seed = control.getColor('seed', context, Colors.orange)!;
    final bool selected = control.getBool('selected', false)!;
    final String label = control.getString('label', '')!;
    final ColorScheme swatch = ColorScheme.fromSeed(seedColor: seed, brightness: Theme.of(context).brightness);
    final IconData? check = control.getIconData('check_icon');
    return LayoutControl(
      control: control,
      child: Semantics(
        button: true,
        selected: selected,
        label: label,
        child: Tooltip(
          message: label,
          child: InkResponse(
            onTap: control.disabled ? null : () => control.triggerEvent('click'),
            radius: 24,
            child: SpringBuilder(
              value: selected ? 1 : 0,
              spring: Springs.fastSpatial,
              builder: (BuildContext context, double t, Widget? _) => DecoratedBox(
                decoration: ShapeDecoration(
                  color: swatch.primary,
                  shape: MorphBorder(morph: _circleToFlower, progress: t, rotation: t * 0.6),
                ),
                child: SizedBox.square(
                  dimension: 40,
                  child: Opacity(
                    opacity: t.clamp(0, 1),
                    child: check == null ? null : Icon(check, color: swatch.onPrimary, size: 20),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
