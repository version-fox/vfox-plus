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

"""Material 3 Expressive segmented lists, as
`flutter/lib/src/ui/widgets/list_group.dart`: items are separate surfaces,
2 dp apart, that together read as one rounded group. Outer corners are large
and inner corners small; the inner corners of a hovered, pressed or selected
item grow, so that it briefly stands out as its own shape.
"""

from collections.abc import Callable, Sequence
from enum import Enum

import flet as ft

from vfox_plus.ui.icons import icon_control
from vfox_plus.ui.tokens import Corners, Gaps
from vfox_plus_m3e import SpringSurface

C = ft.Colors
S = ft.TextThemeStyle

_OUTER = Corners.LARGE_INCREASED
_INNER = Corners.EXTRA_SMALL
_INNER_ACTIVE = Corners.MEDIUM


class ListPosition(Enum):
    """Where an item sits in a group, which decides its corners."""

    SINGLE = "single"
    FIRST = "first"
    MIDDLE = "middle"
    LAST = "last"

    @classmethod
    def of(cls, index: int, count: int) -> ListPosition:
        """The position of item `index` of `count`."""
        if count <= 1:
            return cls.SINGLE
        if index == 0:
            return cls.FIRST
        return cls.LAST if index == count - 1 else cls.MIDDLE

    def radius(self, inner: float) -> ft.BorderRadius:
        """The corners of an item at this position with `inner` corners."""
        top = _OUTER if self in (ListPosition.FIRST, ListPosition.SINGLE) else inner
        bottom = _OUTER if self in (ListPosition.LAST, ListPosition.SINGLE) else inner
        return ft.BorderRadius(top_left=top, top_right=top, bottom_left=bottom, bottom_right=bottom)


def list_item(
    title: str | ft.Control,
    *,
    position: ListPosition = ListPosition.SINGLE,
    leading: str | ft.Control | None = None,
    subtitle: str | ft.Control | None = None,
    trailing: ft.Control | None = None,
    on_click: Callable[[], object] | None = None,
    selected: bool = False,
    title_weight: ft.FontWeight | None = None,
    wrap_trailing: bool = False,
    padding: ft.Padding | None = None,
    key: str | None = None,
    tooltip: str | None = None,
) -> SpringSurface:
    """An item of a segmented list with the layout of a Material list item.

    `leading` may be an icon meaning. With `wrap_trailing`, a wide trailing
    control, such as a button group or a text field, sits below the text.
    """
    foreground = C.ON_SECONDARY_CONTAINER if selected else C.ON_SURFACE
    secondary = C.ON_SECONDARY_CONTAINER if selected else C.ON_SURFACE_VARIANT
    texts: list[ft.Control] = [
        ft.Text(title, theme_style=S.BODY_LARGE, color=foreground, weight=title_weight)
        if isinstance(title, str)
        else title
    ]
    if subtitle is not None:
        texts.append(
            ft.Text(subtitle, theme_style=S.BODY_MEDIUM, color=secondary) if isinstance(subtitle, str) else subtitle
        )
    if wrap_trailing and trailing is not None:
        texts.append(ft.Container(content=trailing, padding=ft.Padding.only(top=Gaps.S)))
    row: list[ft.Control] = []
    if leading is not None:
        row.append(icon_control(leading, color=secondary) if isinstance(leading, str) else leading)
    row.append(ft.Column(texts, spacing=0, expand=True, tight=True))
    if trailing is not None and not wrap_trailing:
        row.append(trailing)
    return SpringSurface(
        content=ft.Container(
            content=ft.Row(row, spacing=Gaps.L, vertical_alignment=ft.CrossAxisAlignment.CENTER),
            padding=padding or ft.Padding.symmetric(horizontal=Gaps.L, vertical=Gaps.M),
        ),
        bgcolor=C.SECONDARY_CONTAINER if selected else C.SURFACE_CONTAINER,
        radius=position.radius(_INNER),
        hover_radius=position.radius(_INNER_ACTIVE),
        selected=selected,
        interactive=on_click is not None,
        semantics_selected=True if selected else None,
        on_click=(lambda _: on_click()) if on_click is not None else None,
        tooltip=tooltip,
        key=key,
    )


def list_group(items: Sequence[Callable[[ListPosition], ft.Control]]) -> ft.Column:
    """A segmented list: each builder gets the position of its item."""
    return ft.Column(
        [build(ListPosition.of(index, len(items))) for index, build in enumerate(items)],
        spacing=Gaps.HAIRLINE,
        horizontal_alignment=ft.CrossAxisAlignment.STRETCH,
        tight=True,
    )
