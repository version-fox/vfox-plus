#!/usr/bin/env bash
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

# Packages the release that `flet build linux` assembled in build/linux as
# vfox-plus-flet_<version>_amd64.deb and a .tar.gz, with
# shared/packaging/linux/build-deb.sh. The sandbox (sandbox/run-task.sh) and
# CI (.github/workflows/flet.yml) both package with this script.
#
# Usage: flet/tools/package.sh <output directory>
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${1:?Usage: package.sh <output directory>}"
BUNDLE="$APP_DIR/build/linux"

# The libraries that the Flutter engine and the plugins link on Ubuntu 26.04
# (see `readelf -d` and `ldd` of the bundle); the installation test of the
# sandbox and of CI checks that they are complete. The Flutter engine loads
# EGL and OpenGL ES at run time through libepoxy, so ldd does not list
# libegl1 and libgles2. gdbus reads the desktop accent colour.
DEPENDS="libgtk-3-0t64 | libgtk-3-0, libglib2.0-0t64 | libglib2.0-0, libglib2.0-bin"
DEPENDS+=", libepoxy0, libegl1, libgles2, libfontconfig1, libblkid1, liblzma5"
DEPENDS+=", libssl3t64 | libssl3, libexpat1, libffi8, libsqlite3-0, libstdc++6, libc6, zlib1g"

[[ -x "$BUNDLE/vfox-plus" ]] || { echo "Run flet build linux first." >&2; exit 1; }
mkdir -p "$APP_DIR/dist/package"
PYTHONPATH="$BUNDLE/site-packages" python3 "$APP_DIR/tools/licenses.py" "$BUNDLE" "$APP_DIR/dist/package/THIRD-PARTY-LICENSES"
# The stage carries the whole bundle, except the desktop entry and the icons
# that `flet build` generates: the package installs the shared ones instead.
STAGE="$APP_DIR/dist/package/stage"
rm -rf "$STAGE"
mkdir -p "$STAGE"
tar -C "$BUNDLE" --create --exclude=./share . | tar -C "$STAGE" --extract
# The console scripts of the bundled Python packages point at the Python of
# the build machine; nothing in the bundle runs them.
rm -rf "$STAGE/site-packages/bin"
# The plugins find their libraries next to themselves.
if command -v patchelf > /dev/null; then
  for library in "$STAGE"/lib/lib*_plugin.so; do
    # shellcheck disable=SC2016 # $ORIGIN is expanded by the dynamic linker.
    patchelf --set-rpath '$ORIGIN' "$library"
  done
else
  echo "warning: patchelf not found; plugin libraries keep their build RUNPATH" >&2
fi
# strip --strip-unneeded leaves the symbol table, which lintian still flags
# on the bundled interpreter; Debian's own libpython is fully stripped.
strip "$STAGE"/lib/libpython3.so "$STAGE"/lib/libpython3.14.so.1.0 \
  "$STAGE"/python3.14/lib-dynload/_dbm*.so "$STAGE"/python3.14/lib-dynload/_tkinter*.so
# The embedded libraries of the prebuilt Flutter engine and interpreter.
overrides=(--override "embedded-library bzip2 [usr/lib/vfox-plus/lib/libpython3.14.so.1.0]")
for library in expat libzstd ncurses zlib; do
  overrides+=(--override "embedded-library $library [usr/lib/vfox-plus/lib/libpython3.14.so.1.0]")
done
for library in freetype libjpeg; do
  overrides+=(--override "embedded-library $library [usr/lib/vfox-plus/lib/libflutter_linux_gtk.so]")
done
# Flet ships its program compiled; the sources are in the repository.
overrides+=(--override "package-installs-python-bytecode")
bash "$APP_DIR/../shared/packaging/linux/build-deb.sh" \
  --package vfox-plus-flet --version "$(sed -n 's/^version = "\(.*\)"$/\1/p' "$APP_DIR/pyproject.toml")" \
  --stage "$STAGE" --executable vfox-plus --strip \
  --technology "Flet" --depends "$DEPENDS" \
  --notice "$APP_DIR/dist/package/THIRD-PARTY-LICENSES" "${overrides[@]}" "$OUTPUT"
