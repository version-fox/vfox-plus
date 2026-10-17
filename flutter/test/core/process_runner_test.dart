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

import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/process_runner.dart';

void main() {
  const runner = IoProcessRunner();
  final Map<String, String> environment = <String, String>{'PATH': Platform.environment['PATH'] ?? '/usr/bin:/bin'};

  test('run collects both streams and the exit code', () async {
    final ProcessOutput output = await runner.run(
      'sh',
      <String>['-c', r'echo "$GREETING from $(pwd)"; echo oops >&2; exit 3'],
      workingDirectory: '/',
      environment: <String, String>{...environment, 'GREETING': '你好'},
    );
    expect(output.exitCode, 3);
    expect(output.stdout, '你好 from /\n');
    expect(output.stderr, 'oops\n');
  });

  test('the parent environment is not inherited', () async {
    final ProcessOutput output = await runner.run(
      'sh',
      <String>['-c', r'echo "[$HOME]"'],
      workingDirectory: '/',
      environment: environment,
    );
    expect(output.stdout.trim(), '[]');
  });

  test('start streams output and closes stdin', () async {
    final RunningProcess process = await runner.start(
      'sh',
      <String>['-c', 'cat; echo done'],
      workingDirectory: '/',
      environment: environment,
    );
    final String stdout = await process.stdout.transform(utf8.decoder).join();
    expect(await process.exitCode, 0);
    expect(stdout, 'done\n');
  });

  test('kill terminates a running process', () async {
    final RunningProcess process = await runner.start(
      'sleep',
      <String>['30'],
      workingDirectory: '/',
      environment: environment,
    );
    expect(process.kill(), isTrue);
    expect(await process.exitCode, isNot(0));
  });

  test('a missing executable throws a ProcessException', () {
    expect(
      runner.run('/nonexistent/vfox', const <String>[], workingDirectory: '/', environment: environment),
      throwsA(isA<ProcessException>()),
    );
  });
}
