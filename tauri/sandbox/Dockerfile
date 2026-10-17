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

# The sandbox of the Tauri implementation: the web sandbox (Node.js, see
# web/sandbox) with the WebKitGTK development files and a pinned Rust. Build
# and use it through sandbox.sh, which passes the versions from
# ../versions.env.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=
ARG RUST_VERSION
ARG RUSTUP_VERSION
ARG RUSTUP_SHA256

RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      build-essential pkg-config libwebkit2gtk-4.1-dev libgtk-3-dev librsvg2-dev libssl-dev \
 && rm -rf /var/lib/apt/lists/*

# Rust, verified against the checksum of rustup-init, for the tester user.
ENV RUSTUP_HOME=/opt/rustup CARGO_HOME=/home/tester/.cargo PATH=/home/tester/.cargo/bin:$PATH
RUN test -n "$RUST_VERSION" && test -n "$RUSTUP_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/rustup-init \
      "https://static.rust-lang.org/rustup/archive/$RUSTUP_VERSION/x86_64-unknown-linux-gnu/rustup-init" \
 && echo "$RUSTUP_SHA256  /tmp/rustup-init" | sha256sum --check --strict - \
 && chmod +x /tmp/rustup-init \
 && install -d -o tester -g tester /opt/rustup /home/tester/.cargo \
 && runuser -u tester -- env RUSTUP_HOME=/opt/rustup CARGO_HOME=/home/tester/.cargo \
      /tmp/rustup-init -y --no-modify-path --profile minimal --default-toolchain "$RUST_VERSION" \
      --component rustfmt --component clippy \
 && rm /tmp/rustup-init \
 && chmod -R a+rX /opt/rustup \
 && install -d -o tester -g tester /home/tester/.cargo/registry
