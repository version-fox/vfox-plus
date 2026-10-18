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

"""The main flows of the user interface, run in the test process against
the scripted vfox of the fixtures."""

import flet as ft

from support import fixture_runner
from ui.harness import launch
from vfox_plus.core.settings.app_settings import ThemePreference
from vfox_plus.e2e.driver import Driver
from vfox_plus_m3e import ExpressiveNavigationRail


async def navigate(driver: Driver, index: int) -> None:
    """Chooses a destination of the navigation rail."""
    rail = next(control for control in driver.walk() if isinstance(control, ExpressiveNavigationRail))
    await driver.send(rail, "change", index)


async def test_home_shows_the_overview() -> None:
    running = await launch(fixture_runner())
    driver = running.driver
    await driver.wait_for_text("Versions in use")
    assert driver.has_text("maven")
    assert driver.has_text("3.9.16")
    await driver.tap_text("nodejs")
    await driver.wait_for_text("Node.js runtime environment.")


async def test_every_destination_is_reachable() -> None:
    running = await launch(fixture_runner())
    driver = running.driver
    await driver.wait_for_text("Versions in use")
    await navigate(driver, 1)
    await driver.wait_for_text("Select an SDK to manage its versions.")
    await navigate(driver, 2)
    await driver.wait_for_text("Every plugin adds", exact=False)
    await navigate(driver, 3)
    await driver.wait_for_text("Nothing is running")
    await navigate(driver, 4)
    await driver.wait_for_text("Plugin registry")
    await navigate(driver, 0)
    await driver.wait_for_text("Versions in use")


async def test_custom_plugin_dialog_opens_and_cancels() -> None:
    running = await launch(fixture_runner())
    driver = running.driver
    await driver.wait_for_text("Versions in use")
    await navigate(driver, 2)
    await driver.tap_text("Add from source…")
    await driver.wait_for_text("Add a plugin from source")
    assert isinstance(await driver.wait_for_key("custom-source"), ft.TextField)
    await driver.tap_key("dialog-cancel")
    await driver.wait_until_gone("Add a plugin from source")


async def test_open_source_licenses_lists_the_bundled_code() -> None:
    running = await launch(fixture_runner())
    driver = running.driver
    await driver.wait_for_text("Versions in use")
    await navigate(driver, 4)
    await driver.wait_for_text("Open-source licenses")
    await driver.tap_text("Open-source licenses")
    await driver.wait_for_text("Apache License", exact=False)
    await driver.tap_key("licenses-close")
    await driver.wait_until_gone("Apache License")


async def test_settings_switches_the_theme() -> None:
    running = await launch(fixture_runner())
    driver = running.driver
    await driver.wait_for_text("Versions in use")
    await navigate(driver, 4)
    await driver.tap_key("theme-dark")
    assert running.application.model.settings.settings.theme == ThemePreference.DARK
    await driver.tap_key("theme-light")
    assert running.application.model.settings.settings.theme == ThemePreference.LIGHT
