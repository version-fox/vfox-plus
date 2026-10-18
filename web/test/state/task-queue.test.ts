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

import { afterEach, beforeEach, expect, it } from 'vitest';

import { launchConfig, VfoxCli } from '../../src/core/vfox/cli';
import { TaskQueue, type VfoxTask } from '../../src/state/task-queue';
import { FakeProcessRunner, FakeRunningProcess, fixtureOutput, settle } from '../support/fakes';

let runner: FakeProcessRunner;
let cli: VfoxCli;
let queue: TaskQueue;
let finished: VfoxTask[];

beforeEach(() => {
  runner = new FakeProcessRunner();
  cli = new VfoxCli(runner, launchConfig({ executable: 'vfox' }));
  queue = new TaskQueue(cli, 3, 2);
  finished = [];
  queue.onFinished((task) => finished.push(task));
});

afterEach(() => {
  queue.dispose();
});

it('runs one command at a time, in order', async () => {
  const first = new FakeRunningProcess();
  const second = new FakeRunningProcess();
  runner.onStart('install --yes nodejs@24', () => first).onStart('install --yes java@21', () => second);
  const a = queue.enqueue(cli.install('nodejs', '24'));
  const b = queue.enqueue(cli.install('java', '21'));
  await settle();
  expect(a.status).toBe('running');
  expect(b.status).toBe('queued');
  expect(queue.activeCount).toBe(2);
  expect(runner.invocations).toHaveLength(1);
  first.exit(0);
  await settle();
  expect(a.status).toBe('succeeded');
  expect(b.status).toBe('running');
  second.exit(0);
  await settle();
  expect(finished).toEqual([a, b]);
  expect(queue.tasks).toEqual([b, a]);
  expect(queue.activeCount).toBe(0);
  expect(a.startedAt).not.toBeNull();
  expect(a.finishedAt).not.toBeNull();
});

it('does not queue an identical active command twice', () => {
  runner.onStart('update --all', () => new FakeRunningProcess());
  const a = queue.enqueue(cli.updateAllPlugins());
  expect(queue.enqueue(cli.updateAllPlugins())).toBe(a);
  expect(queue.tasks).toHaveLength(1);
});

it('turns progress bars into progress and keeps a bounded log', async () => {
  const process = new FakeRunningProcess();
  runner.onStart('install --yes maven@3.9.16', () => process);
  const task = queue.enqueue(cli.install('maven', '3.9.16'));
  await settle();
  process.writeStdout('Preinstalling maven@3.9.16...\n');
  process.writeStderr('\rDownloading...  42% [===>  ] (1 MB/s)\r');
  await settle();
  expect(task.progress).toBe(0.42);
  expect(task.log).toEqual(['Preinstalling maven@3.9.16...']);
  process.writeStdout('one\ntwo\nthree');
  process.exit(0);
  await settle();
  expect(task.log).toEqual(['one', 'two', 'three']);
  expect(task.lastLine).toBe('three');
  expect(task.droppedLines).toBe(1);
  expect(task.status).toBe('succeeded');
  expect(task.exitCode).toBe(0);
});

it('summarises failures with the vfox error message', async () => {
  runner.on('install --yes maven@3.9.16', fixtureOutput('install-failure', 1));
  const task = queue.enqueue(cli.install('maven', '3.9.16'));
  await settle();
  expect(task.status).toBe('failed');
  // vfox 1.0.13 prints the reason on standard output and the generic
  // "failed to install maven" on standard error.
  expect(task.errorMessage).toBe(
    'mark installation in progress: open /tmp/vfox-sb/home/.vfox/cache/maven/.v-3.9.16.installing: permission denied',
  );
  expect(finished).toEqual([task]);
});

it('reports the exit code of a failure without output', async () => {
  runner.on('update java', { exitCode: 4, stdout: '', stderr: '' });
  const task = queue.enqueue(cli.updatePlugin('java'));
  await settle();
  expect(task.errorMessage).toBe('vfox exited with code 4');
});

it('reports a missing executable as a failure', async () => {
  runner.missingExecutable = true;
  const task = queue.enqueue(cli.updatePlugin('java'));
  await settle();
  expect(task.status).toBe('failed');
  expect(task.errorMessage).not.toBe('');
});

it('cancels queued tasks without running them', async () => {
  const process = new FakeRunningProcess();
  runner.onStart('update --all', () => process);
  queue.enqueue(cli.updateAllPlugins());
  const queued = queue.enqueue(cli.updatePlugin('java'));
  queue.cancel(queued);
  process.exit(0);
  await settle();
  expect(queued.status).toBe('cancelled');
  expect(runner.calls('update java')).toEqual([]);
});

it('kills cancelled running tasks', async () => {
  const process = new FakeRunningProcess();
  runner.onStart('install --yes go@1', () => process);
  const task = queue.enqueue(cli.install('go', '1'));
  await settle();
  queue.cancel(task);
  await settle();
  expect(process.killed).toBe(true);
  expect(task.status).toBe('cancelled');
  expect(finished).toEqual([task]);
  // A finished task is not cancelled again.
  queue.cancel(task);
  expect(finished).toEqual([task]);
});

it('retries a failed task and clears finished ones', async () => {
  runner.on('update java', { exitCode: 1, stdout: 'java plugin not installed\n', stderr: '' });
  const failed = queue.enqueue(cli.updatePlugin('java'));
  await settle();
  expect(failed.status).toBe('failed');
  runner.on('update java', fixtureOutput('update'));
  const retried = queue.retry(failed);
  await settle();
  expect(retried).not.toBe(failed);
  expect(retried.status).toBe('succeeded');
  expect(queue.hasFinished).toBe(true);
  queue.clearFinished();
  expect(queue.tasks).toEqual([]);
});

it('forgets the oldest finished tasks', async () => {
  for (const name of ['a', 'b', 'c']) {
    queue.enqueue(cli.updatePlugin(name));
    await settle();
  }
  queue.enqueue(cli.updatePlugin('d'));
  await settle();
  expect(queue.tasks.map((task) => task.command.sdk)).toEqual(['d', 'c', 'b']);
});

it('finds the active task of an SDK version', () => {
  runner.onStart('install --yes go@1', () => new FakeRunningProcess());
  const task = queue.enqueue(cli.install('go', '1'));
  expect(queue.activeTask({ operation: 'install', sdk: 'go', version: '1' })).toBe(task);
  expect(queue.activeTask({ operation: 'uninstall', sdk: 'go' })).toBeNull();
});
