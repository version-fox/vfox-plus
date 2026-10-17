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

# Checks the web front end in throw-away Docker containers, so that nothing
# on the host changes: Node.js, the npm cache, the vfox installation and the
# home directory of the host are neither used nor modified.
#
# Usage: web/sandbox/sandbox.sh <command>
#
#   check   formatting, lint, type checking, unit tests with the coverage gate
#           and the production builds for Tauri and Wails
#   all     the same as check; the end-to-end tests and the packages are in
#           the Tauri and Wails sandboxes
#   dev     serves the front end on http://127.0.0.1:5173 against a fake vfox
#           that replays shared/fixtures, for development in a browser
#   shell   an interactive shell in the sandbox
#   image   builds the sandbox image
#   clean   deletes the sandbox image, the npm cache volume and the results
#
# The containers see the repository read-only and work on a copy of it.
# Results are written to dist/sandbox/web/<command>/. Only downloading the npm
# packages and building the image use the network.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR   Ubuntu archive mirror, for example
#                           http://mirrors.aliyun.com/ubuntu/
#   SANDBOX_NPM_REGISTRY    npm registry mirror, for example
#                           https://registry.npmmirror.com
set -euo pipefail

IMPL=web
IMPL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(dirname "$IMPL_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$IMPL_DIR/versions.env"
IMAGE_ARGS=(--build-arg "NODE_VERSION=$NODE_VERSION" --build-arg "NODE_SHA256=$NODE_SHA256")
CACHE_VOLUMES=(vfox-plus-sandbox-npm-cache:/home/tester/.npm)
PARENT_IMPLS=()
# shellcheck source=SCRIPTDIR/../../shared/sandbox/sandbox-lib.sh
. "$REPO/shared/sandbox/sandbox-lib.sh"

sandbox_all() {
  sandbox_check
}

case "${1:-}" in
  dev)
    ensure_image
    run_task bridge deps
    echo "Serving on http://127.0.0.1:5173; stop with Ctrl+C."
    run_task bridge dev --publish 127.0.0.1:5173:5173 --interactive --tty
    ;;
  e2e | package)
    echo "The web front end is tested end to end and packaged by tauri/ and wails/." >&2
    exit 2
    ;;
  *)
    sandbox_main "$@"
    ;;
esac
