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

"""Checks the licenses of the Rust crates that an implementation ships and
collects their texts into one file, which the application shows and the
packages install.

The crates are those that the package depends on, directly or through other
crates, as normal dependencies on Linux: build-only dependencies do not ship.
Every crate must be licensed under terms compatible with, or no more
restrictive than, the Apache License 2.0 (docs/architecture.md#dependencies).
Of the licenses a crate offers, the one with the shortest notice is used, and
crates whose license texts are identical share one section. Crates under the
file-based weak copyleft of the Mozilla Public License 2.0 are only accepted
unmodified, when they offer nothing else, with a note on where their source
code is, as section 3.2 of that license asks.

Usage: cargo_licenses.py [--offline] [--append] --manifest-path <Cargo.toml> --output <file>

With --append, the licenses are added to the end of the file, such as the
licenses of the npm packages of a web front end.

Run `cargo fetch` first, so that the sources of the crates, which hold the
license texts, are in the cargo cache.
"""

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

# The licenses compatible with the Apache License 2.0, most preferred first:
# when a crate offers several, the shortest notice wins.
PREFERENCE = [
    "MIT",
    "MIT-0",
    "0BSD",
    "ISC",
    "BSD-2-Clause",
    "BSD-3-Clause",
    "Zlib",
    "BSL-1.0",
    "Unicode-3.0",
    "Unicode-DFS-2016",
    "CC0-1.0",
    "Unlicense",
    "Apache-2.0",
    "MPL-2.0",
]

# Words in the names of license files and the licenses they hold.
FILE_HINTS = {
    "MIT": {"MIT", "MIT-0"},
    "APACHE": {"Apache-2.0"},
    "BSD": {"BSD-2-Clause", "BSD-3-Clause", "0BSD"},
    "ZLIB": {"Zlib"},
    "UNICODE": {"Unicode-3.0", "Unicode-DFS-2016"},
    "ISC": {"ISC"},
    "BOOST": {"BSL-1.0"},
    "BSL": {"BSL-1.0"},
    "CC0": {"CC0-1.0"},
}

LICENSE_FILE = re.compile(r"^(LICEN[CS]E|COPYING|COPYRIGHT|NOTICE|UNLICENSE)", re.IGNORECASE)


class LicenseError(Exception):
    """A license expression that cannot be read or is not allowed."""


def tokens(expression):
    """The tokens of an SPDX expression; `/`, which old crates use, means OR."""
    return re.findall(r"\(|\)|[^\s()/]+|/", expression.replace("/", " / "))


def choose(expression):
    """The licenses to comply with for `expression`, or LicenseError."""
    items = ["OR" if token == "/" else token for token in tokens(expression)]
    position = 0

    def peek():
        return items[position] if position < len(items) else None

    def take():
        nonlocal position
        position += 1
        return items[position - 1]

    def rank(choice):
        return sum(PREFERENCE.index(license) for license in choice) + len(choice)

    def alternatives():
        options = [conjunction()]
        while peek() == "OR":
            take()
            options.append(conjunction())
        allowed = [option for option in options if option is not None]
        return min(allowed, key=rank) if allowed else None

    def conjunction():
        parts = [factor()]
        while peek() == "AND":
            take()
            parts.append(factor())
        if any(part is None for part in parts):
            return None
        return frozenset().union(*parts)

    def factor():
        token = take() if peek() is not None else None
        if token is None or token in ("OR", "AND", "WITH", ")"):
            raise LicenseError(f"cannot read the license expression {expression!r}")
        if token == "(":
            inner = alternatives()
            if take() != ")":
                raise LicenseError(f"unbalanced parentheses in {expression!r}")
            return inner
        # Exceptions such as LLVM-exception only add permissions.
        if peek() == "WITH":
            take()
            take()
        license = token.removesuffix("+")
        return frozenset([license]) if license in PREFERENCE else None

    choice = alternatives()
    if peek() is not None:
        raise LicenseError(f"cannot read the license expression {expression!r}")
    if choice is None:
        raise LicenseError(f"{expression} is not compatible with the Apache License 2.0")
    return choice


def license_files(directory, choice):
    """The license files of a crate for the licenses of `choice`."""
    files = sorted(path for path in directory.iterdir() if path.is_file() and LICENSE_FILE.match(path.name))

    def hints(path):
        name = path.name.upper()
        return set().union(*(licenses for word, licenses in FILE_HINTS.items() if word in name))

    chosen = [path for path in files if not hints(path) or hints(path) & choice]
    # NOTICE files belong to the Apache License.
    chosen = [path for path in chosen if not path.name.upper().startswith("NOTICE") or "Apache-2.0" in choice]
    # A crate may only ship the texts of its other licenses: then those apply.
    return chosen or files


def shipped_packages(metadata):
    """The packages that the root package depends on as normal dependencies."""
    nodes = {node["id"]: node for node in metadata["resolve"]["nodes"]}
    root = metadata["resolve"]["root"]
    seen, pending = set(), [root]
    while pending:
        current = pending.pop()
        for dependency in nodes[current]["deps"]:
            normal = any(kind["kind"] is None for kind in dependency["dep_kinds"])
            if normal and dependency["pkg"] not in seen:
                seen.add(dependency["pkg"])
                pending.append(dependency["pkg"])
    packages = {package["id"]: package for package in metadata["packages"]}
    return [packages[identifier] for identifier in seen if identifier != root]


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--manifest-path", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--offline", action="store_true", help="do not access the network")
    parser.add_argument("--append", action="store_true", help="add to the end of the output file")
    arguments = parser.parse_args()

    command = [
        "cargo",
        "metadata",
        "--format-version",
        "1",
        "--locked",
        "--filter-platform",
        "x86_64-unknown-linux-gnu",
        "--manifest-path",
        str(arguments.manifest_path),
    ]
    if arguments.offline:
        command.append("--offline")
    metadata = json.loads(subprocess.run(command, check=True, capture_output=True, text=True).stdout)

    errors = []
    sections = {}
    for package in sorted(shipped_packages(metadata), key=lambda package: (package["name"], package["version"])):
        # name@version, the package specification of cargo. "name version"
        # would read as a version of zlib to lintian for crates such as
        # fdeflate, and the notices are part of the executables.
        title = f"{package['name']}@{package['version']}"
        directory = Path(package["manifest_path"]).parent
        expression = package.get("license")
        if package.get("license_file"):
            files = [directory / package["license_file"]]
            expression = expression or "custom license"
        elif expression is None:
            errors.append(f"{title} declares no license")
            continue
        else:
            try:
                files = license_files(directory, choose(expression))
            except LicenseError as error:
                errors.append(f"{title}: {error}")
                continue
        source = package.get("repository") or package.get("homepage") or f"https://crates.io/crates/{package['name']}"
        if files:
            text = "\n\n".join(path.read_text(encoding="utf-8", errors="replace").strip() for path in files)
        else:
            text = f"The crate does not include the text of its license; see {source}."
        if expression and "MPL-2.0" in choose(expression):
            text = (
                f"The unmodified source code of {title} is available at "
                f"https://crates.io/crates/{package['name']}/{package['version']} and {source}.\n\n{text}"
            )
        sections.setdefault(text, []).append(f"{title} ({expression})")
    if errors:
        print("Crates with licenses that cannot be shipped:\n  " + "\n  ".join(errors), file=sys.stderr)
        sys.exit(1)

    separator = "\n\n" + "-" * 72 + "\n\n"
    rendered = separator.join(sorted("\n".join(crates) + "\n\n" + text for text, crates in sections.items())) + "\n"
    arguments.output.parent.mkdir(parents=True, exist_ok=True)
    if arguments.append and arguments.output.exists():
        rendered = arguments.output.read_text(encoding="utf-8").rstrip("\n") + separator + rendered
    arguments.output.write_text(rendered, encoding="utf-8")
    crates = sum(len(names) for names in sections.values())
    print(f"Wrote {arguments.output} ({crates} crates, {len(sections)} license texts)")


if __name__ == "__main__":
    main()
