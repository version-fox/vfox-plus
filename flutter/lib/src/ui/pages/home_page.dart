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

import 'dart:async';

import 'package:material_new_shapes/material_new_shapes.dart';
import 'package:material_ui/material_ui.dart' hide MaterialShapes, RoundedPolygon;

import '../../core/models.dart';
import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../platform_services.dart';
import '../theme/app_theme.dart';
import '../theme/tokens.dart';
import '../widgets/list_group.dart';
import '../widgets/page_parts.dart';
import '../widgets/polygon_border.dart';

/// The overview: what is installed, what is in use and where to go next.
class HomePage extends StatelessWidget {
  /// Creates the page.
  const HomePage({super.key});

  /// The line that activates vfox in Bash.
  static const String activationCommand = r'eval "$(vfox activate bash)"';

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    return Pane(
      child: ListenableBuilder(
        listenable: Listenable.merge(<Listenable>[model.installed, model.vfox]),
        builder: (BuildContext context, Widget? _) {
          final List<InstalledSdk>? sdks = model.installed.value;
          if (sdks == null) {
            final Object? error = model.installed.error;
            return error != null
                ? ErrorMessage(title: l10n.sdksLoadError, error: error, onRetry: model.installed.load)
                : const LoadingMessage();
          }
          final List<InstalledSdk> active = sdks.where((InstalledSdk sdk) => sdk.current != null).toList();
          final int versionCount = sdks.fold(0, (int sum, InstalledSdk sdk) => sum + sdk.versions.length);

          return CustomScrollView(
            slivers: <Widget>[
              SliverToBoxAdapter(
                child: PageHeader(
                  title: l10n.appTitle,
                  subtitle: <String>[
                    l10n.homeSubtitle(model.vfox.version ?? '?'),
                    if (model.vfox.executable != null) model.vfox.executable!,
                  ].join(' · '),
                  actions: <Widget>[
                    IconButton(
                      tooltip: l10n.actionRefresh,
                      onPressed: model.installed.isLoading ? null : model.installed.load,
                      icon: const Icon(AppIcons.refresh),
                    ),
                  ],
                ),
              ),
              SliverPadding(
                padding: const EdgeInsets.fromLTRB(Gaps.xl, 0, Gaps.xl, Gaps.xl),
                sliver: SliverToBoxAdapter(
                  child: ContentWidth(
                    maxWidth: 1080,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: <Widget>[
                        _Stats(plugins: sdks.length, versions: versionCount, active: active.length),
                        GroupHeader(title: l10n.homeActiveTitle),
                        if (active.isEmpty)
                          ListGroup(
                            children: <Widget>[
                              ListGroupItem(
                                leading: const Icon(AppIcons.info),
                                title: Text(l10n.homeActiveEmpty),
                                onTap: () => context.shell.showSdk(null),
                              ),
                            ],
                          )
                        else
                          ListGroup(
                            children: <Widget>[
                              for (final InstalledSdk sdk in active)
                                ListGroupItem(
                                  leading: ShapeAvatar(name: sdk.name),
                                  title: Text(sdk.name),
                                  subtitle: Text(sdk.current!),
                                  trailing: const Icon(AppIcons.chevronRight),
                                  onTap: () => context.shell.showSdk(sdk.name),
                                ),
                            ],
                          ),
                        const SizedBox(height: Gaps.xl),
                        const _Shortcuts(),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _Stats extends StatelessWidget {
  const _Stats({required this.plugins, required this.versions, required this.active});

  final int plugins;
  final int versions;
  final int active;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    final AppLocalizations l10n = context.l10n;
    List<Widget> tiles({required bool compact}) => <Widget>[
      _StatTile(
        compact: compact,
        value: plugins,
        label: l10n.homeStatPlugins(plugins),
        icon: AppIcons.plugins,
        shape: MaterialShapes.cookie9Sided,
        background: colors.primaryContainer,
        foreground: colors.onPrimaryContainer,
        onTap: () => context.shell.showSdk(null),
      ),
      _StatTile(
        compact: compact,
        value: versions,
        label: l10n.homeStatVersions(versions),
        icon: AppIcons.versions,
        shape: MaterialShapes.clover4Leaf,
        background: colors.secondaryContainer,
        foreground: colors.onSecondaryContainer,
        onTap: () => context.shell.showSdk(null),
      ),
      _StatTile(
        compact: compact,
        value: active,
        label: l10n.homeStatActive,
        icon: AppIcons.current,
        shape: MaterialShapes.sunny,
        background: colors.tertiaryContainer,
        foreground: colors.onTertiaryContainer,
        onTap: () => context.shell.showSdk(null),
      ),
    ];
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        // Narrow windows keep the three tiles in a row, in a compact size.
        final bool compact = constraints.maxWidth < 560;
        final List<Widget> row = tiles(compact: compact);
        return Row(
          children: <Widget>[
            for (var i = 0; i < row.length; i++) ...<Widget>[
              if (i > 0) SizedBox(width: compact ? Gaps.s : Gaps.m),
              Expanded(child: row[i]),
            ],
          ],
        );
      },
    );
  }
}

class _StatTile extends StatelessWidget {
  const _StatTile({
    required this.compact,
    required this.value,
    required this.label,
    required this.icon,
    required this.shape,
    required this.background,
    required this.foreground,
    required this.onTap,
  });

  final bool compact;
  final int value;
  final String label;
  final IconData icon;
  final RoundedPolygon shape;
  final Color background;
  final Color foreground;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final EmphasizedTextTheme emphasized = EmphasizedTextTheme.of(context);
    return Card(
      color: background,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.extraLarge))),
      child: InkWell(
        onTap: onTap,
        child: SizedBox(
          height: 136,
          child: Stack(
            children: <Widget>[
              // A large, partly hidden shape gives the tile its character.
              PositionedDirectional(
                end: -28,
                bottom: -36,
                child: DecoratedBox(
                  decoration: ShapeDecoration(
                    color: foreground.withValues(alpha: 0.08),
                    shape: PolygonBorder(polygon: shape),
                  ),
                  child: const SizedBox.square(dimension: 140),
                ),
              ),
              Padding(
                padding: EdgeInsets.all(compact ? Gaps.l : Gaps.xl),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Icon(icon, color: foreground, fill: 1),
                    const Spacer(),
                    Text(
                      '$value',
                      style: (compact ? emphasized.headlineLarge : emphasized.displaySmall).copyWith(
                        color: foreground,
                        height: 1,
                      ),
                    ),
                    const SizedBox(height: Gaps.xs),
                    Text(
                      label,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: emphasized.labelLarge.copyWith(color: foreground),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Shortcuts extends StatelessWidget {
  const _Shortcuts();

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final AppLocalizations l10n = context.l10n;
    final Widget shell = Card(
      color: colors.surfaceContainerHighest,
      child: Padding(
        padding: const EdgeInsets.all(Gaps.xl),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              children: <Widget>[
                Icon(AppIcons.terminal, color: colors.primary),
                const SizedBox(width: Gaps.m),
                Expanded(child: Text(l10n.homeShellTitle, style: EmphasizedTextTheme.of(context).titleMedium)),
              ],
            ),
            const SizedBox(height: Gaps.s),
            Text(l10n.homeShellBody, style: theme.textTheme.bodyMedium?.copyWith(color: colors.onSurfaceVariant)),
            const SizedBox(height: Gaps.m),
            Material(
              color: colors.surfaceContainerLowest,
              shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.medium))),
              child: Padding(
                padding: const EdgeInsetsDirectional.only(start: Gaps.l, end: Gaps.xs),
                child: Row(
                  children: <Widget>[
                    Expanded(
                      child: SelectableText(
                        HomePage.activationCommand,
                        style: theme.textTheme.bodyMedium?.copyWith(fontFamily: 'monospace'),
                      ),
                    ),
                    IconButton(
                      tooltip: l10n.actionCopy,
                      icon: const Icon(AppIcons.copy),
                      onPressed: () async {
                        final ScaffoldMessengerState messenger = ScaffoldMessenger.of(context);
                        await context.services.copyText(HomePage.activationCommand);
                        messenger.showSnackBar(SnackBar(content: Text(l10n.actionCopied)));
                      },
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
    final Widget explore = _ShortcutCard(
      icon: AppIcons.explore,
      title: l10n.homeExploreTitle,
      body: l10n.homeExploreBody,
      onTap: () => context.shell.go(AppDestination.plugins),
    );
    final Widget docs = _ShortcutCard(
      icon: AppIcons.docs,
      title: l10n.homeDocsTitle,
      body: l10n.homeDocsBody,
      trailing: AppIcons.openInNew,
      onTap: () => unawaited(context.services.openUrl(Links.vfoxDocs)),
    );
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        if (constraints.maxWidth < 760) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              shell,
              const SizedBox(height: Gaps.m),
              explore,
              const SizedBox(height: Gaps.m),
              docs,
            ],
          );
        }
        return IntrinsicHeight(
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              Expanded(flex: 3, child: shell),
              const SizedBox(width: Gaps.m),
              Expanded(
                flex: 2,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[
                    Expanded(child: explore),
                    const SizedBox(height: Gaps.m),
                    Expanded(child: docs),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _ShortcutCard extends StatelessWidget {
  const _ShortcutCard({
    required this.icon,
    required this.title,
    required this.body,
    required this.onTap,
    this.trailing = AppIcons.chevronRight,
  });

  final IconData icon;
  final String title;
  final String body;
  final IconData trailing;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    return Card(
      color: colors.surfaceContainer,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(Gaps.l),
          child: Row(
            children: <Widget>[
              DecoratedBox(
                decoration: ShapeDecoration(
                  color: colors.secondaryContainer,
                  shape: PolygonBorder(polygon: MaterialShapes.cookie6Sided),
                ),
                child: SizedBox.square(dimension: 48, child: Icon(icon, color: colors.onSecondaryContainer)),
              ),
              const SizedBox(width: Gaps.l),
              Expanded(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(title, style: EmphasizedTextTheme.of(context).titleSmall),
                    const SizedBox(height: Gaps.xs),
                    Text(body, style: theme.textTheme.bodySmall?.copyWith(color: colors.onSurfaceVariant)),
                  ],
                ),
              ),
              Icon(trailing, color: colors.onSurfaceVariant),
            ],
          ),
        ),
      ),
    );
  }
}
