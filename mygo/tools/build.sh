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

# Builds the application.
#
# Usage: tools/build.sh [--e2e] <output>
#
# With --e2e the build includes the end-to-end scenarios (`go build -tags e2e`),
# which run in the application when VFOX_PLUS_E2E_SCENARIO is set; release
# builds must not have them. Run tools/prepare.sh first: the application embeds
# what it generates.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tags=(mygo_noinspector)
if [[ "${1:-}" == --e2e ]]; then
  tags+=(e2e)
  shift
fi
output="${1:?Usage: build.sh [--e2e] <output>}"
mkdir -p "$(dirname "$output")"
# The output is relative to where the script was started, not to the module.
output="$(cd "$(dirname "$output")" && pwd)/$(basename "$output")"
cd "$ROOT"
# mygo needs no C compiler: it calls GTK through purego. The tag leaves out the
# inspector of the native user interface, and production=1 what mygo keeps for
# development; -s -w leave out the symbols and the debug information. The
# executable is position independent, as the packages of Debian and Ubuntu are
# hardened.
CGO_ENABLED=0 go build -trimpath -buildmode=pie \
  -tags "$(IFS=,; echo "${tags[*]}")" \
  -ldflags "-s -w -X github.com/egoist/mygo.production=1" -o "$output" .
