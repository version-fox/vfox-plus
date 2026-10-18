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

# Takes screenshots of every page: runs the "screenshots" scenario of an
# application built with the end-to-end tests (web/src/e2e/scenarios.ts, or
# the Rust port of the Dioxus implementation) against the fake vfox of
# shared/fixtures, with a new, empty HOME, and captures the screen with
# ImageMagick whenever the application prints "e2e: SHOT <name>". A second run
# with a slow fake vfox shows an installation in progress.
#
# Usage: shared/tools/screenshots.sh <output directory> <application command...>
#
# The application runs on the current display, which should be 1280 × 800.
set -euo pipefail

SHARED="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${1:?Usage: screenshots.sh <output directory> <application command...>}"
shift
[[ $# -gt 0 ]] || { echo "Usage: screenshots.sh <output directory> <application command...>" >&2; exit 2; }
APPLICATION=("$@")
mkdir -p "$OUTPUT"

# run [variable=value...]: runs the scenario once and captures its shots.
run() {
  local sandbox status=0 printed=0 lines line
  sandbox="$(mktemp -d "${TMPDIR:-/tmp}/vfox-plus-screenshots.XXXXXX")"
  mkdir -p "$sandbox/home/.config/vfox-plus"
  install -m 755 "$SHARED/fixtures/fake-vfox.sh" "$sandbox/vfox"
  printf '{\n  "language": "english",\n  "vfoxExecutable": "%s"\n}\n' "$sandbox/vfox" \
    > "$sandbox/home/.config/vfox-plus/settings.json"
  env -u VFOX_HOME -u XDG_CONFIG_HOME HOME="$sandbox/home" LANG=C.UTF-8 VFOX_PLUS_E2E_SCENARIO=screenshots \
    VFOX_PLUS_FIXTURES="$SHARED/fixtures/vfox-1.0.13" "$@" "${APPLICATION[@]}" > "$sandbox/app.log" 2>&1 &
  local application=$!
  while kill -0 "$application" 2> /dev/null; do
    lines="$(wc -l < "$sandbox/app.log")"
    if [[ $lines -gt $printed ]]; then
      while IFS= read -r line; do
        case "$line" in
          "e2e: SHOT "*)
            import -window root "$OUTPUT/${line#e2e: SHOT }.png"
            echo "${line#e2e: SHOT }.png"
            ;;
          "e2e: "*) echo "$line" ;;
        esac
      done < <(sed -n "$((printed + 1)),${lines}p" "$sandbox/app.log")
      printed=$lines
    fi
    sleep 0.2
  done
  wait "$application" || status=$?
  if [[ $status -ne 0 ]]; then
    cat "$sandbox/app.log" >&2
  fi
  rm -rf "$sandbox"
  return "$status"
}

run
run VFOX_PLUS_FAKE_DELAY=4
