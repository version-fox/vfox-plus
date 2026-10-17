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

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/process_runner.dart';
import 'package:vfox_plus/src/core/vfox/vfox_cli.dart';
import 'package:vfox_plus/src/core/vfox/vfox_command.dart';
import 'package:vfox_plus/src/state/task_queue.dart';

import '../support/fakes.dart';

void main() {
  late FakeProcessRunner runner;
  late VfoxCli cli;
  late TaskQueue queue;
  late List<VfoxTask> finished;

  setUp(() {
    runner = FakeProcessRunner();
    cli = VfoxCli(
      runner: runner,
      launchConfig: const VfoxLaunchConfig(executable: 'vfox'),
    );
    queue = TaskQueue(cli: cli, maxLogLines: 3);
    finished = <VfoxTask>[];
    queue.onFinished.listen(finished.add);
  });

  tearDown(() => queue.dispose());

  Future<void> settle() => pumpEventQueue();

  test('runs one command at a time, in order', () async {
    final first = FakeRunningProcess();
    final second = FakeRunningProcess();
    runner
      ..onStart('install --yes nodejs@24', () => first)
      ..onStart('install --yes java@21', () => second);

    final VfoxTask a = queue.enqueue(cli.install('nodejs', '24'));
    final VfoxTask b = queue.enqueue(cli.install('java', '21'));
    await settle();
    expect(a.status, TaskStatus.running);
    expect(b.status, TaskStatus.queued);
    expect(queue.activeCount, 2);
    expect(runner.invocations, hasLength(1));

    first.exit(0);
    await settle();
    expect(a.status, TaskStatus.succeeded);
    expect(b.status, TaskStatus.running);

    second.exit(0);
    await settle();
    expect(finished, <VfoxTask>[a, b]);
    expect(queue.tasks, <VfoxTask>[b, a]);
    expect(queue.activeCount, 0);
  });

  test('does not queue an identical active command twice', () async {
    runner.onStart('update --all', FakeRunningProcess.new);
    final VfoxTask a = queue.enqueue(cli.updateAllPlugins());
    expect(queue.enqueue(cli.updateAllPlugins()), same(a));
    expect(queue.tasks, hasLength(1));
  });

  test('turns progress bars into progress and keeps a bounded log', () async {
    final process = FakeRunningProcess();
    runner.onStart('install --yes maven@3.9.16', () => process);
    final VfoxTask task = queue.enqueue(cli.install('maven', '3.9.16'));
    await settle();

    process
      ..writeStdout('Preinstalling maven@3.9.16...\n')
      ..writeStderr('\rDownloading...  42% [===>  ] (1 MB/s)\r');
    await settle();
    expect(task.progress, 0.42);
    expect(task.log, <String>['Preinstalling maven@3.9.16...']);

    process
      ..writeStdout('one\ntwo\nthree\n')
      ..exit(0);
    await settle();
    expect(task.log, <String>['one', 'two', 'three']);
    expect(task.droppedLines, 1);
    expect(task.status, TaskStatus.succeeded);
    expect(task.exitCode, 0);
  });

  test('summarises failures with the vfox error message', () async {
    runner.on('install --yes maven@3.9.16', fixtureOutput('install-failure', exitCode: 1));
    final VfoxTask task = queue.enqueue(cli.install('maven', '3.9.16'));
    await settle();
    expect(task.status, TaskStatus.failed);
    expect(
      task.errorMessage,
      'failed to create directory, err:mkdir /tmp/vfox-sb/home/.vfox/cache/maven/v-3.9.16: permission denied',
    );
    expect(finished, <VfoxTask>[task]);
  });

  test('reports a missing executable as a failure', () async {
    runner.missingExecutable = true;
    final VfoxTask task = queue.enqueue(cli.updatePlugin('java'));
    await settle();
    expect(task.status, TaskStatus.failed);
    expect(task.errorMessage, isNotEmpty);
  });

  test('cancels queued tasks without running them', () async {
    final process = FakeRunningProcess();
    runner.onStart('update --all', () => process);
    queue.enqueue(cli.updateAllPlugins());
    final VfoxTask queued = queue.enqueue(cli.updatePlugin('java'));
    queue.cancel(queued);
    process.exit(0);
    await settle();
    expect(queued.status, TaskStatus.cancelled);
    expect(runner.calls('update java'), isEmpty);
  });

  test('kills cancelled running tasks', () async {
    final process = FakeRunningProcess();
    runner.onStart('install --yes go@1', () => process);
    final VfoxTask task = queue.enqueue(cli.install('go', '1'));
    await settle();
    queue.cancel(task);
    await settle();
    expect(process.killed, isTrue);
    expect(task.status, TaskStatus.cancelled);
    expect(finished, <VfoxTask>[task]);
  });

  test('retries a failed task and clears finished ones', () async {
    runner.on('update java', const ProcessOutput(exitCode: 1, stdout: 'java plugin not installed\n'));
    final VfoxTask failed = queue.enqueue(cli.updatePlugin('java'));
    await settle();
    expect(failed.status, TaskStatus.failed);

    runner.on('update java', fixtureOutput('update'));
    final VfoxTask retried = queue.retry(failed);
    await settle();
    expect(retried, isNot(same(failed)));
    expect(retried.status, TaskStatus.succeeded);
    expect(queue.hasFinished, isTrue);

    queue.clearFinished();
    expect(queue.tasks, isEmpty);
  });

  test('finds the active task of an SDK version', () async {
    runner.onStart('install --yes go@1', FakeRunningProcess.new);
    final VfoxTask task = queue.enqueue(cli.install('go', '1'));
    expect(queue.activeTask(operation: VfoxOperation.install, sdk: 'go', version: '1'), same(task));
    expect(queue.activeTask(operation: VfoxOperation.uninstall, sdk: 'go'), isNull);
  });
}
