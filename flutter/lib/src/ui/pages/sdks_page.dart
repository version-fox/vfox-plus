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

import 'package:animations/animations.dart';
import 'package:material_new_shapes/material_new_shapes.dart';
import 'package:material_ui/material_ui.dart';

import '../../core/models.dart';
import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../theme/motion.dart';
import '../theme/tokens.dart';
import '../widgets/list_group.dart';
import '../widgets/page_parts.dart';
import '../widgets/polygon_border.dart';
import 'sdk_detail_view.dart';

/// Installed SDKs, using the list-detail canonical layout.
///
/// Wide windows show the list and the selected SDK side by side. Narrow
/// windows show one of them at a time and move between them along the
/// horizontal shared axis.
class SdksPage extends StatefulWidget {
  /// Creates the page.
  const SdksPage({super.key});

  /// The narrowest width that shows list and detail side by side.
  static const double twoPaneWidth = 840;

  @override
  State<SdksPage> createState() => _SdksPageState();
}

class _SdksPageState extends State<SdksPage> {
  final TextEditingController _search = TextEditingController();

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final ShellController shell = context.shell;
    return ListenableBuilder(
      listenable: Listenable.merge(<Listenable>[model.installed, shell]),
      builder: (BuildContext context, Widget? _) {
        final List<InstalledSdk>? sdks = model.installed.value;
        if (sdks == null) {
          final Object? error = model.installed.error;
          return Pane(
            child: error != null
                ? ErrorMessage(title: context.l10n.sdksLoadError, error: error, onRetry: model.installed.load)
                : const LoadingMessage(),
          );
        }
        final String? selected = sdks.any((InstalledSdk sdk) => sdk.name == shell.selectedSdk)
            ? shell.selectedSdk
            : null;

        return LayoutBuilder(
          builder: (BuildContext context, BoxConstraints constraints) {
            final bool twoPane = constraints.maxWidth >= SdksPage.twoPaneWidth;
            final Widget list = _SdkList(
              sdks: sdks,
              search: _search,
              selected: twoPane ? selected : null,
              showChevrons: !twoPane,
              onSelected: shell.showSdk,
            );
            if (twoPane) {
              return Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  SizedBox(width: 360, child: Pane(child: list)),
                  const SizedBox(width: Gaps.xl),
                  Expanded(
                    child: Pane(
                      child: PageTransitionSwitcher(
                        duration: _duration(context),
                        transitionBuilder: (Widget child, Animation<double> primary, Animation<double> secondary) =>
                            FadeThroughTransition(
                              animation: primary,
                              secondaryAnimation: secondary,
                              fillColor: Colors.transparent,
                              child: child,
                            ),
                        child: selected == null
                            ? const _SelectPrompt(key: ValueKey<String>(''))
                            : SdkDetailView(key: ValueKey<String>(selected), name: selected),
                      ),
                    ),
                  ),
                ],
              );
            }
            return PageTransitionSwitcher(
              duration: _duration(context),
              reverse: selected == null,
              transitionBuilder: (Widget child, Animation<double> primary, Animation<double> secondary) =>
                  SharedAxisTransition(
                    animation: primary,
                    secondaryAnimation: secondary,
                    transitionType: SharedAxisTransitionType.horizontal,
                    fillColor: Colors.transparent,
                    child: child,
                  ),
              child: selected == null
                  ? Pane(key: const ValueKey<String>(''), child: list)
                  : Pane(
                      key: ValueKey<String>(selected),
                      child: SdkDetailView(name: selected, onBack: () => shell.showSdk(null)),
                    ),
            );
          },
        );
      },
    );
  }

  static Duration _duration(BuildContext context) =>
      reduceMotion(context) ? Duration.zero : MotionScheme.of(context).slowEffects.settleDuration;
}

class _SdkList extends StatelessWidget {
  const _SdkList({
    required this.sdks,
    required this.search,
    required this.selected,
    required this.showChevrons,
    required this.onSelected,
  });

  final List<InstalledSdk> sdks;
  final TextEditingController search;
  final String? selected;
  final bool showChevrons;
  final ValueChanged<String?> onSelected;

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final AppLocalizations l10n = context.l10n;
    if (sdks.isEmpty) {
      return StatusMessage(
        icon: AppIcons.empty,
        shape: MaterialShapes.flower,
        title: l10n.sdksEmptyTitle,
        body: l10n.sdksEmptyBody,
        actions: <Widget>[
          FilledButton.icon(
            onPressed: () => context.shell.go(AppDestination.plugins),
            icon: const Icon(AppIcons.add),
            label: Text(l10n.navAddPlugin),
          ),
        ],
      );
    }
    return ListenableBuilder(
      listenable: search,
      builder: (BuildContext context, Widget? _) {
        final String query = search.text.trim().toLowerCase();
        final List<InstalledSdk> visible = sdks
            .where((InstalledSdk sdk) => query.isEmpty || sdk.name.toLowerCase().contains(query))
            .toList();
        return CustomScrollView(
          slivers: <Widget>[
            SliverToBoxAdapter(
              child: PageHeader(
                title: l10n.navSdks,
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
              padding: const EdgeInsets.fromLTRB(Gaps.l, 0, Gaps.l, Gaps.m),
              sliver: SliverToBoxAdapter(
                child: SearchBar(
                  controller: search,
                  hintText: l10n.sdksSearchHint,
                  leading: const Icon(AppIcons.search),
                  trailing: <Widget>[
                    if (search.text.isNotEmpty)
                      IconButton(
                        tooltip: MaterialLocalizations.of(context).deleteButtonTooltip,
                        onPressed: search.clear,
                        icon: const Icon(AppIcons.close),
                      ),
                  ],
                ),
              ),
            ),
            if (visible.isEmpty)
              SliverFillRemaining(
                hasScrollBody: false,
                child: StatusMessage(icon: AppIcons.notFound, title: l10n.sdksNoMatch(search.text.trim())),
              )
            else
              SliverPadding(
                padding: const EdgeInsets.fromLTRB(Gaps.l, 0, Gaps.l, Gaps.l),
                sliver: SliverList.separated(
                  itemCount: visible.length,
                  separatorBuilder: (BuildContext context, int index) => const SizedBox(height: Gaps.hairline),
                  itemBuilder: (BuildContext context, int index) {
                    final InstalledSdk sdk = visible[index];
                    return ListGroupItem(
                      position: ListGroupPosition.of(index, visible.length),
                      selected: sdk.name == selected,
                      leading: ShapeAvatar(name: sdk.name),
                      title: Text(sdk.name),
                      subtitle: Text(
                        '${sdk.current != null ? l10n.sdksCurrent(sdk.current!) : l10n.sdksNoCurrent}\n'
                        '${l10n.sdksVersionCount(sdk.versions.length)}',
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      trailing: showChevrons ? const Icon(AppIcons.chevronRight) : null,
                      onTap: () => onSelected(sdk.name),
                    );
                  },
                ),
              ),
          ],
        );
      },
    );
  }
}

class _SelectPrompt extends StatelessWidget {
  const _SelectPrompt({super.key});

  @override
  Widget build(BuildContext context) {
    return StatusMessage(icon: AppIcons.sdks, shape: MaterialShapes.gem, title: context.l10n.sdksSelectPrompt);
  }
}
