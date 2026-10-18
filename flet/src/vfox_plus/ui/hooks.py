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

"""Hooks that connect Flet components to the state layer."""

from collections.abc import Callable

import flet as ft

from vfox_plus.state.notifier import Listenable, Unsubscribe, listen_all


def use_listenable(*listenables: Listenable | None) -> None:
    """Renders the calling component again whenever one of `listenables`
    notifies its listeners.

    The subscription is made while rendering, not in an effect, which Flet
    runs only after the render: a change in between would otherwise be lost.
    """
    _, set_version = ft.use_state(0)
    subscription: ft.MutableRef[tuple[tuple[int, ...], Unsubscribe]] = ft.use_ref()
    present = [listenable for listenable in listenables if listenable is not None]
    identity = tuple(id(listenable) for listenable in present)
    current = subscription.current
    if current is None or current[0] != identity:
        if current is not None:
            current[1]()
        subscription.current = (identity, listen_all(present, lambda: set_version(lambda version: version + 1)))

    def unsubscribe() -> None:
        if subscription.current is not None:
            subscription.current[1]()
            subscription.current = None

    ft.use_effect(lambda: None, [], cleanup=unsubscribe)


type SizeHandler = Callable[[ft.LayoutSizeChangeEvent[ft.LayoutControl]], None]


def use_width() -> tuple[float | None, SizeHandler]:
    """The width of a control, and the handler to give to its
    `on_size_change`: the counterpart of Flutter's `LayoutBuilder`. The width
    is `None` until the control has been laid out once.
    """
    initial: float | None = None
    width, set_width = ft.use_state(initial)

    def on_size_change(event: ft.LayoutSizeChangeEvent[ft.LayoutControl]) -> None:
        set_width(event.width)

    return width, on_size_change
