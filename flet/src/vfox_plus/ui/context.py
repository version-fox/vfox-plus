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

"""What every component of the user interface can reach: the application
state, the desktop integration, the navigation and the translations.
"""

from collections.abc import Callable
from dataclasses import dataclass
from enum import StrEnum
from typing import cast

import flet as ft

from vfox_plus.core.i18n.localizations import Localizations
from vfox_plus.state.app_model import AppModel
from vfox_plus.state.notifier import ChangeNotifier
from vfox_plus.ui.services import PlatformServices


class Destination(StrEnum):
    """The top-level destinations of the shell, in the order of the
    navigation.
    """

    HOME = "home"
    SDKS = "sdks"
    PLUGINS = "plugins"
    TASKS = "tasks"
    SETTINGS = "settings"


class ShellController(ChangeNotifier):
    """Which destination is shown, and which SDK is selected on the SDKs
    page.
    """

    def __init__(self) -> None:
        super().__init__()
        self.destination = Destination.HOME
        self.selected_sdk: str | None = None
        """The SDK shown on the SDKs page, if any."""

    def go(self, destination: Destination) -> None:
        """Shows `destination`."""
        if destination != self.destination:
            self.destination = destination
            self.notify_listeners()

    def show_sdk(self, name: str | None) -> None:
        """Shows the SDKs page with `name` selected, or with no selection."""
        self.destination = Destination.SDKS
        self.selected_sdk = name
        self.notify_listeners()


class Messenger:
    """Shows snack bars and dialogs on the page."""

    def __init__(self, page: ft.Page) -> None:
        self._page = page

    def snack(
        self,
        text: str,
        *,
        persist: bool = False,
        action: str | None = None,
        on_action: Callable[[], object] | None = None,
    ) -> None:
        """Shows `text` in a snack bar. A persistent one stays until it is
        closed.
        """
        self._page.show_dialog(
            ft.SnackBar(
                content=ft.Text(text),
                duration=4000,
                persist=persist,
                show_close_icon=persist,
                action=action,
                on_action=(lambda _: on_action()) if on_action is not None else None,
            )
        )

    def show(self, dialog: ft.DialogControl) -> None:
        """Opens `dialog`."""
        self._page.show_dialog(dialog)

    def close(self) -> None:
        """Closes the dialog on top."""
        self._page.pop_dialog()


@dataclass(frozen=True)
class AppContext:
    """What every component can reach."""

    model: AppModel
    services: PlatformServices
    shell: ShellController
    messenger: Messenger
    l10n: Localizations


AppScope = ft.create_context(cast("AppContext", None))
"""Provides the :class:`AppContext` to the components below it."""


def use_app() -> AppContext:
    """The :class:`AppContext` of the calling component."""
    return ft.use_context(AppScope)
