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

import 'package:material_ui/material_ui.dart';

import '../core/vfox/vfox_command.dart';
import '../l10n/app_localizations.dart';
import '../state/app_model.dart';
import 'platform_services.dart';

/// The top-level destinations of the application.
enum AppDestination {
  /// Overview.
  home,

  /// Installed SDKs and their versions.
  sdks,

  /// The plugin registry.
  plugins,

  /// Running and finished operations.
  tasks,

  /// Settings.
  settings,
}

/// Navigation state of the application shell.
class ShellController extends ChangeNotifier {
  AppDestination _destination = AppDestination.home;

  /// The visible destination.
  AppDestination get destination => _destination;

  String? _selectedSdk;

  /// The SDK shown on the SDKs page, if any.
  String? get selectedSdk => _selectedSdk;

  /// Shows [destination].
  void go(AppDestination destination) {
    if (destination != _destination) {
      _destination = destination;
      notifyListeners();
    }
  }

  /// Shows the SDKs page with [name] selected, or with no selection.
  void showSdk(String? name) {
    _destination = AppDestination.sdks;
    _selectedSdk = name;
    notifyListeners();
  }
}

/// Makes the application state available to every widget.
class AppScope extends InheritedWidget {
  /// Creates the scope.
  const AppScope({super.key, required this.model, required this.services, required this.shell, required super.child});

  /// Application state.
  final AppModel model;

  /// Desktop integration.
  final PlatformServices services;

  /// Navigation state.
  final ShellController shell;

  /// The closest scope. The objects it holds never change, so looking them
  /// up does not make the caller rebuild.
  static AppScope of(BuildContext context) {
    final AppScope? scope = context.getInheritedWidgetOfExactType<AppScope>();
    assert(scope != null, 'No AppScope found in the widget tree.');
    return scope!;
  }

  @override
  bool updateShouldNotify(AppScope oldWidget) =>
      oldWidget.model != model || oldWidget.services != services || oldWidget.shell != shell;
}

/// Shortcuts for widgets.
extension AppContext on BuildContext {
  /// The translations of the current locale.
  AppLocalizations get l10n => AppLocalizations.of(this);

  /// The application state.
  AppModel get model => AppScope.of(this).model;

  /// Desktop integration.
  PlatformServices get services => AppScope.of(this).services;

  /// Navigation state.
  ShellController get shell => AppScope.of(this).shell;
}

/// The localized title of a task that runs [command].
String taskTitle(AppLocalizations l10n, VfoxCommand command) {
  final String subject = command.subject ?? '';
  return switch (command.operation) {
    VfoxOperation.install => l10n.taskInstall(subject),
    VfoxOperation.uninstall => l10n.taskUninstall(subject),
    VfoxOperation.use => l10n.taskUse(subject),
    VfoxOperation.unuse => l10n.taskUnuse(subject),
    VfoxOperation.addPlugin => subject.isEmpty ? l10n.taskAddPluginFromSource : l10n.taskAddPlugin(subject),
    VfoxOperation.removePlugin => l10n.taskRemovePlugin(subject),
    VfoxOperation.updatePlugin => l10n.taskUpdatePlugin(subject),
    VfoxOperation.updateAllPlugins => l10n.taskUpdateAllPlugins,
  };
}
