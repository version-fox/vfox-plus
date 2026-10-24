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

import 'package:material_new_shapes/material_new_shapes.dart';
import 'package:material_ui/material_ui.dart' hide MaterialShapes, RoundedPolygon;

import '../../core/vfox/vfox_command.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../theme/app_theme.dart';
import '../theme/tokens.dart';
import 'loading_indicator.dart';
import 'polygon_border.dart';

/// A rounded surface that holds the content of a page.
///
/// Pages sit on panes next to the navigation rail, following the Material 3
/// canonical layouts. Panes are 24 dp apart and have extra-large corners.
class Pane extends StatelessWidget {
  /// Creates a pane around [child].
  const Pane({super.key, required this.child, this.color});

  /// The content.
  final Widget child;

  /// Background colour. Defaults to the low surface container colour.
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Material(
      color: color ?? colors.surfaceContainerLow,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.extraLarge))),
      clipBehavior: Clip.antiAlias,
      child: child,
    );
  }
}

/// The large, emphasized header of a page.
class PageHeader extends StatelessWidget {
  /// Creates a header.
  const PageHeader({super.key, required this.title, this.subtitle, this.actions = const <Widget>[], this.leading});

  /// Page title.
  final String title;

  /// Optional line below the title.
  final String? subtitle;

  /// Buttons at the end of the header.
  final List<Widget> actions;

  /// Optional widget before the title, such as a back button.
  final Widget? leading;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final EmphasizedTextTheme emphasized = EmphasizedTextTheme.of(context);
    return Padding(
      padding: const EdgeInsetsDirectional.fromSTEB(Gaps.xl, Gaps.xl, Gaps.l, Gaps.l),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: <Widget>[
          if (leading != null) ...<Widget>[leading!, const SizedBox(width: Gaps.s)],
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Semantics(
                  header: true,
                  child: Text(
                    title,
                    style: emphasized.headlineMedium.copyWith(color: theme.colorScheme.onSurface),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (subtitle != null) ...<Widget>[
                  const SizedBox(height: Gaps.xs),
                  Text(
                    subtitle!,
                    style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ],
            ),
          ),
          if (actions.isNotEmpty) ...<Widget>[
            const SizedBox(width: Gaps.l),
            Wrap(spacing: Gaps.s, crossAxisAlignment: WrapCrossAlignment.center, children: actions),
          ],
        ],
      ),
    );
  }
}

/// Centres page content and limits its width for comfortable reading.
class ContentWidth extends StatelessWidget {
  /// Creates the constraint around [child].
  const ContentWidth({super.key, required this.child, this.maxWidth = 960});

  /// The content.
  final Widget child;

  /// The widest the content may get.
  final double maxWidth;

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: AlignmentDirectional.topCenter,
      child: ConstrainedBox(
        constraints: BoxConstraints(maxWidth: maxWidth),
        child: child,
      ),
    );
  }
}

/// A friendly placeholder for empty, unavailable or failed content, with an
/// expressive shape behind its icon.
class StatusMessage extends StatelessWidget {
  /// Creates a placeholder.
  const StatusMessage({
    super.key,
    required this.icon,
    required this.title,
    this.body,
    this.actions = const <Widget>[],
    this.shape,
    this.isError = false,
  });

  /// Icon on the shape.
  final IconData icon;

  /// Headline.
  final String title;

  /// Explanation.
  final String? body;

  /// Buttons below the text.
  final List<Widget> actions;

  /// The decorative shape. Defaults to a 9-sided cookie.
  final RoundedPolygon? shape;

  /// Whether to use error colours.
  final bool isError;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(Gaps.xxl),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 440),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              DecoratedBox(
                decoration: ShapeDecoration(
                  color: isError ? colors.errorContainer : colors.secondaryContainer,
                  shape: PolygonBorder(polygon: shape ?? MaterialShapes.cookie9Sided),
                ),
                child: SizedBox.square(
                  dimension: 112,
                  child: Icon(
                    icon,
                    size: 48,
                    fill: 1,
                    color: isError ? colors.onErrorContainer : colors.onSecondaryContainer,
                  ),
                ),
              ),
              const SizedBox(height: Gaps.xl),
              Text(
                title,
                textAlign: TextAlign.center,
                style: EmphasizedTextTheme.of(context).titleLarge.copyWith(color: colors.onSurface),
              ),
              if (body != null) ...<Widget>[
                const SizedBox(height: Gaps.s),
                Text(
                  body!,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium?.copyWith(color: colors.onSurfaceVariant),
                ),
              ],
              if (actions.isNotEmpty) ...<Widget>[
                const SizedBox(height: Gaps.xl),
                Wrap(spacing: Gaps.s, runSpacing: Gaps.s, alignment: WrapAlignment.center, children: actions),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// A [StatusMessage] for an error, with a retry button and the error text.
class ErrorMessage extends StatelessWidget {
  /// Creates an error placeholder.
  const ErrorMessage({super.key, required this.title, required this.error, this.onRetry});

  /// What failed.
  final String title;

  /// The error, shown as details.
  final Object error;

  /// Retries the failed operation.
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    return StatusMessage(
      icon: AppIcons.error,
      shape: MaterialShapes.softBurst,
      isError: true,
      title: title,
      body: describeError(error),
      actions: <Widget>[
        if (onRetry != null)
          FilledButton.tonalIcon(
            onPressed: onRetry,
            icon: const Icon(AppIcons.retry),
            label: Text(context.l10n.actionRetry),
          ),
      ],
    );
  }
}

/// A centred loading indicator.
class LoadingMessage extends StatelessWidget {
  /// Creates a loading placeholder with an optional [label].
  const LoadingMessage({super.key, this.label});

  /// Text below the indicator.
  final String? label;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(Gaps.xxl),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            LoadingIndicator(contained: true, size: 64, semanticsLabel: label ?? context.l10n.loading),
            if (label != null) ...<Widget>[
              const SizedBox(height: Gaps.l),
              Text(
                label!,
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// A short, user-facing description of [error].
String describeError(Object error) => switch (error) {
  VfoxException(:final String message) => message,
  ArgumentError(:final Object? message) => '$message',
  _ => '$error',
};
