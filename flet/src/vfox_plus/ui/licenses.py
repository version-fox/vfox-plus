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

"""The open-source licenses of the code that the built application bundles,
gathered where the bundle keeps them: the notices that `flutter build`
collects from the Dart packages and the engine, the license files of the
Python packages, the license of Python itself and the one of the icon font.
The Flutter implementation shows the same with `showLicensePage`.
"""

import gzip
import importlib.metadata
import os
import zlib
from collections.abc import Iterable
from dataclasses import dataclass
from pathlib import Path

from vfox_plus.ui.resources import RESOURCES

_SEPARATOR = "-" * 80
_LICENSE_NAMES = ("LICENSE", "LICENCE", "COPYING", "NOTICE", "AUTHORS")


@dataclass(frozen=True)
class LicenseEntry:
    """The license of one or more packages."""

    packages: tuple[str, ...]
    text: str


def decompress_notices(data: bytes) -> str:
    """Decompresses a Flutter `NOTICES` file, which is gzipped despite its
    `.Z` name; older files may be raw zlib.
    """
    try:
        return gzip.decompress(data).decode("utf-8", errors="replace")
    except OSError:
        return zlib.decompress(data).decode("utf-8", errors="replace")


def parse_notices(notices: str) -> list[LicenseEntry]:
    """Parses the `NOTICES` file of Flutter: entries separated by a line of
    80 dashes, each the names of its packages, a blank line and the text.
    """
    entries = []
    for block in notices.split(f"\n{_SEPARATOR}\n"):
        names, _, text = block.strip("\n").partition("\n\n")
        packages = tuple(name.strip() for name in names.splitlines() if name.strip())
        if packages and text.strip():
            entries.append(LicenseEntry(packages, text.strip("\n")))
    return entries


def flutter_notices(executable: str | None = None) -> list[LicenseEntry]:
    """The notices of the Dart packages and of the Flutter engine, next to
    the executable of the built application; none when running from the
    sources.
    """
    try:
        path = Path(executable or os.readlink("/proc/self/exe")).parent / "data/flutter_assets/NOTICES.Z"
        return parse_notices(decompress_notices(path.read_bytes()))
    except OSError, zlib.error:
        return []


def python_licenses(distributions: Iterable[importlib.metadata.Distribution] | None = None) -> list[LicenseEntry]:
    """The license files of the installed Python packages, and the license
    of Python.
    """
    entries = []
    for distribution in distributions if distributions is not None else importlib.metadata.distributions():
        name = distribution.metadata["Name"]
        texts = []
        for file in distribution.files or ():
            if file.name.upper().startswith(_LICENSE_NAMES) or "licenses" in file.parts:
                try:
                    texts.append(file.read_text(encoding="utf-8"))
                except OSError, UnicodeDecodeError:
                    continue
        if name and texts:
            entries.append(LicenseEntry((f"{name} {distribution.version}",), "\n\n".join(texts).strip("\n")))
    python = Path(os.__file__).parent / "LICENSE.txt"
    if python.is_file():
        entries.append(LicenseEntry(("Python",), python.read_text(encoding="utf-8")))
    return entries


def bundled_licenses() -> list[LicenseEntry]:
    """Every license the application ships, sorted by package."""
    entries = [*flutter_notices(), *python_licenses()]
    font = RESOURCES / "MaterialSymbolsRounded-LICENSE.txt"
    if font.is_file():
        entries.append(LicenseEntry(("Material Symbols Rounded",), font.read_text(encoding="utf-8")))
    return sorted(entries, key=lambda entry: entry.packages[0].lower())
