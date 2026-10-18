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

"""The controllers of the state layer, except the task queue."""

import dataclasses
import logging
from collections.abc import Awaitable, Callable, Mapping, Set
from enum import StrEnum

from vfox_plus.core.models import AvailableVersion, ConfigEntry, PluginInfo
from vfox_plus.core.settings.app_settings import DEFAULT_SETTINGS, AppSettings
from vfox_plus.core.settings.settings_store import SettingsStore
from vfox_plus.core.vfox.cli import VFOX_EXECUTABLE_NAME, VfoxCli
from vfox_plus.core.vfox.command import VfoxError, VfoxNotFoundError
from vfox_plus.core.vfox.locator import VfoxLocator
from vfox_plus.state.async_resource import AsyncResource
from vfox_plus.state.notifier import ChangeNotifier

_logger = logging.getLogger(__name__)


class SettingsController(ChangeNotifier):
    """Holds the :class:`AppSettings` and persists every change."""

    def __init__(self, store: SettingsStore) -> None:
        super().__init__()
        self._store = store
        self._settings = DEFAULT_SETTINGS

    @property
    def settings(self) -> AppSettings:
        """The current settings."""
        return self._settings

    def load(self) -> None:
        """Loads the persisted settings."""
        self._settings = self._store.load()
        self.notify_listeners()

    def update(self, change: Callable[[AppSettings], AppSettings]) -> None:
        """Applies `change` to the settings and saves the result."""
        updated = change(self._settings)
        if updated == self._settings:
            return
        self._settings = updated
        self.notify_listeners()
        self._store.save(updated)

    def replace(self, **changes: object) -> None:
        """Changes the fields `changes` of the settings and saves the result."""
        self.update(lambda settings: dataclasses.replace(settings, **changes))  # type: ignore[arg-type]


class VfoxAvailability(StrEnum):
    """Whether vfox can be used."""

    UNKNOWN = "unknown"
    """Not checked yet."""
    CHECKING = "checking"
    """The executable is being probed."""
    READY = "ready"
    """vfox runs and reported its version."""
    NOT_FOUND = "notFound"
    """No vfox executable was found."""
    BROKEN = "broken"
    """An executable was found but did not behave like vfox."""


class VfoxController(ChangeNotifier):
    """Locates vfox, configures the :class:`VfoxCli` and checks that vfox
    works.
    """

    def __init__(self, cli: VfoxCli, locator: VfoxLocator, environment: Mapping[str, str]) -> None:
        super().__init__()
        self._cli = cli
        self._locator = locator
        self._environment = environment
        self._availability = VfoxAvailability.UNKNOWN
        self._executable: str | None = None
        self._custom = False
        self._version: str | None = None
        self._error: Exception | None = None

    @property
    def availability(self) -> VfoxAvailability:
        """Whether vfox can be used."""
        return self._availability

    @property
    def executable(self) -> str | None:
        """The executable in use, if one was found."""
        return self._executable

    @property
    def is_custom_executable(self) -> bool:
        """Whether :attr:`executable` was chosen by the user rather than
        detected.
        """
        return self._custom

    @property
    def version(self) -> str | None:
        """The vfox version, once vfox is ready."""
        return self._version

    @property
    def error(self) -> Exception | None:
        """Why vfox is not available, if it is not."""
        return self._error

    async def apply(self, settings: AppSettings) -> None:
        """Points the CLI at the executable and environment from `settings`
        and checks the result.
        """
        custom = settings.vfox_executable
        self._custom = custom is not None
        self._executable = custom if custom is not None else self._locator.locate()
        self._cli.launch_config = dataclasses.replace(
            self._cli.launch_config,
            executable=self._executable or VFOX_EXECUTABLE_NAME,
            base_environment=self._environment,
            extra_environment=settings.extra_environment,
        )
        await self.check()

    async def check(self) -> None:
        """Runs `vfox --version` to check that vfox works."""
        self._availability = VfoxAvailability.CHECKING
        self.notify_listeners()
        try:
            self._version = await self._cli.version()
            self._error = None
            self._availability = VfoxAvailability.READY
        except Exception as error:
            self._version = None
            self._error = error
            self._availability = (
                VfoxAvailability.NOT_FOUND if isinstance(error, VfoxNotFoundError) else VfoxAvailability.BROKEN
            )
            if not isinstance(error, VfoxError):
                _logger.exception("Checking vfox failed")
        self.notify_listeners()


class VfoxConfigController(AsyncResource[list[ConfigEntry]]):
    """The vfox configuration (`vfox config`), editable key by key."""

    def __init__(self, cli: VfoxCli) -> None:
        super().__init__(cli.config_entries)
        self._cli = cli
        self._saving: set[str] = set()

    def value_for(self, key: str) -> str | None:
        """The value of `key`, or `None` if vfox does not know the key."""
        return next((entry.value for entry in self.value or () if entry.key == key), None)

    def is_saving(self, key: str) -> bool:
        """Whether `key` is being saved."""
        return key in self._saving

    async def set(self, key: str, new_value: str) -> None:
        """Sets `key` to `new_value` and reloads the configuration.

        Raises a :class:`VfoxError` if vfox rejects the value; the
        configuration is reloaded either way.
        """
        await self._save(key, lambda: self._cli.set_config(key, new_value))

    async def reset(self, key: str) -> None:
        """Resets `key` to its default and reloads the configuration."""
        await self._save(key, lambda: self._cli.unset_config(key))

    async def _save(self, key: str, action: Callable[[], Awaitable[None]]) -> None:
        self._saving.add(key)
        self.notify_listeners()
        try:
            await action()
        finally:
            self._saving.discard(key)
            await self.load()


class VersionFilter(StrEnum):
    """Which versions are listed on the "available" tab."""

    ALL = "all"
    """Every version."""
    WITH_NOTE = "withNote"
    """Versions carrying a note such as `LTS`."""
    INSTALLED = "installed"
    """Versions that are installed already."""


class SdkDetailController(ChangeNotifier):
    """State of the detail view of one installed plugin."""

    def __init__(self, cli: VfoxCli, name: str) -> None:
        super().__init__()
        self.name = name
        """The plugin name."""
        self.info: AsyncResource[PluginInfo] = AsyncResource(lambda: cli.plugin_info(name))
        """Plugin metadata, loaded on demand."""
        self.versions: AsyncResource[list[AvailableVersion]] = AsyncResource(lambda: cli.search_versions(name))
        """Installable versions, loaded on demand because the search can be
        slow."""
        self._query = ""
        self._filter = VersionFilter.ALL
        self._stop_listening = self.versions.add_listener(self.notify_listeners)

    @property
    def query(self) -> str:
        """The text the available versions are filtered by."""
        return self._query

    @query.setter
    def query(self, value: str) -> None:
        if value != self._query:
            self._query = value
            self.notify_listeners()

    @property
    def version_filter(self) -> VersionFilter:
        """Which available versions are shown."""
        return self._filter

    @version_filter.setter
    def version_filter(self, value: VersionFilter) -> None:
        if value != self._filter:
            self._filter = value
            self.notify_listeners()

    def visible_versions(self, installed: Set[str]) -> list[AvailableVersion]:
        """The available versions matching :attr:`query` and
        :attr:`version_filter`.

        `installed` holds the installed versions. It is passed in rather than
        taken from the search output, so that it stays current after installs
        without searching again.
        """
        needle = self._query.strip().lower()
        return [
            version
            for version in self.versions.value or ()
            if self._matches_filter(version, installed)
            and (
                needle == ""
                or needle in version.version.lower()
                or (version.note is not None and needle in version.note.lower())
            )
        ]

    def _matches_filter(self, version: AvailableVersion, installed: Set[str]) -> bool:
        match self._filter:
            case VersionFilter.ALL:
                return True
            case VersionFilter.WITH_NOTE:
                return version.note is not None
            case VersionFilter.INSTALLED:
                return version.version in installed

    def dispose(self) -> None:
        self._stop_listening()
        self.info.dispose()
        self.versions.dispose()
        super().dispose()
