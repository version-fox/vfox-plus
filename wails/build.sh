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

# Builds vfox+ with Wails: the web front end of ../web for Wails, copied into
# frontend/dist, then the Go binary that embeds it.
#
# Usage: wails/build.sh [--e2e] [output]
#
#   --e2e    builds the front end with the end-to-end test runner
#            (VFOX_PLUS_E2E=1), for web/e2e/run.sh
#   output   the binary to write, by default build/vfox-plus
#
# The front end needs its packages and assets first:
# `npm ci && npm run assets` in ../web.
set -euo pipefail

WAILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="$WAILS_DIR/../web"
script=build:wails
dist=wails
if [[ "${1:-}" == --e2e ]]; then
  script=build:e2e:wails
  dist=wails-e2e
  shift
fi
output="${1:-$WAILS_DIR/build/vfox-plus}"

npm --prefix "$WEB_DIR" run "$script"
rm -rf "$WAILS_DIR/frontend/dist"
cp -r "$WEB_DIR/dist/$dist" "$WAILS_DIR/frontend/dist"
mkdir -p "$(dirname "$output")"
# The production tag removes the developer tools of Wails.
(cd "$WAILS_DIR" && CGO_ENABLED=1 CGO_CFLAGS="${CGO_CFLAGS:--O2 -g} -Wno-deprecated-declarations" go build -trimpath -tags production -ldflags "-s -w" -o "$output" .)
echo "Built $output"
