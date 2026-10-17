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

# Runs one task inside a sandbox container of the Dioxus implementation.
# sandbox.sh starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/dioxus.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
APP="$WORK/dioxus"
BINARY="$APP/target/release/vfox-plus"
# The assets that the build generates.
COPY_EXCLUDES+=(
  --exclude=./dioxus/assets/components.js --exclude=./dioxus/assets/e2e-driver.js
  --exclude=./dioxus/assets/MaterialSymbolsRounded.woff2 --exclude=./dioxus/assets/npm-licenses.txt
  --exclude=./dioxus/assets/crate-licenses.txt
)
# Runtime dependencies of the release binary on Ubuntu 26.04; the installation
# test checks that they are complete.
DEPENDS="libwebkit2gtk-4.1-0, libjavascriptcoregtk-4.1-0, libgtk-3-0t64 | libgtk-3-0, libgdk-pixbuf-2.0-0, libcairo2, libsoup-3.0-0, libglib2.0-0t64 | libglib2.0-0, libssl3t64 | libssl3, libxdo3, libc6, libgcc-s1"
# The coverage gate: the lines of core and state, which the unit tests cover;
# the end-to-end tests cover the user interface.
COVERAGE=(--fail-under-lines 85 --ignore-filename-regex '/src/(ui/|main\.rs)|/(testing|tests|cli_tests|contract_tests)\.rs')

# prepare: the web components, the icon font and the licenses, from the caches.
# The end-to-end driver comes from web/src/e2e.
prepare() {
  copy_repository web dioxus
  as_tester "$APP/components" npm ci --no-audit --no-fund --offline
  as_tester "$APP/components" npm run build
  as_tester "$APP" python3 "$WORK/shared/tools/cargo_licenses.py" --offline \
    --manifest-path Cargo.toml --output assets/crate-licenses.txt
}

# build [cargo arguments...]: the release binary.
build() {
  as_tester "$APP" cargo build --release --locked --offline "$@"
}

case "${1:-}" in
  deps)
    copy_repository web dioxus
    as_tester "$APP/components" npm ci --no-audit --no-fund ${SANDBOX_NPM_REGISTRY:+--registry="$SANDBOX_NPM_REGISTRY"}
    as_tester "$APP" cargo fetch --locked
    ;;

  check)
    prepare
    as_tester "$APP/components" npm run format:check
    as_tester "$APP" cargo fmt --check
    as_tester "$APP" cargo clippy --offline --locked --all-targets --all-features -- -D warnings
    as_tester "$APP" cargo llvm-cov --offline --locked --all-features "${COVERAGE[@]}"
    ;;

  e2e)
    prepare
    build --features e2e
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash web/e2e/run.sh fake "$BINARY"
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox bash web/e2e/run.sh real "$BINARY"
    stop_display
    ;;

  measure)
    prepare
    build --features e2e
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash web/e2e/run.sh measure "$BINARY" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  screenshots)
    prepare
    build --features e2e
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash dioxus/tools/screenshots.sh "$BINARY" /home/tester/screenshots
    publish /home/tester/screenshots/*.png
    stop_display
    ;;

  package)
    prepare
    build
    smoke_test "$BINARY" runuser -u tester -- env HOME=/home/tester
    install -d -o tester -g tester /home/tester/stage /home/tester/dist
    install -o tester -m 755 "$BINARY" /home/tester/stage/vfox-plus
    # The licenses of the bundled npm packages, the font and the Rust crates.
    cat "$APP/assets/npm-licenses.txt" "$APP/assets/crate-licenses.txt" > /home/tester/THIRD-PARTY-LICENSES
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" \
      bash "$WORK/shared/packaging/linux/build-deb.sh" \
      --package vfox-plus-dioxus --version "$(sed -n 's/^version = "\(.*\)"$/\1/p' "$APP/Cargo.toml" | head -n 1)" \
      --stage /home/tester/stage --executable vfox-plus --technology Dioxus \
      --depends "$DEPENDS" --notice /home/tester/THIRD-PARTY-LICENSES /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    install_deb_test vfox-plus-dioxus /usr/lib/vfox-plus/vfox-plus
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use dioxus/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
