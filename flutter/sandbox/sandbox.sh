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

# Builds, tests and packages the Flutter implementation in throw-away Docker
# containers, so that nothing on the host changes: the Flutter SDK, the pub
# cache, the vfox installation and the home directory of the host are neither
# used nor modified.
#
# Usage: flutter/sandbox/sandbox.sh <command>
#
#   check        formatting, generated translations, analysis, unit and widget
#                tests with the coverage gate
#   e2e          integration tests of the Linux application, against the
#                captured fixtures and against the real vfox
#   package      release build with a smoke test, .deb and .tar.gz checked by
#                lintian, then an installation into a clean Ubuntu container
#   all          check, e2e and package
#   screenshots  renders the screenshots of the README, for the image host
#   fixtures     captures the vfox outputs of shared/fixtures (uses the network)
#   shell        an interactive shell in the sandbox
#   image        builds the sandbox image again
#   clean        deletes the sandbox images, the pub cache volume and the results
#
# The containers see the repository read-only and work on a copy of it.
# Results are written to dist/sandbox/<command>/. Only downloading the pub
# packages, building the image, the installation test and fixtures use the
# network; everything else runs without it.
#
# Environment variables for slow connections to the default servers:
#   SANDBOX_UBUNTU_MIRROR             Ubuntu archive mirror, for example
#                                     http://mirrors.aliyun.com/ubuntu/
#   SANDBOX_FLUTTER_STORAGE_BASE_URL  Flutter storage mirror, for example
#                                     https://storage.flutter-io.cn
set -euo pipefail

SANDBOX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FLUTTER_DIR="$(dirname "$SANDBOX_DIR")"
REPO="$(dirname "$FLUTTER_DIR")"
# shellcheck source=SCRIPTDIR/../versions.env
. "$FLUTTER_DIR/versions.env"

IMAGE_NAME=vfox-plus-sandbox
IMAGE="$IMAGE_NAME:flutter-$FLUTTER_VERSION-vfox-$VFOX_VERSION"
BASE_IMAGE=ubuntu:26.04
PUB_CACHE_VOLUME=vfox-plus-sandbox-pub-cache
RESULTS="$REPO/dist/sandbox"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git -C "$REPO" log -1 --format=%ct 2> /dev/null || date +%s)}"

usage() {
  sed -n '/^# Usage:/,/^set -euo/{/^set -euo/d; s/^# \{0,1\}//; p}' "${BASH_SOURCE[0]}"
}

build_image() {
  local args=(
    --tag "$IMAGE"
    --build-arg "FLUTTER_VERSION=$FLUTTER_VERSION" --build-arg "FLUTTER_SHA256=$FLUTTER_SHA256"
    --build-arg "VFOX_VERSION=$VFOX_VERSION" --build-arg "VFOX_SHA256=$VFOX_SHA256"
  )
  if [[ -n "${SANDBOX_UBUNTU_MIRROR:-}" ]]; then
    args+=(--build-arg "UBUNTU_MIRROR=$SANDBOX_UBUNTU_MIRROR")
  fi
  if [[ -n "${SANDBOX_FLUTTER_STORAGE_BASE_URL:-}" ]]; then
    args+=(--build-arg "FLUTTER_STORAGE_BASE_URL=$SANDBOX_FLUTTER_STORAGE_BASE_URL")
  fi
  docker build "${args[@]}" "$SANDBOX_DIR"
}

ensure_image() {
  docker image inspect "$IMAGE" > /dev/null 2>&1 || build_image
}

# Under rootless Docker, root in a container is the user on the host. A
# rootful daemon needs to be told whom the results belong to.
OWNER=()
if ! docker info --format '{{.SecurityOptions}}' 2> /dev/null | grep -q rootless; then
  OWNER=(--env "OUT_OWNER=$(id -u):$(id -g)")
fi

# run_task <network> <task> [docker run options...]: runs a task of
# run-task.sh in a new container. The results of the task, if it has any, go
# to dist/sandbox/<task>, which is emptied first.
run_task() {
  local network="$1" task="$2"
  shift 2
  local options=(
    --rm --init --network "$network"
    --volume "$REPO:/src:ro"
    --volume "$PUB_CACHE_VOLUME:/home/tester/.pub-cache"
    --env "SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH"
  )
  case "$task" in
    check | screenshots | package | fixtures)
      rm -rf "${RESULTS:?}/$task"
      mkdir -p "$RESULTS/$task"
      options+=(--volume "$RESULTS/$task:/out") ;;
  esac
  docker run "${options[@]}" ${OWNER[@]+"${OWNER[@]}"} "$@" "$IMAGE" "$task"
}

# Installs the .deb of the package task into a clean Ubuntu container, which
# downloads the dependencies the package declares.
install_test() {
  docker run --rm --init --network bridge \
    --volume "$SANDBOX_DIR/run-task.sh:/run-task.sh:ro" \
    --volume "$RESULTS/package:/pkg:ro" \
    --env "UBUNTU_MIRROR=${SANDBOX_UBUNTU_MIRROR:-}" \
    "$BASE_IMAGE" bash /run-task.sh install-test
}

case "${1:-}" in
  check | e2e | screenshots)
    ensure_image
    run_task bridge deps
    run_task none "$1"
    ;;
  package)
    ensure_image
    run_task bridge deps
    run_task none package
    install_test
    ;;
  all)
    ensure_image
    run_task bridge deps
    run_task none check
    run_task none e2e
    run_task none package
    install_test
    ;;
  fixtures)
    ensure_image
    run_task bridge fixtures
    echo "Review the changes with: diff -ru shared/fixtures/vfox-$VFOX_VERSION dist/sandbox/fixtures/vfox-$VFOX_VERSION"
    ;;
  shell)
    ensure_image
    run_task bridge deps
    run_task bridge shell --interactive --tty
    ;;
  image)
    build_image
    ;;
  clean)
    docker image ls --format '{{.Repository}}:{{.Tag}}' "$IMAGE_NAME" | xargs --no-run-if-empty docker image rm
    docker volume rm --force "$PUB_CACHE_VOLUME" > /dev/null
    rm -rf "${RESULTS:?}"
    ;;
  -h | --help | help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
