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

# Packages the release build of an implementation as a .deb and a .tar.gz,
# with the same layout, desktop entry, AppStream metadata and icon for every
# implementation: the application goes to /usr/lib/vfox-plus and is started
# as /usr/bin/vfox-plus.
#
# Usage: shared/packaging/linux/build-deb.sh [options] <output directory>
#
#   --package NAME        Debian package name, e.g. vfox-plus-tauri (required)
#   --version VERSION     version of the application (required)
#   --stage DIRECTORY     files of the application, installed to
#                         /usr/lib/vfox-plus (required)
#   --executable PATH     the program in the stage directory that
#                         /usr/bin/vfox-plus points to (required)
#   --depends LIST        the Depends field (required)
#   --recommends LIST     packages to recommend besides vfox
#   --technology TEXT     the technology, for the description (required)
#   --notice FILE         third-party licenses, installed as
#                         /usr/share/doc/<package>/THIRD-PARTY-LICENSES
#   --override TEXT       a lintian override, may be repeated
#   --setuid FILE         a program of the stage that runs as root, such as the
#                         sandbox helper of Chromium, may be repeated
#   --strip               removes the symbols of the ELF files of the stage, for
#                         stages that bundle unstripped native code, such as a
#                         Java runtime
#
# Every package provides, conflicts with and replaces "vfox-plus", so that the
# implementations, which share the application ID dev.vfox.VfoxPlus, replace
# each other. Set DEB_MAINTAINER to "Name <email>" for published packages and
# SOURCE_DATE_EPOCH for reproducible timestamps.
set -euo pipefail
# Package files must not be group-writable, whatever the caller's umask is.
umask 022

PACKAGING_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$PACKAGING_DIR/../../.." && pwd)"
APP_ID="dev.vfox.VfoxPlus"
MAINTAINER="${DEB_MAINTAINER:-Han Li and contributors <https://github.com/version-fox/vfox-plus/issues>}"

package="" version="" stage="" executable="" depends="" recommends="" technology="" notice="" strip=false
overrides=()
setuid=()
while [[ $# -gt 1 ]]; do
  if [[ "$1" == --strip ]]; then
    strip=true
    shift
    continue
  fi
  case "$1" in
    --package) package="$2" ;;
    --version) version="$2" ;;
    --stage) stage="$2" ;;
    --executable) executable="$2" ;;
    --depends) depends="$2" ;;
    --recommends) recommends="$2" ;;
    --technology) technology="$2" ;;
    --notice) notice="$2" ;;
    --override) overrides+=("$2") ;;
    --setuid) setuid+=("$2") ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
  shift 2
done
output="${1:?Usage: build-deb.sh [options] <output directory>}"
for required in package version stage executable depends technology; do
  [[ -n "${!required}" ]] || { echo "--$required is required" >&2; exit 2; }
done
[[ -x "$stage/$executable" ]] || { echo "$stage/$executable is not an executable" >&2; exit 1; }

case "$(uname -m)" in
  x86_64) DEB_ARCH=amd64; ARCH=x64 ;;
  aarch64) DEB_ARCH=arm64; ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
mkdir -p "$output"
output="$(cd "$output" && pwd)"

TEMPORARY="$(mktemp -d)"
trap 'rm -rf "$TEMPORARY"' EXIT
ROOT="$TEMPORARY/$package"

install -d "$ROOT/usr/lib/vfox-plus" "$ROOT/usr/bin"
cp -a "$stage/." "$ROOT/usr/lib/vfox-plus/"
ln -s "../lib/vfox-plus/$executable" "$ROOT/usr/bin/vfox-plus"
if [[ "$strip" == true ]]; then
  while IFS= read -r -d '' file; do
    if file --brief "$file" | grep -q '^ELF'; then
      strip --strip-unneeded "$file"
    fi
  done < <(find "$ROOT/usr/lib/vfox-plus" -type f -print0)
fi

install -Dm644 "$PACKAGING_DIR/$APP_ID.desktop" "$ROOT/usr/share/applications/$APP_ID.desktop"
install -Dm644 "$PACKAGING_DIR/$APP_ID.metainfo.xml" "$ROOT/usr/share/metainfo/$APP_ID.metainfo.xml"
install -Dm644 "$PACKAGING_DIR/$APP_ID.svg" "$ROOT/usr/share/icons/hicolor/scalable/apps/$APP_ID.svg"

DOC="$ROOT/usr/share/doc/$package"
install -d "$DOC"
cat > "$DOC/copyright" << EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: vfox-plus
Source: https://github.com/version-fox/vfox-plus

Files: *
Copyright: 2026 Han Li and contributors
License: Apache-2.0
Comment: The licenses of the bundled third-party components, including a subset
 of the Material Symbols Rounded font (Apache-2.0, Google LLC), are listed in
 /usr/share/doc/$package/THIRD-PARTY-LICENSES when that file exists.

License: Apache-2.0
 On Debian systems, the full text of the Apache License 2.0 can be found in
 /usr/share/common-licenses/Apache-2.0.
EOF
if [[ -n "$notice" ]]; then
  install -m644 "$notice" "$DOC/THIRD-PARTY-LICENSES"
fi
gzip -9n > "$DOC/changelog.gz" << CHANGELOG
$package ($version) unstable; urgency=medium

  * Release $version. See https://github.com/version-fox/vfox-plus/releases.

 -- $MAINTAINER  $(date -R -d "@${SOURCE_DATE_EPOCH:-$(date +%s)}")
CHANGELOG

install -d "$ROOT/usr/share/lintian/overrides"
{
  echo "# A graphical application without command line options."
  echo "$package: no-manual-page [usr/bin/vfox-plus]"
  for override in ${overrides[@]+"${overrides[@]}"}; do
    echo "$package: $override"
  done
} > "$ROOT/usr/share/lintian/overrides/$package"

install -d "$ROOT/DEBIAN"
cat > "$ROOT/DEBIAN/control" << EOF
Package: $package
Version: $version
Section: devel
Priority: optional
Architecture: $DEB_ARCH
Maintainer: $MAINTAINER
Installed-Size: $(du -sk "$ROOT/usr" | cut -f1)
Depends: $depends
Recommends: vfox${recommends:+, $recommends}
Provides: vfox-plus
Conflicts: vfox-plus
Replaces: vfox-plus
Homepage: https://github.com/version-fox/vfox-plus
Description: Material 3 Expressive desktop GUI for vfox ($technology)
 vfox+ shows the installed SDKs and the versions in use, installs and
 uninstalls versions, pins versions in projects, browses the vfox plugin
 registry and edits the vfox configuration. It drives the vfox command line
 tool, which must be installed separately.
 .
 This package is the implementation of vfox+ built with $technology.
EOF

chmod -R u+rwX,go+rX,go-w "$ROOT"
# After stripping, which rewrites the files; dpkg-deb makes root their owner.
for file in ${setuid[@]+"${setuid[@]}"}; do
  [[ -f "$ROOT/usr/lib/vfox-plus/$file" ]] || { echo "$file is not in the stage" >&2; exit 1; }
  chmod 4755 "$ROOT/usr/lib/vfox-plus/$file"
done
DEB="$output/${package}_${version}_${DEB_ARCH}.deb"
TARBALL="$output/${package}-${version}-linux-${ARCH}.tar.gz"
dpkg-deb --root-owner-group -Zxz --build "$ROOT" "$DEB" > /dev/null
# The tarball also carries the licenses, which the .deb has in its documentation.
tar --owner=0 --group=0 -czf "$TARBALL" -C "$ROOT/usr/lib/vfox-plus" . -C "$REPO" LICENSE \
  ${notice:+-C "$(dirname "$notice")" "$(basename "$notice")"}
echo "$DEB"
echo "$TARBALL"
