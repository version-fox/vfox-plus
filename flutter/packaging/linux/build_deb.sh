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

# Packages the Linux release bundle as a .deb and as a .tar.gz.
#
# Usage: packaging/linux/build_deb.sh [output directory]
#
# Run `flutter build linux --release` first. The output directory defaults to
# ../dist. Set DEB_MAINTAINER to "Name <email>" for published packages.
set -euo pipefail
# Package files must not be group-writable, whatever the caller's umask is.
umask 022

FLUTTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PACKAGING_DIR="$FLUTTER_DIR/packaging/linux"
OUTPUT_DIR="$(mkdir -p "${1:-$FLUTTER_DIR/../dist}" && cd "${1:-$FLUTTER_DIR/../dist}" && pwd)"
APP_ID="dev.vfox.VfoxPlus"
PACKAGE="vfox-plus"
MAINTAINER="${DEB_MAINTAINER:-Han Li and contributors <https://github.com/version-fox/vfox-plus/issues>}"

VERSION="$(sed -n 's/^version: *\([^+ ]*\).*/\1/p' "$FLUTTER_DIR/pubspec.yaml")"
case "$(uname -m)" in
  x86_64) FLUTTER_ARCH=x64; DEB_ARCH=amd64 ;;
  aarch64) FLUTTER_ARCH=arm64; DEB_ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
BUNDLE="$FLUTTER_DIR/build/linux/$FLUTTER_ARCH/release/bundle"
[[ -x "$BUNDLE/$PACKAGE" ]] || { echo "No release bundle in $BUNDLE; run 'flutter build linux --release'." >&2; exit 1; }

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
ROOT="$STAGE/$PACKAGE"

# The bundle stays together under /usr/lib, because the executable finds its
# data and libraries relative to itself.
install -d "$ROOT/usr/lib/$PACKAGE" "$ROOT/usr/bin"
cp -a "$BUNDLE/." "$ROOT/usr/lib/$PACKAGE/"
ln -s "../lib/$PACKAGE/$PACKAGE" "$ROOT/usr/bin/$PACKAGE"

# Flutter links plugins with a RUNPATH into the build directory; the bundled
# libraries sit next to each other, so $ORIGIN is all they need.
if command -v patchelf > /dev/null; then
  for library in "$ROOT/usr/lib/$PACKAGE"/lib/lib*_plugin.so; do
    # shellcheck disable=SC2016 # $ORIGIN is expanded by the dynamic linker.
    patchelf --set-rpath '$ORIGIN' "$library"
  done
else
  echo "warning: patchelf not found; plugin libraries keep their build RUNPATH" >&2
fi
# The runner and the plugins are built with symbols; the engine and the AOT
# snapshot are already stripped.
strip --strip-unneeded "$ROOT/usr/lib/$PACKAGE/$PACKAGE" "$ROOT/usr/lib/$PACKAGE"/lib/lib*_plugin.so

install -Dm644 "$PACKAGING_DIR/$APP_ID.desktop" "$ROOT/usr/share/applications/$APP_ID.desktop"
install -Dm644 "$PACKAGING_DIR/$APP_ID.metainfo.xml" "$ROOT/usr/share/metainfo/$APP_ID.metainfo.xml"
install -Dm644 "$FLUTTER_DIR/assets/brand/vfox-plus.svg" "$ROOT/usr/share/icons/hicolor/scalable/apps/$APP_ID.svg"
install -d "$ROOT/usr/share/doc/$PACKAGE"
cat > "$ROOT/usr/share/doc/$PACKAGE/copyright" <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: vfox-plus
Source: https://github.com/version-fox/vfox-plus

Files: *
Copyright: 2026 Han Li and contributors
License: Apache-2.0

Files: usr/lib/$PACKAGE/data/flutter_assets/assets/fonts/*
Copyright: Google LLC
License: Apache-2.0
Comment: A subset of Material Symbols Rounded,
 https://github.com/google/material-design-icons.

License: Apache-2.0
 On Debian systems, the full text of the Apache License 2.0 can be found in
 /usr/share/common-licenses/Apache-2.0. The licenses of the bundled Flutter
 engine and Dart packages are listed in
 /usr/lib/$PACKAGE/data/flutter_assets/NOTICES.Z.
EOF

# Findings that come from the prebuilt Flutter engine rather than from us.
install -d "$ROOT/usr/share/lintian/overrides"
cat > "$ROOT/usr/share/lintian/overrides/$PACKAGE" <<OVERRIDES
# The Flutter engine statically links FreeType and libjpeg.
$PACKAGE: embedded-library *[usr/lib/$PACKAGE/lib/libflutter_linux_gtk.so]
# The Dart AOT snapshot is not a regular shared library.
$PACKAGE: shared-library-lacks-prerequisites [usr/lib/$PACKAGE/lib/libapp.so]
# A graphical application without command line options.
$PACKAGE: no-manual-page [usr/bin/$PACKAGE]
OVERRIDES
gzip -9n > "$ROOT/usr/share/doc/$PACKAGE/changelog.gz" <<CHANGELOG
$PACKAGE ($VERSION) unstable; urgency=medium

  * Release $VERSION. See https://github.com/version-fox/vfox-plus/releases.

 -- $MAINTAINER  $(date -R -d "@${SOURCE_DATE_EPOCH:-$(date +%s)}")
CHANGELOG

install -d "$ROOT/DEBIAN"
# The Flutter engine loads EGL and OpenGL ES at run time through libepoxy, so
# ldd does not list libegl1 and libgles2; the sandbox installation test checks
# that the application starts with these dependencies only.
cat > "$ROOT/DEBIAN/control" <<EOF
Package: $PACKAGE
Version: $VERSION
Section: devel
Priority: optional
Architecture: $DEB_ARCH
Maintainer: $MAINTAINER
Installed-Size: $(du -sk "$ROOT/usr" | cut -f1)
Depends: libgtk-3-0t64 | libgtk-3-0, libglib2.0-0t64 | libglib2.0-0, libepoxy0, libegl1, libgles2, libfontconfig1, libstdc++6, libc6, zlib1g
Recommends: vfox, librsvg2-common
Homepage: https://github.com/version-fox/vfox-plus
Description: Material 3 Expressive desktop GUI for the vfox version manager
 vfox+ shows the installed SDKs and the versions in use, installs and
 uninstalls versions, pins versions in projects, browses the vfox plugin
 registry and edits the vfox configuration. It drives the vfox command line
 tool, which must be installed separately.
EOF

chmod -R u+rwX,go+rX,go-w "$ROOT"

DEB="$OUTPUT_DIR/${PACKAGE}_${VERSION}_${DEB_ARCH}.deb"
TARBALL="$OUTPUT_DIR/${PACKAGE}-${VERSION}-linux-${FLUTTER_ARCH}.tar.gz"
dpkg-deb --root-owner-group -Zxz --build "$ROOT" "$DEB" > /dev/null
# The tarball also carries the license, which the .deb has in its copyright file.
tar --owner=0 --group=0 -czf "$TARBALL" -C "$BUNDLE" . -C "$FLUTTER_DIR/.." LICENSE

echo "$DEB"
echo "$TARBALL"
