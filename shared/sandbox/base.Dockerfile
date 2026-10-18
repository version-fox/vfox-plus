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
# tools, the icon font tooling and a pinned vfox. The build context is the
# shared/ directory; sandbox-lib.sh builds it with the vfox version of the
# implementation.
FROM ubuntu:26.04

# An Ubuntu archive mirror such as http://mirrors.aliyun.com/ubuntu/, if the
# default archive is slow from where the image is built.
ARG UBUNTU_MIRROR=
ARG VFOX_DOWNLOAD_BASE=https://github.com/version-fox/vfox/releases/download
ARG VFOX_VERSION
ARG VFOX_SHA256

RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates curl git unzip xz-utils zip file jq patchelf binutils \
      dpkg-dev fakeroot lintian desktop-file-utils appstream \
      xvfb xauth x11-utils x11-apps imagemagick dbus-x11 at-spi2-core \
      fonts-ubuntu fonts-noto-core fonts-noto-cjk fonts-noto-color-emoji \
      fonttools python3-brotli \
 && rm -rf /var/lib/apt/lists/*

# vfox, verified against the checksum published with the release.
RUN test -n "$VFOX_VERSION" && test -n "$VFOX_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/vfox.tar.gz \
      "$VFOX_DOWNLOAD_BASE/v$VFOX_VERSION/vfox_${VFOX_VERSION}_linux_x86_64.tar.gz" \
 && echo "$VFOX_SHA256  /tmp/vfox.tar.gz" | sha256sum --check --strict - \
 && tar -xzf /tmp/vfox.tar.gz -C /tmp \
 && install -m 755 "/tmp/vfox_${VFOX_VERSION}_linux_x86_64/vfox" /usr/local/bin/vfox \
 && rm -rf /tmp/vfox*

# The Material Symbols variable font that the icon fonts are cut from, so that
# builds without network access can generate them.
COPY tools/icon_font.py /opt/vfox-plus/icon_font.py
RUN python3 /opt/vfox-plus/icon_font.py download /opt/vfox-plus/symbols \
 && chmod -R a+rX /opt/vfox-plus
ENV VFOX_PLUS_SYMBOLS=/opt/vfox-plus/symbols

# Tests run as an unprivileged user. The implementation images add their
# toolchains on top.
RUN useradd --create-home --shell /bin/bash tester \
 && install -d -o tester -g tester /home/tester/work /home/tester/.cache
WORKDIR /home/tester
