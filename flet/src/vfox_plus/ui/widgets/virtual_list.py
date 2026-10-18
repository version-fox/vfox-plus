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

"""A scrolling list that builds only the rows near the visible part.

Flet builds the Flutter widgets of a list lazily, but every row is a tree of
Python controls that is created, sent and compared on every render. The
version list of Node.js alone has close to 900 rows, so this list keeps only
the rows within :data:`BUFFER` pixels of the viewport and stands in for the
others with two spacers, as the other implementations do with their
virtualizers. Rows have fixed heights, which keeps the spacers exact.
"""

import bisect
import itertools
from collections.abc import Callable, Sequence
from dataclasses import dataclass

import flet as ft

BUFFER = 800
"""Pixels above and below the viewport whose rows are built."""

STEP = 200
"""The scroll distance that builds the rows again."""


@dataclass(frozen=True)
class Row:
    """A row of a :func:`VirtualList`."""

    key: str
    """Identifies the row across renders."""
    height: float
    """The height of the row, in logical pixels."""
    build: Callable[[], ft.Control]
    """Builds the content of the row."""


def visible_range(tops: Sequence[float], heights: Sequence[float], start: float, end: float) -> tuple[int, int]:
    """The rows between the offsets `start` and `end` of the list, as a
    half-open range of indexes, given the top and the height of each row.
    """
    first = max(0, bisect.bisect_right(tops, start) - 1)
    while first < len(tops) and tops[first] + heights[first] < start:
        first += 1
    last = bisect.bisect_right(tops, end)
    return first, max(first, last)


@ft.component
def VirtualList(
    header: Sequence[ft.Control],
    rows: Sequence[Row],
    *,
    spacing: float = 0,
    padding: ft.Padding | None = None,
    footer: Sequence[ft.Control] = (),
    key: str | None = None,
) -> ft.Control:
    """The `header` controls followed by `rows`, `spacing` apart, in one
    scrolling list.
    """
    del key
    initial: tuple[float, float, float] = (0, 1000, 0)
    window, set_window = ft.use_state(initial)
    offset, viewport, header_height = window
    tops = list(itertools.accumulate((row.height + spacing for row in rows), initial=0.0))
    total = tops[-1] - spacing if rows else 0
    tops = tops[:-1]
    heights = [row.height for row in rows]
    first, last = visible_range(
        tops, heights, offset - header_height - BUFFER, offset - header_height + viewport + BUFFER
    )

    def on_scroll(event: ft.OnScrollEvent) -> None:
        # Building again for every pixel would flood the connection.
        stepped = round(event.pixels / STEP) * STEP
        set_window(lambda current: (stepped, event.viewport_dimension, current[2]))

    def on_header_size(event: ft.LayoutSizeChangeEvent[ft.LayoutControl]) -> None:
        set_window(lambda current: (current[0], current[1], event.height))

    controls: list[ft.Control] = [
        ft.Container(
            content=ft.Column(list(header), spacing=0), on_size_change=on_header_size, size_change_interval=100
        )
    ]
    if first > 0:
        controls.append(ft.Container(height=tops[first], key="virtual-top"))
    for index in range(first, last):
        row = rows[index]
        controls.append(
            ft.Container(
                content=row.build(),
                height=row.height,
                margin=ft.Margin.only(bottom=spacing if index < len(rows) - 1 else 0),
                key=row.key,
            )
        )
    if last < len(rows):
        controls.append(ft.Container(height=total - tops[last], key="virtual-bottom"))
    controls.extend(footer)
    return ft.ListView(
        controls,
        padding=padding,
        on_scroll=on_scroll,
        scroll_interval=100,
        expand=True,
    )
