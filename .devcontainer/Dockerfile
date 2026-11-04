# syntax=docker/dockerfile:1
# check=skip=FromPlatformFlagConstDisallowed
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

# The development container (docs/en/dev/environment.md#the-dev-container):
# Ubuntu 26.04 with what CI installs to build, test and package the application,
# the pinned vfox of the end-to-end tests against the real vfox, and what the
# Android client needs to build (a JDK and the Android SDK, but no emulator).
#
# Every version comes from mygo/versions.env or android/versions.env, which the
# steps below read, so the container moves with CI and the sandbox. The build
# context is the root of the repository, which Dockerfile.dockerignore reduces
# to those two files. Each step is its own layer, so that changing one input
# rebuilds only what comes after it.
#
# What the user changes is under /home/vscode (the module cache of Go, the
# home of Gradle); the tools are not the user's to change, so a change of the
# user's uid (updateRemoteUserUID) has only the home to chown.

# x86-64 only, as the archives that the versions files pin are. On another host,
# Docker runs the image under emulation.
FROM --platform=linux/amd64 ubuntu:26.04

# An Ubuntu archive mirror such as http://mirrors.aliyun.com/ubuntu/, if the
# default archive is slow from where the image is built (the build argument
# that devcontainer.json fills from SANDBOX_UBUNTU_MIRROR).
ARG UBUNTU_MIRROR=

# The packages of LINUX_PACKAGES in .github/workflows/mygo.yml (a C compiler
# for the race detector, GTK 3, the library of the tray icon and the fonts that
# the application loads, a virtual display, lintian and desktop-file-utils for
# the .deb), and the tools
# of a shell that works in a repository.
RUN <<EOF
set -eu
if [ -n "$UBUNTU_MIRROR" ]; then
  sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
    /etc/apt/sources.list.d/ubuntu.sources
fi
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  build-essential libgtk-3-0t64 libayatana-appindicator3-1 fontconfig fonts-ubuntu fonts-noto-cjk \
  xvfb xauth dbus-x11 lintian desktop-file-utils \
  ca-certificates curl file git jq less nano openssh-client sudo unzip xz-utils zip
rm -rf /var/lib/apt/lists/*
EOF

# The user. The image of Ubuntu has a user "ubuntu" at 1000, which this one
# replaces. The directories are where the volumes of devcontainer.json mount.
RUN <<EOF
set -eu
if id -u ubuntu > /dev/null 2>&1; then
  userdel --remove ubuntu
fi
groupadd --gid 1000 vscode
useradd --uid 1000 --gid 1000 --create-home --shell /bin/bash vscode
echo "vscode ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/vscode
chmod 0440 /etc/sudoers.d/vscode
install -d -o vscode -g vscode /home/vscode/go /home/vscode/.gradle
EOF

# The Android SDK. The JDK is the one of JAVA_VERSION (CI uses Temurin, another
# build of OpenJDK; the Ubuntu archive has this one, with its updates), the
# command-line tools are verified against the checksum that android/versions.env
# pins, and the packages are the ones that the build uses, so that the first build
# downloads no package of the SDK. The SDK belongs to root: the build only reads it.
ENV ANDROID_HOME=/opt/android-sdk
RUN --mount=type=bind,source=android/versions.env,target=/tmp/android.env <<EOF
set -eu
. /tmp/android.env
test -n "$JAVA_VERSION"
test -n "$ANDROID_CMDLINE_TOOLS_BUILD"
test -n "$ANDROID_CMDLINE_TOOLS_SHA256"
test -n "$ANDROID_SDK_PLATFORM"
test -n "$ANDROID_SDK_BUILD_TOOLS"
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "openjdk-$JAVA_VERSION-jdk-headless"
rm -rf /var/lib/apt/lists/*
curl -fsSL --retry 5 --retry-all-errors -o /tmp/cmdline-tools.zip \
  "https://dl.google.com/android/repository/commandlinetools-linux-${ANDROID_CMDLINE_TOOLS_BUILD}_latest.zip"
echo "$ANDROID_CMDLINE_TOOLS_SHA256  /tmp/cmdline-tools.zip" | sha256sum --check --strict -
mkdir -p "$ANDROID_HOME/cmdline-tools"
unzip -q /tmp/cmdline-tools.zip -d "$ANDROID_HOME/cmdline-tools"
mv "$ANDROID_HOME/cmdline-tools/cmdline-tools" "$ANDROID_HOME/cmdline-tools/latest"
rm /tmp/cmdline-tools.zip
sdkmanager="$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager"
yes | "$sdkmanager" --licenses > /dev/null
"$sdkmanager" "platform-tools" "platforms;android-$ANDROID_SDK_PLATFORM" "build-tools;$ANDROID_SDK_BUILD_TOOLS"
EOF

# Go, verified against the published checksum.
RUN --mount=type=bind,source=mygo/versions.env,target=/tmp/mygo.env <<EOF
set -eu
. /tmp/mygo.env
test -n "$GO_VERSION"
test -n "$GO_SHA256"
curl -fsSL --retry 5 --retry-all-errors -o /tmp/go.tar.gz "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz"
echo "$GO_SHA256  /tmp/go.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/go.tar.gz -C /usr/local
rm /tmp/go.tar.gz
EOF

# golangci-lint, verified against the published checksum.
RUN --mount=type=bind,source=mygo/versions.env,target=/tmp/mygo.env <<EOF
set -eu
. /tmp/mygo.env
test -n "$GOLANGCI_LINT_VERSION"
test -n "$GOLANGCI_LINT_SHA256"
curl -fsSL --retry 5 --retry-all-errors -o /tmp/golangci-lint.tar.gz \
  "https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_LINT_VERSION/golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64.tar.gz"
echo "$GOLANGCI_LINT_SHA256  /tmp/golangci-lint.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/golangci-lint.tar.gz -C /tmp
install -m 755 "/tmp/golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64/golangci-lint" /usr/local/bin/golangci-lint
rm -rf /tmp/golangci-lint*
EOF

# vfox, verified against the checksum published with the release: the one that
# the end-to-end tests against the real vfox run (VFOX_PLUS_E2E_VFOX), where it
# has no ~/.vfox of yours to touch.
RUN --mount=type=bind,source=mygo/versions.env,target=/tmp/mygo.env <<EOF
set -eu
. /tmp/mygo.env
test -n "$VFOX_VERSION"
test -n "$VFOX_SHA256"
curl -fsSL --retry 5 --retry-all-errors -o /tmp/vfox.tar.gz \
  "https://github.com/version-fox/vfox/releases/download/v$VFOX_VERSION/vfox_${VFOX_VERSION}_linux_x86_64.tar.gz"
echo "$VFOX_SHA256  /tmp/vfox.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/vfox.tar.gz -C /tmp
install -m 755 "/tmp/vfox_${VFOX_VERSION}_linux_x86_64/vfox" /usr/local/bin/vfox
rm -rf /tmp/vfox*
EOF

# GOTOOLCHAIN=local: never download another Go than the pinned one, as in the
# sandbox. The module cache and the build cache of Go share one volume.
ENV LANG=C.UTF-8 \
    GOPATH=/home/vscode/go \
    GOCACHE=/home/vscode/go/build-cache \
    GOTOOLCHAIN=local \
    GRADLE_USER_HOME=/home/vscode/.gradle \
    VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox \
    PATH=/usr/local/go/bin:/home/vscode/go/bin:/opt/android-sdk/cmdline-tools/latest/bin:/opt/android-sdk/platform-tools:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

USER vscode
WORKDIR /home/vscode
