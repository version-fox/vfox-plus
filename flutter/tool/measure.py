#!/usr/bin/env python3
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

"""Measures the memory that the release build of the Flutter implementation uses.

shared/tools/measure.sh measures the other implementations of vfox+ the same
way: the application runs its "measure" scenario with a new, empty HOME and the
fake vfox of shared/fixtures, and the script adds up the resident (RSS) and
proportional (PSS) set size of the application and its child processes on the
Home page and with the available versions of Node.js listed. The Flutter
implementation has no such scenario: its pages keep their state in widgets,
where only a test can reach it, and the integration tests run a debug build.
This script measures the release build from the outside instead. It starts the
application with the same environment, opens the same pages with the mouse
events of the XTEST extension of the X server, and measures when the screen
has shown something and has not changed for 3 seconds, as long as the other
scenarios wait.

Usage: measure.py [--output <directory>] <executable>

Run it in a virtual display, for example
`xvfb-run -a -s "-screen 0 1280x800x24" python3 tool/measure.py <executable>`.
The application opens a window of 1280 x 820 (flutter/linux/runner/
my_application.cc), which fills the screen when there is no window manager; the
positions that the script clicks are for that window. Prints lines
"memory (<point>): RSS <n> MB, PSS <n> MB in <n> processes", as measure.sh does.
With --output, the directory also gets these lines in memory.txt, a screenshot
of every page the script opened and the output of the application, so that a
change of the layout that moves a button shows in the screenshots.

The script needs nothing but Python 3 and the libraries that GTK, and so the
application, loads: libX11 and libXtst (through ctypes).
"""

import argparse
import ctypes
import os
import shutil
import struct
import subprocess
import sys
import tempfile
import time
import zlib
from dataclasses import dataclass
from pathlib import Path

REPOSITORY = Path(__file__).resolve().parents[2]
FIXTURES = REPOSITORY / "shared" / "fixtures"

# The screen has to be unchanged this long before it is measured, which is what
# the "measure" scenarios of the other implementations wait.
SETTLE_SECONDS = 3.0
# How often the screen is looked at, and how long the application may take to
# show its window and to settle on a page.
POLL_SECONDS = 0.25
START_TIMEOUT_SECONDS = 60.0
SETTLE_TIMEOUT_SECONDS = 60.0
# A click must change at least this part of the screen. Opening a page changes
# 16% to 50% of it in the scenario; a click that misses only moves the highlight
# of the row under the pointer, a few percent.
MIN_CHANGE = 0.10

# Where to click, in pixels from the top left corner of the window.
RAIL_SDKS = (48, 245)
SDK_NODEJS = (276, 396)
TAB_AVAILABLE = (702, 174)

# Constants of Xlib.h.
INPUT_OUTPUT = 1
IS_VIEWABLE = 2
Z_PIXMAP = 2
LSB_FIRST = 0
ALL_PLANES = ctypes.c_ulong(-1).value


class MeasureError(Exception):
    """The measurement cannot go on; the message says why."""


class _Attributes(ctypes.Structure):
    """XWindowAttributes."""

    _fields_ = [
        ("x", ctypes.c_int),
        ("y", ctypes.c_int),
        ("width", ctypes.c_int),
        ("height", ctypes.c_int),
        ("border_width", ctypes.c_int),
        ("depth", ctypes.c_int),
        ("visual", ctypes.c_void_p),
        ("root", ctypes.c_ulong),
        ("window_class", ctypes.c_int),
        ("bit_gravity", ctypes.c_int),
        ("win_gravity", ctypes.c_int),
        ("backing_store", ctypes.c_int),
        ("backing_planes", ctypes.c_ulong),
        ("backing_pixel", ctypes.c_ulong),
        ("save_under", ctypes.c_int),
        ("colormap", ctypes.c_ulong),
        ("map_installed", ctypes.c_int),
        ("map_state", ctypes.c_int),
        ("all_event_masks", ctypes.c_long),
        ("your_event_mask", ctypes.c_long),
        ("do_not_propagate_mask", ctypes.c_long),
        ("override_redirect", ctypes.c_int),
        ("screen", ctypes.c_void_p),
    ]


class _XImage(ctypes.Structure):
    """XImage, with the functions that it carries."""

    _fields_ = [
        ("width", ctypes.c_int),
        ("height", ctypes.c_int),
        ("xoffset", ctypes.c_int),
        ("format", ctypes.c_int),
        ("data", ctypes.c_void_p),
        ("byte_order", ctypes.c_int),
        ("bitmap_unit", ctypes.c_int),
        ("bitmap_bit_order", ctypes.c_int),
        ("bitmap_pad", ctypes.c_int),
        ("depth", ctypes.c_int),
        ("bytes_per_line", ctypes.c_int),
        ("bits_per_pixel", ctypes.c_int),
        ("red_mask", ctypes.c_ulong),
        ("green_mask", ctypes.c_ulong),
        ("blue_mask", ctypes.c_ulong),
        ("obdata", ctypes.c_void_p),
        ("create_image", ctypes.c_void_p),
        ("destroy_image", ctypes.c_void_p),
        ("get_pixel", ctypes.c_void_p),
        ("put_pixel", ctypes.c_void_p),
        ("sub_image", ctypes.c_void_p),
        ("add_pixel", ctypes.c_void_p),
    ]


_ERROR_HANDLER = ctypes.CFUNCTYPE(ctypes.c_int, ctypes.c_void_p, ctypes.c_void_p)
_DESTROY_IMAGE = ctypes.CFUNCTYPE(ctypes.c_int, ctypes.c_void_p)


@dataclass(frozen=True)
class Window:
    """A top-level window, in pixels from the top left corner of the screen."""

    x: int
    y: int
    width: int
    height: int


@dataclass(frozen=True)
class Image:
    """The screen: 4 bytes per pixel, in the order blue, green, red, unused."""

    width: int
    height: int
    pixels: bytes

    def samples(self, step: int) -> list[bytes]:
        """The pixels on a grid, a few thousand of them: enough to tell pages apart."""
        return [
            self.pixels[(y * self.width + x) * 4 : (y * self.width + x) * 4 + 4]
            for y in range(0, self.height, step)
            for x in range(0, self.width, step)
        ]

    def has_content(self) -> bool:
        """Whether more than a few colours are on the screen, which a blank window is not."""
        return len(set(self.samples(16))) > 8

    def changed_from(self, other: "Image") -> float:
        """The part of the grid of samples that differs from another image."""
        mine, theirs = self.samples(8), other.samples(8)
        return sum(a != b for a, b in zip(mine, theirs)) / len(mine)

    def write_png(self, path: Path) -> None:
        """Saves the image as a PNG file."""
        rows = bytearray()
        for y in range(self.height):
            row = self.pixels[y * self.width * 4 : (y + 1) * self.width * 4]
            # A filter type (0, none) and then red, green and blue of each pixel.
            line = bytearray(1 + self.width * 3)
            line[1::3] = row[2::4]
            line[2::3] = row[1::4]
            line[3::3] = row[0::4]
            rows += line

        def chunk(kind: bytes, data: bytes) -> bytes:
            body = kind + data
            return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body))

        path.write_bytes(
            b"\x89PNG\r\n\x1a\n"
            + chunk(b"IHDR", struct.pack(">IIBBBBB", self.width, self.height, 8, 2, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress(bytes(rows), 6))
            + chunk(b"IEND", b"")
        )


class XServer:
    """The X server of DISPLAY: what the script sees of it and does to it."""

    def __init__(self) -> None:
        try:
            self._x11 = ctypes.CDLL("libX11.so.6")
            self._xtest = ctypes.CDLL("libXtst.so.6")
        except OSError as error:
            raise MeasureError(f"Cannot load the X11 libraries: {error}") from error
        x11, xtest = self._x11, self._xtest
        x11.XOpenDisplay.restype = ctypes.c_void_p
        x11.XOpenDisplay.argtypes = [ctypes.c_char_p]
        x11.XCloseDisplay.argtypes = [ctypes.c_void_p]
        x11.XDefaultRootWindow.restype = ctypes.c_ulong
        x11.XDefaultRootWindow.argtypes = [ctypes.c_void_p]
        x11.XSetErrorHandler.argtypes = [_ERROR_HANDLER]
        x11.XQueryTree.argtypes = [
            ctypes.c_void_p,
            ctypes.c_ulong,
            ctypes.POINTER(ctypes.c_ulong),
            ctypes.POINTER(ctypes.c_ulong),
            ctypes.POINTER(ctypes.POINTER(ctypes.c_ulong)),
            ctypes.POINTER(ctypes.c_uint),
        ]
        x11.XGetWindowAttributes.argtypes = [
            ctypes.c_void_p,
            ctypes.c_ulong,
            ctypes.POINTER(_Attributes),
        ]
        x11.XGetImage.restype = ctypes.POINTER(_XImage)
        x11.XGetImage.argtypes = [
            ctypes.c_void_p,
            ctypes.c_ulong,
            ctypes.c_int,
            ctypes.c_int,
            ctypes.c_uint,
            ctypes.c_uint,
            ctypes.c_ulong,
            ctypes.c_int,
        ]
        x11.XFree.argtypes = [ctypes.c_void_p]
        x11.XSync.argtypes = [ctypes.c_void_p, ctypes.c_int]
        xtest.XTestFakeMotionEvent.argtypes = [
            ctypes.c_void_p,
            ctypes.c_int,
            ctypes.c_int,
            ctypes.c_int,
            ctypes.c_ulong,
        ]
        xtest.XTestFakeButtonEvent.argtypes = [
            ctypes.c_void_p,
            ctypes.c_uint,
            ctypes.c_int,
            ctypes.c_ulong,
        ]

        # A window can go away between looking for it and asking about it; Xlib
        # would end the process on that error.
        self._ignore_errors = _ERROR_HANDLER(lambda display, event: 0)
        x11.XSetErrorHandler(self._ignore_errors)
        self._display = x11.XOpenDisplay(None)
        if not self._display:
            raise MeasureError("Cannot open the display; run the script in a virtual display.")
        self._root = x11.XDefaultRootWindow(self._display)
        screen = _Attributes()
        x11.XGetWindowAttributes(self._display, self._root, ctypes.byref(screen))
        self.width, self.height = screen.width, screen.height

    def close(self) -> None:
        """Closes the connection."""
        self._x11.XCloseDisplay(self._display)

    def application_window(self) -> Window | None:
        """The window of the application: the largest viewable window of the screen."""
        root, parent = ctypes.c_ulong(), ctypes.c_ulong()
        children = ctypes.POINTER(ctypes.c_ulong)()
        count = ctypes.c_uint()
        if not self._x11.XQueryTree(
            self._display,
            self._root,
            ctypes.byref(root),
            ctypes.byref(parent),
            ctypes.byref(children),
            ctypes.byref(count),
        ):
            return None
        try:
            identifiers = [children[index] for index in range(count.value)]
        finally:
            if children:
                self._x11.XFree(children)
        found = None
        for identifier in identifiers:
            attributes = _Attributes()
            if not self._x11.XGetWindowAttributes(
                self._display, identifier, ctypes.byref(attributes)
            ):
                continue
            if attributes.window_class != INPUT_OUTPUT or attributes.map_state != IS_VIEWABLE:
                continue
            window = Window(attributes.x, attributes.y, attributes.width, attributes.height)
            if found is None or window.width * window.height > found.width * found.height:
                found = window
        if found is None or found.width < 600 or found.height < 400:
            return None
        return found

    def capture(self) -> Image:
        """The screen as it is shown."""
        pointer = self._x11.XGetImage(
            self._display, self._root, 0, 0, self.width, self.height, ALL_PLANES, Z_PIXMAP
        )
        if not pointer:
            raise MeasureError("Cannot read the screen.")
        image = pointer.contents
        try:
            if (
                image.bits_per_pixel != 32
                or image.byte_order != LSB_FIRST
                or (image.red_mask, image.green_mask, image.blue_mask) != (0xFF0000, 0xFF00, 0xFF)
                or image.bytes_per_line != self.width * 4
            ):
                raise MeasureError("The screen does not have 32 bits per pixel in BGRX order.")
            pixels = ctypes.string_at(image.data, image.bytes_per_line * image.height)
        finally:
            _DESTROY_IMAGE(image.destroy_image)(ctypes.addressof(image))
        return Image(self.width, self.height, pixels)

    def click(self, window: Window, position: tuple[int, int]) -> None:
        """Moves the pointer to a position in the window and presses the left button."""
        x, y = window.x + position[0], window.y + position[1]
        self._xtest.XTestFakeMotionEvent(self._display, -1, x, y, 0)
        self._x11.XSync(self._display, 0)
        # The application sees the pointer arrive before the button, as with a mouse.
        time.sleep(0.2)
        self._xtest.XTestFakeButtonEvent(self._display, 1, 1, 0)
        self._x11.XSync(self._display, 0)
        time.sleep(0.1)
        self._xtest.XTestFakeButtonEvent(self._display, 1, 0, 0)
        self._x11.XSync(self._display, 0)


def memory_of(pid: int) -> tuple[int, int, int]:
    """The RSS and PSS, in kB, of a process and its descendants, and how many there are.

    The same sums as the function memory_of of shared/tools/measure.sh.
    """
    pids = [pid]
    index = 0
    while index < len(pids):
        for path in Path(f"/proc/{pids[index]}/task").glob("*/children"):
            try:
                pids += [int(child) for child in path.read_text().split()]
            except OSError:
                pass
        index += 1
    rss = pss = 0
    for process in pids:
        try:
            lines = Path(f"/proc/{process}/smaps_rollup").read_text().splitlines()
        except OSError:
            continue
        for line in lines:
            if line.startswith("Rss:"):
                rss += int(line.split()[1])
            elif line.startswith("Pss:"):
                pss += int(line.split()[1])
    return rss, pss, len(pids)


class Scenario:
    """The application, the screen and what has been seen of it so far."""

    def __init__(self, server: XServer, process: subprocess.Popen[bytes], output: Path | None):
        self.server = server
        self.process = process
        self.output = output
        self.memory: list[str] = []

    def check_running(self) -> None:
        """Fails if the application has ended."""
        if self.process.poll() is not None:
            raise MeasureError(f"The application ended with status {self.process.returncode}.")

    def wait_for_window(self) -> Window:
        """Waits for the window of the application to be shown."""
        deadline = time.monotonic() + START_TIMEOUT_SECONDS
        while time.monotonic() < deadline:
            self.check_running()
            window = self.server.application_window()
            if window is not None:
                return window
            time.sleep(POLL_SECONDS)
        raise MeasureError("The application did not show its window.")

    def settle(self, name: str) -> Image:
        """Waits until the screen shows something and has not changed for a while.

        The page is not loading or moving then, and the memory that it needs is
        in use. Saves a screenshot of it as <name>.png.
        """
        deadline = time.monotonic() + SETTLE_TIMEOUT_SECONDS
        previous = self.server.capture()
        still_since = time.monotonic()
        while True:
            self.check_running()
            time.sleep(POLL_SECONDS)
            image = self.server.capture()
            now = time.monotonic()
            if image.pixels != previous.pixels or not image.has_content():
                still_since = now
            elif now - still_since >= SETTLE_SECONDS:
                if self.output is not None:
                    image.write_png(self.output / f"{name}.png")
                return image
            if now > deadline:
                if self.output is not None:
                    image.write_png(self.output / f"{name}.png")
                raise MeasureError(f"The screen did not settle on the page {name}.")
            previous = image

    def open_page(self, name: str, position: tuple[int, int], before: Image) -> Image:
        """Clicks a position in the window and waits for the page that it opens."""
        window = self.server.application_window()
        if window is None:
            raise MeasureError("The window of the application is gone.")
        self.server.click(window, position)
        image = self.settle(name)
        changed = image.changed_from(before)
        print(f"opened {name}: {changed:.0%} of the screen changed", flush=True)
        if changed < MIN_CHANGE:
            raise MeasureError(
                f"The click at {position} did not open the page {name}: only {changed:.1%} of "
                "the screen changed. Look at the screenshots; the layout may have moved it."
            )
        return image

    def report(self, point: str) -> None:
        """Prints the memory of the application."""
        rss, pss, processes = memory_of(self.process.pid)
        line = f"memory ({point}): RSS {rss // 1024} MB, PSS {pss // 1024} MB in {processes} processes"
        print(line, flush=True)
        self.memory.append(line)

    def run(self) -> None:
        """Opens the Home page and the available versions of Node.js, and measures both."""
        self.wait_for_window()
        home = self.settle("home")
        self.report("home")
        sdks = self.open_page("sdks", RAIL_SDKS, home)
        installed = self.open_page("nodejs-installed", SDK_NODEJS, sdks)
        self.open_page("nodejs", TAB_AVAILABLE, installed)
        self.report("nodejs")


def run(executable: Path, output: Path | None) -> int:
    """Starts the application in a new HOME, with the fake vfox, and measures it."""
    if output is not None:
        output.mkdir(parents=True, exist_ok=True)
    server = XServer()
    try:
        return measure(server, executable, output)
    finally:
        server.close()


def measure(server: XServer, executable: Path, output: Path | None) -> int:
    """Runs the scenario against the application and returns the exit status."""
    with tempfile.TemporaryDirectory(prefix="vfox-plus-measure.", ignore_cleanup_errors=True) as work:
        sandbox = Path(work)
        home = sandbox / "home"
        (home / ".config" / "vfox-plus").mkdir(parents=True)
        # The fake vfox runs as if the application was started from an activated
        # shell, as in shared/tools/measure.sh. The settings tell the application
        # where to find vfox.
        vfox = sandbox / "vfox"
        shutil.copy(FIXTURES / "fake-vfox.sh", vfox)
        vfox.chmod(0o755)
        (home / ".config" / "vfox-plus" / "settings.json").write_text(
            '{\n  "language": "english",\n  "vfoxExecutable": "%s"\n}\n' % vfox
        )
        environment = {
            name: value
            for name, value in os.environ.items()
            if name not in ("VFOX_HOME", "XDG_CONFIG_HOME")
        }
        environment.update(
            HOME=str(home),
            LANG="C.UTF-8",
            VFOX_PLUS_FIXTURES=str(FIXTURES / "vfox-1.0.13"),
            __VFOX_PID="1",
            __VFOX_SHELL="bash",
        )
        log = sandbox / "app.log"
        try:
            with log.open("wb") as log_file:
                process = subprocess.Popen(
                    [str(executable)], env=environment, stdout=log_file, stderr=subprocess.STDOUT
                )
        except OSError as error:
            raise MeasureError(f"Cannot start {executable}: {error}") from error
        scenario = Scenario(server, process, output)
        status = 0
        try:
            scenario.run()
        except MeasureError as error:
            # The reason comes last, where it is read first.
            print("Output of the application:", file=sys.stderr)
            print(log.read_text(errors="replace"), file=sys.stderr)
            print(f"measure.py: {error}", file=sys.stderr)
            status = 1
        finally:
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            if output is not None:
                shutil.copy(log, output / "app.log")
                (output / "memory.txt").write_text("".join(f"{line}\n" for line in scenario.memory))
        return status


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("executable", type=Path, help="the executable of the release build")
    parser.add_argument(
        "--output",
        type=Path,
        help="a directory for memory.txt, the screenshots and the output of the application",
    )
    arguments = parser.parse_args()
    try:
        return run(arguments.executable.resolve(), arguments.output)
    except MeasureError as error:
        print(f"measure.py: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
