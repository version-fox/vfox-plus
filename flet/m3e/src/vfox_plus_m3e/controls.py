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

"""The controls of the extension.

Each control is rendered by the widget of the same name in the Flutter
package `src/flutter/vfox_plus_m3e` (see `lib/src/extension.dart`).
"""

from collections.abc import Callable
from dataclasses import field
from typing import cast, dataclass_transform

import flet as ft


@dataclass_transform()
def control[C: type[ft.BaseControl]](name: str) -> Callable[[C], C]:
    """`ft.control(name)`: declares a control rendered by the Flutter widget
    `name`.

    Flet's overload of `ft.control` with a name gives mypy no way to infer the
    decorated class, hence this typed alias.
    """
    return cast("Callable[[C], C]", ft.control(name))


SYMBOLS_SET_ID = 0x56
"""The icon set ID of Material Symbols Rounded (see `Extension` in Dart)."""

FULL = 9999
"""A corner radius that rounds a corner fully: half of the shorter side."""


class Symbol(ft.IconData, package_name="vfox_plus_m3e", class_name="Symbol"):
    """An icon of the Material Symbols Rounded font that the extension bundles.

    Flet encodes icons as `set << 16 | index`; the Flutter side of the
    extension maps set 0x56 to the font, with the code point as index.
    Create symbols with :meth:`of`.
    """

    # An enumeration without members cannot be called; symbols are created on
    # demand by `_missing_`, as Flet does for its Material icons.
    _PLACEHOLDER = -1

    @classmethod
    def _missing_(cls, value: object) -> Symbol:
        if not isinstance(value, int) or value >> 16 != SYMBOLS_SET_ID:
            raise ValueError(f"{value!r} is not a Material Symbols icon code")
        member = int.__new__(cls, value)
        member._value_ = value
        member._name_ = f"U{value & 0xFFFF:04X}"
        cls._value2member_map_[value] = member
        return member

    @classmethod
    def of(cls, codepoint: int) -> Symbol:
        """The symbol with the code point `codepoint`."""
        return cls(SYMBOLS_SET_ID << 16 | codepoint)


@control("LoadingIndicator")
class LoadingIndicator(ft.LayoutControl):
    """The Material 3 Expressive loading indicator: a shape that morphs
    through Material shapes while it turns.

    See https://m3.material.io/components/loading-indicator.
    """

    size: ft.Number = 48
    """Width and height, including the container."""

    contained: bool = False
    """Whether to draw the indicator on a primary container circle."""

    color: ft.ColorValue | None = None
    """Colour of the shape; primary, or on-primary-container when contained."""

    semantics_label: str | None = None
    """Accessible label."""


@control("WavyProgressIndicator")
class WavyProgressIndicator(ft.LayoutControl):
    """The Material 3 Expressive linear progress indicator with a wavy
    active track: determinate with a `value`, indeterminate without.

    See https://m3.material.io/components/progress-indicators.
    """

    value: ft.Number | None = None
    """Progress between 0 and 1, or `None` while unknown."""

    color: ft.ColorValue | None = None
    """Colour of the active indicator; primary by default."""

    track_color: ft.ColorValue | None = None
    """Colour of the track; secondary container by default."""

    semantics_label: str | None = None
    """Accessible label."""


@control("MaterialShape")
class MaterialShape(ft.LayoutControl):
    """`content` centred on a shape of the Material shape library, such as
    `cookie9Sided` or `softBurst`.

    See https://m3.material.io/styles/shape/overview-principles.
    """

    shape: str = "circle"
    """The name of the shape in `MaterialShapes` of `material_new_shapes`."""

    size: ft.Number = 48
    """Width and height."""

    bgcolor: ft.ColorValue | None = None
    """Colour of the shape; secondary container by default."""

    angle: ft.Number = 0
    """Rotation of the shape, in degrees."""

    hover_angle: ft.Number = 0
    """Rotation added on a spring while the pointer is over the shape, in
    degrees."""

    content: ft.Control | None = None
    """The control on the shape, usually an icon."""


@control("ShapeAvatar")
class ShapeAvatar(ft.LayoutControl):
    """A plugin avatar: the initial of `name` on a decorative shape whose form
    and colours are derived from the name. Hovering spins the shape.
    """

    name: str = ""
    """The plugin name."""

    size: ft.Number = 40
    """Width and height."""

    icon: ft.IconData | None = None
    """An icon to show instead of the initial."""


@control("ColorSwatch")
class ColorSwatch(ft.LayoutControl):
    """A seed colour to choose. The selected swatch morphs from a circle into
    a flower and shows `check_icon`.
    """

    seed: ft.ColorValue = "#F28C1B"
    """The seed colour; the swatch shows the primary colour of its scheme."""

    selected: bool = False
    """Whether this colour is the chosen one."""

    label: str = ""
    """Accessible name and tooltip."""

    check_icon: ft.IconData | None = None
    """The icon shown on the selected swatch."""

    on_click: ft.ControlEventHandler[ColorSwatch] | None = None
    """Called when the swatch is chosen."""


@control("SpringSurface")
class SpringSurface(ft.LayoutControl):
    """An interactive surface whose corners morph on a spring as it is
    hovered, selected and pressed: the shape morphing of Material 3
    Expressive.

    A state without its own radius has the radius of the state before it, in
    the order rest, hovered, selected, pressed. :data:`FULL` rounds a corner
    fully.
    """

    content: ft.Control | None = None
    """The content of the surface."""

    bgcolor: ft.ColorValue | None = None
    """Colour of the surface, animated when it changes."""

    radius: ft.BorderRadiusValue | None = None
    """Corners at rest."""

    hover_radius: ft.BorderRadiusValue | None = None
    """Corners while hovered."""

    selected_radius: ft.BorderRadiusValue | None = None
    """Corners while selected."""

    pressed_radius: ft.BorderRadiusValue | None = None
    """Corners while pressed."""

    selected: bool = False
    """Whether the surface is selected."""

    interactive: bool = True
    """Whether the surface reacts to the pointer and the keyboard."""

    spring: str | None = None
    """The spring of the corners, such as `fast_spatial` (the default)."""

    semantics_button: bool = True
    """Whether an interactive surface is announced as a button."""

    semantics_selected: bool | None = None
    """The selected state announced to assistive technologies; by default
    :attr:`selected` when it is set."""

    mutually_exclusive: bool | None = None
    """Whether the surface is one choice of a group with single selection."""

    on_click: ft.ControlEventHandler[SpringSurface] | None = None
    """Called when the surface is activated."""


@control("ConstrainedBox")
class ConstrainedBox(ft.LayoutControl):
    """Limits the size of `content`, aligned by `alignment` in the space
    left: Flet controls have no maximum size.
    """

    content: ft.Control | None = None
    """The content."""

    min_width: ft.Number | None = None
    max_width: ft.Number | None = None
    min_height: ft.Number | None = None
    max_height: ft.Number | None = None

    alignment: ft.Alignment | None = None
    """Where the content sits when it is smaller than the space."""


@control("TransitionSwitcher")
class TransitionSwitcher(ft.LayoutControl):
    """Replaces its content with a Material transition whenever
    `content_key` changes: `fade_through` or `shared_axis_x`.
    """

    content: ft.Control | None = None
    """The current content."""

    content_key: str = ""
    """Identifies the content; a new key starts a transition."""

    transition: str = "fade_through"
    """`fade_through` or `shared_axis_x`."""

    reverse: bool = False
    """Whether a shared axis transition goes backwards."""


@control("ExpressiveDialog")
class ExpressiveDialog(ft.DialogControl):
    """An alert dialog that scales in on a spring. Open it with
    `page.show_dialog()`, like Flet's `AlertDialog`.
    """

    title: ft.StrOrControl | None = None
    """The title."""

    content: ft.Control | None = None
    """The content below the title."""

    actions: list[ft.Control] = field(default_factory=list)
    """The buttons at the bottom."""

    icon: ft.IconData | None = None
    """An icon above the title."""

    icon_color: ft.ColorValue | None = None
    """Colour of the icon."""

    modal: bool = False
    """Whether tapping outside of the dialog leaves it open."""

    scrollable: bool = False
    """Whether the content scrolls when it is too tall."""

    semantics_label: str | None = None
    """Accessible label of the dialog."""


@control("RailDestination")
class RailDestination(ft.Control):
    """A destination of an :class:`ExpressiveNavigationRail`."""

    icon: ft.IconData | None = None
    """The icon, outlined when unselected and filled when selected."""

    label: str = ""
    """The label."""

    badge_count: int = 0
    """A number shown in a badge, if greater than zero."""


@control("ExpressiveNavigationRail")
class ExpressiveNavigationRail(ft.LayoutControl):
    """The Material 3 Expressive navigation rail, collapsed (96 dp, labels
    below the icons) or expanded (240 dp, labels next to them), with a menu
    button and a floating action button.

    See https://m3.material.io/components/navigation-rail.
    """

    destinations: list[RailDestination] = field(default_factory=list)
    """The destinations, from top to bottom."""

    selected_index: int = 0
    """The index of the selected destination."""

    expanded: bool = False
    """Whether the labels are next to the icons."""

    expand_icon: ft.IconData | None = None
    """Icon of the menu button while collapsed."""

    collapse_icon: ft.IconData | None = None
    """Icon of the menu button while expanded."""

    expand_tooltip: str = ""
    """Tooltip of the menu button while collapsed."""

    collapse_tooltip: str = ""
    """Tooltip of the menu button while expanded."""

    fab_icon: ft.IconData | None = None
    """Icon of the floating action button; without one there is none."""

    fab_label: str = ""
    """Label of the floating action button, shown when expanded."""

    fab_disabled: bool = False
    """Whether the floating action button is disabled."""

    on_change: ft.ControlEventHandler[ExpressiveNavigationRail] | None = None
    """Called with the index of the chosen destination as `data`."""

    on_toggle: ft.ControlEventHandler[ExpressiveNavigationRail] | None = None
    """Called with the requested expansion as `data`."""

    on_fab_click: ft.ControlEventHandler[ExpressiveNavigationRail] | None = None
    """Called when the floating action button is pressed."""
