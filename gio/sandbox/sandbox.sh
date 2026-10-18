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

# Builds, tests and packages the Gio implementation in throw-away Docker
# containers, so that nothing on the host changes: Go, the module cache and
# vfox are neither used nor modified.
#
# Usage: gio/sandbox/sandbox.sh <command>
#
#   check        gofmt, go vet, golangci-lint and the unit tests with the
#                coverage gate
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
# Results are written to dist/sandbox/gio/<command>/.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR    Ubuntu archive mirror, for example
#                            http://mirrors.aliyun.com/ubuntu/
#   SANDBOX_GOPROXY          Go module proxy, for example
#                            https://goproxy.cn,direct
set -euo pipefail

IMPL=gio
IMPL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(dirname "$IMPL_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$IMPL_DIR/versions.env"
IMAGE_ARGS=(
  --build-arg "GO_VERSION=$GO_VERSION" --build-arg "GO_SHA256=$GO_SHA256"
  --build-arg "GOLANGCI_LINT_VERSION=$GOLANGCI_LINT_VERSION"
  --build-arg "GOLANGCI_LINT_SHA256=$GOLANGCI_LINT_SHA256"
)
CACHE_VOLUMES=(
  vfox-plus-sandbox-go-modules:/home/tester/go/pkg/mod
  vfox-plus-sandbox-go-build:/home/tester/.cache/go-build
)
PARENT_IMPLS=()
# Software rendering and the embedded test server need more shared memory
# than Docker gives containers by default.
RUN_OPTIONS=(--init --shm-size=1g)
# shellcheck source=SCRIPTDIR/../../shared/sandbox/sandbox-lib.sh
. "$REPO/shared/sandbox/sandbox-lib.sh"

sandbox_main "$@"
