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

"""Reports the line coverage of the libraries of the application from the
Cobertura file that `dotnet test` writes with the XPlat Code Coverage collector
(see tests/coverlet.runsettings), and fails when the gated libraries are below
the minimum.

Cobertura lists the lines of a file once for every class that has code in it,
and the compiler-made classes of lambdas and iterators have the lines of the
methods they come from, so the lines are merged by file and line number: a line
is covered when any class that has it covered it.

Usage: coverage.py <cobertura file> <minimum percent> <gated assembly>...

The assemblies that are not named are reported, and not gated.
"""

import sys
import xml.etree.ElementTree as ET


def main():
    if len(sys.argv) < 4:
        sys.exit(__doc__)
    path, minimum, gated = sys.argv[1], float(sys.argv[2]), set(sys.argv[3:])
    lines = {}
    for package in ET.parse(path).getroot().iter("package"):
        name = package.get("name")
        covered = lines.setdefault(name, {})
        for cls in package.iter("class"):
            for line in cls.iter("line"):
                key = (cls.get("filename"), int(line.get("number")))
                covered[key] = covered.get(key, 0) + int(line.get("hits"))

    def percent(names):
        total = sum(len(lines[name]) for name in names)
        hit = sum(1 for name in names for count in lines[name].values() if count > 0)
        return (100.0 * hit / total) if total else 100.0, hit, total

    for name in sorted(lines):
        share, hit, total = percent([name])
        print(f"{name:24} {share:5.1f} %  ({hit} of {total} lines){'  gated' if name in gated else ''}")
    unknown = gated - set(lines)
    if unknown:
        sys.exit(f"No coverage for {', '.join(sorted(unknown))}.")
    share, hit, total = percent(sorted(gated))
    print(f"{'gated, together':24} {share:5.1f} %  ({hit} of {total} lines)")
    if share < minimum:
        sys.exit(f"The coverage of {', '.join(sorted(gated))} is {share:.1f} %, below the minimum of {minimum:g} %.")


if __name__ == "__main__":
    main()
