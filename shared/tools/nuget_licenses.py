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

The packages are those of project.assets.json (the restored dependency graph;
run `dotnet restore` first) whose assemblies are among the ones a publish of
the application actually kept after trimming — project.assets.json alone is
the graph before trimming runs, and over-approximates what ships. Which
assemblies survived comes from --used-assemblies, one assembly name per line:
for a Native AOT publish, `-p:IlcGenerateMstatFile=true` writes `<name>.mstat`
next to the binary, a .NET assembly whose AssemblyRef table names every
assembly that is still referenced; shared/tools/mstat-assemblies reads it.
The .NET runtime that a self-contained application bundles is credited from the
installation of the SDK that published it (its own LICENSE.txt and
ThirdPartyNotices.txt), not from a NuGet package: Native AOT compiles the
runtime into the binary directly, with no runtime-pack package to read a
license from, and crediting the SDK instead works the same way either way.
A license expression must consist of licenses compatible with, or no more
restrictive than, the Apache License 2.0 (docs/architecture.md#dependencies);
a package that says nothing is checked against its own license file, or, if it
has none, listed in KNOWN_LICENSES with where its license was found.

Usage: nuget_licenses.py --assets <file> --used-assemblies <file>
                         --license <file> --output <file>
                         [--dotnet-root <directory>] [--packages <directory>] [--append]

--license is the license of the application itself (the Apache License 2.0,
which the packages under that license share). --dotnet-root defaults to
$DOTNET_ROOT. With --append, the licenses are added to the end of the file.
"""

import argparse
import hashlib
import json
import os
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


def restored_target(assets):
    """The one target of project.assets.json that names a runtime identifier
    (`<framework>/<runtime identifier>`): the graph a self-contained publish
    resolves, before trimming decides what of it survives."""
    content = json.loads(assets.read_text(encoding="utf-8"))
    candidates = [key for key in content["targets"] if "/" in key]
    if len(candidates) != 1:
        raise SystemExit(f"expected one runtime-specific target in {assets}, found {candidates}: "
                          "restore with a runtime identifier first")
    return content, content["targets"][candidates[0]]


def package_assemblies(assets):
    """Every NuGet package of project.assets.json's one runtime-specific
    target, with the name (without the .dll) of every assembly it compiles
    against, runs or loads natively: not always the name of the package
    itself (Uno.WinUI.Runtime.Skia.X11 ships Uno.UI.Runtime.Skia.X11.dll)."""
    found = {}
    content, target = restored_target(assets)
    for key, library in target.items():
        if content["libraries"].get(key, {}).get("type") != "package":
            continue
        assemblies = {
            Path(file).stem
            for group in ("compile", "runtime", "native")
            for file in library.get(group, {})
            if file.lower().endswith(".dll")
        }
        name, version = key.split("/", 1)
        found[name] = (version, assemblies)
    return found


def shipped(assets, used_assemblies):
    """The packages of project.assets.json with at least one assembly in
    `used_assemblies` (see --used-assemblies): trimming may have decided that
    the rest of what was restored, project references and the packages that
    exist on their own account for the build (such as Native AOT's compiler),
    is not part of the published application."""
    return {
        name: version
        for name, (version, assemblies) in package_assemblies(assets).items()
        if assemblies & used_assemblies
    }


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
    parser.add_argument("--assets", required=True, type=Path, help="the obj/project.assets.json of the project")
    parser.add_argument("--used-assemblies", required=True, type=Path,
                         help="assembly names the publish kept, one per line (see shared/tools/mstat-assemblies)")
    parser.add_argument("--dotnet-root", type=Path, default=os.environ.get("DOTNET_ROOT"),
                         help="the installation of the .NET SDK that published the application (default: $DOTNET_ROOT)")
    parser.add_argument("--license", required=True, type=Path, help="the Apache License 2.0")
    parser.add_argument("--packages", type=Path, default=Path.home() / ".nuget" / "packages", help="the NuGet cache")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--append", action="store_true", help="add to the end of the output file")
    arguments = parser.parse_args()
    if not arguments.dotnet_root:
        parser.error("--dotnet-root is required: $DOTNET_ROOT is not set")

    used_assemblies = set(arguments.used_assemblies.read_text(encoding="utf-8").split())
    names = shipped(arguments.assets, used_assemblies)

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

    license_text = (arguments.dotnet_root / "LICENSE.txt").read_text(encoding="utf-8", errors="replace").strip()
    sections.append(f"The .NET runtime that the application bundles (MIT)\n\n{license_text}")
    files.append(("The third-party notices of the .NET runtime",
                  (arguments.dotnet_root / "ThirdPartyNotices.txt").read_text(encoding="utf-8", errors="replace")))

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
