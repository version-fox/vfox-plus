# Copyright 2026 Han Li and contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""The task queue, against a scripted process runner."""

from collections.abc import AsyncIterator

import pytest

from support import FakeProcessRunner, FakeRunningProcess, fixture_output, settle
from vfox_plus.core.process import ProcessOutput
from vfox_plus.core.vfox.cli import VfoxCli, VfoxLaunchConfig
from vfox_plus.core.vfox.command import VfoxOperation
from vfox_plus.state.task_queue import TaskQueue, TaskStatus, VfoxTask


@pytest.fixture
def runner() -> FakeProcessRunner:
    return FakeProcessRunner()


@pytest.fixture
def cli(runner: FakeProcessRunner) -> VfoxCli:
    return VfoxCli(runner, VfoxLaunchConfig(executable="vfox"))


@pytest.fixture
async def queue(cli: VfoxCli) -> AsyncIterator[TaskQueue]:
    queue = TaskQueue(cli, max_log_lines=3, max_finished_tasks=2)
    yield queue
    queue.dispose()


@pytest.fixture
def finished(queue: TaskQueue) -> list[VfoxTask]:
    tasks: list[VfoxTask] = []
    queue.on_finished(tasks.append)
    return tasks


async def test_runs_one_command_at_a_time_in_order(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue, finished: list[VfoxTask]
) -> None:
    first = FakeRunningProcess()
    second = FakeRunningProcess()
    runner.on_start("install --yes nodejs@24", lambda: first).on_start("install --yes java@21", lambda: second)
    a = queue.enqueue(cli.install("nodejs", "24"))
    b = queue.enqueue(cli.install("java", "21"))
    await settle()
    assert a.status == TaskStatus.RUNNING
    assert b.status == TaskStatus.QUEUED
    assert queue.active_count == 2
    assert len(runner.invocations) == 1
    first.exit(0)
    await settle()
    assert a.status == TaskStatus.SUCCEEDED
    assert b.status == TaskStatus.RUNNING
    second.exit(0)
    await settle()
    assert finished == [a, b]
    assert queue.tasks == [b, a]
    assert queue.active_count == 0
    assert a.started_at is not None
    assert a.finished_at is not None


async def test_does_not_queue_an_identical_active_command_twice(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    runner.on_start("update --all", FakeRunningProcess)
    a = queue.enqueue(cli.update_all_plugins())
    assert queue.enqueue(cli.update_all_plugins()) is a
    assert len(queue.tasks) == 1


async def test_turns_progress_bars_into_progress_and_keeps_a_bounded_log(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    process = FakeRunningProcess()
    runner.on_start("install --yes maven@3.9.16", lambda: process)
    task = queue.enqueue(cli.install("maven", "3.9.16"))
    notified: list[TaskStatus] = []
    task.add_listener(lambda: notified.append(task.status))
    await settle()
    process.write_stdout("Preinstalling maven@3.9.16...\n")
    process.write_stderr("\rDownloading...  42% [===>  ] (1 MB/s)\r")
    await settle()
    assert task.progress == 0.42
    assert task.log == ["Preinstalling maven@3.9.16..."]
    process.write_stdout("one\ntwo\nthree")
    process.exit(0)
    await settle()
    assert task.log == ["one", "two", "three"]
    assert task.last_line == "three"
    assert task.dropped_lines == 1
    assert task.status == TaskStatus.SUCCEEDED
    assert task.exit_code == 0
    assert notified


async def test_summarises_failures_with_the_vfox_error_message(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue, finished: list[VfoxTask]
) -> None:
    runner.on("install --yes maven@3.9.16", fixture_output("install-failure", 1))
    task = queue.enqueue(cli.install("maven", "3.9.16"))
    await settle()
    assert task.status == TaskStatus.FAILED
    # vfox 1.0.13 prints the reason on standard output and the generic
    # "failed to install maven" on standard error.
    assert task.error_message == (
        "mark installation in progress: open /tmp/vfox-sb/home/.vfox/cache/maven/.v-3.9.16.installing: "
        "permission denied"
    )
    assert finished == [task]


async def test_reports_the_exit_code_of_a_failure_without_output(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    runner.on("update java", ProcessOutput(4, "", ""))
    task = queue.enqueue(cli.update_plugin("java"))
    await settle()
    assert task.error_message == "vfox exited with code 4"


async def test_reports_a_missing_executable_as_a_failure(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    runner.missing_executable = True
    task = queue.enqueue(cli.update_plugin("java"))
    await settle()
    assert task.status == TaskStatus.FAILED
    assert task.error_message


async def test_reports_unexpected_errors_as_a_failure(
    cli: VfoxCli, queue: TaskQueue, runner: FakeProcessRunner
) -> None:
    def explode() -> FakeRunningProcess:
        raise RuntimeError("no more processes")

    runner.on_start("update java", explode)
    task = queue.enqueue(cli.update_plugin("java"))
    await settle()
    assert task.status == TaskStatus.FAILED
    assert task.error_message == "no more processes"


async def test_cancels_queued_tasks_without_running_them(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    process = FakeRunningProcess()
    runner.on_start("update --all", lambda: process)
    queue.enqueue(cli.update_all_plugins())
    queued = queue.enqueue(cli.update_plugin("java"))
    queue.cancel(queued)
    process.exit(0)
    await settle()
    assert queued.status == TaskStatus.CANCELLED
    assert runner.calls("update java") == []


async def test_kills_cancelled_running_tasks(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue, finished: list[VfoxTask]
) -> None:
    process = FakeRunningProcess()
    runner.on_start("install --yes go@1", lambda: process)
    task = queue.enqueue(cli.install("go", "1"))
    await settle()
    queue.cancel(task)
    await settle()
    assert process.killed
    assert task.status == TaskStatus.CANCELLED
    assert finished == [task]
    # A finished task is not cancelled again.
    queue.cancel(task)
    assert finished == [task]


async def test_kills_a_task_cancelled_while_its_process_starts(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    process = FakeRunningProcess()
    runner.on_start("install --yes go@1", lambda: process)
    task = queue.enqueue(cli.install("go", "1"))
    queue.cancel(task)
    await settle()
    assert process.killed
    assert task.status == TaskStatus.CANCELLED


async def test_retries_a_failed_task_and_clears_finished_ones(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    runner.on("update java", ProcessOutput(1, "java plugin not installed\n", ""))
    failed = queue.enqueue(cli.update_plugin("java"))
    await settle()
    assert failed.status == TaskStatus.FAILED
    runner.on("update java", fixture_output("update"))
    retried = queue.retry(failed)
    await settle()
    assert retried is not failed
    assert retried.status == TaskStatus.SUCCEEDED
    assert queue.has_finished
    queue.clear_finished()
    assert queue.tasks == []
    assert not queue.has_finished


async def test_forgets_the_oldest_finished_tasks(cli: VfoxCli, queue: TaskQueue) -> None:
    for name in ("a", "b", "c"):
        queue.enqueue(cli.update_plugin(name))
        await settle()
    queue.enqueue(cli.update_plugin("d"))
    await settle()
    assert [task.command.sdk for task in queue.tasks] == ["d", "c", "b"]


async def test_finds_the_active_task_of_an_sdk_version(
    runner: FakeProcessRunner, cli: VfoxCli, queue: TaskQueue
) -> None:
    runner.on_start("install --yes go@1", FakeRunningProcess)
    task = queue.enqueue(cli.install("go", "1"))
    assert queue.active_task(VfoxOperation.INSTALL, "go", "1") is task
    assert queue.active_task(VfoxOperation.UNINSTALL, "go") is None
    assert queue.active_task() is task


async def test_disposing_kills_running_processes(runner: FakeProcessRunner, cli: VfoxCli) -> None:
    process = FakeRunningProcess()
    runner.on_start("install --yes go@1", lambda: process)
    queue = TaskQueue(cli)
    queue.enqueue(cli.install("go", "1"))
    await settle()
    queue.dispose()
    assert process.killed
    await settle()
