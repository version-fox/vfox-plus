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
# assets/, where the application embeds them:
#
#   fonts/MaterialSymbolsRounded.ttf        the icons of shared/icons.json cut out
#                                           of Material Symbols Rounded by
#                                           shared/tools/icon_font.py, outlined;
#   fonts/MaterialSymbolsRoundedFilled.ttf  the same icons, filled (mygo draws a
#                                           font at its weight and cannot set its
#                                           fill axis, so the fill that the user
#                                           interface animates is the cross-fade
#                                           of the two);
#   icons.json                              their code points, by meaning;
#   i18n/*.arb                              the catalogues of shared/i18n, as they
#                                           are;
#   licenses.txt                            the licenses of what the application is
#                                           built from, which it shows, and the
#                                           packages install.
#
# The variable font is taken from VFOX_PLUS_SYMBOLS (the sandbox images have it)
# or downloaded once into ~/.cache/vfox-plus/material-symbols. Run it before
# building, testing and linting the application:
#
#   tools/prepare.sh
#
# The licenses of the Go modules need them in the module cache: it runs
# `go mod download` for them, unless MYGO_SKIP_LICENSES is set, which the checks
# that do not ship the application use.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SHARED="$ROOT/../shared"
ASSETS="$ROOT/assets"

symbols="${VFOX_PLUS_SYMBOLS:-}"
if [[ -z "$symbols" ]]; then
  symbols="${XDG_CACHE_HOME:-$HOME/.cache}/vfox-plus/material-symbols"
  python3 "$SHARED/tools/icon_font.py" download "$symbols"
fi

rm -rf "$ASSETS/i18n"
mkdir -p "$ASSETS/fonts" "$ASSETS/i18n"
# The user interface draws every icon at weight 400, grade 0 and optical size
# 24, outlined or filled.
python3 "$SHARED/tools/icon_font.py" subset --fonts "$symbols" \
  --output "$ASSETS/fonts/MaterialSymbolsRounded.ttf" --json "$ASSETS/icons.json" \
  --pin wght=400 --pin GRAD=0 --pin opsz=24 --pin FILL=0
python3 "$SHARED/tools/icon_font.py" subset --fonts "$symbols" \
  --output "$ASSETS/fonts/MaterialSymbolsRoundedFilled.ttf" \
  --pin wght=400 --pin GRAD=0 --pin opsz=24 --pin FILL=1
python3 "$ROOT/tools/rename_font.py" "$ASSETS/fonts/MaterialSymbolsRoundedFilled.ttf" "Material Symbols Rounded Filled"
cp "$SHARED"/i18n/*.arb "$ASSETS/i18n/"

{
  cat "$ROOT/NOTICE.txt"
  printf '\n\n================================================================================\n'
  printf 'Material Symbols Rounded (the icon fonts)\n'
  printf '================================================================================\n\n'
  cat "$symbols/LICENSE"
  printf '\n'
} > "$ASSETS/licenses.txt"
if [[ -z "${MYGO_SKIP_LICENSES:-}" ]]; then
  (cd "$ROOT" && go mod download)
  python3 "$SHARED/tools/go_licenses.py" --module "$ROOT" --output "$ASSETS/licenses.txt" --append
fi
echo "Prepared $ASSETS"
