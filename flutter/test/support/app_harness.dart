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

// Pumps the whole application against a scripted vfox.

import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:vfox_plus/src/core/settings/app_settings.dart';
import 'package:vfox_plus/src/core/settings/settings_store.dart';
import 'package:vfox_plus/src/core/vfox/vfox_locator.dart';
import 'package:vfox_plus/src/state/app_model.dart';
import 'package:vfox_plus/src/ui/app.dart';
import 'package:vfox_plus/src/ui/widgets/navigation_rail.dart';

import 'fakes.dart';

/// Pumps the application at [size] with animations disabled, so that the
/// loading indicators stop and `pumpAndSettle` can settle.
Future<AppModel> pumpApp(
  WidgetTester tester, {
  FakeProcessRunner? runner,
  FakePlatformServices? services,
  AppSettings? settings,
  Size size = const Size(1280, 820),
}) async {
  tester.view
    ..physicalSize = size
    ..devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  tester.platformDispatcher.accessibilityFeaturesTestValue = const FakeAccessibilityFeatures(disableAnimations: true);
  addTearDown(tester.platformDispatcher.clearAccessibilityFeaturesTestValue);

  final model = AppModel(
    runner: runner ?? fixtureRunner(),
    settingsStore: MemorySettingsStore(settings ?? AppSettings(language: LanguagePreference.english)),
    environment: const <String, String>{'HOME': '/home/user'},
    locator: VfoxLocator(environment: const <String, String>{}, isExecutable: (String path) => path == '/usr/bin/vfox'),
  );
  addTearDown(model.dispose);
  await tester.pumpWidget(VfoxPlusApp(model: model, services: services ?? FakePlatformServices()));
  await model.start();
  await tester.pumpAndSettle();
  return model;
}

/// The destination [label] of the navigation rail.
Finder railDestination(String label) =>
    find.descendant(of: find.byType(ExpressiveNavigationRail), matching: find.bySemanticsLabel(label));
