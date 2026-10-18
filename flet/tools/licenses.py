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

"""Writes THIRD-PARTY-LICENSES for the .deb: the licenses of everything the
release bundle of `flet build linux` ships that is not written by this
repository — the Dart packages and the Flutter engine, the Python packages
and Python itself, and the icon font. The application shows the same texts
in its about section (see `src/vfox_plus/ui/licenses.py`).

Usage: tools/licenses.py <bundle directory> <output file>
"""

import importlib.metadata
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent / "src"))

from vfox_plus.ui.licenses import LicenseEntry, decompress_notices, parse_notices


def main() -> None:
    bundle = pathlib.Path(sys.argv[1])
    output = pathlib.Path(sys.argv[2])
    entries: list[LicenseEntry] = []

    notices = bundle / "data/flutter_assets/NOTICES.Z"
    entries.extend(parse_notices(decompress_notices(notices.read_bytes())))

    for distribution in sorted(importlib.metadata.distributions(), key=lambda d: (d.metadata["Name"] or "").lower()):
        name = distribution.metadata["Name"]
        if not name:
            continue
        texts = []
        for file in distribution.files or ():
            upper = file.name.upper()
            if upper.startswith(("LICENSE", "LICENCE", "COPYING", "NOTICE", "AUTHORS")) or "licenses" in file.parts:
                try:
                    texts.append(file.read_text(encoding="utf-8"))
                except OSError, UnicodeDecodeError:
                    continue
        if texts:
            entries.append(LicenseEntry((f"{name} {distribution.version}",), "\n\n".join(texts).strip("\n")))
    # Run with the site-packages of the bundle on the path (see package.sh),
    # so that the licenses collected are the ones of the bundled packages.
    python_licenses = sorted(bundle.glob("python*/LICENSE.txt"))
    if python_licenses:
        entries.append(LicenseEntry(("Python",), python_licenses[0].read_text(encoding="utf-8")))

    font_license = bundle / "app/vfox_plus/resources/MaterialSymbolsRounded-LICENSE.txt"
    if font_license.is_file():
        entries.append(LicenseEntry(("Material Symbols Rounded",), font_license.read_text(encoding="utf-8")))

    with output.open("w", encoding="utf-8") as file:
        file.write(
            "Third-party licenses of vfox+ (Flet implementation).\n"
            "The application shows the same texts in Settings, About, Open-source licenses.\n\n"
        )
        for entry in entries:
            file.write(f"{'=' * 78}\n{', '.join(entry.packages)}\n{'=' * 78}\n\n{entry.text}\n\n")
    print(f"Wrote {output} ({len(entries)} packages)")


if __name__ == "__main__":
    main()
