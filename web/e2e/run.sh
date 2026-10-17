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

# Runs an end-to-end scenario of the web front end (src/e2e/scenarios.ts) in an
# application built with `VFOX_PLUS_E2E=1`, such as the e2e build of the Tauri
# or the Wails implementation.
#
# Usage: web/e2e/run.sh <scenario> <application command...>
#
#   fake     against e2e/fake-vfox.sh, which replays shared/fixtures
#   real     against the real vfox in VFOX_PLUS_E2E_VFOX, with the offline
#            plugin of shared/test-plugin and e2e/test-server.mjs (needs node)
#   measure  against the fake vfox; prints the memory use of the application
#            on the Home page and with the versions of Node.js listed
#
# The application runs on the current display with a new, empty HOME, so it
# never sees the vfox installation and the settings of whoever runs the test.
# Exits with the exit code of the application.
set -euo pipefail

E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$E2E_DIR/../.." && pwd)"
SCENARIO="${1:?Usage: run.sh <scenario> <application command...>}"
shift
[[ $# -gt 0 ]] || { echo "Usage: run.sh <scenario> <application command...>" >&2; exit 2; }

SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/vfox-plus-e2e.XXXXXX")"
HOME_DIR="$SANDBOX/home"
mkdir -p "$HOME_DIR/.config/vfox-plus"
SERVER_PID=""
APP_PID=""
# shellcheck disable=SC2329 # Invoked by the trap below.
cleanup() {
  [[ -z "$APP_PID" ]] || kill "$APP_PID" 2> /dev/null || true
  [[ -z "$SERVER_PID" ]] || kill "$SERVER_PID" 2> /dev/null || true
  rm -rf "$SANDBOX"
}
trap cleanup EXIT

environment=(HOME="$HOME_DIR" VFOX_PLUS_E2E_SCENARIO="$SCENARIO" LANG=C.UTF-8)
case "$SCENARIO" in
  fake | measure)
    vfox="$SANDBOX/vfox"
    install -m 755 "$E2E_DIR/fake-vfox.sh" "$vfox"
    # Simulate a GUI started from a terminal in which vfox is activated.
    environment+=(VFOX_PLUS_FIXTURES="$REPO/shared/fixtures/vfox-1.0.13" __VFOX_PID=1 __VFOX_SHELL=bash)
    ;;
  real)
    vfox="${VFOX_PLUS_E2E_VFOX:?Set VFOX_PLUS_E2E_VFOX to the vfox executable.}"
    mkdir -p "$SANDBOX/server"
    node "$E2E_DIR/test-server.mjs" "$SANDBOX/server" > "$SANDBOX/server.log" 2>&1 &
    SERVER_PID=$!
    for _ in $(seq 50); do
      [[ -s "$SANDBOX/server.log" ]] && break
      sleep 0.1
    done
    server="$(head -n 1 "$SANDBOX/server.log")"
    [[ "$server" == http://* ]] || { cat "$SANDBOX/server.log" >&2; exit 1; }
    cp "$REPO/shared/test-plugin/testsdk.lua" "$SANDBOX/my-plugin.lua"
    environment+=(VFOX_PLUS_TEST_SERVER="$server" VFOX_PLUS_E2E_LOCAL_PLUGIN="$SANDBOX/my-plugin.lua")
    ;;
  *)
    echo "Unknown scenario: $SCENARIO" >&2
    exit 2
    ;;
esac
environment+=(VFOX_PLUS_E2E_VFOX="$vfox")
printf '{\n  "language": "english",\n  "vfoxExecutable": "%s"\n}\n' "$vfox" > "$HOME_DIR/.config/vfox-plus/settings.json"

# memory_of <pid>: the resident and proportional set size of a process and its
# descendants.
memory_of() {
  local pids=("$1") i=0 child pid rss=0 pss=0
  while [[ $i -lt ${#pids[@]} ]]; do
    # shellcheck disable=SC2013 # The files list process IDs separated by spaces.
    for child in $(cat "/proc/${pids[$i]}/task/"*/children 2> /dev/null); do
      pids+=("$child")
    done
    i=$((i + 1))
  done
  for pid in "${pids[@]}"; do
    if [[ -r "/proc/$pid/smaps_rollup" ]]; then
      rss=$((rss + $(awk '/^Rss:/ {print $2}' "/proc/$pid/smaps_rollup")))
      pss=$((pss + $(awk '/^Pss:/ {print $2}' "/proc/$pid/smaps_rollup")))
    fi
  done
  echo "RSS $((rss / 1024)) MB, PSS $((pss / 1024)) MB in ${#pids[@]} processes"
}

# VFOX_HOME would make vfox use a shared vfox directory instead of HOME.
env -u VFOX_HOME -u XDG_CONFIG_HOME "${environment[@]}" "$@" > "$SANDBOX/app.log" 2>&1 &
APP_PID=$!
printed=0
deadline=$((SECONDS + ${VFOX_PLUS_E2E_TIMEOUT:-300}))
while kill -0 "$APP_PID" 2> /dev/null; do
  lines="$(wc -l < "$SANDBOX/app.log")"
  if [[ $lines -gt $printed ]]; then
    tail -n "+$((printed + 1))" "$SANDBOX/app.log" | head -n "$((lines - printed))" | while IFS= read -r line; do
      echo "$line"
      if [[ "$line" == *"e2e: MEASURE "* ]]; then
        echo "memory (${line##*MEASURE }): $(memory_of "$APP_PID")"
      fi
    done
    printed=$lines
  fi
  if [[ $SECONDS -gt $deadline ]]; then
    echo "The scenario did not finish in time." >&2
    exit 1
  fi
  sleep 0.2
done
tail -n "+$((printed + 1))" "$SANDBOX/app.log"
status=0
wait "$APP_PID" || status=$?
APP_PID=""
grep -q "e2e: scenario $SCENARIO passed" "$SANDBOX/app.log" || status=$((status == 0 ? 1 : status))
exit "$status"
