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

"""The plugin registry: every plugin that `vfox add` can install."""

import math
from enum import StrEnum

import flet as ft

from vfox_plus.core.models import AvailablePlugin
from vfox_plus.core.vfox.command import VfoxOperation
from vfox_plus.state.notifier import spawn
from vfox_plus.state.task_queue import VfoxTask
from vfox_plus.ui.context import AppContext, use_app
from vfox_plus.ui.hooks import use_listenable, use_width
from vfox_plus.ui.icons import icon, icon_control
from vfox_plus.ui.presentation import repository_path
from vfox_plus.ui.tokens import Corners, Gaps
from vfox_plus.ui.widgets.buttons import Segment, button_group
from vfox_plus.ui.widgets.dialogs import ask_custom_plugin
from vfox_plus.ui.widgets.parts import (
    error_message,
    icon_button,
    loading_message,
    page_header,
    pane,
    search_bar,
    status_message,
    text,
    tonal_button,
)
from vfox_plus_m3e import ConstrainedBox, LoadingIndicator, ShapeAvatar

C = ft.Colors
S = ft.TextThemeStyle

_CARD_WIDTH = 340
"""The widest a card gets before the grid adds a column."""
_CARD_HEIGHT = 152


class _Filter(StrEnum):
    ALL = "all"
    OFFICIAL = "official"
    COMMUNITY = "community"


@ft.component
def PluginsPage() -> ft.Control:
    """The Plugins page."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(model.registry, model.installed, model.tasks)
    query, set_query = ft.use_state("")
    shown, set_shown = ft.use_state(_Filter.ALL)
    width, on_size_change = use_width()

    def load() -> None:
        spawn(model.registry.ensure_loaded())

    ft.use_effect(load, [])

    plugins = model.registry.value
    installed = {sdk.name for sdk in model.installed.value or ()}
    needle = query.strip().lower()
    visible = [
        plugin
        for plugin in plugins or ()
        if (shown == _Filter.ALL or (plugin.official if shown == _Filter.OFFICIAL else not plugin.official))
        and (not needle or needle in plugin.name.lower() or needle in plugin.homepage)
    ]
    updating_all = model.tasks.active_task(VfoxOperation.UPDATE_ALL_PLUGINS)

    async def add_custom() -> None:
        source = await ask_custom_plugin(app)
        if source is not None:
            model.add_plugin_from_source(source.source, source.alias)

    header = page_header(
        l10n.t("pluginsTitle"),
        subtitle=l10n.t("pluginsSubtitle"),
        actions=[
            LoadingIndicator(size=40)
            if updating_all is not None
            else icon_button(
                "update",
                l10n.t("pluginsUpdateAll"),
                (lambda: model.run(model.cli.update_all_plugins())) if installed else None,
            ),
            icon_button("refresh", l10n.t("actionRefresh"), None if model.registry.is_loading else model.registry.load),
            tonal_button(l10n.t("pluginsAddCustom"), lambda: spawn(add_custom()), meaning="add"),
        ],
    )
    toolbar = ft.Container(
        padding=ft.Padding.only(left=Gaps.XL, right=Gaps.XL, bottom=Gaps.L),
        content=ft.Row(
            [
                ConstrainedBox(
                    max_width=420,
                    content=search_bar(
                        l10n.t("pluginsSearchHint"), query, set_query, clear_tooltip=l10n.t("actionClose")
                    ),
                ),
                button_group(
                    [
                        Segment(_Filter.ALL, l10n.t("pluginsFilterAll"), key="plugins-all"),
                        Segment(_Filter.OFFICIAL, l10n.t("pluginsFilterOfficial"), "verified", key="plugins-official"),
                        Segment(_Filter.COMMUNITY, l10n.t("pluginsFilterCommunity"), key="plugins-community"),
                    ],
                    shown,
                    set_shown,
                ),
            ],
            spacing=Gaps.M,
            run_spacing=Gaps.M,
            wrap=True,
            vertical_alignment=ft.CrossAxisAlignment.CENTER,
        ),
    )
    if plugins is None:
        body: list[ft.Control] = [
            ft.Container(
                height=360,
                content=error_message(
                    l10n.t("pluginsLoadError"), model.registry.error, l10n.t("actionRetry"), model.registry.load
                )
                if model.registry.error is not None
                else loading_message(semantics_label=l10n.t("loading")),
            )
        ]
    elif not visible:
        body = [ft.Container(height=360, content=status_message("notFound", l10n.t("pluginsNoMatch")))]
    else:
        body = [
            ft.Container(
                padding=ft.Padding.only(left=Gaps.XL, right=Gaps.XL, bottom=Gaps.XL),
                content=_grid(app, visible, installed, (width or 1000) - 2 * Gaps.XL),
            )
        ]
    return pane(
        ft.Container(
            content=ft.Column([header, toolbar, *body], spacing=0, scroll=ft.ScrollMode.AUTO, expand=True),
            expand=True,
            on_size_change=on_size_change,
            size_change_interval=100,
        )
    )


def _grid(app: AppContext, plugins: list[AvailablePlugin], installed: set[str], width: float) -> ft.Column:
    """The cards in as many columns as fit, each at most 340 dp wide, as
    Flutter's grid with a maximum cross-axis extent.
    """
    columns = max(1, math.ceil((width + Gaps.M) / (_CARD_WIDTH + Gaps.M)))
    rows: list[ft.Control] = []
    for start in range(0, len(plugins), columns):
        cells: list[ft.Control] = [
            _card(
                app,
                plugin,
                plugin.name in installed,
                app.model.tasks.active_task(VfoxOperation.ADD_PLUGIN, plugin.name),
            )
            for plugin in plugins[start : start + columns]
        ]
        cells += [ft.Container(expand=True) for _ in range(columns - len(cells))]
        rows.append(
            ft.Row(cells, spacing=Gaps.M, height=_CARD_HEIGHT, vertical_alignment=ft.CrossAxisAlignment.STRETCH)
        )
    return ft.Column(rows, spacing=Gaps.M)


def _card(app: AppContext, plugin: AvailablePlugin, installed: bool, task: VfoxTask | None) -> ft.Card:
    model = app.model
    l10n = app.l10n
    if task is not None:
        action: ft.Control = LoadingIndicator(size=40)
    elif installed:
        action = ft.TextButton(
            content=l10n.t("pluginsAdded"),
            icon=icon("check"),
            on_click=lambda _: app.shell.show_sdk(plugin.name),
            key=f"added-{plugin.name}",
        )
    else:
        action = tonal_button(
            l10n.t("pluginsAdd"),
            lambda: model.run(model.cli.add_plugin(plugin.name)),
            meaning="add",
            key=f"add-{plugin.name}",
        )
    name: list[ft.Control] = [
        ft.Container(
            content=text(plugin.name, S.TITLE_MEDIUM, emphasized=True, max_lines=1), expand_loose=True, expand=True
        )
    ]
    if plugin.official:
        name.append(
            ft.Container(
                content=icon_control("verified", size=18, filled=True, color=C.PRIMARY),
                tooltip=l10n.t("pluginsOfficial"),
            )
        )
    homepage = plugin.homepage
    return ft.Card(
        bgcolor=C.SURFACE_CONTAINER_HIGHEST if installed else C.SURFACE_CONTAINER,
        shape=ft.RoundedRectangleBorder(radius=Corners.LARGE_INCREASED),
        expand=True,
        content=ft.Container(
            padding=Gaps.L,
            content=ft.Column(
                [
                    ft.Row(
                        [
                            ShapeAvatar(name=plugin.name, size=44),
                            ft.Column(
                                [
                                    ft.Row(name, spacing=Gaps.XS),
                                    text(
                                        repository_path(homepage), S.BODY_SMALL, color=C.ON_SURFACE_VARIANT, max_lines=1
                                    ),
                                ],
                                spacing=0,
                                expand=True,
                            ),
                        ],
                        spacing=Gaps.M,
                    ),
                    ft.Container(expand=True),
                    ft.Row(
                        [
                            icon_button(
                                "openInNew",
                                l10n.t("actionOpenHomepage"),
                                (lambda: spawn(app.services.open_url(homepage))) if homepage else None,
                                compact=True,
                            ),
                            ft.Container(expand=True),
                            action,
                        ]
                    ),
                ],
                spacing=0,
            ),
        ),
    )
