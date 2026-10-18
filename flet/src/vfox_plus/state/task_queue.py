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

"""The queue that runs state-changing vfox commands one after another."""

import time
from collections import deque
from collections.abc import Callable
from enum import StrEnum

from vfox_plus.core.process import RunningProcess
from vfox_plus.core.terminal_text import TerminalLineSplitter
from vfox_plus.core.vfox.cli import VfoxCli
from vfox_plus.core.vfox.command import VfoxCommand, VfoxError, VfoxOperation
from vfox_plus.core.vfox.output import parse_download_progress, summarize_error
from vfox_plus.state.notifier import ChangeNotifier, Emitter, Unsubscribe, spawn


class TaskStatus(StrEnum):
    """The life cycle of a :class:`VfoxTask`."""

    QUEUED = "queued"
    """Waiting for earlier tasks to finish."""
    RUNNING = "running"
    """The vfox process is running."""
    SUCCEEDED = "succeeded"
    """vfox exited with code zero."""
    FAILED = "failed"
    """vfox could not be started or exited with a non-zero code."""
    CANCELLED = "cancelled"
    """The user cancelled the task."""

    @property
    def is_active(self) -> bool:
        """Whether a task with this status has not finished yet."""
        return self in (TaskStatus.QUEUED, TaskStatus.RUNNING)


class VfoxTask(ChangeNotifier):
    """One execution of a state-changing :class:`VfoxCommand`. Created by
    :class:`TaskQueue`.
    """

    def __init__(self, task_id: int, command: VfoxCommand, created_at: float, max_log_lines: int) -> None:
        super().__init__()
        self.id = task_id
        """Sequential identifier, unique within a :class:`TaskQueue`."""
        self.command = command
        """The command that is executed."""
        self.created_at = created_at
        """When the task was queued, in seconds since the epoch."""
        self._lines: deque[str] = deque(maxlen=max_log_lines)
        self._stdout_lines: list[str] = []
        self._stderr_lines: list[str] = []
        self._dropped = 0
        self.process: RunningProcess | None = None
        self.status = TaskStatus.QUEUED
        self.progress: float | None = None
        """Download progress between 0 and 1, if vfox reported any."""
        self.started_at: float | None = None
        """When the process started."""
        self.finished_at: float | None = None
        """When the task finished."""
        self.exit_code: int | None = None
        """The exit code of vfox, if it ran to completion."""
        self.error_message: str | None = None
        """A short description of the failure, if the task failed."""

    @property
    def log(self) -> list[str]:
        """Output lines of the process (standard output and error
        interleaved), without progress bar redraws. Only the most recent lines
        are kept.
        """
        return list(self._lines)

    @property
    def dropped_lines(self) -> int:
        """How many old lines were dropped from :attr:`log`."""
        return self._dropped

    @property
    def last_line(self) -> str | None:
        """The text that summarises a successful run, usually its last line."""
        return self._lines[-1] if self._lines else None

    def append(self, line: str, is_error: bool) -> None:
        progress = parse_download_progress(line)
        if progress is not None:
            self.progress = progress
        else:
            (self._stderr_lines if is_error else self._stdout_lines).append(line)
            if len(self._lines) == self._lines.maxlen:
                self._dropped += 1
            self._lines.append(line)
        self.notify_listeners()

    def summary(self) -> str:
        """The error summary of the collected output."""
        return summarize_error("\n".join(self._stdout_lines), "\n".join(self._stderr_lines))

    def set_status(self, status: TaskStatus) -> None:
        self.status = status
        self.notify_listeners()


class _TaskOutput:
    """Feeds the output of a process into a task, line by line."""

    def __init__(self, task: VfoxTask) -> None:
        self._task = task
        self._stdout = TerminalLineSplitter()
        self._stderr = TerminalLineSplitter()

    def stdout(self, chunk: bytes) -> None:
        for line in self._stdout.push(chunk):
            self._task.append(line, False)

    def stderr(self, chunk: bytes) -> None:
        for line in self._stderr.push(chunk):
            self._task.append(line, True)

    def close(self) -> None:
        for line in self._stdout.close():
            self._task.append(line, False)
        for line in self._stderr.close():
            self._task.append(line, True)


class TaskQueue(ChangeNotifier):
    """Runs state-changing vfox commands one after another.

    vfox does not lock its data directory, so concurrent installs, uses and
    removals could corrupt `.vfox.toml` or the symlinks. The queue therefore
    runs at most one command at a time, in the order they were enqueued.
    """

    def __init__(self, cli: VfoxCli, max_log_lines: int = 2000, max_finished_tasks: int = 50) -> None:
        super().__init__()
        self._cli = cli
        self.max_log_lines = max_log_lines
        """How many log lines each task keeps."""
        self.max_finished_tasks = max_finished_tasks
        """How many finished tasks are kept before the oldest are forgotten."""
        self._all: list[VfoxTask] = []
        self._finished: Emitter[VfoxTask] = Emitter()
        self._next_id = 1
        self._running: VfoxTask | None = None

    @property
    def tasks(self) -> list[VfoxTask]:
        """All known tasks, newest first."""
        return list(reversed(self._all))

    @property
    def active_count(self) -> int:
        """The number of queued and running tasks."""
        return sum(1 for task in self._all if task.status.is_active)

    @property
    def has_finished(self) -> bool:
        """Whether any finished task can be cleared."""
        return any(not task.status.is_active for task in self._all)

    def on_finished(self, listener: Callable[[VfoxTask], None]) -> Unsubscribe:
        """Calls `listener` with every task once it has finished, failed or
        been cancelled.
        """
        return self._finished.listen(listener)

    def enqueue(self, command: VfoxCommand) -> VfoxTask:
        """Queues `command` and returns its task.

        If an identical command is already queued or running, its task is
        returned instead of queueing the command twice.
        """
        for task in self._all:
            if task.status.is_active and _same_command(task.command, command):
                return task
        task = VfoxTask(self._next_id, command, time.time(), self.max_log_lines)
        self._next_id += 1
        self._all.append(task)
        self._trim_finished()
        self.notify_listeners()
        self._pump()
        return task

    def retry(self, task: VfoxTask) -> VfoxTask:
        """Queues the command of `task` again."""
        return self.enqueue(task.command)

    def cancel(self, task: VfoxTask) -> None:
        """Cancels `task`: a queued task is dropped, a running process is
        killed.
        """
        match task.status:
            case TaskStatus.QUEUED:
                self._finish(task, TaskStatus.CANCELLED)
            case TaskStatus.RUNNING:
                if task.process is not None:
                    task.process.kill()
                # The status is settled in _execute() once the process has
                # exited.
                task.error_message = None
                task.set_status(TaskStatus.CANCELLED)
            case _:
                pass

    def clear_finished(self) -> None:
        """Forgets every finished task."""
        self._all = [task for task in self._all if task.status.is_active]
        self.notify_listeners()

    def active_task(
        self, operation: VfoxOperation | None = None, sdk: str | None = None, version: str | None = None
    ) -> VfoxTask | None:
        """The active task matching the given fields, if any."""
        return next(
            (
                task
                for task in self._all
                if task.status.is_active
                and (operation is None or task.command.operation == operation)
                and (sdk is None or task.command.sdk == sdk)
                and (version is None or task.command.version == version)
            ),
            None,
        )

    def _pump(self) -> None:
        if self._running is not None or self.is_disposed:
            return
        upcoming = next((task for task in self._all if task.status == TaskStatus.QUEUED), None)
        if upcoming is None:
            return
        self._running = upcoming
        # The task is running from now on, so that cancelling it before its
        # process has started kills the process once it has.
        upcoming.started_at = time.time()
        upcoming.set_status(TaskStatus.RUNNING)
        self.notify_listeners()
        spawn(self._execute(upcoming))

    async def _execute(self, task: VfoxTask) -> None:
        output = _TaskOutput(task)
        try:
            process = await self._cli.start(task.command, output)
            task.process = process
            if task.status == TaskStatus.CANCELLED:
                # Cancelled while the process was being spawned.
                process.kill()
            exit_code = await process.wait()
            output.close()
            task.exit_code = exit_code
            if task.status == TaskStatus.CANCELLED:
                self._finish(task, TaskStatus.CANCELLED)
            elif exit_code == 0:
                self._finish(task, TaskStatus.SUCCEEDED)
            else:
                task.error_message = task.summary() or f"vfox exited with code {exit_code}"
                self._finish(task, TaskStatus.FAILED)
        except Exception as error:  # noqa: BLE001 - the failure is reported on the task.
            task.error_message = error.message if isinstance(error, VfoxError) else str(error)
            self._finish(task, TaskStatus.FAILED)
        finally:
            task.process = None
            self._running = None
            self._pump()

    def _finish(self, task: VfoxTask, status: TaskStatus) -> None:
        task.finished_at = time.time()
        task.set_status(status)
        if not self.is_disposed:
            self.notify_listeners()
            self._finished.emit(task)

    def _trim_finished(self) -> None:
        finished = [task for task in self._all if not task.status.is_active]
        excess = len(finished) - self.max_finished_tasks
        if excess > 0:
            forgotten = set(map(id, finished[:excess]))
            self._all = [task for task in self._all if id(task) not in forgotten]

    def dispose(self) -> None:
        for task in self._all:
            if task.process is not None:
                task.process.kill()
        self._finished.close()
        super().dispose()


def _same_command(a: VfoxCommand, b: VfoxCommand) -> bool:
    return a.working_directory == b.working_directory and a.arguments == b.arguments
