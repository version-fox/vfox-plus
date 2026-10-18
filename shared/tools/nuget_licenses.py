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

"""Checks the licenses of the NuGet packages that a published .NET application
ships, and collects their notices and the license texts into one file, which
the application shows and the packages install.

The packages are those of the dependency file of the published application
(`<name>.deps.json`, which a trimmed publish keeps accurate) and the .NET runtime
that a self-contained application bundles. A license expression must consist of
licenses compatible with, or no more restrictive than, the Apache License 2.0
(docs/architecture.md#dependencies); a package that says nothing is checked
against its own license file, or, if it has none, listed in KNOWN_LICENSES with
where its license was found.

Usage: nuget_licenses.py --deps <file> --license <file> --output <file>
                         [--packages <directory>] [--append]

--license is the license of the application itself (the Apache License 2.0,
which the packages under that license share). With --append, the licenses are
added to the end of the file. Run `dotnet restore` first: the licenses are read
from the NuGet cache.
"""

import argparse
import hashlib
import json
import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

ALLOWED = {"Apache-2.0", "MIT"}

# Packages that declare no license in their metadata: where their license was
# found.
KNOWN_LICENSES = {
    # https://github.com/unoplatform/Uno.Diagnostics.Eventing/blob/master/LICENSE
    "Uno.Diagnostics.Eventing": "Apache-2.0",
}

# Parts of the notices of SkiaSharp that are not part of its binary: Skia lists
# everything in its third_party directory, and the library has none of the code
# of libmicrohttpd (it has no MHD_ symbols), which is LGPL.
OMITTED_NOTICES = ["libmicrohttpd"]

MIT_TEXT = """Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE."""

SEPARATOR = "\n\n" + "-" * 72 + "\n\n"


class Package:
    """A package of the NuGet cache, as its nuspec describes it."""

    def __init__(self, root, name, version):
        self.name, self.version = name, version
        self.directory = root / name.lower() / version
        spec = self.directory / f"{name.lower()}.nuspec"
        if not spec.exists():
            raise SystemExit(f"{name} {version} is not in {root}: restore the project first")
        metadata = {}
        self.license_type, self.license = None, None
        for element in ET.parse(spec).getroot().iter():
            tag = element.tag.rsplit("}", 1)[-1]
            if tag == "license":
                self.license_type, self.license = element.get("type"), (element.text or "").strip()
            elif tag in ("copyright", "projectUrl", "authors"):
                metadata[tag] = (element.text or "").strip()
        self.copyright = metadata.get("copyright") or metadata.get("authors", "")
        self.url = metadata.get("projectUrl", "")

    @property
    def title(self):
        return f"{self.name} {self.version}"


def allowed(expression):
    """Whether an SPDX license expression is made of allowed licenses: one of
    the alternatives of an OR, all of the terms of an AND."""
    cleaned = re.sub(r"[()]", " ", expression)
    return any(all(term in ALLOWED for term in alternative.split(" AND ")) for alternative in cleaned.split(" OR "))


def shipped(deps):
    """The packages of a dependency file that have code or native libraries."""
    found = {}
    content = json.loads(deps.read_text(encoding="utf-8"))
    for target in content["targets"].values():
        for key, library in target.items():
            if content["libraries"].get(key, {}).get("type") != "package" or key.startswith("runtimepack."):
                continue
            if library.get("runtime") or library.get("native"):
                name, version = key.split("/", 1)
                found[name] = version
    return found


def runtime_pack(deps):
    """The runtime pack that the dependency file says the application bundles."""
    for target in json.loads(deps.read_text(encoding="utf-8"))["targets"].values():
        for key in target:
            if key.startswith("runtimepack."):
                name, version = key[len("runtimepack."):].split("/", 1)
                return name, version
    return None


def omit(notices):
    """The notices of SkiaSharp without the parts that its binary does not contain."""
    lines = notices.split("\n")
    kept, skipping = [], None
    for index, line in enumerate(lines):
        if line.startswith("# ") and index > 0 and lines[index - 1].startswith("#####"):
            name = line[2:].strip()
            if name in OMITTED_NOTICES:
                skipping = name
                # The line of # that opens the part is already kept.
                if kept and kept[-1].startswith("#####"):
                    kept.pop()
                continue
        if skipping:
            if line.startswith("# END:") and skipping in line:
                skipping = None
            continue
        kept.append(line)
    return "\n".join(kept)


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--deps", required=True, type=Path, help="the <name>.deps.json of the published application")
    parser.add_argument("--license", required=True, type=Path, help="the Apache License 2.0")
    parser.add_argument("--packages", type=Path, default=Path.home() / ".nuget" / "packages", help="the NuGet cache")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--append", action="store_true", help="add to the end of the output file")
    arguments = parser.parse_args()

    names = shipped(arguments.deps)

    errors, by_license, files = [], {}, []
    for name in sorted(names):
        package = Package(arguments.packages, name, names[name])
        license_id = package.license if package.license_type == "expression" else KNOWN_LICENSES.get(name)
        if license_id is None and package.license_type == "file":
            text = (package.directory / package.license).read_text(encoding="utf-8", errors="replace")
            flat = " ".join(text.split())
            if "Permission is hereby granted, free of charge" in flat:
                license_id = "MIT"
        if license_id is None or not allowed(license_id):
            errors.append(f"{package.title}: {license_id or 'no license'} is not compatible with the Apache License 2.0")
            continue
        by_license.setdefault("Apache-2.0" if "Apache-2.0" in license_id else "MIT", []).append(package)
        for notice in package.directory.glob("THIRD-PARTY-NOTICES*"):
            files.append((f"The third-party notices of {package.title}", notice.read_text(encoding="utf-8", errors="replace")))
    if errors:
        print("Packages with licenses that cannot be shipped:\n  " + "\n  ".join(errors), file=sys.stderr)
        sys.exit(1)

    sections = []
    for license_id, packages in sorted(by_license.items()):
        lines = [f"{package.title}: {package.copyright}" + (f" ({package.url})" if package.url else "") for package in packages]
        text = arguments.license.read_text(encoding="utf-8").strip() if license_id == "Apache-2.0" else MIT_TEXT
        sections.append(f"The packages below are licensed under the {license_id} license; their copyright notices:\n\n"
                        + "\n".join(lines) + f"\n\n{text}")

    pack = runtime_pack(arguments.deps)
    if pack:
        directory = arguments.packages / pack[0].lower() / pack[1]
        license_text = (directory / "LICENSE.TXT").read_text(encoding="utf-8", errors="replace").strip()
        sections.append(f"The .NET runtime, {pack[0]} {pack[1]}, which the application bundles (MIT)\n\n{license_text}")
        files.append((f"The third-party notices of the .NET runtime {pack[1]}",
                      (directory / "THIRD-PARTY-NOTICES.TXT").read_text(encoding="utf-8", errors="replace")))

    seen = set()
    for title, text in files:
        digest = hashlib.sha256(text.encode()).hexdigest()
        if digest not in seen:
            seen.add(digest)
            sections.append(f"{title}\n\n{omit(text).strip()}")

    rendered = SEPARATOR.join(sections) + "\n"
    arguments.output.parent.mkdir(parents=True, exist_ok=True)
    if arguments.append and arguments.output.exists():
        rendered = arguments.output.read_text(encoding="utf-8").rstrip("\n") + SEPARATOR + rendered
    arguments.output.write_text(rendered, encoding="utf-8")
    count = sum(len(packages) for packages in by_license.values())
    print(f"Wrote {arguments.output} ({count} NuGet packages and the .NET runtime)")


if __name__ == "__main__":
    main()
