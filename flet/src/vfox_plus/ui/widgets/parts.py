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

"""The parts pages are made of: panes, headers, placeholders and small
controls, as `flutter/lib/src/ui/widgets/page_parts.dart`.
"""

from collections.abc import Callable, Sequence

import flet as ft

from vfox_plus.core.vfox.command import VfoxError
from vfox_plus.ui.icons import icon, icon_control
from vfox_plus.ui.theme import icon_button_style
from vfox_plus.ui.tokens import Corners, Gaps
from vfox_plus_m3e import ConstrainedBox, LoadingIndicator, MaterialShape

C = ft.Colors
W = ft.FontWeight


def text(
    value: str,
    style: ft.TextThemeStyle,
    *,
    color: ft.ColorValue | None = None,
    emphasized: bool = False,
    weight: ft.FontWeight | None = None,
    max_lines: int | None = None,
    selectable: bool = False,
    text_align: ft.TextAlign | None = None,
    font_family: str | None = None,
    height: float | None = None,
    key: str | None = None,
) -> ft.Text:
    """A text in a role of the type scale. `emphasized` uses the heavier
    weight of Material 3 Expressive (see `docs/design-system.md#typography`).
    """
    if emphasized and weight is None:
        weight = _EMPHASIZED.get(style, W.W_700)
    return ft.Text(
        value,
        theme_style=style,
        color=color,
        weight=weight,
        max_lines=max_lines,
        overflow=ft.TextOverflow.ELLIPSIS if max_lines is not None else ft.TextOverflow.CLIP,
        selectable=selectable,
        text_align=text_align or ft.TextAlign.START,
        font_family=font_family,
        style=ft.TextStyle(height=height) if height is not None else None,
        key=key,
    )


S = ft.TextThemeStyle
_EMPHASIZED = {
    S.DISPLAY_LARGE: W.W_500,
    S.DISPLAY_MEDIUM: W.W_500,
    S.DISPLAY_SMALL: W.W_500,
    S.HEADLINE_LARGE: W.W_500,
    S.HEADLINE_MEDIUM: W.W_500,
    S.HEADLINE_SMALL: W.W_500,
    S.TITLE_LARGE: W.W_500,
    S.TITLE_MEDIUM: W.W_700,
    S.TITLE_SMALL: W.W_700,
    S.BODY_LARGE: W.W_500,
    S.BODY_MEDIUM: W.W_500,
    S.LABEL_LARGE: W.W_700,
    S.LABEL_MEDIUM: W.W_700,
}


def icon_button(
    meaning: str,
    tooltip: str,
    on_click: Callable[[], object] | None,
    *,
    filled_tonal: bool = False,
    key: str | None = None,
    color: ft.ColorValue | None = None,
    compact: bool = False,
) -> ft.IconButton:
    """An icon-only button; its tooltip is also its accessible name.
    `compact` buttons, in list items, are 40 dp instead of 48 dp.
    """
    return ft.IconButton(
        icon=icon(meaning),
        tooltip=tooltip,
        on_click=(lambda _: on_click()) if on_click is not None else None,
        disabled=on_click is None,
        icon_color=color,
        style=icon_button_style(bgcolor=C.SECONDARY_CONTAINER, color=C.ON_SECONDARY_CONTAINER)
        if filled_tonal
        else None,
        # The Material 3 Expressive icon buttons of the Flutter implementation
        # keep a 48 dp target; Flutter shrinks them to 40 dp on desktops.
        size_constraints=None if compact else ft.BoxConstraints(min_width=48, min_height=48),
        key=key,
    )


def pane(content: ft.Control, *, bgcolor: ft.ColorValue | None = None) -> ft.Container:
    """A rounded surface that holds the content of a page. Pages sit on panes
    next to the navigation rail, following the Material 3 canonical layouts.
    """
    return ft.Container(
        content=content,
        bgcolor=bgcolor or C.SURFACE_CONTAINER_LOW,
        border_radius=Corners.EXTRA_LARGE,
        clip_behavior=ft.ClipBehavior.ANTI_ALIAS,
        expand=True,
    )


def page_header(
    title: str,
    *,
    subtitle: str | None = None,
    actions: Sequence[ft.Control] = (),
    leading: ft.Control | None = None,
) -> ft.Container:
    """The large, emphasized header of a page."""
    texts: list[ft.Control] = [
        ft.Semantics(
            header=True,
            content=text(title, S.HEADLINE_MEDIUM, color=C.ON_SURFACE, emphasized=True, max_lines=1),
        )
    ]
    if subtitle is not None:
        texts.append(text(subtitle, S.BODY_MEDIUM, color=C.ON_SURFACE_VARIANT, max_lines=2))
    row: list[ft.Control] = []
    if leading is not None:
        row.append(leading)
    row.append(ft.Column(texts, spacing=Gaps.XS, expand=True, tight=True))
    if actions:
        row.append(ft.Row(list(actions), spacing=Gaps.S, wrap=True, tight=True))
    return ft.Container(
        content=ft.Row(row, spacing=Gaps.S, vertical_alignment=ft.CrossAxisAlignment.CENTER),
        padding=ft.Padding.only(left=Gaps.XL, top=Gaps.XL, right=Gaps.L, bottom=Gaps.L),
    )


def content_width(content: ft.Control, max_width: float = 960) -> ConstrainedBox:
    """Centres page content and limits its width for comfortable reading."""
    return ConstrainedBox(content=content, max_width=max_width, alignment=ft.Alignment.TOP_CENTER)


def group_header(title: str, trailing: ft.Control | None = None) -> ft.Container:
    """A heading above a segmented list."""
    row: list[ft.Control] = [
        ft.Semantics(header=True, expand=True, content=text(title, S.TITLE_SMALL, color=C.PRIMARY)),
    ]
    if trailing is not None:
        row.append(trailing)
    return ft.Container(
        content=ft.Row(row, vertical_alignment=ft.CrossAxisAlignment.CENTER),
        padding=ft.Padding.only(left=Gaps.L, right=Gaps.XS, top=Gaps.XL, bottom=Gaps.S),
    )


def status_message(
    meaning: str,
    title: str,
    *,
    body: str | None = None,
    actions: Sequence[ft.Control] = (),
    shape: str = "cookie9Sided",
    is_error: bool = False,
) -> ft.Container:
    """A friendly placeholder for empty, unavailable or failed content, with
    an expressive shape behind its icon.
    """
    column: list[ft.Control] = [
        MaterialShape(
            shape=shape,
            size=112,
            bgcolor=C.ERROR_CONTAINER if is_error else C.SECONDARY_CONTAINER,
            content=icon_control(
                meaning, size=48, filled=True, color=C.ON_ERROR_CONTAINER if is_error else C.ON_SECONDARY_CONTAINER
            ),
        ),
        ft.Container(height=Gaps.XL - Gaps.S),
        text(title, S.TITLE_LARGE, color=C.ON_SURFACE, emphasized=True, text_align=ft.TextAlign.CENTER),
    ]
    if body is not None:
        column.append(text(body, S.BODY_MEDIUM, color=C.ON_SURFACE_VARIANT, text_align=ft.TextAlign.CENTER))
    if actions:
        column.append(ft.Container(height=Gaps.XL - Gaps.S))
        column.append(
            ft.Row(list(actions), spacing=Gaps.S, run_spacing=Gaps.S, wrap=True, alignment=ft.MainAxisAlignment.CENTER)
        )
    return ft.Container(
        content=ft.Column(
            [
                ConstrainedBox(
                    max_width=440 + 2 * Gaps.XXL,
                    content=ft.Container(
                        content=ft.Column(
                            column,
                            spacing=Gaps.S,
                            horizontal_alignment=ft.CrossAxisAlignment.CENTER,
                            tight=True,
                        ),
                        padding=Gaps.XXL,
                    ),
                )
            ],
            scroll=ft.ScrollMode.AUTO,
            horizontal_alignment=ft.CrossAxisAlignment.CENTER,
            alignment=ft.MainAxisAlignment.CENTER,
            expand=True,
        ),
        alignment=ft.Alignment.CENTER,
        expand=True,
    )


def describe_error(error: BaseException | None) -> str:
    """A short, user-facing description of `error`."""
    if isinstance(error, VfoxError):
        return error.message
    return str(error)


def error_message(
    title: str, error: BaseException | None, retry_label: str, on_retry: Callable[[], object]
) -> ft.Container:
    """A placeholder for an error, with the error text and a retry button."""
    return status_message(
        "error",
        title,
        body=describe_error(error),
        shape="softBurst",
        is_error=True,
        actions=[tonal_button(retry_label, on_retry, meaning="retry")],
    )


def loading_message(label: str | None = None, semantics_label: str | None = None) -> ft.Container:
    """A centred loading indicator, with a short text when the wait can be
    long.
    """
    column: list[ft.Control] = [LoadingIndicator(contained=True, size=64, semantics_label=label or semantics_label)]
    if label is not None:
        column.append(text(label, S.BODY_MEDIUM, color=C.ON_SURFACE_VARIANT, text_align=ft.TextAlign.CENTER))
    return ft.Container(
        content=ft.Column(column, spacing=Gaps.L, horizontal_alignment=ft.CrossAxisAlignment.CENTER, tight=True),
        alignment=ft.Alignment.CENTER,
        padding=Gaps.XXL,
        expand=True,
    )


def tonal_button(
    label: str, on_click: Callable[[], object] | None, *, meaning: str | None = None, key: str | None = None
) -> ft.FilledTonalButton:
    """A filled tonal button with an optional icon."""
    return ft.FilledTonalButton(
        content=label,
        icon=icon(meaning) if meaning is not None else None,
        on_click=(lambda _: on_click()) if on_click is not None else None,
        disabled=on_click is None,
        key=key,
    )


def filled_button(
    label: str,
    on_click: Callable[[], object] | None,
    *,
    meaning: str | None = None,
    destructive: bool = False,
    key: str | None = None,
) -> ft.FilledButton:
    """A filled button, in error colours when `destructive`."""
    return ft.FilledButton(
        content=label,
        icon=icon(meaning) if meaning is not None else None,
        on_click=(lambda _: on_click()) if on_click is not None else None,
        disabled=on_click is None,
        bgcolor=C.ERROR if destructive else None,
        color=C.ON_ERROR if destructive else None,
        key=key,
    )


def outlined_button(
    label: str, on_click: Callable[[], object] | None, *, meaning: str | None = None, key: str | None = None
) -> ft.OutlinedButton:
    """An outlined button with an optional icon."""
    return ft.OutlinedButton(
        content=label,
        icon=icon(meaning) if meaning is not None else None,
        on_click=(lambda _: on_click()) if on_click is not None else None,
        disabled=on_click is None,
        key=key,
    )


def text_button(label: str, on_click: Callable[[], object] | None, *, key: str | None = None) -> ft.TextButton:
    """A text button."""
    return ft.TextButton(
        content=label,
        on_click=(lambda _: on_click()) if on_click is not None else None,
        disabled=on_click is None,
        key=key,
    )


def tag(label: str, *, meaning: str | None = None, emphasized: bool = False) -> ft.Container:
    """A small, fully rounded label, such as the version in use or the note
    of a version.
    """
    foreground = C.ON_TERTIARY_CONTAINER if emphasized else C.ON_SURFACE_VARIANT
    row: list[ft.Control] = []
    if meaning is not None:
        row.append(icon_control(meaning, size=16, color=foreground, filled=True))
    row.append(text(label, S.LABEL_MEDIUM, color=foreground, max_lines=1))
    return ft.Container(
        content=ft.Row(row, spacing=Gaps.XS, tight=True),
        bgcolor=C.TERTIARY_CONTAINER if emphasized else C.SURFACE_CONTAINER_HIGHEST,
        border_radius=Corners.FULL,
        padding=ft.Padding.symmetric(horizontal=Gaps.S + 2, vertical=3),
    )


def text_field(
    label: str | None = None,
    *,
    value: str = "",
    hint: str | None = None,
    helper: str | None = None,
    error: str | None = None,
    multiline: bool = False,
    autofocus: bool = False,
    monospace: bool = False,
    prefix: str | None = None,
    dense: bool = False,
    on_change: Callable[[str], object] | None = None,
    on_submit: Callable[[str], object] | None = None,
    on_blur: Callable[[str], object] | None = None,
    key: str | None = None,
    expand: bool | int | None = None,
    ref: ft.Ref[ft.TextField] | None = None,
) -> ft.TextField:
    """A filled text field without border and with medium corners, as the
    input decoration theme of the Flutter implementation.
    """
    field = ft.TextField(
        value=value,
        label=label,
        hint_text=hint,
        helper=helper,
        error=error,
        multiline=multiline,
        min_lines=3 if multiline else None,
        max_lines=8 if multiline else 1,
        autofocus=autofocus,
        filled=True,
        fill_color=C.SURFACE_CONTAINER_HIGHEST,
        border={
            ft.ControlState.DEFAULT: ft.OutlineInputBorder(
                side=ft.BorderSide(width=0, color=ft.Colors.TRANSPARENT), border_radius=Corners.MEDIUM
            ),
            ft.ControlState.FOCUSED: ft.OutlineInputBorder(
                side=ft.BorderSide(width=2, color=C.PRIMARY), border_radius=Corners.MEDIUM
            ),
        },
        text_style=ft.TextStyle(font_family="monospace") if monospace else None,
        prefix_icon=icon(prefix) if prefix is not None else None,
        dense=dense,
        key=key,
        expand=expand,
        ref=ref,
    )
    if on_change is not None:
        field.on_change = lambda event: on_change(event.control.value)
    if on_submit is not None:
        field.on_submit = lambda event: on_submit(event.control.value)
    if on_blur is not None:
        field.on_blur = lambda event: on_blur(event.control.value)
    return field


def search_bar(
    hint: str,
    value: str,
    on_change: Callable[[str], object],
    *,
    clear_tooltip: str | None = None,
    key: str | None = None,
) -> ConstrainedBox:
    """The Material 3 search bar: a fully rounded field with a search icon,
    and a button that clears it.
    """
    field = ft.TextField(
        value=value,
        hint_text=hint,
        border=ft.NoInputBorder(),
        prefix_icon=icon("search"),
        content_padding=ft.Padding.symmetric(horizontal=Gaps.L, vertical=Gaps.L),
        on_change=lambda event: on_change(event.control.value),
        key=key,
        expand=True,
    )
    row: list[ft.Control] = [field]
    if value and clear_tooltip is not None:
        row.append(icon_button("close", clear_tooltip, lambda: on_change(""), compact=True))
    return ConstrainedBox(
        max_width=720,
        content=ft.Container(
            content=ft.Row(row, spacing=0),
            bgcolor=C.SURFACE_CONTAINER_HIGH,
            border_radius=Corners.FULL,
            height=56,
            padding=ft.Padding.only(left=Gaps.XS, right=Gaps.S),
        ),
    )
