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

# Runs one task inside a sandbox container of the Wails implementation.
# sandbox.sh starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/wails.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
WEB="$WORK/web"
APP="$WORK/wails"
BINARY=/home/tester/build/vfox-plus
COPY_EXCLUDES+=(--exclude=./wails/build --exclude=./wails/frontend/dist)
# The C code of Wails calls GTK functions that GTK 4 deprecates.
export CGO_CFLAGS="-O2 -g -Wno-deprecated-declarations"
# Runtime dependencies of the release binary on Ubuntu 26.04; the installation
# test checks that they are complete.
DEPENDS="libwebkitgtk-6.0-4, libjavascriptcoregtk-6.0-1, libgtk-4-1, libsoup-3.0-0, libglib2.0-0t64 | libglib2.0-0, libx11-6, libc6"

# Only the deps task downloads modules; with GOPROXY=off a missing module is an
# error rather than an attempt to download it.
go_offline=(env GOPROXY=off GOFLAGS=-mod=readonly)

prepare() {
  copy_repository web wails
  as_tester "$WEB" npm ci --no-audit --no-fund --offline
  as_tester "$WEB" npm run assets
}

case "${1:-}" in
  deps)
    copy_repository web wails
    as_tester "$WEB" npm ci --no-audit --no-fund ${SANDBOX_NPM_REGISTRY:+--registry="$SANDBOX_NPM_REGISTRY"}
    as_tester "$APP" env ${SANDBOX_GOPROXY:+GOPROXY="$SANDBOX_GOPROXY"} go mod download
    ;;

  check)
    prepare
    as_tester "$APP" "${go_offline[@]}" bash build.sh /home/tester/build/vfox-plus
    as_tester "$APP" "${go_offline[@]}" go mod verify
    as_tester "$APP" "${go_offline[@]}" golangci-lint fmt --diff
    as_tester "$APP" "${go_offline[@]}" golangci-lint run
    as_tester "$APP" "${go_offline[@]}" go test -race ./...
    ;;

  e2e)
    prepare
    as_tester "$APP" "${go_offline[@]}" bash build.sh --e2e "$BINARY"
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash web/e2e/run.sh fake "$BINARY"
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox bash web/e2e/run.sh real "$BINARY"
    stop_display
    ;;

  measure)
    prepare
    as_tester "$APP" "${go_offline[@]}" bash build.sh --e2e "$BINARY"
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash web/e2e/run.sh measure "$BINARY" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  screenshots)
    prepare
    as_tester "$APP" "${go_offline[@]}" bash build.sh --e2e "$BINARY"
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash shared/tools/screenshots.sh /home/tester/screenshots "$BINARY"
    publish /home/tester/screenshots/*.png
    stop_display
    ;;

  package)
    prepare
    as_tester "$APP" "${go_offline[@]}" bash build.sh "$BINARY"
    smoke_test "$BINARY" runuser -u tester -- env HOME=/home/tester
    install -d -o tester -g tester /home/tester/stage /home/tester/dist
    install -o tester -m 755 "$BINARY" /home/tester/stage/vfox-plus
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" \
      bash "$WORK/shared/packaging/linux/build-deb.sh" \
      --package vfox-plus-wails --version "$(jq -r .version "$WEB/package.json")" \
      --stage /home/tester/stage --executable vfox-plus --technology Wails \
      --depends "$DEPENDS" --notice "$APP/frontend/dist/licenses.txt" /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    install_deb_test vfox-plus-wails /usr/lib/vfox-plus/vfox-plus
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use wails/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
