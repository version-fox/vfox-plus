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

import 'dart:async';
import 'dart:io';

import 'package:flutter/widgets.dart';

import 'src/core/process_runner.dart';
import 'src/core/settings/settings_store.dart';
import 'src/state/app_model.dart';
import 'src/ui/app.dart';
import 'src/ui/licenses.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  registerAssetLicenses();
  final model = AppModel(
    runner: const IoProcessRunner(),
    settingsStore: JsonFileSettingsStore.standard(),
    environment: Platform.environment,
  );
  runApp(VfoxPlusApp(model: model));
  unawaited(model.start());
}
