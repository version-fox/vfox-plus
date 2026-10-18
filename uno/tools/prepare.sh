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

# Generates the inputs of the application that are not committed, into
# src/VfoxPlus.App/Assets, where the build copies them next to the executable:
#
#   Fonts/MaterialSymbolsRounded-Outlined.ttf   the icons of shared/icons.json cut out
#   Fonts/MaterialSymbolsRounded-Filled.ttf     of Material Symbols Rounded by
#                                               shared/tools/icon_font.py, once as
#                                               outlines and once filled: Uno draws
#                                               text with static fonts, so the fill
#                                               of an icon, which animates, is a
#                                               cross-fade between the two;
#   Generated/icons.json                        their code points, by meaning;
#   Generated/i18n/*.arb                        the catalogues of shared/i18n, as they are;
#   Generated/licenses.txt                      the licenses of what the application is
#                                               built from, which it shows and the
#                                               packages install: here those of the
#                                               code that is ported and of the icon
#                                               font; tools/publish.sh adds those of
#                                               the NuGet packages and of the .NET
#                                               runtime that the published
#                                               application ships.
#
# The variable font is taken from VFOX_PLUS_SYMBOLS (the sandbox images have it)
# or downloaded once into ~/.cache/vfox-plus/material-symbols. Run it before
# building and testing the application:
#
#   tools/prepare.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SHARED="$ROOT/../shared"
APP="$ROOT/src/VfoxPlus.App"
ASSETS="$APP/Assets"

symbols="${VFOX_PLUS_SYMBOLS:-}"
if [[ -z "$symbols" ]]; then
  symbols="${XDG_CACHE_HOME:-$HOME/.cache}/vfox-plus/material-symbols"
  python3 "$SHARED/tools/icon_font.py" download "$symbols"
fi

rm -rf "$ASSETS/Generated"
mkdir -p "$ASSETS/Fonts" "$ASSETS/Generated/i18n"
# The user interface draws every icon at weight 400, grade 0 and optical size
# 24, and fills some of them.
for fill in 0 1; do
  name=Outlined
  [[ "$fill" == 1 ]] && name=Filled
  python3 "$SHARED/tools/icon_font.py" subset --fonts "$symbols" \
    --output "$ASSETS/Fonts/MaterialSymbolsRounded-$name.ttf" --json "$ASSETS/Generated/icons.json" \
    --pin wght=400 --pin GRAD=0 --pin opsz=24 --pin "FILL=$fill"
done
cp "$SHARED"/i18n/*.arb "$ASSETS/Generated/i18n/"

{
  cat "$ROOT/NOTICE.txt"
  printf '\n\n================================================================================\n'
  printf 'Material Symbols Rounded (the icon font)\n'
  printf '================================================================================\n\n'
  cat "$symbols/LICENSE"
  printf '\n'
} > "$ASSETS/Generated/licenses.txt"
echo "Prepared $ASSETS"
