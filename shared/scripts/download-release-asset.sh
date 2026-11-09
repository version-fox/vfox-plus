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

# Download a release asset, falling back to the GitHub API when the release
# download is slow or answers 504: GitHub serves the two from different
# services, and the API one has been the dependable one.
#
#   download-release-asset.sh OWNER/REPO TAG ASSET OUTPUT [SHA256]
#
# GH_TOKEN, when set, raises the API rate limit; public repositories need no
# token. jq is required for the fallback.
set -euo pipefail

if [ "$#" -lt 4 ] || [ "$#" -gt 5 ]; then
  echo "usage: $0 OWNER/REPO TAG ASSET OUTPUT [SHA256]" >&2
  exit 2
fi

repo=$1
tag=$2
asset=$3
output=$4
want=${5:-}

curl_options=(--fail --silent --show-error --location
  --retry 3 --retry-all-errors --connect-timeout 20 --max-time 600)
auth=()
if [ -n "${GH_TOKEN:-}" ]; then
  auth=(-H "Authorization: Bearer ${GH_TOKEN}")
fi

direct="https://github.com/${repo}/releases/download/${tag}/${asset}"
echo "Downloading ${direct}" >&2
# No token on the release URL: curl would carry the header to the asset host
# of the redirect, and the API below is the reliable way in any case.
if ! curl "${curl_options[@]}" -o "${output}" "${direct}"; then
  echo "The release download failed; asking the GitHub API for the asset." >&2
  release=$(curl "${curl_options[@]}" "${auth[@]}" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${repo}/releases/tags/${tag}")
  id=$(printf '%s' "${release}" | jq -r --arg name "${asset}" \
    '[.assets[]? | select(.name == $name) | .id] | first // empty')
  if [ -z "${id}" ]; then
    echo "No asset named ${asset} in ${repo} ${tag}." >&2
    exit 1
  fi
  curl "${curl_options[@]}" "${auth[@]}" \
    -H "Accept: application/octet-stream" \
    -o "${output}" "https://api.github.com/repos/${repo}/releases/assets/${id}"
fi

if [ -n "${want}" ]; then
  echo "${want}  ${output}" | sha256sum --check --strict -
fi
