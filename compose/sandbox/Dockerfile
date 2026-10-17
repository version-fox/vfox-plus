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

# The sandbox of the Compose Multiplatform implementation: the shared base
# image (see shared/sandbox) with the libraries AWT needs, a pinned Temurin
# JDK and a pinned Gradle. Build and use it through sandbox.sh, which passes
# the versions from ../versions.env.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=
ARG JDK_MAJOR
ARG JDK_RELEASE
ARG JDK_SHA256
ARG GRADLE_VERSION
ARG GRADLE_SHA256

RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      libxext6 libxrender1 libxtst6 libxi6 libxrandr2 libfreetype6 fontconfig libgl1 libegl1 \
 && rm -rf /var/lib/apt/lists/*

# The JDK and Gradle, verified against the published checksums.
RUN test -n "$JDK_RELEASE" && test -n "$JDK_SHA256" \
 && version="${JDK_RELEASE#jdk-}" \
 && curl -fsSL --retry 5 -o /tmp/jdk.tar.gz \
      "https://github.com/adoptium/temurin${JDK_MAJOR}-binaries/releases/download/$(echo "$JDK_RELEASE" | sed 's/+/%2B/')/OpenJDK${JDK_MAJOR}U-jdk_x64_linux_hotspot_$(echo "$version" | tr '+' '_').tar.gz" \
 && echo "$JDK_SHA256  /tmp/jdk.tar.gz" | sha256sum --check --strict - \
 && mkdir /opt/jdk \
 && tar -xzf /tmp/jdk.tar.gz -C /opt/jdk --strip-components=1 \
 && rm /tmp/jdk.tar.gz \
 && test -n "$GRADLE_VERSION" && test -n "$GRADLE_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/gradle.zip "https://services.gradle.org/distributions/gradle-$GRADLE_VERSION-bin.zip" \
 && echo "$GRADLE_SHA256  /tmp/gradle.zip" | sha256sum --check --strict - \
 && unzip -q /tmp/gradle.zip -d /opt \
 && mv "/opt/gradle-$GRADLE_VERSION" /opt/gradle \
 && rm /tmp/gradle.zip \
 && install -d -o tester -g tester /home/tester/.gradle

ENV JAVA_HOME=/opt/jdk \
    PATH=/opt/jdk/bin:/opt/gradle/bin:$PATH \
    GRADLE_OPTS=-Dorg.gradle.daemon=false
