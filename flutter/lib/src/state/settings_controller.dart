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

import '../core/settings/app_settings.dart';
import '../core/settings/settings_store.dart';

/// Holds the [AppSettings] and persists every change.
class SettingsController extends ChangeNotifier {
  /// Creates a controller backed by `store`.
  SettingsController(this._store);

  final SettingsStore _store;
  AppSettings _settings = AppSettings();

  /// The current settings.
  AppSettings get settings => _settings;

  /// Loads the persisted settings.
  Future<void> load() async {
    _settings = await _store.load();
    notifyListeners();
  }

  /// Applies [change] to the settings and saves the result.
  Future<void> update(AppSettings Function(AppSettings settings) change) async {
    final AppSettings updated = change(_settings);
    if (updated == _settings) {
      return;
    }
    _settings = updated;
    notifyListeners();
    await _store.save(updated);
  }
}
