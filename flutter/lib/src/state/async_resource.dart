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

/// A value that is loaded asynchronously and can be reloaded.
///
/// While a reload is in flight the previous [value] stays available, so the
/// UI can keep showing stale data with a progress indicator instead of
/// flashing an empty state. When loads overlap, only the latest one wins.
class AsyncResource<T> extends ChangeNotifier {
  /// Creates a resource that obtains its value from `loader`.
  AsyncResource(this._loader);

  final Future<T> Function() _loader;

  T? _value;
  bool _hasValue = false;
  Object? _error;
  bool _loading = false;
  int _generation = 0;
  bool _disposed = false;
  Future<void>? _inFlight;

  /// The last successfully loaded value.
  T? get value => _value;

  /// Whether a value was loaded at least once.
  bool get hasValue => _hasValue;

  /// The error of the last load, or `null` if it succeeded.
  Object? get error => _error;

  /// Whether a load is in progress.
  bool get isLoading => _loading;

  /// Loads the value, replacing any load that is still in progress.
  Future<void> load() {
    final int generation = ++_generation;
    _loading = true;
    _notify();
    return _inFlight = _run(generation);
  }

  /// Loads the value unless it is already loaded or loading.
  Future<void> ensureLoaded() {
    if (_loading) {
      return _inFlight ?? Future<void>.value();
    }
    return _hasValue ? Future<void>.value() : load();
  }

  /// Replaces the value without loading, for example after an optimistic
  /// update.
  void setValue(T value) {
    _generation++;
    _value = value;
    _hasValue = true;
    _error = null;
    _loading = false;
    _notify();
  }

  Future<void> _run(int generation) async {
    try {
      final T value = await _loader();
      if (generation != _generation) {
        return;
      }
      _value = value;
      _hasValue = true;
      _error = null;
    } on Object catch (error) {
      if (generation != _generation) {
        return;
      }
      _error = error;
    }
    _loading = false;
    _notify();
  }

  void _notify() {
    if (!_disposed) {
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
