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

import '../core/models.dart';
import '../core/vfox/vfox_cli.dart';
import 'async_resource.dart';

/// The vfox configuration (`vfox config`), editable key by key.
class VfoxConfigController extends AsyncResource<List<ConfigEntry>> {
  /// Creates a controller that reads and writes the configuration with `cli`.
  VfoxConfigController(this._cli) : super(_cli.configEntries);

  final VfoxCli _cli;
  final Set<String> _saving = <String>{};

  /// The value of [key], or `null` if vfox does not know the key.
  String? valueOf(String key) {
    for (final ConfigEntry entry in value ?? const <ConfigEntry>[]) {
      if (entry.key == key) {
        return entry.value;
      }
    }
    return null;
  }

  /// Whether [key] is being saved.
  bool isSaving(String key) => _saving.contains(key);

  /// Sets [key] to [newValue] and reloads the configuration.
  ///
  /// Throws a `VfoxException` if vfox rejects the value; the configuration
  /// is reloaded either way.
  Future<void> set(String key, String newValue) => _save(key, () => _cli.setConfig(key, newValue));

  /// Resets [key] to its default and reloads the configuration.
  Future<void> reset(String key) => _save(key, () => _cli.unsetConfig(key));

  Future<void> _save(String key, Future<void> Function() action) async {
    _saving.add(key);
    notifyListeners();
    try {
      await action();
    } finally {
      _saving.remove(key);
      await load();
    }
  }
}
