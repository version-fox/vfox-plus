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

"""The dialogs of the application, which scale in on a spring, as
`flutter/lib/src/ui/widgets/dialogs.dart`. Each returns the answer of the
user once the dialog is closed.

The dialogs are created imperatively when they are shown, so their controls
are never frozen: the handlers change the fields in place. Titles are
`ft.Text`, so that the end-to-end driver sees them; a raw string would only
become text on the client.
"""

import asyncio
import os
from dataclasses import dataclass

import flet as ft

from vfox_plus.core.vfox.cli import is_valid_plugin_name
from vfox_plus.state.notifier import spawn
from vfox_plus.ui.context import AppContext
from vfox_plus.ui.icons import icon
from vfox_plus.ui.tokens import Gaps
from vfox_plus.ui.widgets.parts import filled_button, icon_button, text_button, text_field
from vfox_plus_m3e import ConstrainedBox, ExpressiveDialog

C = ft.Colors


class _Answer[T]:
    """The answer of a dialog: the first one wins, dismissing gives
    `fallback`.
    """

    def __init__(self, app: AppContext, fallback: T) -> None:
        self._app = app
        self._fallback = fallback
        self.future: asyncio.Future[T] = asyncio.get_running_loop().create_future()

    def give(self, value: T) -> None:
        if not self.future.done():
            self.future.set_result(value)
        self._app.messenger.close()

    def dismissed(self, _: object = None) -> None:
        if not self.future.done():
            self.future.set_result(self._fallback)


async def confirm(
    app: AppContext,
    *,
    title: str,
    body: str,
    confirm_label: str,
    meaning: str | None = None,
    destructive: bool = False,
) -> bool:
    """Asks the user to confirm an action. Returns whether they did."""
    answer = _Answer(app, False)
    dialog = ExpressiveDialog(
        icon=icon(meaning) if meaning is not None else None,
        icon_color=C.ERROR if destructive else C.SECONDARY,
        title=ft.Text(title),
        content=ft.Text(body),
        actions=[
            text_button(app.l10n.t("actionCancel"), lambda: answer.give(False), key="dialog-cancel"),
            filled_button(confirm_label, lambda: answer.give(True), destructive=destructive, key="dialog-confirm"),
        ],
        on_dismiss=answer.dismissed,
    )
    app.messenger.show(dialog)
    return await answer.future


async def ask_project_directory(app: AppContext, *, name: str, version: str) -> str | None:
    """Asks for the project directory in which to pin `name` `version`."""
    l10n = app.l10n
    answer: _Answer[str | None] = _Answer(app, None)
    field = text_field(l10n.t("useProjectDirectory"), autofocus=True, expand=True, key="project-directory")
    dialog: ExpressiveDialog

    def submit() -> None:
        path = (field.value or "").strip()
        if not path or not os.path.isdir(path):
            field.error = l10n.t("useProjectInvalid")
            field.update()
            return
        answer.give(path)

    async def browse() -> None:
        directory = await app.services.pick_directory(field.value or os.environ.get("HOME"))
        if directory is not None:
            field.value = directory
            field.error = None
            field.update()

    field.on_submit = lambda _: submit()
    dialog = ExpressiveDialog(
        icon=icon("projectScope"),
        icon_color=C.SECONDARY,
        title=ft.Text(l10n.t("useProjectTitle", name=name, version=version)),
        content=ConstrainedBox(
            max_width=480,
            content=ft.Column(
                [
                    ft.Text(l10n.t("useProjectBody")),
                    ft.Container(height=Gaps.S),
                    ft.Row(
                        [
                            field,
                            ft.Container(
                                content=icon_button(
                                    "folder", l10n.t("actionBrowse"), lambda: spawn(browse()), filled_tonal=True
                                ),
                                padding=ft.Padding.only(top=Gaps.XS),
                            ),
                        ],
                        spacing=Gaps.S,
                        vertical_alignment=ft.CrossAxisAlignment.START,
                    ),
                ],
                tight=True,
                spacing=Gaps.S,
            ),
        ),
        actions=[
            text_button(l10n.t("actionCancel"), lambda: answer.give(None), key="dialog-cancel"),
            filled_button(l10n.t("useProjectConfirm"), submit, key="dialog-confirm"),
        ],
        on_dismiss=answer.dismissed,
    )
    app.messenger.show(dialog)
    return await answer.future


@dataclass(frozen=True)
class CustomPluginSource:
    """The source and optional alias of a plugin added from a URL or a path."""

    source: str
    alias: str | None


async def ask_custom_plugin(app: AppContext) -> CustomPluginSource | None:
    """Asks for the source of a plugin that is not in the registry."""
    l10n = app.l10n
    answer: _Answer[CustomPluginSource | None] = _Answer(app, None)
    source = text_field(
        l10n.t("pluginsCustomSource"),
        helper=l10n.t("pluginsCustomSourceHelp"),
        autofocus=True,
        expand=True,
        key="custom-source",
    )
    alias = text_field(l10n.t("pluginsCustomAlias"), helper=l10n.t("pluginsCustomAliasHelp"), key="custom-alias")

    def submit() -> None:
        value = (source.value or "").strip()
        name = (alias.value or "").strip()
        source.error = None if value else l10n.t("pluginsCustomRequired")
        alias.error = l10n.t("pluginsCustomInvalidName") if name and not is_valid_plugin_name(name) else None
        if source.error is None and alias.error is None:
            answer.give(CustomPluginSource(value, name or None))
        else:
            source.update()
            alias.update()

    async def browse() -> None:
        # The portal names the filter of the extensions; Flet does not let
        # it be labelled with pluginsCustomFileType.
        path = await app.services.pick_file(os.environ.get("HOME"), ("zip", "lua"))
        if path is not None:
            source.value = path
            source.error = None
            source.update()

    source.on_submit = lambda _: submit()
    alias.on_submit = lambda _: submit()
    dialog = ExpressiveDialog(
        icon=icon("plugins"),
        icon_color=C.SECONDARY,
        title=ft.Text(l10n.t("pluginsCustomTitle")),
        content=ConstrainedBox(
            max_width=480,
            content=ft.Column(
                [
                    ft.Row(
                        [
                            source,
                            ft.Container(
                                content=icon_button(
                                    "folder", l10n.t("actionBrowse"), lambda: spawn(browse()), filled_tonal=True
                                ),
                                padding=ft.Padding.only(top=Gaps.XS),
                            ),
                        ],
                        spacing=Gaps.S,
                        vertical_alignment=ft.CrossAxisAlignment.START,
                    ),
                    alias,
                ],
                tight=True,
                spacing=Gaps.L,
            ),
        ),
        actions=[
            text_button(l10n.t("actionCancel"), lambda: answer.give(None), key="dialog-cancel"),
            filled_button(l10n.t("pluginsCustomConfirm"), submit, key="dialog-confirm"),
        ],
        on_dismiss=answer.dismissed,
    )
    app.messenger.show(dialog)
    return await answer.future
