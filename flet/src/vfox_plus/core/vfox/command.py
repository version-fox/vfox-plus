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

"""Descriptions of vfox invocations that change state, and vfox errors."""

from collections.abc import Sequence
from dataclasses import dataclass
from enum import StrEnum


class VfoxOperation(StrEnum):
    """The kinds of state-changing operations the application performs."""

    INSTALL = "install"
    """`vfox install --yes <name>@<version>`"""
    UNINSTALL = "uninstall"
    """`vfox uninstall <name>@<version>`"""
    USE = "use"
    """`vfox use --global|--project <name>@<version>`"""
    UNUSE = "unuse"
    """`vfox unuse --global|--project <name>`"""
    ADD_PLUGIN = "addPlugin"
    """`vfox add [--source <url>] [--alias <alias>] <name>`"""
    REMOVE_PLUGIN = "removePlugin"
    """`vfox remove --yes <name>`"""
    UPDATE_PLUGIN = "updatePlugin"
    """`vfox update <name>`"""
    UPDATE_ALL_PLUGINS = "updateAllPlugins"
    """`vfox update --all`"""


@dataclass(frozen=True, slots=True)
class VfoxCommand:
    """A state-changing vfox invocation.

    Commands are values: they are built by `VfoxCli`, queued and executed by
    the task queue, and can be retried by running them again.
    """

    operation: VfoxOperation
    arguments: tuple[str, ...]
    """Arguments passed to the vfox executable."""
    sdk: str | None = None
    """The plugin the command acts on, if any."""
    version: str | None = None
    """The SDK version the command acts on, if any."""
    working_directory: str | None = None
    """Directory to run in. `None` means a neutral directory, so that no
    project configuration is picked up."""
    requires_hook_environment: bool = False
    """Whether vfox must believe that it runs inside an activated shell.

    `vfox use` refuses to run on Unix without shell integration, but with
    `--global` or `--project` it only edits `.vfox.toml` and symlinks, which is
    exactly what a GUI needs. See `docs/vfox-cli-contract.md`."""

    @property
    def subject(self) -> str | None:
        """The SDK and version as `name@version`, or just the name."""
        if self.sdk is not None and self.version is not None:
            return f"{self.sdk}@{self.version}"
        return self.sdk

    @property
    def command_line(self) -> str:
        """The command line, as shown in the task output."""
        return " ".join(["vfox", *self.arguments])


class VfoxError(Exception):
    """A vfox invocation failed."""

    def __init__(
        self,
        message: str,
        *,
        arguments: Sequence[str] = (),
        exit_code: int | None = None,
        stdout: str = "",
        stderr: str = "",
    ) -> None:
        super().__init__(message)
        self.message = message
        self.arguments = tuple(arguments)
        """The arguments vfox was started with."""
        self.exit_code = exit_code
        """The exit code, if the process ran at all."""
        self.stdout = stdout
        self.stderr = stderr


class VfoxNotFoundError(VfoxError):
    """The vfox executable could not be started."""

    def __init__(self, executable: str, message: str) -> None:
        super().__init__(message)
        self.executable = executable
        """The path or name that could not be executed."""
