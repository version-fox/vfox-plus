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
import '../core/vfox/vfox_cli.dart';
import '../core/vfox/vfox_command.dart';
import '../core/vfox/vfox_locator.dart';

/// Whether vfox can be used.
enum VfoxAvailability {
  /// Not checked yet.
  unknown,

  /// The executable is being probed.
  checking,

  /// vfox runs and reported its version.
  ready,

  /// No vfox executable was found.
  notFound,

  /// An executable was found but did not behave like vfox.
  broken,
}

/// Locates vfox, configures the [VfoxCli] and checks that vfox works.
class VfoxController extends ChangeNotifier {
  /// Creates a controller.
  VfoxController({required this._cli, required this._locator, required this._environment});

  final VfoxCli _cli;
  final VfoxLocator _locator;
  final Map<String, String> _environment;

  VfoxAvailability _availability = VfoxAvailability.unknown;

  /// Whether vfox can be used.
  VfoxAvailability get availability => _availability;

  String? _executable;

  /// The executable in use, if one was found.
  String? get executable => _executable;

  bool _isCustomExecutable = false;

  /// Whether [executable] was chosen by the user rather than detected.
  bool get isCustomExecutable => _isCustomExecutable;

  String? _version;

  /// The vfox version, once [availability] is [VfoxAvailability.ready].
  String? get version => _version;

  Object? _error;

  /// Why vfox is not available, if it is not.
  Object? get error => _error;

  /// Points the CLI at the executable and environment from [settings] and
  /// checks the result.
  Future<void> apply(AppSettings settings) async {
    final String? custom = settings.vfoxExecutable;
    _isCustomExecutable = custom != null;
    _executable = custom ?? _locator.locate();
    _cli.launchConfig = _cli.launchConfig.copyWith(
      executable: _executable ?? VfoxLocator.executableName,
      baseEnvironment: _environment,
      extraEnvironment: settings.extraEnvironment,
    );
    await check();
  }

  /// Runs `vfox --version` to check that vfox works.
  Future<void> check() async {
    _availability = VfoxAvailability.checking;
    notifyListeners();
    try {
      _version = await _cli.version();
      _error = null;
      _availability = VfoxAvailability.ready;
    } on VfoxNotFoundException catch (error) {
      _version = null;
      _error = error;
      _availability = VfoxAvailability.notFound;
    } on VfoxException catch (error) {
      _version = null;
      _error = error;
      _availability = VfoxAvailability.broken;
    }
    notifyListeners();
  }
}
