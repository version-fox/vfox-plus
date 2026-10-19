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

# Reports what the unit tests cover, by area, from a coverage profile, and
# fails when the lines of core and state, which have no user interface, are
# covered less than the minimum, or the lines of the user interface, which the
# tests of its components and pages cover, less than 75 %. The end-to-end
# scenarios do not count here.
#
# Usage: tools/coverage.sh <profile> [minimum percent of core and state]
set -euo pipefail

profile="${1:?Usage: coverage.sh <profile> [minimum percent]}"
minimum="${2:-85}"
awk -v minimum="$minimum" -v minimum_ui=75 '
  NR > 1 {
    split($1, location, ":")
    file = location[1]
    area = "other"
    if (file ~ /\/mygo\/core\//) area = "core"
    else if (file ~ /\/mygo\/state\//) area = "state"
    else if (file ~ /\/mygo\/ui\//) area = "ui"
    statements[area] += $2
    if ($3 > 0) covered[area] += $2
  }
  END {
    status = 0
    for (area in statements) {
      printf "%-6s %5.1f %% of %d statements\n", area, covered[area] / statements[area] * 100, statements[area]
    }
    gated = statements["core"] + statements["state"]
    percent = (covered["core"] + covered["state"]) / gated * 100
    printf "core and state: %.1f %% (minimum %s %%)\n", percent, minimum
    if (percent < minimum) status = 1
    ui_percent = covered["ui"] / statements["ui"] * 100
    printf "ui: %.1f %% (minimum %s %%)\n", ui_percent, minimum_ui
    if (ui_percent < minimum_ui) status = 1
    exit status
  }
' "$profile"
