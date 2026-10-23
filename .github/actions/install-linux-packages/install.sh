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

# Installs Ubuntu packages on a runner of GitHub Actions in two steps, which the
# action next to this file (action.yml) wraps in a cache:
#
#   1. Downloads the packages into /var/cache/apt/archives, with a time limit
#      for each attempt, and tries again when an attempt runs out of time. The
#      Ubuntu mirror of the runners is sometimes throttled to about 40 KB/s, and
#      downloading 94 MB took 45 minutes then (a minute at other times), which
#      is what a job of 25 or 45 minutes cannot wait for. The throttle seems to
#      be for each connection (the rate did not change while the number of jobs
#      that were downloading fell from 11 to 5), and apt downloads from a host
#      over one connection, one file after the other: eight files at a time
#      (fetch.sh) are eight times as fast then, and as fast as before when the
#      mirror is not throttled. apt then fetches what is left and checks every
#      archive. What an attempt has downloaded is kept, so the next one
#      continues with the rest.
#   2. Installs them from the archives, which needs no network.
#
# Finally it removes the archives of versions that the package lists no longer
# have, so that the cache does not grow from week to week.
#
# Usage: install.sh <package>...
#
# Set APT_ATTEMPTS and APT_ATTEMPT_LIMIT (a time for timeout(1), such as 5m) to
# change the number of attempts of the download (3) and the time of each (5m):
# the three of them have to end before the timeout of the job, for the action to
# save what they downloaded. APT_PARALLEL is the number of files that are
# downloaded at the same time (8; 0 leaves it all to apt).
set -euo pipefail

[[ $# -gt 0 ]] || { echo "Usage: install.sh <package>..." >&2; exit 2; }

packages=("$@")
attempts="${APT_ATTEMPTS:-3}"
limit="${APT_ATTEMPT_LIMIT:-5m}"
parallel="${APT_PARALLEL:-8}"
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
archives=/var/cache/apt/archives

# A request that gets no data for 30 seconds fails and is tried again, up to 3
# times; apt waits for 2 minutes by default. The archives stay where they are
# after the installation: a runner has no hook that removes them, but a
# container of Docker has.
apt_options=(-o Acquire::Retries=3 -o Acquire::http::Timeout=30 -o Acquire::https::Timeout=30
  -o APT::Keep-Downloaded-Packages=true)

# apt_get <time limit> <arguments...>: apt-get as root, for at most the time (0 is
# no limit), without questions.
apt_get() {
  local time="$1"
  shift
  sudo timeout "$time" env DEBIAN_FRONTEND=noninteractive apt-get "${apt_options[@]}" "$@"
}

# prefetch: downloads the archives that the installation needs and that are not in
# the directory, $parallel at a time, from the mirror that apt would use. It only
# saves time: what it does not fetch, apt does.
prefetch() {
  local mirror list
  # The runners list their mirrors in a file, and apt prints the address of the file
  # for the mirror in the addresses of the packages.
  mirror="$(awk 'NF { print $1; exit }' /etc/apt/apt-mirrors.txt 2> /dev/null || true)"
  list="$(mktemp)"
  # The lines are 'address' file size checksum.
  apt_get 2m install --print-uris -y --no-install-recommends "${packages[@]}" 2> /dev/null \
    | sed -n "s/^'\([^']*\)' \([^ ]*\) \([0-9]*\) .*/\1 \2 \3/p" \
    | while read -r address name size; do
      if [[ -n "$mirror" ]]; then
        address="${address/#mirror+file:\/etc\/apt\/apt-mirrors.txt\//$mirror}"
      fi
      if [[ "$(stat --format %s "$archives/$name" 2> /dev/null || echo 0)" -ne "$size" ]]; then
        echo "$address $name"
      fi
    done > "$list" || true
  sudo mkdir -p "$archives/partial"
  # One line, an address and a file name, for each run of fetch.sh, and no run
  # when there is no line.
  sudo timeout "$limit" xargs -r -a "$list" -P "$parallel" -L 1 "$here/fetch.sh" "$archives" \
    || echo "::warning::Not every archive was downloaded by fetch.sh; apt fetches the rest."
  rm -f "$list"
}

for ((attempt = 1; attempt <= attempts; attempt++)); do
  if apt_get 2m update \
    && { [[ "$parallel" -eq 0 ]] || prefetch; } \
    && apt_get "$limit" install --download-only -y --no-install-recommends "${packages[@]}"; then
    break
  fi
  if [[ $attempt -eq $attempts ]]; then
    echo "::error::Could not download the packages in $attempts attempts of $limit; the mirror is probably slow."
    exit 1
  fi
  echo "::warning::Attempt $attempt of $attempts to download the packages failed or took more than $limit."
  sleep 5
done

# Killing dpkg half way would leave packages unpacked but not configured, so
# the installation has no time limit; it needs no network.
apt_get 0 install -y --no-install-recommends "${packages[@]}"
apt_get 0 autoclean
