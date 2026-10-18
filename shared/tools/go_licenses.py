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

"""Checks the licenses of the Go modules that an implementation compiles into
its executable, and of the Go standard library, and collects their texts into
one file, which the application shows and the packages install.

The modules are those of the packages that the main package imports, directly
or indirectly, with the given build tags. Go modules declare no license, so it
is recognised from the text of the license file. Every module must be licensed
under terms compatible with, or no more restrictive than, the Apache License
2.0 (docs/architecture.md#dependencies).

Usage: go_licenses.py [--tags TAGS] [--append] --module <directory> --output <file>

With --append, the licenses are added to the end of the file, such as the
licenses of the npm packages of a web front end. Run `go mod download` first.
"""

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

LICENSE_FILE = re.compile(r"^(LICEN[CS]E|COPYING|NOTICE)", re.IGNORECASE)

# Phrases that identify the licenses compatible with the Apache License 2.0,
# checked in order: the first that matches names the license.
SIGNATURES = [
    ("Apache-2.0", [r"Apache License", r"Version 2\.0"]),
    ("MIT", [r"Permission is hereby granted, free of charge"]),
    ("BSD-3-Clause", [r"Redistribution and use in source and binary forms", r"Neither the name|names of its contributors"]),
    ("BSD-2-Clause", [r"Redistribution and use in source and binary forms"]),
    ("ISC", [r"Permission to use, copy, modify, and(/or)? distribute this software for any purpose"]),
]


def recognise(text):
    """The license of `text`, or None."""
    flat = " ".join(text.split())
    for name, phrases in SIGNATURES:
        if all(re.search(phrase, flat) for phrase in phrases):
            return name
    return None


def modules(directory, tags):
    """The modules of the packages that the main package of `directory` builds
    with, and whether it uses the standard library."""
    command = ["go", "list", "-deps", "-json"]
    if tags:
        command += ["-tags", tags]
    output = subprocess.run(command + ["."], cwd=directory, check=True, capture_output=True, text=True).stdout
    found, standard = {}, False
    decoder = json.JSONDecoder()
    position = 0
    while position < len(output):
        while position < len(output) and output[position].isspace():
            position += 1
        if position == len(output):
            break
        package, position = decoder.raw_decode(output, position)
        if package.get("Standard"):
            standard = True
        module = package.get("Module")
        if module and not module.get("Main"):
            found[module["Path"]] = module
    return list(found.values()), standard


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--module", required=True, type=Path, help="the directory of the main package")
    parser.add_argument("--tags", default="", help="the build tags")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--append", action="store_true", help="add to the end of the output file")
    arguments = parser.parse_args()

    found, standard = modules(arguments.module, arguments.tags)
    entries = []
    if standard:
        root = Path(subprocess.run(["go", "env", "GOROOT"], check=True, capture_output=True, text=True).stdout.strip())
        version = subprocess.run(["go", "env", "GOVERSION"], check=True, capture_output=True, text=True).stdout.strip()
        entries.append((f"The Go standard library and runtime, {version}", [root / "LICENSE"]))
    for module in sorted(found, key=lambda module: module["Path"]):
        directory = Path(module["Dir"])
        files = sorted(path for path in directory.iterdir() if path.is_file() and LICENSE_FILE.match(path.name))
        entries.append((f"{module['Path']}@{module['Version']}", files))

    errors, sections = [], []
    for title, files in entries:
        licenses = {recognise(path.read_text(encoding="utf-8", errors="replace")) for path in files} - {None}
        if not files or not licenses:
            errors.append(f"{title}: no license file with a license compatible with the Apache License 2.0")
            continue
        text = "\n\n".join(path.read_text(encoding="utf-8", errors="replace").strip() for path in files)
        sections.append(f"{title} ({' AND '.join(sorted(licenses))})\n\n{text}")
    if errors:
        print("Modules with licenses that cannot be shipped:\n  " + "\n  ".join(errors), file=sys.stderr)
        sys.exit(1)

    separator = "\n\n" + "-" * 72 + "\n\n"
    rendered = separator.join(sections) + "\n"
    arguments.output.parent.mkdir(parents=True, exist_ok=True)
    if arguments.append and arguments.output.exists():
        rendered = arguments.output.read_text(encoding="utf-8").rstrip("\n") + separator + rendered
    arguments.output.write_text(rendered, encoding="utf-8")
    print(f"Wrote {arguments.output} ({len(sections)} licenses of Go modules and the standard library)")


if __name__ == "__main__":
    main()
