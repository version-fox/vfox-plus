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

# The sandbox of the web front end: the shared base image (see
# shared/sandbox) with a pinned Node.js. The Tauri and Wails sandboxes build on
# it. Build and use it through sandbox.sh, which passes the versions from
# ../versions.env.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG NODE_VERSION
ARG NODE_SHA256

# Node.js, verified against the checksum published with the release.
RUN test -n "$NODE_VERSION" && test -n "$NODE_SHA256" \
 && curl -fsSL --retry 5 -o /tmp/node.tar.xz \
      "https://nodejs.org/dist/v$NODE_VERSION/node-v$NODE_VERSION-linux-x64.tar.xz" \
 && echo "$NODE_SHA256  /tmp/node.tar.xz" | sha256sum --check --strict - \
 && mkdir /opt/node \
 && tar -xJf /tmp/node.tar.xz -C /opt/node --strip-components=1 \
 && rm /tmp/node.tar.xz \
 && install -d -o tester -g tester /home/tester/.npm

ENV PATH=/opt/node/bin:$PATH \
    npm_config_update_notifier=false \
    npm_config_fund=false \
    npm_config_audit=false
