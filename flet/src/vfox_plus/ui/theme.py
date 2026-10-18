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

"""The Material 3 Expressive theme of the application.

Colours come from Flutter's `ColorScheme.fromSeed`, the tonal spot variant
of the 2021 specification, as in every implementation (see
`docs/design-system.md#colour`); the component themes follow `AppTheme` of the
Flutter implementation.
"""

from typing import Any

import flet as ft

from vfox_plus.ui import motion
from vfox_plus.ui.tokens import Corners, Gaps

C = ft.Colors


def color_hex(argb: int) -> str:
    """An ARGB colour as Flet's `#AARRGGBB`."""
    return f"#{argb & 0xFFFFFFFF:08X}"


def button_style(**overrides: Any) -> ft.ButtonStyle:
    """The common buttons of Material 3 Expressive: round at rest, squarer
    while pressed, the shape animating as long as the fast effects spring
    needs to settle.
    """
    fields: dict[str, Any] = {
        "shape": {
            ft.ControlState.PRESSED: ft.RoundedRectangleBorder(radius=Corners.SMALL),
            ft.ControlState.DEFAULT: ft.StadiumBorder(),
        },
        "animation_duration": motion.FAST_EFFECTS.settle_duration,
        "padding": ft.Padding.symmetric(horizontal=Gaps.L),
        "icon_size": 20,
    }
    return ft.ButtonStyle(**(fields | overrides))


def icon_button_style(**overrides: Any) -> ft.ButtonStyle:
    """Icon buttons, which squeeze from a circle into a rounded square while
    pressed.
    """
    fields: dict[str, Any] = {
        "shape": {
            ft.ControlState.PRESSED: ft.RoundedRectangleBorder(radius=Corners.MEDIUM),
            ft.ControlState.DEFAULT: ft.CircleBorder(),
        },
        "animation_duration": motion.FAST_EFFECTS.settle_duration,
    }
    return ft.ButtonStyle(**(fields | overrides))


def build_theme(seed: int) -> ft.Theme:
    """The theme for the ARGB `seed`, for light and dark mode alike: Flet
    derives the colour scheme of each brightness from the seed.
    """
    medium = ft.RoundedRectangleBorder(radius=Corners.MEDIUM)
    large = ft.RoundedRectangleBorder(radius=Corners.LARGE)
    return ft.Theme(
        color_scheme_seed=color_hex(seed),
        visual_density=ft.VisualDensity.STANDARD,
        scaffold_bgcolor=C.SURFACE,
        icon_theme=ft.IconTheme(color=C.ON_SURFACE_VARIANT, size=24),
        button_theme=ft.ButtonTheme(style=button_style()),
        filled_button_theme=ft.FilledButtonTheme(style=button_style()),
        outlined_button_theme=ft.OutlinedButtonTheme(style=button_style()),
        text_button_theme=ft.TextButtonTheme(style=button_style()),
        icon_button_theme=ft.IconButtonTheme(style=icon_button_style()),
        floating_action_button_theme=ft.FloatingActionButtonTheme(
            bgcolor=C.PRIMARY_CONTAINER,
            foreground_color=C.ON_PRIMARY_CONTAINER,
            elevation=3,
            focus_elevation=3,
            hover_elevation=4,
            highlight_elevation=3,
            shape=large,
        ),
        card_theme=ft.CardTheme(
            elevation=0, margin=0, color=C.SURFACE_CONTAINER_LOW, clip_behavior=ft.ClipBehavior.ANTI_ALIAS, shape=large
        ),
        dialog_theme=ft.DialogTheme(
            bgcolor=C.SURFACE_CONTAINER_HIGH, shape=ft.RoundedRectangleBorder(radius=Corners.EXTRA_LARGE)
        ),
        popup_menu_theme=ft.PopupMenuTheme(color=C.SURFACE_CONTAINER, shape=large, elevation=2),
        navigation_bar_theme=ft.NavigationBarTheme(
            height=64, bgcolor=C.SURFACE_CONTAINER, indicator_color=C.SECONDARY_CONTAINER
        ),
        snackbar_theme=ft.SnackBarTheme(
            behavior=ft.SnackBarBehavior.FLOATING,
            bgcolor=C.INVERSE_SURFACE,
            action_text_color=C.INVERSE_PRIMARY,
            shape=medium,
            width=480,
        ),
        chip_theme=ft.ChipTheme(shape=ft.RoundedRectangleBorder(radius=Corners.SMALL)),
        badge_theme=ft.BadgeTheme(bgcolor=C.ERROR, text_color=C.ON_ERROR),
        tooltip_theme=ft.TooltipTheme(wait_duration=500),
        divider_theme=ft.DividerTheme(color=C.OUTLINE_VARIANT, space=1, thickness=1),
    )
