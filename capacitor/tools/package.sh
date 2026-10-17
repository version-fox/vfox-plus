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

# Packages the release that `npm run build && npm run stage` assembled in
# dist/stage as vfox-plus-capacitor_<version>_amd64.deb and a .tar.gz, with
# shared/packaging/linux/build-deb.sh. The sandbox (sandbox/run-task.sh) and
# CI (.github/workflows/capacitor.yml) both package with this script.
#
# Usage: capacitor/tools/package.sh <output directory>
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${1:?Usage: package.sh <output directory>}"

# The libraries that the Electron runtime links on Ubuntu 26.04, gdbus for the
# accent colour and xdg-open for links. The installation test of the sandbox
# and of CI checks that they are complete.
DEPENDS="libasound2t64, libatk-bridge2.0-0t64, libatk1.0-0t64, libatspi2.0-0t64, libc6, libcairo2, libcups2t64"
DEPENDS+=", libdbus-1-3, libexpat1, libgbm1, libgcc-s1, libglib2.0-0t64, libglib2.0-bin, libgtk-3-0t64, libnspr4"
DEPENDS+=", libnss3, libpango-1.0-0, libudev1, libx11-6, libxcb1, libxcomposite1, libxdamage1, libxext6, libxfixes3"
DEPENDS+=", libxkbcommon0, libxrandr2, xdg-utils"

# Electron is Chromium and Node.js, which link most of their third-party
# libraries statically, as Debian does not want; it cannot use those of the
# system. Chromium confines its processes with a helper that runs as root.
overrides=(--override "elevated-privileges 4755 root/root [usr/lib/vfox-plus/chrome-sandbox]")
for library in freetype lcms2 libjpeg openjpeg srtp tiff; do
  overrides+=(--override "embedded-library $library [usr/lib/vfox-plus/vfox-plus]")
done

[[ -x "$APP_DIR/dist/stage/vfox-plus" ]] || { echo "Run npm run build && npm run stage first." >&2; exit 1; }
bash "$APP_DIR/../shared/packaging/linux/build-deb.sh" \
  --package vfox-plus-capacitor --version "$(node -p "require('$APP_DIR/package.json').version")" \
  --stage "$APP_DIR/dist/stage" --executable vfox-plus --setuid chrome-sandbox --strip \
  --technology "Ionic and Capacitor" --depends "$DEPENDS" \
  --notice "$APP_DIR/dist/app/renderer/licenses.txt" "${overrides[@]}" "$OUTPUT"
