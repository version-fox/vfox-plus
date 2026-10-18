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

"""The controllers and the application model, against a scripted process
runner."""

import asyncio
import dataclasses
from collections.abc import AsyncIterator
from pathlib import Path

import pytest

from support import FakeProcessRunner, fixture_output, fixture_runner, settle
from vfox_plus.core.process import ProcessOutput
from vfox_plus.core.settings.app_settings import DEFAULT_SETTINGS, ThemePreference
from vfox_plus.core.settings.settings_store import MemorySettingsStore
from vfox_plus.core.vfox.cli import VfoxCli, VfoxLaunchConfig
from vfox_plus.core.vfox.command import VfoxError
from vfox_plus.core.vfox.locator import VfoxLocator
from vfox_plus.state.app_model import AppModel
from vfox_plus.state.async_resource import AsyncResource
from vfox_plus.state.controllers import (
    SdkDetailController,
    SettingsController,
    VersionFilter,
    VfoxAvailability,
    VfoxConfigController,
    VfoxController,
)
from vfox_plus.state.notifier import Emitter, listen_all
from vfox_plus.state.task_queue import TaskStatus


async def test_resource_keeps_the_previous_value_while_reloading_and_records_errors() -> None:
    calls = 0

    async def load() -> int:
        nonlocal calls
        calls += 1
        if calls == 2:
            raise RuntimeError("boom")
        return calls

    resource = AsyncResource(load)
    await resource.load()
    assert resource.value == 1
    assert resource.has_value
    reload = resource.load()
    assert resource.is_loading
    assert resource.value == 1
    await reload
    assert str(resource.error) == "boom"
    assert resource.value == 1
    assert not resource.is_loading


async def test_resource_lets_the_latest_of_overlapping_loads_win() -> None:
    futures: list[asyncio.Future[str]] = []

    def load() -> asyncio.Future[str]:
        future: asyncio.Future[str] = asyncio.get_running_loop().create_future()
        futures.append(future)
        return future

    resource = AsyncResource(load)
    first = resource.load()
    second = resource.load()
    await settle()
    futures[1].set_result("new")
    futures[0].set_result("old")
    await asyncio.gather(first, second)
    assert resource.value == "new"


async def test_resource_ignores_the_failure_of_a_replaced_load() -> None:
    futures: list[asyncio.Future[str]] = []

    def load() -> asyncio.Future[str]:
        future: asyncio.Future[str] = asyncio.get_running_loop().create_future()
        futures.append(future)
        return future

    resource = AsyncResource(load)
    first = resource.load()
    second = resource.load()
    await settle()
    futures[1].set_result("new")
    futures[0].set_exception(RuntimeError("old"))
    await asyncio.gather(first, second)
    assert resource.value == "new"
    assert resource.error is None


async def test_resource_ensure_loaded_loads_once_and_set_value_replaces_the_value() -> None:
    calls = 0

    async def load() -> int:
        nonlocal calls
        calls += 1
        return calls

    resource = AsyncResource(load)
    await asyncio.gather(resource.ensure_loaded(), resource.ensure_loaded())
    await resource.ensure_loaded()
    assert calls == 1
    resource.set_value(9)
    assert resource.value == 9


async def test_resource_notifies_listeners_until_disposed() -> None:
    async def load() -> int:
        return 1

    resource = AsyncResource(load)
    notified = 0

    def count() -> None:
        nonlocal notified
        notified += 1

    stop = listen_all([resource], count)
    await resource.load()
    assert notified == 2
    stop()
    stop()
    await resource.load()
    assert notified == 2
    resource.dispose()
    assert resource.is_disposed
    resource.notify_listeners()


def test_emitter() -> None:
    emitter: Emitter[int] = Emitter()
    seen: list[int] = []
    stop = emitter.listen(seen.append)
    emitter.emit(1)
    stop()
    stop()
    emitter.emit(2)
    emitter.listen(seen.append)
    emitter.close()
    emitter.emit(3)
    assert seen == [1]


def test_settings_controller_persists_changes_and_skips_no_ops() -> None:
    store = MemorySettingsStore(dataclasses.replace(DEFAULT_SETTINGS, theme=ThemePreference.DARK))
    controller = SettingsController(store)
    controller.load()
    assert controller.settings.theme == ThemePreference.DARK
    controller.update(lambda settings: dataclasses.replace(settings, theme=ThemePreference.DARK))
    assert store.save_count == 0
    controller.replace(theme=ThemePreference.LIGHT)
    assert store.save_count == 1
    assert store.load().theme == ThemePreference.LIGHT


@pytest.fixture
def runner() -> FakeProcessRunner:
    return fixture_runner()


def _vfox_controller(runner: FakeProcessRunner, found: str | None = None) -> tuple[VfoxController, VfoxCli]:
    cli = VfoxCli(runner, VfoxLaunchConfig(executable="vfox"))
    return VfoxController(cli, VfoxLocator({}, lambda path: path == found), {"HOME": "/home/u"}), cli


async def test_vfox_uses_the_detected_executable_and_reads_the_version(runner: FakeProcessRunner) -> None:
    vfox, cli = _vfox_controller(runner, "/usr/local/bin/vfox")
    await vfox.apply(dataclasses.replace(DEFAULT_SETTINGS, extra_environment={"A": "1"}))
    assert vfox.availability == VfoxAvailability.READY
    assert vfox.version == "1.0.13"
    assert vfox.executable == "/usr/local/bin/vfox"
    assert not vfox.is_custom_executable
    assert vfox.error is None
    assert dict(cli.launch_config.extra_environment) == {"A": "1"}
    assert [call.executable for call in runner.invocations] == ["/usr/local/bin/vfox"]


async def test_vfox_prefers_the_executable_chosen_by_the_user(runner: FakeProcessRunner) -> None:
    vfox, _ = _vfox_controller(runner, "/usr/local/bin/vfox")
    await vfox.apply(dataclasses.replace(DEFAULT_SETTINGS, vfox_executable="/opt/vfox"))
    assert vfox.executable == "/opt/vfox"
    assert vfox.is_custom_executable


async def test_vfox_reports_a_missing_and_a_broken_executable(runner: FakeProcessRunner) -> None:
    vfox, _ = _vfox_controller(runner)
    runner.missing_executable = True
    await vfox.apply(DEFAULT_SETTINGS)
    assert vfox.availability == VfoxAvailability.NOT_FOUND
    assert vfox.executable is None
    runner.missing_executable = False
    runner.on("--version", ProcessOutput(0, "hello", ""))
    await vfox.check()
    assert vfox.availability == VfoxAvailability.BROKEN
    assert isinstance(vfox.error, VfoxError)


async def test_vfox_reports_unexpected_failures_as_broken(runner: FakeProcessRunner) -> None:
    vfox, _ = _vfox_controller(runner)

    def explode() -> ProcessOutput:
        raise RuntimeError("unexpected")

    runner.on_dynamic("--version", explode)
    await vfox.check()
    assert vfox.availability == VfoxAvailability.BROKEN


async def test_sdk_detail_filters_available_versions_by_text_note_and_installation() -> None:
    controller = SdkDetailController(VfoxCli(fixture_runner()), "nodejs")
    await controller.versions.load()
    installed = {"24.11.1"}
    assert len(controller.visible_versions(installed)) == 872
    controller.version_filter = VersionFilter.WITH_NOTE
    assert all(version.note == "LTS" for version in controller.visible_versions(installed))
    controller.version_filter = VersionFilter.ALL
    controller.query = "24.11."
    assert controller.query == "24.11."
    assert [version.version for version in controller.visible_versions(installed)] == ["24.11.1", "24.11.0"]
    controller.query = "lts"
    assert len(controller.visible_versions(installed)) > 10
    controller.query = ""
    controller.version_filter = VersionFilter.INSTALLED
    assert controller.version_filter == VersionFilter.INSTALLED
    assert [version.version for version in controller.visible_versions(installed)] == ["24.11.1"]
    controller.dispose()


async def test_config_saves_and_reloads_and_exposes_values_by_key() -> None:
    runner = fixture_runner().on("config gitignore.enable false", ProcessOutput(0, "", ""))
    controller = VfoxConfigController(VfoxCli(runner))
    await controller.load()
    assert controller.value_for("gitignore.enable") == "true"
    assert controller.value_for("missing") is None
    seen: list[bool] = []
    controller.add_listener(lambda: seen.append(controller.is_saving("gitignore.enable")))
    await controller.set("gitignore.enable", "false")
    assert seen[0] is True
    assert not controller.is_saving("gitignore.enable")
    assert len(runner.calls("config --list")) == 2
    runner.on("config --unset proxy.url", ProcessOutput(0, "", ""))
    await controller.reset("proxy.url")
    assert len(runner.calls("config --unset proxy.url")) == 1


async def test_config_rethrows_rejected_values_after_reloading() -> None:
    runner = fixture_runner().on(
        "config cache.availableHookDuration x", fixture_output("config-set-invalid-duration", 1)
    )
    controller = VfoxConfigController(VfoxCli(runner))
    with pytest.raises(VfoxError):
        await controller.set("cache.availableHookDuration", "x")
    assert controller.has_value


@pytest.fixture
async def model(runner: FakeProcessRunner, tmp_path: Path) -> AsyncIterator[AppModel]:
    model = AppModel(
        runner,
        MemorySettingsStore(),
        {"HOME": str(tmp_path)},
        VfoxLocator({}, lambda path: path == "/usr/bin/vfox"),
    )
    yield model
    model.dispose()


async def test_model_starts_by_probing_vfox_and_loading_the_installed_sdks(model: AppModel) -> None:
    await model.start()
    await settle()
    assert model.vfox.availability == VfoxAvailability.READY
    assert [sdk.name for sdk in model.installed.value or ()] == ["maven", "mvn2", "nodejs"]
    installed = model.installed_sdk("nodejs")
    assert installed is not None
    assert installed.current == "24.11.1"
    assert model.installed_sdk("python") is None
    assert model.sdk_detail("nodejs").name == "nodejs"


async def test_model_reloads_the_installed_sdks_after_a_task(model: AppModel, runner: FakeProcessRunner) -> None:
    await model.start()
    await settle()
    before = len(runner.calls("list"))
    model.run(model.cli.use("nodejs", "22.21.0"))
    await settle()
    assert len(runner.calls("list")) == before + 1
    assert model.tasks.tasks[0].status == TaskStatus.SUCCEEDED


async def test_model_reloads_the_registry_and_the_configuration_when_vfox_becomes_available_again(
    model: AppModel, runner: FakeProcessRunner
) -> None:
    await model.start()
    await settle()
    await model.registry.load()
    await model.vfox_config.load()
    model.settings.replace(extra_environment={"X": "1"})
    await settle()
    assert len(runner.calls("available")) == 2
    assert len(runner.calls("config --list")) == 2
    assert runner.invocations[-1].environment["X"] == "1"


async def test_model_hands_vfox_a_copy_of_a_local_lua_plugin_and_deletes_it_afterwards(
    model: AppModel, runner: FakeProcessRunner, tmp_path: Path
) -> None:
    original = tmp_path / "mine.lua"
    original.write_text("PLUGIN = {}", encoding="utf-8")
    await model.start()
    await settle()
    task = model.add_plugin_from_source(str(original), "mine")
    args = task.command.arguments
    copy = args[args.index("--source") + 1]
    # vfox renames the file into its temporary directory, which only works on
    # the same filesystem.
    assert copy.startswith(f"{tmp_path}/.vfox/tmp/vfox-plus-plugin-")
    assert args[-2:] == ("--alias", "mine")
    await settle()
    assert task.status == TaskStatus.SUCCEEDED
    assert original.exists()
    assert not Path(copy).exists()


async def test_model_passes_urls_to_vfox_as_they_are(model: AppModel, runner: FakeProcessRunner) -> None:
    await model.start()
    model.add_plugin_from_source("https://example.com/p.zip", None)
    await settle()
    assert runner.calls("add")[0].arguments == ("add", "--source", "https://example.com/p.zip")


async def test_model_re_applies_the_settings_when_the_executable_changes(
    model: AppModel, runner: FakeProcessRunner
) -> None:
    await model.start()
    await settle()
    model.settings.replace(vfox_executable="/opt/vfox")
    await settle()
    assert model.vfox.executable == "/opt/vfox"
    assert runner.invocations[-1].executable == "/opt/vfox"


async def test_model_does_not_reload_after_a_task_that_never_ran(model: AppModel, runner: FakeProcessRunner) -> None:
    await model.start()
    await settle()
    before = len(runner.calls("list"))
    running = model.run(model.cli.update_plugin("a"))
    queued = model.run(model.cli.update_plugin("b"))
    model.tasks.cancel(queued)
    await settle()
    assert running.status == TaskStatus.SUCCEEDED
    assert len(runner.calls("list")) == before + 1


async def test_model_deletes_pending_plugin_copies_when_disposed(runner: FakeProcessRunner, tmp_path: Path) -> None:
    original = tmp_path / "mine.lua"
    original.write_text("PLUGIN = {}", encoding="utf-8")
    model = AppModel(runner, MemorySettingsStore(), {"HOME": str(tmp_path)}, VfoxLocator({}, lambda path: False))
    task = model.add_plugin_from_source(str(original), None)
    copy = task.command.arguments[2]
    assert Path(copy).exists()
    model.dispose()
    assert not Path(copy).exists()
    await settle()
