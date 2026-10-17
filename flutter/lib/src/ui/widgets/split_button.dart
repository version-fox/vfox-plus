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

import 'package:material_ui/material_ui.dart';

import '../icons/app_icons.g.dart';
import '../theme/motion.dart';
import '../theme/tokens.dart';

/// Colour roles of a [SplitButton].
enum SplitButtonVariant {
  /// Primary colours, for the most important action.
  filled,

  /// Secondary container colours.
  tonal,
}

/// A Material 3 Expressive split button: a main action and a menu of related
/// actions, joined by a 2 dp gap.
///
/// While the menu is open the trailing button becomes a circle and its
/// chevron turns upside down. See https://m3.material.io/components/split-button.
class SplitButton extends StatefulWidget {
  /// Creates a split button.
  const SplitButton({
    super.key,
    required this.label,
    required this.onPressed,
    required this.menuChildren,
    this.icon,
    this.menuTooltip,
    this.variant = SplitButtonVariant.tonal,
  });

  /// Label of the main action.
  final String label;

  /// Icon of the main action.
  final IconData? icon;

  /// The main action. `null` disables the whole button.
  final VoidCallback? onPressed;

  /// Items of the menu, usually [MenuItemButton]s.
  final List<Widget> menuChildren;

  /// Tooltip of the menu button.
  final String? menuTooltip;

  /// Colour roles.
  final SplitButtonVariant variant;

  @override
  State<SplitButton> createState() => _SplitButtonState();
}

class _SplitButtonState extends State<SplitButton> {
  final MenuController _menu = MenuController();
  bool _menuOpen = false;
  bool _leadingPressed = false;
  bool _trailingPressed = false;

  static const double _height = 40;
  static const double _inner = Corners.extraSmall;
  static const double _innerPressed = Corners.small;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final MotionScheme motion = MotionScheme.of(context);
    final bool enabled = widget.onPressed != null;
    final (Color background, Color foreground) = !enabled
        ? (colors.onSurface.withValues(alpha: 0.1), colors.onSurface.withValues(alpha: 0.38))
        : switch (widget.variant) {
            SplitButtonVariant.filled => (colors.primary, colors.onPrimary),
            SplitButtonVariant.tonal => (colors.secondaryContainer, colors.onSecondaryContainer),
          };
    const double full = _height / 2;

    Widget segment({
      required double innerRadius,
      required bool isLeading,
      required VoidCallback? onTap,
      required ValueChanged<bool> onHighlight,
      required Widget child,
    }) {
      final shape = RoundedRectangleBorder(
        borderRadius: BorderRadiusDirectional.horizontal(
          start: Radius.circular(isLeading ? full : innerRadius),
          end: Radius.circular(isLeading ? innerRadius : full),
        ),
      );
      return Material(
        color: background,
        shape: shape,
        clipBehavior: Clip.antiAlias,
        child: InkWell(customBorder: shape, onTap: onTap, onHighlightChanged: onHighlight, child: child),
      );
    }

    final TextStyle? labelStyle = theme.textTheme.labelLarge?.copyWith(color: foreground);

    final Widget leading = SpringBuilder(
      value: _leadingPressed ? 1 : 0,
      spring: motion.fastSpatial,
      builder: (BuildContext context, double press, Widget? child) => segment(
        innerRadius: lerpDouble(_inner, _innerPressed, press)!,
        isLeading: true,
        onTap: widget.onPressed,
        onHighlight: (bool value) => setState(() => _leadingPressed = value),
        child: child!,
      ),
      child: SizedBox(
        height: _height,
        child: Padding(
          padding: const EdgeInsetsDirectional.only(start: Gaps.l, end: Gaps.m),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              if (widget.icon != null) ...<Widget>[
                Icon(widget.icon, size: 20, color: foreground),
                const SizedBox(width: Gaps.s),
              ],
              Text(widget.label, style: labelStyle),
            ],
          ),
        ),
      ),
    );

    final Widget trailing = MenuAnchor(
      controller: _menu,
      menuChildren: widget.menuChildren,
      alignmentOffset: const Offset(0, Gaps.xs),
      onOpen: () => setState(() => _menuOpen = true),
      onClose: () => setState(() => _menuOpen = false),
      builder: (BuildContext context, MenuController controller, Widget? _) {
        return SpringBuilder(
          value: _menuOpen ? 1 : 0,
          spring: motion.fastSpatial,
          builder: (BuildContext context, double open, Widget? _) {
            return SpringBuilder(
              value: _trailingPressed ? 1 : 0,
              spring: motion.fastSpatial,
              builder: (BuildContext context, double press, Widget? _) {
                final double innerRadius = lerpDouble(lerpDouble(_inner, _innerPressed, press), full, open)!;
                final Widget button = segment(
                  innerRadius: innerRadius,
                  isLeading: false,
                  onTap: enabled ? () => controller.isOpen ? controller.close() : controller.open() : null,
                  onHighlight: (bool value) => setState(() => _trailingPressed = value),
                  child: SizedBox(
                    height: _height,
                    width: _height + 4,
                    child: Transform.rotate(
                      angle: open * math.pi,
                      child: Icon(AppIcons.expandMore, size: 22, color: foreground),
                    ),
                  ),
                );
                final String? tooltip = widget.menuTooltip;
                return tooltip == null ? button : Tooltip(message: tooltip, child: button);
              },
            );
          },
        );
      },
    );

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        leading,
        const SizedBox(width: Gaps.hairline),
        trailing,
      ],
    );
  }
}
