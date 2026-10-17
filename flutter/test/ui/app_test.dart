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

// Widget tests of the whole application against a scripted vfox.

import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:vfox_plus/src/core/process_runner.dart';
import 'package:vfox_plus/src/core/settings/app_settings.dart';
import 'package:vfox_plus/src/state/app_model.dart';
import 'package:vfox_plus/src/state/task_queue.dart';
import 'package:vfox_plus/src/ui/icons/app_icons.g.dart';
import 'package:vfox_plus/src/ui/platform_services.dart';
import 'package:vfox_plus/src/ui/widgets/navigation_rail.dart';

import '../support/app_harness.dart';
import '../support/fakes.dart';

void main() {
  testWidgets('the home page summarises the installed SDKs', (WidgetTester tester) async {
    await pumpApp(tester);
    expect(find.text('vfox+'), findsWidgets);
    expect(find.text('vfox 1.0.13 · /usr/bin/vfox'), findsOneWidget);
    // 3 plugins, 4 installed versions, 2 in use.
    expect(find.text('3'), findsOneWidget);
    expect(find.text('4'), findsOneWidget);
    expect(find.text('2'), findsOneWidget);
    expect(find.text('Versions in use'), findsOneWidget);
    expect(find.text('24.11.1'), findsOneWidget);
  });

  testWidgets('copies the shell activation line', (WidgetTester tester) async {
    final services = FakePlatformServices();
    await pumpApp(tester, services: services);
    await tester.tap(find.byTooltip('Copy'));
    await tester.pumpAndSettle();
    expect(services.copiedTexts, <String>[r'eval "$(vfox activate bash)"']);
    expect(find.text('Copied to the clipboard'), findsOneWidget);
  });

  testWidgets('selecting an SDK shows its versions next to the list', (WidgetTester tester) async {
    final runner = fixtureRunner();
    final AppModel model = await pumpApp(tester, runner: runner);
    await tester.tap(railDestination('SDKs'));
    await tester.pumpAndSettle();
    expect(find.text('Select an SDK to manage its versions.'), findsOneWidget);

    await tester.tap(find.text('nodejs'));
    await tester.pumpAndSettle();
    expect(find.text('Node.js runtime environment.'), findsOneWidget);
    expect(find.text('22.21.0'), findsOneWidget);
    expect(find.text('In use'), findsOneWidget);

    // Make 22.21.0 the global default.
    await tester.tap(find.widgetWithText(InkWell, 'Use'));
    await tester.pumpAndSettle();
    expect(runner.calls('use').single.arguments, <String>['use', '--global', 'nodejs@22.21.0']);
    expect(runner.calls('use').single.environment['__VFOX_SHELL'], 'bash');
    expect(model.tasks.tasks.single.status, TaskStatus.succeeded);
  });

  testWidgets('installs a version from the available tab', (WidgetTester tester) async {
    final runner = fixtureRunner();
    final process = FakeRunningProcess();
    runner.onStart('install --yes maven@4.0.0-rc-7', () => process);
    final AppModel model = await pumpApp(tester, runner: runner);
    await tester.tap(railDestination('SDKs'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('maven'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Available'));
    await tester.pumpAndSettle();
    expect(find.text('5 versions'), findsOneWidget);
    // 3.9.16 is installed already.
    expect(find.widgetWithText(TextButton, 'Installed'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Install').first);
    await tester.pump();
    process.writeStderr('\rDownloading...  42% [==>   ]\r');
    await tester.pump();
    await tester.pump();
    expect(find.text('Downloading… 42%'), findsOneWidget);

    process.exit(0);
    await tester.pumpAndSettle();
    expect(model.tasks.tasks.single.status, TaskStatus.succeeded);
    expect(find.text('Install maven@4.0.0-rc-7: done'), findsOneWidget);
  });

  testWidgets('removing a plugin asks for confirmation', (WidgetTester tester) async {
    final runner = fixtureRunner()..on('remove --yes nodejs', fixtureOutput('remove'));
    await pumpApp(tester, runner: runner);
    await tester.tap(railDestination('SDKs'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('nodejs'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('More actions'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Remove plugin'));
    await tester.pumpAndSettle();
    expect(find.text('Remove nodejs?'), findsOneWidget);
    expect(find.text('The plugin and its 2 installed versions will be deleted.'), findsOneWidget);
    await tester.tap(find.widgetWithText(FilledButton, 'Remove'));
    await tester.pumpAndSettle();
    expect(runner.calls('remove').single.arguments, <String>['remove', '--yes', 'nodejs']);
  });

  testWidgets('the plugin registry adds plugins and marks installed ones', (WidgetTester tester) async {
    final runner = fixtureRunner()..on('add zig', fixtureOutput('add'));
    await pumpApp(tester, runner: runner);
    await tester.tap(railDestination('Plugins'));
    await tester.pumpAndSettle();
    expect(find.text('bun'), findsOneWidget);

    await tester.enterText(find.byType(SearchBar), 'node');
    await tester.pumpAndSettle();
    expect(find.text('bun'), findsNothing);
    expect(find.text('Added'), findsOneWidget);

    await tester.enterText(find.byType(SearchBar), 'zig');
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Add'));
    await tester.pumpAndSettle();
    expect(runner.calls('add').single.arguments, <String>['add', 'zig']);
  });

  testWidgets('failed tasks can be inspected and retried', (WidgetTester tester) async {
    final runner = fixtureRunner()..on('update --all', const ProcessOutput(exitCode: 1, stdout: 'network down\n'));
    final AppModel model = await pumpApp(tester, runner: runner);
    model.run(model.cli.updateAllPlugins());
    await tester.pumpAndSettle();
    expect(find.text('Update all plugins: failed\nnetwork down'), findsOneWidget);

    await tester.tap(find.text('View'));
    await tester.pumpAndSettle();
    expect(find.text('Failed · network down'), findsOneWidget);
    await tester.tap(find.byTooltip('Show output'));
    await tester.pumpAndSettle();
    expect(find.textContaining(r'$ vfox update --all'), findsOneWidget);

    runner.on('update --all', fixtureOutput('update-all'));
    await tester.tap(find.byTooltip('Retry'));
    await tester.pumpAndSettle();
    expect(model.tasks.tasks.first.status, TaskStatus.succeeded);
  });

  testWidgets('switches the language and the theme', (WidgetTester tester) async {
    final AppModel model = await pumpApp(tester);
    await tester.tap(railDestination('Settings'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Dark'));
    await tester.pumpAndSettle();
    expect(Theme.of(tester.element(find.text('Appearance'))).brightness, Brightness.dark);

    await tester.tap(find.text('简体中文'));
    await tester.pumpAndSettle();
    expect(find.text('外观'), findsOneWidget);
    expect(model.settings.settings.language, LanguagePreference.simplifiedChinese);
    expect(model.settings.settings.theme, ThemePreference.dark);
  });

  testWidgets('edits the vfox configuration through vfox config', (WidgetTester tester) async {
    final runner = fixtureRunner()
      ..on('config gitignore.enable false', const ProcessOutput(exitCode: 0))
      ..on('config proxy.url http://127.0.0.1:7890', const ProcessOutput(exitCode: 0));
    await pumpApp(tester, runner: runner);
    await tester.tap(railDestination('Settings'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Add .vfox/ to .gitignore'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Add .vfox/ to .gitignore'));
    await tester.pumpAndSettle();
    expect(runner.calls('config gitignore.enable').single.arguments.last, 'false');

    final Finder proxy = find.widgetWithText(TextField, 'http://127.0.0.1:7890');
    await tester.ensureVisible(proxy);
    await tester.pumpAndSettle();
    await tester.enterText(proxy, 'http://127.0.0.1:7890');
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();
    expect(runner.calls('config proxy.url'), hasLength(1));
  });

  testWidgets('saves extra environment variables for vfox', (WidgetTester tester) async {
    final AppModel model = await pumpApp(tester);
    await tester.tap(railDestination('Settings'));
    await tester.pumpAndSettle();
    final Finder field = find.widgetWithText(TextField, 'VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node');
    await tester.ensureVisible(find.text('Save'));
    await tester.pumpAndSettle();
    await tester.enterText(field, 'not valid');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(find.text('Line 1 is not KEY=VALUE.'), findsOneWidget);

    await tester.enterText(field, 'VFOX_NODEJS_MIRROR=https://mirror');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(model.settings.settings.extraEnvironment, <String, String>{'VFOX_NODEJS_MIRROR': 'https://mirror'});
    expect(model.cli.launchConfig.environment()['VFOX_NODEJS_MIRROR'], 'https://mirror');
  });

  testWidgets('explains how to get vfox when it is missing', (WidgetTester tester) async {
    final runner = fixtureRunner()..missingExecutable = true;
    final services = FakePlatformServices(file: '/opt/vfox');
    final AppModel model = await pumpApp(tester, runner: runner, services: services);
    expect(find.text('vfox is not installed'), findsOneWidget);

    await tester.tap(find.text('Installation guide'));
    expect(services.openedUrls, <Uri>[Links.vfoxInstall]);

    runner.missingExecutable = false;
    await tester.tap(find.text('Locate vfox…'));
    await tester.pumpAndSettle();
    expect(model.settings.settings.vfoxExecutable, '/opt/vfox');
    expect(find.text('vfox is not installed'), findsNothing);
    expect(find.text('Versions in use'), findsOneWidget);
  });

  testWidgets('compact windows use a navigation bar and a single pane', (WidgetTester tester) async {
    await pumpApp(tester, size: const Size(420, 800));
    expect(find.byType(NavigationBar), findsOneWidget);
    expect(find.byType(ExpressiveNavigationRail), findsNothing);

    await tester.tap(find.widgetWithIcon(NavigationDestination, AppIcons.sdks));
    await tester.pumpAndSettle();
    await tester.tap(find.text('nodejs'));
    await tester.pumpAndSettle();
    expect(find.text('Node.js runtime environment.'), findsOneWidget);
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    expect(find.text('Node.js runtime environment.'), findsNothing);
  });

  testWidgets('expanding the rail is remembered', (WidgetTester tester) async {
    final AppModel model = await pumpApp(tester);
    await tester.tap(find.byTooltip('Show labels'));
    await tester.pumpAndSettle();
    expect(model.settings.settings.railExpanded, isTrue);
    expect(tester.getSize(find.byType(ExpressiveNavigationRail)).width, ExpressiveNavigationRail.expandedWidth);
  });
}
