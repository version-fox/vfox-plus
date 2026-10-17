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

// Test doubles shared by the unit and widget tests.

import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:material_ui/material_ui.dart';
import 'package:vfox_plus/src/core/process_runner.dart';
import 'package:vfox_plus/src/core/vfox/vfox_output.dart';
import 'package:vfox_plus/src/ui/platform_services.dart';

/// The captured vfox outputs in `shared/fixtures`.
const String fixturesVersion = '1.0.12';

/// Reads a capture of `shared/fixtures`, in which control characters are
/// written as `\xHH` and backslashes as `\\`.
String readCapture(String path) => File(path).readAsStringSync().replaceAllMapped(
  RegExp(r'\\(?:x([0-9a-f]{2})|\\)'),
  (Match match) => match[1] == null ? r'\' : String.fromCharCode(int.parse(match[1]!, radix: 16)),
);

/// Standard output of the fixture [name].
String fixtureStdout(String name) => readCapture('../shared/fixtures/vfox-$fixturesVersion/$name.stdout');

/// Standard error of the fixture [name].
String fixtureStderr(String name) => readCapture('../shared/fixtures/vfox-$fixturesVersion/$name.stderr');

/// The result of the fixture [name], with [exitCode].
ProcessOutput fixtureOutput(String name, {int exitCode = 0}) =>
    ProcessOutput(exitCode: exitCode, stdout: fixtureStdout(name), stderr: fixtureStderr(name));

/// One recorded invocation of a [FakeProcessRunner].
class ProcessCall {
  ProcessCall(this.executable, this.arguments, this.workingDirectory, this.environment, {required this.streamed});

  final String executable;
  final List<String> arguments;
  final String workingDirectory;
  final Map<String, String> environment;
  final bool streamed;

  @override
  String toString() => '$executable ${arguments.join(' ')} (in $workingDirectory)';
}

/// A scripted [ProcessRunner].
///
/// Responses are matched by the joined argument list, so tests read like the
/// command lines they expect. Unmatched invocations fail with exit code 127.
class FakeProcessRunner implements ProcessRunner {
  final Map<String, ProcessOutput Function()> _responses = <String, ProcessOutput Function()>{};
  final Map<String, FakeRunningProcess Function()> _streams = <String, FakeRunningProcess Function()>{};

  /// Every invocation, in order.
  final List<ProcessCall> invocations = <ProcessCall>[];

  /// When set, every invocation throws as if the executable did not exist.
  bool missingExecutable = false;

  /// Responds to `vfox <arguments>` with [output].
  void on(String arguments, ProcessOutput output) => _responses[arguments] = () => output;

  /// Responds to `vfox <arguments>` with the result of [respond].
  void onDynamic(String arguments, ProcessOutput Function() respond) => _responses[arguments] = respond;

  /// Responds to a started `vfox <arguments>` with the process from [create].
  void onStart(String arguments, FakeRunningProcess Function() create) => _streams[arguments] = create;

  /// ProcessCalls whose arguments start with [prefix].
  List<ProcessCall> calls(String prefix) =>
      invocations.where((ProcessCall call) => call.arguments.join(' ').startsWith(prefix)).toList();

  @override
  Future<ProcessOutput> run(
    String executable,
    List<String> arguments, {
    required String workingDirectory,
    required Map<String, String> environment,
  }) async {
    invocations.add(ProcessCall(executable, arguments, workingDirectory, environment, streamed: false));
    if (missingExecutable) {
      throw ProcessException(executable, arguments, 'No such file or directory', 2);
    }
    final ProcessOutput Function()? respond = _responses[arguments.join(' ')];
    return respond?.call() ?? const ProcessOutput(exitCode: 127, stderr: 'unexpected command');
  }

  @override
  Future<RunningProcess> start(
    String executable,
    List<String> arguments, {
    required String workingDirectory,
    required Map<String, String> environment,
  }) async {
    invocations.add(ProcessCall(executable, arguments, workingDirectory, environment, streamed: true));
    if (missingExecutable) {
      throw ProcessException(executable, arguments, 'No such file or directory', 2);
    }
    final FakeRunningProcess Function()? create = _streams[arguments.join(' ')];
    if (create != null) {
      return create();
    }
    final ProcessOutput Function()? respond = _responses[arguments.join(' ')];
    final ProcessOutput output = respond?.call() ?? const ProcessOutput(exitCode: 0);
    return FakeRunningProcess.completed(output);
  }
}

/// A [RunningProcess] whose output and exit are driven by the test.
class FakeRunningProcess implements RunningProcess {
  /// Creates a process that runs until [exit] is called.
  FakeRunningProcess();

  /// Creates a process that prints [output] and exits immediately.
  factory FakeRunningProcess.completed(ProcessOutput output) {
    final process = FakeRunningProcess();
    scheduleMicrotask(() {
      process
        ..writeStdout(output.stdout)
        ..writeStderr(output.stderr)
        ..exit(output.exitCode);
    });
    return process;
  }

  final StreamController<List<int>> _stdout = StreamController<List<int>>();
  final StreamController<List<int>> _stderr = StreamController<List<int>>();
  final Completer<int> _exitCode = Completer<int>();

  /// Whether [kill] was called.
  bool killed = false;

  /// Writes [text] to standard output.
  void writeStdout(String text) {
    if (text.isNotEmpty) {
      _stdout.add(utf8.encode(text));
    }
  }

  /// Writes [text] to standard error.
  void writeStderr(String text) {
    if (text.isNotEmpty) {
      _stderr.add(utf8.encode(text));
    }
  }

  /// Ends the process with [code].
  void exit(int code) {
    if (_exitCode.isCompleted) {
      return;
    }
    unawaited(_stdout.close());
    unawaited(_stderr.close());
    _exitCode.complete(code);
  }

  @override
  Stream<List<int>> get stdout => _stdout.stream;

  @override
  Stream<List<int>> get stderr => _stderr.stream;

  @override
  Future<int> get exitCode => _exitCode.future;

  @override
  bool kill() {
    killed = true;
    exit(-15);
    return true;
  }
}

/// [PlatformServices] that record what the application asked for.
class FakePlatformServices implements PlatformServices {
  FakePlatformServices({this.accent, this.directory, this.file});

  Color? accent;
  String? directory;
  String? file;
  final List<Uri> openedUrls = <Uri>[];
  final List<String> copiedTexts = <String>[];

  /// The extensions the last file chooser accepted.
  List<String>? pickedExtensions;

  @override
  Future<Color?> accentColor() async => accent;

  @override
  Future<void> copyText(String text) async => copiedTexts.add(text);

  @override
  Future<bool> openUrl(Uri url) async {
    openedUrls.add(url);
    return true;
  }

  @override
  Future<String?> pickDirectory({String? initialDirectory}) async => directory;

  @override
  Future<String?> pickFile({String? initialDirectory, List<String>? extensions, String? typeLabel}) async {
    pickedExtensions = extensions;
    return file;
  }
}

/// A runner that answers the read-only commands like the `list` and `current`
/// fixtures: maven 3.9.16 (current) and 3.8.9, mvn2 without versions and
/// nodejs 24.11.1 (current) and 22.21.0.
FakeProcessRunner fixtureRunner() {
  return FakeProcessRunner()
    ..on('--version', fixtureOutput('version'))
    ..on('list', fixtureOutput('list'))
    ..on('current', fixtureOutput('current'))
    ..on('available', fixtureOutput('available'))
    ..on('config --list', fixtureOutput('config-list'))
    ..on('search maven', fixtureOutput('search-maven-installed'))
    ..on('search nodejs', fixtureOutput('search-nodejs'))
    ..on('info nodejs --format $pluginInfoTemplate', fixtureOutput('info'));
}
