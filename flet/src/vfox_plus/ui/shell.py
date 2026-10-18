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

"""The adaptive frame around the pages.

Compact windows get a navigation bar at the bottom; wider windows get the
expressive navigation rail, which the user can expand to show labels.
Switching pages uses the fade-through transition (see `docs/ui-spec.md`).
"""

import dataclasses

import flet as ft

from vfox_plus.state.controllers import VfoxAvailability
from vfox_plus.ui.context import Destination, use_app
from vfox_plus.ui.hooks import use_listenable, use_width
from vfox_plus.ui.icons import icon
from vfox_plus.ui.pages.home import HomePage
from vfox_plus.ui.pages.plugins import PluginsPage
from vfox_plus.ui.pages.sdks import SdksPage
from vfox_plus.ui.pages.settings import SettingsPage
from vfox_plus.ui.pages.tasks import TasksPage
from vfox_plus.ui.pages.unavailable import UnavailablePage
from vfox_plus.ui.tokens import Gaps, WindowSizeClass
from vfox_plus_m3e import ExpressiveNavigationRail, RailDestination, TransitionSwitcher

_ICONS = {
    Destination.HOME: "home",
    Destination.SDKS: "sdks",
    Destination.PLUGINS: "plugins",
    Destination.TASKS: "tasks",
    Destination.SETTINGS: "settings",
}
_LABELS = {
    Destination.HOME: "navHome",
    Destination.SDKS: "navSdks",
    Destination.PLUGINS: "navPlugins",
    Destination.TASKS: "navTasks",
    Destination.SETTINGS: "navSettings",
}


def _page(destination: Destination, ready: bool) -> ft.Control:
    # Settings stay reachable so that the user can point to the executable.
    if not ready and destination != Destination.SETTINGS:
        return UnavailablePage()
    match destination:
        case Destination.HOME:
            return HomePage()
        case Destination.SDKS:
            return SdksPage()
        case Destination.PLUGINS:
            return PluginsPage()
        case Destination.TASKS:
            return TasksPage()
        case Destination.SETTINGS:
            return SettingsPage()


@ft.component
def AppShell() -> ft.Control:
    """The navigation and the page of the current destination."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(app.shell, model.vfox, model.tasks, model.settings)
    width, on_size_change = use_width()
    destination = app.shell.destination
    ready = model.vfox.availability == VfoxAvailability.READY
    active = model.tasks.active_count
    page = TransitionSwitcher(
        content_key=f"{destination.value}:{ready or destination == Destination.SETTINGS}",
        content=_page(destination, ready),
        expand=True,
    )
    destinations = list(Destination)

    def go(index: int) -> None:
        app.shell.go(destinations[index])

    if width is not None and WindowSizeClass.of(width) == WindowSizeClass.COMPACT:
        body: ft.Control = ft.Column(
            [
                ft.Container(content=page, expand=True),
                ft.NavigationBar(
                    selected_index=destination_index(destination),
                    on_change=lambda event: go(int(event.control.selected_index)),
                    destinations=[
                        ft.NavigationBarDestination(
                            icon=_badged(
                                ft.Icon(icon(_ICONS[each]), fill=0), active if each == Destination.TASKS else 0
                            ),
                            selected_icon=_badged(
                                ft.Icon(icon(_ICONS[each]), fill=1), active if each == Destination.TASKS else 0
                            ),
                            label=l10n.t(_LABELS[each]),
                        )
                        for each in destinations
                    ],
                ),
            ],
            spacing=0,
            expand=True,
        )
    else:
        settings = model.settings.settings
        body = ft.Row(
            [
                ExpressiveNavigationRail(
                    destinations=[
                        RailDestination(
                            icon=icon(_ICONS[each]),
                            label=l10n.t(_LABELS[each]),
                            badge_count=active if each == Destination.TASKS else 0,
                        )
                        for each in destinations
                    ],
                    selected_index=destination_index(destination),
                    expanded=settings.rail_expanded,
                    expand_icon=icon("menu"),
                    collapse_icon=icon("menuOpen"),
                    expand_tooltip=l10n.t("navExpand"),
                    collapse_tooltip=l10n.t("navCollapse"),
                    fab_icon=icon("add"),
                    fab_label=l10n.t("navAddPlugin"),
                    fab_disabled=not ready,
                    on_change=lambda event: go(int(event.data)),
                    on_toggle=lambda event: model.settings.update(
                        lambda current: dataclasses.replace(current, rail_expanded=bool(event.data))
                    ),
                    on_fab_click=lambda _: app.shell.go(Destination.PLUGINS),
                ),
                ft.Container(
                    content=page,
                    padding=ft.Padding.only(top=Gaps.L, right=Gaps.L, bottom=Gaps.L),
                    expand=True,
                ),
            ],
            spacing=0,
            vertical_alignment=ft.CrossAxisAlignment.STRETCH,
            expand=True,
        )
    return ft.Container(
        content=body,
        bgcolor=ft.Colors.SURFACE,
        expand=True,
        on_size_change=on_size_change,
        size_change_interval=100,
    )


def destination_index(destination: Destination) -> int:
    """The index of `destination` in the navigation."""
    return list(Destination).index(destination)


def _badged(control: ft.Icon, count: int) -> ft.Icon:
    if count > 0:
        control.badge = ft.Badge(label=str(count))
    return control
