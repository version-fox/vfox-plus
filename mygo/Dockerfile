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

# The published vfox-plus image: a minimal Ubuntu runtime with vfox and a
# `vfox-plus serve` that browsers pair with. It needs no display server: the
# pairing server is plain Go without a UI toolkit.
#
# Build from the repository root (the build reads mygo/ and shared/):
#
#   docker build -f mygo/Dockerfile -t vfox-plus .
#
# Run it with the port mapped, then open the pairing link it prints, with
# 0.0.0.0 replaced by the reachable address. The link is https: a
# self-signed certificate is generated in the container on first start,
# and its fingerprint is printed for comparison in the browser.
#
#   docker run --rm -p 8080:8080 vfox-plus
#
# A browser on another machine pairs the same way, over the network the
# container is reachable on. Prefer an encrypted tunnel outside trusted
# networks. A data volume keeps the installed SDKs between runs:
#
#   docker run --rm -p 8080:8080 -v vfox-plus-data:/home/vfoxplus vfox-plus
#
# The toolchain versions default to mygo/versions.env; keep them in sync.
# Only linux/amd64 is built here: vfox publishes per-architecture archives,
# and the URL below names the x86_64 one.

ARG GO_VERSION=1.27.3
ARG GO_SHA256=65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40
ARG VFOX_VERSION=1.0.13
ARG VFOX_SHA256=7ad0a4c82c8173f09393e09abecc3e27468253bab5fe53ffbf58f2b12b3fc768

FROM ubuntu:26.04 AS build
ARG GO_VERSION
ARG GO_SHA256
ARG VFOX_VERSION
ARG VFOX_SHA256

# Go, verified against the published checksum.
RUN <<EOF
set -eu
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl
rm -rf /var/lib/apt/lists/*
curl -fsSL --retry 5 -o /tmp/go.tar.gz "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz"
echo "$GO_SHA256  /tmp/go.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/go.tar.gz -C /opt
rm /tmp/go.tar.gz
EOF
ENV PATH=/opt/go/bin:$PATH \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0

# vfox, verified against the checksum published with the release.
RUN <<EOF
set -eu
mkdir -p /out
curl -fsSL --retry 5 -o /tmp/vfox.tar.gz \
  "https://github.com/version-fox/vfox/releases/download/v$VFOX_VERSION/vfox_${VFOX_VERSION}_linux_x86_64.tar.gz"
echo "$VFOX_SHA256  /tmp/vfox.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/vfox.tar.gz -C /tmp
install -m 755 "/tmp/vfox_${VFOX_VERSION}_linux_x86_64/vfox" /out/vfox
rm -rf /tmp/vfox*
EOF

WORKDIR /src
COPY mygo/go.mod mygo/go.sum /src/mygo/
COPY shared/tooling/go.mod shared/tooling/go.sum /src/shared/tooling/
RUN cd /src/mygo && go mod download
RUN cd /src/shared/tooling && go mod download
COPY mygo /src/mygo
COPY shared /src/shared
RUN go -C /src/shared/tooling build -o /tmp/vfxtool ./cmd/vfxtool \
  && /tmp/vfxtool build --root /src/mygo --package ./cmd/vfox-plus-serve /out/vfox-plus-serve \
  && /out/vfox-plus-serve serve --help

FROM ubuntu:26.04
RUN <<EOF
set -eu
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates
rm -rf /var/lib/apt/lists/*
useradd --create-home --shell /bin/bash vfoxplus
EOF
COPY --from=build /out/vfox /usr/local/bin/vfox
COPY --from=build /out/vfox-plus-serve /usr/local/bin/vfox-plus-serve

USER vfoxplus
WORKDIR /home/vfoxplus
ENV HOME=/home/vfoxplus
VOLUME /home/vfoxplus/.vfox
VOLUME /home/vfoxplus/.version-fox
EXPOSE 8080
# The image exists for remote access, so it opts in explicitly: remote access
# stays off by default and `serve` refuses non-loopback binds without it.
ENTRYPOINT ["/usr/local/bin/vfox-plus-serve"]
CMD ["serve", "--bind", "0.0.0.0", "--port", "8080", "--allow-remote"]
