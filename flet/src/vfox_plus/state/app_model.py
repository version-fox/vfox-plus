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

"""The application state: every controller, wired together."""

from collections.abc import Mapping

from vfox_plus.core.models import AvailablePlugin, InstalledSdk
from vfox_plus.core.process import ProcessRunner
from vfox_plus.core.settings.app_settings import AppSettings
from vfox_plus.core.settings.settings_store import SettingsStore
from vfox_plus.core.vfox.cli import VfoxCli, VfoxLaunchConfig, vfox_environment
from vfox_plus.core.vfox.command import VfoxCommand
from vfox_plus.core.vfox.locator import VfoxLocator
from vfox_plus.core.vfox.plugin_source import PreparedPluginSource, prepare_plugin_source, vfox_temporary_directory
from vfox_plus.state.async_resource import AsyncResource
from vfox_plus.state.controllers import (
    SdkDetailController,
    SettingsController,
    VfoxAvailability,
    VfoxConfigController,
    VfoxController,
)
from vfox_plus.state.notifier import spawn
from vfox_plus.state.task_queue import TaskQueue, TaskStatus, VfoxTask


class AppModel:
    """The application state: every controller, wired together.

    The model owns the :class:`VfoxCli` and keeps the derived data
    consistent: installed SDKs are reloaded whenever vfox becomes available or
    a task that changes them finishes.
    """

    def __init__(
        self,
        runner: ProcessRunner,
        settings_store: SettingsStore,
        environment: Mapping[str, str],
        locator: VfoxLocator | None = None,
    ) -> None:
        self.cli = VfoxCli(runner, VfoxLaunchConfig(base_environment=environment))
        """The vfox client shared by all controllers."""
        self.settings = SettingsController(settings_store)
        """Application preferences."""
        self.vfox = VfoxController(self.cli, locator or VfoxLocator(environment), environment)
        """Availability of the vfox executable."""
        self.installed: AsyncResource[list[InstalledSdk]] = AsyncResource(self.cli.installed_sdks)
        """Installed plugins and their versions."""
        self.registry: AsyncResource[list[AvailablePlugin]] = AsyncResource(self.cli.available_plugins)
        """Plugins available in the registry."""
        self.vfox_config = VfoxConfigController(self.cli)
        """The vfox configuration."""
        self.tasks = TaskQueue(self.cli)
        """State-changing operations."""
        self._plugin_copies: dict[int, PreparedPluginSource] = {}
        self._last_availability = VfoxAvailability.UNKNOWN
        self._applied_settings: AppSettings | None = None
        self._stops = [
            self.tasks.on_finished(self._on_task_finished),
            self.vfox.add_listener(self._on_vfox_changed),
            self.settings.add_listener(self._on_settings_changed),
        ]

    async def start(self) -> None:
        """Loads the settings and probes vfox."""
        self.settings.load()
        await self._apply_settings()

    def installed_sdk(self, name: str) -> InstalledSdk | None:
        """The installed SDK called `name`, if it is installed."""
        return next((sdk for sdk in self.installed.value or () if sdk.name == name), None)

    def sdk_detail(self, name: str) -> SdkDetailController:
        """Creates the controller for the detail view of `name`."""
        return SdkDetailController(self.cli, name)

    def run(self, command: VfoxCommand) -> VfoxTask:
        """Queues `command` and returns its task."""
        return self.tasks.enqueue(command)

    def add_plugin_from_source(self, source: str, alias: str | None) -> VfoxTask:
        """Adds a plugin from `source`, a URL or a local `.zip` or `.lua`
        file, optionally under `alias`.

        vfox moves a local `.lua` file away instead of reading it, so such a
        file is copied into the temporary directory of vfox first and vfox
        gets the copy; the copy is deleted once the task has finished.
        """
        temporary_root = vfox_temporary_directory(vfox_environment(self.cli.launch_config))
        prepared = prepare_plugin_source(source, temporary_root)
        task = self.run(self.cli.add_plugin("", source=prepared.path, alias=alias))
        if prepared.is_copy:
            self._plugin_copies[task.id] = prepared
        return task

    def _on_vfox_changed(self) -> None:
        availability = self.vfox.availability
        if availability == self._last_availability:
            return
        self._last_availability = availability
        if availability == VfoxAvailability.READY:
            self.installed.load()
            if self.registry.has_value or self.registry.error is not None:
                self.registry.load()
            if self.vfox_config.has_value or self.vfox_config.error is not None:
                self.vfox_config.load()

    def _on_settings_changed(self) -> None:
        previous = self._applied_settings
        current = self.settings.settings
        if previous is not None and (
            previous.vfox_executable != current.vfox_executable
            or dict(previous.extra_environment) != dict(current.extra_environment)
        ):
            spawn(self._apply_settings())

    async def _apply_settings(self) -> None:
        self._applied_settings = self.settings.settings
        # Force a reload of the dependent data even if vfox stays available.
        self._last_availability = VfoxAvailability.UNKNOWN
        await self.vfox.apply(self.settings.settings)

    def _on_task_finished(self, task: VfoxTask) -> None:
        copy = self._plugin_copies.pop(task.id, None)
        if copy is not None:
            copy.dispose()
        if task.status == TaskStatus.CANCELLED and task.started_at is None:
            return
        self.installed.load()

    def dispose(self) -> None:
        """Releases every resource."""
        for stop in self._stops:
            stop()
        for copy in self._plugin_copies.values():
            copy.dispose()
        self._plugin_copies.clear()
        self.tasks.dispose()
        self.installed.dispose()
        self.registry.dispose()
        self.vfox_config.dispose()
        self.vfox.dispose()
        self.settings.dispose()
