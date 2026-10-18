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

"""The client that drives the vfox command line interface."""

import asyncio
import re
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field

from vfox_plus.core.models import AvailablePlugin, AvailableVersion, ConfigEntry, InstalledSdk, PluginInfo, UseScope
from vfox_plus.core.process import (
    ProcessListener,
    ProcessOptions,
    ProcessOutput,
    ProcessRunner,
    ProcessStartError,
    RunningProcess,
)
from vfox_plus.core.vfox.command import VfoxCommand, VfoxError, VfoxNotFoundError, VfoxOperation
from vfox_plus.core.vfox.output import (
    PLUGIN_INFO_TEMPLATE,
    merge_installed_sdks,
    parse_available_plugins,
    parse_config_list,
    parse_current,
    parse_install_path,
    parse_installed_tree,
    parse_plugin_info,
    parse_search,
    parse_vfox_version,
    summarize_error,
)

VFOX_EXECUTABLE_NAME = "vfox"
"""The name of the vfox executable, looked up in `PATH` when no path is known."""

SESSION_VARIABLES = frozenset({"__VFOX_SHELL", "__VFOX_PID", "__VFOX_CURTMPPATH", "__VFOX_INITIALIZED"})
"""Variables that tie a process to an activated shell session. They are
removed so that a GUI started from an activated terminal does not act on that
terminal's session."""

_NAME_PATTERN = re.compile(r"[A-Za-z0-9][A-Za-z0-9._+-]*")
_CONFIG_KEY_PATTERN = re.compile(r"[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*")
_VERSION_FORBIDDEN = re.compile(r"[\s@]")


@dataclass(frozen=True, slots=True)
class VfoxLaunchConfig:
    """How vfox processes are launched."""

    executable: str = VFOX_EXECUTABLE_NAME
    """Path to the vfox executable, or a bare name to look up in `PATH`."""
    base_environment: Mapping[str, str] = field(default_factory=dict)
    """The environment inherited from the application."""
    extra_environment: Mapping[str, str] = field(default_factory=dict)
    """Variables added by the user, for example download mirrors such as
    `VFOX_NODEJS_MIRROR`. They override :attr:`base_environment`."""
    neutral_directory: str = "/"
    """Working directory for commands that are not tied to a project.

    vfox reads a project `.vfox.toml` (and legacy files such as `.nvmrc`) from
    its working directory, so global queries must run somewhere that cannot
    contain one. The filesystem root is used by default."""


def vfox_environment(config: VfoxLaunchConfig, *, hook_shell: bool = False) -> dict[str, str]:
    """The complete environment of a vfox child process."""
    environment = {
        key: value
        for key, value in {**config.base_environment, **config.extra_environment}.items()
        if key not in SESSION_VARIABLES
    }
    # Disable colours; vfox (through pterm) honours https://no-color.org.
    environment["NO_COLOR"] = "1"
    if hook_shell:
        environment["__VFOX_SHELL"] = "bash"
    return environment


def is_valid_plugin_name(name: str) -> bool:
    """Whether `name` is accepted as a plugin name or alias."""
    return _NAME_PATTERN.fullmatch(name) is not None


class VfoxCli:
    """Drives the vfox CLI.

    Read-only queries are executed directly and parsed into models.
    State-changing operations are returned as :class:`VfoxCommand` values
    instead, so that the caller can queue them, stream their output and retry
    them.

    Every command line built here is documented in
    `docs/vfox-cli-contract.md`.
    """

    def __init__(self, runner: ProcessRunner, config: VfoxLaunchConfig | None = None) -> None:
        self._runner = runner
        self.launch_config = config or VfoxLaunchConfig()
        """How vfox is launched. Can be replaced at any time, for example after
        the user picked a different executable; affects subsequent invocations
        only."""

    # Queries

    async def version(self) -> str:
        """The version of vfox, for example `1.0.12`."""
        args = ["--version"]
        output = await self._query(args)
        version = parse_vfox_version(f"{output.stdout}\n{output.stderr}")
        if version is None:
            raise VfoxError("Unrecognised version output", arguments=args, stdout=output.stdout, stderr=output.stderr)
        return version

    async def installed_sdks(self) -> list[InstalledSdk]:
        """All installed plugins with their installed and current versions.

        Runs `vfox list` and `vfox current` and merges their results.
        """
        listing, current_output = await asyncio.gather(
            self._query(["list"], allow_failure=True),
            self._query(["current"]),
        )
        current = parse_current(current_output.stdout)
        if listing.exit_code != 0:
            # `vfox list` fails with "you don't have any sdk installed yet"
            # when no plugin is installed; `vfox current` then prints nothing.
            if not current:
                return []
            raise self._failure(["list"], listing)
        return merge_installed_sdks(parse_installed_tree(listing.stdout), current)

    async def plugin_info(self, name: str) -> PluginInfo:
        """Metadata of the installed plugin `name`."""
        args = ["info", _check_name(name), "--format", PLUGIN_INFO_TEMPLATE]
        output = await self._query(args)
        info = parse_plugin_info(output.stdout)
        if info is None:
            raise VfoxError(
                "Unrecognised plugin information", arguments=args, stdout=output.stdout, stderr=output.stderr
            )
        return info

    async def install_path(self, name: str, version: str) -> str | None:
        """The directory of an installed version, or `None` if it is not
        installed.
        """
        output = await self._query(["info", f"{_check_name(name)}@{_check_version(version)}"])
        return parse_install_path(output.stdout)

    async def search_versions(self, name: str) -> list[AvailableVersion]:
        """Versions of `name` that can be installed, newest first.

        May take a while: plugins fetch the list from the network unless vfox
        has it cached (see the `cache.availableHookDuration` setting).
        """
        output = await self._query(["search", _check_name(name)])
        return parse_search(output.stdout)

    async def available_plugins(self) -> list[AvailablePlugin]:
        """Plugins listed in the vfox registry."""
        output = await self._query(["available"])
        return parse_available_plugins(output.stdout)

    async def config_entries(self) -> list[ConfigEntry]:
        """The vfox configuration as `key = value` entries."""
        output = await self._query(["config", "--list"])
        return parse_config_list(output.stdout)

    async def set_config(self, key: str, value: str) -> None:
        """Sets the configuration `key` to `value`."""
        await self._query(["config", _check_config_key(key), value])

    async def unset_config(self, key: str) -> None:
        """Resets the configuration `key` to its default value."""
        await self._query(["config", "--unset", _check_config_key(key)])

    # Commands

    def install(self, name: str, version: str) -> VfoxCommand:
        """Installs `version` of `name`."""
        return VfoxCommand(
            VfoxOperation.INSTALL,
            ("install", "--yes", f"{_check_name(name)}@{_check_version(version)}"),
            sdk=name,
            version=version,
        )

    def uninstall(self, name: str, version: str) -> VfoxCommand:
        """Uninstalls `version` of `name`."""
        return VfoxCommand(
            VfoxOperation.UNINSTALL,
            ("uninstall", f"{_check_name(name)}@{_check_version(version)}"),
            sdk=name,
            version=version,
        )

    def use(
        self, name: str, version: str, scope: UseScope = "global", project_directory: str | None = None
    ) -> VfoxCommand:
        """Makes `version` of `name` the current version in `scope`.

        `project_directory` is required for the project scope.
        """
        return VfoxCommand(
            VfoxOperation.USE,
            ("use", _scope_flag(scope), f"{_check_name(name)}@{_check_version(version)}"),
            sdk=name,
            version=version,
            working_directory=_scope_directory(scope, project_directory),
            requires_hook_environment=True,
        )

    def unuse(self, name: str, scope: UseScope = "global", project_directory: str | None = None) -> VfoxCommand:
        """Removes the current version of `name` from `scope`."""
        return VfoxCommand(
            VfoxOperation.UNUSE,
            ("unuse", _scope_flag(scope), _check_name(name)),
            sdk=name,
            working_directory=_scope_directory(scope, project_directory),
        )

    def add_plugin(self, name: str, *, source: str | None = None, alias: str | None = None) -> VfoxCommand:
        """Adds the plugin `name` from the registry, or from `source` (a URL
        or a local path) when given. `alias` installs the plugin under another
        name.
        """
        trimmed_source = _blank_to_none(source)
        trimmed_alias = _blank_to_none(alias)
        trimmed_name = _blank_to_none(name)
        if trimmed_name is None and trimmed_source is None:
            raise ValueError("A plugin name or a source is required")
        args = ["add"]
        if trimmed_source is not None:
            args += ["--source", trimmed_source]
        if trimmed_alias is not None:
            args += ["--alias", _check_name(trimmed_alias)]
        if trimmed_name is not None:
            args.append(_check_name(trimmed_name))
        return VfoxCommand(VfoxOperation.ADD_PLUGIN, tuple(args), sdk=trimmed_alias or trimmed_name)

    def remove_plugin(self, name: str) -> VfoxCommand:
        """Removes the plugin `name` together with all of its installed
        versions.
        """
        return VfoxCommand(VfoxOperation.REMOVE_PLUGIN, ("remove", "--yes", _check_name(name)), sdk=name)

    def update_plugin(self, name: str) -> VfoxCommand:
        """Updates the plugin `name` to its latest version."""
        return VfoxCommand(VfoxOperation.UPDATE_PLUGIN, ("update", _check_name(name)), sdk=name)

    def update_all_plugins(self) -> VfoxCommand:
        """Updates every installed plugin."""
        return VfoxCommand(VfoxOperation.UPDATE_ALL_PLUGINS, ("update", "--all"))

    async def start(self, command: VfoxCommand, listener: ProcessListener) -> RunningProcess:
        """Starts `command` and returns the running process.

        Raises :class:`VfoxNotFoundError` if vfox cannot be started.
        """
        config = self.launch_config
        try:
            return await self._runner.start(
                config.executable,
                command.arguments,
                ProcessOptions(
                    working_directory=command.working_directory or config.neutral_directory,
                    environment=vfox_environment(config, hook_shell=command.requires_hook_environment),
                ),
                listener,
            )
        except ProcessStartError as error:
            raise VfoxNotFoundError(config.executable, str(error)) from error

    # Internals

    async def _query(self, args: Sequence[str], *, allow_failure: bool = False) -> ProcessOutput:
        config = self.launch_config
        try:
            output = await self._runner.run(
                config.executable,
                args,
                ProcessOptions(working_directory=config.neutral_directory, environment=vfox_environment(config)),
            )
        except ProcessStartError as error:
            raise VfoxNotFoundError(config.executable, str(error)) from error
        if output.exit_code != 0 and not allow_failure:
            raise self._failure(args, output)
        return output

    @staticmethod
    def _failure(args: Sequence[str], output: ProcessOutput) -> VfoxError:
        summary = summarize_error(output.stdout, output.stderr)
        return VfoxError(
            summary or f"vfox exited with code {output.exit_code}",
            arguments=args,
            exit_code=output.exit_code,
            stdout=output.stdout,
            stderr=output.stderr,
        )


def _scope_flag(scope: UseScope) -> str:
    return "--global" if scope == "global" else "--project"


def _scope_directory(scope: UseScope, project_directory: str | None) -> str | None:
    if scope == "global":
        return None
    directory = _blank_to_none(project_directory)
    if directory is None:
        raise ValueError("A project directory is required for the project scope")
    return directory


def _check_name(name: str) -> str:
    """Rejects names that vfox could mistake for flags or that contain
    separators. Arguments never pass through a shell, so this only guards
    against option injection and typos.
    """
    if not is_valid_plugin_name(name):
        raise ValueError(f"Not a valid plugin name: {name}")
    return name


def _check_version(version: str) -> str:
    if version == "" or version.startswith("-") or _VERSION_FORBIDDEN.search(version):
        raise ValueError(f"Not a valid version: {version}")
    return version


def _check_config_key(key: str) -> str:
    if _CONFIG_KEY_PATTERN.fullmatch(key) is None:
        raise ValueError(f"Not a valid configuration key: {key}")
    return key


def _blank_to_none(value: str | None) -> str | None:
    trimmed = value.strip() if value is not None else ""
    return trimmed or None
