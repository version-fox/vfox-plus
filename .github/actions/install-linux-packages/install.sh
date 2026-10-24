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
#   1. Downloads the packages that the cache lacks into /var/cache/apt/archives, in
#      15 minutes at most. fetch.sh does it, eight files at a time and the largest
#      first, from the mirrors of the runner: it continues what was cut, tries the
#      next mirror when one is slow, and checks the hash of every archive, those of
#      the cache too, because apt takes an archive of the right size for a good one.
#      Then apt fetches what is left.
#   2. Installs them from the archives, which needs no network.
#
# Why not apt alone: the Ubuntu mirror of the runners is slow for some of them. On
# 2026-10-23 a job got 40 KB/s for a connection while other jobs of the same minute
# downloaded 90 MB in seconds, and its 94 MB did not come in 30 minutes (a minute on a
# good day), which a job of 25 or 45 minutes cannot wait for; a file of 25 MB alone
# takes 10 minutes at that speed. apt downloads from one mirror, over one connection,
# one file after the other, however slow it is.
#
# Finally it removes the archives of versions that the package lists no longer
# have, so that the cache does not grow from week to week.
#
# Usage: install.sh <package>...
#
# Set APT_DOWNLOAD_SECONDS to change the time of the download (900): it has to end
# before the timeout of the job, for the action to save what was downloaded.
# APT_PARALLEL is the number of files that are downloaded at the same time (8; 0
# leaves all of it to apt). fetch.sh has two more, APT_MIN_SPEED and APT_SPEED_TIME.
set -euo pipefail

[[ $# -gt 0 ]] || { echo "Usage: install.sh <package>..." >&2; exit 2; }

packages=("$@")
seconds="${APT_DOWNLOAD_SECONDS:-900}"
parallel="${APT_PARALLEL:-8}"
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
archives=/var/cache/apt/archives
deadline=$(($(date +%s) + seconds))

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

# archive_size <file name>: the size of the archive, 0 when it is not there.
archive_size() {
  stat --format %s "$archives/$1" 2> /dev/null || echo 0
}

# missing <list>: the number of archives of the list that are not in the directory
# with their size.
missing() {
  local size name count=0
  while read -r size name _; do
    [[ "$(archive_size "$name")" -eq "$size" ]] || count=$((count + 1))
  done < "$1"
  echo "$count"
}

# prefetch: makes the archives that the installation needs available in the
# directory, and checks the ones that are there already. It only saves time: what it
# does not fetch, apt does.
prefetch() {
  local mirrors list empty line path total before after size name hash address mirror
  if ! command -v curl > /dev/null; then
    echo "curl is missing, apt downloads everything."
    return
  fi
  # The runners list their mirrors in a file, one for each line, the best first.
  mapfile -t mirrors < <(awk 'NF { print $1 }' /etc/apt/apt-mirrors.txt 2> /dev/null)
  list="$(mktemp)"
  # apt prints 'address' name size hash for each archive (a SHA256 hash is what
  # fetch.sh checks), but not for the archives that are in the directory, which it
  # takes for good ones: it is shown an empty directory, and fetch.sh checks them. The
  # address of a runner starts with that of the file of the mirrors, which stands for
  # one of them: the line of an archive has its address on each mirror, for fetch.sh.
  # The largest come first, so that none of them is the last and alone.
  empty="$(mktemp -d)"
  apt_get 2m -o Dir::Cache::Archives="$empty" -o Acquire::ForceHash=SHA256 install --print-uris -y --no-install-recommends "${packages[@]}" 2> /dev/null \
    | sed -n "s/^'\([^']*\)' \([^ ]*\) \([0-9]*\) \([^ ]*\).*/\3 \2 \4 \1/p" \
    | while read -r size name hash address; do
      line="$size $name $hash"
      path="${address#mirror+file:/etc/apt/apt-mirrors.txt/}"
      if [[ "$path" != "$address" && ${#mirrors[@]} -gt 0 ]]; then
        for mirror in "${mirrors[@]}"; do
          line+=" ${mirror%/}/$path"
        done
      else
        line+=" $address"
      fi
      echo "$line"
    done \
    | sort -k 1,1nr > "$list" || true
  rmdir "$empty" 2> /dev/null || true
  total="$(wc -l < "$list")"
  before="$(missing "$list")"
  echo "$before of $total archives are not in $archives."
  sudo mkdir -p "$archives/partial"
  # One line, a size, a name, a hash and addresses, for each run of fetch.sh, and no
  # run when there is no line. fetch.sh ends at the deadline by itself; the limit here
  # is for what goes wrong in it.
  sudo timeout "$((deadline - $(date +%s) + 60))" \
    env APT_DEADLINE="$deadline" APT_MIN_SPEED="${APT_MIN_SPEED:-}" APT_SPEED_TIME="${APT_SPEED_TIME:-}" \
    xargs -r -a "$list" -P "$parallel" -L 1 "$here/fetch.sh" "$archives" || true
  after="$(missing "$list")"
  echo "fetch.sh downloaded $((before - after)) archives."
  if ((after > 0)); then
    echo "::warning::$after archives are left for apt."
  fi
  rm -f "$list"
}

for ((attempt = 1; attempt <= 3; attempt++)); do
  apt_get 2m update && break
  if ((attempt == 3)); then
    echo "::error::apt-get update failed in 3 attempts."
    exit 1
  fi
  sleep 5
done

[[ "$parallel" -eq 0 ]] || prefetch

# apt downloads what is still missing, and checks what it downloads, with what is
# left of the time (a minute at least, which is enough to see that nothing is).
left=$((deadline - $(date +%s)))
((left > 60)) || left=60
if ! apt_get "${left}s" install --download-only -y --no-install-recommends "${packages[@]}"; then
  echo "::error::Could not download the packages (the limit is $seconds seconds); the mirrors are probably slow."
  exit 1
fi

# Killing dpkg half way would leave packages unpacked but not configured, so
# the installation has no time limit; it needs no network.
apt_get 0 install -y --no-install-recommends "${packages[@]}"
apt_get 0 autoclean
