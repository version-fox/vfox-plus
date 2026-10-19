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

"""Gives a font another family name.

Usage: rename_font.py FONT FAMILY

The outlined and the filled icon font are two instances of Material Symbols
Rounded and have the same family name, so that fontconfig, which mygo draws text
with, cannot tell them apart: it picks one of the two faces for both. The filled
font is renamed to make the two families. Requires fontTools (`apt install
fonttools` or `pip install fonttools`).
"""

import pathlib
import sys

from fontTools.ttLib import TTFont


def main() -> None:
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    path, family = pathlib.Path(sys.argv[1]), sys.argv[2]
    font = TTFont(path, recalcTimestamp=False)
    postscript = family.replace(" ", "") + "-Regular"
    for record in font["name"].names:
        if record.nameID in (1, 4, 16):
            record.string = family
        elif record.nameID in (3, 6):
            record.string = postscript
    font.save(path)
    print(f"Renamed the family of {path} to {family}")


if __name__ == "__main__":
    main()
