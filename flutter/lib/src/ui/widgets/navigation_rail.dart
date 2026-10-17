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

import '../icons/app_icons.g.dart';
import '../theme/motion.dart';
import '../theme/tokens.dart';

/// A destination of an [ExpressiveNavigationRail] or a navigation bar.
@immutable
class NavigationDestinationData {
  /// Creates a destination.
  const NavigationDestinationData({required this.icon, required this.label, this.badgeCount = 0});

  /// Icon, drawn outlined when unselected and filled when selected.
  final IconData icon;

  /// Label.
  final String label;

  /// Number shown in a badge, if greater than zero.
  final int badgeCount;
}

/// The Material 3 Expressive navigation rail, collapsed or expanded.
///
/// Collapsed, it is 96 dp wide with labels below the icons. Expanded, it
/// shows labels next to the icons and replaces the navigation drawer. The
/// rail, its destinations and its floating action button morph between the
/// two layouts on a spring. See https://m3.material.io/components/navigation-rail.
class ExpressiveNavigationRail extends StatelessWidget {
  /// Creates a navigation rail.
  const ExpressiveNavigationRail({
    super.key,
    required this.destinations,
    required this.selectedIndex,
    required this.onDestinationSelected,
    required this.expanded,
    required this.onExpandedChanged,
    required this.expandTooltip,
    required this.collapseTooltip,
    this.fabIcon,
    this.fabLabel,
    this.onFabPressed,
  });

  /// The destinations, from top to bottom.
  final List<NavigationDestinationData> destinations;

  /// Index of the selected destination.
  final int selectedIndex;

  /// Called when a destination is chosen.
  final ValueChanged<int> onDestinationSelected;

  /// Whether the rail shows labels next to the icons.
  final bool expanded;

  /// Called when the menu button toggles [expanded].
  final ValueChanged<bool> onExpandedChanged;

  /// Tooltip of the menu button while collapsed.
  final String expandTooltip;

  /// Tooltip of the menu button while expanded.
  final String collapseTooltip;

  /// Icon of the optional floating action button.
  final IconData? fabIcon;

  /// Label of the floating action button, shown when expanded.
  final String? fabLabel;

  /// Action of the floating action button.
  final VoidCallback? onFabPressed;

  /// Width when collapsed.
  static const double collapsedWidth = 96;

  /// Width when expanded.
  static const double expandedWidth = 240;

  @override
  Widget build(BuildContext context) {
    final MotionScheme motion = MotionScheme.of(context);
    return SpringBuilder(
      value: expanded ? 1 : 0,
      spring: motion.defaultSpatial,
      builder: (BuildContext context, double t, Widget? _) {
        final double width = lerpDouble(collapsedWidth, expandedWidth, t)!;
        return SizedBox(
          width: width,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              const SizedBox(height: Gaps.l),
              _MenuButton(
                expansion: t,
                expanded: expanded,
                tooltip: expanded ? collapseTooltip : expandTooltip,
                onPressed: () => onExpandedChanged(!expanded),
              ),
              if (fabIcon != null && onFabPressed != null) ...<Widget>[
                const SizedBox(height: Gaps.s),
                _RailFab(expansion: t, icon: fabIcon!, label: fabLabel ?? '', onPressed: onFabPressed!),
              ],
              const SizedBox(height: Gaps.xl),
              Expanded(
                child: SingleChildScrollView(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: <Widget>[
                      for (var i = 0; i < destinations.length; i++)
                        _RailItem(
                          destination: destinations[i],
                          selected: i == selectedIndex,
                          expansion: t,
                          width: width,
                          onTap: () => onDestinationSelected(i),
                        ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _MenuButton extends StatelessWidget {
  const _MenuButton({required this.expansion, required this.expanded, required this.tooltip, required this.onPressed});

  final double expansion;
  final bool expanded;
  final String tooltip;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    // Centred over a collapsed rail, aligned with the item icons when expanded.
    final double start = lerpDouble((ExpressiveNavigationRail.collapsedWidth - 48) / 2, 16, expansion)!;
    return Padding(
      padding: EdgeInsetsDirectional.only(start: start),
      child: Align(
        alignment: AlignmentDirectional.centerStart,
        child: IconButton(
          tooltip: tooltip,
          onPressed: onPressed,
          icon: Icon(expanded ? AppIcons.menuOpen : AppIcons.menu),
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
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final double start = lerpDouble((ExpressiveNavigationRail.collapsedWidth - 56) / 2, 12, expansion)!;
    final double labelOpacity = ((expansion - 0.5) * 2).clamp(0, 1);
    return Padding(
      padding: EdgeInsetsDirectional.only(start: start),
      child: Align(
        alignment: AlignmentDirectional.centerStart,
        child: _maybeTooltip(
          expansion < 0.5 ? label : null,
          Material(
            color: colors.primaryContainer,
            elevation: 1,
            shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.large))),
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
                      Icon(icon, color: colors.onPrimaryContainer),
                      ClipRect(
                        child: Align(
                          alignment: AlignmentDirectional.centerStart,
                          widthFactor: expansion.clamp(0, 1),
                          child: Opacity(
                            opacity: labelOpacity,
                            child: Padding(
                              padding: const EdgeInsetsDirectional.only(start: 12, end: 4),
                              child: Text(
                                label,
                                maxLines: 1,
                                style: theme.textTheme.titleSmall?.copyWith(color: colors.onPrimaryContainer),
                              ),
                            ),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  static Widget _maybeTooltip(String? message, Widget child) =>
      message == null || message.isEmpty ? child : Tooltip(message: message, child: child);
}

class _RailItem extends StatelessWidget {
  const _RailItem({
    required this.destination,
    required this.selected,
    required this.expansion,
    required this.width,
    required this.onTap,
  });

  final NavigationDestinationData destination;
  final bool selected;
  final double expansion;
  final double width;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final MotionScheme motion = MotionScheme.of(context);
    final double t = expansion;

    // Collapsed: a 56 × 32 indicator with the label below, in a 64 dp item.
    // Expanded: a 56 dp high indicator around icon and label.
    final double height = lerpDouble(64, 56, t)!;
    final Rect indicator = Rect.lerp(
      Rect.fromLTWH((ExpressiveNavigationRail.collapsedWidth - 56) / 2, 4, 56, 32),
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
    final TextStyle collapsedLabel = (selected ? theme.textTheme.labelMedium : theme.textTheme.labelMedium)!.copyWith(
      color: selected ? colors.secondary : colors.onSurfaceVariant,
      fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
    );
    final TextStyle expandedLabel = theme.textTheme.labelLarge!.copyWith(
      color: selected ? colors.onSecondaryContainer : colors.onSurfaceVariant,
      fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
    );

    Widget icon = SpringBuilder(
      value: selected ? 1 : 0,
      spring: motion.fastEffects,
      builder: (BuildContext context, double fill, Widget? _) =>
          Icon(destination.icon, color: iconColor, fill: fill.clamp(0, 1), size: 24),
    );
    if (destination.badgeCount > 0) {
      icon = Badge.count(count: destination.badgeCount, child: icon);
    }

    return Semantics(
      selected: selected,
      button: true,
      label: destination.label,
      excludeSemantics: true,
      child: SizedBox(
        height: height + Gaps.xs,
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
                spring: motion.fastSpatial,
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
              child: IgnorePointer(child: icon),
            ),
            if (collapsedLabelOpacity > 0)
              PositionedDirectional(
                start: 0,
                width: ExpressiveNavigationRail.collapsedWidth,
                top: indicator.bottom + Gaps.xs,
                child: IgnorePointer(
                  child: Opacity(
                    opacity: collapsedLabelOpacity,
                    child: Text(
                      destination.label,
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
                start: indicator.left + 16 + 24 + Gaps.m,
                end: 24,
                top: indicator.center.dy - 10,
                child: IgnorePointer(
                  child: Opacity(
                    opacity: expandedLabelOpacity,
                    child: Text(destination.label, maxLines: 1, overflow: TextOverflow.ellipsis, style: expandedLabel),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
