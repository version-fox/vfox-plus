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

// End-to-end test of the Linux application against a fake vfox.
//
// The real application, with its real process runner, settings file and
// desktop plugins, talks to a shell script that replays the captured outputs
// in shared/fixtures and checks the environment required by the CLI contract.
// It needs no network and no vfox. Run it with a display, for example:
//
//   xvfb-run -a flutter test integration_test/fake_vfox_test.dart -d linux

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:vfox_plus/src/core/process_runner.dart';
import 'package:vfox_plus/src/core/settings/app_settings.dart';
import 'package:vfox_plus/src/core/settings/settings_store.dart';
import 'package:vfox_plus/src/state/app_model.dart';
import 'package:vfox_plus/src/state/task_queue.dart';
import 'package:vfox_plus/src/ui/app.dart';

import 'support/harness.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  late Directory sandbox;
  late AppModel model;

  setUp(() async {
    sandbox = Directory.systemTemp.createTempSync('vfox-plus-fake-e2e');
    final String fixtures = '${sharedDirectory().path}/fixtures/vfox-1.0.12';
    final File vfox = await writeExecutable('${sandbox.path}/vfox', fakeVfoxScript(fixtures));
    final settings = JsonFileSettingsStore(File('${sandbox.path}/config/vfox-plus/settings.json'));
    await settings.save(AppSettings(language: LanguagePreference.english, vfoxExecutable: vfox.path));
    model = AppModel(
      runner: const IoProcessRunner(),
      settingsStore: settings,
      // Simulate a GUI started from a terminal in which vfox is activated.
      environment: <String, String>{...Platform.environment, '__VFOX_PID': '1', '__VFOX_SHELL': 'bash'},
    );
  });

  tearDown(() {
    model.dispose();
    sandbox.deleteSync(recursive: true);
  });

  testWidgets('installs and selects a version end to end', (WidgetTester tester) async {
    tester.view
      ..physicalSize = const Size(1280, 820)
      ..devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(VfoxPlusApp(model: model));
    await model.start();
    await pumpUntil(tester, find.text('Versions in use'));
    expect(find.text('vfox 1.0.12 · ${sandbox.path}/vfox'), findsOneWidget);

    await tester.tap(railDestination('SDKs'));
    await pumpUntil(tester, find.text('nodejs'));
    await tester.tap(find.text('nodejs'));
    await pumpUntil(tester, find.text('Node.js runtime environment.'));

    // Select 22.21.0 as the global default.
    await tester.tap(find.widgetWithText(InkWell, 'Use'));
    await pumpUntil(tester, find.text('Use nodejs@22.21.0: done'));

    // Install a version from the available tab.
    await tester.tap(find.text('Available'));
    await pumpUntil(tester, find.text('872 versions'));
    await tester.enterText(find.widgetWithText(TextField, 'Filter versions'), '24.11.0');
    await pumpUntil(tester, find.text('1 version'));
    await tester.tap(find.widgetWithText(FilledButton, 'Install'));
    await pumpUntil(tester, find.text('Install nodejs@24.11.0: done'));

    final List<VfoxTask> tasks = model.tasks.tasks;
    expect(tasks.map((VfoxTask task) => task.status), everyElement(TaskStatus.succeeded));
    expect(tasks.first.progress, 1);
    expect(tasks.first.log, contains('Install nodejs@24.11.0 success!'));
  });
}
