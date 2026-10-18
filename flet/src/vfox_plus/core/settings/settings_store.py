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

"""Persistence of :class:`AppSettings`."""

import json
import os
import tempfile
from collections.abc import Mapping
from typing import Protocol

from vfox_plus.core.settings.app_settings import DEFAULT_SETTINGS, AppSettings, settings_from_json, settings_to_json


class SettingsStore(Protocol):
    """Loads and saves :class:`AppSettings`."""

    def load(self) -> AppSettings:
        """Loads the settings, or returns the defaults if none are stored."""
        ...

    def save(self, settings: AppSettings) -> None:
        """Persists `settings`."""
        ...


def settings_path(environment: Mapping[str, str], fallback_home: str = "/tmp") -> str:
    """The settings file for `environment`:
    `$XDG_CONFIG_HOME/vfox-plus/settings.json`, where `XDG_CONFIG_HOME`
    defaults to `~/.config` as defined by the XDG Base Directory
    Specification.
    """
    xdg_config_home = environment.get("XDG_CONFIG_HOME", "")
    base = xdg_config_home if xdg_config_home.startswith("/") else f"{environment.get('HOME', fallback_home)}/.config"
    return f"{base}/vfox-plus/settings.json"


class JsonFileSettingsStore:
    """Stores settings as JSON in a file, by default the one of
    :func:`settings_path`.

    Corrupted files are ignored rather than fatal: the application then
    starts with the defaults.
    """

    def __init__(self, path: str) -> None:
        self.path = path
        """The settings file."""

    def load(self) -> AppSettings:
        try:
            with open(self.path, encoding="utf-8") as file:
                return settings_from_json(json.load(file))
        except OSError, ValueError:
            return DEFAULT_SETTINGS

    def save(self, settings: AppSettings) -> None:
        # A temporary file is written first and renamed, so that a crash never
        # leaves a truncated settings file behind.
        directory = os.path.dirname(self.path)
        os.makedirs(directory, exist_ok=True)
        descriptor, temporary = tempfile.mkstemp(prefix=".settings-", suffix=".json", dir=directory)
        try:
            with os.fdopen(descriptor, "w", encoding="utf-8") as file:
                file.write(json.dumps(settings_to_json(settings), indent=2, ensure_ascii=False) + "\n")
            os.replace(temporary, self.path)
        except BaseException:
            os.unlink(temporary)
            raise


class MemorySettingsStore:
    """Keeps settings in memory. Used by tests and as a fallback."""

    def __init__(self, settings: AppSettings = DEFAULT_SETTINGS) -> None:
        self.settings = settings
        self.save_count = 0
        """How many times :meth:`save` was called."""

    def load(self) -> AppSettings:
        return self.settings

    def save(self, settings: AppSettings) -> None:
        self.save_count += 1
        self.settings = settings
