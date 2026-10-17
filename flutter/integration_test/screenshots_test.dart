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

// Renders the screenshots of README.md. Images are not committed: they are
// uploaded to an image host and linked, see CONTRIBUTING.md.
//
// The application runs against the fake vfox that replays shared/fixtures, so
// the screenshots show realistic data and are reproducible. The images are
// written to the directory in VFOX_PLUS_SCREENSHOTS_DIR; the test is skipped
// when it is not set. The fake vfox is written to VFOX_PLUS_SCREENSHOTS_VFOX,
// a writable file on the PATH, where the application finds it like an
// installed vfox, or else to a temporary directory.
// `flutter/sandbox/sandbox.sh screenshots` runs it with the fonts of Ubuntu.

import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:vfox_plus/src/core/process_runner.dart';
import 'package:vfox_plus/src/core/settings/app_settings.dart';
import 'package:vfox_plus/src/core/settings/settings_store.dart';
import 'package:vfox_plus/src/state/app_model.dart';
import 'package:vfox_plus/src/ui/app.dart';

import 'support/harness.dart';

final String? _output = Platform.environment['VFOX_PLUS_SCREENSHOTS_DIR'];
final String? _installedVfox = Platform.environment['VFOX_PLUS_SCREENSHOTS_VFOX'];

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  late Directory sandbox;
  late File vfox;
  final GlobalKey screen = GlobalKey();

  setUp(() async {
    sandbox = Directory.systemTemp.createTempSync('vfox-plus-screenshots');
    vfox = await writeExecutable(
      _installedVfox ?? '${sandbox.path}/vfox',
      fakeVfoxScript('${sharedDirectory().path}/fixtures/vfox-1.0.12'),
    );
  });

  tearDown(() => sandbox.deleteSync(recursive: true));

  /// Starts the application with [settings] in a 1280 × 820 window.
  Future<AppModel> launch(WidgetTester tester, AppSettings settings) async {
    tester.view
      ..physicalSize = const Size(1280, 820)
      ..devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final store = JsonFileSettingsStore(File('${sandbox.path}/settings.json'));
    final AppSettings shown = settings.copyWith(useSystemColor: false);
    // An installed fake vfox is detected; otherwise it is chosen explicitly.
    await store.save(_installedVfox == null ? shown.copyWith(vfoxExecutable: vfox.path) : shown);
    final model = AppModel(runner: const IoProcessRunner(), settingsStore: store, environment: Platform.environment);
    addTearDown(model.dispose);
    await tester.pumpWidget(
      RepaintBoundary(
        key: screen,
        child: VfoxPlusApp(model: model),
      ),
    );
    await model.start();
    return model;
  }

  /// Saves what is on screen as `<name>.png`.
  Future<void> capture(WidgetTester tester, String name) async {
    // Let springs, transitions and ripples settle first.
    await pumpFor(tester, const Duration(seconds: 2));
    final boundary = screen.currentContext!.findRenderObject()! as RenderRepaintBoundary;
    final ui.Image image = await boundary.toImage();
    final png = await image.toByteData(format: ui.ImageByteFormat.png);
    final file = File('$_output/$name.png')..createSync(recursive: true);
    await file.writeAsBytes(png!.buffer.asUint8List());
  }

  testWidgets('home page, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(
      tester,
      AppSettings(language: LanguagePreference.english, theme: ThemePreference.light, railExpanded: true),
    );
    await pumpUntil(tester, find.text('Versions in use'));
    await capture(tester, 'home-light');
  });

  testWidgets('SDK versions, English, dark', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.english, theme: ThemePreference.dark));
    await pumpUntil(tester, find.text('Versions in use'));
    await tester.tap(railDestination('SDKs'));
    await pumpUntil(tester, find.text('nodejs'));
    await tester.tap(find.text('nodejs'));
    await pumpUntil(tester, find.text('Node.js runtime environment.'));
    await tester.tap(find.text('Available'));
    await pumpUntil(tester, find.text('872 versions'));
    await capture(tester, 'sdks-dark');
  });

  testWidgets('settings, Simplified Chinese, light', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.simplifiedChinese, theme: ThemePreference.light));
    await pumpUntil(tester, find.text('正在使用的版本'));
    await tester.tap(railDestination('设置'));
    await pumpUntil(tester, find.text('外观'));
    await capture(tester, 'settings-zh');
  });
}
