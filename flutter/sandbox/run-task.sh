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

# Runs one task inside a sandbox container. sandbox.sh starts the containers;
# do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The repository is mounted read-only at /src and the results directory, if
# the task has results, at /out. A task works on a private copy of the
# repository as the unprivileged user tester, so the only thing that reaches
# the host is what it copies to /out. The steps mirror the jobs of
# .github/workflows/flutter.yml; keep them in sync.
set -euo pipefail

SRC=/src
OUT=/out
WORK=/home/tester/work
APP="$WORK/flutter"
SCREEN=(-screen 0 1280x800x24)

# as_tester <command...>: runs a command as tester in the Flutter project.
as_tester() {
  (cd "$APP" && runuser -u tester -- env HOME=/home/tester "$@")
}

# Copies the Flutter implementation, the shared resources and the license to
# $WORK, without version control and what the host has built. The other
# implementations, and whatever they have built, are left out.
copy_repository() {
  install -d -o tester -g tester "$WORK"
  tar -C "$SRC" --create --exclude-vcs \
    --exclude=.idea --exclude='*.iml' --exclude=node_modules --exclude=__pycache__ \
    --exclude=./flutter/build --exclude=./flutter/coverage --exclude=./flutter/.dart_tool \
    --exclude=./flutter/.flutter-plugins-dependencies --exclude=./flutter/linux/flutter/ephemeral \
    --exclude=./flutter/assets/fonts/MaterialSymbolsRounded.ttf \
    ./flutter ./shared ./LICENSE | runuser -u tester -- tar -C "$WORK" --extract
}

# get_packages [--offline]: gets the packages of pubspec.lock. The deps task
# downloads them into the pub cache volume; the other tasks run without network
# and take them from there.
get_packages() {
  as_tester flutter pub get --enforce-lockfile "$@"
}

# Generates the icon font, a build output that is not committed.
generate_icon_font() {
  as_tester dart run tool/generate_icon_font.dart
}

# publish <path...>: copies results to /out for the user on the host.
publish() {
  cp -r "$@" "$OUT/"
  # Only set with a rootful Docker daemon; under rootless Docker, root in the
  # container already is the user on the host.
  if [[ -n "${OUT_OWNER:-}" ]]; then
    chown -R "$OUT_OWNER" "$OUT"
  fi
}

# smoke_test <executable> [command prefix...]: starts the application on a
# virtual display and checks that it still runs 15 seconds later.
smoke_test() {
  local executable="$1" xvfb app
  shift
  Xvfb :42 -nolisten tcp "${SCREEN[@]}" > /dev/null 2>&1 &
  xvfb=$!
  sleep 1
  "$@" env DISPLAY=:42 "$executable" > /tmp/vfox-plus.log 2>&1 &
  app=$!
  sleep 15
  if ! kill -0 "$app" 2> /dev/null; then
    cat /tmp/vfox-plus.log >&2
    echo "vfox+ exited during startup" >&2
    return 1
  fi
  kill "$app" "$xvfb"
  echo "vfox+ started: $executable"
}

case "${1:-}" in
  deps)
    copy_repository
    get_packages
    ;;

  check)
    copy_repository
    get_packages --offline
    cp -r "$APP/lib/src/l10n" /tmp/l10n
    cp -r "$APP/lib/src/ui/icons" /tmp/icons
    generate_icon_font
    as_tester flutter gen-l10n
    if ! diff -ru /tmp/l10n "$APP/lib/src/l10n" || ! diff -ru /tmp/icons "$APP/lib/src/ui/icons"; then
      echo "Regenerate the translations and the icons, and commit the result." >&2
      exit 1
    fi
    as_tester dart format --output=none --set-exit-if-changed lib test tool integration_test
    as_tester flutter analyze --no-pub --fatal-infos --fatal-warnings
    as_tester flutter test --no-pub --coverage
    as_tester dart run tool/coverage.dart --min=85
    publish "$APP/coverage/lcov.info"
    ;;

  e2e)
    copy_repository
    get_packages --offline
    generate_icon_font
    # flutter test starts a desktop application only once per run, so every
    # file runs on its own.
    for test in "$APP"/integration_test/*_test.dart; do
      as_tester env VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox \
        xvfb-run -a -s "${SCREEN[*]}" flutter test --no-pub "integration_test/${test##*/}" -d linux
    done
    ;;

  screenshots)
    copy_repository
    get_packages --offline
    generate_icon_font
    # The screenshots show vfox where its installer puts it: in this throw-away
    # container, the fake vfox of the test replaces the real one.
    install -o tester -m 755 /dev/null /usr/local/bin/vfox
    as_tester env VFOX_PLUS_SCREENSHOTS_DIR=/home/tester/screenshots VFOX_PLUS_SCREENSHOTS_VFOX=/usr/local/bin/vfox \
      xvfb-run -a -s "${SCREEN[*]}" flutter test --no-pub integration_test/screenshots_test.dart -d linux
    publish /home/tester/screenshots/*.png
    ;;

  measure)
    copy_repository
    get_packages --offline
    generate_icon_font
    as_tester flutter build linux --no-pub --release
    install -d -o tester -g tester /home/tester/measure
    # The application has no scenario of its own for shared/tools/measure.sh:
    # tool/measure.py drives the release build with the mouse, on the same
    # kind of virtual display and with a session bus, as the sandboxes of the
    # other implementations. The results are published even if it fails, as
    # the screenshots show where it stopped.
    status=0
    as_tester xvfb-run -a -s "${SCREEN[*]}" dbus-run-session -- \
      python3 tool/measure.py --output /home/tester/measure build/linux/x64/release/bundle/vfox-plus || status=$?
    publish /home/tester/measure/*
    exit "$status"
    ;;

  package)
    copy_repository
    get_packages --offline
    generate_icon_font
    as_tester flutter build linux --no-pub --release
    smoke_test "$APP/build/linux/x64/release/bundle/vfox-plus" runuser -u tester -- env HOME=/home/tester
    as_tester env SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date +%s)}" packaging/linux/build_deb.sh /home/tester/dist
    desktop-file-validate "$WORK/shared/packaging/linux/dev.vfox.VfoxPlus.desktop"
    as_tester lintian --fail-on error --suppress-tags malformed-contact,bogus-mail-host-in-debian-changelog \
      /home/tester/dist/*.deb
    publish /home/tester/dist/*
    ;;

  install-test)
    # Runs in a clean Ubuntu container, with the .deb mounted at /pkg: the
    # declared dependencies must be all the application needs.
    if [[ -n "${UBUNTU_MIRROR:-}" ]]; then
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources
    fi
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends /pkg/vfox-plus_*.deb > /dev/null
    if ldd /usr/lib/vfox-plus/vfox-plus /usr/lib/vfox-plus/lib/*.so | grep 'not found'; then
      echo "The dependencies of the package miss the libraries above." >&2
      exit 1
    fi
    test -x /usr/bin/vfox-plus
    test -f /usr/share/applications/dev.vfox.VfoxPlus.desktop
    test -f /usr/share/icons/hicolor/scalable/apps/dev.vfox.VfoxPlus.svg
    apt-get install -y -qq --no-install-recommends xvfb > /dev/null
    smoke_test /usr/bin/vfox-plus
    apt-get remove -y -qq vfox-plus > /dev/null
    test ! -e /usr/lib/vfox-plus
    echo "The package installs, starts and uninstalls cleanly."
    ;;

  fixtures)
    copy_repository
    runuser -u tester -- env HOME=/home/tester bash "$WORK/shared/fixtures/capture.sh" /usr/local/bin/vfox
    publish "$WORK"/shared/fixtures/vfox-*
    ;;

  shell)
    copy_repository
    get_packages --offline
    generate_icon_font
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use flutter/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
