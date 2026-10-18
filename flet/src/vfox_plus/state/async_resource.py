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

"""A value that is loaded asynchronously and can be reloaded."""

import asyncio
from collections.abc import Awaitable, Callable

from vfox_plus.state.notifier import ChangeNotifier, spawn


class AsyncResource[T](ChangeNotifier):
    """A value that is loaded asynchronously and can be reloaded.

    While a reload is in flight the previous :attr:`value` stays available,
    so the user interface can keep showing stale data with a progress
    indicator instead of flashing an empty state. When loads overlap, only the
    latest one wins.
    """

    def __init__(self, loader: Callable[[], Awaitable[T]]) -> None:
        super().__init__()
        self._loader = loader
        self._value: T | None = None
        self._loaded = False
        self._error: BaseException | None = None
        self._loading = False
        self._generation = 0
        self._in_flight: asyncio.Task[None] | None = None

    @property
    def value(self) -> T | None:
        """The last successfully loaded value."""
        return self._value

    @property
    def has_value(self) -> bool:
        """Whether a value was loaded at least once."""
        return self._loaded

    @property
    def error(self) -> BaseException | None:
        """The error of the last load, or `None` if it succeeded."""
        return self._error

    @property
    def is_loading(self) -> bool:
        """Whether a load is in progress."""
        return self._loading

    def load(self) -> asyncio.Task[None]:
        """Loads the value, replacing any load that is still in progress.

        Returns the load, which may be awaited.
        """
        self._generation += 1
        self._loading = True
        self.notify_listeners()
        self._in_flight = spawn(self._run(self._generation))
        return self._in_flight

    async def ensure_loaded(self) -> None:
        """Loads the value unless it is already loaded or loading."""
        if self._loading and self._in_flight is not None:
            await self._in_flight
        elif not self._loaded:
            await self.load()

    def set_value(self, value: T) -> None:
        """Replaces the value without loading, for example after an
        optimistic update.
        """
        self._generation += 1
        self._value = value
        self._loaded = True
        self._error = None
        self._loading = False
        self.notify_listeners()

    async def _run(self, generation: int) -> None:
        try:
            value = await self._loader()
        except Exception as error:  # noqa: BLE001 - any failure is shown to the user.
            if generation != self._generation:
                return
            self._error = error
        else:
            if generation != self._generation:
                return
            self._value = value
            self._loaded = True
            self._error = None
        self._loading = False
        self.notify_listeners()
