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

# Runs one task inside a sandbox container of the Uno implementation.
# sandbox.sh starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/uno.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
APP="$WORK/uno"
STAGE=/home/tester/stage
EXECUTABLE=VfoxPlus
# What the build generates.
COPY_EXCLUDES+=(
  --exclude='./uno/src/*/bin' --exclude='./uno/src/*/obj' --exclude='./uno/tests/*/bin' --exclude='./uno/tests/*/obj'
  --exclude=./uno/TestResults --exclude=./uno/src/VfoxPlus.App/Assets/Fonts --exclude=./uno/src/VfoxPlus.App/Assets/Generated
)
# Runtime dependencies of the release build on Ubuntu 26.04; the installation
# test checks that they are complete. Uno loads Skia, which draws with the
# processor, through X11 (it runs through XWayland on a Wayland desktop), and ICU
# at startup. Fonts: Ubuntu Sans, or any other sans-serif font of the desktop.
DEPENDS="libc6, libstdc++6, libgcc-s1, libx11-6, libxext6, libxfixes3, libxi6, libxrandr2, libfontconfig1, libfreetype6, libexpat1, zlib1g, libbz2-1.0, libpng16-16t64, libbrotli1, libicu78, libssl3t64, fontconfig, fonts-ubuntu | fonts-noto-core"
# Chinese text needs a CJK font, files are chosen with the dialogs of the
# desktop portal, zenity or kdialog, and links open in the browser.
RECOMMENDS="fonts-noto-cjk, xdg-desktop-portal | zenity | kdialog, xdg-utils"
# A self-contained .NET application bundles its runtime, and SkiaSharp its native
# libraries; lintian takes that for the embedded copies of libraries that policy
# forbids. The copies are those of the NuGet packages, whose licenses the package
# lists.
OVERRIDES=(
  "embedded-library freetype [usr/lib/vfox-plus/libSkiaSharp.so]"
  "embedded-library libjpeg [usr/lib/vfox-plus/libSkiaSharp.so]"
  "embedded-library libpng [usr/lib/vfox-plus/libSkiaSharp.so]"
  "embedded-library zlib [usr/lib/vfox-plus/libSystem.IO.Compression.Native.so]"
)
override_args=()
for override in "${OVERRIDES[@]}"; do
  override_args+=(--override "$override")
done

# prepare: the generated inputs. web/ comes along for its end-to-end runner,
# which every implementation shares.
prepare() {
  copy_repository uno web
  as_tester "$APP" bash tools/prepare.sh
}

# restore_debug, restore_release: the packages of the solution, or of the
# published application, from the NuGet cache that the deps task filled. The
# configurations restore different packages (the Uno development server is only
# there in Debug, and a release build refuses it), and the last restore decides
# what the build sees. The audit of the packages asks nuget.org, which the tasks
# cannot reach.
restore_debug() {
  as_tester "$APP" dotnet restore VfoxPlus.slnx -p:NuGetAudit=false
}

restore_release() {
  as_tester "$APP" dotnet restore src/VfoxPlus.App --runtime linux-x64 -p:Configuration=Release -p:NuGetAudit=false
}

# in_display <command...>: runs a command as tester on the virtual display.
in_display() {
  as_tester "$WORK" env DISPLAY="$DISPLAY" DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDRESS" "$@"
}

# publish_app <e2e|release> <directory>: publishes the application as tester.
publish_app() {
  install -d -o tester -g tester "$(dirname "$2")"
  if [[ "$1" == e2e ]]; then
    as_tester "$APP" bash tools/publish.sh --e2e "$2"
  else
    as_tester "$APP" bash tools/publish.sh "$2"
  fi
}

case "${1:-}" in
  deps)
    copy_repository uno web
    as_tester "$APP" dotnet restore VfoxPlus.slnx
    as_tester "$APP" dotnet restore src/VfoxPlus.App --runtime linux-x64 -p:Configuration=Release
    ;;

  check)
    prepare
    restore_debug
    as_tester "$APP" dotnet format VfoxPlus.slnx --verify-no-changes --no-restore
    as_tester "$APP" dotnet build VfoxPlus.slnx --no-restore --nologo
    # The end-to-end agent is code too.
    as_tester "$APP" dotnet build src/VfoxPlus.App --no-restore --nologo -p:E2E=true
    as_tester "$APP" dotnet test tests/VfoxPlus.Tests --no-restore --nologo \
      --collect:"XPlat Code Coverage" --results-directory TestResults --settings tests/coverlet.runsettings
    as_tester "$APP" bash -c 'python3 tools/coverage.py TestResults/*/coverage.cobertura.xml 90 VfoxPlus.Core VfoxPlus.State VfoxPlus.Design VfoxPlus.Platform'
    # The shell and Python scripts of the implementation.
    as_tester "$APP" bash -n tools/prepare.sh tools/publish.sh
    as_tester "$APP" python3 -m py_compile tools/coverage.py
    ;;

  build)
    prepare
    restore_release
    publish_app release "$STAGE"
    ;;

  e2e)
    prepare
    restore_release
    publish_app e2e "$STAGE"
    start_display
    in_display env VFOX_PLUS_E2E_TIMEOUT=300 bash web/e2e/run.sh fake "$STAGE/$EXECUTABLE"
    in_display env VFOX_PLUS_E2E_TIMEOUT=300 VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox bash web/e2e/run.sh real "$STAGE/$EXECUTABLE"
    stop_display
    ;;

  measure)
    prepare
    restore_release
    publish_app e2e "$STAGE"
    start_display
    in_display bash shared/tools/measure.sh "$STAGE/$EXECUTABLE" | tee /tmp/measure.log
    grep '^memory' /tmp/measure.log > /tmp/memory.txt
    publish /tmp/memory.txt
    stop_display
    ;;

  screenshots)
    prepare
    restore_release
    publish_app e2e "$STAGE"
    start_display
    in_display bash shared/tools/screenshots.sh /home/tester/screenshots "$STAGE/$EXECUTABLE"
    publish /home/tester/screenshots/*.png
    stop_display
    ;;

  package)
    prepare
    restore_release
    publish_app release "$STAGE"
    install -d -o tester -g tester /home/tester/dist
    as_tester "$APP" env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" \
      bash "$WORK/shared/packaging/linux/build-deb.sh" \
      --package vfox-plus-uno --version "$(tr -d '[:space:]' < "$APP/VERSION")" \
      --stage "$STAGE" --executable "$EXECUTABLE" --technology "Uno Platform" --strip \
      --depends "$DEPENDS" --recommends "$RECOMMENDS" "${override_args[@]}" \
      --notice "$STAGE/Assets/Generated/licenses.txt" /home/tester/dist
    # The packaged files, not the stage, are what runs.
    dpkg-deb --extract /home/tester/dist/*.deb /home/tester/installed
    smoke_test /home/tester/installed/usr/lib/vfox-plus/"$EXECUTABLE" runuser -u tester -- env HOME=/home/tester
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester "$APP" lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg.
    # The libraries are a pattern that the function expands once the package is installed.
    install_deb_test vfox-plus-uno /usr/lib/vfox-plus/"$EXECUTABLE" '/usr/lib/vfox-plus/*.so'
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use uno/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
