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

import 'package:flutter/foundation.dart';

import '../core/models.dart';
import '../core/vfox/vfox_cli.dart';
import 'async_resource.dart';

/// Which versions are listed on the "available" tab.
enum VersionFilter {
  /// Every version.
  all,

  /// Versions carrying a note such as `LTS`.
  withNote,

  /// Versions that are installed already.
  installed,
}

/// State of the detail view of one installed plugin.
class SdkDetailController extends ChangeNotifier {
  /// Creates the controller for the plugin [name].
  SdkDetailController({required VfoxCli cli, required this.name})
    : info = AsyncResource<PluginInfo>(() => cli.pluginInfo(name)),
      versions = AsyncResource<List<AvailableVersion>>(() => cli.searchVersions(name)) {
    versions.addListener(notifyListeners);
  }

  /// The plugin name.
  final String name;

  /// Plugin metadata, loaded on demand.
  final AsyncResource<PluginInfo> info;

  /// Installable versions, loaded on demand because the search can be slow.
  final AsyncResource<List<AvailableVersion>> versions;

  String _query = '';

  /// The text the available versions are filtered by.
  String get query => _query;

  set query(String value) {
    if (value != _query) {
      _query = value;
      notifyListeners();
    }
  }

  VersionFilter _filter = VersionFilter.all;

  /// Which available versions are shown.
  VersionFilter get filter => _filter;

  set filter(VersionFilter value) {
    if (value != _filter) {
      _filter = value;
      notifyListeners();
    }
  }

  /// The available versions matching [query] and [filter].
  ///
  /// [installed] holds the installed versions. It is passed in rather than
  /// taken from the search output, so that it stays current after installs
  /// without searching again.
  List<AvailableVersion> visibleVersions({required Set<String> installed}) {
    final List<AvailableVersion> all = versions.value ?? const <AvailableVersion>[];
    final String needle = _query.trim().toLowerCase();
    return <AvailableVersion>[
      for (final AvailableVersion version in all)
        if (_matchesFilter(version, installed) && (needle.isEmpty || _matchesQuery(version, needle))) version,
    ];
  }

  bool _matchesFilter(AvailableVersion version, Set<String> installed) => switch (_filter) {
    VersionFilter.all => true,
    VersionFilter.withNote => version.note != null,
    VersionFilter.installed => installed.contains(version.version),
  };

  static bool _matchesQuery(AvailableVersion version, String needle) =>
      version.version.toLowerCase().contains(needle) || (version.note?.toLowerCase().contains(needle) ?? false);

  @override
  void dispose() {
    versions.removeListener(notifyListeners);
    info.dispose();
    versions.dispose();
    super.dispose();
  }
}
