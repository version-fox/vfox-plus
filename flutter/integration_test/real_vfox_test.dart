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

// End-to-end test of the Linux application against the real vfox.
//
// The test drives every operation of the application, including cancelling an
// installation, through the real vfox executable named by the
// VFOX_PLUS_E2E_VFOX environment variable; it is skipped when the variable is
// not set. vfox runs with HOME set to a new temporary directory, and the
// offline test plugin of shared/test-plugin downloads its SDK from a local
// server, so the test needs no network and never touches an existing vfox
// installation. Run it with a display, for example:
//
//   VFOX_PLUS_E2E_VFOX=/usr/local/bin/vfox xvfb-run -a flutter test integration_test/real_vfox_test.dart -d linux

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
import 'support/test_server.dart';

final String? _vfox = Platform.environment['VFOX_PLUS_E2E_VFOX'];

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  late Directory sandbox;
  late String home;
  late Map<String, String> environment;
  late TestServer server;
  late AppModel model;

  /// Runs vfox in the sandbox, to check what the application did.
  Future<String> vfox(List<String> arguments) async {
    final ProcessResult result = await Process.run(
      _vfox!,
      arguments,
      environment: <String, String>{...environment, 'NO_COLOR': '1'},
      includeParentEnvironment: false,
      workingDirectory: '/',
    );
    return '${result.stdout}';
  }

  setUp(() async {
    sandbox = Directory.systemTemp.createTempSync('vfox-plus-real-e2e');
    home = (Directory('${sandbox.path}/home')..createSync()).path;
    server = await TestServer.start(
      plugin: File('${sharedDirectory().path}/test-plugin/testsdk.lua'),
      workDirectory: Directory('${sandbox.path}/server')..createSync(),
    );
    // Only the sandbox: vfox keeps its state in HOME (or in VFOX_HOME, which
    // must therefore not leak in from the environment of the test run).
    environment = <String, String>{...Platform.environment, 'HOME': home, 'VFOX_PLUS_TEST_SERVER': server.url}
      ..remove('VFOX_HOME');
    final settings = JsonFileSettingsStore(File('${sandbox.path}/config/vfox-plus/settings.json'));
    await settings.save(AppSettings(language: LanguagePreference.english, vfoxExecutable: _vfox));
    model = AppModel(runner: const IoProcessRunner(), settingsStore: settings, environment: environment);
  });

  tearDown(() async {
    model.dispose();
    await server.close();
    sandbox.deleteSync(recursive: true);
  });

  testWidgets('manages plugins and versions with the real vfox', skip: _vfox == null, (WidgetTester tester) async {
    tester.view
      ..physicalSize = const Size(1280, 820)
      ..devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(VfoxPlusApp(model: model));
    await model.start();
    await pumpUntil(tester, find.text('Versions in use'));
    expect(model.installed.value, isEmpty);

    // Point vfox to the local registry through the vfox configuration editor.
    await tester.tap(railDestination('Settings'));
    final Finder registry = find.widgetWithText(TextField, 'Official registry');
    await pumpUntil(tester, registry);
    await tester.ensureVisible(registry);
    await tester.enterText(registry, server.registry);
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await pumpUntil(tester, find.text('Saved'));
    expect((await vfox(<String>['config', 'registry.address'])).trim(), server.registry);

    // Add the plugin from the registry.
    await tester.tap(railDestination('Plugins'));
    await pumpUntil(tester, find.text('testsdk'));
    await tester.tap(find.widgetWithText(FilledButton, 'Add'));
    await pumpUntil(tester, find.text('Add plugin testsdk: done'));
    await pumpUntil(tester, find.text('Added'));
    expect(server.requests, containsAll(<String>['/registry/testsdk.json', '/plugin/testsdk.lua']));

    // Add a local copy of the plugin under another name. vfox moves local .lua
    // files away, so the application must hand it a copy.
    final File localPlugin = File('${sandbox.path}/my-plugin.lua')
      ..writeAsStringSync(File('${sharedDirectory().path}/test-plugin/testsdk.lua').readAsStringSync());
    await tester.tap(find.text('Add from source…'));
    await pumpUntil(tester, find.text('Add a plugin from source'));
    await tester.enterText(find.widgetWithText(TextField, 'Source'), localPlugin.path);
    await tester.enterText(find.widgetWithText(TextField, 'Name (optional)'), 'localsdk');
    await tester.tap(find.widgetWithText(FilledButton, 'Add plugin'));
    await pumpUntil(tester, find.text('Add plugin localsdk: done'));
    expect(localPlugin.existsSync(), isTrue, reason: 'The source file must survive vfox add --source.');

    // Search and install both versions.
    await tester.tap(railDestination('SDKs'));
    await pumpUntil(tester, find.text('testsdk'));
    await tester.tap(find.text('testsdk'));
    await pumpUntil(tester, find.text('An offline SDK for the vfox-plus end-to-end tests.'));
    await tester.tap(find.text('Available'));
    await pumpUntil(tester, find.text('2 versions'));
    expect(find.text('LTS'), findsOneWidget);
    expect(find.text('testtool 2.0'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Install').first);
    await pumpUntil(tester, find.text('Install testsdk@2.0.0: done'));
    expect(model.tasks.tasks.first.progress, 1, reason: 'vfox reports the download progress on standard error.');

    // Cancel the installation of 1.0.0 during the download. From 1.0.13 vfox
    // leaves the partial installation behind, but must neither list it nor
    // refuse to install the version again.
    server.holdDownloads();
    await tester.tap(find.widgetWithText(FilledButton, 'Install'));
    await pumpUntilTrue(tester, () => (model.tasks.tasks.first.progress ?? 0) > 0, reason: 'the download started');
    await tester.tap(find.byTooltip('Cancel'));
    await pumpUntilTrue(tester, () => model.tasks.tasks.first.finishedAt != null, reason: 'vfox exited');
    expect(model.tasks.tasks.first.status, TaskStatus.cancelled);
    expect(await vfox(<String>['list', 'testsdk']), isNot(contains('1.0.0')));
    server.releaseDownloads();
    await pumpUntil(tester, find.widgetWithText(FilledButton, 'Install'));
    await tester.tap(find.widgetWithText(FilledButton, 'Install'));
    await pumpUntil(tester, find.text('Install testsdk@1.0.0: done'));

    // Make 2.0.0 the global default.
    await tester.tap(find.text('Installed (2)'));
    await pumpUntil(tester, find.text('1.0.0'));
    await tester.tap(find.widgetWithText(InkWell, 'Use').first);
    await pumpUntil(tester, find.text('Use testsdk@2.0.0: done'));
    await pumpUntil(tester, find.text('In use: 2.0.0'));
    expect(File('$home/.vfox/.vfox.toml').readAsStringSync(), contains('testsdk = "2.0.0"'));
    expect(Link('$home/.vfox/sdks/testsdk').existsSync(), isTrue);

    // Uninstall 1.0.0.
    await tester.tap(find.byTooltip('More actions for 1.0.0'));
    await pumpUntil(tester, find.text('Uninstall'));
    await tester.tap(find.text('Uninstall'));
    await pumpUntil(tester, find.text('Uninstall testsdk 1.0.0?'));
    await tester.tap(find.widgetWithText(FilledButton, 'Uninstall'));
    await pumpUntil(tester, find.text('Uninstall testsdk@1.0.0: done'));
    await pumpUntil(tester, find.text('Installed (1)'));

    // Remove the second plugin.
    await tester.tap(find.text('localsdk'));
    await pumpUntil(tester, find.byTooltip('More actions'));
    await tester.tap(find.byTooltip('More actions'));
    await pumpUntil(tester, find.text('Remove plugin'));
    await tester.tap(find.text('Remove plugin'));
    await pumpUntil(tester, find.text('Remove localsdk?'));
    await tester.tap(find.widgetWithText(FilledButton, 'Remove'));
    await pumpUntil(tester, find.text('Remove plugin localsdk: done'));
    await pumpUntil(tester, find.text('Select an SDK to manage its versions.'));
    expect(find.text('localsdk'), findsNothing);

    // Every task but the cancelled one succeeded, and vfox agrees with what the
    // application shows.
    expect(
      model.tasks.tasks
          .map((VfoxTask task) => task.status)
          .where((TaskStatus status) => status != TaskStatus.succeeded),
      <TaskStatus>[TaskStatus.cancelled],
    );
    expect(
      Directory('$home/.vfox/tmp')
          .listSync()
          .where((FileSystemEntity entry) => entry.path.split('/').last.startsWith('vfox-plus-plugin-')),
      isEmpty,
      reason: 'The copy of the local plugin is deleted.',
    );
    expect(await vfox(<String>['current']), 'testsdk -> v2.0.0\n');
    expect(await vfox(<String>['list', 'testsdk']), contains('2.0.0'));
  });
}
