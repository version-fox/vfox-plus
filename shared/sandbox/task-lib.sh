# shellcheck shell=bash
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

# Container side of the Docker sandbox: sourced by
# mygo/sandbox/run-task.sh inside a sandbox container. See
# README.md next to this file.
#
# The repository is mounted read-only at /src and the results directory at
# /out. A task works on a private copy of the repository as the unprivileged
# user tester, so the only thing that reaches the host is what it copies to
# /out.

# shellcheck disable=SC2034 # The variables are used by the sourcing scripts.

SRC=/src
OUT=/out
WORK=/home/tester/work
SCREEN=(-screen 0 1280x800x24)

# as_tester <directory> <command...>: runs a command as tester in a directory.
as_tester() {
  local directory="$1"
  shift
  (cd "$directory" && runuser -u tester -- env HOME=/home/tester "$@")
}

# What the host may have built or downloaded in the copied directories, which a
# task must neither see nor spend time on copying. A run-task.sh may add
# patterns before it calls copy_repository.
COPY_EXCLUDES=(
  --exclude-vcs --exclude=.idea --exclude='*.iml'
  # Bytecode of the tests of .github/scripts.
  --exclude=__pycache__
)

# copy_repository <directory...>: copies the shared resources, the license and
# the given top-level directories of the repository to $WORK, without what the
# host has built in them.
copy_repository() {
  install -d -o tester -g tester "$WORK"
  local directories=(./shared ./LICENSE) directory
  for directory in "$@"; do
    directories+=("./$directory")
  done
  tar -C "$SRC" --create "${COPY_EXCLUDES[@]}" "${directories[@]}" | runuser -u tester -- tar -C "$WORK" --extract
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

# icon_font <output> [icon_font.py options...]: cuts the icon font out of the
# Material Symbols font in the image.
icon_font() {
  local output="$1"
  shift
  runuser -u tester -- python3 "$WORK/shared/tools/icon_font.py" subset --fonts "$VFOX_PLUS_SYMBOLS" --output "$output" "$@"
}

# start_display: starts a virtual display and a session bus, and exports
# DISPLAY and DBUS_SESSION_BUS_ADDRESS for the applications started afterwards.
start_display() {
  Xvfb :42 -nolisten tcp "${SCREEN[@]}" > /dev/null 2>&1 &
  XVFB_PID=$!
  export DISPLAY=:42
  sleep 1
  eval "$(runuser -u tester -- dbus-launch --sh-syntax)"
  export DBUS_SESSION_BUS_ADDRESS
}

stop_display() {
  kill "${XVFB_PID:-}" 2> /dev/null || true
}

# smoke_test <executable> [command prefix...]: starts the application on a
# virtual display and checks that it still runs 15 seconds later.
smoke_test() {
  local executable="$1" app
  shift
  start_display
  "$@" env DISPLAY="$DISPLAY" "$executable" > /tmp/vfox-plus.log 2>&1 &
  app=$!
  sleep 15
  if ! kill -0 "$app" 2> /dev/null; then
    cat /tmp/vfox-plus.log >&2
    echo "vfox+ exited during startup" >&2
    return 1
  fi
  kill "$app"
  stop_display
  echo "vfox+ started: $executable"
}

# install_deb_test <package> <executables...>: runs in a clean Ubuntu container
# with the .deb mounted at /pkg: the declared dependencies must be all the
# application needs. Checks the shared libraries of the executables, which may
# be patterns that are expanded after the installation, the desktop integration,
# starts the application and removes the package again.
install_deb_test() {
  local package="$1"
  shift
  if [[ -n "${UBUNTU_MIRROR:-}" ]]; then
    sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
      /etc/apt/sources.list.d/ubuntu.sources
  fi
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y -qq --no-install-recommends /pkg/"$package"_*.deb > /dev/null
  local files=() pattern
  for pattern in "$@"; do
    # shellcheck disable=SC2206 # The patterns are meant to be expanded, now that the package is installed.
    files+=($pattern)
  done
  if ldd "${files[@]}" | grep 'not found'; then
    echo "The dependencies of the package miss the libraries above." >&2
    exit 1
  fi
  test -x /usr/bin/vfox-plus
  test -f /usr/share/applications/dev.vfox.VfoxPlus.desktop
  test -f /usr/share/icons/hicolor/scalable/apps/dev.vfox.VfoxPlus.svg
  test -f /usr/share/metainfo/dev.vfox.VfoxPlus.metainfo.xml
  apt-get install -y -qq --no-install-recommends xvfb xauth dbus-x11 > /dev/null
  useradd --create-home tester 2> /dev/null || true
  smoke_test /usr/bin/vfox-plus runuser -u tester -- env HOME=/home/tester
  apt-get remove -y -qq "$package" > /dev/null
  test ! -e /usr/bin/vfox-plus
  echo "The package installs, starts and uninstalls cleanly."
}
