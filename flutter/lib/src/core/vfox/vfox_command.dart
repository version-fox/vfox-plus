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

/// Descriptions of vfox invocations that change state, and vfox errors.
library;

/// The kinds of state-changing operations the application performs.
enum VfoxOperation {
  /// `vfox install --yes <name>@<version>`
  install,

  /// `vfox uninstall <name>@<version>`
  uninstall,

  /// `vfox use --global|--project <name>@<version>`
  use,

  /// `vfox unuse --global|--project <name>`
  unuse,

  /// `vfox add [--source <url>] [--alias <alias>] <name>`
  addPlugin,

  /// `vfox remove --yes <name>`
  removePlugin,

  /// `vfox update <name>`
  updatePlugin,

  /// `vfox update --all`
  updateAllPlugins,
}

/// A state-changing vfox invocation.
///
/// Commands are values: they are built by `VfoxCli`, queued and executed by
/// the task queue, and can be retried by running them again.
final class VfoxCommand {
  /// Creates a command. Prefer the factories on `VfoxCli`.
  const VfoxCommand({
    required this.operation,
    required this.arguments,
    this.sdk,
    this.version,
    this.workingDirectory,
    this.requiresHookEnvironment = false,
  });

  /// What the command does.
  final VfoxOperation operation;

  /// Arguments passed to the vfox executable.
  final List<String> arguments;

  /// The plugin the command acts on, if any.
  final String? sdk;

  /// The SDK version the command acts on, if any.
  final String? version;

  /// Directory to run in. `null` means a neutral directory, so that no
  /// project configuration is picked up.
  final String? workingDirectory;

  /// Whether vfox must believe that it runs inside an activated shell.
  ///
  /// `vfox use` refuses to run on Unix without shell integration, but with
  /// `--global` or `--project` it only edits `.vfox.toml` and symlinks,
  /// which is exactly what a GUI needs. See `docs/vfox-cli-contract.md`.
  final bool requiresHookEnvironment;

  /// The SDK and version as `name@version`, or just the name.
  String? get subject => switch ((sdk, version)) {
    (final String sdk, final String version) => '$sdk@$version',
    (final String sdk, null) => sdk,
    _ => null,
  };

  @override
  String toString() => 'vfox ${arguments.join(' ')}';
}

/// A vfox invocation failed.
class VfoxException implements Exception {
  /// Creates an exception for a failed vfox invocation.
  const VfoxException(
    this.message, {
    this.arguments = const <String>[],
    this.exitCode,
    this.stdout = '',
    this.stderr = '',
  });

  /// A short, human readable description of the failure.
  final String message;

  /// The arguments vfox was started with.
  final List<String> arguments;

  /// The exit code, if the process ran at all.
  final int? exitCode;

  /// Standard output of the failed process.
  final String stdout;

  /// Standard error of the failed process.
  final String stderr;

  @override
  String toString() => 'VfoxException: $message (vfox ${arguments.join(' ')}, exit code $exitCode)';
}

/// The vfox executable could not be started.
final class VfoxNotFoundException extends VfoxException {
  /// Creates the exception for the missing [executable].
  const VfoxNotFoundException(this.executable, String message) : super(message);

  /// The path or name that could not be executed.
  final String executable;

  @override
  String toString() => 'VfoxNotFoundException: $executable: $message';
}
