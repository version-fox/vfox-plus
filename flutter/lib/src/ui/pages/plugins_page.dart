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

import 'package:material_ui/material_ui.dart';

import '../../core/models.dart';
import '../../core/vfox/vfox_command.dart';
import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../../state/task_queue.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../theme/app_theme.dart';
import '../theme/tokens.dart';
import '../widgets/button_group.dart';
import '../widgets/dialogs.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/page_parts.dart';
import '../widgets/polygon_border.dart';

enum _PluginFilter { all, official, community }

/// The plugin registry: every plugin that `vfox add` can install.
class PluginsPage extends StatefulWidget {
  /// Creates the page.
  const PluginsPage({super.key});

  @override
  State<PluginsPage> createState() => _PluginsPageState();
}

class _PluginsPageState extends State<PluginsPage> {
  final TextEditingController _search = TextEditingController();
  _PluginFilter _filter = _PluginFilter.all;
  bool _requested = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_requested) {
      _requested = true;
      unawaited(context.model.registry.ensureLoaded());
    }
  }

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  Future<void> _addCustom() async {
    final AppModel model = context.model;
    final CustomPluginSource? source = await askCustomPlugin(context);
    if (source != null) {
      await model.addPluginFromSource(source.source, alias: source.alias);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    return Pane(
      child: ListenableBuilder(
        listenable: Listenable.merge(<Listenable>[model.registry, model.installed, model.tasks, _search]),
        builder: (BuildContext context, Widget? _) {
          final List<AvailablePlugin>? plugins = model.registry.value;
          final Set<String> installed = <String>{
            for (final InstalledSdk sdk in model.installed.value ?? const <InstalledSdk>[]) sdk.name,
          };
          final String query = _search.text.trim().toLowerCase();
          final List<AvailablePlugin> visible = <AvailablePlugin>[
            for (final AvailablePlugin plugin in plugins ?? const <AvailablePlugin>[])
              if (switch (_filter) {
                    _PluginFilter.all => true,
                    _PluginFilter.official => plugin.official,
                    _PluginFilter.community => !plugin.official,
                  } &&
                  (query.isEmpty || plugin.name.toLowerCase().contains(query) || plugin.homepage.contains(query)))
                plugin,
          ];
          final VfoxTask? updatingAll = model.tasks.activeTask(operation: VfoxOperation.updateAllPlugins);

          return CustomScrollView(
            slivers: <Widget>[
              SliverToBoxAdapter(
                child: PageHeader(
                  title: l10n.pluginsTitle,
                  subtitle: l10n.pluginsSubtitle,
                  actions: <Widget>[
                    if (updatingAll != null)
                      const LoadingIndicator(size: 40)
                    else
                      IconButton(
                        tooltip: l10n.pluginsUpdateAll,
                        onPressed: installed.isEmpty ? null : () => model.run(model.cli.updateAllPlugins()),
                        icon: const Icon(AppIcons.update),
                      ),
                    IconButton(
                      tooltip: l10n.actionRefresh,
                      onPressed: model.registry.isLoading ? null : model.registry.load,
                      icon: const Icon(AppIcons.refresh),
                    ),
                    FilledButton.tonalIcon(
                      onPressed: _addCustom,
                      icon: const Icon(AppIcons.add),
                      label: Text(l10n.pluginsAddCustom),
                    ),
                  ],
                ),
              ),
              SliverPadding(
                padding: const EdgeInsets.fromLTRB(Gaps.xl, 0, Gaps.xl, Gaps.l),
                sliver: SliverToBoxAdapter(
                  child: Wrap(
                    spacing: Gaps.m,
                    runSpacing: Gaps.m,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: <Widget>[
                      ConstrainedBox(
                        constraints: const BoxConstraints(maxWidth: 420),
                        child: SearchBar(
                          controller: _search,
                          hintText: l10n.pluginsSearchHint,
                          leading: const Icon(AppIcons.search),
                        ),
                      ),
                      ConnectedButtonGroup<_PluginFilter>(
                        segments: <ButtonGroupSegment<_PluginFilter>>[
                          ButtonGroupSegment<_PluginFilter>(value: _PluginFilter.all, label: l10n.pluginsFilterAll),
                          ButtonGroupSegment<_PluginFilter>(
                            value: _PluginFilter.official,
                            label: l10n.pluginsFilterOfficial,
                            icon: AppIcons.verified,
                          ),
                          ButtonGroupSegment<_PluginFilter>(
                            value: _PluginFilter.community,
                            label: l10n.pluginsFilterCommunity,
                          ),
                        ],
                        selected: _filter,
                        onSelected: (_PluginFilter filter) => setState(() => _filter = filter),
                      ),
                    ],
                  ),
                ),
              ),
              if (plugins == null)
                SliverFillRemaining(
                  hasScrollBody: false,
                  child: model.registry.error != null
                      ? ErrorMessage(
                          title: l10n.pluginsLoadError,
                          error: model.registry.error!,
                          onRetry: model.registry.load,
                        )
                      : const LoadingMessage(),
                )
              else if (visible.isEmpty)
                SliverFillRemaining(
                  hasScrollBody: false,
                  child: StatusMessage(icon: AppIcons.notFound, title: l10n.pluginsNoMatch),
                )
              else
                SliverPadding(
                  padding: const EdgeInsets.fromLTRB(Gaps.xl, 0, Gaps.xl, Gaps.xl),
                  sliver: SliverGrid.builder(
                    gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                      maxCrossAxisExtent: 340,
                      mainAxisExtent: 152,
                      crossAxisSpacing: Gaps.m,
                      mainAxisSpacing: Gaps.m,
                    ),
                    itemCount: visible.length,
                    itemBuilder: (BuildContext context, int index) {
                      final AvailablePlugin plugin = visible[index];
                      return _PluginCard(
                        plugin: plugin,
                        installed: installed.contains(plugin.name),
                        task: model.tasks.activeTask(operation: VfoxOperation.addPlugin, sdk: plugin.name),
                      );
                    },
                  ),
                ),
            ],
          );
        },
      ),
    );
  }
}

class _PluginCard extends StatelessWidget {
  const _PluginCard({required this.plugin, required this.installed, required this.task});

  final AvailablePlugin plugin;
  final bool installed;
  final VfoxTask? task;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final AppLocalizations l10n = context.l10n;
    final AppModel model = context.model;
    final Uri? homepage = Uri.tryParse(plugin.homepage);
    final String source = homepage == null || homepage.pathSegments.isEmpty
        ? plugin.homepage
        : homepage.pathSegments.where((String segment) => segment.isNotEmpty).join('/');

    final Widget action;
    if (task != null) {
      action = const LoadingIndicator(size: 40);
    } else if (installed) {
      action = TextButton.icon(
        onPressed: () => context.shell.showSdk(plugin.name),
        icon: const Icon(AppIcons.check),
        label: Text(l10n.pluginsAdded),
      );
    } else {
      action = FilledButton.tonalIcon(
        onPressed: () => model.run(model.cli.addPlugin(plugin.name)),
        icon: const Icon(AppIcons.add),
        label: Text(l10n.pluginsAdd),
      );
    }

    return Card(
      color: installed ? colors.surfaceContainerHighest : colors.surfaceContainer,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(Corners.largeIncreased))),
      child: Padding(
        padding: const EdgeInsets.all(Gaps.l),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              children: <Widget>[
                ShapeAvatar(name: plugin.name, size: 44),
                const SizedBox(width: Gaps.m),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      Row(
                        children: <Widget>[
                          Flexible(
                            child: Text(
                              plugin.name,
                              style: EmphasizedTextTheme.of(context).titleMedium,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          if (plugin.official) ...<Widget>[
                            const SizedBox(width: Gaps.xs),
                            Tooltip(
                              message: l10n.pluginsOfficial,
                              child: Icon(AppIcons.verified, size: 18, fill: 1, color: colors.primary),
                            ),
                          ],
                        ],
                      ),
                      Text(
                        source,
                        style: theme.textTheme.bodySmall?.copyWith(color: colors.onSurfaceVariant),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const Spacer(),
            Row(
              children: <Widget>[
                IconButton(
                  tooltip: l10n.actionOpenHomepage,
                  onPressed: homepage == null ? null : () => unawaited(context.services.openUrl(homepage)),
                  icon: const Icon(AppIcons.openInNew),
                ),
                const Spacer(),
                action,
              ],
            ),
          ],
        ),
      ),
    );
  }
}
