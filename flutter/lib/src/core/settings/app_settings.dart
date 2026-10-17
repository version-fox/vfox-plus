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

/// Preferences of the application itself (not of vfox).
library;

import 'dart:collection';

/// Light or dark appearance.
enum ThemePreference {
  /// Follow the desktop's preference.
  system,

  /// Always light.
  light,

  /// Always dark.
  dark,
}

/// The user's language choice.
enum LanguagePreference {
  /// Follow the desktop's locale, falling back to English.
  system(null),

  /// English.
  english('en'),

  /// Simplified Chinese.
  simplifiedChinese('zh');

  const LanguagePreference(this.languageCode);

  /// The language code of the translation catalogue, or `null` for [system].
  final String? languageCode;
}

/// Preferences of the application, persisted as JSON.
///
/// The JSON keys are part of the shared specification in
/// `docs/architecture.md#settings`, so that every implementation of the GUI
/// can read the same file.
final class AppSettings {
  /// Creates settings. The defaults are used for a fresh installation.
  AppSettings({
    this.theme = ThemePreference.system,
    this.useSystemColor = true,
    this.seedColor = defaultSeedColor,
    this.language = LanguagePreference.system,
    this.vfoxExecutable,
    Map<String, String> extraEnvironment = const <String, String>{},
    this.railExpanded = false,
  }) : extraEnvironment = UnmodifiableMapView(extraEnvironment);

  /// Restores settings from [json], ignoring unknown or malformed values.
  factory AppSettings.fromJson(Map<String, Object?> json) {
    final Object? environment = json['extraEnvironment'];
    return AppSettings(
      theme: _enumByName(ThemePreference.values, json['theme']) ?? ThemePreference.system,
      useSystemColor: json['useSystemColor'] is bool ? json['useSystemColor']! as bool : true,
      seedColor: json['seedColor'] is int ? json['seedColor']! as int : defaultSeedColor,
      language: _enumByName(LanguagePreference.values, json['language']) ?? LanguagePreference.system,
      vfoxExecutable: json['vfoxExecutable'] is String && (json['vfoxExecutable']! as String).isNotEmpty
          ? json['vfoxExecutable']! as String
          : null,
      extraEnvironment: environment is Map
          ? <String, String>{
              for (final MapEntry<Object?, Object?> entry in environment.entries)
                if (entry.key is String && entry.value is String) entry.key! as String: entry.value! as String,
            }
          : const <String, String>{},
      railExpanded: json['railExpanded'] is bool && json['railExpanded']! as bool,
    );
  }

  /// The brand colour used when [useSystemColor] is off or the desktop does
  /// not expose an accent colour: the orange of the vfox logo.
  static const int defaultSeedColor = 0xFFF28C1B;

  /// Light or dark appearance.
  final ThemePreference theme;

  /// Whether to derive the colour scheme from the desktop accent colour.
  final bool useSystemColor;

  /// ARGB seed of the colour scheme when [useSystemColor] is off.
  final int seedColor;

  /// The user interface language.
  final LanguagePreference language;

  /// Path of the vfox executable chosen by the user, or `null` to detect it.
  final String? vfoxExecutable;

  /// Additional environment variables for vfox, such as mirror settings.
  final Map<String, String> extraEnvironment;

  /// Whether the navigation rail shows labels next to the icons.
  final bool railExpanded;

  /// Returns a copy with the given fields replaced.
  AppSettings copyWith({
    ThemePreference? theme,
    bool? useSystemColor,
    int? seedColor,
    LanguagePreference? language,
    String? vfoxExecutable,
    bool clearVfoxExecutable = false,
    Map<String, String>? extraEnvironment,
    bool? railExpanded,
  }) {
    return AppSettings(
      theme: theme ?? this.theme,
      useSystemColor: useSystemColor ?? this.useSystemColor,
      seedColor: seedColor ?? this.seedColor,
      language: language ?? this.language,
      vfoxExecutable: clearVfoxExecutable ? null : vfoxExecutable ?? this.vfoxExecutable,
      extraEnvironment: extraEnvironment ?? this.extraEnvironment,
      railExpanded: railExpanded ?? this.railExpanded,
    );
  }

  /// Serialises the settings.
  Map<String, Object?> toJson() => <String, Object?>{
    'theme': theme.name,
    'useSystemColor': useSystemColor,
    'seedColor': seedColor,
    'language': language.name,
    if (vfoxExecutable != null) 'vfoxExecutable': vfoxExecutable,
    'extraEnvironment': extraEnvironment,
    'railExpanded': railExpanded,
  };

  @override
  bool operator ==(Object other) =>
      other is AppSettings &&
      other.theme == theme &&
      other.useSystemColor == useSystemColor &&
      other.seedColor == seedColor &&
      other.language == language &&
      other.vfoxExecutable == vfoxExecutable &&
      other.railExpanded == railExpanded &&
      _mapEquals(other.extraEnvironment, extraEnvironment);

  @override
  int get hashCode => Object.hash(
    theme,
    useSystemColor,
    seedColor,
    language,
    vfoxExecutable,
    railExpanded,
    Object.hashAllUnordered(extraEnvironment.entries.map((MapEntry<String, String> e) => Object.hash(e.key, e.value))),
  );
}

/// Parses `KEY=VALUE` lines into environment variables.
///
/// Blank lines and lines starting with `#` are ignored. Keys must be valid
/// POSIX variable names. Returns the variables and the 1-based numbers of the
/// lines that could not be parsed.
({Map<String, String> variables, List<int> invalidLines}) parseEnvironmentLines(String text) {
  final variables = <String, String>{};
  final invalidLines = <int>[];
  final keyPattern = RegExp(r'^[A-Za-z_][A-Za-z0-9_]*$');
  final List<String> lines = text.split('\n');
  for (var i = 0; i < lines.length; i++) {
    final String line = lines[i].trim();
    if (line.isEmpty || line.startsWith('#')) {
      continue;
    }
    final int equals = line.indexOf('=');
    final String key = equals < 0 ? line : line.substring(0, equals).trim();
    if (equals < 0 || !keyPattern.hasMatch(key)) {
      invalidLines.add(i + 1);
      continue;
    }
    variables[key] = line.substring(equals + 1).trim();
  }
  return (variables: variables, invalidLines: invalidLines);
}

/// Formats environment variables as `KEY=VALUE` lines, sorted by key.
String formatEnvironmentLines(Map<String, String> variables) {
  final List<String> keys = variables.keys.toList()..sort();
  return keys.map((String key) => '$key=${variables[key]}').join('\n');
}

T? _enumByName<T extends Enum>(List<T> values, Object? name) {
  for (final T value in values) {
    if (value.name == name) {
      return value;
    }
  }
  return null;
}

bool _mapEquals(Map<String, String> a, Map<String, String> b) {
  if (a.length != b.length) {
    return false;
  }
  for (final MapEntry<String, String> entry in a.entries) {
    if (b[entry.key] != entry.value) {
      return false;
    }
  }
  return true;
}
