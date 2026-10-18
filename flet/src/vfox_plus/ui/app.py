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

"""The application: the page, its theme and language, and the root
component.
"""

import os
from collections.abc import Mapping

import flet as ft

from vfox_plus.core.i18n.localizations import Localizations, SupportedLanguage, resolve_language, system_locales
from vfox_plus.core.process import AsyncioProcessRunner, ProcessRunner
from vfox_plus.core.settings.app_settings import ThemePreference
from vfox_plus.core.settings.settings_store import JsonFileSettingsStore, SettingsStore, settings_path
from vfox_plus.state.app_model import AppModel
from vfox_plus.state.notifier import spawn
from vfox_plus.state.task_queue import TaskStatus, VfoxTask
from vfox_plus.ui.context import AppContext, AppScope, Destination, Messenger, ShellController
from vfox_plus.ui.hooks import use_listenable
from vfox_plus.ui.presentation import task_title
from vfox_plus.ui.resources import catalogue
from vfox_plus.ui.services import FletPlatformServices, PlatformServices
from vfox_plus.ui.shell import AppShell
from vfox_plus.ui.theme import build_theme

MINIMUM_SIZE = (360, 480)
"""The smallest window, in logical pixels (see `docs/ui-spec.md`)."""

# The default size of 1280 x 820 (`docs/ui-spec.md`) is set by
# `tools/patch_build_template.py` in the cached Flet build template, before
# the first frame: resizing the window at run time crashes the renderer from
# time to time on an X server without a window manager.


class Application:
    """The state of a running application and its bridge to the page: the
    theme, the language and the announcements of finished tasks follow the
    settings and the state.
    """

    def __init__(
        self,
        page: ft.Page,
        model: AppModel,
        services: PlatformServices,
        environment: Mapping[str, str],
    ) -> None:
        self.page = page
        self.model = model
        self.services = services
        self.shell = ShellController()
        self.messenger = Messenger(page)
        self._locales = system_locales(environment)
        self._accent: int | None = None
        self._applied: tuple[object, ...] | None = None
        model.settings.add_listener(self.apply_appearance)
        model.tasks.on_finished(self._announce)

    @property
    def language(self) -> SupportedLanguage:
        """The language of the user interface."""
        return resolve_language(self.model.settings.settings.language, self._locales)

    def localizations(self) -> Localizations:
        """The messages of the current language."""
        language = self.language
        return Localizations(language, catalogue(language), catalogue("en"))

    def context(self) -> AppContext:
        """What the components can reach, for the current language."""
        return AppContext(self.model, self.services, self.shell, self.messenger, self.localizations())

    async def load_accent_color(self) -> None:
        """Reads the accent colour of the desktop and applies it."""
        self._accent = await self.services.accent_color()
        self.apply_appearance()

    def apply_appearance(self) -> None:
        """Applies the colours, the brightness and the language of the
        settings to the page, if they changed.
        """
        settings = self.model.settings.settings
        seed = self._accent if settings.use_system_color and self._accent is not None else settings.seed_color
        language = self.language
        appearance = (seed, settings.theme, language)
        if appearance == self._applied:
            return
        self._applied = appearance
        theme = build_theme(seed)
        self.page.theme = theme
        self.page.dark_theme = theme
        self.page.theme_mode = {
            ThemePreference.SYSTEM: ft.ThemeMode.SYSTEM,
            ThemePreference.LIGHT: ft.ThemeMode.LIGHT,
            ThemePreference.DARK: ft.ThemeMode.DARK,
        }[settings.theme]
        # The locale picks the Chinese glyphs of the fonts and the language of
        # Flutter's own texts, such as tooltips of text fields.
        self.page.locale_configuration = ft.LocaleConfiguration(
            supported_locales=[ft.Locale("en"), ft.Locale("zh")], current_locale=ft.Locale(language)
        )
        self.page.title = self.localizations().t("appTitle")
        self.page.update()

    def _announce(self, task: VfoxTask) -> None:
        """Tells the user how a task ended, unless they are watching the task
        list.
        """
        if task.status == TaskStatus.CANCELLED:
            return
        failed = task.status == TaskStatus.FAILED
        if not failed and self.shell.destination == Destination.TASKS:
            return
        l10n = self.localizations()
        title = task_title(l10n, task.command)
        if failed:
            message = f"{l10n.t('taskFailed', title=title)}\n{task.error_message or ''}".strip()
            # Failures stay until dismissed, so that they can be read in full.
            self.messenger.snack(
                message, persist=True, action=l10n.t("taskView"), on_action=lambda: self.shell.go(Destination.TASKS)
            )
        else:
            self.messenger.snack(l10n.t("taskSucceeded", title=title))


@ft.component
def App(application: Application) -> ft.Control:
    """The root component: provides the context to the shell."""
    use_listenable(application.model.settings)
    return AppScope(application.context(), AppShell)


def configure_page(page: ft.Page) -> None:
    """The window: its size limits, its first size and its padding."""
    page.padding = 0
    page.spacing = 0
    page.window.min_width, page.window.min_height = MINIMUM_SIZE


def create_application(
    page: ft.Page,
    *,
    environment: Mapping[str, str] | None = None,
    runner: ProcessRunner | None = None,
    settings_store: SettingsStore | None = None,
    services: PlatformServices | None = None,
) -> Application:
    """Creates the application on `page`; tests replace its dependencies."""
    environment = dict(os.environ) if environment is None else environment
    model = AppModel(
        runner or AsyncioProcessRunner(),
        settings_store or JsonFileSettingsStore(settings_path(environment)),
        environment,
    )
    return Application(page, model, services or FletPlatformServices(), environment)


async def start(application: Application) -> None:
    """Shows the application and starts it: loads the settings, probes vfox
    and reads the accent colour.
    """
    page = application.page
    configure_page(page)
    application.model.settings.load()
    application.apply_appearance()
    page.render(App, application)
    spawn(application.load_accent_color())
    await application.model.start()


async def main(page: ft.Page) -> None:
    """The entry point that Flet calls with the page of the window."""
    application = create_application(page)
    await start(application)
    scenario = os.environ.get("VFOX_PLUS_E2E_SCENARIO")
    if scenario:
        from vfox_plus.e2e.runner import run_scenario  # noqa: PLC0415 - only for the end-to-end tests.

        spawn(run_scenario(application, scenario))
