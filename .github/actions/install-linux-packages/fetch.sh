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

# Makes one archive of a package available for install.sh, which starts several of
# these at the same time:
#
#   fetch.sh <archive directory> <size> <file name> <hash> <address>...
#
# The hash is the one that apt prints, TYPE:digest (SHA256:... with
# -o Acquire::ForceHash=SHA256). apt checks the hash of what it downloads, but takes
# an archive that is in the directory with its size for a good one, so that the
# archive has to be checked here, before it is moved there. An archive that is
# there already is checked too, and removed when it is wrong, which is what a damaged
# cache would otherwise break the installation with.
#
# The addresses are those of the file on the mirrors, in the order to try them. The
# download goes to partial/ first, where apt keeps what it is downloading, and
# continues from what is there, so that a connection that is cut, or dropped for being
# slow, loses nothing.
#
# A connection that gets less than APT_MIN_SPEED bytes a second (262144 by default)
# for APT_SPEED_TIME seconds (10) is dropped, and the download continues from the next
# mirror: a mirror is slow for some runners and not for others, and the speed of the
# next one is not known before it is tried. When every mirror has been slow in turn, a
# slow download is better than none: the best of them continues, and only a stalled
# connection is dropped. A file that no mirror sends in three rounds, or that has the
# wrong hash three times, is given up.
#
# APT_DEADLINE, in seconds since the epoch, is the time when the download ends.
set -uo pipefail

[[ $# -ge 5 ]] || { echo "Usage: fetch.sh <archive directory> <size> <file name> <hash> <address>..." >&2; exit 2; }

directory="$1"
size="$2"
name="$3"
hash="$4"
shift 4
addresses=("$@")
deadline="${APT_DEADLINE:?APT_DEADLINE is the time when the download ends, in seconds since the epoch}"
minimum="${APT_MIN_SPEED:-262144}"
window="${APT_SPEED_TIME:-10}"
final="$directory/$name"
partial="$directory/partial/$name"
error="$(mktemp)"
trap 'rm -f "$error"' EXIT

# The size of what has been downloaded.
have() {
  stat --format %s "$partial" 2> /dev/null || echo 0
}

# right <file>: whether the file has the hash that apt printed.
right() {
  local tool
  case "${hash%%:*}" in
    MD5Sum) tool=md5sum ;;
    SHA1) tool=sha1sum ;;
    SHA256) tool=sha256sum ;;
    SHA512) tool=sha512sum ;;
    *) return 1 ;;
  esac
  [[ "$("$tool" "$1" 2> /dev/null | cut -d ' ' -f 1)" == "${hash#*:}" ]]
}

if [[ -e "$final" ]]; then
  if [[ "$(stat --format %s "$final")" -eq "$size" ]] && right "$final"; then
    exit 0
  fi
  echo "$name: the archive in the directory is not the one that apt expects; removed"
  rm -f "$final"
fi

fruitless=0
wrong=0
last=""
while true; do
  if [[ "$(have)" -eq "$size" ]]; then
    right "$partial" && break
    echo "$name: the hash is not $hash; starting again"
    rm -f "$partial"
    wrong=$((wrong + 1))
    ((wrong < 3)) || { echo "$name: no mirror sends the right file" >&2; exit 1; }
    # The mirror that completed it is the last one to try again.
    addresses=()
    for address in "$@"; do
      [[ "$address" == "$last" ]] || addresses+=("$address")
    done
    [[ -z "$last" ]] || addresses+=("$last")
  fi
  # More than the file has is not a part of it.
  [[ "$(have)" -le "$size" ]] || rm -f "$partial"
  start="$(have)"
  best=""
  best_gain=0
  for address in "${addresses[@]}"; do
    before="$(have)"
    [[ "$before" -lt "$size" ]] || break
    left=$((deadline - $(date +%s)))
    ((left > 0)) || exit 1
    last="$address"
    speed="$(curl --fail --silent --show-error --location --connect-timeout 10 --max-time "$left" \
      --speed-limit "$minimum" --speed-time "$window" --continue-at - \
      --write-out '%{speed_download}' --output "$partial" "$address" 2> "$error")"
    status=$?
    # A server that does not continue a download sends all of the file, which curl
    # does not take (33): start again.
    [[ $status -ne 33 ]] || rm -f "$partial"
    gain=$(($(have) - before))
    if ((gain > best_gain)); then
      best="$address"
      best_gain=$gain
    fi
    if [[ "$(have)" -ne "$size" ]]; then
      host="${address#*://}"
      echo "$name: $(have) of $size bytes, ${speed%.*} bytes a second from ${host%%/*}: $(tr '\n' ' ' < "$error")"
    fi
  done
  # All of it came, to be checked at the top.
  [[ "$(have)" -ne "$size" ]] || continue
  if [[ "$(have)" -gt "$start" ]]; then
    fruitless=0
  else
    fruitless=$((fruitless + 1))
    ((fruitless < 3)) || { echo "$name: no mirror sends it" >&2; exit 1; }
  fi
  # Every mirror was slow in turn, or failed: go on with the best of them.
  minimum=1000
  if [[ -n "$best" ]]; then
    addresses=("$best")
    for address in "$@"; do
      [[ "$address" == "$best" ]] || addresses+=("$address")
    done
  fi
  sleep 1
done
mv "$partial" "$final"
