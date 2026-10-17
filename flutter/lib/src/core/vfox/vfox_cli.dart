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

/// The client that drives the vfox command line interface.
library;

import 'dart:io';

import '../models.dart';
import '../process_runner.dart';
import 'vfox_command.dart';
import 'vfox_output.dart';

/// How vfox processes are launched.
final class VfoxLaunchConfig {
  /// Creates a launch configuration.
  const VfoxLaunchConfig({
    required this.executable,
    this.baseEnvironment = const <String, String>{},
    this.extraEnvironment = const <String, String>{},
    this.neutralDirectory = '/',
  });

  /// Path to the vfox executable, or a bare name to look up in `PATH`.
  final String executable;

  /// The environment inherited from the application, usually
  /// `Platform.environment`.
  final Map<String, String> baseEnvironment;

  /// Variables added by the user, for example download mirrors such as
  /// `VFOX_NODEJS_MIRROR`. They override [baseEnvironment].
  final Map<String, String> extraEnvironment;

  /// Working directory for commands that are not tied to a project.
  ///
  /// vfox reads a project `.vfox.toml` (and legacy files such as `.nvmrc`)
  /// from its working directory, so global queries must run somewhere that
  /// cannot contain one. The filesystem root is used by default.
  final String neutralDirectory;

  /// Variables that tie a process to an activated shell session. They are
  /// removed so that a GUI started from an activated terminal does not act
  /// on that terminal's session.
  static const Set<String> sessionVariables = <String>{
    '__VFOX_SHELL',
    '__VFOX_PID',
    '__VFOX_CURTMPPATH',
    '__VFOX_INITIALIZED',
  };

  /// The complete environment of a vfox child process.
  Map<String, String> environment({bool hookShell = false}) {
    final env = <String, String>{...baseEnvironment, ...extraEnvironment}
      ..removeWhere((String key, String _) => sessionVariables.contains(key));
    // Disable colours; vfox (through pterm) honours https://no-color.org.
    env['NO_COLOR'] = '1';
    if (hookShell) {
      env['__VFOX_SHELL'] = 'bash';
    }
    return env;
  }

  /// Returns a copy with the given fields replaced.
  VfoxLaunchConfig copyWith({
    String? executable,
    Map<String, String>? baseEnvironment,
    Map<String, String>? extraEnvironment,
    String? neutralDirectory,
  }) {
    return VfoxLaunchConfig(
      executable: executable ?? this.executable,
      baseEnvironment: baseEnvironment ?? this.baseEnvironment,
      extraEnvironment: extraEnvironment ?? this.extraEnvironment,
      neutralDirectory: neutralDirectory ?? this.neutralDirectory,
    );
  }
}

/// Drives the vfox CLI.
///
/// Read-only queries are executed directly and parsed into models.
/// State-changing operations are returned as [VfoxCommand] values instead,
/// so that the caller can queue them, stream their output and retry them.
///
/// Every command line built here is documented in
/// `docs/vfox-cli-contract.md`.
final class VfoxCli {
  /// Creates a client that launches vfox through `runner`.
  VfoxCli({required this._runner, required this.launchConfig});

  final ProcessRunner _runner;

  /// How vfox is launched.
  ///
  /// Can be replaced at any time, for example after the user picked a
  /// different executable. Affects subsequent invocations only.
  VfoxLaunchConfig launchConfig;

  // ---------------------------------------------------------------------
  // Queries
  // ---------------------------------------------------------------------

  /// The version of vfox, for example `1.0.12`.
  Future<String> version() async {
    final ProcessOutput output = await _query(const <String>['--version']);
    final String? version = parseVfoxVersion('${output.stdout}\n${output.stderr}');
    if (version == null) {
      throw VfoxException(
        'Unrecognised version output',
        arguments: const <String>['--version'],
        stdout: output.stdout,
        stderr: output.stderr,
      );
    }
    return version;
  }

  /// All installed plugins with their installed and current versions.
  ///
  /// Runs `vfox list` and `vfox current` and merges their results.
  Future<List<InstalledSdk>> installedSdks() async {
    final List<ProcessOutput> outputs = await Future.wait(<Future<ProcessOutput>>[
      _query(const <String>['list'], allowFailure: true),
      _query(const <String>['current']),
    ]);
    final ProcessOutput list = outputs[0];
    final Map<String, String?> current = parseCurrent(outputs[1].stdout);
    if (list.exitCode != 0) {
      // `vfox list` fails with "you don't have any sdk installed yet" when no
      // plugin is installed; `vfox current` then prints nothing.
      if (current.isEmpty) {
        return const <InstalledSdk>[];
      }
      throw _failure(const <String>['list'], list);
    }
    return mergeInstalledSdks(parseInstalledTree(list.stdout), current);
  }

  /// Metadata of the installed plugin [name].
  Future<PluginInfo> pluginInfo(String name) async {
    final args = <String>['info', _checkName(name), '--format', pluginInfoTemplate];
    final ProcessOutput output = await _query(args);
    final PluginInfo? info = parsePluginInfo(output.stdout);
    if (info == null) {
      throw VfoxException(
        'Unrecognised plugin information',
        arguments: args,
        stdout: output.stdout,
        stderr: output.stderr,
      );
    }
    return info;
  }

  /// The directory of an installed version, or `null` if it is not installed.
  Future<String?> installPath(String name, String version) async {
    final ProcessOutput output = await _query(<String>['info', '${_checkName(name)}@${_checkVersion(version)}']);
    return parseInstallPath(output.stdout);
  }

  /// Versions of [name] that can be installed, newest first.
  ///
  /// May take a while: plugins fetch the list from the network unless vfox
  /// has it cached (see the `cache.availableHookDuration` setting).
  Future<List<AvailableVersion>> searchVersions(String name) async {
    final ProcessOutput output = await _query(<String>['search', _checkName(name)]);
    return parseSearch(output.stdout);
  }

  /// Plugins listed in the vfox registry.
  Future<List<AvailablePlugin>> availablePlugins() async {
    final ProcessOutput output = await _query(const <String>['available']);
    return parseAvailablePlugins(output.stdout);
  }

  /// The vfox configuration as `key = value` entries.
  Future<List<ConfigEntry>> configEntries() async {
    final ProcessOutput output = await _query(const <String>['config', '--list']);
    return parseConfigList(output.stdout);
  }

  /// Sets the configuration [key] to [value].
  Future<void> setConfig(String key, String value) async {
    await _query(<String>['config', _checkConfigKey(key), value]);
  }

  /// Resets the configuration [key] to its default value.
  Future<void> unsetConfig(String key) async {
    await _query(<String>['config', '--unset', _checkConfigKey(key)]);
  }

  // ---------------------------------------------------------------------
  // Commands
  // ---------------------------------------------------------------------

  /// Installs [version] of [name].
  VfoxCommand install(String name, String version) => VfoxCommand(
    operation: VfoxOperation.install,
    arguments: <String>['install', '--yes', '${_checkName(name)}@${_checkVersion(version)}'],
    sdk: name,
    version: version,
  );

  /// Uninstalls [version] of [name].
  VfoxCommand uninstall(String name, String version) => VfoxCommand(
    operation: VfoxOperation.uninstall,
    arguments: <String>['uninstall', '${_checkName(name)}@${_checkVersion(version)}'],
    sdk: name,
    version: version,
  );

  /// Makes [version] of [name] the current version in [scope].
  ///
  /// [projectDirectory] is required for [UseScope.project].
  VfoxCommand use(String name, String version, {UseScope scope = UseScope.global, String? projectDirectory}) {
    return VfoxCommand(
      operation: VfoxOperation.use,
      arguments: <String>['use', _scopeFlag(scope), '${_checkName(name)}@${_checkVersion(version)}'],
      sdk: name,
      version: version,
      workingDirectory: _scopeDirectory(scope, projectDirectory),
      requiresHookEnvironment: true,
    );
  }

  /// Removes the current version of [name] from [scope].
  VfoxCommand unuse(String name, {UseScope scope = UseScope.global, String? projectDirectory}) {
    return VfoxCommand(
      operation: VfoxOperation.unuse,
      arguments: <String>['unuse', _scopeFlag(scope), _checkName(name)],
      sdk: name,
      workingDirectory: _scopeDirectory(scope, projectDirectory),
    );
  }

  /// Adds the plugin [name] from the registry, or from [source] (a URL or a
  /// local path) when given. [alias] installs the plugin under another name.
  VfoxCommand addPlugin(String name, {String? source, String? alias}) {
    final String? trimmedSource = _blankToNull(source);
    final String? trimmedAlias = _blankToNull(alias);
    final String? trimmedName = _blankToNull(name);
    if (trimmedName == null && trimmedSource == null) {
      throw ArgumentError.value(name, 'name', 'A plugin name or a source is required');
    }
    return VfoxCommand(
      operation: VfoxOperation.addPlugin,
      arguments: <String>[
        'add',
        if (trimmedSource != null) ...<String>['--source', trimmedSource],
        if (trimmedAlias != null) ...<String>['--alias', _checkName(trimmedAlias)],
        if (trimmedName != null) _checkName(trimmedName),
      ],
      sdk: trimmedAlias ?? trimmedName,
    );
  }

  /// Removes the plugin [name] together with all of its installed versions.
  VfoxCommand removePlugin(String name) => VfoxCommand(
    operation: VfoxOperation.removePlugin,
    arguments: <String>['remove', '--yes', _checkName(name)],
    sdk: name,
  );

  /// Updates the plugin [name] to its latest version.
  VfoxCommand updatePlugin(String name) =>
      VfoxCommand(operation: VfoxOperation.updatePlugin, arguments: <String>['update', _checkName(name)], sdk: name);

  /// Updates every installed plugin.
  VfoxCommand updateAllPlugins() =>
      const VfoxCommand(operation: VfoxOperation.updateAllPlugins, arguments: <String>['update', '--all']);

  /// Starts [command] and returns the running process.
  ///
  /// Throws a [VfoxNotFoundException] if vfox cannot be started.
  Future<RunningProcess> start(VfoxCommand command) async {
    try {
      return await _runner.start(
        launchConfig.executable,
        command.arguments,
        workingDirectory: command.workingDirectory ?? launchConfig.neutralDirectory,
        environment: launchConfig.environment(hookShell: command.requiresHookEnvironment),
      );
    } on ProcessException catch (error) {
      throw VfoxNotFoundException(launchConfig.executable, error.message);
    }
  }

  // ---------------------------------------------------------------------
  // Internals
  // ---------------------------------------------------------------------

  Future<ProcessOutput> _query(List<String> arguments, {bool allowFailure = false}) async {
    final ProcessOutput output;
    try {
      output = await _runner.run(
        launchConfig.executable,
        arguments,
        workingDirectory: launchConfig.neutralDirectory,
        environment: launchConfig.environment(),
      );
    } on ProcessException catch (error) {
      throw VfoxNotFoundException(launchConfig.executable, error.message);
    }
    if (output.exitCode != 0 && !allowFailure) {
      throw _failure(arguments, output);
    }
    return output;
  }

  VfoxException _failure(List<String> arguments, ProcessOutput output) {
    final String summary = summarizeError(output.stdout, output.stderr);
    return VfoxException(
      summary.isEmpty ? 'vfox exited with code ${output.exitCode}' : summary,
      arguments: arguments,
      exitCode: output.exitCode,
      stdout: output.stdout,
      stderr: output.stderr,
    );
  }

  static String _scopeFlag(UseScope scope) => switch (scope) {
    UseScope.global => '--global',
    UseScope.project => '--project',
  };

  static String? _scopeDirectory(UseScope scope, String? projectDirectory) {
    if (scope == UseScope.global) {
      return null;
    }
    final String? directory = _blankToNull(projectDirectory);
    if (directory == null) {
      throw ArgumentError.value(projectDirectory, 'projectDirectory', 'Required for the project scope');
    }
    return directory;
  }

  static final RegExp _namePattern = RegExp(r'^[A-Za-z0-9][A-Za-z0-9._+-]*$');
  static final RegExp _configKeyPattern = RegExp(r'^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*$');

  /// Rejects names that vfox could mistake for flags or that contain
  /// separators. Arguments never pass through a shell, so this only guards
  /// against option injection and typos.
  static String _checkName(String name) {
    if (!_namePattern.hasMatch(name)) {
      throw ArgumentError.value(name, 'name', 'Not a valid plugin name');
    }
    return name;
  }

  static String _checkVersion(String version) {
    if (version.isEmpty || version.startsWith('-') || version.contains(RegExp(r'[\s@]'))) {
      throw ArgumentError.value(version, 'version', 'Not a valid version');
    }
    return version;
  }

  static String _checkConfigKey(String key) {
    if (!_configKeyPattern.hasMatch(key)) {
      throw ArgumentError.value(key, 'key', 'Not a valid configuration key');
    }
    return key;
  }

  static String? _blankToNull(String? value) {
    final String? trimmed = value?.trim();
    return trimmed == null || trimmed.isEmpty ? null : trimmed;
  }
}
