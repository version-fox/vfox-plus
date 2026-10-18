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

"""Test doubles shared by the tests."""

import asyncio
import re
from collections.abc import Callable, Mapping, Sequence
from dataclasses import dataclass
from pathlib import Path

from vfox_plus.core.process import ProcessListener, ProcessOptions, ProcessOutput, ProcessStartError
from vfox_plus.core.vfox.output import PLUGIN_INFO_TEMPLATE

SHARED = Path(__file__).resolve().parents[2] / "shared"
"""The shared resources of the repository."""

FIXTURES_VERSION = "1.0.13"
"""The captured vfox outputs in `shared/fixtures` that the tests use by
default."""

_ESCAPE = re.compile(r"\\(?:x([0-9a-f]{2})|\\)")


def read_capture(path: Path) -> str:
    """Reads a capture of `shared/fixtures`, in which control characters are
    written as `\\xHH` and backslashes as `\\\\`."""
    text = path.read_text(encoding="utf-8")
    return _ESCAPE.sub(lambda match: "\\" if match.group(1) is None else chr(int(match.group(1), 16)), text)


def fixture_output(name: str, exit_code: int = 0) -> ProcessOutput:
    """The result of the fixture `name`, with `exit_code`."""
    base = SHARED / "fixtures" / f"vfox-{FIXTURES_VERSION}"
    return ProcessOutput(exit_code, read_capture(base / f"{name}.stdout"), read_capture(base / f"{name}.stderr"))


@dataclass(frozen=True)
class ProcessCall:
    """One recorded invocation of a :class:`FakeProcessRunner`."""

    executable: str
    arguments: tuple[str, ...]
    working_directory: str
    environment: Mapping[str, str]
    streamed: bool

    @property
    def command(self) -> str:
        return " ".join(self.arguments)


class FakeRunningProcess:
    """A running process whose output and exit are driven by the test."""

    def __init__(self) -> None:
        self.killed = False
        """Whether :meth:`kill` was called."""
        self._listener: ProcessListener | None = None
        self._exit: asyncio.Future[int] = asyncio.get_running_loop().create_future()

    @classmethod
    def completed(cls, output: ProcessOutput) -> FakeRunningProcess:
        """A process that prints `output` and exits right away."""
        process = cls()

        def finish() -> None:
            process.write_stdout(output.stdout)
            process.write_stderr(output.stderr)
            process.exit(output.exit_code)

        asyncio.get_running_loop().call_soon(finish)
        return process

    def attach(self, listener: ProcessListener) -> None:
        self._listener = listener

    def write_stdout(self, text: str) -> None:
        if text and self._listener is not None:
            self._listener.stdout(text.encode())

    def write_stderr(self, text: str) -> None:
        if text and self._listener is not None:
            self._listener.stderr(text.encode())

    def exit(self, code: int) -> None:
        """Ends the process with `code`."""
        if not self._exit.done():
            self._exit.set_result(code)

    async def wait(self) -> int:
        return await self._exit

    def kill(self) -> None:
        self.killed = True
        self.exit(-15)


class FakeProcessRunner:
    """A scripted process runner.

    Responses are matched by the joined argument list, so tests read like the
    command lines they expect. Unmatched invocations fail with exit code 127.
    """

    def __init__(self) -> None:
        self.invocations: list[ProcessCall] = []
        """Every invocation, in order."""
        self.missing_executable = False
        """When set, every invocation fails as if the executable did not
        exist."""
        self._responses: dict[str, Callable[[], ProcessOutput]] = {}
        self._streams: dict[str, Callable[[], FakeRunningProcess]] = {}

    def on(self, args: str, output: ProcessOutput) -> FakeProcessRunner:
        """Responds to `vfox <args>` with `output`."""
        self._responses[args] = lambda: output
        return self

    def on_dynamic(self, args: str, respond: Callable[[], ProcessOutput]) -> FakeProcessRunner:
        """Responds to `vfox <args>` with the result of `respond`."""
        self._responses[args] = respond
        return self

    def on_start(self, args: str, create: Callable[[], FakeRunningProcess]) -> FakeProcessRunner:
        """Responds to a started `vfox <args>` with the process from
        `create`."""
        self._streams[args] = create
        return self

    def calls(self, prefix: str) -> list[ProcessCall]:
        """The calls whose arguments start with `prefix`."""
        return [call for call in self.invocations if call.command.startswith(prefix)]

    async def run(self, executable: str, args: Sequence[str], options: ProcessOptions) -> ProcessOutput:
        self._record(executable, args, options, streamed=False)
        if self.missing_executable:
            raise ProcessStartError(executable, "No such file or directory")
        respond = self._responses.get(" ".join(args))
        return respond() if respond else ProcessOutput(127, "", "unexpected command")

    async def start(
        self, executable: str, args: Sequence[str], options: ProcessOptions, listener: ProcessListener
    ) -> FakeRunningProcess:
        self._record(executable, args, options, streamed=True)
        if self.missing_executable:
            raise ProcessStartError(executable, "No such file or directory")
        key = " ".join(args)
        create = self._streams.get(key)
        respond = self._responses.get(key)
        process = (
            create() if create else FakeRunningProcess.completed(respond() if respond else ProcessOutput(0, "", ""))
        )
        process.attach(listener)
        return process

    def _record(self, executable: str, args: Sequence[str], options: ProcessOptions, *, streamed: bool) -> None:
        self.invocations.append(
            ProcessCall(executable, tuple(args), options.working_directory, dict(options.environment), streamed)
        )


def fixture_runner() -> FakeProcessRunner:
    """A runner that answers the read-only commands like the `list` and
    `current` fixtures: maven 3.9.16 (current) and 3.8.9, mvn2 without
    versions and nodejs 24.11.1 (current) and 22.21.0."""
    return (
        FakeProcessRunner()
        .on("--version", fixture_output("version"))
        .on("list", fixture_output("list"))
        .on("current", fixture_output("current"))
        .on("available", fixture_output("available"))
        .on("config --list", fixture_output("config-list"))
        .on("search maven", fixture_output("search-maven-installed"))
        .on("search nodejs", fixture_output("search-nodejs"))
        .on(f"info nodejs --format {PLUGIN_INFO_TEMPLATE}", fixture_output("info"))
    )


async def settle(rounds: int = 10) -> None:
    """Waits until the pending callbacks and tasks have run."""
    for _ in range(rounds):
        await asyncio.sleep(0)
