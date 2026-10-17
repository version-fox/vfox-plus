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

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:vfox_plus/src/core/settings/app_settings.dart';

import '../support/app_harness.dart';
import '../support/fakes.dart';

void main() {
  testWidgets('pins a version in a project directory', (WidgetTester tester) async {
    final Directory project = Directory.systemTemp.createTempSync('vfox-project');
    addTearDown(() => project.deleteSync(recursive: true));
    final runner = fixtureRunner();
    await pumpApp(
      tester,
      runner: runner,
      services: FakePlatformServices(directory: project.path),
    );
    await tester.tap(railDestination('SDKs'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('nodejs'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('More ways to use this version'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Use in a project…'));
    await tester.pumpAndSettle();
    expect(find.text('Use nodejs 22.21.0 in a project'), findsOneWidget);

    await tester.tap(find.text('Use in project'));
    await tester.pumpAndSettle();
    expect(find.text('Choose an existing directory.'), findsOneWidget);

    await tester.tap(find.byTooltip('Browse…'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Use in project'));
    await tester.pumpAndSettle();
    final ProcessCall call = runner.calls('use').single;
    expect(call.arguments, <String>['use', '--project', 'nodejs@22.21.0']);
    expect(call.workingDirectory, project.path);
  });

  testWidgets('adds a plugin from a source with an alias', (WidgetTester tester) async {
    final runner = fixtureRunner()..on('add --source https://example.com/p.zip --alias mine', fixtureOutput('add'));
    await pumpApp(tester, runner: runner);
    await tester.tap(railDestination('Plugins'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Add from source…'));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(FilledButton, 'Add plugin'));
    await tester.pumpAndSettle();
    expect(find.text('Enter a URL or a path.'), findsOneWidget);

    await tester.enterText(find.widgetWithText(TextField, 'Source'), 'https://example.com/p.zip');
    await tester.enterText(find.widgetWithText(TextField, 'Name (optional)'), '-bad');
    await tester.tap(find.widgetWithText(FilledButton, 'Add plugin'));
    await tester.pumpAndSettle();
    expect(find.text('Use letters, digits, dots, dashes and underscores only.'), findsOneWidget);

    await tester.enterText(find.widgetWithText(TextField, 'Name (optional)'), 'mine');
    await tester.tap(find.widgetWithText(FilledButton, 'Add plugin'));
    await tester.pumpAndSettle();
    expect(runner.calls('add').single.arguments, <String>[
      'add',
      '--source',
      'https://example.com/p.zip',
      '--alias',
      'mine',
    ]);
  });

  testWidgets('chooses a plugin file with the file chooser', (WidgetTester tester) async {
    final runner = fixtureRunner()..on('add --source /home/u/plugin.zip', fixtureOutput('add'));
    final services = FakePlatformServices(file: '/home/u/plugin.zip');
    await pumpApp(tester, runner: runner, services: services);
    await tester.tap(railDestination('Plugins'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Add from source…'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Browse…'));
    await tester.pumpAndSettle();
    expect(services.pickedExtensions, <String>['zip', 'lua']);
    expect(find.widgetWithText(TextField, '/home/u/plugin.zip'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Add plugin'));
    await tester.pumpAndSettle();
    expect(runner.calls('add').single.arguments, <String>['add', '--source', '/home/u/plugin.zip']);
  });

  testWidgets('uninstalling asks for confirmation and can be cancelled', (WidgetTester tester) async {
    final runner = fixtureRunner();
    await pumpApp(tester, runner: runner);
    await tester.tap(railDestination('SDKs'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('nodejs'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('More actions for 24.11.1'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Uninstall'));
    await tester.pumpAndSettle();
    expect(find.text('Uninstall nodejs 24.11.1?'), findsOneWidget);
    expect(find.textContaining('This version is in use.'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(runner.calls('uninstall'), isEmpty);
  });

  testWidgets('every page renders in Simplified Chinese', (WidgetTester tester) async {
    await pumpApp(tester, settings: AppSettings(language: LanguagePreference.simplifiedChinese));
    expect(find.text('正在使用的版本'), findsOneWidget);
    for (final (String destination, String text) in <(String, String)>[
      ('SDK', '选择一个 SDK 以管理其版本。'),
      ('插件', '每个插件都为 vfox 添加一种语言或工具。'),
      ('任务', '没有正在进行的任务'),
      ('设置', '外观'),
    ]) {
      await tester.tap(railDestination(destination));
      await tester.pumpAndSettle();
      expect(find.text(text), findsOneWidget, reason: destination);
    }
    expect(tester.takeException(), isNull);
  });
}
