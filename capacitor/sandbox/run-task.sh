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

# Runs one task inside a sandbox container of the Capacitor implementation.
# sandbox.sh starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/capacitor.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
APP="$WORK/capacitor"
ELECTRON="$APP/node_modules/electron/dist/electron"
# What the build generates.
COPY_EXCLUDES+=(
  --exclude=./capacitor/dist --exclude=./capacitor/coverage
  --exclude=./capacitor/public/fonts --exclude=./capacitor/public/licenses.txt
)

# prepare: the packages and the Electron runtime from the caches, then the icon
# font and the licenses.
prepare() {
  copy_repository web capacitor
  as_tester "$APP" npm ci --no-audit --no-fund --offline
  as_tester "$APP" npm run electron
  as_tester "$APP" npm run assets
}

# in_display <command...>: runs a command as tester on the virtual display.
in_display() {
  as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" "$@"
}

case "${1:-}" in
  deps)
    copy_repository web capacitor
    as_tester "$APP" npm ci --no-audit --no-fund ${SANDBOX_NPM_REGISTRY:+--registry="$SANDBOX_NPM_REGISTRY"}
    # Downloads the Electron runtime into the cache, checked against the
    # checksums of the electron package.
    as_tester "$APP" env ${SANDBOX_ELECTRON_MIRROR:+ELECTRON_MIRROR="$SANDBOX_ELECTRON_MIRROR"} npm run electron
    ;;

  check)
    prepare
    as_tester "$APP" npm run format:check
    as_tester "$APP" npm run typecheck
    as_tester "$APP" npm run lint
    as_tester "$APP" npm run test:coverage
    ;;

  e2e)
    prepare
    as_tester "$APP" npm run build:e2e
    start_display
    in_display bash web/e2e/run.sh fake "$ELECTRON" "$APP/dist/app-e2e"
    in_display env VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox bash web/e2e/run.sh real "$ELECTRON" "$APP/dist/app-e2e"
    stop_display
    ;;

  measure)
    prepare
    as_tester "$APP" npm run build:e2e
    start_display
    in_display bash web/e2e/run.sh measure "$ELECTRON" "$APP/dist/app-e2e" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  screenshots)
    prepare
    as_tester "$APP" npm run build:e2e
    start_display
    in_display bash shared/tools/screenshots.sh /home/tester/screenshots "$ELECTRON" "$APP/dist/app-e2e"
    publish /home/tester/screenshots/*.png
    stop_display
    ;;

  package)
    prepare
    as_tester "$APP" npm run build
    as_tester "$APP" npm run stage
    smoke_test "$APP/dist/stage/vfox-plus" runuser -u tester -- env HOME=/home/tester
    install -d -o tester -g tester /home/tester/dist
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" bash tools/package.sh /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    install_deb_test vfox-plus-capacitor /usr/lib/vfox-plus/vfox-plus /usr/lib/vfox-plus/chrome_crashpad_handler
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use capacitor/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
