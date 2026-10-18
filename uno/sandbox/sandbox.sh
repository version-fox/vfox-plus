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

# Builds, tests and packages the Uno implementation in throw-away Docker
# containers, so that nothing on the host changes: the .NET SDK, the NuGet cache
# and vfox are neither used nor modified.
#
# Usage: uno/sandbox/sandbox.sh <command>
#
#   check        dotnet format, the build with warnings as errors and the unit
#                tests with the coverage gate
#   e2e          the end-to-end scenarios, against the fixtures and against
#                the real vfox
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
# Results are written to dist/sandbox/uno/<command>/.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR    Ubuntu archive mirror, for example
#                            http://mirrors.aliyun.com/ubuntu/
set -euo pipefail

IMPL=uno
IMPL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(dirname "$IMPL_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$IMPL_DIR/versions.env"
IMAGE_ARGS=(
  --build-arg "DOTNET_VERSION=$DOTNET_VERSION" --build-arg "DOTNET_SHA512=$DOTNET_SHA512"
)
CACHE_VOLUMES=(
  vfox-plus-sandbox-uno-nuget:/home/tester/.nuget
  vfox-plus-sandbox-uno-dotnet:/home/tester/.dotnet
)
PARENT_IMPLS=()
# The scenarios of the end-to-end tests and the private bus of the portal tests
# need more shared memory than Docker gives containers by default.
RUN_OPTIONS=(--init --shm-size=1g)
# shellcheck source=SCRIPTDIR/../../shared/sandbox/sandbox-lib.sh
. "$REPO/shared/sandbox/sandbox-lib.sh"

sandbox_main "$@"
