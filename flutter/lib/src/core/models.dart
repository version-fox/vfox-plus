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

/// Domain models shared by every layer of the application.
///
/// The models mirror the concepts of the vfox CLI that are described in
/// `docs/vfox-cli-contract.md`. They are plain immutable value objects so
/// that they can be compared in tests and rebuilt cheaply.
library;

import 'dart:collection';

/// An installed vfox plugin together with the SDK versions it manages.
///
/// vfox does not distinguish between a plugin and the SDK it installs:
/// `vfox add nodejs` adds the `nodejs` plugin, and `vfox install nodejs@24`
/// installs a version of the `nodejs` SDK through that plugin.
final class InstalledSdk {
  /// Creates an installed SDK.
  InstalledSdk({required this.name, required List<String> versions, this.current})
    : versions = UnmodifiableListView(versions);

  /// Plugin name, for example `nodejs`.
  final String name;

  /// Installed versions, in the order reported by vfox (newest first).
  final List<String> versions;

  /// The version that is currently in effect, or `null` if none is.
  final String? current;

  /// Whether at least one version is installed.
  bool get hasVersions => versions.isNotEmpty;

  /// Returns a copy with the given fields replaced.
  InstalledSdk copyWith({List<String>? versions, String? current, bool clearCurrent = false}) {
    return InstalledSdk(
      name: name,
      versions: versions ?? this.versions,
      current: clearCurrent ? null : current ?? this.current,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is InstalledSdk && other.name == name && other.current == current && _listEquals(other.versions, versions);

  @override
  int get hashCode => Object.hash(name, current, Object.hashAll(versions));

  @override
  String toString() => 'InstalledSdk($name, current: $current, versions: $versions)';
}

/// Metadata of an installed plugin, as reported by `vfox info <name>`.
final class PluginInfo {
  /// Creates plugin metadata.
  const PluginInfo({
    required this.name,
    required this.version,
    required this.homepage,
    required this.installPath,
    required this.description,
  });

  /// Plugin name.
  final String name;

  /// Version of the plugin itself (not of the SDK).
  final String version;

  /// Homepage of the plugin, usually a GitHub repository.
  final String homepage;

  /// Directory in which the SDK versions of this plugin are installed.
  final String installPath;

  /// Human readable description.
  final String description;

  @override
  bool operator ==(Object other) =>
      other is PluginInfo &&
      other.name == name &&
      other.version == version &&
      other.homepage == homepage &&
      other.installPath == installPath &&
      other.description == description;

  @override
  int get hashCode => Object.hash(name, version, homepage, installPath, description);

  @override
  String toString() => 'PluginInfo($name $version)';
}

/// A plugin listed in the vfox plugin registry (`vfox available`).
final class AvailablePlugin {
  /// Creates a registry entry.
  const AvailablePlugin({required this.name, required this.homepage, required this.official});

  /// Plugin name that is passed to `vfox add`.
  final String name;

  /// Homepage of the plugin.
  final String homepage;

  /// Whether the plugin is maintained by the version-fox organisation.
  final bool official;

  @override
  bool operator ==(Object other) =>
      other is AvailablePlugin && other.name == name && other.homepage == homepage && other.official == official;

  @override
  int get hashCode => Object.hash(name, homepage, official);

  @override
  String toString() => 'AvailablePlugin($name, official: $official)';
}

/// A tool that is bundled with an SDK version, such as `npm` for Node.js.
final class VersionAddition {
  /// Creates an addition.
  const VersionAddition({required this.name, required this.version});

  /// Name of the bundled tool.
  final String name;

  /// Version of the bundled tool. May be empty when the plugin omits it.
  final String version;

  @override
  bool operator ==(Object other) => other is VersionAddition && other.name == name && other.version == version;

  @override
  int get hashCode => Object.hash(name, version);

  @override
  String toString() => version.isEmpty ? name : '$name $version';
}

/// A version that can be installed, as listed by `vfox search <name>`.
final class AvailableVersion {
  /// Creates an installable version.
  AvailableVersion({
    required this.version,
    this.note,
    List<VersionAddition> additions = const <VersionAddition>[],
    this.installed = false,
  }) : additions = UnmodifiableListView(additions);

  /// The version string to pass to `vfox install <name>@<version>`.
  final String version;

  /// Optional remark from the plugin, for example `LTS`.
  final String? note;

  /// Tools bundled with this version.
  final List<VersionAddition> additions;

  /// Whether this version is already installed.
  final bool installed;

  @override
  bool operator ==(Object other) =>
      other is AvailableVersion &&
      other.version == version &&
      other.note == note &&
      other.installed == installed &&
      _listEquals(other.additions, additions);

  @override
  int get hashCode => Object.hash(version, note, installed, Object.hashAll(additions));

  @override
  String toString() => 'AvailableVersion($version, note: $note, installed: $installed)';
}

/// One `key = value` line of `vfox config --list`.
final class ConfigEntry {
  /// Creates a configuration entry.
  const ConfigEntry({required this.key, required this.value});

  /// Dotted configuration key, for example `proxy.url`.
  final String key;

  /// Current value as printed by vfox. Empty when the value is unset.
  final String value;

  @override
  bool operator ==(Object other) => other is ConfigEntry && other.key == key && other.value == value;

  @override
  int get hashCode => Object.hash(key, value);

  @override
  String toString() => '$key = $value';
}

/// Where a version selected with `vfox use` takes effect.
///
/// vfox also knows a `session` scope, which only exists inside an activated
/// shell. A desktop application has no shell session, so it is not offered.
enum UseScope {
  /// The version becomes the default for the current user
  /// (`~/.vfox/.vfox.toml`).
  global,

  /// The version is pinned in the `.vfox.toml` of a project directory.
  project,
}

bool _listEquals<T>(List<T> a, List<T> b) {
  if (identical(a, b)) {
    return true;
  }
  if (a.length != b.length) {
    return false;
  }
  for (var i = 0; i < a.length; i++) {
    if (a[i] != b[i]) {
      return false;
    }
  }
  return true;
}
