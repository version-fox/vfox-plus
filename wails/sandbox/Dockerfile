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

# The sandbox of the Wails implementation: the web sandbox (Node.js, see
# web/sandbox) with the GTK 4 and WebKitGTK 6.0 development files, a pinned Go
# and golangci-lint. Build and use it through sandbox.sh, which passes the
# versions from ../versions.env.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=
ARG GO_VERSION
ARG GO_SHA256
ARG GOLANGCI_LINT_VERSION
ARG GOLANGCI_LINT_SHA256

RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev \
 && rm -rf /var/lib/apt/lists/*

# Go and golangci-lint, verified against the published checksums.
RUN test -n "$GO_VERSION" && test -n "$GO_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/go.tar.gz "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz" \
 && echo "$GO_SHA256  /tmp/go.tar.gz" | sha256sum --check --strict - \
 && tar -xzf /tmp/go.tar.gz -C /opt \
 && rm /tmp/go.tar.gz \
 && test -n "$GOLANGCI_LINT_VERSION" && test -n "$GOLANGCI_LINT_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/golangci-lint.tar.gz \
      "https://github.com/golangci/golangci-lint/releases/download/v$GOLANGCI_LINT_VERSION/golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64.tar.gz" \
 && echo "$GOLANGCI_LINT_SHA256  /tmp/golangci-lint.tar.gz" | sha256sum --check --strict - \
 && tar -xzf /tmp/golangci-lint.tar.gz -C /tmp \
 && install -m 755 "/tmp/golangci-lint-$GOLANGCI_LINT_VERSION-linux-amd64/golangci-lint" /usr/local/bin/golangci-lint \
 && rm -rf /tmp/golangci-lint* \
 && install -d -o tester -g tester /home/tester/go /home/tester/go/pkg/mod /home/tester/.cache/go-build

# GOTOOLCHAIN=local: never download another Go than the pinned one.
ENV PATH=/opt/go/bin:/home/tester/go/bin:$PATH \
    GOPATH=/home/tester/go \
    GOTOOLCHAIN=local
