// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/// Sources passed to `vfox add --source`.
library;

import 'dart:io';

/// Whether [source] is a URL rather than a local path.
bool isRemotePluginSource(String source) => source.startsWith('http://') || source.startsWith('https://');

/// Whether vfox would move [source] instead of reading it.
///
/// vfox 1.0.12 renames a local `.lua` plugin into its temporary directory,
/// which deletes the user's file and fails when the file is on another
/// filesystem; local `.zip` archives and URLs are only read. See
/// `docs/vfox-cli-contract.md`.
bool vfoxMovesPluginSource(String source) => !isRemotePluginSource(source) && source.toLowerCase().endsWith('.lua');

/// The temporary directory of vfox when it runs with [environment]:
/// `~/.version-fox/tmp` if the legacy `~/.version-fox` exists, `~/.vfox/tmp`
/// otherwise. Without `HOME`, where vfox does not run at all, the system
/// temporary directory.
Directory vfoxTemporaryDirectory(Map<String, String> environment) {
  final String? home = environment['HOME'];
  if (home == null || home.isEmpty) {
    return Directory.systemTemp;
  }
  final legacy = Directory('$home/.version-fox');
  return Directory('${legacy.existsSync() ? legacy.path : '$home/.vfox'}/tmp');
}

/// A plugin source that is safe to hand to vfox.
final class PreparedPluginSource {
  const PreparedPluginSource._(this.path, this._copyDirectory);

  /// The path or URL to pass to `vfox add --source`.
  final String path;

  final Directory? _copyDirectory;

  /// Whether [path] is a private copy of the original source.
  bool get isCopy => _copyDirectory != null;

  /// Deletes the private copy, if one was made.
  Future<void> dispose() async {
    final Directory? directory = _copyDirectory;
    if (directory != null && directory.existsSync()) {
      await directory.delete(recursive: true);
    }
  }
}

/// Prepares [source] for `vfox add --source`.
///
/// A local `.lua` file is copied into a new directory under [temporaryRoot],
/// so that vfox moves the copy and the user's file stays where it is. Pass the
/// [vfoxTemporaryDirectory]: vfox moves the file with a rename into a new
/// directory there, which only works on the same filesystem. Other sources
/// are returned as they are. If the copy cannot be made, the original path is
/// returned and vfox reports the problem; it fails to create its own directory
/// in [temporaryRoot] before moving anything.
Future<PreparedPluginSource> preparePluginSource(String source, {required Directory temporaryRoot}) async {
  if (!vfoxMovesPluginSource(source)) {
    return PreparedPluginSource._(source, null);
  }
  final original = File(source);
  Directory? directory;
  try {
    directory = await (await temporaryRoot.create(recursive: true)).createTemp('vfox-plus-plugin-');
    final File copy = await original.copy('${directory.path}/${original.uri.pathSegments.last}');
    return PreparedPluginSource._(copy.path, directory);
  } on FileSystemException {
    await directory?.delete(recursive: true);
    return PreparedPluginSource._(source, null);
  }
}
