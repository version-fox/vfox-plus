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

/// A thin, testable abstraction over spawning child processes.
library;

import 'dart:async';
import 'dart:convert';
import 'dart:io';

/// Everything a finished process wrote, and how it exited.
final class ProcessOutput {
  /// Creates the result of a finished process.
  const ProcessOutput({required this.exitCode, this.stdout = '', this.stderr = ''});

  /// The exit code. Zero means success.
  final int exitCode;

  /// Everything written to standard output, decoded as UTF-8.
  final String stdout;

  /// Everything written to standard error, decoded as UTF-8.
  final String stderr;
}

/// A child process whose output is consumed while it runs.
abstract interface class RunningProcess {
  /// Raw standard output.
  Stream<List<int>> get stdout;

  /// Raw standard error.
  Stream<List<int>> get stderr;

  /// Completes with the exit code when the process ends.
  Future<int> get exitCode;

  /// Asks the process to terminate. Returns whether the signal was delivered.
  bool kill();
}

/// Spawns child processes.
///
/// The application never touches `dart:io` processes directly, which keeps
/// the vfox client testable with a fake runner and leaves room for a remote
/// implementation (see `docs/architecture.md`).
abstract interface class ProcessRunner {
  /// Runs [executable] to completion and collects its output.
  ///
  /// Throws a [ProcessException] if the executable cannot be started.
  Future<ProcessOutput> run(
    String executable,
    List<String> arguments, {
    required String workingDirectory,
    required Map<String, String> environment,
  });

  /// Starts [executable] and returns immediately.
  ///
  /// Throws a [ProcessException] if the executable cannot be started.
  Future<RunningProcess> start(
    String executable,
    List<String> arguments, {
    required String workingDirectory,
    required Map<String, String> environment,
  });
}

/// The [ProcessRunner] backed by `dart:io`.
///
/// The `environment` passed to [run] and [start] is the complete environment
/// of the child: the parent environment is not inherited implicitly, so that
/// callers can remove variables as well as add them.
final class IoProcessRunner implements ProcessRunner {
  /// Creates a process runner.
  const IoProcessRunner();

  @override
  Future<ProcessOutput> run(
    String executable,
    List<String> arguments, {
    required String workingDirectory,
    required Map<String, String> environment,
  }) async {
    final ProcessResult result = await Process.run(
      executable,
      arguments,
      workingDirectory: workingDirectory,
      environment: environment,
      includeParentEnvironment: false,
      stdoutEncoding: const Utf8Codec(allowMalformed: true),
      stderrEncoding: const Utf8Codec(allowMalformed: true),
    );
    return ProcessOutput(exitCode: result.exitCode, stdout: result.stdout as String, stderr: result.stderr as String);
  }

  @override
  Future<RunningProcess> start(
    String executable,
    List<String> arguments, {
    required String workingDirectory,
    required Map<String, String> environment,
  }) async {
    final Process process = await Process.start(
      executable,
      arguments,
      workingDirectory: workingDirectory,
      environment: environment,
      includeParentEnvironment: false,
    );
    // Nothing is ever written to the child, and an open stdin would make an
    // unexpected interactive prompt wait forever.
    unawaited(process.stdin.close());
    return _IoRunningProcess(process);
  }
}

final class _IoRunningProcess implements RunningProcess {
  _IoRunningProcess(this._process);

  final Process _process;

  @override
  Stream<List<int>> get stdout => _process.stdout;

  @override
  Stream<List<int>> get stderr => _process.stderr;

  @override
  Future<int> get exitCode => _process.exitCode;

  @override
  bool kill() => _process.kill();
}
