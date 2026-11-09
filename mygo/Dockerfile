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

# The vfox-plus-gateway image: the web interface of one vfox+, served from a
# container. It is a client of vfox+, as the Android app is, and not vfox+ in a
# container: it has no vfox, no window and no display, and shows the vfox+ that
# it is paired with (package gateway, docs/en/user/remote.md#the-gateway-docker).
#
# Build from the directory mygo/, which is the build context:
#
#   docker build -f mygo/Dockerfile -t vfox-plus-gateway mygo
#
# Run it with the port mapped. It prints a link that pairs a browser with it,
# over HTTPS with a certificate of its own (and the fingerprint of it, to compare
# with what the browser shows). On its page the browser is given the pairing
# link of a vfox+, from its settings, and the gateway is paired with that.
#
#   docker run --rm -p 8080:8080 vfox-plus-gateway
#
# A volume keeps what it remembers between runs: the vfox+ it is paired with,
# its session there, and its certificate. A certificate of your own goes there
# too, as cert.pem and key.pem.
#
#   docker run --rm -p 8080:8080 -v vfox-plus-gateway:/home/vfoxplus/.config/vfox-plus-gateway vfox-plus-gateway
#
# The program takes no arguments and reads no switches. To reach a vfox+ on the
# same computer through its address on the loopback interface (the "Open in
# browser" link, http://127.0.0.1:...), run the container with --network host;
# the link for other devices needs nothing of the kind.
#
# The toolchain defaults to mygo/versions.env; keep them in sync. Go runs on
# linux/amd64 here and builds for the platform of the image, so that one build
# serves both linux/amd64 and linux/arm64 (docker buildx build --platform ...).
# Where the archive of Ubuntu or the proxy of the Go modules is slow, say where
# else to get them from:
#
#   docker build --build-arg UBUNTU_MIRROR=http://mirrors.aliyun.com/ubuntu/ \
#     --build-arg GOPROXY=https://goproxy.cn,direct -f mygo/Dockerfile -t vfox-plus-gateway mygo

ARG GO_VERSION=1.27.3
ARG GO_SHA256=65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40

FROM --platform=linux/amd64 ubuntu:26.04 AS build
ARG GO_VERSION
ARG GO_SHA256
ARG TARGETARCH
# An Ubuntu archive mirror such as http://mirrors.aliyun.com/ubuntu/, and the
# proxy of the Go modules, if the defaults are slow from where the image is built.
ARG UBUNTU_MIRROR=
ARG GOPROXY

# Go, verified against the published checksum of the archive.
RUN <<EOF
set -eu
if [ -n "$UBUNTU_MIRROR" ]; then
  sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
    /etc/apt/sources.list.d/ubuntu.sources
fi
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl
rm -rf /var/lib/apt/lists/*
# Go publishes the same archive on go.dev and dl.google.com; take the other
# when one of them is slow, and let the checksum below judge the bytes.
curl -fsSL --retry 5 --retry-all-errors -o /tmp/go.tar.gz \
    "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz" ||
  curl -fsSL --retry 5 --retry-all-errors -o /tmp/go.tar.gz \
    "https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz"
echo "$GO_SHA256  /tmp/go.tar.gz" | sha256sum --check --strict -
tar -xzf /tmp/go.tar.gz -C /opt
rm /tmp/go.tar.gz
EOF
ENV PATH=/opt/go/bin:$PATH \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0

WORKDIR /src
COPY . /src
# Static, with nothing of the machine in it: the image is the program and the
# certificates of the authorities, and nothing else.
RUN GOOS=linux GOARCH="${TARGETARCH:-amd64}" go build -trimpath -ldflags "-s -w" -o /out/vfox-plus-gateway ./cmd/vfox-plus-gateway

# The directory that the gateway remembers in (HOME is /home/vfoxplus below),
# made here because the image has no shell to make it with.
RUN mkdir -p /out/home/vfoxplus/.config/vfox-plus-gateway

FROM scratch
LABEL org.opencontainers.image.title="vfox-plus-gateway" \
      org.opencontainers.image.description="The web interface of one vfox+, served from a container." \
      org.opencontainers.image.source="https://github.com/version-fox/vfox-plus" \
      org.opencontainers.image.licenses="Apache-2.0"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/vfox-plus-gateway /vfox-plus-gateway
COPY --from=build --chown=10001:10001 /out/home /home

USER 10001:10001
ENV HOME=/home/vfoxplus
VOLUME /home/vfoxplus/.config/vfox-plus-gateway
EXPOSE 8080
ENTRYPOINT ["/vfox-plus-gateway"]
