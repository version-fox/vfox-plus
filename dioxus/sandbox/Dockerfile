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

# The sandbox of the Dioxus implementation: the Tauri sandbox (Node.js, the
# WebKitGTK development files and the pinned Rust, see tauri/sandbox) with
# libxdo, which the menus of Dioxus link, and the coverage tools. Build and use
# it through sandbox.sh, which passes the versions from ../versions.env.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=
ARG CARGO_LLVM_COV_VERSION
ARG CARGO_LLVM_COV_SHA256

RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends libxdo-dev \
 && rm -rf /var/lib/apt/lists/*

# cargo-llvm-cov, verified against its checksum, and the LLVM tools of the
# pinned Rust, which it uses.
RUN test -n "$CARGO_LLVM_COV_VERSION" && test -n "$CARGO_LLVM_COV_SHA256" \
 && curl -fsSL --retry 5 --retry-all-errors -o /tmp/cargo-llvm-cov.tar.gz \
      "https://github.com/taiki-e/cargo-llvm-cov/releases/download/v$CARGO_LLVM_COV_VERSION/cargo-llvm-cov-x86_64-unknown-linux-gnu.tar.gz" \
 && echo "$CARGO_LLVM_COV_SHA256  /tmp/cargo-llvm-cov.tar.gz" | sha256sum --check --strict - \
 && tar -xzf /tmp/cargo-llvm-cov.tar.gz -C /usr/local/bin \
 && rm /tmp/cargo-llvm-cov.tar.gz \
 && runuser -u tester -- env RUSTUP_HOME=/opt/rustup CARGO_HOME=/home/tester/.cargo RUSTUP_MAX_RETRIES=10 \
      /home/tester/.cargo/bin/rustup component add llvm-tools-preview \
 && chmod -R a+rX /opt/rustup
