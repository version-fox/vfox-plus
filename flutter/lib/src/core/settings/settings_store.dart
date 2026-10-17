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

/// Persistence of [AppSettings].
library;

import 'dart:convert';
import 'dart:io';

import 'app_settings.dart';

/// Loads and saves [AppSettings].
abstract interface class SettingsStore {
  /// Loads the settings, or returns the defaults if none are stored.
  Future<AppSettings> load();

  /// Persists [settings].
  Future<void> save(AppSettings settings);
}

/// Stores settings as JSON in `$XDG_CONFIG_HOME/vfox-plus/settings.json`.
///
/// `XDG_CONFIG_HOME` defaults to `~/.config` as defined by the XDG Base
/// Directory Specification. Corrupted files are ignored rather than fatal:
/// the application then starts with the defaults.
final class JsonFileSettingsStore implements SettingsStore {
  /// Creates a store that reads and writes [file].
  JsonFileSettingsStore(this.file);

  /// Creates a store at the default location for [environment].
  factory JsonFileSettingsStore.standard({Map<String, String>? environment}) {
    final Map<String, String> env = environment ?? Platform.environment;
    final String? xdgConfigHome = env['XDG_CONFIG_HOME'];
    final String base = xdgConfigHome != null && xdgConfigHome.startsWith('/')
        ? xdgConfigHome
        : '${env['HOME'] ?? Directory.systemTemp.path}/.config';
    return JsonFileSettingsStore(File('$base/vfox-plus/settings.json'));
  }

  /// The settings file.
  final File file;

  @override
  Future<AppSettings> load() async {
    try {
      if (!file.existsSync()) {
        return AppSettings();
      }
      final Object? json = jsonDecode(await file.readAsString());
      return json is Map<String, Object?> ? AppSettings.fromJson(json) : AppSettings();
    } on FormatException {
      return AppSettings();
    } on FileSystemException {
      return AppSettings();
    }
  }

  @override
  Future<void> save(AppSettings settings) async {
    await file.parent.create(recursive: true);
    // Write to a temporary file first so that a crash never leaves a
    // truncated settings file behind.
    final temporary = File('${file.path}.tmp');
    await temporary.writeAsString('${const JsonEncoder.withIndent('  ').convert(settings.toJson())}\n', flush: true);
    await temporary.rename(file.path);
  }
}

/// Keeps settings in memory. Used by tests and as a fallback.
final class MemorySettingsStore implements SettingsStore {
  /// Creates a store holding [settings].
  MemorySettingsStore([AppSettings? settings]) : _settings = settings ?? AppSettings();

  AppSettings _settings;

  /// How many times [save] was called.
  int saveCount = 0;

  @override
  Future<AppSettings> load() async => _settings;

  @override
  Future<void> save(AppSettings settings) async {
    saveCount++;
    _settings = settings;
  }
}
