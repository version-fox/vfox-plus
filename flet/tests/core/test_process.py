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

"""The process runner of the desktop, with real processes."""

import asyncio
from pathlib import Path

import pytest

from vfox_plus.core.process import AsyncioProcessRunner, ProcessOptions, ProcessStartError

OPTIONS = ProcessOptions(working_directory="/", environment={"PATH": "/usr/bin:/bin", "GREETING": "hello"})


class _Collector:
    def __init__(self) -> None:
        self.out = b""
        self.err = b""

    def stdout(self, chunk: bytes) -> None:
        self.out += chunk

    def stderr(self, chunk: bytes) -> None:
        self.err += chunk


async def test_run_collects_the_output_the_exit_code_and_uses_only_the_given_environment(tmp_path: Path) -> None:
    script = 'echo "$GREETING $HOME" ; echo oops >&2 ; pwd ; read line || echo "no input" ; exit 3'
    output = await AsyncioProcessRunner().run(
        "/bin/sh", ["-c", script], ProcessOptions(str(tmp_path), {"GREETING": "hello"})
    )
    assert output.exit_code == 3
    assert output.stdout == f"hello \n{tmp_path}\nno input\n"
    assert output.stderr == "oops\n"


async def test_run_decodes_malformed_output() -> None:
    output = await AsyncioProcessRunner().run("/bin/sh", ["-c", r"printf 'a\377b'"], OPTIONS)
    assert output.stdout == "a�b"


async def test_run_reports_a_missing_executable() -> None:
    with pytest.raises(ProcessStartError) as raised:
        await AsyncioProcessRunner().run("/nonexistent/vfox", [], OPTIONS)
    assert raised.value.executable == "/nonexistent/vfox"
    assert "/nonexistent/vfox" in str(raised.value)


async def test_start_streams_the_output_until_the_process_exits() -> None:
    collector = _Collector()
    process = await AsyncioProcessRunner().start(
        "/bin/sh", ["-c", "echo one; echo two >&2; echo three"], OPTIONS, collector
    )
    assert await process.wait() == 0
    assert collector.out == b"one\nthree\n"
    assert collector.err == b"two\n"
    process.kill()  # Killing an ended process does nothing.


async def test_kill_terminates_a_running_process() -> None:
    collector = _Collector()
    process = await AsyncioProcessRunner().start("/bin/sh", ["-c", "echo started; exec sleep 30"], OPTIONS, collector)
    while collector.out == b"":
        await asyncio.sleep(0.01)
    process.kill()
    assert await asyncio.wait_for(process.wait(), 5) == -15


async def test_start_reports_a_missing_executable() -> None:
    with pytest.raises(ProcessStartError):
        await AsyncioProcessRunner().start("/nonexistent/vfox", [], OPTIONS, _Collector())
