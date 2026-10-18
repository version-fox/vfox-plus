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

# The sandbox of the Capacitor implementation: the sandbox of the web front
# end (Node.js, see web/sandbox) with the libraries that Electron needs. The
# Electron runtime itself is downloaded by the deps task into a cache volume.
# Build and use it through sandbox.sh.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG UBUNTU_MIRROR=

# The libraries that the Electron runtime links, the same as the Depends of the
# package (see run-task.sh), and gdbus, which reads the accent colour.
RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s#http://archive.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g; s#http://security.ubuntu.com/ubuntu/#$UBUNTU_MIRROR#g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      libasound2t64 libatk-bridge2.0-0t64 libatk1.0-0t64 libatspi2.0-0t64 libcairo2 libcups2t64 libdbus-1-3 \
      libexpat1 libgbm1 libglib2.0-0t64 libglib2.0-bin libgtk-3-0t64 libnspr4 libnss3 libpango-1.0-0 libudev1 \
      libx11-6 libxcb1 libxcomposite1 libxdamage1 libxext6 libxfixes3 libxkbcommon0 libxrandr2 xdg-utils \
 && rm -rf /var/lib/apt/lists/* \
 && install -d -o tester -g tester /home/tester/.cache/electron
