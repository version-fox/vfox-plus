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

/// Where an item sits in a [ListGroup], which decides its corners.
enum ListGroupPosition {
  /// The only item: all corners large.
  single,

  /// The first item: large top corners.
  first,

  /// An item between others: small corners.
  middle,

  /// The last item: large bottom corners.
  last;

  /// The position of item [index] of [count].
  static ListGroupPosition of(int index, int count) {
    if (count <= 1) {
      return single;
    }
    if (index == 0) {
      return first;
    }
    return index == count - 1 ? last : middle;
  }
}

/// A Material 3 Expressive segmented list: items are separate surfaces, 2 dp
/// apart, that together read as one rounded group.
///
/// Outer corners of the group are large and inner corners small. When an
/// item is hovered or pressed its inner corners grow, so the item briefly
/// stands out as its own shape.
class ListGroup extends StatelessWidget {
  /// Creates a group of [children], usually [ListGroupItem]s.
  const ListGroup({super.key, required this.children});

  /// The items.
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        for (var i = 0; i < children.length; i++) ...<Widget>[
          if (i > 0) const SizedBox(height: Gaps.hairline),
          _ListGroupScope(position: ListGroupPosition.of(i, children.length), child: children[i]),
        ],
      ],
    );
  }
}

class _ListGroupScope extends InheritedWidget {
  const _ListGroupScope({required this.position, required super.child});

  final ListGroupPosition position;

  @override
  bool updateShouldNotify(_ListGroupScope oldWidget) => oldWidget.position != position;
}

/// An item of a [ListGroup] with the layout of a Material list item.
class ListGroupItem extends StatefulWidget {
  /// Creates an item.
  const ListGroupItem({
    super.key,
    required this.title,
    this.leading,
    this.subtitle,
    this.trailing,
    this.onTap,
    this.selected = false,
    this.position,
    this.padding = const EdgeInsetsDirectional.symmetric(horizontal: Gaps.l, vertical: Gaps.m),
    this.wrapTrailing = false,
  });

  /// Widget before the text, usually an icon or an avatar.
  final Widget? leading;

  /// Primary text.
  final Widget title;

  /// Secondary text.
  final Widget? subtitle;

  /// Widget after the text, such as a switch or buttons.
  final Widget? trailing;

  /// Called when the item is activated. `null` makes the item static.
  final VoidCallback? onTap;

  /// Whether the item is the selected one.
  final bool selected;

  /// Position in the group. Defaults to the position given by an enclosing
  /// [ListGroup]; required for items built lazily in a sliver.
  final ListGroupPosition? position;

  /// Padding around the content.
  final EdgeInsetsGeometry padding;

  /// Whether a wide [trailing] widget, such as a button group or a text
  /// field, moves below the text when the item is narrower than
  /// [wrapWidth].
  final bool wrapTrailing;

  /// The width below which [wrapTrailing] takes effect.
  static const double wrapWidth = 560;

  @override
  State<ListGroupItem> createState() => _ListGroupItemState();
}

class _ListGroupItemState extends State<ListGroupItem> {
  bool _hovered = false;
  bool _pressed = false;

  static const double _outer = Corners.largeIncreased;
  static const double _inner = Corners.extraSmall;
  static const double _innerActive = Corners.medium;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final MotionScheme motion = MotionScheme.of(context);
    final ListGroupPosition position =
        widget.position ??
        context.dependOnInheritedWidgetOfExactType<_ListGroupScope>()?.position ??
        ListGroupPosition.single;
    final bool interactive = widget.onTap != null;
    final bool active = interactive && (_hovered || _pressed) || widget.selected;

    final Color background = widget.selected ? colors.secondaryContainer : colors.surfaceContainer;
    final Color foreground = widget.selected ? colors.onSecondaryContainer : colors.onSurface;

    final Widget? trailing = widget.trailing == null
        ? null
        : IconTheme.merge(
            data: IconThemeData(color: colors.onSurfaceVariant),
            child: widget.trailing!,
          );
    final Widget text = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        DefaultTextStyle.merge(
          style: theme.textTheme.bodyLarge?.copyWith(color: foreground),
          child: widget.title,
        ),
        if (widget.subtitle != null)
          DefaultTextStyle.merge(
            style: theme.textTheme.bodyMedium?.copyWith(
              color: widget.selected ? colors.onSecondaryContainer : colors.onSurfaceVariant,
            ),
            child: widget.subtitle!,
          ),
      ],
    );
    final Widget content = Padding(
      padding: widget.padding,
      child: LayoutBuilder(
        builder: (BuildContext context, BoxConstraints constraints) {
          final bool wrap = widget.wrapTrailing && trailing != null && constraints.maxWidth < ListGroupItem.wrapWidth;
          return Row(
            children: <Widget>[
              if (widget.leading != null) ...<Widget>[
                IconTheme.merge(
                  data: IconThemeData(color: widget.selected ? colors.onSecondaryContainer : colors.onSurfaceVariant),
                  child: widget.leading!,
                ),
                const SizedBox(width: Gaps.l),
              ],
              Expanded(
                child: wrap
                    ? Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisSize: MainAxisSize.min,
                        children: <Widget>[
                          text,
                          const SizedBox(height: Gaps.s),
                          Align(alignment: AlignmentDirectional.centerStart, child: trailing),
                        ],
                      )
                    : text,
              ),
              if (trailing != null && !wrap) ...<Widget>[const SizedBox(width: Gaps.l), trailing],
            ],
          );
        },
      ),
    );

    return SpringBuilder(
      value: active ? 1 : 0,
      spring: motion.fastSpatial,
      builder: (BuildContext context, double activation, Widget? child) {
        final double inner = lerpDouble(_inner, _innerActive, activation)!.clamp(0, _outer);
        final Radius top = Radius.circular(
          position == ListGroupPosition.first || position == ListGroupPosition.single ? _outer : inner,
        );
        final Radius bottom = Radius.circular(
          position == ListGroupPosition.last || position == ListGroupPosition.single ? _outer : inner,
        );
        final shape = RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: top, bottom: bottom),
        );
        return Material(
          color: background,
          shape: shape,
          clipBehavior: Clip.antiAlias,
          animationDuration: motion.fastEffects.settleDuration,
          child: interactive
              ? InkWell(
                  customBorder: shape,
                  onTap: widget.onTap,
                  onHover: (bool value) => setState(() => _hovered = value),
                  onHighlightChanged: (bool value) => setState(() => _pressed = value),
                  child: child,
                )
              : child,
        );
      },
      child: Semantics(selected: widget.selected, child: content),
    );
  }
}

/// A heading above a [ListGroup].
class GroupHeader extends StatelessWidget {
  /// Creates a heading with [title] and optional [trailing] actions.
  const GroupHeader({super.key, required this.title, this.trailing});

  /// The heading text.
  final String title;

  /// Widgets at the end of the heading, such as buttons.
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsetsDirectional.only(start: Gaps.l, end: Gaps.xs, top: Gaps.xl, bottom: Gaps.s),
      child: Row(
        children: <Widget>[
          Expanded(
            child: Semantics(
              header: true,
              child: Text(title, style: theme.textTheme.titleSmall?.copyWith(color: theme.colorScheme.primary)),
            ),
          ),
          ?trailing,
        ],
      ),
    );
  }
}
