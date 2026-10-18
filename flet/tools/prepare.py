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

"""Generates the inputs of the application that are not committed.

- `m3e/src/flutter/vfox_plus_m3e/fonts/MaterialSymbolsRounded.ttf`: the icons
  of shared/icons.json cut out of Material Symbols Rounded by
  shared/tools/icon_font.py, keeping the fill axis only;
- `src/vfox_plus/resources/icons.json`: their code points, by meaning;
- `src/vfox_plus/resources/i18n/`: the catalogues of shared/i18n, as they are,
  so that `flet build` packages them with the code;
- `src/vfox_plus/resources/MaterialSymbolsRounded-LICENSE.txt`: the license
  of the font, shown by the application.

The variable font is taken from VFOX_PLUS_SYMBOLS (the sandbox images have it)
or downloaded once into ~/.cache/vfox-plus/material-symbols. Run it before
`flet run`, `flet build` and the tests:

    python3 tools/prepare.py
"""

import os
import pathlib
import shutil
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
SHARED = ROOT.parent / "shared"
TOOL = SHARED / "tools" / "icon_font.py"
FONTS = ROOT / "m3e" / "src" / "flutter" / "vfox_plus_m3e" / "fonts"
RESOURCES = ROOT / "src" / "vfox_plus" / "resources"


def main() -> None:
    symbols = os.environ.get("VFOX_PLUS_SYMBOLS") or str(pathlib.Path.home() / ".cache/vfox-plus/material-symbols")
    if not os.environ.get("VFOX_PLUS_SYMBOLS"):
        subprocess.run([sys.executable, TOOL, "download", symbols], check=True)
    FONTS.mkdir(parents=True, exist_ok=True)
    RESOURCES.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        [
            sys.executable,
            TOOL,
            "subset",
            "--fonts",
            symbols,
            "--output",
            FONTS / "MaterialSymbolsRounded.ttf",
            "--json",
            RESOURCES / "icons.json",
            # The user interface draws every icon at weight 400, grade 0 and
            # optical size 24, and fills some of them.
            "--pin",
            "wght=400",
            "--pin",
            "GRAD=0",
            "--pin",
            "opsz=24",
        ],
        check=True,
    )
    shutil.copyfile(pathlib.Path(symbols) / "LICENSE", RESOURCES / "MaterialSymbolsRounded-LICENSE.txt")
    i18n = RESOURCES / "i18n"
    shutil.rmtree(i18n, ignore_errors=True)
    shutil.copytree(SHARED / "i18n", i18n)
    # The application icon for `flet build`, which renders every platform
    # icon from `assets/icon.png`: the brand mark of shared/packaging/linux.
    assets = ROOT / "src" / "assets"
    assets.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        [
            "convert",
            "-background",
            "none",
            "-density",
            "384",
            str(SHARED / "packaging/linux/dev.vfox.VfoxPlus.svg"),
            "-resize",
            "1024x1024",
            str(assets / "icon.png"),
        ],
        check=True,
    )
    print(f"Prepared {FONTS.relative_to(ROOT)} and {RESOURCES.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
