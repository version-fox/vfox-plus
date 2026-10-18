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

"""Observable state, the counterpart of Flutter's `ChangeNotifier`, and
background tasks.
"""

import asyncio
import logging
from collections.abc import Callable, Coroutine, Sequence
from typing import Any, Protocol

type Listener = Callable[[], None]
"""Called when an observed object changed."""

type Unsubscribe = Callable[[], None]
"""Stops a listener."""

_logger = logging.getLogger(__name__)
_background: set[asyncio.Task[Any]] = set()


class Listenable(Protocol):
    """Something whose changes can be observed."""

    def add_listener(self, listener: Listener) -> Unsubscribe:
        """Calls `listener` after every change; returns a function that stops
        it.
        """
        ...


class ChangeNotifier:
    """A :class:`Listenable` that notifies its listeners when told to."""

    def __init__(self) -> None:
        self._listeners: list[Listener] = []
        self._disposed = False

    def add_listener(self, listener: Listener) -> Unsubscribe:
        self._listeners.append(listener)

        def remove() -> None:
            if listener in self._listeners:
                self._listeners.remove(listener)

        return remove

    @property
    def is_disposed(self) -> bool:
        """Whether the object was disposed."""
        return self._disposed

    def notify_listeners(self) -> None:
        """Calls every listener. Does nothing once disposed."""
        if self._disposed:
            return
        for listener in list(self._listeners):
            listener()

    def dispose(self) -> None:
        """Releases the listeners; the object must not be used afterwards."""
        self._disposed = True
        self._listeners.clear()


def listen_all(listenables: Sequence[Listenable], listener: Listener) -> Unsubscribe:
    """Listens to several objects at once; returns a function that stops it."""
    stops = [listenable.add_listener(listener) for listenable in listenables]

    def stop() -> None:
        for each in stops:
            each()

    return stop


class Emitter[T]:
    """A simple event source for values."""

    def __init__(self) -> None:
        self._listeners: list[Callable[[T], None]] = []

    def listen(self, listener: Callable[[T], None]) -> Unsubscribe:
        """Calls `listener` with every emitted value; returns a function that
        stops it.
        """
        self._listeners.append(listener)

        def remove() -> None:
            if listener in self._listeners:
                self._listeners.remove(listener)

        return remove

    def emit(self, value: T) -> None:
        """Emits `value` to every listener."""
        for listener in list(self._listeners):
            listener(value)

    def close(self) -> None:
        """Removes every listener."""
        self._listeners.clear()


def spawn[T](coroutine: Coroutine[Any, Any, T]) -> asyncio.Task[T]:
    """Runs `coroutine` in the background.

    The event loop keeps only weak references to tasks, so the task is kept
    until it ends; an exception that nobody awaits is logged instead of being
    lost.
    """
    task = asyncio.get_running_loop().create_task(coroutine)
    _background.add(task)
    task.add_done_callback(_finished)
    return task


def _finished(task: asyncio.Task[Any]) -> None:
    _background.discard(task)
    if not task.cancelled() and (error := task.exception()) is not None:
        _logger.error("Background task failed", exc_info=error)
