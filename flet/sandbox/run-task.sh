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

# Runs one task inside a sandbox container of the Flet implementation.
# sandbox.sh starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/flet.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
APP="$WORK/flet"
M3E="$APP/m3e/src/flutter/vfox_plus_m3e"
# What the build generates.
COPY_EXCLUDES+=(
  --exclude=./flet/.venv --exclude=./flet/build --exclude=./flet/dist
  --exclude=./flet/coverage.xml --exclude=./flet/htmlcov
  --exclude=./flet/src/vfox_plus/resources --exclude=./flet/src/assets
  --exclude=./flet/m3e/src/flutter/vfox_plus_m3e/fonts
  --exclude=./flet/m3e/src/flutter/vfox_plus_m3e/.dart_tool
)
# Runtime dependencies of the release bundle on Ubuntu 26.04; the installation
# test checks that they are complete.
DEPENDS="libgtk-3-0t64 | libgtk-3-0, libglib2.0-0t64 | libglib2.0-0, libglib2.0-bin"
DEPENDS+=", libepoxy0, libegl1, libgles2, libfontconfig1, libblkid1, liblzma5"
DEPENDS+=", libssl3t64 | libssl3, libexpat1, libffi8, libsqlite3-0, libstdc++6, libc6, zlib1g"
# The coverage gate: the lines of core and state, which the unit tests cover;
# the end-to-end scenarios and the interface tests cover the user interface.
COVERAGE=(tests/core tests/state)

# build_app: `flet build` with retries. The network to the package
# registries drops connections from time to time, which neither curl nor pip
# retries on its own. PIP_NO_BUILD_ISOLATION uses the setuptools and wheel of
# the environment (see pyproject.toml) instead of downloading them again.
build_app() {
  local attempt
  for attempt in 1 2 3; do
    # PIP_NO_BUILD_ISOLATION goes through env: runuser does not pass the
    # environment of the caller through.
    if as_tester "$APP" env PIP_NO_BUILD_ISOLATION=1 uv run flet build linux --yes; then
      return 0
    fi
    echo "flet build failed, retrying ($attempt)" >&2
    sleep "$attempt"
  done
  return 1
}

# prepare: the packages from the caches, then the generated inputs. web/ comes
# along for its end-to-end runner, which the Flet scenarios follow.
prepare() {
  copy_repository flet web
  # uv caches what it found under the index it asked, so the mirror of the deps
  # task has to be named again to find the build requirements of m3e.
  as_tester "$APP" uv sync --frozen --offline ${SANDBOX_UV_MIRROR:+--index-url "$SANDBOX_UV_MIRROR"}
  as_tester "$APP" python3 tools/patch_build_template.py
  as_tester "$APP" python3 tools/prepare.py
  as_tester "$M3E" dart pub get --enforce-lockfile --offline
}

# in_display <command...>: runs a command as tester on the virtual display.
in_display() {
  as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" "$@"
}

case "${1:-}" in
  deps)
    copy_repository flet web
    as_tester "$APP" uv sync --frozen ${SANDBOX_UV_MIRROR:+--index-url "$SANDBOX_UV_MIRROR"}
    as_tester "$APP" python3 tools/patch_build_template.py
    as_tester "$M3E" dart pub get --enforce-lockfile
    # Warms every cache a build needs (the Python manifest, the embedded
    # Python, the packages of the Flutter app), so that the rebuilds of the
    # other tasks stay fast.
    as_tester "$APP" python3 tools/prepare.py
    build_app
    ;;

  check)
    prepare
    as_tester "$APP" uv run ruff format --check .
    as_tester "$APP" uv run ruff check .
    as_tester "$APP" uv run mypy
    as_tester "$APP" uv run coverage run -m pytest "${COVERAGE[@]}" -q
    as_tester "$APP" uv run coverage report --fail-under=85 \
      --include='*/vfox_plus/core/*,*/vfox_plus/state/*'
    as_tester "$M3E" dart format --set-exit-if-changed lib
    as_tester "$M3E" flutter analyze --no-pub
    ;;

  e2e)
    prepare
    build_app
    start_display
    in_display env VFOX_PLUS_E2E_TIMEOUT=300 bash web/e2e/run.sh fake "$APP/build/linux/vfox-plus"
    in_display env VFOX_PLUS_E2E_TIMEOUT=300 VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox \
      bash web/e2e/run.sh real "$APP/build/linux/vfox-plus"
    stop_display
    ;;

  measure)
    prepare
    build_app
    start_display
    in_display bash shared/tools/measure.sh "$APP/build/linux/vfox-plus" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  screenshots)
    prepare
    build_app
    start_display
    in_display bash shared/tools/screenshots.sh /home/tester/screenshots "$APP/build/linux/vfox-plus"
    publish /home/tester/screenshots/*.png
    stop_display
    ;;

  package)
    prepare
    build_app
    smoke_test "$APP/build/linux/vfox-plus" runuser -u tester -- env HOME=/home/tester
    install -d -o tester -g tester /home/tester/dist
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" bash tools/package.sh /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    install_deb_test vfox-plus-flet /usr/lib/vfox-plus/vfox-plus
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use flet/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
