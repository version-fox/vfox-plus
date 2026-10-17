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

# Runs one task inside a sandbox container of the web front end. sandbox.sh
# starts the containers; do not run this script on the host.
#
# Usage: run-task.sh <task>
#
# The steps mirror the jobs of .github/workflows/web.yml; keep them in sync.
set -euo pipefail

# shellcheck source=SCRIPTDIR/../../shared/sandbox/task-lib.sh
. "${TASK_LIB:-/src/shared/sandbox/task-lib.sh}"
APP="$WORK/web"

# npm_install [--offline]: installs the packages of package-lock.json. The deps
# task downloads them into the npm cache volume; the other tasks run without
# network and take them from there.
npm_install() {
  as_tester "$APP" npm ci --no-audit --no-fund "$@" ${SANDBOX_NPM_REGISTRY:+--registry="$SANDBOX_NPM_REGISTRY"}
}

prepare() {
  copy_repository
  npm_install --offline
  as_tester "$APP" npm run assets
}

case "${1:-}" in
  deps)
    copy_repository
    npm_install
    ;;

  check)
    prepare
    as_tester "$APP" npm run format:check
    as_tester "$APP" npm run lint
    as_tester "$APP" npm run typecheck
    as_tester "$APP" npm run test:coverage
    as_tester "$APP" npm run build
    publish "$APP/coverage/lcov.info"
    ;;

  dev)
    prepare
    as_tester "$APP" npm run dev -- --host 0.0.0.0
    ;;

  shell)
    prepare
    cd "$APP"
    exec runuser -u tester -- env HOME=/home/tester bash
    ;;

  *)
    echo "Unknown task: ${1:-}. Use web/sandbox/sandbox.sh." >&2
    exit 2
    ;;
esac
