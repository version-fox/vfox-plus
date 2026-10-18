#!/usr/bin/env bash
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

# Publishes the application as a directory that runs on its own: the .NET
# runtime is bundled, the code is compiled ahead of time (the application starts
# in half a second instead of one) and trimmed to what it uses (under 90 MB
# instead of 158 MB), and what a release does not need is left out.
#
# Usage: tools/publish.sh [--e2e] <output directory>
#
#   --e2e   adds the agent of the end-to-end tests (src/VfoxPlus.App/E2E), which
#           runs the scenario that VFOX_PLUS_E2E_SCENARIO names
#
# Run tools/prepare.sh first. The licenses of the NuGet packages and of the .NET
# runtime that the published application ships are added to its licenses.txt. The
# packages must be restored for the runtime of the machine, in the
# release configuration (`dotnet restore src/VfoxPlus.App --runtime linux-x64
# -p:Configuration=Release`); this works offline then.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SHARED="$ROOT/../shared"
APP="$ROOT/src/VfoxPlus.App"

e2e=false
if [[ "${1:-}" == --e2e ]]; then
  e2e=true
  shift
fi
output="${1:?Usage: publish.sh [--e2e] <output directory>}"
case "$(uname -m)" in
  x86_64) runtime=linux-x64 ;;
  aarch64) runtime=linux-arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

rm -rf "$output"
dotnet publish "$APP" --configuration Release --framework net10.0-desktop --runtime "$runtime" --self-contained \
  --no-restore -p:E2E="$e2e" --output "$output" --nologo

# What the runtime ships for debuggers and diagnostics, and for an alternative
# garbage collector that nothing asks for.
rm -f "$output/createdump" "$output/libmscordbi.so" "$output/libmscordaccore.so" \
  "$output/libcoreclrtraceptprovider.so" "$output/libclrgc.so" "$output/libclrgcexp.so"
# The fonts that the themes of Uno bring (Fluent icons, and the text fonts Inter,
# Open Sans and Roboto) are not used: the text is set in the fonts of the desktop
# and the icons come from the icon font of the application. The screenshots of
# every page are the same pixel for pixel without them.
rm -rf "$output"/Uno.Fonts.*

# Only the application is executable: the libraries and the data are not, whatever
# the build gave them.
find "$output" -type f -exec chmod 644 {} +
chmod 755 "$output/VfoxPlus"

if [[ "$e2e" == false ]]; then
  python3 "$SHARED/tools/nuget_licenses.py" --deps "$output/VfoxPlus.deps.json" \
    --license "$ROOT/../LICENSE" --output "$output/Assets/Generated/licenses.txt" --append
fi
echo "Published to $output ($(du -sh "$output" | cut -f1))"
