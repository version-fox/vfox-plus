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

"""Finds the vfox executable."""

import os
import stat
from collections.abc import Callable, Mapping

from vfox_plus.core.vfox.cli import VFOX_EXECUTABLE_NAME


def is_executable_file(path: str) -> bool:
    """Whether `path` is an existing regular file that anybody may execute."""
    try:
        mode = os.stat(path).st_mode
    except OSError:
        return False
    return stat.S_ISREG(mode) and mode & (stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH) != 0


class VfoxLocator:
    """Looks for the vfox executable in `PATH` and in common install
    locations.

    Applications started from a desktop launcher do not source the user's
    shell profile, so directories such as `~/.local/bin` are often missing
    from their `PATH`. The well-known install locations of the official
    installers and package managers are therefore searched as well.
    """

    def __init__(
        self, environment: Mapping[str, str], is_executable: Callable[[str], bool] = is_executable_file
    ) -> None:
        self._environment = environment
        self._is_executable = is_executable

    @property
    def fallback_directories(self) -> list[str]:
        """Directories searched in addition to `PATH`, in order."""
        home = self._environment.get("HOME")
        return [
            "/usr/local/bin",
            "/usr/bin",
            *([] if home is None else [f"{home}/.local/bin", f"{home}/bin", f"{home}/.linuxbrew/bin"]),
            "/home/linuxbrew/.linuxbrew/bin",
            "/snap/bin",
        ]

    @property
    def search_directories(self) -> list[str]:
        """All directories that are searched, without duplicates."""
        path = [entry for entry in self._environment.get("PATH", "").split(":") if entry != ""]
        return list(dict.fromkeys([*path, *self.fallback_directories]))

    def locate(self) -> str | None:
        """The path of the first vfox executable found, or `None`."""
        for directory in self.search_directories:
            candidate = f"{directory}/{VFOX_EXECUTABLE_NAME}"
            if self._is_executable(candidate):
                return candidate
        return None
