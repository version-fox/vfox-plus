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
import '../../core/vfox/vfox_command.dart';
import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../../state/sdk_detail_controller.dart';
import '../../state/task_queue.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../theme/app_theme.dart';
import '../theme/tokens.dart';
import '../widgets/button_group.dart';
import '../widgets/dialogs.dart';
import '../widgets/list_group.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/page_parts.dart';
import '../widgets/polygon_border.dart';
import '../widgets/split_button.dart';
import '../widgets/wavy_progress_indicator.dart';

enum _Tab { installed, available }

/// Everything about one installed plugin: its metadata, its installed
/// versions and the versions that can be installed.
class SdkDetailView extends StatefulWidget {
  /// Creates the view for the plugin [name].
  const SdkDetailView({super.key, required this.name, this.onBack});

  /// The plugin name.
  final String name;

  /// Called by the back button, which is only shown when given.
  final VoidCallback? onBack;

  @override
  State<SdkDetailView> createState() => _SdkDetailViewState();
}

class _SdkDetailViewState extends State<SdkDetailView> {
  SdkDetailController? _controller;
  StreamSubscription<VfoxTask>? _taskResults;
  final TextEditingController _filter = TextEditingController();
  _Tab _tab = _Tab.installed;

  SdkDetailController get controller => _controller!;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_controller == null) {
      final AppModel model = context.model;
      _controller = model.sdkDetail(widget.name);
      unawaited(controller.info.load());
      _filter.addListener(() => controller.query = _filter.text);
      _taskResults = model.tasks.onFinished.listen((VfoxTask task) {
        if (task.command.sdk == widget.name && task.command.operation == VfoxOperation.updatePlugin) {
          unawaited(controller.info.load());
        }
      });
    }
  }

  @override
  void dispose() {
    unawaited(_taskResults?.cancel());
    _controller?.dispose();
    _filter.dispose();
    super.dispose();
  }

  void _showTab(_Tab tab) {
    setState(() => _tab = tab);
    if (tab == _Tab.available) {
      unawaited(controller.versions.ensureLoaded());
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    return ListenableBuilder(
      listenable: Listenable.merge(<Listenable>[model.installed, model.tasks, controller, controller.info]),
      builder: (BuildContext context, Widget? _) {
        final InstalledSdk sdk = model.installedSdk(widget.name) ?? InstalledSdk(name: widget.name, versions: const []);
        return CustomScrollView(
          slivers: <Widget>[
            SliverToBoxAdapter(
              child: _Header(sdk: sdk, info: controller.info.value, onBack: widget.onBack),
            ),
            SliverPadding(
              padding: const EdgeInsets.symmetric(horizontal: Gaps.xl),
              sliver: SliverToBoxAdapter(
                child: Align(
                  alignment: AlignmentDirectional.centerStart,
                  child: ConnectedButtonGroup<_Tab>(
                    segments: <ButtonGroupSegment<_Tab>>[
                      ButtonGroupSegment<_Tab>(
                        value: _Tab.installed,
                        label: '${l10n.sdkTabInstalled} (${sdk.versions.length})',
                        icon: AppIcons.versions,
                      ),
                      ButtonGroupSegment<_Tab>(
                        value: _Tab.available,
                        label: l10n.sdkTabAvailable,
                        icon: AppIcons.install,
                      ),
                    ],
                    selected: _tab,
                    onSelected: _showTab,
                  ),
                ),
              ),
            ),
            const SliverToBoxAdapter(child: SizedBox(height: Gaps.l)),
            ...switch (_tab) {
              _Tab.installed => _installedSlivers(context, sdk),
              _Tab.available => _availableSlivers(context, sdk),
            },
            const SliverToBoxAdapter(child: SizedBox(height: Gaps.xl)),
          ],
        );
      },
    );
  }

  List<Widget> _installedSlivers(BuildContext context, InstalledSdk sdk) {
    final AppLocalizations l10n = context.l10n;
    if (sdk.versions.isEmpty) {
      return <Widget>[
        SliverToBoxAdapter(
          child: StatusMessage(
            icon: AppIcons.install,
            shape: MaterialShapes.pill,
            title: l10n.sdkInstalledEmpty(sdk.name),
            actions: <Widget>[
              FilledButton.icon(
                onPressed: () => _showTab(_Tab.available),
                icon: const Icon(AppIcons.install),
                label: Text(l10n.sdkBrowseVersions),
              ),
            ],
          ),
        ),
      ];
    }
    return <Widget>[
      SliverPadding(
        padding: const EdgeInsets.symmetric(horizontal: Gaps.xl),
        sliver: SliverList.separated(
          itemCount: sdk.versions.length,
          separatorBuilder: (BuildContext context, int index) => const SizedBox(height: Gaps.hairline),
          itemBuilder: (BuildContext context, int index) => _InstalledVersionItem(
            sdk: sdk,
            version: sdk.versions[index],
            position: ListGroupPosition.of(index, sdk.versions.length),
          ),
        ),
      ),
    ];
  }

  List<Widget> _availableSlivers(BuildContext context, InstalledSdk sdk) {
    final AppLocalizations l10n = context.l10n;
    final Set<String> installed = sdk.versions.toSet();
    final List<AvailableVersion> visible = controller.visibleVersions(installed: installed);
    final Widget toolbar = Padding(
      padding: const EdgeInsets.fromLTRB(Gaps.xl, 0, Gaps.xl, Gaps.m),
      child: Wrap(
        spacing: Gaps.m,
        runSpacing: Gaps.s,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: <Widget>[
          SizedBox(
            width: 280,
            child: TextField(
              controller: _filter,
              decoration: InputDecoration(
                hintText: l10n.availableSearchHint,
                prefixIcon: const Icon(AppIcons.search),
                isDense: true,
              ),
            ),
          ),
          ConnectedButtonGroup<VersionFilter>(
            segments: <ButtonGroupSegment<VersionFilter>>[
              ButtonGroupSegment<VersionFilter>(value: VersionFilter.all, label: l10n.availableFilterAll),
              ButtonGroupSegment<VersionFilter>(value: VersionFilter.withNote, label: l10n.availableFilterTagged),
              ButtonGroupSegment<VersionFilter>(value: VersionFilter.installed, label: l10n.availableFilterInstalled),
            ],
            selected: controller.filter,
            onSelected: (VersionFilter filter) => controller.filter = filter,
          ),
          if (controller.versions.hasValue)
            Text(
              l10n.availableCount(visible.length),
              style: Theme.of(context).textTheme.labelLarge
                  ?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant),
            ),
        ],
      ),
    );

    if (!controller.versions.hasValue) {
      final Object? error = controller.versions.error;
      return <Widget>[
        SliverToBoxAdapter(
          child: error != null
              ? ErrorMessage(title: l10n.availableError(sdk.name), error: error, onRetry: controller.versions.load)
              : LoadingMessage(label: l10n.availableLoading),
        ),
      ];
    }
    return <Widget>[
      SliverToBoxAdapter(child: toolbar),
      if (visible.isEmpty)
        SliverToBoxAdapter(
          child: StatusMessage(icon: AppIcons.notFound, title: l10n.availableEmpty),
        )
      else
        SliverPadding(
          padding: const EdgeInsets.symmetric(horizontal: Gaps.xl),
          sliver: SliverList.separated(
            itemCount: visible.length,
            separatorBuilder: (BuildContext context, int index) => const SizedBox(height: Gaps.hairline),
            itemBuilder: (BuildContext context, int index) => _AvailableVersionItem(
              sdk: sdk.name,
              version: visible[index],
              installed: installed.contains(visible[index].version),
              position: ListGroupPosition.of(index, visible.length),
            ),
          ),
        ),
    ];
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.sdk, required this.info, required this.onBack});

  final InstalledSdk sdk;
  final PluginInfo? info;
  final VoidCallback? onBack;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final AppLocalizations l10n = context.l10n;
    final AppModel model = context.model;
    final VfoxTask? updating = model.tasks.activeTask(operation: VfoxOperation.updatePlugin, sdk: sdk.name);
    final PluginInfo? info = this.info;

    final Widget text = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          sdk.name,
          style: EmphasizedTextTheme.of(context).headlineMedium.copyWith(color: colors.onSurface),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        const SizedBox(height: Gaps.xs),
        Wrap(
          spacing: Gaps.s,
          runSpacing: Gaps.xs,
          children: <Widget>[
            _Tag(
              label: sdk.current == null ? l10n.sdksNoCurrent : l10n.sdksCurrent(sdk.current!),
              icon: sdk.current == null ? null : AppIcons.current,
              emphasized: sdk.current != null,
            ),
            if (info != null && info.version.isNotEmpty) _Tag(label: l10n.sdkPluginVersion(info.version)),
          ],
        ),
        if (info != null && info.description.isNotEmpty) ...<Widget>[
          const SizedBox(height: Gaps.s),
          Text(
            info.description,
            style: theme.textTheme.bodyMedium?.copyWith(color: colors.onSurfaceVariant),
            maxLines: 3,
            overflow: TextOverflow.ellipsis,
          ),
        ],
      ],
    );
    final Widget? back = onBack == null
        ? null
        : IconButton(
            tooltip: MaterialLocalizations.of(context).backButtonTooltip,
            onPressed: onBack,
            icon: const Icon(AppIcons.back),
          );
    final List<Widget> actions = <Widget>[
      if (updating != null)
        const Padding(padding: EdgeInsets.all(Gaps.xs), child: LoadingIndicator(size: 40))
      else
        IconButton(
          tooltip: l10n.sdkUpdatePlugin,
          onPressed: () => model.run(model.cli.updatePlugin(sdk.name)),
          icon: const Icon(AppIcons.update),
        ),
      MenuAnchor(
        menuChildren: <Widget>[
          if (info != null && info.homepage.isNotEmpty)
            MenuItemButton(
              leadingIcon: const Icon(AppIcons.openInNew),
              onPressed: () => unawaited(context.services.openUrl(Uri.parse(info.homepage))),
              child: Text(l10n.actionOpenHomepage),
            ),
          MenuItemButton(
            leadingIcon: Icon(AppIcons.removePlugin, color: colors.error),
            onPressed: () => unawaited(_remove(context)),
            child: Text(l10n.sdkRemovePlugin, style: TextStyle(color: colors.error)),
          ),
        ],
        builder: (BuildContext context, MenuController controller, Widget? _) => IconButton(
          tooltip: l10n.sdkMoreActions,
          onPressed: () => controller.isOpen ? controller.close() : controller.open(),
          icon: const Icon(AppIcons.more),
        ),
      ),
    ];

    return Padding(
      padding: const EdgeInsetsDirectional.fromSTEB(Gaps.l, Gaps.xl, Gaps.l, Gaps.xl),
      child: LayoutBuilder(
        builder: (BuildContext context, BoxConstraints constraints) {
          // Narrow panes put the buttons on their own row above the name.
          if (constraints.maxWidth < 520) {
            return Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                Row(children: <Widget>[?back, const Spacer(), ...actions]),
                const SizedBox(height: Gaps.s),
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    const SizedBox(width: Gaps.s),
                    ShapeAvatar(name: sdk.name, size: 56),
                    const SizedBox(width: Gaps.l),
                    Expanded(child: text),
                  ],
                ),
              ],
            );
          }
          return Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              if (back != null) ...<Widget>[back, const SizedBox(width: Gaps.xs)] else const SizedBox(width: Gaps.s),
              ShapeAvatar(name: sdk.name, size: 72),
              const SizedBox(width: Gaps.l),
              Expanded(child: text),
              const SizedBox(width: Gaps.s),
              ...actions,
            ],
          );
        },
      ),
    );
  }

  Future<void> _remove(BuildContext context) async {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    final ShellController shell = context.shell;
    final bool confirmed = await confirmAction(
      context,
      title: l10n.sdkRemoveTitle(sdk.name),
      body: l10n.sdkRemoveBody(sdk.versions.length),
      confirmLabel: l10n.sdkRemoveConfirm,
      icon: AppIcons.removePlugin,
      destructive: true,
    );
    if (confirmed) {
      model.run(model.cli.removePlugin(sdk.name));
      shell.showSdk(null);
    }
  }
}

/// A small label such as "In use: 24.11.1" or "LTS".
class _Tag extends StatelessWidget {
  const _Tag({required this.label, this.icon, this.emphasized = false});

  final String label;
  final IconData? icon;
  final bool emphasized;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final Color background = emphasized ? colors.tertiaryContainer : colors.surfaceContainerHighest;
    final Color foreground = emphasized ? colors.onTertiaryContainer : colors.onSurfaceVariant;
    return DecoratedBox(
      decoration: ShapeDecoration(color: background, shape: const StadiumBorder()),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: Gaps.s + 2, vertical: 3),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            if (icon != null) ...<Widget>[Icon(icon, size: 16, color: foreground, fill: 1), const SizedBox(width: 4)],
            Flexible(
              child: Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.labelMedium?.copyWith(color: foreground),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _InstalledVersionItem extends StatelessWidget {
  const _InstalledVersionItem({required this.sdk, required this.version, required this.position});

  final InstalledSdk sdk;
  final String version;
  final ListGroupPosition position;

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    final ColorScheme colors = Theme.of(context).colorScheme;
    final bool current = sdk.current == version;
    final VfoxTask? task = model.tasks.activeTask(sdk: sdk.name, version: version);

    final Widget actions;
    if (task != null) {
      actions = const LoadingIndicator(size: 40);
    } else {
      actions = Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          if (current)
            _Tag(label: l10n.versionInUse, icon: AppIcons.current, emphasized: true)
          else
            SplitButton(
              label: l10n.versionUse,
              icon: AppIcons.check,
              menuTooltip: l10n.versionUseOptions,
              onPressed: () => model.run(model.cli.use(sdk.name, version)),
              menuChildren: <Widget>[
                MenuItemButton(
                  leadingIcon: const Icon(AppIcons.globalScope),
                  onPressed: () => model.run(model.cli.use(sdk.name, version)),
                  child: Text(l10n.versionUseGlobally),
                ),
                MenuItemButton(
                  leadingIcon: const Icon(AppIcons.projectScope),
                  onPressed: () => unawaited(_useInProject(context)),
                  child: Text(l10n.versionUseInProject),
                ),
              ],
            ),
          const SizedBox(width: Gaps.xs),
          MenuAnchor(
            menuChildren: <Widget>[
              if (current) ...<Widget>[
                MenuItemButton(
                  leadingIcon: const Icon(AppIcons.projectScope),
                  onPressed: () => unawaited(_useInProject(context)),
                  child: Text(l10n.versionUseInProject),
                ),
                MenuItemButton(
                  leadingIcon: const Icon(AppIcons.cancel),
                  onPressed: () => model.run(model.cli.unuse(sdk.name)),
                  child: Text(l10n.versionUnuse),
                ),
              ],
              MenuItemButton(
                leadingIcon: const Icon(AppIcons.copy),
                onPressed: () => unawaited(_copyPath(context)),
                child: Text(l10n.versionCopyPath),
              ),
              MenuItemButton(
                leadingIcon: Icon(AppIcons.uninstall, color: colors.error),
                onPressed: () => unawaited(_uninstall(context, current)),
                child: Text(l10n.versionUninstall, style: TextStyle(color: colors.error)),
              ),
            ],
            builder: (BuildContext context, MenuController controller, Widget? _) => IconButton(
              tooltip: l10n.versionMoreActions(version),
              onPressed: () => controller.isOpen ? controller.close() : controller.open(),
              icon: const Icon(AppIcons.more),
            ),
          ),
        ],
      );
    }

    return ListGroupItem(
      position: position,
      wrapTrailing: true,
      leading: Icon(
        current ? AppIcons.current : AppIcons.versions,
        fill: current ? 1 : 0,
        color: current ? colors.primary : null,
      ),
      title: Text(
        version,
        style: current ? EmphasizedTextTheme.of(context).bodyLarge.copyWith(color: colors.onSurface) : null,
      ),
      trailing: actions,
    );
  }

  Future<void> _useInProject(BuildContext context) async {
    final AppModel model = context.model;
    final String? directory = await askProjectDirectory(context, name: sdk.name, version: version);
    if (directory != null) {
      model.run(model.cli.use(sdk.name, version, scope: UseScope.project, projectDirectory: directory));
    }
  }

  Future<void> _copyPath(BuildContext context) async {
    final AppModel model = context.model;
    final ScaffoldMessengerState messenger = ScaffoldMessenger.of(context);
    final AppLocalizations l10n = context.l10n;
    final String? path = await model.cli.installPath(sdk.name, version);
    if (path != null && context.mounted) {
      await context.services.copyText(path);
      messenger.showSnackBar(SnackBar(content: Text(l10n.actionCopied)));
    }
  }

  Future<void> _uninstall(BuildContext context, bool current) async {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    final bool confirmed = await confirmAction(
      context,
      title: l10n.versionUninstallTitle(sdk.name, version),
      body: current ? l10n.versionUninstallCurrentBody : l10n.versionUninstallBody,
      confirmLabel: l10n.versionUninstall,
      icon: AppIcons.uninstall,
      destructive: true,
    );
    if (confirmed) {
      model.run(model.cli.uninstall(sdk.name, version));
    }
  }
}

class _AvailableVersionItem extends StatelessWidget {
  const _AvailableVersionItem({
    required this.sdk,
    required this.version,
    required this.installed,
    required this.position,
  });

  final String sdk;
  final AvailableVersion version;
  final bool installed;
  final ListGroupPosition position;

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    final VfoxTask? task = model.tasks.activeTask(operation: VfoxOperation.install, sdk: sdk, version: version.version);
    final String additions = version.additions.map((VersionAddition addition) => '$addition').join(' · ');

    final Widget trailing;
    if (task != null) {
      trailing = _InstallProgress(task: task);
    } else if (installed) {
      trailing = TextButton.icon(onPressed: null, icon: const Icon(AppIcons.check), label: Text(l10n.versionInstalled));
    } else {
      trailing = FilledButton.tonalIcon(
        onPressed: () => model.run(model.cli.install(sdk, version.version)),
        icon: const Icon(AppIcons.install),
        label: Text(l10n.versionInstall),
      );
    }

    return ListGroupItem(
      position: position,
      wrapTrailing: true,
      padding: const EdgeInsetsDirectional.symmetric(horizontal: Gaps.l, vertical: Gaps.s),
      title: Wrap(
        spacing: Gaps.s,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: <Widget>[
          Text(version.version),
          if (version.note != null) _Tag(label: version.note!, emphasized: true),
        ],
      ),
      subtitle: additions.isEmpty ? null : Text(additions),
      trailing: trailing,
    );
  }
}

class _InstallProgress extends StatelessWidget {
  const _InstallProgress({required this.task});

  final VfoxTask task;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = context.l10n;
    final ThemeData theme = Theme.of(context);
    return ListenableBuilder(
      listenable: task,
      builder: (BuildContext context, Widget? _) {
        final double? progress = task.progress;
        return SizedBox(
          width: 220,
          child: Row(
            children: <Widget>[
              Expanded(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    WavyProgressIndicator(value: progress, semanticsLabel: taskTitle(l10n, task.command)),
                    const SizedBox(height: Gaps.xs),
                    Text(
                      progress == null
                          ? (task.status == TaskStatus.queued ? l10n.taskStatusQueued : l10n.taskStatusRunning)
                          : l10n.taskDownloading((progress * 100).round()),
                      style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                    ),
                  ],
                ),
              ),
              IconButton(
                tooltip: l10n.actionCancel,
                onPressed: () => context.model.tasks.cancel(task),
                icon: const Icon(AppIcons.cancel),
              ),
            ],
          ),
        );
      },
    );
  }
}
