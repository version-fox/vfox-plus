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
import 'package:material_ui/material_ui.dart';

import '../../l10n/app_localizations.dart';
import '../../state/app_model.dart';
import '../../state/vfox_controller.dart';
import '../app_scope.dart';
import '../icons/app_icons.g.dart';
import '../pages/home_page.dart';
import '../pages/plugins_page.dart';
import '../pages/sdks_page.dart';
import '../pages/settings_page.dart';
import '../pages/tasks_page.dart';
import '../pages/vfox_unavailable_page.dart';
import '../theme/motion.dart';
import '../theme/tokens.dart';
import '../widgets/navigation_rail.dart';

/// The adaptive frame around the pages.
///
/// Compact windows get a navigation bar at the bottom; wider windows get the
/// expressive navigation rail, which the user can expand to show labels.
/// Switching pages uses the Material fade-through transition.
class AppShell extends StatelessWidget {
  /// Creates the shell.
  const AppShell({super.key});

  @override
  Widget build(BuildContext context) {
    final AppModel model = context.model;
    final ShellController shell = context.shell;
    return ListenableBuilder(
      listenable: Listenable.merge(<Listenable>[shell, model.vfox, model.tasks, model.settings]),
      builder: (BuildContext context, Widget? _) {
        final AppLocalizations l10n = context.l10n;
        final WindowSizeClass size = WindowSizeClass.of(context);
        final AppDestination destination = shell.destination;
        final destinations = <NavigationDestinationData>[
          NavigationDestinationData(icon: AppIcons.home, label: l10n.navHome),
          NavigationDestinationData(icon: AppIcons.sdks, label: l10n.navSdks),
          NavigationDestinationData(icon: AppIcons.plugins, label: l10n.navPlugins),
          NavigationDestinationData(icon: AppIcons.tasks, label: l10n.navTasks, badgeCount: model.tasks.activeCount),
          NavigationDestinationData(icon: AppIcons.settings, label: l10n.navSettings),
        ];

        final Widget page = _PageSwitcher(destination: destination, child: _page(model, destination));

        if (size == WindowSizeClass.compact) {
          return Scaffold(
            body: SafeArea(child: page),
            bottomNavigationBar: NavigationBar(
              selectedIndex: destination.index,
              onDestinationSelected: (int index) => shell.go(AppDestination.values[index]),
              destinations: <Widget>[
                for (final NavigationDestinationData data in destinations)
                  NavigationDestination(
                    icon: _badged(Icon(data.icon), data.badgeCount),
                    selectedIcon: _badged(Icon(data.icon, fill: 1), data.badgeCount),
                    label: data.label,
                  ),
              ],
            ),
          );
        }

        return Scaffold(
          body: SafeArea(
            child: Row(
              children: <Widget>[
                ExpressiveNavigationRail(
                  destinations: destinations,
                  selectedIndex: destination.index,
                  onDestinationSelected: (int index) => shell.go(AppDestination.values[index]),
                  expanded: model.settings.settings.railExpanded,
                  onExpandedChanged: (bool expanded) =>
                      model.settings.update((settings) => settings.copyWith(railExpanded: expanded)),
                  expandTooltip: l10n.navExpand,
                  collapseTooltip: l10n.navCollapse,
                  fabIcon: AppIcons.add,
                  fabLabel: l10n.navAddPlugin,
                  onFabPressed: model.vfox.availability == VfoxAvailability.ready
                      ? () => shell.go(AppDestination.plugins)
                      : null,
                ),
                Expanded(
                  child: Padding(
                    padding: const EdgeInsetsDirectional.only(top: Gaps.l, end: Gaps.l, bottom: Gaps.l),
                    child: page,
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  static Widget _badged(Widget icon, int count) => count > 0 ? Badge.count(count: count, child: icon) : icon;

  static Widget _page(AppModel model, AppDestination destination) {
    final VfoxAvailability availability = model.vfox.availability;
    // Settings stay reachable so that the user can point to the executable.
    if (availability != VfoxAvailability.ready && destination != AppDestination.settings) {
      return const VfoxUnavailablePage();
    }
    return switch (destination) {
      AppDestination.home => const HomePage(),
      AppDestination.sdks => const SdksPage(),
      AppDestination.plugins => const PluginsPage(),
      AppDestination.tasks => const TasksPage(),
      AppDestination.settings => const SettingsPage(),
    };
  }
}

/// Fades through between the pages of the shell.
class _PageSwitcher extends StatelessWidget {
  const _PageSwitcher({required this.destination, required this.child});

  final AppDestination destination;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final MotionScheme motion = MotionScheme.of(context);
    return PageTransitionSwitcher(
      duration: reduceMotion(context) ? Duration.zero : motion.slowEffects.settleDuration,
      transitionBuilder: (Widget child, Animation<double> primary, Animation<double> secondary) {
        return FadeThroughTransition(
          animation: primary,
          secondaryAnimation: secondary,
          fillColor: Colors.transparent,
          child: child,
        );
      },
      child: KeyedSubtree(key: ValueKey<Object>((destination, child.runtimeType)), child: child),
    );
  }
}
