# shellcheck shell=bash
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

# Host side of the Docker sandboxes: sourced by <implementation>/sandbox/sandbox.sh.
# See README.md next to this file.
#
# The sourcing script sets, before calling sandbox_main "$@":
#
#   IMPL            the directory name of the implementation, e.g. tauri
#   IMPL_DIR        its absolute path
#   REPO            the absolute path of the repository
#   VFOX_VERSION, VFOX_SHA256
#                   the vfox of the end-to-end tests (from its versions.env)
#   IMAGE_ARGS      docker build arguments of its image (an array)
#   CACHE_VOLUMES   NAME:PATH pairs of the volumes that keep downloaded
#                   dependencies between runs (an array)
#   PARENT_IMPLS    optional: implementations whose images this one builds on,
#                   in order (an array); their sandbox directories are built first
#   RUN_OPTIONS     optional: docker run options of the containers of every task
#                   and of the installation test (an array)
#
# and may override the steps of each command with these functions:
#
#   sandbox_check, sandbox_e2e, sandbox_package, sandbox_all

# shellcheck disable=SC2034 # The variables are used by the sourcing scripts.

: "${IMPL:?}" "${IMPL_DIR:?}" "${REPO:?}" "${VFOX_VERSION:?}" "${VFOX_SHA256:?}"

SHARED_SANDBOX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SANDBOX_DIR="$IMPL_DIR/sandbox"
RESULTS="$REPO/dist/sandbox/$IMPL"
BASE_IMAGE_NAME=vfox-plus-sandbox-base
IMAGE_NAME="vfox-plus-sandbox-$IMPL"
CLEAN_BASE_IMAGE=ubuntu:26.04
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git -C "$REPO" log -1 --format=%ct 2> /dev/null || date +%s)}"

# content_tag <files...>: a short hash of the files, so that an image is rebuilt
# exactly when its inputs change.
content_tag() {
  cat "$@" | sha256sum | cut -c1-12
}

base_image() {
  echo "$BASE_IMAGE_NAME:$(content_tag "$SHARED_SANDBOX_DIR/base.Dockerfile" "$REPO/shared/tools/icon_font.py")-vfox-$VFOX_VERSION"
}

# The image this implementation's image is built on: the image of the last of
# PARENT_IMPLS, or the base image.
parent_image() {
  if [[ ${#PARENT_IMPLS[@]} -gt 0 ]]; then
    "$REPO/${PARENT_IMPLS[-1]}/sandbox/sandbox.sh" image-name
  else
    base_image
  fi
}

# The image of the implementation, tagged by its Dockerfile, its pinned versions
# and the image it builds on, so that it is rebuilt when any of them changes.
impl_image() {
  local parent
  parent="$(parent_image)"
  echo "$IMAGE_NAME:$(echo "$parent" | cat "$SANDBOX_DIR/Dockerfile" "$IMPL_DIR/versions.env" - | sha256sum | cut -c1-12)"
}

mirror_args() {
  if [[ -n "${SANDBOX_UBUNTU_MIRROR:-}" ]]; then
    echo "--build-arg=UBUNTU_MIRROR=$SANDBOX_UBUNTU_MIRROR"
  fi
}

build_base_image() {
  local image
  image="$(base_image)"
  docker image inspect "$image" > /dev/null 2>&1 && return 0
  # shellcheck disable=SC2046 # mirror_args prints zero or one argument.
  docker build --tag "$image" --file "$SHARED_SANDBOX_DIR/base.Dockerfile" \
    --build-arg "VFOX_VERSION=$VFOX_VERSION" --build-arg "VFOX_SHA256=$VFOX_SHA256" \
    $(mirror_args) "$REPO/shared"
}

ensure_image() {
  build_base_image
  local impl
  for impl in ${PARENT_IMPLS[@]+"${PARENT_IMPLS[@]}"}; do
    # The parent's own sandbox.sh knows its build arguments.
    "$REPO/$impl/sandbox/sandbox.sh" image
  done
  local image
  image="$(impl_image)"
  docker image inspect "$image" > /dev/null 2>&1 && return 0
  # shellcheck disable=SC2046 # mirror_args prints zero or one argument.
  docker build --tag "$image" --build-arg "BASE_IMAGE=$(parent_image)" $(mirror_args) \
    ${IMAGE_ARGS[@]+"${IMAGE_ARGS[@]}"} "$SANDBOX_DIR"
}

# Under rootless Docker, root in a container is the user on the host. A rootful
# daemon needs to be told whom the results belong to.
owner_args() {
  if ! docker info --format '{{.SecurityOptions}}' 2> /dev/null | grep -q rootless; then
    echo "--env=OUT_OWNER=$(id -u):$(id -g)"
  fi
}

# run_task <network> <task> [docker run options...]: runs a task of the
# implementation's run-task.sh in a new container that sees the repository
# read-only. The results of the task, if it has any, go to
# dist/sandbox/<implementation>/<task>, which is emptied first.
run_task() {
  local network="$1" task="$2"
  shift 2
  local options=(--rm --init --network "$network" --volume "$REPO:/src:ro" --env "SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH")
  # The mirror settings (SANDBOX_NPM_REGISTRY, SANDBOX_GOPROXY, …) reach the tasks.
  local name
  for name in $(compgen -e); do
    if [[ "$name" == SANDBOX_* ]]; then
      options+=(--env "$name")
    fi
  done
  local volume
  for volume in ${CACHE_VOLUMES[@]+"${CACHE_VOLUMES[@]}"}; do
    options+=(--volume "$volume")
  done
  rm -rf "${RESULTS:?}/$task"
  mkdir -p "$RESULTS/$task"
  options+=(--volume "$RESULTS/$task:/out")
  # shellcheck disable=SC2046 # owner_args prints zero or one argument.
  docker run "${options[@]}" ${RUN_OPTIONS[@]+"${RUN_OPTIONS[@]}"} $(owner_args) "$@" "$(impl_image)" \
    bash "/src/$IMPL/sandbox/run-task.sh" "$task"
  # Keep only the result directories that received something.
  rmdir "$RESULTS/$task" 2> /dev/null || true
}

# Installs the .deb of the package task into a clean Ubuntu container, which
# downloads the dependencies the package declares, and starts the application.
install_test() {
  docker run --rm --init --network bridge \
    --volume "$SHARED_SANDBOX_DIR/task-lib.sh:/task-lib.sh:ro" \
    --volume "$IMPL_DIR/sandbox/run-task.sh:/run-task.sh:ro" \
    --volume "$RESULTS/package:/pkg:ro" \
    --env "UBUNTU_MIRROR=${SANDBOX_UBUNTU_MIRROR:-}" \
    --env "TASK_LIB=/task-lib.sh" \
    ${RUN_OPTIONS[@]+"${RUN_OPTIONS[@]}"} "$CLEAN_BASE_IMAGE" bash /run-task.sh install-test
}

sandbox_check() {
  run_task bridge deps
  run_task none check
}

sandbox_e2e() {
  run_task bridge deps
  run_task none e2e
}

sandbox_package() {
  run_task bridge deps
  run_task none package
  install_test
}

sandbox_all() {
  run_task bridge deps
  run_task none check
  run_task none e2e
  run_task none package
  install_test
}

sandbox_usage() {
  sed -n '/^# Usage:/,/^set -euo/{/^set -euo/d; s/^# \{0,1\}//; p}' "$SANDBOX_DIR/sandbox.sh"
}

sandbox_main() {
  case "${1:-}" in
    check)
      ensure_image
      sandbox_check
      ;;
    e2e)
      ensure_image
      sandbox_e2e
      ;;
    package)
      ensure_image
      sandbox_package
      ;;
    all)
      ensure_image
      sandbox_all
      ;;
    measure | screenshots)
      ensure_image
      run_task bridge deps
      run_task none "$1"
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
