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

"""Builds the icon font of an implementation from shared/icons.json.

The icons of vfox-plus are Material Symbols Rounded (see
docs/design-system.md#icons). Implementations ship only the glyphs they use,
cut out of the variable font of google/material-design-icons at a pinned
commit. The font is a build output and never committed.

Usage:

  icon_font.py download DIR
      Downloads the pinned variable font, its codepoint list and its license
      (LICENSE, to ship with the application) into DIR and verifies their
      checksums. Does nothing if DIR already holds them.

  icon_font.py subset --fonts DIR --output FILE [--json FILE] [--pin AXIS=VALUE ...]
      Writes the subset to FILE (.ttf or .woff2). The subset keeps the
      codepoints of the icons and the ligatures of their names, so that both
      "home" and U+E9B2 draw the home icon, and every variable axis that is
      not pinned. --json writes the codepoint of every icon, by meaning.

The same inputs always give the same output. Requires fontTools (and brotli
for WOFF2): `apt install fonttools python3-brotli` or `pip install fonttools
brotli`.
"""

import argparse
import hashlib
import json
import pathlib
import sys
import urllib.request

# google/material-design-icons, "Update Symbols" of 2026-10-23.
COMMIT = "10ecc14790612ca8fd164e3f4ffe9c16fc50d5b7"
BASE_URL = f"https://raw.githubusercontent.com/google/material-design-icons/{COMMIT}"
FONT_NAME = "MaterialSymbolsRounded[FILL,GRAD,opsz,wght]"
# Paths in the repository, by local file name, with their checksums. The
# license (Apache-2.0) ships with every application that bundles the font.
FILES = {
    f"{FONT_NAME}.ttf": (
        f"variablefont/{FONT_NAME}.ttf",
        "7e74e36533ccee12c2b81e75bfcf3ccf123d3bb7e7e46d65d94445f23db0c97c",
    ),
    f"{FONT_NAME}.codepoints": (
        f"variablefont/{FONT_NAME}.codepoints",
        "a7404d9e5a1b63f496cf05ace45d386075886b5a7f123fba9f29527cb6006fde",
    ),
    "LICENSE": ("LICENSE", "58d1e17ffe5109a7ae296caafcadfdbe6a7d176f0bc4ab01e12a689b0499d8bd"),
}
ICONS_JSON = pathlib.Path(__file__).resolve().parent.parent / "icons.json"
# Ligatures are spelled with these characters.
LIGATURE_CHARACTERS = "abcdefghijklmnopqrstuvwxyz0123456789_"


def sha256(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as file:
        for block in iter(lambda: file.read(1 << 20), b""):
            digest.update(block)
    return digest.hexdigest()


def download(directory: pathlib.Path) -> None:
    directory.mkdir(parents=True, exist_ok=True)
    for name, (path, checksum) in FILES.items():
        target = directory / name
        if target.exists() and sha256(target) == checksum:
            continue
        url = f"{BASE_URL}/{urllib.request.quote(path)}"
        partial = target.with_suffix(target.suffix + ".part")
        with urllib.request.urlopen(url, timeout=120) as response, partial.open("wb") as file:
            while block := response.read(1 << 20):
                file.write(block)
        actual = sha256(partial)
        if actual != checksum:
            partial.unlink()
            sys.exit(f"Checksum mismatch for {url}: expected {checksum}, got {actual}")
        partial.replace(target)
        print(f"Downloaded {target}")


def read_codepoints(path: pathlib.Path) -> dict[str, int]:
    codepoints = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        parts = line.split()
        if len(parts) == 2:
            codepoints[parts[0]] = int(parts[1], 16)
    return codepoints


def subset(fonts: pathlib.Path, output: pathlib.Path, json_output: pathlib.Path | None, pins: dict[str, float]) -> None:
    from fontTools import subset as ft_subset
    from fontTools.ttLib import TTFont
    from fontTools.varLib import instancer

    for name, (_, checksum) in FILES.items():
        if not (fonts / name).exists() or sha256(fonts / name) != checksum:
            sys.exit(f"{fonts / name} is missing or corrupt; run `icon_font.py download {fonts}`.")
    icons: dict[str, str] = json.loads(ICONS_JSON.read_text(encoding="utf-8"))["icons"]
    codepoints = read_codepoints(fonts / f"{FONT_NAME}.codepoints")
    missing = sorted(symbol for symbol in icons.values() if symbol not in codepoints)
    if missing:
        sys.exit(f"Unknown Material Symbols: {', '.join(missing)}")

    # A fixed modification time keeps the output reproducible.
    font = TTFont(fonts / f"{FONT_NAME}.ttf", recalcTimestamp=False)
    options = ft_subset.Options()
    # Keep the ligatures of the listed icons only: without closure, the
    # ligature lookups lose every entry whose glyph is not in the subset.
    options.layout_features = ["*"]
    options.layout_closure = False
    options.name_IDs = ["*"]
    options.name_languages = ["*"]
    options.notdef_outline = True
    options.flavor = "woff2" if output.suffix == ".woff2" else None
    unicodes = sorted({codepoints[symbol] for symbol in icons.values()} | {ord(c) for c in LIGATURE_CHARACTERS})
    subsetter = ft_subset.Subsetter(options)
    subsetter.populate(unicodes=unicodes)
    subsetter.subset(font)
    if pins:
        # Pinning the small subset is much faster than pinning the full font.
        font = instancer.instantiateVariableFont(font, pins)
        font.recalcTimestamp = False
    without_ligature = sorted(set(icons.values()) - ligature_names(font))
    if without_ligature:
        sys.exit(f"The subset lost the ligatures of: {', '.join(without_ligature)}")
    output.parent.mkdir(parents=True, exist_ok=True)
    ft_subset.save_font(font, str(output), options)
    print(f"Wrote {output} ({output.stat().st_size // 1024} KB, {len(icons)} icons)")

    if json_output is not None:
        mapping = {
            "font": "Material Symbols Rounded",
            "icons": {meaning: {"symbol": symbol, "codepoint": codepoints[symbol]} for meaning, symbol in icons.items()},
        }
        json_output.parent.mkdir(parents=True, exist_ok=True)
        json_output.write_text(json.dumps(mapping, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        print(f"Wrote {json_output}")


def ligature_names(font) -> set[str]:
    """The texts that a ligature of `font` replaces, such as "home"."""
    names = set()
    reverse_cmap = {glyph: chr(code) for code, glyph in font.getBestCmap().items()}
    for lookup in font["GSUB"].table.LookupList.Lookup:
        for table in lookup.SubTable:
            table = getattr(table, "ExtSubTable", table)
            for first, ligatures in getattr(table, "ligatures", {}).items():
                for ligature in ligatures:
                    names.add("".join(reverse_cmap.get(glyph, "?") for glyph in [first, *ligature.Component]))
    return names


def parse_pin(text: str) -> tuple[str, float]:
    axis, _, value = text.partition("=")
    if axis not in ("FILL", "GRAD", "opsz", "wght") or value == "":
        raise argparse.ArgumentTypeError(f"Expected AXIS=VALUE with AXIS one of FILL, GRAD, opsz, wght: {text}")
    return axis, float(value)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest="command", required=True)
    download_parser = commands.add_parser("download", help="download the pinned variable font")
    download_parser.add_argument("directory", type=pathlib.Path)
    subset_parser = commands.add_parser("subset", help="write the subset of the icons in shared/icons.json")
    subset_parser.add_argument("--fonts", type=pathlib.Path, required=True, help="directory of `download`")
    subset_parser.add_argument("--output", type=pathlib.Path, required=True, help="the .ttf or .woff2 to write")
    subset_parser.add_argument("--json", type=pathlib.Path, help="also write the codepoints of the icons")
    subset_parser.add_argument("--pin", type=parse_pin, action="append", default=[], help="pin an axis, e.g. wght=400")
    arguments = parser.parse_args()
    if arguments.command == "download":
        download(arguments.directory)
    else:
        subset(arguments.fonts, arguments.output, arguments.json, dict(arguments.pin))


if __name__ == "__main__":
    main()
