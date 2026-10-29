# syntax=docker/dockerfile:1
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

# The base of the sandbox images of the implementations (see README.md next to
# this file): Ubuntu 26.04 with a virtual display, fonts, the Debian packaging
# tools, the pinned Material Symbols font that the icon fonts are cut from, and
# a pinned vfox. The build context is the shared/ directory; the sandbox builds
# it with the vfox version of the implementation.
#
# Each step is its own layer, so that changing one input rebuilds only what
# comes after it.
FROM ubuntu:26.04

# An Ubuntu archive mirror such as http://mirrors.aliyun.com/ubuntu/, if the
# default archive is slow from where the image is built.
ARG UBUNTU_MIRROR=
ARG VFOX_DOWNLOAD_BASE=https://github.com/version-fox/vfox/releases/download
ARG VFOX_VERSION
ARG VFOX_SHA256

RUN <<EOF
set -eu
if [ -n "$UBUNTU_MIRROR" ]; then
  sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
    /etc/apt/sources.list.d/ubuntu.sources
fi
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
  ca-certificates curl git unzip xz-utils zip file jq patchelf binutils \
  dpkg-dev fakeroot lintian desktop-file-utils appstream \
  xvfb xauth x11-utils x11-apps imagemagick dbus-x11 at-spi2-core \
  fonts-ubuntu fonts-noto-core fonts-noto-cjk fonts-noto-color-emoji
rm -rf /var/lib/apt/lists/*
EOF

# vfox, verified against the checksum published with the release.
RUN <<EOF
set -eu
test -n "$VFOX_VERSION"
test -n "$VFOX_SHA256"
curl -fsSL --retry 5 --retry-all-errors -o /tmp/vfox.tar.gz \
  "$VFOX_DOWNLOAD_BASE/v$VFOX_VERSION/vfox_${VFOX_VERSION}_linux_x86_64.tar.gz"
echo "$VFOX_SHA256  /tmp/vfox.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/vfox.tar.gz -C /tmp
install -m 755 "/tmp/vfox_${VFOX_VERSION}_linux_x86_64/vfox" /usr/local/bin/vfox
rm -rf /tmp/vfox*
EOF

# The Material Symbols variable font that the icon fonts are cut from, so that
# builds without network access can generate them. The same pinned files that
# `vfxtool iconfont download` fetches, verified against the same checksums
# (see internal/iconfont/iconfont.go).
ENV VFOX_PLUS_SYMBOLS=/opt/vfox-plus/symbols
RUN <<EOF
set -eu
install -d /opt/vfox-plus/symbols
cd /opt/vfox-plus/symbols
base=https://raw.githubusercontent.com/google/material-design-icons/10ecc14790612ca8fd164e3f4ffe9c16fc50d5b7
font='MaterialSymbolsRounded[FILL,GRAD,opsz,wght]'
curl -fsSL --retry 5 --retry-all-errors --globoff -o "$font.ttf" "$base/variablefont/$font.ttf"
echo '7e74e36533ccee12c2b81e75bfcf3ccf123d3bb7e7e46d65d94445f23db0c97c  MaterialSymbolsRounded[FILL,GRAD,opsz,wght].ttf' | sha256sum --check --strict -
curl -fsSL --retry 5 --retry-all-errors --globoff -o "$font.codepoints" "$base/variablefont/$font.codepoints"
echo 'a7404d9e5a1b63f496cf05ace45d386075886b5a7f123fba9f29527cb6006fde  MaterialSymbolsRounded[FILL,GRAD,opsz,wght].codepoints' | sha256sum --check --strict -
curl -fsSL --retry 5 --retry-all-errors -o LICENSE "$base/LICENSE"
echo '58d1e17ffe5109a7ae296caafcadfdbe6a7d176f0bc4ab01e12a689b0499d8bd  LICENSE' | sha256sum --check --strict -
chmod -R a+rX /opt/vfox-plus
EOF

# Tests run as an unprivileged user. The implementation images add their
# toolchains on top.
RUN <<EOF
set -eu
useradd --create-home --shell /bin/bash tester
install -d -o tester -g tester /home/tester/work /home/tester/.cache
EOF
WORKDIR /home/tester
