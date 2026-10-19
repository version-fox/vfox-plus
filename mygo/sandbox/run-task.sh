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

# Runs one task inside a sandbox container of the mygo implementation.
# sandbox.sh starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/mygo.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
APP="$WORK/mygo"
BINARY=/home/tester/build/vfox-plus
# What the build generates.
COPY_EXCLUDES+=(
  --exclude=./mygo/build --exclude=./mygo/coverage.out
  --exclude=./mygo/assets/fonts --exclude=./mygo/assets/i18n --exclude=./mygo/assets/icons.json
  --exclude=./mygo/assets/licenses.txt
)
# Runtime dependencies of the release binary on Ubuntu 26.04; the installation
# test checks that they are complete. The executable is linked to the C library
# alone: mygo loads GTK 3, which brings GLib, Pango, Cairo and the rest, when it
# starts (docs/architecture.md in mygo, "Native interop without cgo"). Fonts:
# Ubuntu Sans, or any other sans-serif font of the desktop.
DEPENDS="libc6, libgtk-3-0t64, libglib2.0-0t64, libpango-1.0-0, libpangoft2-1.0-0, libcairo2, libgdk-pixbuf-2.0-0, fontconfig, fonts-ubuntu | fonts-noto-core"
# Chinese text needs a CJK font, the file choosers go through the desktop portal
# where there is one, and links open in the browser.
RECOMMENDS="fonts-noto-cjk, xdg-desktop-portal, xdg-utils"

# Only the deps task downloads modules; with GOPROXY=off a missing module is an
# error rather than an attempt to download it.
go_offline=(env GOPROXY=off GOFLAGS=-mod=readonly)

# prepare: the generated inputs. web/ comes along for its end-to-end runner,
# which every implementation shares. The licenses of the Go modules need the
# module cache, which only the tasks that ship the application use.
prepare() {
  copy_repository mygo web
  as_tester "$APP" "${go_offline[@]}" bash tools/prepare.sh
}

# prepare_light: the same without the licenses of the Go modules.
prepare_light() {
  copy_repository mygo web
  as_tester "$APP" env MYGO_SKIP_LICENSES=1 bash tools/prepare.sh
}

# in_display <command...>: runs a command as tester on the virtual display.
in_display() {
  as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" "$@"
}

# build <e2e|release> <output>: builds the application as tester.
build() {
  install -d -o tester -g tester "$(dirname "$2")"
  if [[ "$1" == e2e ]]; then
    as_tester "$APP" "${go_offline[@]}" bash tools/build.sh --e2e "$2"
  else
    as_tester "$APP" "${go_offline[@]}" bash tools/build.sh "$2"
  fi
}

case "${1:-}" in
  deps)
    copy_repository mygo web
    as_tester "$APP" env ${SANDBOX_GOPROXY:+GOPROXY="$SANDBOX_GOPROXY"} go mod download
    ;;

  check)
    prepare_light
    as_tester "$APP" "${go_offline[@]}" go mod verify
    as_tester "$APP" "${go_offline[@]}" golangci-lint fmt --diff
    as_tester "$APP" "${go_offline[@]}" golangci-lint run
    as_tester "$APP" "${go_offline[@]}" golangci-lint run --build-tags e2e
    as_tester "$APP" "${go_offline[@]}" go vet ./...
    # The race detector needs the C compiler; the application is built without.
    as_tester "$APP" "${go_offline[@]}" env CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./...
    as_tester "$APP" bash tools/coverage.sh coverage.out 85
    # The shell scripts of the implementation.
    as_tester "$APP" bash -n tools/prepare.sh tools/build.sh tools/coverage.sh
    as_tester "$APP" python3 -m py_compile tools/rename_font.py
    ;;

  build)
    prepare
    build release "$BINARY"
    ;;

  e2e)
    prepare_light
    build e2e "$BINARY"
    start_display
    in_display env VFOX_PLUS_E2E_TIMEOUT=300 bash web/e2e/run.sh fake "$BINARY"
    in_display env VFOX_PLUS_E2E_TIMEOUT=300 VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox bash web/e2e/run.sh real "$BINARY"
    stop_display
    ;;

  measure)
    prepare
    build e2e "$BINARY"
    start_display
    in_display bash shared/tools/measure.sh "$BINARY" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  screenshots)
    prepare
    build e2e "$BINARY"
    start_display
    in_display bash shared/tools/screenshots.sh /home/tester/screenshots "$BINARY"
    publish /home/tester/screenshots/*.png
    stop_display
    ;;

  package)
    prepare
    build release "$BINARY"
    smoke_test "$BINARY" runuser -u tester -- env HOME=/home/tester
    install -d -o tester -g tester /home/tester/stage /home/tester/dist
    install -o tester -m 755 "$BINARY" /home/tester/stage/vfox-plus
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" \
      bash "$WORK/shared/packaging/linux/build-deb.sh" \
      --package vfox-plus-mygo --version "$(tr -d '[:space:]' < "$APP/VERSION")" \
      --stage /home/tester/stage --executable vfox-plus --technology mygo \
      --depends "$DEPENDS" --recommends "$RECOMMENDS" --notice "$APP/assets/licenses.txt" /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    install_deb_test vfox-plus-mygo /usr/lib/vfox-plus/vfox-plus
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use mygo/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
