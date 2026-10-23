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

# Downloads one archive for install.sh, which starts several of these at the same
# time: fetch.sh <archive directory> <URL> <file name>.
#
# The file goes to partial/ first, where apt keeps what it is downloading, so that
# apt continues with what is left of a download that was cut, and checks it, as it
# checks every archive, before it installs it.
set -euo pipefail

directory="$1"
url="$2"
name="$3"

curl --fail --silent --show-error --location --retry 3 --connect-timeout 10 --max-time 600 \
  --output "$directory/partial/$name" "$url"
mv "$directory/partial/$name" "$directory/$name"
