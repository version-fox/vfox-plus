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

# Runs one task inside a sandbox container of the Compose Multiplatform
# implementation. sandbox.sh starts the containers; do not run this script on
# the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/compose.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
APP="$WORK/compose"
IMAGE="$APP/build/compose/binaries/main/app/vfox-plus"
COPY_EXCLUDES+=(--exclude=./compose/build --exclude=./compose/.kotlin)
# Runtime dependencies of the application image on Ubuntu 26.04: the bundled
# Java runtime, AWT and Skia. libfontconfig1 brings a font. The installation
# test checks that they are complete.
DEPENDS="libc6, libstdc++6, libgcc-s1, zlib1g, libx11-6, libxext6, libxi6, libxrender1, libxtst6, libfontconfig1, libgl1"
# jpackage keeps the launcher, the JARs and the Java runtime together in
# /usr/lib/vfox-plus, and the runtime and Skia bundle their own copies of
# zlib, FreeType, libjpeg, Little CMS and Expat, which cannot be replaced by
# the system libraries.
OVERRIDES=(--override embedded-library --override jar-not-in-usr-share --override codeless-jar)

# gradle <arguments...>: Gradle without network; only the deps task downloads.
gradle_offline() {
  as_tester "$APP" gradle --offline --console=plain "$@"
}

case "${1:-}" in
  deps)
    copy_repository compose
    # Everything the other tasks run, so that Gradle downloads every
    # dependency, plugin and tool (ktlint, the Kover agent) into its cache.
    as_tester "$APP" gradle --console=plain spotlessCheck checkLicense test koverVerify createDistributable
    ;;

  check)
    copy_repository compose
    gradle_offline spotlessCheck checkLicense test koverVerify
    ;;

  e2e)
    copy_repository compose
    # Against the fixtures, and against the vfox of the image with the offline test plugin.
    as_tester "$APP" env VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox gradle --offline --console=plain e2eTest
    ;;

  screenshots)
    copy_repository compose
    gradle_offline screenshots
    publish "$APP"/build/screenshots/*.png
    ;;

  measure)
    copy_repository compose
    gradle_offline createDistributable
    start_display
    as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" \
      bash shared/tools/measure.sh "$IMAGE/bin/vfox-plus" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  package)
    copy_repository compose
    gradle_offline createDistributable
    smoke_test "$IMAGE/bin/vfox-plus" runuser -u tester -- env HOME=/home/tester
    install -d -o tester -g tester /home/tester/dist
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" \
      bash "$WORK/shared/packaging/linux/build-deb.sh" \
      --package vfox-plus-compose --version "$(sed -n 's/^version = "\(.*\)"$/\1/p' "$APP/build.gradle.kts")" \
      --stage "$IMAGE" --executable bin/vfox-plus --technology "Compose Multiplatform" --strip "${OVERRIDES[@]}" \
      --depends "$DEPENDS" --notice "$APP/build/reports/dependency-license/THIRD-PARTY-NOTICES.txt" /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    install_deb_test vfox-plus-compose /usr/lib/vfox-plus/bin/vfox-plus /usr/lib/vfox-plus/lib/app/libskiko-linux-x64.so
    ;;

  shell)
    copy_repository compose
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use compose/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
