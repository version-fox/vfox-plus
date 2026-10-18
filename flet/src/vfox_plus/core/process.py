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

"""A thin, testable abstraction over spawning child processes."""

import asyncio
import contextlib
from collections.abc import Callable, Mapping, Sequence
from dataclasses import dataclass
from typing import Protocol


@dataclass(frozen=True, slots=True)
class ProcessOutput:
    """Everything a finished process wrote, and how it exited."""

    exit_code: int
    """The exit code. Zero means success."""
    stdout: str
    """Everything written to standard output, decoded as UTF-8."""
    stderr: str
    """Everything written to standard error, decoded as UTF-8."""


@dataclass(frozen=True, slots=True)
class ProcessOptions:
    """Where and how a child process runs."""

    working_directory: str
    """The working directory of the child."""
    environment: Mapping[str, str]
    """The complete environment of the child: the environment of the
    application is not inherited implicitly, so that callers can remove
    variables as well as add them."""


class ProcessListener(Protocol):
    """Receives the output of a started process while it runs."""

    def stdout(self, chunk: bytes) -> None:
        """A chunk of raw standard output."""

    def stderr(self, chunk: bytes) -> None:
        """A chunk of raw standard error."""


class RunningProcess(Protocol):
    """A child process whose output is consumed while it runs."""

    async def wait(self) -> int:
        """The exit code, once the process has ended and every chunk of its
        output was delivered to the listener.
        """
        ...

    def kill(self) -> None:
        """Asks the process to terminate with `SIGTERM`."""


class ProcessRunner(Protocol):
    """Spawns child processes.

    The application never spawns processes directly, which keeps the vfox
    client testable with a fake runner and leaves room for a remote
    implementation (see `docs/architecture.md`). Standard input is closed
    immediately: nothing is ever written to the child, and an open stdin would
    make an unexpected interactive prompt wait forever.
    """

    async def run(self, executable: str, args: Sequence[str], options: ProcessOptions) -> ProcessOutput:
        """Runs `executable` to completion and collects its output.

        Raises :class:`ProcessStartError` if the executable cannot be started.
        """
        ...

    async def start(
        self, executable: str, args: Sequence[str], options: ProcessOptions, listener: ProcessListener
    ) -> RunningProcess:
        """Starts `executable` and returns as soon as it runs.

        Raises :class:`ProcessStartError` if the executable cannot be started.
        """
        ...


class ProcessStartError(Exception):
    """An executable could not be started, for example because it is missing."""

    def __init__(self, executable: str, message: str) -> None:
        super().__init__(message)
        self.executable = executable


class AsyncioProcessRunner:
    """The :class:`ProcessRunner` of the desktop, on asyncio subprocesses."""

    async def run(self, executable: str, args: Sequence[str], options: ProcessOptions) -> ProcessOutput:
        process = await self._spawn(executable, args, options)
        stdout, stderr = await process.communicate()
        return ProcessOutput(
            exit_code=_exit_code(process.returncode),
            stdout=stdout.decode("utf-8", errors="replace"),
            stderr=stderr.decode("utf-8", errors="replace"),
        )

    async def start(
        self, executable: str, args: Sequence[str], options: ProcessOptions, listener: ProcessListener
    ) -> RunningProcess:
        process = await self._spawn(executable, args, options)
        return _AsyncioRunningProcess(process, listener)

    @staticmethod
    async def _spawn(executable: str, args: Sequence[str], options: ProcessOptions) -> asyncio.subprocess.Process:
        try:
            return await asyncio.create_subprocess_exec(
                executable,
                *args,
                cwd=options.working_directory,
                env=dict(options.environment),
                stdin=asyncio.subprocess.DEVNULL,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
        except OSError as error:
            raise ProcessStartError(executable, f"{executable}: {error.strerror or error}") from error


class _AsyncioRunningProcess:
    def __init__(self, process: asyncio.subprocess.Process, listener: ProcessListener) -> None:
        self._process = process
        assert process.stdout is not None
        assert process.stderr is not None
        self._readers = asyncio.gather(
            _pump(process.stdout, listener.stdout),
            _pump(process.stderr, listener.stderr),
        )

    async def wait(self) -> int:
        await self._readers
        return _exit_code(await self._process.wait())

    def kill(self) -> None:
        if self._process.returncode is None:
            with contextlib.suppress(ProcessLookupError):
                self._process.terminate()


async def _pump(stream: asyncio.StreamReader, deliver: Callable[[bytes], None]) -> None:
    while chunk := await stream.read(65536):
        deliver(chunk)


def _exit_code(code: int | None) -> int:
    # A process killed by a signal has the negative signal number as return
    # code, which is a failure like any other non-zero code.
    return 1 if code is None else code
