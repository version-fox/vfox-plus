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

import '../core/settings/app_settings.dart';
import '../l10n/app_localizations.dart';
import '../state/app_model.dart';
import '../state/task_queue.dart';
import 'app_scope.dart';
import 'platform_services.dart';
import 'shell/app_shell.dart';
import 'theme/app_theme.dart';

/// The root widget of vfox+.
class VfoxPlusApp extends StatefulWidget {
  /// Creates the application around [model].
  const VfoxPlusApp({super.key, required this.model, this.services = const DesktopPlatformServices()});

  /// Application state.
  final AppModel model;

  /// Desktop integration.
  final PlatformServices services;

  @override
  State<VfoxPlusApp> createState() => _VfoxPlusAppState();
}

class _VfoxPlusAppState extends State<VfoxPlusApp> {
  final ShellController _shell = ShellController();
  final GlobalKey<ScaffoldMessengerState> _messenger = GlobalKey<ScaffoldMessengerState>();
  late final StreamSubscription<VfoxTask> _taskResults;
  Color? _accentColor;

  @override
  void initState() {
    super.initState();
    _taskResults = widget.model.tasks.onFinished.listen(_announce);
    unawaited(_loadAccentColor());
  }

  Future<void> _loadAccentColor() async {
    final Color? color = await widget.services.accentColor();
    if (mounted && color != null) {
      setState(() => _accentColor = color);
    }
  }

  /// Tells the user how a task ended, unless they are watching the task list.
  void _announce(VfoxTask task) {
    final ScaffoldMessengerState? messenger = _messenger.currentState;
    if (messenger == null || task.status == TaskStatus.cancelled) {
      return;
    }
    final AppLocalizations l10n = AppLocalizations.of(messenger.context);
    final String title = taskTitle(l10n, task.command);
    final bool failed = task.status == TaskStatus.failed;
    if (!failed && _shell.destination == AppDestination.tasks) {
      return;
    }
    messenger
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Text(
            failed ? '${l10n.taskFailed(title)}\n${task.errorMessage ?? ''}'.trim() : l10n.taskSucceeded(title),
          ),
          duration: const Duration(seconds: 4),
          // Failures stay until dismissed, so that they can be read in full.
          persist: failed,
          showCloseIcon: failed,
          action: failed
              ? SnackBarAction(label: l10n.taskView, onPressed: () => _shell.go(AppDestination.tasks))
              : null,
        ),
      );
  }

  @override
  void dispose() {
    unawaited(_taskResults.cancel());
    _shell.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AppScope(
      model: widget.model,
      services: widget.services,
      shell: _shell,
      child: ListenableBuilder(
        listenable: widget.model.settings,
        builder: (BuildContext context, Widget? _) {
          final AppSettings settings = widget.model.settings.settings;
          final Color seed = settings.useSystemColor && _accentColor != null
              ? _accentColor!
              : Color(settings.seedColor);
          final String? language = settings.language.languageCode;
          return MaterialApp(
            onGenerateTitle: (BuildContext context) => AppLocalizations.of(context).appTitle,
            debugShowCheckedModeBanner: false,
            scaffoldMessengerKey: _messenger,
            theme: AppTheme.build(AppTheme.scheme(seed, Brightness.light)),
            darkTheme: AppTheme.build(AppTheme.scheme(seed, Brightness.dark)),
            themeMode: switch (settings.theme) {
              ThemePreference.system => ThemeMode.system,
              ThemePreference.light => ThemeMode.light,
              ThemePreference.dark => ThemeMode.dark,
            },
            locale: language == null ? null : Locale(language),
            supportedLocales: AppLocalizations.supportedLocales,
            localizationsDelegates: const <LocalizationsDelegate<Object?>>[
              AppLocalizations.delegate,
              ...GlobalMaterialLocalizations.delegates,
            ],
            home: const AppShell(),
          );
        },
      ),
    );
  }
}
