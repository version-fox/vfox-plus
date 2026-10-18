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

# Builds, tests and packages the Tauri implementation in throw-away Docker
# containers, so that nothing on the host changes: Rust, Node.js, their caches,
# the vfox installation and the home directory of the host are neither used
# nor modified.
#
# Usage: tauri/sandbox/sandbox.sh <command>
#
#   check     formatting, Clippy and the unit tests of the Rust side
#   e2e       the end-to-end scenarios of the web front end in the application,
#             against the captured fixtures and against the real vfox
#   package   release build with a smoke test, .deb and .tar.gz checked by
#             lintian, then an installation into a clean Ubuntu container
#   measure   the memory use of the release build on the Home page and with
#             the versions of Node.js listed
#   all       check, e2e and package
#   shell     an interactive shell in the sandbox
#   image     builds the sandbox images
#   clean     deletes the sandbox image, the cache volumes and the results
#
# The front end is checked by web/sandbox/sandbox.sh. The containers see the
# repository read-only and work on a copy of it. Results are written to
# dist/sandbox/tauri/<command>/. Only downloading the packages, building the
# images and the installation test use the network.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR   Ubuntu archive mirror, for example
#                           http://mirrors.aliyun.com/ubuntu/
#   SANDBOX_NPM_REGISTRY    npm registry mirror, for example
#                           https://registry.npmmirror.com
set -euo pipefail

IMPL=tauri
IMPL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(dirname "$IMPL_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$IMPL_DIR/versions.env"
IMAGE_ARGS=(
  --build-arg "RUST_VERSION=$RUST_VERSION"
  --build-arg "RUSTUP_VERSION=$RUSTUP_VERSION"
  --build-arg "RUSTUP_SHA256=$RUSTUP_SHA256"
)
CACHE_VOLUMES=(
  vfox-plus-sandbox-npm-cache:/home/tester/.npm
  vfox-plus-sandbox-cargo-registry:/home/tester/.cargo/registry
)
PARENT_IMPLS=(web)
# shellcheck source=SCRIPTDIR/../../shared/sandbox/sandbox-lib.sh
. "$REPO/shared/sandbox/sandbox-lib.sh"

sandbox_main "$@"
