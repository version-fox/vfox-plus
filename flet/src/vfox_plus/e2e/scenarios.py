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

"""The end-to-end scenarios, the same as those of the other implementations
(`web/src/e2e/scenarios.ts`, `flutter/integration_test`).
"""

import asyncio
import dataclasses
import os
import subprocess
from collections.abc import Awaitable, Callable, Mapping
from dataclasses import dataclass
from typing import TYPE_CHECKING

import flet as ft

from vfox_plus.core.settings.app_settings import LanguagePreference, ThemePreference
from vfox_plus.e2e.driver import Driver, DriverError, text_of, tooltip_of
from vfox_plus.state.task_queue import TaskStatus
from vfox_plus.ui.presentation import task_title
from vfox_plus_m3e import ExpressiveNavigationRail, RailDestination, SpringSurface

if TYPE_CHECKING:
    from vfox_plus.ui.app import Application


@dataclass(frozen=True)
class ScenarioContext:
    """What a scenario can use."""

    application: Application
    driver: Driver
    environment: Mapping[str, str]
    step: Callable[[str], None]
    """Reports progress on the output of the application."""


type Scenario = Callable[[ScenarioContext], Awaitable[None]]


def check(condition: bool, description: str) -> None:
    """Fails the scenario unless `condition` holds."""
    if not condition:
        raise DriverError(f"Check failed: {description}")


async def pause(milliseconds: int) -> None:
    """Waits for the user interface to settle, as a person would."""
    await asyncio.sleep(milliseconds / 1000)


async def navigate(context: ScenarioContext, label: str) -> None:
    """Chooses the destination `label` in the navigation rail or bar."""
    driver = context.driver

    async def choose(control: ft.BaseControl, send: str | int) -> None:
        await driver.send(control, "change", send)

    def find() -> tuple[ft.BaseControl, str | int] | None:
        for control in driver.walk():
            if isinstance(control, ExpressiveNavigationRail):
                for index, destination in enumerate(control.destinations):
                    if isinstance(destination, RailDestination) and destination.label == label:
                        return control, index
            if isinstance(control, ft.NavigationBar):
                for index, item in enumerate(control.destinations):
                    if item.label == label:
                        return control, str(index)
        return None

    found: list[tuple[ft.BaseControl, str | int]] = []

    def present() -> bool:
        match = find()
        if match is not None:
            found.append(match)
        return match is not None

    await driver.wait_until(present, f'the destination "{label}"')
    control, send = found[-1]
    if isinstance(control, ft.NavigationBar):
        driver.page.session.apply_patch(control._i, {"selected_index": int(send)})
    await choose(control, send)


async def click_button(context: ScenarioContext, name: str) -> None:
    """Clicks the button that shows `name` or whose tooltip is `name`."""
    driver = context.driver

    def matches(control: ft.BaseControl) -> bool:
        return (text_of(control) == name or tooltip_of(control) == name) and not getattr(control, "disabled", False)

    control = await driver.wait_for(matches, f'the button "{name}"')
    await driver.click(control)


async def click_item(context: ScenarioContext, title: str) -> None:
    """Clicks the list item whose title starts with `title`."""
    driver = context.driver

    def matches(control: ft.BaseControl) -> bool:
        if not isinstance(control, SpringSurface) or control.on_click is None:
            return False
        texts = [text for each in driver._subtree(control) if (text := text_of(each)) is not None]
        return bool(texts) and texts[0].startswith(title)

    await driver.click(await driver.wait_for(matches, f'the item "{title}"'))


async def click_button_near(context: ScenarioContext, row: str, name: str) -> None:
    """Clicks the button `name` in the list item that shows `row`."""
    driver = context.driver

    def find() -> ft.BaseControl | None:
        for control in driver.walk():
            if not isinstance(control, SpringSurface):
                continue
            subtree = list(driver._subtree(control))
            if not any(text_of(each) == row for each in subtree):
                continue
            for each in subtree:
                if (text_of(each) == name or tooltip_of(each) == name) and not getattr(each, "disabled", False):
                    return each
        return None

    found: list[ft.BaseControl] = []

    def present() -> bool:
        control = find()
        if control is not None:
            found.append(control)
        return control is not None

    await driver.wait_until(present, f'the button "{name}" next to "{row}"')
    await driver.click(found[-1])


async def fill(context: ScenarioContext, label: str, value: str, *, submit: bool = False) -> None:
    """Types `value` into the text field labelled or hinted `label`, or into
    the text field of the list item titled `label`.
    """
    driver = context.driver

    def field_of(control: ft.BaseControl) -> ft.BaseControl | None:
        if isinstance(control, ft.TextField) and label in (control.label, control.hint_text):
            return control
        if isinstance(control, SpringSurface):
            subtree = list(driver._subtree(control))
            texts = [text for each in subtree if (text := text_of(each)) is not None]
            if texts and texts[0] == label:
                return next((each for each in subtree if isinstance(each, ft.TextField)), None)
        return None

    found: list[ft.BaseControl] = []

    def present() -> bool:
        for control in driver.walk():
            field = field_of(control)
            if field is not None:
                found.append(field)
                return True
        return False

    await driver.wait_until(present, f'the field "{label}"')
    await driver.enter_text(found[-1], value, submit=submit)


async def choose_menu_item(context: ScenarioContext, menu_tooltip: str, item: str) -> None:
    """Opens the menu with `menu_tooltip` and chooses `item`."""
    await context.driver.choose_from_menu(menu_tooltip, item)


async def wait_for_task(context: ScenarioContext, title: str) -> None:
    """Waits until the task with `title` (in English) has succeeded; fails if
    it failed.
    """
    application = context.application
    l10n = application.localizations()

    def done() -> bool:
        task = next(
            (each for each in application.model.tasks.tasks if task_title(l10n, each.command) == title),
            None,
        )
        if task is not None and task.status == TaskStatus.FAILED:
            raise DriverError(f"{title} failed: {task.error_message or ''}")
        return task is not None and task.status == TaskStatus.SUCCEEDED

    await context.driver.wait_until(done, f'the task "{title}" to finish', timeout=60)


async def vfox(context: ScenarioContext, *args: str) -> str:
    """Runs vfox as the tests see it: in the sandbox, without colours."""
    executable = context.environment.get("VFOX_PLUS_E2E_VFOX", "vfox")
    process = await asyncio.create_subprocess_exec(
        executable,
        *args,
        cwd="/",
        env={**context.environment, "NO_COLOR": "1"},
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
    )
    stdout, _ = await process.communicate()
    return stdout.decode()


async def fake_scenario(context: ScenarioContext) -> None:
    """The application against the fake vfox of
    `shared/fixtures/fake-vfox.sh`, which replays the fixtures.
    """
    model = context.application.model
    driver = context.driver
    await driver.wait_for_text("Versions in use")
    context.step("home")
    await driver.wait_for_text(f"vfox 1.0.13 · {model.vfox.executable or ''}")
    check(
        model.vfox.executable == context.environment.get("VFOX_PLUS_E2E_VFOX"),
        "the executable from the settings is used",
    )

    await navigate(context, "SDKs")
    await click_item(context, "nodejs")
    await driver.wait_for_text("Node.js runtime environment.")
    context.step("detail")

    # Select 22.21.0 as the global default.
    await click_button(context, "Use")
    await wait_for_task(context, "Use nodejs@22.21.0")

    # Install a version from the available tab.
    await click_button(context, "Available")
    await driver.wait_for_text("872 versions")
    await fill(context, "Filter versions", "24.11.0")
    await driver.wait_for_text("1 version")
    await click_button_near(context, "24.11.0", "Install")
    await wait_for_task(context, "Install nodejs@24.11.0")

    tasks = model.tasks.tasks
    check(all(task.status == TaskStatus.SUCCEEDED for task in tasks), "every task succeeded")
    check(tasks[0].progress == 1, "the download progress reached 100 %")
    check("Install nodejs@24.11.0 success!" in tasks[0].log, "the output of vfox is kept")


async def real_scenario(context: ScenarioContext) -> None:
    """Every operation against the real vfox and the offline test plugin of
    `shared/test-plugin`.
    """
    model = context.application.model
    driver = context.driver
    environment = context.environment
    server = environment.get("VFOX_PLUS_TEST_SERVER", "")
    home = environment.get("HOME", "")
    local_plugin = environment.get("VFOX_PLUS_E2E_LOCAL_PLUGIN", "")

    async def control(action: str) -> None:
        process = await asyncio.create_subprocess_exec(
            "curl", "-fsS", f"{server}/control/{action}", stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
        )
        await process.wait()

    await driver.wait_for_text("Versions in use")
    check(model.installed.value == [], "no plugin is installed yet")

    # Point vfox to the local registry through the vfox configuration editor.
    context.step("configure the registry")
    await navigate(context, "Settings")
    await fill(context, "Plugin registry", f"{server}/registry", submit=True)
    await driver.wait_until(
        lambda: model.vfox_config.value_for("registry.address") == f"{server}/registry", "the registry to be saved"
    )
    check(
        (await vfox(context, "config", "registry.address")).strip() == f"{server}/registry", "vfox stored the registry"
    )

    # Add the plugin from the registry.
    context.step("add a plugin from the registry")
    await navigate(context, "Plugins")
    await driver.wait_for_text("testsdk")
    await click_button(context, "Add")
    await wait_for_task(context, "Add plugin testsdk")
    await driver.wait_for_text("Added")

    # Add a local copy of the plugin under another name. vfox moves local
    # .lua files away, so the application must hand it a copy.
    context.step("add a plugin from a local file")
    await click_button(context, "Add from source…")
    await driver.wait_for_text("Add a plugin from source")
    await fill(context, "Source", local_plugin)
    await fill(context, "Name (optional)", "localsdk")
    await click_button(context, "Add plugin")
    await wait_for_task(context, "Add plugin localsdk")
    check(os.path.exists(local_plugin), "the source file survives vfox add --source")

    # Search and install both versions.
    context.step("install")
    await navigate(context, "SDKs")
    await click_item(context, "testsdk")
    await driver.wait_for_text("An offline SDK for the vfox-plus end-to-end tests.")
    await click_button(context, "Available")
    await driver.wait_for_text("2 versions")
    await driver.wait_for_text("LTS")
    await driver.wait_for_text("testtool 2.0")
    check(driver.texts().count("LTS") == 1, "the note is shown once")
    await click_button_near(context, "2.0.0", "Install")
    await wait_for_task(context, "Install testsdk@2.0.0")
    check(model.tasks.tasks[0].progress == 1, "vfox reports the download progress on standard error")

    # Cancel the installation of 1.0.0 during the download. From 1.0.13 vfox
    # leaves the partial installation behind, but must neither list it nor
    # refuse to install the version again.
    context.step("cancel an installation")
    await control("hold")
    await click_button_near(context, "1.0.0", "Install")
    await driver.wait_until(lambda: (model.tasks.tasks[0].progress or 0) > 0, "the download to start")
    await click_button(context, "Cancel")
    await driver.wait_until(lambda: model.tasks.tasks[0].finished_at is not None, "vfox to exit")
    check(model.tasks.tasks[0].status == TaskStatus.CANCELLED, "the task is cancelled")
    check("1.0.0" not in await vfox(context, "list", "testsdk"), "vfox does not list the partial version")
    await control("release")
    await pause(500)
    await click_button_near(context, "1.0.0", "Install")
    await wait_for_task(context, "Install testsdk@1.0.0")

    # Make 2.0.0 the global default.
    context.step("use")
    await click_button(context, "Installed (2)")
    await driver.wait_for_text("1.0.0")
    await click_button_near(context, "2.0.0", "Use")
    await wait_for_task(context, "Use testsdk@2.0.0")
    await driver.wait_for_text("In use: 2.0.0")
    with open(f"{home}/.vfox/.vfox.toml", encoding="utf-8") as file:
        check('testsdk = "2.0.0"' in file.read(), "the global .vfox.toml pins 2.0.0")
    check(os.path.exists(f"{home}/.vfox/sdks/testsdk"), "the SDK is linked")

    # Uninstall 1.0.0.
    context.step("uninstall")
    await choose_menu_item(context, "More actions for 1.0.0", "Uninstall")
    await driver.wait_for_text("Uninstall testsdk 1.0.0?")
    await driver.tap_key("dialog-confirm")
    await wait_for_task(context, "Uninstall testsdk@1.0.0")
    await driver.wait_for_text("Installed (1)")

    # Remove the second plugin.
    context.step("remove")
    await click_item(context, "localsdk")
    await driver.wait_for_text("localsdk")
    await choose_menu_item(context, "More actions", "Remove plugin")
    await driver.wait_for_text("Remove localsdk?")
    await driver.tap_key("dialog-confirm")
    await wait_for_task(context, "Remove plugin localsdk")
    await driver.wait_for_text("Select an SDK to manage its versions.")
    await driver.wait_until(
        lambda: all(sdk.name != "localsdk" for sdk in model.installed.value or ()),
        "the removed plugin to disappear from the list",
    )

    # Every task but the cancelled one succeeded, and vfox agrees with what
    # the application shows.
    statuses = [task.status for task in model.tasks.tasks if task.status != TaskStatus.SUCCEEDED]
    check(statuses == [TaskStatus.CANCELLED], f"only the cancelled task did not succeed: {statuses}")
    check(await vfox(context, "current") == "testsdk -> v2.0.0\n", "vfox uses testsdk 2.0.0")
    check("2.0.0" in await vfox(context, "list", "testsdk"), "vfox lists testsdk 2.0.0")
    # The copy of the local plugin is deleted once its task has finished.
    leftovers = [name for name in os.listdir(f"{home}/.vfox/tmp") if name.startswith("vfox-plus-plugin-")]
    check(not leftovers, f"the copy of the local plugin is deleted: {leftovers}")


async def measure_scenario(context: ScenarioContext) -> None:
    """Opens the Home page and the versions of Node.js, so that the harness
    can measure the memory of the application at both points (see
    `docs/new-implementation.md`).
    """
    driver = context.driver
    await driver.wait_for_text("Versions in use")
    await pause(3000)
    context.step("MEASURE home")
    await pause(5000)
    await navigate(context, "SDKs")
    await click_item(context, "nodejs")
    await click_button(context, "Available")
    await driver.wait_for_text("872 versions")
    await pause(3000)
    context.step("MEASURE nodejs")
    await pause(5000)
    await navigate(context, "Home")


async def _shot(context: ScenarioContext, name: str) -> None:
    """Waits for the page to settle and asks the harness to capture the
    screen.
    """
    await pause(1500)
    context.step(f"SHOT {name}")
    await pause(1500)


async def screenshots_scenario(context: ScenarioContext) -> None:
    """Shows every page against the fake vfox; the harness captures the
    screen at every `e2e: SHOT <name>` (see `shared/tools/screenshots.sh`).
    With VFOX_PLUS_FAKE_DELAY, it shows an installation in progress
    instead.
    """
    application = context.application
    model = application.model
    driver = context.driver
    await driver.wait_for_text("Versions in use")
    if context.environment.get("VFOX_PLUS_FAKE_DELAY"):
        await navigate(context, "SDKs")
        await click_item(context, "nodejs")
        await click_button(context, "Available")
        await driver.wait_for_text("872 versions")
        await click_button_near(context, "26.12.0", "Install")
        await driver.wait_until(lambda: (model.tasks.tasks[0].progress or 0) > 0, "the download to start")
        await _shot(context, "installing")
        return
    await _shot(context, "home")
    await navigate(context, "SDKs")
    await click_item(context, "nodejs")
    await driver.wait_for_text("Node.js runtime environment.")
    await _shot(context, "sdks")
    await click_button(context, "Available")
    await driver.wait_for_text("872 versions")
    await _shot(context, "available")
    await fill(context, "Filter versions", "24.11.0")
    await driver.wait_for_text("1 version")
    await click_button_near(context, "24.11.0", "Install")
    await wait_for_task(context, "Install nodejs@24.11.0")
    await navigate(context, "Tasks")
    await click_button(context, "Show output")
    await _shot(context, "tasks")
    await navigate(context, "Plugins")
    await driver.wait_for_text("dotnet")
    await _shot(context, "plugins")
    await click_button(context, "Add from source…")
    await driver.wait_for_text("Add a plugin from source")
    await _shot(context, "dialog")
    await driver.tap_key("dialog-cancel")
    await navigate(context, "Settings")
    await driver.wait_for_text("Plugin registry")
    await _shot(context, "settings")
    model.settings.update(
        lambda settings: dataclasses.replace(settings, theme=ThemePreference.DARK, rail_expanded=True)
    )
    await navigate(context, "Home")
    await _shot(context, "dark-expanded")
    await navigate(context, "SDKs")
    await click_item(context, "maven")
    await driver.wait_for_text("3.8.9")
    await _shot(context, "dark-sdks")
    model.settings.update(
        lambda settings: dataclasses.replace(
            settings,
            theme=ThemePreference.LIGHT,
            rail_expanded=False,
            language=LanguagePreference.SIMPLIFIED_CHINESE,
        )
    )
    await navigate(context, "首页")
    await driver.wait_for_text("正在使用的版本")
    await _shot(context, "zh-home")
    await navigate(context, "设置")
    await driver.wait_for_text("插件仓库")
    await _shot(context, "zh-settings")
    model.settings.update(lambda settings: dataclasses.replace(settings, language=LanguagePreference.ENGLISH))
    application.page.window.width, application.page.window.height = 412, 800
    application.page.update()
    await navigate(context, "Home")
    await driver.wait_for_text("Versions in use")
    await _shot(context, "compact-home")
    # Maven is still selected on the SDKs page, which compact windows would
    # show; the versions in use on the Home page lead to their SDK instead.
    await click_item(context, "nodejs")
    await driver.wait_for_text("Node.js runtime environment.")
    await _shot(context, "compact-detail")


SCENARIOS: dict[str, Scenario] = {
    "fake": fake_scenario,
    "real": real_scenario,
    "measure": measure_scenario,
    "screenshots": screenshots_scenario,
}
"""The scenarios by name."""
