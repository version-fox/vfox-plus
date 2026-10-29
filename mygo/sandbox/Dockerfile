# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
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

# The sandbox of the mygo implementation: the shared base image (see
# shared/sandbox) with GTK 3, which mygo loads at run time (the application needs
# no development files and no C compiler), a pinned Go and golangci-lint, and a
# window manager with the Greybird theme, so that the screenshots show a title
# bar the way modern GNOME applications look (a dark bar over light content),
# over the default wallpaper of Ubuntu instead of the black of a bare X server.
# The C compiler is only there for the race detector of the unit tests. Build
# and use it through `vfxtool sandbox`, which passes the versions from ../versions.env.
#
# Each step is its own layer, so that changing one input rebuilds only what
# comes after it.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=
ARG GO_VERSION
ARG GO_SHA256
ARG GOLANGCI_LINT_VERSION
ARG GOLANGCI_LINT_SHA256

RUN <<EOF
set -eu
if [ -n "$UBUNTU_MIRROR" ]; then
  sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
    /etc/apt/sources.list.d/ubuntu.sources
fi
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  libgtk-3-0t64 fontconfig libgl1-mesa-dri libegl1 libgles2 \
  gcc libc6-dev xfwm4 xfconf greybird-gtk-theme \
  feh ubuntu-wallpapers
rm -rf /var/lib/apt/lists/*
EOF

# Go, verified against the published checksum.
RUN <<EOF
set -eu
test -n "$GO_VERSION"
test -n "$GO_SHA256"
curl -fsSL --retry 5 --retry-all-errors -o /tmp/go.tar.gz "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz"
echo "$GO_SHA256  /tmp/go.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/go.tar.gz -C /opt
rm /tmp/go.tar.gz
install -d -o tester -g tester /home/tester/go /home/tester/go/pkg/mod /home/tester/.cache/go-build /home/tester/.cache/vfox-plus
EOF

# golangci-lint, verified against the published checksum.
RUN <<EOF
set -eu
test -n "$GOLANGCI_LINT_VERSION"
test -n "$GOLANGCI_LINT_SHA256"
curl -fsSL --retry 5 --retry-all-errors -o /tmp/golangci-lint.tar.gz \
  "https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_LINT_VERSION/golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64.tar.gz"
echo "$GOLANGCI_LINT_SHA256  /tmp/golangci-lint.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/golangci-lint.tar.gz -C /tmp
install -m 755 "/tmp/golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64/golangci-lint" /usr/local/bin/golangci-lint
rm -rf /tmp/golangci-lint*
EOF

# GOTOOLCHAIN=local: never download another Go than the pinned one.
ENV PATH=/opt/go/bin:/home/tester/go/bin:$PATH \
    GOPATH=/home/tester/go \
    GOTOOLCHAIN=local
