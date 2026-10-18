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

# The sandbox of the Flet implementation: the shared base image (see
# shared/sandbox) with the pinned uv, the pinned Flutter SDK that `flet
# build` needs, and the build dependencies of a Flutter Linux application.
# Node.js is only the test server of the end-to-end tests against the real
# vfox. Build and use it through sandbox.sh, which passes the versions from
# ../versions.env.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=
ARG UV_VERSION
ARG UV_SHA256
ARG FLUTTER_VERSION
ARG FLUTTER_SHA256

# Build tools of a Flutter Linux application (the same as the development
# setup), rsvg-convert for the application icon, and Node.js for the test
# server of the end-to-end tests.
RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      clang cmake ninja-build pkg-config libgtk-3-dev lld llvm binutils libsecret-1-dev \
      librsvg2-bin nodejs \
 && rm -rf /var/lib/apt/lists/*

# Downloads retry until the checksum matches: the downloads are large and the
# connection may corrupt them, which curl alone does not retry. The archives
# come from `.downloads/` of the build context instead: sandbox.sh downloads
# them on the host, whose connection is reliable, and the Dockerfile only
# verifies and extracts them.
#
# uv, verified against the checksum published with the release.
COPY .downloads/uv.tar.gz /tmp/uv.tar.gz
RUN test -n "$UV_VERSION" && test -n "$UV_SHA256" \
 && echo "$UV_SHA256  /tmp/uv.tar.gz" | sha256sum --check --strict - \
 && tar -xzf /tmp/uv.tar.gz -C /usr/local/bin --strip-components=1 \
 && rm /tmp/uv.tar.gz

# Flutter, verified against the checksum published with the release.
COPY .downloads/flutter.tar.xz /tmp/flutter.tar.xz
RUN test -n "$FLUTTER_VERSION" && test -n "$FLUTTER_SHA256" \
 && echo "$FLUTTER_SHA256  /tmp/flutter.tar.xz" | sha256sum --check --strict - \
 && tar -xJf /tmp/flutter.tar.xz -C /opt \
 && rm /tmp/flutter.tar.xz

# The tester owns the Flutter SDK because Flutter writes its engine stamp
# and cache into it; the pub cache is a named volume.
RUN git config --system --add safe.directory /opt/flutter \
 && chown -R tester:tester /opt/flutter \
 && install -d -o tester -g tester /home/tester/.cache/uv /home/tester/.flet /home/tester/.pub-cache \
 && install -d -o tester -g tester /home/tester/work /home/tester/work/flet /home/tester/work/flet/build

ENV PATH=/opt/flutter/bin:/usr/local/bin:$PATH \
    PUB_CACHE=/home/tester/.pub-cache

# The tester warms up the Flutter tool: its version check, analytics settings
# and engine artifacts, so that task containers without network do not try
# to reach the network on every invocation.
USER tester
RUN flutter config --no-analytics --no-cli-animations \
 && flutter --suppress-analytics --version \
 && flutter --suppress-analytics precache --linux --no-android --no-ios --no-web --no-windows --no-macos
USER root
WORKDIR /home/tester
