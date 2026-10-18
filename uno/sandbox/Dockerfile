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

# The sandbox of the Uno implementation: the shared base image (see
# shared/sandbox) with the libraries that Uno loads on X11 and a pinned .NET SDK.
# Build and use it through sandbox.sh, which passes the versions from
# ../versions.env.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=
ARG DOTNET_VERSION
ARG DOTNET_SHA512

# The shared libraries of the Uno runtime for X11 and of Skia, which draws with the
# processor, and Node.js, which the end-to-end tests against the real vfox serve
# their registry with. Uno needs libicu at startup, also in a build that has the
# invariant globalization of .NET.
RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      libx11-6 libxext6 libxfixes3 libxi6 libxrandr2 \
      libfontconfig1 fontconfig libfreetype6 libexpat1 zlib1g libbz2-1.0 libpng16-16t64 libbrotli1 \
      libicu78 libssl3t64 libgcc-s1 libstdc++6 \
      nodejs \
 && rm -rf /var/lib/apt/lists/*

# The .NET SDK, verified against the published checksum.
RUN test -n "$DOTNET_VERSION" && test -n "$DOTNET_SHA512" \
 && curl -fsSL --retry 5 -o /tmp/dotnet.tar.gz \
      "https://builds.dotnet.microsoft.com/dotnet/Sdk/$DOTNET_VERSION/dotnet-sdk-$DOTNET_VERSION-linux-x64.tar.gz" \
 && echo "$DOTNET_SHA512  /tmp/dotnet.tar.gz" | sha512sum --check --strict - \
 && mkdir /opt/dotnet \
 && tar -xzf /tmp/dotnet.tar.gz -C /opt/dotnet \
 && rm /tmp/dotnet.tar.gz \
 && ln -s /opt/dotnet/dotnet /usr/local/bin/dotnet \
 && install -d -o tester -g tester /home/tester/.nuget /home/tester/.dotnet

# No telemetry, no welcome message; the NuGet cache is in a volume, so that the tasks
# that have no network find what the deps task downloaded.
ENV DOTNET_ROOT=/opt/dotnet \
    DOTNET_CLI_TELEMETRY_OPTOUT=1 \
    DOTNET_NOLOGO=1 \
    DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1 \
    DOTNET_GENERATE_ASPNET_CERTIFICATE=false \
    NUGET_PACKAGES=/home/tester/.nuget/packages
