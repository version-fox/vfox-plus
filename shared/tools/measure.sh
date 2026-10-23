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

# Measures the memory that an implementation of vfox+ uses, the same way for
# every implementation: starts the application with a new, empty HOME and the
# fake vfox of shared/fixtures, in its "measure" scenario, and prints the
# resident (RSS) and proportional (PSS) set size of the application and all of
# its child processes, such as the processes of a web view, at each measuring
# point the application reports.
#
# Usage: shared/tools/measure.sh <application command...>
#
# The application runs on the current display. With VFOX_PLUS_E2E_SCENARIO set
# to "measure" it must show the Home page and print "e2e: MEASURE home", list
# the available versions of Node.js and print "e2e: MEASURE nodejs", then print
# "e2e: scenario measure passed" and exit with code 0, waiting a few seconds
# after each point so that the measurement sees a settled application. Prints
# lines "memory (<point>): RSS <n> MB, PSS <n> MB in <n> processes".
#
# With VFOX_PLUS_MEASURE_OUTPUT set to a file name, the same lines also replace
# the content of that file. CI uploads it for the memory report of pull
# requests (.github/workflows/memory.yml).
set -euo pipefail

TOOLS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SHARED="$(dirname "$TOOLS_DIR")"
[[ $# -gt 0 ]] || { echo "Usage: measure.sh <application command...>" >&2; exit 2; }

SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/vfox-plus-measure.XXXXXX")"
HOME_DIR="$SANDBOX/home"
APP_PID=""
# shellcheck disable=SC2317,SC2329 # Invoked by the trap below.
cleanup() {
  [[ -z "$APP_PID" ]] || kill "$APP_PID" 2> /dev/null || true
  # The processes of a web view may write their caches for a moment after the
  # application has ended; the result of the test does not depend on them.
  local attempt
  for attempt in 1 2 3 4 5; do
    rm -rf "$SANDBOX" 2> /dev/null && return 0
    sleep "$attempt"
  done
  echo "Could not remove $SANDBOX" >&2
}
trap cleanup EXIT

OUTPUT="${VFOX_PLUS_MEASURE_OUTPUT:-}"
if [[ -n "$OUTPUT" ]]; then
  mkdir -p "$(dirname "$OUTPUT")"
  : > "$OUTPUT"
fi

mkdir -p "$HOME_DIR/.config/vfox-plus"
vfox="$SANDBOX/vfox"
install -m 755 "$SHARED/fixtures/fake-vfox.sh" "$vfox"
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

# VFOX_HOME would make vfox use a shared vfox directory instead of HOME. The
# fake vfox runs as if the application was started from an activated shell.
env -u VFOX_HOME -u XDG_CONFIG_HOME HOME="$HOME_DIR" LANG=C.UTF-8 VFOX_PLUS_E2E_SCENARIO=measure \
  VFOX_PLUS_E2E_VFOX="$vfox" VFOX_PLUS_FIXTURES="$SHARED/fixtures/vfox-1.0.13" __VFOX_PID=1 __VFOX_SHELL=bash \
  "$@" > "$SANDBOX/app.log" 2>&1 &
APP_PID=$!
printed=0
deadline=$((SECONDS + ${VFOX_PLUS_E2E_TIMEOUT:-300}))
show_new_lines() {
  local lines result
  lines="$(wc -l < "$SANDBOX/app.log")"
  if [[ $lines -gt $printed ]]; then
    while IFS= read -r line; do
      echo "$line"
      if [[ "$line" == *"e2e: MEASURE "* ]]; then
        result="memory (${line##*MEASURE }): $(memory_of "$APP_PID")"
        echo "$result"
        [[ -z "$OUTPUT" ]] || echo "$result" >> "$OUTPUT"
      fi
    done < <(sed -n "$((printed + 1)),${lines}p" "$SANDBOX/app.log")
    printed=$lines
  fi
}
while kill -0 "$APP_PID" 2> /dev/null; do
  show_new_lines
  if [[ $SECONDS -gt $deadline ]]; then
    echo "The application did not finish the measure scenario in time." >&2
    exit 1
  fi
  sleep 0.2
done
show_new_lines
status=0
wait "$APP_PID" || status=$?
APP_PID=""
if ! grep -q "e2e: scenario measure passed" "$SANDBOX/app.log"; then
  tail -n +"$((printed + 1))" "$SANDBOX/app.log"
  echo "The application did not report the end of the measure scenario." >&2
  status=$((status == 0 ? 1 : status))
fi
exit "$status"
