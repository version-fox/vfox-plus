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

# Runs one task inside a sandbox container of the Tauri implementation.
# sandbox.sh starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/tauri.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
WEB="$WORK/web"
APP="$WORK/tauri"
CRATE="$APP/src-tauri"
BINARY="$CRATE/target/release/vfox-plus"
COPY_EXCLUDES+=(--exclude=./tauri/src-tauri/gen --exclude=./tauri/src-tauri/icons)
# Runtime dependencies of the release binary on Ubuntu 26.04; the installation
# test checks that they are complete.
DEPENDS="libwebkit2gtk-4.1-0, libgtk-3-0t64 | libgtk-3-0, libglib2.0-0t64 | libglib2.0-0, libsoup-3.0-0, libjavascriptcoregtk-4.1-0, libcairo2, libgdk-pixbuf-2.0-0, libpango-1.0-0, libc6, libgcc-s1"

npm_install() {
  local directory
  for directory in "$WEB" "$APP"; do
    as_tester "$directory" npm ci --no-audit --no-fund "$@" ${SANDBOX_NPM_REGISTRY:+--registry="$SANDBOX_NPM_REGISTRY"}
  done
}

# crate_licenses <output> [--append]: checks the licenses of the Rust crates
# and writes their texts to <output>.
crate_licenses() {
  as_tester "$WORK" python3 shared/tools/cargo_licenses.py --offline \
    --manifest-path tauri/src-tauri/Cargo.toml --output "$@"
}

# prepare [frontend build script]: installs the packages from the caches,
# generates the assets and icons and builds the front end, which Tauri embeds.
# The release front end also shows the licenses of the Rust crates, as the
# beforeBuildCommand of tauri.conf.json adds them.
prepare() {
  copy_repository web tauri
  npm_install --offline
  as_tester "$WEB" npm run assets
  as_tester "$WEB" npm run "${1:-build:tauri}"
  if [[ "${1:-build:tauri}" == build:tauri ]]; then
    crate_licenses "$WEB/dist/tauri/licenses.txt" --append
  fi
  as_tester "$APP" npm run icons > /dev/null 2>&1
}

# build [frontend directory]: the release binary, with the front end of
# web/dist/<directory>.
build() {
  local frontend="${1:-tauri}"
  as_tester "$APP" env CARGO_NET_OFFLINE=true npx tauri build --no-bundle \
    --config "{\"build\": {\"frontendDist\": \"../../web/dist/$frontend\", \"beforeBuildCommand\": \"\"}}"
}

case "${1:-}" in
  deps)
    copy_repository web tauri
    npm_install
    as_tester "$CRATE" cargo fetch --locked
    ;;

  check)
    prepare
    crate_licenses /tmp/crate-licenses.txt
    as_tester "$CRATE" cargo fmt --check
    as_tester "$CRATE" cargo clippy --offline --locked --all-targets -- -D warnings
    as_tester "$CRATE" cargo test --offline --locked
    ;;

  e2e)
    prepare build:e2e:tauri
    build tauri-e2e
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash web/e2e/run.sh fake "$BINARY"
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox bash web/e2e/run.sh real "$BINARY"
    stop_display
    ;;

  measure)
    prepare build:e2e:tauri
    build tauri-e2e
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash web/e2e/run.sh measure "$BINARY" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  screenshots)
    prepare build:e2e:tauri
    build tauri-e2e
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash shared/tools/screenshots.sh /home/tester/screenshots "$BINARY"
    publish /home/tester/screenshots/*.png
    stop_display
    ;;

  package)
    prepare
    build
    smoke_test "$BINARY" runuser -u tester -- env HOME=/home/tester
    install -d -o tester -g tester /home/tester/stage /home/tester/dist
    install -o tester -m 755 "$BINARY" /home/tester/stage/vfox-plus
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" \
      bash "$WORK/shared/packaging/linux/build-deb.sh" \
      --package vfox-plus-tauri --version "$(jq -r .version "$CRATE/tauri.conf.json")" \
      --stage /home/tester/stage --executable vfox-plus --technology Tauri \
      --depends "$DEPENDS" --notice "$WEB/dist/tauri/licenses.txt" /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    install_deb_test vfox-plus-tauri /usr/lib/vfox-plus/vfox-plus
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use tauri/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
