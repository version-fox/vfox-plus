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

# A disposable Ubuntu 26.04 environment with everything needed to build, test
# and package the Flutter implementation: the Linux toolchain, a virtual
# display, fonts, a pinned Flutter SDK and a pinned vfox. Build and use it
# through sandbox.sh, which passes the versions from ../versions.env.
FROM ubuntu:26.04

# An Ubuntu archive mirror such as http://mirrors.aliyun.com/ubuntu/, if the
# default archive is slow from where the image is built.
ARG UBUNTU_MIRROR=
ARG FLUTTER_STORAGE_BASE_URL=https://storage.googleapis.com
ARG FLUTTER_VERSION
ARG FLUTTER_SHA256
ARG VFOX_DOWNLOAD_BASE=https://github.com/version-fox/vfox/releases/download
ARG VFOX_VERSION
ARG VFOX_SHA256

RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates curl git unzip xz-utils zip file \
      clang cmake ninja-build pkg-config libgtk-3-dev liblzma-dev libstdc++-15-dev \
      patchelf binutils dpkg-dev lintian desktop-file-utils \
      xvfb xauth fonts-ubuntu fonts-noto-core fonts-noto-cjk \
 && rm -rf /var/lib/apt/lists/*

# vfox, verified against the checksum published with the release.
RUN test -n "$VFOX_VERSION" && test -n "$VFOX_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/vfox.tar.gz \
      "$VFOX_DOWNLOAD_BASE/v$VFOX_VERSION/vfox_${VFOX_VERSION}_linux_x86_64.tar.gz" \
 && echo "$VFOX_SHA256  /tmp/vfox.tar.gz" | sha256sum --check --strict - \
 && tar -xzf /tmp/vfox.tar.gz -C /tmp \
 && install -m 755 "/tmp/vfox_${VFOX_VERSION}_linux_x86_64/vfox" /usr/local/bin/vfox \
 && rm -rf /tmp/vfox*

# Tests run as an unprivileged user, who also owns the Flutter SDK because
# flutter writes to its cache. The pub cache is a named volume.
RUN useradd --create-home --shell /bin/bash tester \
 && install -d -o tester -g tester /opt/flutter /home/tester/.pub-cache /home/tester/work

USER tester
# The Flutter SDK, verified against the checksum of the release archive.
# Extracting it as tester avoids a second copy in a chown layer.
RUN test -n "$FLUTTER_VERSION" && test -n "$FLUTTER_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/flutter.tar.xz \
      "$FLUTTER_STORAGE_BASE_URL/flutter_infra_release/releases/stable/linux/flutter_linux_${FLUTTER_VERSION}-stable.tar.xz" \
 && echo "$FLUTTER_SHA256  /tmp/flutter.tar.xz" | sha256sum --check --strict - \
 && tar -xJf /tmp/flutter.tar.xz -C /opt \
 && rm /tmp/flutter.tar.xz

ENV PATH=/opt/flutter/bin:/opt/flutter/bin/cache/dart-sdk/bin:$PATH \
    FLUTTER_STORAGE_BASE_URL=$FLUTTER_STORAGE_BASE_URL \
    PUB_CACHE=/home/tester/.pub-cache \
    FLUTTER_SUPPRESS_ANALYTICS=true
RUN git config --global --add safe.directory '*' \
 && flutter --suppress-analytics config --no-analytics --no-cli-animations \
 && dart --disable-analytics \
 && flutter --suppress-analytics precache --linux --no-android --no-ios --no-web \
 && flutter --suppress-analytics --version

# sandbox.sh mounts the repository read-only at /src; the task runner is taken
# from there, so the image only changes when the pinned versions change.
USER root
WORKDIR /home/tester
ENTRYPOINT ["bash", "/src/flutter/sandbox/run-task.sh"]
