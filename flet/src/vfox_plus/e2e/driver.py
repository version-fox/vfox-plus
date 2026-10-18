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

"""Drives the user interface as a user would: by the texts, tooltips and
keys of what the page shows.

The driver walks the tree of Flet controls of the page, the same tree that
the Flutter client draws, and sends events to them through the session, as
the client does when the user clicks or types. It serves the tests of the
user interface, which run the application in the test process, and the
end-to-end scenarios that the built application runs on itself.
"""

import asyncio
import dataclasses
import time
from collections.abc import Callable, Iterator
from typing import Any, cast

import flet as ft
from flet.components.component import Component

type Predicate = Callable[[ft.BaseControl], bool]


class DriverError(AssertionError):
    """What the driver looked for is not there."""


def _children(control: ft.BaseControl) -> Iterator[ft.BaseControl]:
    if isinstance(control, Component):
        yield from _values(control._b)
        return
    for field in dataclasses.fields(control):
        if field.name.startswith("_") or field.name in ("parent", "page", "data"):
            continue
        yield from _values(getattr(control, field.name, None))


def _values(value: Any) -> Iterator[ft.BaseControl]:
    if isinstance(value, ft.BaseControl):
        yield value
    elif isinstance(value, (list, tuple)):
        for item in value:
            if isinstance(item, ft.BaseControl):
                yield item


def _visible(control: ft.BaseControl) -> bool:
    return getattr(control, "visible", True) is not False


def text_of(control: ft.BaseControl) -> str | None:
    """The text a control shows by itself, if any."""
    if isinstance(control, ft.Text):
        return control.value
    if isinstance(control, (ft.Button, ft.FilledButton, ft.FilledTonalButton, ft.OutlinedButton, ft.TextButton)):
        content = control.content
        return content if isinstance(content, str) else None
    return None


def tooltip_of(control: ft.BaseControl) -> str | None:
    """The tooltip of a control, which is also its accessible name."""
    tooltip = getattr(control, "tooltip", None)
    if isinstance(tooltip, str):
        return tooltip
    if isinstance(tooltip, ft.Tooltip):
        return tooltip.message if isinstance(tooltip.message, str) else None
    return None


class Driver:
    """Finds controls on `page` and acts on them."""

    def __init__(self, page: ft.Page, timeout: float = 15) -> None:
        self.page = page
        self.timeout = timeout

    # Finding

    def roots(self) -> list[ft.BaseControl]:
        """The views of the page and the open dialogs, top first."""
        dialogs = [dialog for dialog in reversed(self.page._dialogs.controls) if dialog.open]
        return [*dialogs, *self.page.views]

    def walk(self) -> Iterator[ft.BaseControl]:
        """Every visible control, the open dialogs first."""
        stack = list(reversed(self.roots()))
        seen: set[int] = set()
        while stack:
            control = stack.pop()
            if id(control) in seen or not _visible(control):
                continue
            seen.add(id(control))
            yield control
            stack.extend(reversed(list(_children(control))))

    def find_all(self, predicate: Predicate) -> list[ft.BaseControl]:
        """Every visible control that matches `predicate`."""
        return [control for control in self.walk() if predicate(control)]

    def find(self, predicate: Predicate) -> ft.BaseControl | None:
        """The first visible control that matches `predicate`."""
        return next((control for control in self.walk() if predicate(control)), None)

    def texts(self) -> list[str]:
        """Every text on the page, in order."""
        return [text for control in self.walk() if (text := text_of(control)) is not None]

    def has_text(self, text: str, *, exact: bool = True) -> bool:
        """Whether the page shows `text`, or a text containing it."""
        return any(text == shown or (not exact and text in shown) for shown in self.texts())

    @staticmethod
    def by_text(text: str, *, exact: bool = True) -> Predicate:
        """Matches controls that show `text`, or a text containing it."""

        def matches(control: ft.BaseControl) -> bool:
            shown = text_of(control)
            return shown is not None and (shown == text or (not exact and text in shown))

        return matches

    @staticmethod
    def by_tooltip(tooltip: str) -> Predicate:
        """Matches controls whose tooltip is `tooltip`."""
        return lambda control: tooltip_of(control) == tooltip

    @staticmethod
    def by_key(key: str) -> Predicate:
        """Matches controls whose key is `key`."""
        return lambda control: getattr(control, "key", None) == key

    # Waiting

    async def settle(self, rounds: int = 20) -> None:
        """Lets the pending events, renders and effects run."""
        for _ in range(rounds):
            await asyncio.sleep(0)
        await asyncio.sleep(0.01)
        for _ in range(rounds):
            await asyncio.sleep(0)

    async def wait_until(self, condition: Callable[[], bool], description: str, timeout: float | None = None) -> None:
        """Waits until `condition` holds."""
        deadline = time.monotonic() + (timeout or self.timeout)
        while not condition():
            if time.monotonic() > deadline:
                raise DriverError(f"Timed out waiting for {description}; the page shows: {self.texts()}")
            await asyncio.sleep(0.05)

    async def wait_for(self, predicate: Predicate, description: str, timeout: float | None = None) -> ft.BaseControl:
        """Waits for a visible control that matches `predicate`."""
        found: list[ft.BaseControl] = []

        def present() -> bool:
            control = self.find(predicate)
            if control is None:
                return False
            found.append(control)
            return True

        await self.wait_until(present, description, timeout)
        return found[-1]

    async def wait_for_text(self, text: str, *, exact: bool = True, timeout: float | None = None) -> ft.BaseControl:
        """Waits until the page shows `text`."""
        return await self.wait_for(self.by_text(text, exact=exact), f'the text "{text}"', timeout)

    async def wait_for_tooltip(self, tooltip: str, timeout: float | None = None) -> ft.BaseControl:
        """Waits for a control whose tooltip is `tooltip`."""
        return await self.wait_for(self.by_tooltip(tooltip), f'the tooltip "{tooltip}"', timeout)

    async def wait_for_key(self, key: str, timeout: float | None = None) -> ft.BaseControl:
        """Waits for the control with `key`."""
        return await self.wait_for(self.by_key(key), f'the key "{key}"', timeout)

    async def wait_until_gone(self, text: str, timeout: float | None = None) -> None:
        """Waits until the page no longer shows `text`."""
        await self.wait_until(lambda: not self.has_text(text), f'"{text}" to disappear', timeout)

    # Acting

    async def send(self, control: ft.BaseControl, event: str, data: Any = None) -> None:
        """Sends the event `event` to `control`, as the client does."""
        await self.page.session.dispatch_event(control._i, event, data)
        await self.settle()

    def _clickable(self, control: ft.BaseControl) -> ft.BaseControl:
        """`control`, or its closest ancestor that reacts to clicks."""
        current: ft.BaseControl | None = control
        while current is not None:
            if getattr(current, "on_click", None) is not None and not getattr(current, "disabled", False):
                return current
            current = current.parent
        raise DriverError(f"Nothing clickable around {control}")

    async def click(self, control: ft.BaseControl) -> None:
        """Clicks `control`, or the closest clickable control around it."""
        await self.send(self._clickable(control), "click")

    async def tap_text(self, text: str, *, exact: bool = True) -> None:
        """Clicks what shows `text`."""
        await self.click(await self.wait_for_text(text, exact=exact))

    async def tap_tooltip(self, tooltip: str) -> None:
        """Clicks the control whose tooltip is `tooltip`."""
        control = await self.wait_for_tooltip(tooltip)
        if isinstance(control, ft.PopupMenuButton):
            raise DriverError(f'"{tooltip}" opens a menu; use choose_menu_item()')
        await self.click(control)

    async def tap_key(self, key: str) -> None:
        """Clicks the control with `key`."""
        await self.click(await self.wait_for_key(key))

    async def choose_menu_item(self, menu: ft.BaseControl, label: str) -> None:
        """Opens `menu`, a popup menu button or a context menu, and chooses
        the item that shows `label`.
        """
        items = cast("list[ft.PopupMenuItem]", getattr(menu, "items", []))
        for item in items:
            if any(text_of(control) == label for control in self._subtree(item)):
                if isinstance(menu, ft.ContextMenu):
                    await self.send(menu, "select", {"id": item._i, "idx": items.index(item), "g": {"x": 0, "y": 0}})
                await self.send(item, "click")
                return
        raise DriverError(f'The menu has no item "{label}"')

    async def choose_from_menu(self, tooltip: str, label: str) -> None:
        """Opens the menu of the button with `tooltip` and chooses `label`."""
        control = await self.wait_for_tooltip(tooltip)
        current: ft.BaseControl | None = control
        while current is not None and not isinstance(current, (ft.PopupMenuButton, ft.ContextMenu)):
            current = current.parent
        if current is None:
            raise DriverError(f'"{tooltip}" does not open a menu')
        await self.choose_menu_item(current, label)

    async def enter_text(self, field: ft.BaseControl, value: str, *, submit: bool = False) -> None:
        """Types `value` into the text field `field`, replacing its text."""
        if not isinstance(field, ft.TextField):
            raise DriverError(f"{field} is not a text field")
        # The declarative renderer freezes controls; the client changes them
        # through a property patch.
        self.page.session.apply_patch(field._i, {"value": value})
        await self.send(field, "change", value)
        if submit:
            await self.send(field, "submit", value)

    async def enter_text_by_key(self, key: str, value: str, *, submit: bool = False) -> None:
        """Types `value` into the text field with `key`."""
        await self.enter_text(await self.wait_for_key(key), value, submit=submit)

    async def leave(self, field: ft.BaseControl) -> None:
        """Moves the focus out of `field`."""
        await self.send(field, "blur")

    def _subtree(self, control: ft.BaseControl) -> Iterator[ft.BaseControl]:
        """`control` and every control below it, in order."""
        stack = [control]
        seen: set[int] = set()
        while stack:
            current = stack.pop()
            if id(current) in seen:
                continue
            seen.add(id(current))
            yield current
            stack.extend(reversed(list(_children(current))))
