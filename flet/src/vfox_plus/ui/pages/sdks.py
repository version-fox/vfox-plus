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

"""Installed SDKs, using the list-detail canonical layout.

Wide pages show the list and the selected SDK side by side. Narrow pages
show one of them at a time and move between them along the horizontal shared
axis.
"""

from collections.abc import Callable, Sequence

import flet as ft

from vfox_plus.core.models import InstalledSdk
from vfox_plus.ui.context import AppContext, Destination, use_app
from vfox_plus.ui.hooks import use_listenable, use_width
from vfox_plus.ui.icons import icon_control
from vfox_plus.ui.pages.sdk_detail import SdkDetail
from vfox_plus.ui.presentation import LIST_PANE_WIDTH, TWO_PANE_WIDTH
from vfox_plus.ui.tokens import Gaps
from vfox_plus.ui.widgets.lists import ListPosition, list_item
from vfox_plus.ui.widgets.parts import (
    error_message,
    filled_button,
    icon_button,
    loading_message,
    page_header,
    pane,
    search_bar,
    status_message,
)
from vfox_plus_m3e import ShapeAvatar, TransitionSwitcher

C = ft.Colors


@ft.component
def SdksPage() -> ft.Control:
    """The SDKs page."""
    app = use_app()
    model = app.model
    l10n = app.l10n
    use_listenable(model.installed, app.shell)
    width, on_size_change = use_width()
    query, set_query = ft.use_state("")
    sdks = model.installed.value
    if sdks is None:
        if model.installed.error is not None:
            content: ft.Control = pane(
                error_message(
                    l10n.t("sdksLoadError"), model.installed.error, l10n.t("actionRetry"), model.installed.load
                )
            )
        else:
            content = pane(loading_message(semantics_label=l10n.t("loading")))
        return ft.Container(content=content, expand=True, on_size_change=on_size_change)
    names = {sdk.name for sdk in sdks}
    selected = app.shell.selected_sdk if app.shell.selected_sdk in names else None
    two_pane = (width or TWO_PANE_WIDTH) >= TWO_PANE_WIDTH
    sdk_list = _sdk_list(
        app,
        sdks,
        query,
        set_query,
        selected=selected if two_pane else None,
        show_chevrons=not two_pane,
    )
    if two_pane:
        body: ft.Control = ft.Row(
            [
                ft.Container(content=pane(sdk_list), width=LIST_PANE_WIDTH),
                pane(
                    TransitionSwitcher(
                        content_key=selected or "",
                        content=SdkDetail(selected, key=selected)
                        if selected is not None
                        else status_message("sdks", l10n.t("sdksSelectPrompt"), shape="gem"),
                        expand=True,
                    )
                ),
            ],
            spacing=Gaps.XL,
            vertical_alignment=ft.CrossAxisAlignment.STRETCH,
            expand=True,
        )
    else:
        body = TransitionSwitcher(
            content_key=selected or "",
            transition="shared_axis_x",
            reverse=selected is None,
            content=pane(sdk_list)
            if selected is None
            else pane(SdkDetail(selected, on_back=lambda: app.shell.show_sdk(None), key=selected)),
            expand=True,
        )
    return ft.Container(content=body, expand=True, on_size_change=on_size_change, size_change_interval=100)


def _sdk_list(
    app: AppContext,
    sdks: Sequence[InstalledSdk],
    query: str,
    set_query: Callable[[str], None],
    *,
    selected: str | None,
    show_chevrons: bool,
) -> ft.Control:
    model = app.model
    l10n = app.l10n
    if not sdks:
        return status_message(
            "empty",
            l10n.t("sdksEmptyTitle"),
            body=l10n.t("sdksEmptyBody"),
            shape="flower",
            actions=[filled_button(l10n.t("navAddPlugin"), lambda: app.shell.go(Destination.PLUGINS), meaning="add")],
        )
    needle = query.strip().lower()
    visible = [sdk for sdk in sdks if not needle or needle in sdk.name.lower()]

    def item(index: int, sdk: InstalledSdk) -> ft.Control:
        current = l10n.t("sdksCurrent", version=sdk.current) if sdk.current is not None else l10n.t("sdksNoCurrent")
        return list_item(
            sdk.name,
            position=ListPosition.of(index, len(visible)),
            leading=ShapeAvatar(name=sdk.name),
            subtitle=ft.Text(
                f"{current}\n{l10n.t('sdksVersionCount', count=len(sdk.versions))}",
                theme_style=ft.TextThemeStyle.BODY_MEDIUM,
                color=C.ON_SECONDARY_CONTAINER if sdk.name == selected else C.ON_SURFACE_VARIANT,
                max_lines=2,
                overflow=ft.TextOverflow.ELLIPSIS,
            ),
            trailing=icon_control("chevronRight", color=C.ON_SURFACE_VARIANT) if show_chevrons else None,
            selected=sdk.name == selected,
            on_click=lambda: app.shell.show_sdk(sdk.name),
            key=f"sdk-{sdk.name}",
        )

    items: list[ft.Control] = (
        [
            status_message("notFound", l10n.t("sdksNoMatch", query=query.strip())),
        ]
        if not visible
        else [
            ft.Container(
                content=ft.Column(
                    [item(index, sdk) for index, sdk in enumerate(visible)],
                    spacing=Gaps.HAIRLINE,
                    horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
                ),
                padding=ft.Padding.only(left=Gaps.L, right=Gaps.L, bottom=Gaps.L),
            )
        ]
    )
    return ft.Column(
        [
            page_header(
                l10n.t("navSdks"),
                actions=[
                    icon_button(
                        "refresh", l10n.t("actionRefresh"), None if model.installed.is_loading else model.installed.load
                    )
                ],
            ),
            ft.Container(
                content=search_bar(l10n.t("sdksSearchHint"), query, set_query, clear_tooltip=l10n.t("actionClose")),
                padding=ft.Padding.only(left=Gaps.L, right=Gaps.L, bottom=Gaps.M),
            ),
            *items,
        ],
        spacing=0,
        scroll=ft.ScrollMode.AUTO,
        expand=True,
    )
