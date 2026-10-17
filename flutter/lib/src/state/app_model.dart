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

import 'package:flutter/foundation.dart';

import '../core/models.dart';
import '../core/process_runner.dart';
import '../core/settings/app_settings.dart';
import '../core/settings/settings_store.dart';
import '../core/vfox/plugin_source.dart';
import '../core/vfox/vfox_cli.dart';
import '../core/vfox/vfox_command.dart';
import '../core/vfox/vfox_locator.dart';
import 'async_resource.dart';
import 'sdk_detail_controller.dart';
import 'settings_controller.dart';
import 'task_queue.dart';
import 'vfox_config_controller.dart';
import 'vfox_controller.dart';

/// The application state: every controller, wired together.
///
/// The model owns the [VfoxCli] and keeps the derived data consistent:
/// installed SDKs are reloaded whenever vfox becomes available or a task
/// that changes them finishes.
class AppModel {
  /// Creates the model.
  AppModel({
    required ProcessRunner runner,
    required SettingsStore settingsStore,
    required Map<String, String> environment,
    VfoxLocator? locator,
  }) : cli = VfoxCli(
         runner: runner,
         launchConfig: VfoxLaunchConfig(executable: VfoxLocator.executableName, baseEnvironment: environment),
       ),
       settings = SettingsController(settingsStore) {
    vfox = VfoxController(
      cli: cli,
      locator: locator ?? VfoxLocator(environment: environment),
      environment: environment,
    );
    installed = AsyncResource<List<InstalledSdk>>(cli.installedSdks);
    registry = AsyncResource<List<AvailablePlugin>>(cli.availablePlugins);
    vfoxConfig = VfoxConfigController(cli);
    tasks = TaskQueue(cli: cli);
    _taskSubscription = tasks.onFinished.listen(_onTaskFinished);
    vfox.addListener(_onVfoxChanged);
    settings.addListener(_onSettingsChanged);
  }

  /// The vfox client shared by all controllers.
  final VfoxCli cli;

  /// Application preferences.
  final SettingsController settings;

  /// Availability of the vfox executable.
  late final VfoxController vfox;

  /// Installed plugins and their versions.
  late final AsyncResource<List<InstalledSdk>> installed;

  /// Plugins available in the registry.
  late final AsyncResource<List<AvailablePlugin>> registry;

  /// The vfox configuration.
  late final VfoxConfigController vfoxConfig;

  /// State-changing operations.
  late final TaskQueue tasks;

  late final StreamSubscription<VfoxTask> _taskSubscription;
  final Map<VfoxTask, PreparedPluginSource> _pluginCopies = <VfoxTask, PreparedPluginSource>{};
  VfoxAvailability _lastAvailability = VfoxAvailability.unknown;
  AppSettings? _appliedSettings;

  /// Loads the settings and probes vfox.
  Future<void> start() async {
    await settings.load();
    await _applySettings();
  }

  /// The installed SDK called [name], if it is installed.
  InstalledSdk? installedSdk(String name) {
    for (final InstalledSdk sdk in installed.value ?? const <InstalledSdk>[]) {
      if (sdk.name == name) {
        return sdk;
      }
    }
    return null;
  }

  /// Creates the controller for the detail view of [name].
  SdkDetailController sdkDetail(String name) => SdkDetailController(cli: cli, name: name);

  /// Queues [command] and returns its task.
  VfoxTask run(VfoxCommand command) => tasks.enqueue(command);

  /// Adds a plugin from [source], a URL or a local `.zip` or `.lua` file,
  /// optionally under [alias].
  ///
  /// vfox moves a local `.lua` file away instead of reading it, so such a file
  /// is copied into the temporary directory of vfox first and vfox gets the
  /// copy; the copy is deleted once the task has finished.
  Future<VfoxTask> addPluginFromSource(String source, {String? alias}) async {
    final PreparedPluginSource prepared = await preparePluginSource(
      source,
      temporaryRoot: vfoxTemporaryDirectory(cli.launchConfig.environment()),
    );
    final VfoxTask task = run(cli.addPlugin('', source: prepared.path, alias: alias));
    if (prepared.isCopy) {
      _pluginCopies[task] = prepared;
    }
    return task;
  }

  void _onVfoxChanged() {
    final VfoxAvailability availability = vfox.availability;
    if (availability == _lastAvailability) {
      return;
    }
    _lastAvailability = availability;
    if (availability == VfoxAvailability.ready) {
      unawaited(installed.load());
      if (registry.hasValue || registry.error != null) {
        unawaited(registry.load());
      }
      if (vfoxConfig.hasValue || vfoxConfig.error != null) {
        unawaited(vfoxConfig.load());
      }
    }
  }

  void _onSettingsChanged() {
    final AppSettings? previous = _appliedSettings;
    final AppSettings current = settings.settings;
    if (previous != null &&
        (previous.vfoxExecutable != current.vfoxExecutable ||
            !mapEquals(previous.extraEnvironment, current.extraEnvironment))) {
      unawaited(_applySettings());
    }
  }

  Future<void> _applySettings() async {
    _appliedSettings = settings.settings;
    // Force a reload of the dependent data even if vfox stays available.
    _lastAvailability = VfoxAvailability.unknown;
    await vfox.apply(settings.settings);
  }

  void _onTaskFinished(VfoxTask task) {
    unawaited(_pluginCopies.remove(task)?.dispose());
    if (task.status == TaskStatus.cancelled && task.startedAt == null) {
      return;
    }
    unawaited(installed.load());
  }

  /// Releases every resource.
  void dispose() {
    unawaited(_taskSubscription.cancel());
    for (final PreparedPluginSource copy in _pluginCopies.values) {
      unawaited(copy.dispose());
    }
    _pluginCopies.clear();
    vfox.removeListener(_onVfoxChanged);
    settings.removeListener(_onSettingsChanged);
    tasks.dispose();
    installed.dispose();
    registry.dispose();
    vfoxConfig.dispose();
    vfox.dispose();
    settings.dispose();
  }
}
