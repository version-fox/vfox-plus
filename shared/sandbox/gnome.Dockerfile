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

# The image of a GNOME Shell with no screen, in which the end-to-end tests of the icon in the
# tray meet the shell that the icon is for (`vfxtool gnome-session`, see README.md next to this
# file): the shell and the extension of Ubuntu that puts the icons of applications in its panel,
# the libraries that the application loads, and an ordinary user to run the session as. The
# build context is this directory; nothing is copied into the image: the CI mounts vfxtool and the
# build of the application in.
FROM ubuntu:26.04

# An Ubuntu archive mirror such as http://mirrors.aliyun.com/ubuntu/, if the
# default archive is slow from where the image is built.
ARG UBUNTU_MIRROR=

ENV DEBIAN_FRONTEND=noninteractive

# GNOME Shell with mutter, which serves Wayland without a GPU, and Xwayland, which it starts;
# gnome-shell-ubuntu-extensions has the AppIndicator extension (ubuntu-appindicators@ubuntu.com);
# a system bus, which the shell looks for, and the session bus and dconf that its settings use; the
# software drivers of Mesa that mutter draws with; and what the application loads: GTK 3, the library
# of the icon in the tray, fontconfig and the fonts.
RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s|http://archive.ubuntu.com/ubuntu/|$UBUNTU_MIRROR|g; s|http://security.ubuntu.com/ubuntu/|$UBUNTU_MIRROR|g" /etc/apt/sources.list.d/ubuntu.sources; \
    fi \
 && apt-get update \
 && apt-get install -y --no-install-recommends \
      gnome-shell gnome-shell-ubuntu-extensions mutter xwayland \
      dbus dbus-bin dbus-daemon dbus-user-session dbus-x11 \
      dconf-gsettings-backend dconf-service gsettings-desktop-schemas adwaita-icon-theme accountsservice \
      libgl1-mesa-dri libegl1 libgles2 mesa-vulkan-drivers \
      libgtk-3-0t64 libayatana-appindicator3-1 fontconfig fonts-ubuntu fonts-noto-cjk \
      xdg-utils ca-certificates procps \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --create-home --shell /bin/bash tester
