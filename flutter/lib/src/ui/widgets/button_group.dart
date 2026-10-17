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

import 'dart:ui' show lerpDouble;

import 'package:material_ui/material_ui.dart';

import '../theme/motion.dart';
import '../theme/tokens.dart';

/// One choice of a [ConnectedButtonGroup].
@immutable
class ButtonGroupSegment<T> {
  /// Creates a segment.
  const ButtonGroupSegment({required this.value, required this.label, this.icon, this.tooltip});

  /// The value selected by this segment.
  final T value;

  /// Visible label.
  final String label;

  /// Optional leading icon.
  final IconData? icon;

  /// Optional tooltip.
  final String? tooltip;
}

/// A Material 3 Expressive connected button group with single selection.
///
/// The buttons sit 2 dp apart. Outer corners are fully rounded and inner
/// corners small; the selected button morphs into a fully rounded shape and
/// a pressed button squeezes its inner corners. It replaces the segmented
/// button of Material 3. See https://m3.material.io/components/button-groups.
class ConnectedButtonGroup<T> extends StatelessWidget {
  /// Creates a connected button group.
  const ConnectedButtonGroup({
    super.key,
    required this.segments,
    required this.selected,
    required this.onSelected,
    this.expand = false,
    this.height = 40,
  });

  /// The choices, from start to end.
  final List<ButtonGroupSegment<T>> segments;

  /// The selected value.
  final T selected;

  /// Called when the user picks another value. `null` disables the group.
  final ValueChanged<T>? onSelected;

  /// Whether the buttons share all the available width equally.
  final bool expand;

  /// Height of the buttons: 40 dp is the small size.
  final double height;

  @override
  Widget build(BuildContext context) {
    // Inside a bounded width the segments shrink and ellipsize their labels
    // instead of overflowing; inside an unbounded one they take their
    // natural width.
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) => _buildRow(shrinkable: constraints.hasBoundedWidth),
    );
  }

  Widget _buildRow({required bool shrinkable}) {
    final children = <Widget>[];
    for (var i = 0; i < segments.length; i++) {
      final ButtonGroupSegment<T> segment = segments[i];
      final Widget button = _ConnectedButton(
        label: segment.label,
        icon: segment.icon,
        tooltip: segment.tooltip,
        selected: segment.value == selected,
        isFirst: i == 0,
        isLast: i == segments.length - 1,
        height: height,
        onPressed: onSelected == null ? null : () => onSelected!(segment.value),
      );
      if (i > 0) {
        children.add(const SizedBox(width: Gaps.hairline));
      }
      children.add(
        expand
            ? Expanded(child: button)
            : shrinkable
            ? Flexible(child: button)
            : button,
      );
    }
    return Semantics(
      container: true,
      child: Row(mainAxisSize: expand ? MainAxisSize.max : MainAxisSize.min, children: children),
    );
  }
}

class _ConnectedButton extends StatefulWidget {
  const _ConnectedButton({
    required this.label,
    required this.icon,
    required this.tooltip,
    required this.selected,
    required this.isFirst,
    required this.isLast,
    required this.height,
    required this.onPressed,
  });

  final String label;
  final IconData? icon;
  final String? tooltip;
  final bool selected;
  final bool isFirst;
  final bool isLast;
  final double height;
  final VoidCallback? onPressed;

  @override
  State<_ConnectedButton> createState() => _ConnectedButtonState();
}

class _ConnectedButtonState extends State<_ConnectedButton> {
  bool _pressed = false;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final MotionScheme motion = MotionScheme.of(context);
    final double full = widget.height / 2;
    const double inner = Corners.small;
    const double innerPressed = Corners.extraSmall;
    final bool enabled = widget.onPressed != null;

    final Color background = widget.selected ? colors.primary : colors.secondaryContainer;
    final Color foreground = widget.selected ? colors.onPrimary : colors.onSecondaryContainer;

    Widget button = SpringBuilder(
      value: widget.selected ? 1 : 0,
      spring: motion.fastSpatial,
      builder: (BuildContext context, double selection, Widget? child) {
        return SpringBuilder(
          value: _pressed ? 1 : 0,
          spring: motion.fastSpatial,
          builder: (BuildContext context, double press, Widget? child) {
            final double innerRadius = lerpDouble(
              lerpDouble(inner, full, selection),
              innerPressed,
              press,
            )!.clamp(0, full);
            final Radius start = Radius.circular(widget.isFirst ? full : innerRadius);
            final Radius end = Radius.circular(widget.isLast ? full : innerRadius);
            final shape = RoundedRectangleBorder(
              borderRadius: BorderRadiusDirectional.horizontal(start: start, end: end),
            );
            return TweenAnimationBuilder<Color?>(
              tween: ColorTween(end: enabled ? background : colors.onSurface.withValues(alpha: 0.1)),
              duration: motion.fastEffects.settleDuration,
              curve: motion.fastEffects.curve,
              builder: (BuildContext context, Color? color, Widget? _) {
                return Material(
                  color: color,
                  shape: shape,
                  clipBehavior: Clip.antiAlias,
                  child: InkWell(
                    customBorder: shape,
                    onTap: widget.onPressed,
                    onHighlightChanged: (bool value) => setState(() => _pressed = value),
                    child: child,
                  ),
                );
              },
            );
          },
          child: child,
        );
      },
      child: ConstrainedBox(
        constraints: BoxConstraints(minHeight: widget.height, minWidth: 48),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: Gaps.l),
          child: IconTheme.merge(
            data: IconThemeData(size: 20, color: enabled ? foreground : colors.onSurface.withValues(alpha: 0.38)),
            child: DefaultTextStyle.merge(
              style: theme.textTheme.labelLarge?.copyWith(
                color: enabled ? foreground : colors.onSurface.withValues(alpha: 0.38),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                mainAxisAlignment: MainAxisAlignment.center,
                children: <Widget>[
                  if (widget.icon != null) ...<Widget>[Icon(widget.icon), const SizedBox(width: Gaps.s)],
                  Flexible(child: Text(widget.label, maxLines: 1, overflow: TextOverflow.ellipsis)),
                ],
              ),
            ),
          ),
        ),
      ),
    );

    button = Semantics(button: true, selected: widget.selected, inMutuallyExclusiveGroup: true, child: button);
    if (widget.tooltip != null) {
      button = Tooltip(message: widget.tooltip, child: button);
    }
    return button;
  }
}
