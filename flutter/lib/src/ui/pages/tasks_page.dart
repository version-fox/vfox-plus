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
import 'package:material_ui/material_ui.dart';

import '../../core/vfox/vfox_command.dart';
import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../../state/task_queue.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../theme/app_theme.dart';
import '../theme/motion.dart';
import '../theme/tokens.dart';
import '../widgets/list_group.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/page_parts.dart';
import '../widgets/wavy_progress_indicator.dart';

/// Running, queued and finished operations with their output.
class TasksPage extends StatelessWidget {
  /// Creates the page.
  const TasksPage({super.key});

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    return Pane(
      child: ListenableBuilder(
        listenable: model.tasks,
        builder: (BuildContext context, Widget? _) {
          final List<VfoxTask> tasks = model.tasks.tasks;
          return CustomScrollView(
            slivers: <Widget>[
              SliverToBoxAdapter(
                child: PageHeader(
                  title: l10n.navTasks,
                  actions: <Widget>[
                    TextButton.icon(
                      onPressed: model.tasks.hasFinished ? model.tasks.clearFinished : null,
                      icon: const Icon(AppIcons.succeeded),
                      label: Text(l10n.tasksClearFinished),
                    ),
                  ],
                ),
              ),
              if (tasks.isEmpty)
                SliverFillRemaining(
                  hasScrollBody: false,
                  child: StatusMessage(
                    icon: AppIcons.tasks,
                    shape: MaterialShapes.sunny,
                    title: l10n.tasksEmptyTitle,
                    body: l10n.tasksEmptyBody,
                  ),
                )
              else
                SliverPadding(
                  padding: const EdgeInsets.fromLTRB(Gaps.xl, 0, Gaps.xl, Gaps.xl),
                  sliver: SliverList.separated(
                    itemCount: tasks.length,
                    separatorBuilder: (BuildContext context, int index) => const SizedBox(height: Gaps.hairline),
                    itemBuilder: (BuildContext context, int index) => _TaskItem(
                      key: ObjectKey(tasks[index]),
                      task: tasks[index],
                      position: ListGroupPosition.of(index, tasks.length),
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

class _TaskItem extends StatefulWidget {
  const _TaskItem({super.key, required this.task, required this.position});

  final VfoxTask task;
  final ListGroupPosition position;

  @override
  State<_TaskItem> createState() => _TaskItemState();
}

class _TaskItemState extends State<_TaskItem> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = context.l10n;
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final AppModel model = context.model;
    final MotionScheme motion = MotionScheme.of(context);
    final VfoxTask task = widget.task;

    return ListenableBuilder(
      listenable: task,
      builder: (BuildContext context, Widget? _) {
        final TaskStatus status = task.status;
        final Widget leading = switch (status) {
          TaskStatus.queued => Icon(AppIcons.queued, color: colors.onSurfaceVariant),
          TaskStatus.running => const LoadingIndicator(size: 32),
          TaskStatus.succeeded => Icon(AppIcons.succeeded, color: colors.primary, fill: 1),
          TaskStatus.failed => Icon(AppIcons.error, color: colors.error, fill: 1),
          TaskStatus.cancelled => Icon(AppIcons.cancel, color: colors.onSurfaceVariant),
        };
        final String statusText = switch (status) {
          TaskStatus.queued => l10n.taskStatusQueued,
          TaskStatus.running =>
            task.progress == null ? l10n.taskStatusRunning : l10n.taskDownloading((task.progress! * 100).round()),
          TaskStatus.succeeded => l10n.taskStatusSucceeded,
          TaskStatus.failed => '${l10n.taskStatusFailed} · ${task.errorMessage ?? ''}',
          TaskStatus.cancelled => l10n.taskStatusCancelled,
        };

        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            ListGroupItem(
              position: _expanded && widget.position == ListGroupPosition.last
                  ? ListGroupPosition.middle
                  : _expanded && widget.position == ListGroupPosition.single
                  ? ListGroupPosition.first
                  : widget.position,
              leading: SizedBox.square(dimension: 32, child: Center(child: leading)),
              title: Text(taskTitle(l10n, task.command), style: EmphasizedTextTheme.of(context).bodyLarge),
              subtitle: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    statusText,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: status == TaskStatus.failed ? TextStyle(color: colors.error) : null,
                  ),
                  if (status == TaskStatus.running && task.command.operation == VfoxOperation.install) ...<Widget>[
                    const SizedBox(height: Gaps.s),
                    ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 480),
                      child: WavyProgressIndicator(value: task.progress),
                    ),
                  ],
                ],
              ),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  if (status.isActive)
                    IconButton(
                      tooltip: l10n.actionCancel,
                      onPressed: () => model.tasks.cancel(task),
                      icon: const Icon(AppIcons.cancel),
                    )
                  else if (status == TaskStatus.failed || status == TaskStatus.cancelled)
                    IconButton(
                      tooltip: l10n.actionRetry,
                      onPressed: () => model.tasks.retry(task),
                      icon: const Icon(AppIcons.retry),
                    ),
                  IconButton(
                    tooltip: _expanded ? l10n.taskHideOutput : l10n.taskShowOutput,
                    onPressed: () => setState(() => _expanded = !_expanded),
                    icon: SpringBuilder(
                      value: _expanded ? 0.5 : 0,
                      spring: motion.fastSpatial,
                      builder: (BuildContext context, double turns, Widget? child) =>
                          Transform.rotate(angle: turns * 6.283185307179586, child: child),
                      child: const Icon(AppIcons.expandMore),
                    ),
                  ),
                ],
              ),
              onTap: () => setState(() => _expanded = !_expanded),
            ),
            AnimatedSize(
              duration: motion.defaultSpatial.settleDuration,
              curve: motion.defaultSpatial.curve,
              alignment: AlignmentDirectional.topStart,
              child: _expanded
                  ? Padding(
                      padding: const EdgeInsets.only(top: Gaps.hairline),
                      child: _TaskOutput(
                        task: task,
                        isLast:
                            widget.position == ListGroupPosition.last || widget.position == ListGroupPosition.single,
                      ),
                    )
                  : const SizedBox(width: double.infinity),
            ),
          ],
        );
      },
    );
  }
}

class _TaskOutput extends StatelessWidget {
  const _TaskOutput({required this.task, required this.isLast});

  final VfoxTask task;
  final bool isLast;

  @override
  Widget build(BuildContext context) {
    final ThemeData theme = Theme.of(context);
    final ColorScheme colors = theme.colorScheme;
    final AppLocalizations l10n = context.l10n;
    final List<String> lines = task.log;
    final String text = <String>['\$ ${task.command}', ...lines].join('\n');
    return Material(
      color: colors.surfaceContainerHighest,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(
          top: const Radius.circular(Corners.extraSmall),
          bottom: Radius.circular(isLast ? Corners.largeIncreased : Corners.extraSmall),
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(Gaps.l, Gaps.s, Gaps.xs, Gaps.m),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Expanded(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxHeight: 320),
                child: SingleChildScrollView(
                  reverse: true,
                  child: SelectableText(
                    lines.isEmpty ? '\$ ${task.command}\n${l10n.taskNoOutput}' : text,
                    style: theme.textTheme.bodySmall?.copyWith(fontFamily: 'monospace', height: 1.5),
                  ),
                ),
              ),
            ),
            IconButton(
              tooltip: l10n.taskCopyOutput,
              icon: const Icon(AppIcons.copy),
              onPressed: () async {
                final ScaffoldMessengerState messenger = ScaffoldMessenger.of(context);
                await context.services.copyText(text);
                messenger.showSnackBar(SnackBar(content: Text(l10n.actionCopied)));
              },
            ),
          ],
        ),
      ),
    );
  }
}
