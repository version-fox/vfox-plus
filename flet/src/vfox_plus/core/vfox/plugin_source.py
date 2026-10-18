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

"""Sources passed to `vfox add --source`."""

import os
import shutil
import tempfile
from collections.abc import Mapping
from dataclasses import dataclass


def is_remote_plugin_source(source: str) -> bool:
    """Whether `source` is a URL rather than a local path."""
    return source.startswith(("http://", "https://"))


def vfox_moves_plugin_source(source: str) -> bool:
    """Whether vfox would move `source` instead of reading it.

    vfox (still in 1.0.13) renames a local `.lua` plugin into its temporary
    directory, which deletes the user's file and fails when the file is on
    another filesystem; local `.zip` archives and URLs are only read. See
    `docs/vfox-cli-contract.md`.
    """
    return not is_remote_plugin_source(source) and source.lower().endswith(".lua")


def vfox_temporary_directory(environment: Mapping[str, str]) -> str:
    """The temporary directory of vfox when it runs with `environment`:
    `~/.version-fox/tmp` if the legacy `~/.version-fox` exists, `~/.vfox/tmp`
    otherwise. Without `HOME`, where vfox does not run at all, the system
    temporary directory.
    """
    home = environment.get("HOME", "")
    if home == "":
        return tempfile.gettempdir()
    legacy = f"{home}/.version-fox"
    return f"{legacy if os.path.exists(legacy) else f'{home}/.vfox'}/tmp"


@dataclass(frozen=True, slots=True)
class PreparedPluginSource:
    """A plugin source that is safe to hand to vfox."""

    path: str
    """The path or URL to pass to `vfox add --source`."""
    copy_directory: str | None = None
    """The directory of the private copy of the original source, if one was
    made."""

    @property
    def is_copy(self) -> bool:
        """Whether :attr:`path` is a private copy of the original source."""
        return self.copy_directory is not None

    def dispose(self) -> None:
        """Deletes the private copy, if one was made."""
        if self.copy_directory is not None:
            shutil.rmtree(self.copy_directory, ignore_errors=True)


def prepare_plugin_source(source: str, temporary_root: str) -> PreparedPluginSource:
    """Prepares `source` for `vfox add --source`.

    A local `.lua` file is copied into a new directory under
    `temporary_root`, so that vfox moves the copy and the user's file stays
    where it is. Pass the :func:`vfox_temporary_directory`: vfox moves the
    file with a rename into a new directory there, which only works on the
    same filesystem. Other sources are returned as they are. If the copy
    cannot be made, the original path is returned and vfox reports the
    problem; it fails to create its own directory in `temporary_root` before
    moving anything.
    """
    if not vfox_moves_plugin_source(source):
        return PreparedPluginSource(source)
    directory: str | None = None
    try:
        os.makedirs(temporary_root, exist_ok=True)
        directory = tempfile.mkdtemp(prefix="vfox-plus-plugin-", dir=temporary_root)
        copy = os.path.join(directory, os.path.basename(source) or "plugin.lua")
        shutil.copyfile(source, copy)
        return PreparedPluginSource(copy, directory)
    except OSError:
        if directory is not None:
            shutil.rmtree(directory, ignore_errors=True)
        return PreparedPluginSource(source)
