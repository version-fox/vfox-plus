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

# A stand-in for vfox that replays the captured outputs of shared/fixtures and
# enforces the environment that the CLI contract (docs/vfox-cli-contract.md)
# requires. Implementations use it in development and in end-to-end tests
# against the fixtures, and shared/tools/measure.sh measures the memory of the
# applications with it. It keeps no state: installing a version does not add it
# to the output of `vfox list`.
#
# VFOX_PLUS_FIXTURES is the fixture directory, by default the one of vfox 1.0.13.
set -u

fixtures="${VFOX_PLUS_FIXTURES:-$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/vfox-1.0.13" && pwd)}"

# The captures escape control characters as \xHH and backslashes as \\;
# printf %b turns them back into the bytes that vfox printed.
show() {
  local text
  text="$(cat "$1"; echo .)"
  printf '%b' "${text%.}"
}
reply() {
  show "$fixtures/$1.stdout"
  show "$fixtures/$1.stderr" >&2
}

# The GUI must always disable colours and must not leak a shell session.
[ "${NO_COLOR:-}" = 1 ] || { echo "NO_COLOR is not set"; exit 90; }
[ -z "${__VFOX_PID:-}" ] || { echo "__VFOX_PID leaked"; exit 91; }
# Commands that are not about a project run in the filesystem root.
[ "$PWD" = / ] || [ "${1:-}" = use ] || [ "${1:-}" = unuse ] || { echo "not run in /: $PWD"; exit 92; }

case "$*" in
  --version) reply version ;;
  list) reply list ;;
  current) reply current ;;
  available) reply available ;;
  "config --list") reply config-list ;;
  "config "*) ;;
  "search maven") reply search-maven-installed ;;
  "search nodejs") reply search-nodejs ;;
  "search "*)
    printf 'Available versions:\n - 2.1.0 (LTS) [tool 2.1]\n - 2.0.0\n - 1.0.0\n' ;;
  "info nodejs --format "*) reply info ;;
  "info "*" --format "*)
    # Plugin information for plugins without a captured fixture.
    printf '%s\037%s\037%s\037%s\037%s' "$2" 0.1.2 "https://github.com/version-fox/vfox-$2" /tmp "$2 runtime environment." ;;
  "info "*@*) reply info-version-path ;;
  "install --yes "*)
    echo "Preinstalling ${3}..."
    for percent in 10 45 80 100; do
      printf '\rDownloading... %3d%% [=====>    ]\r' "$percent" >&2
      sleep "${VFOX_PLUS_FAKE_DELAY:-0.2}"
    done
    echo "Install ${3} success!" ;;
  "uninstall "*) echo "Uninstalled ${2} successfully!" ;;
  "use --global "* | "use --project "*)
    # vfox refuses to use a version outside of an activated shell.
    [ -n "${__VFOX_SHELL:-}" ] || { reply use-without-hook; exit 1; }
    echo "Now using ${3}." ;;
  "unuse "*) echo "Unset ${3} successfully." ;;
  "add "*) echo "Plugin ${*: -1} added successfully!" ;;
  "remove --yes "*) echo "Remove ${3} plugin successfully!" ;;
  "update "*) echo "Update ${2} plugin successfully!" ;;
  *) echo "unexpected command: $*" >&2; exit 2 ;;
esac
