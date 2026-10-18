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

# Builds, tests and packages the Compose Multiplatform implementation in
# throw-away Docker containers, so that nothing on the host changes: the JDK,
# Gradle and its caches, the vfox installation and the home directory of the
# host are neither used nor modified.
#
# Usage: compose/sandbox/sandbox.sh <command>
#
#   check     formatting, compilation with warnings as errors, the license
#             check and the unit tests with the coverage gate
#   e2e       the end-to-end tests, against the captured fixtures and against
#             the real vfox
#   package   the application image with a smoke test, .deb and .tar.gz checked
#             by lintian, then an installation into a clean Ubuntu container
#   measure   the memory use of the application on the Home page and with the
#             versions of Node.js listed
#   screenshots  screenshots of every page, against the fixtures
#   all       check, e2e and package
#   shell     an interactive shell in the sandbox
#   image     builds the sandbox image
#   clean     deletes the sandbox image, the cache volumes and the results
#
# The containers see the repository read-only and work on a copy of it.
# Results are written to dist/sandbox/compose/<command>/. Only downloading the
# dependencies, building the image and the installation test use the network.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR   Ubuntu archive mirror, for example
#                           http://mirrors.aliyun.com/ubuntu/
set -euo pipefail

IMPL=compose
IMPL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(dirname "$IMPL_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$IMPL_DIR/versions.env"
IMAGE_ARGS=(
  --build-arg "JDK_MAJOR=$JDK_MAJOR"
  --build-arg "JDK_RELEASE=$JDK_RELEASE"
  --build-arg "JDK_SHA256=$JDK_SHA256"
  --build-arg "GRADLE_VERSION=$GRADLE_VERSION"
  --build-arg "GRADLE_SHA256=$GRADLE_SHA256"
)
CACHE_VOLUMES=(vfox-plus-sandbox-gradle:/home/tester/.gradle)
PARENT_IMPLS=()
# shellcheck source=SCRIPTDIR/../../shared/sandbox/sandbox-lib.sh
. "$REPO/shared/sandbox/sandbox-lib.sh"

sandbox_main "$@"
