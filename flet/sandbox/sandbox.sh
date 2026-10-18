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

# Builds, tests and packages the Flet implementation in throw-away Docker
# containers, so that nothing on the host changes: the Flutter SDK, uv and
# its caches, the Flet template cache, the vfox installation and the home
# directory of the host are neither used nor modified.
#
# Usage: flet/sandbox/sandbox.sh <command>
#
#   check        formatting, type checks, the linter, the unit tests with the
#                coverage gate, and the Dart analysis of the extension
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
# Results are written to dist/sandbox/flet/<command>/. Only downloading the
# packages, building the images and the installation test use the network.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR    Ubuntu archive mirror, for example
#                            http://mirrors.aliyun.com/ubuntu/
#   SANDBOX_UV_MIRROR        PyPI mirror for uv, for example
#                            https://mirrors.aliyun.com/pypi/simple/
set -euo pipefail

IMPL=flet
IMPL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO="$(dirname "$IMPL_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$IMPL_DIR/versions.env"

# The toolchains, downloaded on the host for COPY in the Dockerfile: downloads
# inside `docker build` are corrupted from time to time on some networks,
# which curl does not retry, while the host connection is reliable. Each file
# is verified against the checksum of its release and downloaded only once.
fetch() {
  local file="$1" url="$2" checksum="$3"
  if [[ -f "$file" ]] && echo "$checksum  $file" | sha256sum --check --strict - > /dev/null 2>&1; then
    return 0
  fi
  local attempt
  for attempt in 1 2 3 4 5 6 7 8; do
    if curl -fsSL --retry 3 -o "$file.part" "$url" \
      && echo "$checksum  $file.part" | sha256sum --check --strict -; then
      mv "$file.part" "$file"
      return 0
    fi
    echo "Downloading $url failed, retrying ($attempt)" >&2
    sleep "$attempt"
  done
  rm -f "$file.part"
  return 1
}
mkdir -p "$IMPL_DIR/sandbox/.downloads"
fetch "$IMPL_DIR/sandbox/.downloads/uv.tar.gz" \
  "https://github.com/astral-sh/uv/releases/download/$UV_VERSION/uv-x86_64-unknown-linux-gnu.tar.gz" \
  "$UV_SHA256"
fetch "$IMPL_DIR/sandbox/.downloads/flutter.tar.xz" \
  "https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_${FLUTTER_VERSION}-stable.tar.xz" \
  "$FLUTTER_SHA256"
IMAGE_ARGS=(
  --build-arg "UV_VERSION=$UV_VERSION" --build-arg "UV_SHA256=$UV_SHA256"
  --build-arg "FLUTTER_VERSION=$FLUTTER_VERSION" --build-arg "FLUTTER_SHA256=$FLUTTER_SHA256"
)
CACHE_VOLUMES=(
  vfox-plus-sandbox-uv-cache:/home/tester/.cache/uv
  vfox-plus-sandbox-flet-cache:/home/tester/.flet
  vfox-plus-sandbox-pub-cache:/home/tester/.pub-cache
  # The release build: deps fills it with a full `flet build`, whose resolved
  # packages and compiled outputs the tasks without network reuse. `flet
  # build` offers no offline mode, so without this every one of them would
  # reach the network. The paths inside are the same in every container.
  vfox-plus-sandbox-flet-build:/home/tester/work/flet/build
)
PARENT_IMPLS=()
# The embedded Python and the Flutter engine need more shared memory than
# Docker gives containers.
RUN_OPTIONS=(--init --shm-size=1g)
# shellcheck source=SCRIPTDIR/../../shared/sandbox/sandbox-lib.sh
. "$REPO/shared/sandbox/sandbox-lib.sh"

# `flet build` offers no offline mode: the Python and Flutter resolvers, pip's
# build isolation and the Flutter tool's own checks always reach the network,
# unlike cargo --offline or npm --offline, even with every cache warm. The
# tasks that (re)build therefore keep the network; deps warms the caches so
# that they stay fast. Only check, which never builds, runs without network.
# This shadows sandbox_main of shared/sandbox/sandbox-lib.sh; keep the other
# branches identical to it.
sandbox_e2e() {
  run_task bridge deps
  run_task bridge e2e
}

sandbox_package() {
  run_task bridge deps
  run_task bridge package
  install_test
}

sandbox_all() {
  run_task bridge deps
  run_task none check
  run_task bridge e2e
  run_task bridge package
  install_test
}

sandbox_main() {
  case "${1:-}" in
    check)
      ensure_image
      sandbox_check
      ;;
    e2e | package | all)
      ensure_image
      "sandbox_$1"
      ;;
    measure | screenshots)
      ensure_image
      run_task bridge deps
      run_task bridge "$1"
      ;;
    shell)
      ensure_image
      run_task bridge deps
      run_task bridge shell --interactive --tty
      ;;
    image)
      ensure_image
      ;;
    image-name)
      impl_image
      ;;
    clean)
      docker image ls --format '{{.Repository}}:{{.Tag}}' "$IMAGE_NAME" | xargs --no-run-if-empty docker image rm
      local volume
      for volume in ${CACHE_VOLUMES[@]+"${CACHE_VOLUMES[@]}"}; do
        docker volume rm --force "${volume%%:*}" > /dev/null
      done
      rm -rf "${RESULTS:?}"
      ;;
    -h | --help | help)
      sandbox_usage
      ;;
    *)
      sandbox_usage >&2
      exit 2
      ;;
  esac
}

sandbox_main "$@"
