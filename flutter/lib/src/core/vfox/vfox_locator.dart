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

/// Finds the vfox executable.
library;

import 'dart:io';

/// Looks for the vfox executable in `PATH` and in common install locations.
///
/// Applications started from a desktop launcher do not source the user's
/// shell profile, so directories such as `~/.local/bin` are often missing
/// from their `PATH`. The well-known install locations of the official
/// installers and package managers are therefore searched as well.
final class VfoxLocator {
  /// Creates a locator.
  ///
  /// [environment] defaults to `Platform.environment`; [isExecutable] can be
  /// replaced in tests.
  VfoxLocator({Map<String, String>? environment, bool Function(String path)? isExecutable})
    : _environment = environment ?? Platform.environment,
      _isExecutable = isExecutable ?? _isExecutableFile;

  final Map<String, String> _environment;
  final bool Function(String path) _isExecutable;

  /// The executable name.
  static const String executableName = 'vfox';

  /// Directories searched in addition to `PATH`, in order.
  List<String> get fallbackDirectories {
    final String? home = _environment['HOME'];
    return <String>[
      '/usr/local/bin',
      '/usr/bin',
      if (home != null) ...<String>['$home/.local/bin', '$home/bin', '$home/.linuxbrew/bin'],
      '/home/linuxbrew/.linuxbrew/bin',
      '/snap/bin',
    ];
  }

  /// All directories that are searched, without duplicates.
  List<String> get searchDirectories {
    final List<String> path = (_environment['PATH'] ?? '')
        .split(':')
        .where((String entry) => entry.isNotEmpty)
        .toList();
    return <String>{...path, ...fallbackDirectories}.toList();
  }

  /// Returns the path of the first vfox executable found, or `null`.
  String? locate() {
    for (final String directory in searchDirectories) {
      final candidate = '$directory/$executableName';
      if (_isExecutable(candidate)) {
        return candidate;
      }
    }
    return null;
  }

  /// Whether [path] is an existing, executable regular file.
  bool isUsable(String path) => _isExecutable(path);

  static bool _isExecutableFile(String path) {
    try {
      final FileStat stat = FileStat.statSync(path);
      // Any of the user, group or other execute bits.
      return stat.type == FileSystemEntityType.file && stat.mode & 0x49 != 0;
    } on FileSystemException {
      return false;
    }
  }
}
