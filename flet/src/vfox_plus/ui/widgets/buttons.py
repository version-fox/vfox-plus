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

"""Material 3 Expressive connected button groups, split buttons and menus,
as `button_group.dart` and `split_button.dart` of
`flutter/lib/src/ui/widgets/`.
"""

import math
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from typing import cast

import flet as ft

from vfox_plus.ui import motion
from vfox_plus.ui.icons import icon, icon_control
from vfox_plus.ui.tokens import Corners, Gaps
from vfox_plus_m3e import FULL, SpringSurface

C = ft.Colors
S = ft.TextThemeStyle


@dataclass(frozen=True)
class Segment[T]:
    """One choice of a connected button group."""

    value: T
    label: str
    meaning: str | None = None
    """The icon before the label, if any."""
    tooltip: str | None = None
    key: str | None = None


def _horizontal(start: float, end: float) -> ft.BorderRadius:
    return ft.BorderRadius(top_left=start, bottom_left=start, top_right=end, bottom_right=end)


def button_group[T](
    segments: Sequence[Segment[T]],
    selected: T,
    on_select: Callable[[T], object] | None,
    *,
    expand: bool = False,
    height: float = 40,
) -> ft.Row:
    """A connected button group with single selection, which replaces the
    segmented button of Material 3.

    The buttons sit 2 dp apart. Outer corners are fully rounded and inner
    corners small; the selected button morphs into a fully rounded shape and a
    pressed button squeezes its inner corners. See
    https://m3.material.io/components/button-groups.
    """
    enabled = on_select is not None
    buttons: list[ft.Control] = []
    for index, segment in enumerate(segments):
        is_selected = segment.value == selected
        first = index == 0
        last = index == len(segments) - 1
        foreground = (
            (C.ON_PRIMARY if is_selected else C.ON_SECONDARY_CONTAINER)
            if enabled
            else ft.Colors.with_opacity(0.38, C.ON_SURFACE)
        )
        label: list[ft.Control] = []
        if segment.meaning is not None:
            label.append(icon_control(segment.meaning, size=20, color=foreground))
        label.append(
            ft.Text(
                segment.label,
                theme_style=S.LABEL_LARGE,
                color=foreground,
                max_lines=1,
                overflow=ft.TextOverflow.ELLIPSIS,
            )
        )

        def choose(_: ft.Event[SpringSurface], value: T = segment.value) -> None:
            cast("Callable[[T], object]", on_select)(value)

        buttons.append(
            SpringSurface(
                content=ft.Container(
                    content=ft.Row(label, spacing=Gaps.S, tight=True, alignment=ft.MainAxisAlignment.CENTER),
                    height=height,
                    padding=ft.Padding.symmetric(horizontal=Gaps.L),
                    alignment=ft.Alignment.CENTER,
                ),
                bgcolor=(C.PRIMARY if is_selected else C.SECONDARY_CONTAINER)
                if enabled
                else ft.Colors.with_opacity(0.1, C.ON_SURFACE),
                radius=_horizontal(FULL if first else Corners.SMALL, FULL if last else Corners.SMALL),
                selected_radius=_horizontal(FULL, FULL),
                pressed_radius=_horizontal(
                    FULL if first else Corners.EXTRA_SMALL, FULL if last else Corners.EXTRA_SMALL
                ),
                selected=is_selected,
                mutually_exclusive=True,
                semantics_selected=is_selected,
                disabled=not enabled,
                tooltip=segment.tooltip,
                expand=expand,
                key=segment.key,
                on_click=choose,
            )
        )
    return ft.Row(buttons, spacing=Gaps.HAIRLINE, tight=not expand)


@dataclass(frozen=True)
class MenuEntry:
    """An item of a menu."""

    label: str
    on_click: Callable[[], object]
    meaning: str | None = None
    destructive: bool = False
    key: str | None = None


def _menu_item(entry: MenuEntry) -> ft.PopupMenuItem:
    color = C.ERROR if entry.destructive else C.ON_SURFACE
    row: list[ft.Control] = []
    if entry.meaning is not None:
        row.append(icon_control(entry.meaning, size=20, color=C.ERROR if entry.destructive else C.ON_SURFACE_VARIANT))
    row.append(ft.Text(entry.label, color=color))
    return ft.PopupMenuItem(
        content=ft.Row(row, spacing=Gaps.M, tight=True),
        on_click=lambda _: entry.on_click(),
        key=entry.key,
    )


def overflow_menu(tooltip: str, entries: Sequence[MenuEntry], *, key: str | None = None) -> ft.PopupMenuButton:
    """The `more` icon button with a menu of `entries`."""
    return ft.PopupMenuButton(
        icon=icon("more"),
        tooltip=tooltip,
        items=[_menu_item(entry) for entry in entries],
        menu_position=ft.PopupMenuPosition.UNDER,
        key=key,
    )


@ft.component
def SplitButton(
    label: str,
    on_click: Callable[[], object] | None,
    entries: Sequence[MenuEntry],
    menu_tooltip: str,
    *,
    meaning: str | None = None,
    filled: bool = False,
    key_prefix: str = "",
) -> ft.Control:
    """A split button: a main action and a menu of related actions, joined by
    a 2 dp gap. While the menu is open the trailing button becomes a circle
    and its chevron turns upside down. See
    https://m3.material.io/components/split-button.
    """
    menu_open, set_menu_open = ft.use_state(False)
    # A new reference for every render, as the event handlers of this render
    # address the controls of this render.
    menu: ft.Ref[ft.ContextMenu] = ft.Ref()
    enabled = on_click is not None
    if not enabled:
        background: ft.ColorValue = ft.Colors.with_opacity(0.1, C.ON_SURFACE)
        foreground: ft.ColorValue = ft.Colors.with_opacity(0.38, C.ON_SURFACE)
    elif filled:
        background, foreground = C.PRIMARY, C.ON_PRIMARY
    else:
        background, foreground = C.SECONDARY_CONTAINER, C.ON_SECONDARY_CONTAINER
    height = 40
    label_row: list[ft.Control] = []
    if meaning is not None:
        label_row.append(icon_control(meaning, size=20, color=foreground))
    label_row.append(ft.Text(label, theme_style=S.LABEL_LARGE, color=foreground))

    async def open_menu(_: ft.Event[SpringSurface]) -> None:
        context_menu = menu.current
        if context_menu is None:
            return
        set_menu_open(True)
        await context_menu.open(local_position=ft.Offset(0, height + Gaps.XS))

    leading = SpringSurface(
        content=ft.Container(
            content=ft.Row(label_row, spacing=Gaps.S, tight=True),
            height=height,
            padding=ft.Padding.only(left=Gaps.L, right=Gaps.M),
            alignment=ft.Alignment.CENTER,
        ),
        bgcolor=background,
        radius=_horizontal(FULL, Corners.EXTRA_SMALL),
        pressed_radius=_horizontal(FULL, Corners.SMALL),
        disabled=not enabled,
        on_click=(lambda _: cast("Callable[[], object]", on_click)()) if enabled else None,
        key=f"{key_prefix}main" if key_prefix else None,
    )
    trailing = ft.ContextMenu(
        ref=menu,
        content=SpringSurface(
            content=ft.Container(
                content=ft.Icon(
                    icon("expandMore"),
                    size=22,
                    color=foreground,
                    rotate=math.pi if menu_open else 0,
                    animate_rotation=ft.Animation(motion.FAST_SPATIAL.settle_duration, ft.AnimationCurve.EASE_OUT_BACK),
                ),
                width=height + 4,
                height=height,
                alignment=ft.Alignment.CENTER,
            ),
            bgcolor=background,
            radius=_horizontal(Corners.EXTRA_SMALL, FULL),
            selected_radius=_horizontal(FULL, FULL),
            pressed_radius=_horizontal(Corners.SMALL, FULL),
            selected=menu_open,
            semantics_selected=None,
            disabled=not enabled,
            tooltip=menu_tooltip,
            on_click=open_menu if enabled else None,
            key=f"{key_prefix}menu" if key_prefix else None,
        ),
        items=[_menu_item(entry) for entry in entries],
        secondary_trigger=None,
        tertiary_trigger=None,
        on_select=lambda _: set_menu_open(False),
        on_dismiss=lambda _: set_menu_open(False),
    )
    return ft.Row([leading, trailing], spacing=Gaps.HAIRLINE, tight=True)
