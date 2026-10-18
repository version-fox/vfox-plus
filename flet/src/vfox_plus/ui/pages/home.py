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

"""The overview: what is installed, what is in use and where to go next."""

from collections.abc import Callable

import flet as ft

from vfox_plus.core.models import InstalledSdk
from vfox_plus.state.notifier import spawn
from vfox_plus.ui import links
from vfox_plus.ui.context import AppContext, Destination, use_app
from vfox_plus.ui.hooks import use_listenable, use_width
from vfox_plus.ui.icons import icon_control
from vfox_plus.ui.presentation import ACTIVATION_COMMAND
from vfox_plus.ui.tokens import Corners, Gaps
from vfox_plus.ui.widgets.lists import ListPosition, list_group, list_item
from vfox_plus.ui.widgets.parts import (
    content_width,
    error_message,
    group_header,
    icon_button,
    loading_message,
    page_header,
    pane,
    text,
)
from vfox_plus_m3e import MaterialShape, ShapeAvatar

C = ft.Colors
S = ft.TextThemeStyle


@ft.component
def HomePage() -> ft.Control:
    """The Home page."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(model.installed, model.vfox)
    width, on_size_change = use_width()
    sdks = model.installed.value
    if sdks is None:
        if model.installed.error is not None:
            return pane(
                error_message(
                    l10n.t("sdksLoadError"), model.installed.error, l10n.t("actionRetry"), model.installed.load
                )
            )
        return pane(loading_message(semantics_label=l10n.t("loading")))
    active = [sdk for sdk in sdks if sdk.current is not None]
    version_count = sum(len(sdk.versions) for sdk in sdks)
    content = (width or 1080) - 2 * Gaps.XL
    subtitle = " · ".join(
        [
            l10n.t("homeSubtitle", version=model.vfox.version or "?"),
            *([model.vfox.executable] if model.vfox.executable else []),
        ]
    )

    def active_item(sdk: InstalledSdk) -> Callable[[ListPosition], ft.Control]:
        return lambda position: list_item(
            sdk.name,
            position=position,
            leading=ShapeAvatar(name=sdk.name),
            subtitle=sdk.current,
            trailing=icon_control("chevronRight", color=C.ON_SURFACE_VARIANT),
            on_click=lambda: app.shell.show_sdk(sdk.name),
            key=f"home-active-{sdk.name}",
        )

    active_list = (
        list_group([active_item(sdk) for sdk in active])
        if active
        else list_group(
            [
                lambda position: list_item(
                    l10n.t("homeActiveEmpty"),
                    position=position,
                    leading="info",
                    on_click=lambda: app.shell.show_sdk(None),
                )
            ]
        )
    )
    return pane(
        ft.Container(
            on_size_change=on_size_change,
            size_change_interval=100,
            expand=True,
            content=ft.Column(
                [
                    page_header(
                        l10n.t("appTitle"),
                        subtitle=subtitle,
                        actions=[
                            icon_button(
                                "refresh",
                                l10n.t("actionRefresh"),
                                None if model.installed.is_loading else model.installed.load,
                            )
                        ],
                    ),
                    ft.Container(
                        padding=ft.Padding.only(left=Gaps.XL, right=Gaps.XL, bottom=Gaps.XL),
                        content=content_width(
                            ft.Column(
                                [
                                    _stats(
                                        app, len(sdks), version_count, len(active), compact=min(content, 1080) < 560
                                    ),
                                    group_header(l10n.t("homeActiveTitle")),
                                    active_list,
                                    ft.Container(height=Gaps.XL),
                                    _shortcuts(app, narrow=min(content, 1080) < 760),
                                ],
                                spacing=0,
                                horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
                            ),
                            max_width=1080,
                        ),
                    ),
                ],
                spacing=0,
                scroll=ft.ScrollMode.AUTO,
                expand=True,
            ),
        )
    )


def _stats(app: AppContext, plugins: int, versions: int, active: int, *, compact: bool) -> ft.Row:
    l10n = app.l10n

    def show_sdks() -> None:
        app.shell.show_sdk(None)

    tiles: list[ft.Control] = [
        _stat_tile(
            plugins,
            l10n.t("homeStatPlugins", count=plugins),
            "plugins",
            "cookie9Sided",
            C.PRIMARY_CONTAINER,
            C.ON_PRIMARY_CONTAINER,
            show_sdks,
            compact=compact,
        ),
        _stat_tile(
            versions,
            l10n.t("homeStatVersions", count=versions),
            "versions",
            "clover4Leaf",
            C.SECONDARY_CONTAINER,
            C.ON_SECONDARY_CONTAINER,
            show_sdks,
            compact=compact,
        ),
        _stat_tile(
            active,
            l10n.t("homeStatActive"),
            "current",
            "sunny",
            C.TERTIARY_CONTAINER,
            C.ON_TERTIARY_CONTAINER,
            show_sdks,
            compact=compact,
        ),
    ]
    # Narrow windows keep the three tiles in a row, in a compact size.
    return ft.Row(tiles, spacing=Gaps.S if compact else Gaps.M)


def _stat_tile(
    value: int,
    label: str,
    meaning: str,
    shape: str,
    background: ft.ColorValue,
    foreground: ft.ColorValue,
    on_click: Callable[[], object],
    *,
    compact: bool,
) -> ft.Card:
    return ft.Card(
        bgcolor=background,
        shape=ft.RoundedRectangleBorder(radius=Corners.EXTRA_LARGE),
        expand=True,
        content=ft.Container(
            height=136,
            ink=True,
            on_click=lambda _: on_click(),
            content=ft.Stack(
                [
                    # A large, partly hidden shape gives the tile its character.
                    MaterialShape(
                        shape=shape, size=140, bgcolor=ft.Colors.with_opacity(0.08, foreground), right=-28, bottom=-36
                    ),
                    ft.Container(
                        padding=Gaps.L if compact else Gaps.XL,
                        content=ft.Column(
                            [
                                icon_control(meaning, color=foreground, filled=True),
                                ft.Container(expand=True),
                                text(
                                    str(value),
                                    S.HEADLINE_LARGE if compact else S.DISPLAY_SMALL,
                                    color=foreground,
                                    emphasized=True,
                                    height=1,
                                ),
                                text(label, S.LABEL_LARGE, color=foreground, emphasized=True, max_lines=2),
                            ],
                            spacing=Gaps.XS,
                        ),
                    ),
                ],
                clip_behavior=ft.ClipBehavior.HARD_EDGE,
                expand=True,
            ),
        ),
    )


def _shortcuts(app: AppContext, *, narrow: bool) -> ft.Control:
    l10n = app.l10n

    async def copy() -> None:
        await app.services.copy_text(ACTIVATION_COMMAND)
        app.messenger.snack(l10n.t("actionCopied"))

    shell = ft.Card(
        bgcolor=C.SURFACE_CONTAINER_HIGHEST,
        expand=3 if not narrow else None,
        content=ft.Container(
            padding=Gaps.XL,
            content=ft.Column(
                [
                    ft.Row(
                        [
                            icon_control("terminal", color=C.PRIMARY),
                            ft.Container(
                                content=text(l10n.t("homeShellTitle"), S.TITLE_MEDIUM, emphasized=True), expand=True
                            ),
                        ],
                        spacing=Gaps.M,
                    ),
                    text(l10n.t("homeShellBody"), S.BODY_MEDIUM, color=C.ON_SURFACE_VARIANT),
                    ft.Container(
                        bgcolor=C.SURFACE_CONTAINER_LOWEST,
                        border_radius=Corners.MEDIUM,
                        padding=ft.Padding.only(left=Gaps.L, right=Gaps.XS),
                        content=ft.Row(
                            [
                                ft.Container(
                                    content=text(
                                        ACTIVATION_COMMAND, S.BODY_MEDIUM, selectable=True, font_family="monospace"
                                    ),
                                    expand=True,
                                ),
                                icon_button("copy", l10n.t("actionCopy"), lambda: spawn(copy())),
                            ]
                        ),
                    ),
                ],
                spacing=Gaps.S,
            ),
        ),
    )
    explore = _shortcut_card(
        "explore",
        l10n.t("homeExploreTitle"),
        l10n.t("homeExploreBody"),
        lambda: app.shell.go(Destination.PLUGINS),
        expand=not narrow,
    )
    docs = _shortcut_card(
        "docs",
        l10n.t("homeDocsTitle"),
        l10n.t("homeDocsBody"),
        lambda: spawn(app.services.open_url(links.VFOX_DOCS)),
        trailing="openInNew",
        expand=not narrow,
    )
    if narrow:
        return ft.Column([shell, explore, docs], spacing=Gaps.M, horizontal_alignment=ft.CrossAxisAlignment.STRETCH)
    return ft.Row(
        [
            shell,
            ft.Column(
                [explore, docs],
                spacing=Gaps.M,
                expand=2,
                horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
            ),
        ],
        spacing=Gaps.M,
        intrinsic_height=True,
        vertical_alignment=ft.CrossAxisAlignment.STRETCH,
    )


def _shortcut_card(
    meaning: str,
    title: str,
    body: str,
    on_click: Callable[[], object],
    *,
    trailing: str = "chevronRight",
    expand: bool = False,
) -> ft.Card:
    return ft.Card(
        bgcolor=C.SURFACE_CONTAINER,
        expand=expand,
        content=ft.Container(
            ink=True,
            on_click=lambda _: on_click(),
            padding=Gaps.L,
            content=ft.Row(
                [
                    MaterialShape(
                        shape="cookie6Sided",
                        size=48,
                        bgcolor=C.SECONDARY_CONTAINER,
                        content=icon_control(meaning, color=C.ON_SECONDARY_CONTAINER),
                    ),
                    ft.Column(
                        [
                            text(title, S.TITLE_SMALL, emphasized=True),
                            text(body, S.BODY_SMALL, color=C.ON_SURFACE_VARIANT),
                        ],
                        spacing=Gaps.XS,
                        alignment=ft.MainAxisAlignment.CENTER,
                        expand=True,
                    ),
                    icon_control(trailing, color=C.ON_SURFACE_VARIANT),
                ],
                spacing=Gaps.L,
            ),
        ),
    )
