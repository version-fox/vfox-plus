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

# Captures the output of the vfox CLI into shared/fixtures/vfox-<version>/.
#
# Usage: shared/fixtures/capture.sh [path/to/vfox]
#
# Run it as a regular user: one scenario relies on file permissions.
#
# Every command runs against a throw-away HOME, so the vfox installation of the
# current user is never touched. The script needs network access: it adds the
# nodejs and maven plugins, searches their versions and installs Maven.
# After a run, compare the new outputs with cases.json and update the expected
# values there by hand. See README.md.
set -euo pipefail

VFOX="${1:-vfox}"
command -v "$VFOX" > /dev/null || { echo "vfox not found: $VFOX" >&2; exit 1; }

VERSION="$("$VFOX" --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+[^ ]*')"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="$SCRIPT_DIR/vfox-$VERSION"
SANDBOX="${VFOX_FIXTURE_SANDBOX:-/tmp/vfox-sb}"
MAVEN_VERSION="${MAVEN_VERSION:-3.9.16}"
INFO_FORMAT='{{.Name}}{{"\x1f"}}{{.Version}}{{"\x1f"}}{{.Homepage}}{{"\x1f"}}{{.InstallPath}}{{"\x1f"}}{{.Description}}'

# A fixed sandbox path keeps the captured paths stable between runs.
rm -rf "$SANDBOX"
mkdir -p "$SANDBOX/home" "$OUT"
export HOME="$SANDBOX/home" NO_COLOR=1
# VFOX_HOME would make vfox use (and change) a shared vfox directory instead.
unset VFOX_HOME __VFOX_SHELL __VFOX_PID __VFOX_CURTMPPATH __VFOX_INITIALIZED CI GITHUB_ACTIONS
cd /

# escape_controls <file...>: keeps the captures plain text, as README.md
# describes: a backslash becomes \\, and a control character other than tab and
# line feed becomes \xHH.
escape_controls() {
  perl -pi -e 's/\\/\\\\/g; s/([\x00-\x08\x0b-\x1f\x7f])/sprintf("\\x%02x", ord $1)/ge' "$@"
}

# capture <name> [VAR=value...] -- <vfox arguments...>
capture() {
  local name="$1" status=0
  shift
  local env_vars=()
  while [[ "$1" != "--" ]]; do env_vars+=("$1"); shift; done
  shift
  env ${env_vars[@]+"${env_vars[@]}"} "$VFOX" "$@" \
    > "$OUT/$name.stdout" 2> "$OUT/$name.stderr" < /dev/null || status=$?
  escape_controls "$OUT/$name.stdout" "$OUT/$name.stderr"
  printf '%-30s exit %s   vfox %s\n' "$name" "$status" "$*"
}

# retry <vfox arguments...>: runs a setup step, retrying on network errors.
retry() {
  local attempt
  for attempt in 1 2 3 4 5; do
    "$VFOX" "$@" > /dev/null 2>&1 < /dev/null && return 0
    echo "  vfox $* failed, retrying ($attempt)" >&2
    sleep 3
  done
  echo "vfox $* keeps failing" >&2
  return 1
}

# Fakes an installed version, so that the tree output contains several
# versions without downloading them.
fake_install() {
  mkdir -p "$HOME/.vfox/cache/$1/v-$2/$1-$2/bin"
}

capture version -- --version
capture current-empty -- current
capture list-empty -- list

for attempt in 1 2 3 4 5; do
  capture add -- add nodejs
  [[ -d "$HOME/.vfox/plugin/nodejs" ]] && break
  sleep 3
done
retry add maven
capture add-existing -- add maven
capture add-unknown -- add nosuchplugin
for attempt in 1 2 3 4 5; do
  capture add-alias -- add --alias mvn2 maven
  [[ -d "$HOME/.vfox/plugin/mvn2" ]] && break
  sleep 3
done

capture current-unset -- current
capture current-sdk-unset -- current maven
capture list-sdk-without-versions -- list maven

capture search-maven -- search maven
capture search-nodejs -- search nodejs
capture search-unknown -- search nosuchsdk
capture available -- available
capture info -- info nodejs --format "$INFO_FORMAT"
capture info-unknown -- info nosuchsdk

capture config-list -- config --list
capture config-set -- config proxy.url http://127.0.0.1:7890
capture config-get -- config proxy.url
"$VFOX" config --unset proxy.url
capture config-set-invalid-duration -- config cache.availableHookDuration notaduration

capture install-without-version -- install --yes maven
for attempt in 1 2 3 4 5; do
  capture install -- install --yes "maven@$MAVEN_VERSION"
  [[ -d "$HOME/.vfox/cache/maven/v-$MAVEN_VERSION" ]] && break
  echo "  download failed, retrying ($attempt)"
  sleep 3
done
capture search-maven-installed -- search maven

capture use-without-hook -- use --global "maven@$MAVEN_VERSION"
capture use-global __VFOX_SHELL=bash -- use --global "maven@$MAVEN_VERSION"

fake_install maven 3.8.9
fake_install nodejs 24.11.1
fake_install nodejs 22.21.0
__VFOX_SHELL=bash "$VFOX" use --global nodejs@24.11.1 > /dev/null
capture list -- list
capture current -- current
capture current-sdk -- current nodejs
capture list-sdk -- list nodejs
capture info-version-path -- info nodejs@24.11.1
capture info-version-not-installed -- info nodejs@1.0.0

capture remove-without-yes -- remove mvn2
capture remove -- remove --yes mvn2

capture unuse-global -- unuse --global maven
capture uninstall-not-installed -- uninstall maven@1.2.3
capture uninstall -- uninstall maven@3.8.9

capture update -- update maven
capture update-all -- update --all
capture update-unknown -- update nosuch

# A read-only install directory makes the installation fail deterministically.
# (Do not run this script as root, which ignores file permissions.)
rm -rf "$HOME/.vfox/cache/maven/v-$MAVEN_VERSION"
chmod a-w "$HOME/.vfox/cache/maven"
capture install-failure -- install --yes "maven@$MAVEN_VERSION"
chmod u+w "$HOME/.vfox/cache/maven"

echo "Fixtures written to $OUT"
