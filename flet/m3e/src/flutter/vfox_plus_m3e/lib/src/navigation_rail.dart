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

import 'package:flet/flet.dart';
import 'package:flutter/material.dart';

import 'motion.dart';

/// The `ExpressiveNavigationRail` control: the Material 3 Expressive
/// navigation rail, collapsed or expanded.
///
/// Collapsed, it is 96 dp wide with labels below the icons. Expanded, it
/// shows labels next to the icons. The rail, its destinations and its
/// floating action button morph between the two layouts on a spring. A port
/// of `flutter/lib/src/ui/widgets/navigation_rail.dart`; see
/// https://m3.material.io/components/navigation-rail.
///
/// Its destinations are `RailDestination` controls, which are not rendered
/// themselves. Events: `change` with the index of the chosen destination,
/// `toggle` with the requested expansion, and `fab_click`.
class ExpressiveNavigationRailControl extends StatelessWidget {
  const ExpressiveNavigationRailControl({super.key, required this.control});

  final Control control;

  /// Width when collapsed.
  static const double collapsedWidth = 96;

  /// Width when expanded.
  static const double expandedWidth = 240;

  @override
  Widget build(BuildContext context) {
    final bool expanded = control.getBool('expanded', false)!;
    final int selected = control.getInt('selected_index', 0)!;
    final List<Control> destinations = control.children('destinations');
    final IconData? fabIcon = control.getIconData('fab_icon');
    final bool fabEnabled = !control.getBool('fab_disabled', false)!;
    return LayoutControl(
      control: control,
      child: SpringBuilder(
        value: expanded ? 1 : 0,
        spring: Springs.defaultSpatial,
        builder: (BuildContext context, double t, Widget? _) {
          final double width = lerpDouble(collapsedWidth, expandedWidth, t)!;
          return SizedBox(
            width: width,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                const SizedBox(height: 16),
                _MenuButton(
                  expansion: t,
                  icon: control.getIconData(expanded ? 'collapse_icon' : 'expand_icon'),
                  tooltip: control.getString(expanded ? 'collapse_tooltip' : 'expand_tooltip', '')!,
                  onPressed: () => control.triggerEvent('toggle', !expanded),
                ),
                if (fabIcon != null) ...<Widget>[
                  const SizedBox(height: 8),
                  _RailFab(
                    expansion: t,
                    icon: fabIcon,
                    label: control.getString('fab_label', '')!,
                    onPressed: fabEnabled ? () => control.triggerEvent('fab_click') : null,
                  ),
                ],
                const SizedBox(height: 24),
                Expanded(
                  child: SingleChildScrollView(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: <Widget>[
                        for (var i = 0; i < destinations.length; i++)
                          _RailItem(
                            icon: destinations[i].getIconData('icon'),
                            label: destinations[i].getString('label', '')!,
                            badgeCount: destinations[i].getInt('badge_count', 0)!,
                            selected: i == selected,
                            expansion: t,
                            width: width,
                            onTap: () => control.triggerEvent('change', i),
                          ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}

class _MenuButton extends StatelessWidget {
  const _MenuButton({required this.expansion, required this.icon, required this.tooltip, required this.onPressed});

  final double expansion;
  final IconData? icon;
  final String tooltip;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    // Centred over a collapsed rail, aligned with the item icons when expanded.
    final double start = lerpDouble((ExpressiveNavigationRailControl.collapsedWidth - 48) / 2, 16, expansion)!;
    return Padding(
      padding: EdgeInsetsDirectional.only(start: start),
      child: Align(
        alignment: AlignmentDirectional.centerStart,
        // A 48 dp target, as the icon buttons of the Flutter implementation.
        child: IconButton(
          tooltip: tooltip,
          onPressed: onPressed,
          icon: Icon(icon),
          constraints: const BoxConstraints(minWidth: 48, minHeight: 48),
        ),
      ),
    );
  }
}

class _RailFab extends StatelessWidget {
  const _RailFab({required this.expansion, required this.icon, required this.label, required this.onPressed});

  final double expansion;
  final IconData icon;
  final String label;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final double start = lerpDouble((ExpressiveNavigationRailControl.collapsedWidth - 56) / 2, 12, expansion)!;
    final double labelOpacity = ((expansion - 0.5) * 2).clamp(0, 1);
    final bool enabled = onPressed != null;
    final Color foreground = enabled ? colors.onPrimaryContainer : colors.onSurface.withValues(alpha: 0.38);
    Widget fab = Material(
      color: enabled ? colors.primaryContainer : colors.onSurface.withValues(alpha: 0.1),
      elevation: enabled ? 1 : 0,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(16))),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onPressed,
        child: SizedBox(
          height: 56,
          child: Padding(
            padding: const EdgeInsetsDirectional.symmetric(horizontal: 16),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                Icon(icon, color: foreground),
                ClipRect(
                  child: Align(
                    alignment: AlignmentDirectional.centerStart,
                    widthFactor: expansion.clamp(0, 1),
                    child: Opacity(
                      opacity: labelOpacity,
                      child: Padding(
                        padding: const EdgeInsetsDirectional.only(start: 12, end: 4),
                        child: Text(label, maxLines: 1, style: theme.textTheme.titleSmall?.copyWith(color: foreground)),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    if (expansion < 0.5 && label.isNotEmpty) {
      fab = Tooltip(message: label, child: fab);
    }
    return Padding(
      padding: EdgeInsetsDirectional.only(start: start),
      child: Align(
        alignment: AlignmentDirectional.centerStart,
        child: Semantics(button: true, label: label, enabled: enabled, excludeSemantics: true, child: fab),
      ),
    );
  }
}

class _RailItem extends StatelessWidget {
  const _RailItem({
    required this.icon,
    required this.label,
    required this.badgeCount,
    required this.selected,
    required this.expansion,
    required this.width,
    required this.onTap,
  });

  final IconData? icon;
  final String label;
  final int badgeCount;
  final bool selected;
  final double expansion;
  final double width;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final double t = expansion;

    // Collapsed: a 56 × 32 indicator with the label below, in a 64 dp item.
    // Expanded: a 56 dp high indicator around icon and label.
    final double height = lerpDouble(64, 56, t)!;
    final Rect indicator = Rect.lerp(
      const Rect.fromLTWH((ExpressiveNavigationRailControl.collapsedWidth - 56) / 2, 4, 56, 32),
      Rect.fromLTWH(12, 0, width - 24, 56),
      t,
    )!;
    final Offset iconOffset = Offset.lerp(
      Offset(indicator.center.dx - 12, indicator.center.dy - 12),
      Offset(indicator.left + 16, indicator.center.dy - 12),
      t,
    )!;
    final double collapsedLabelOpacity = (1 - t * 2).clamp(0, 1);
    final double expandedLabelOpacity = ((t - 0.5) * 2).clamp(0, 1);
    final Color iconColor = selected ? colors.onSecondaryContainer : colors.onSurfaceVariant;
    final FontWeight weight = selected ? FontWeight.w700 : FontWeight.w500;
    final List<FontVariation> variations = <FontVariation>[FontVariation('wght', selected ? 700 : 500)];
    final TextStyle collapsedLabel = theme.textTheme.labelMedium!.copyWith(
      color: selected ? colors.secondary : colors.onSurfaceVariant,
      fontWeight: weight,
      fontVariations: variations,
    );
    final TextStyle expandedLabel = theme.textTheme.labelLarge!.copyWith(
      color: selected ? colors.onSecondaryContainer : colors.onSurfaceVariant,
      fontWeight: weight,
      fontVariations: variations,
    );

    Widget iconWidget = SpringBuilder(
      value: selected ? 1 : 0,
      spring: Springs.fastEffects,
      builder: (BuildContext context, double fill, Widget? _) =>
          Icon(icon, color: iconColor, fill: fill.clamp(0, 1), size: 24),
    );
    if (badgeCount > 0) {
      iconWidget = Badge.count(count: badgeCount, child: iconWidget);
    }

    return Semantics(
      selected: selected,
      button: true,
      label: label,
      excludeSemantics: true,
      child: SizedBox(
        height: height + 4,
        child: Stack(
          clipBehavior: Clip.none,
          children: <Widget>[
            PositionedDirectional(
              start: indicator.left,
              top: indicator.top,
              width: indicator.width,
              height: indicator.height,
              child: SpringBuilder(
                value: selected ? 1 : 0,
                spring: Springs.fastSpatial,
                builder: (BuildContext context, double s, Widget? _) {
                  // The indicator grows out of the icon's centre.
                  final double scale = 0.5 + 0.5 * s.clamp(0, 1.2);
                  return Material(
                    type: MaterialType.transparency,
                    child: InkWell(
                      customBorder: const StadiumBorder(),
                      onTap: onTap,
                      child: Transform.scale(
                        scaleX: scale,
                        child: DecoratedBox(
                          decoration: ShapeDecoration(
                            color: colors.secondaryContainer.withValues(alpha: s.clamp(0, 1)),
                            shape: const StadiumBorder(),
                          ),
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
            PositionedDirectional(
              start: iconOffset.dx,
              top: iconOffset.dy,
              child: IgnorePointer(child: iconWidget),
            ),
            if (collapsedLabelOpacity > 0)
              PositionedDirectional(
                start: 0,
                width: ExpressiveNavigationRailControl.collapsedWidth,
                top: indicator.bottom + 4,
                child: IgnorePointer(
                  child: Opacity(
                    opacity: collapsedLabelOpacity,
                    child: Text(
                      label,
                      textAlign: TextAlign.center,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: collapsedLabel,
                    ),
                  ),
                ),
              ),
            if (expandedLabelOpacity > 0)
              PositionedDirectional(
                start: indicator.left + 16 + 24 + 12,
                end: 24,
                top: indicator.center.dy - 10,
                child: IgnorePointer(
                  child: Opacity(
                    opacity: expandedLabelOpacity,
                    child: Text(label, maxLines: 1, overflow: TextOverflow.ellipsis, style: expandedLabel),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
