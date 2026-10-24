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

"""Patches the cached Flet build template of the Linux window: it opens at the
default size of the UI specification (1280 x 820), and draws on the processor.

`flet build` renders its Flutter shell from `flet-build-template.zip`, which
it downloads once into `~/.flet/cache` (or `$FLET_CACHE_DIR`) and never
revalidates. The template hardcodes `gtk_window_set_default_size(window,
1280, 720)`; resizing the window at run time instead crashes the renderer
from time to time on an X server without a window manager, such as the Xvfb
of the tests. Patching the cached template sets the size before the first
frame, with no resize at run time.

The template also draws with OpenGL, which means Mesa's software rasterizer
and its compiler when there is no real GPU, and takes as much memory as the
rest of the application (docs/memory.md). The patched `main` has the engine
draw with Skia on the processor (`FLUTTER_LINUX_RENDERER`) and GDK leave
OpenGL alone, as the Flutter implementation does.

Run it before `flet build` (the sandbox image and CI do):

    python3 tools/patch_build_template.py

It downloads the template of the pinned Flet version when the cache lacks it
and does nothing when the cache is already patched.
"""

import os
import pathlib
import sys
import urllib.request
import zipfile

FLET_VERSION = "1.0.5"
"""Pinned in `pyproject.toml`; the template comes from its release."""

WIDTH = 1280
HEIGHT = 820
"""The default window size of the UI specification (`docs/ui-spec.md`)."""

_DIRECTORY = "build/{{cookiecutter.out_dir}}/linux/"
SIZE = (
    _DIRECTORY + "my_application.cc",
    "gtk_window_set_default_size(window, 1280, 720);",
    f"gtk_window_set_default_size(window, {WIDTH}, {HEIGHT});",
)
"""The window size, in the application of the GTK window."""

_MAIN = "  g_autoptr(MyApplication) app = my_application_new();\n"
_SETTINGS = '  g_setenv("FLUTTER_LINUX_RENDERER", "software", FALSE);\n  g_setenv("GDK_GL", "disable", FALSE);\n'
RENDERER = (_DIRECTORY + "main.cc", _MAIN, _SETTINGS + _MAIN)
"""The renderer, which `main` sets before GTK and the engine start."""

PATCHES = (SIZE, RENDERER)


def cache_path() -> pathlib.Path:
    """Where `flet build` keeps the template (see `template_cache.py` of
    flet-cli).
    """
    root = os.environ.get("FLET_CACHE_DIR") or str(pathlib.Path.home() / ".flet" / "cache")
    return pathlib.Path(root) / "build-template" / f"v{FLET_VERSION}" / "flet-build-template.zip"


def main() -> None:
    path = cache_path()
    if not (path.exists() and path.stat().st_size > 0):
        url = f"https://github.com/flet-dev/flet/releases/download/v{FLET_VERSION}/flet-build-template.zip"
        path.parent.mkdir(parents=True, exist_ok=True)
        partial = path.with_suffix(path.suffix + ".tmp")
        with urllib.request.urlopen(url, timeout=120) as response, partial.open("wb") as file:
            while block := response.read(1 << 20):
                file.write(block)
        partial.replace(path)
        print(f"Downloaded {path}")
    with zipfile.ZipFile(path) as archive:
        members = [(info, archive.read(info.filename)) for info in archive.infolist()]
    pending = []
    for member, old, new in PATCHES:
        copies = [data.decode() for info, data in members if info.filename == member]
        if not copies:
            sys.exit(f"{member} is not in {path}; the template of Flet {FLET_VERSION} changed.")
        if new in copies[0]:
            print(f"{member} is already patched.")
            continue
        if old not in copies[0]:
            sys.exit(f"Cannot find {old!r} in {member}; the template of Flet {FLET_VERSION} changed.")
        pending.append((member, old, new, copies[0]))
    if not pending:
        return
    replacements = {member: (old, new, copy) for member, old, new, copy in pending}
    seen: set[str] = set()
    with zipfile.ZipFile(path.with_suffix(path.suffix + ".tmp"), "w", zipfile.ZIP_DEFLATED) as archive:
        for info, data in members:
            if info.filename not in replacements:
                archive.writestr(info, data)
            elif info.filename not in seen:
                # There is only one copy after this rewrite, even if an
                # earlier run of this script appended one already.
                seen.add(info.filename)
                old, new, copy = replacements[info.filename]
                archive.writestr(info, copy.replace(old, new))
    path.with_suffix(path.suffix + ".tmp").replace(path)
    for member in replacements:
        print(f"Patched {member} in {path}")


if __name__ == "__main__":
    main()
