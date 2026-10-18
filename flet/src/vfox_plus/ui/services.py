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

"""Desktop integration that the user interface needs: the browser, file
choosers, the clipboard and the accent colour, behind a small interface that
tests replace (see `docs/architecture.md#layers`).
"""

import asyncio
import contextlib
import re
from collections.abc import Sequence
from typing import Protocol

import flet as ft


class PlatformServices(Protocol):
    """Desktop integration of the user interface."""

    async def accent_color(self) -> int | None:
        """The accent colour of the desktop as ARGB, if it exposes one."""
        ...

    async def open_url(self, url: str) -> None:
        """Opens `url` in the default browser."""
        ...

    async def pick_directory(self, initial_directory: str | None = None) -> str | None:
        """Lets the user choose a directory."""
        ...

    async def pick_file(self, initial_directory: str | None, extensions: Sequence[str] = ()) -> str | None:
        """Lets the user choose a file, with one of `extensions` if there are
        any.
        """
        ...

    async def copy_text(self, text: str) -> None:
        """Copies `text` to the clipboard."""
        ...


_ACCENT = re.compile(r"\(\s*([^\s(),<>]+)\s*,\s*([^\s(),<>]+)\s*,\s*([^\s(),<>]+)\s*\)")


def parse_accent_color(reply: str) -> int | None:
    """Converts the reply of gdbus, such as `(<(0.2, 0.4, 0.6)>,)`, to ARGB.

    The portal reports components outside of 0 to 1 when no accent colour is set.
    """
    match = _ACCENT.search(reply)
    if match is None:
        return None
    try:
        channels = [float(channel) for channel in match.groups()]
    except ValueError:
        return None
    if any(not 0 <= channel <= 1 for channel in channels):
        return None
    red, green, blue = (round(channel * 255) for channel in channels)
    return 0xFF000000 | red << 16 | green << 8 | blue


async def read_accent_color() -> int | None:
    """The accent colour of the desktop from the XDG desktop portal
    (`org.freedesktop.appearance` `accent-color`), read with `gdbus` of GLib
    as the Capacitor implementation does; Flutter does not expose it.
    """
    try:
        process = await asyncio.create_subprocess_exec(
            "gdbus",
            "call",
            "--session",
            "--timeout",
            "2",
            "--dest",
            "org.freedesktop.portal.Desktop",
            "--object-path",
            "/org/freedesktop/portal/desktop",
            "--method",
            "org.freedesktop.portal.Settings.ReadOne",
            "org.freedesktop.appearance",
            "accent-color",
            stdin=asyncio.subprocess.DEVNULL,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.DEVNULL,
        )
        stdout, _ = await asyncio.wait_for(process.communicate(), 3)
    except OSError, TimeoutError:
        # No portal, no session bus or no gdbus: the seed colour of the
        # settings applies.
        return None
    return parse_accent_color(stdout.decode(errors="replace")) if process.returncode == 0 else None


class FletPlatformServices:
    """The desktop integration with Flet's services."""

    def __init__(self) -> None:
        self._files = ft.FilePicker()
        self._clipboard = ft.Clipboard()
        self._urls = ft.UrlLauncher()

    async def accent_color(self) -> int | None:
        return await read_accent_color()

    async def open_url(self, url: str) -> None:
        await self._urls.launch_url(url)

    async def pick_directory(self, initial_directory: str | None = None) -> str | None:
        return await self._files.get_directory_path(initial_directory=initial_directory)

    async def pick_file(self, initial_directory: str | None, extensions: Sequence[str] = ()) -> str | None:
        # The portal labels the filter of the extensions "Custom"; Flet's file
        # picker does not let it be named.
        files = await self._files.pick_files(
            initial_directory=initial_directory,
            file_type=ft.FilePickerFileType.CUSTOM if extensions else ft.FilePickerFileType.ANY,
            allowed_extensions=list(extensions) or None,
        )
        with contextlib.suppress(IndexError):
            return files[0].path
        return None

    async def copy_text(self, text: str) -> None:
        await self._clipboard.set(text)
