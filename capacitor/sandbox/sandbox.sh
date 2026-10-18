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

# Builds, tests and packages the Capacitor implementation in throw-away Docker
# containers, so that nothing on the host changes: Node.js, Electron, their
# caches, the vfox installation and the home directory of the host are neither
# used nor modified.
#
# Usage: capacitor/sandbox/sandbox.sh <command>
#
#   check        formatting, type checks, ESLint, and the unit tests with the
#                coverage gate
#   e2e          the end-to-end scenarios against the captured fixtures and
#                against the real vfox
#   package      release build with a smoke test, .deb and .tar.gz checked by
#                lintian, then an installation into a clean Ubuntu container
#   measure      the memory use on the Home page and with the versions of
#                Node.js listed
#   screenshots  screenshots of every page, against the fixtures
#   all          check, e2e and package
#   shell        an interactive shell in the sandbox
#   image        builds the sandbox images
#   clean        deletes the sandbox image, the cache volumes and the results
#
# The containers see the repository read-only and work on a copy of it.
# Results are written to dist/sandbox/capacitor/<command>/. Only downloading
# the packages, building the images and the installation test use the network.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR    Ubuntu archive mirror, for example
#                            http://mirrors.aliyun.com/ubuntu/
#   SANDBOX_NPM_REGISTRY     npm registry mirror, for example
#                            https://registry.npmmirror.com
#   SANDBOX_ELECTRON_MIRROR  mirror of the Electron releases, for example
#                            https://npmmirror.com/mirrors/electron/
set -euo pipefail

IMPL=capacitor
IMPL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(dirname "$IMPL_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$IMPL_DIR/versions.env"
IMAGE_ARGS=()
CACHE_VOLUMES=(
  vfox-plus-sandbox-npm-cache:/home/tester/.npm
  vfox-plus-sandbox-electron-cache:/home/tester/.cache/electron
)
PARENT_IMPLS=(web)
# Chromium keeps fonts and other shared data in /dev/shm, of which Docker gives
# containers 64 MB; the fonts of Chinese alone need more.
RUN_OPTIONS=(--shm-size=1g)
# shellcheck source=SCRIPTDIR/../../shared/sandbox/sandbox-lib.sh
. "$REPO/shared/sandbox/sandbox-lib.sh"

sandbox_main "$@"
