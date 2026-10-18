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
// The screenshots show every page, in both themes and languages, the compact
// layout and an installation in progress; the README links a selection.
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
import 'package:vfox_plus/src/state/task_queue.dart';
import 'package:vfox_plus/src/ui/app.dart';
import 'package:vfox_plus/src/ui/widgets/list_group.dart';

import 'support/harness.dart';

final String? _output = Platform.environment['VFOX_PLUS_SCREENSHOTS_DIR'];
final String? _installedVfox = Platform.environment['VFOX_PLUS_SCREENSHOTS_VFOX'];

/// The size of the window of the desktop screenshots.
const Size _desktop = Size(1280, 820);

/// The size of a compact window, like the one of a phone.
const Size _compact = Size(412, 800);

/// A GUI started from a terminal in which vfox is activated, which `use` needs.
const Map<String, String> _activatedShell = <String, String>{'__VFOX_PID': '1', '__VFOX_SHELL': 'bash'};

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  late Directory sandbox;
  late File vfox;
  final GlobalKey screen = GlobalKey();

  setUp(() async {
    sandbox = Directory.systemTemp.createTempSync('vfox-plus-screenshots');
    vfox = await writeExecutable(
      _installedVfox ?? '${sandbox.path}/vfox',
      fakeVfoxScript('${sharedDirectory().path}/fixtures/vfox-1.0.13'),
    );
  });

  tearDown(() => sandbox.deleteSync(recursive: true));

  /// Starts the application with [settings] in a window of [size], with
  /// [environment] added to the one of the test.
  Future<AppModel> launch(
    WidgetTester tester,
    AppSettings settings, {
    Size size = _desktop,
    Map<String, String> environment = const <String, String>{},
  }) async {
    tester.view
      ..physicalSize = size
      ..devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final store = JsonFileSettingsStore(File('${sandbox.path}/settings.json'));
    final AppSettings shown = settings.copyWith(useSystemColor: false);
    // An installed fake vfox is detected; otherwise it is chosen explicitly.
    await store.save(_installedVfox == null ? shown.copyWith(vfoxExecutable: vfox.path) : shown);
    final model = AppModel(
      runner: const IoProcessRunner(),
      settingsStore: store,
      environment: <String, String>{...Platform.environment, ...environment},
    );
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

  /// Opens the SDK nodejs on the SDKs page of an application that shows the
  /// Home page.
  Future<void> openNodejs(WidgetTester tester) async {
    await pumpUntil(tester, find.text('Versions in use'));
    await tester.tap(railDestination('SDKs'));
    await pumpUntil(tester, find.text('nodejs'));
    await tester.tap(find.text('nodejs'));
    await pumpUntil(tester, find.text('Node.js runtime environment.'));
  }

  /// Opens the available versions of the SDK that is open.
  Future<void> openAvailable(WidgetTester tester) async {
    await tester.tap(find.text('Available'));
    await pumpUntil(tester, find.text('872 versions'));
  }

  /// Opens the Plugins page of an application that shows the Home page.
  Future<void> openPlugins(WidgetTester tester) async {
    await pumpUntil(tester, find.text('Versions in use'));
    await tester.tap(railDestination('Plugins'));
    await pumpUntil(tester, find.text('dotnet'));
  }

  testWidgets('home page, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(
      tester,
      AppSettings(language: LanguagePreference.english, theme: ThemePreference.light, railExpanded: true),
    );
    await pumpUntil(tester, find.text('Versions in use'));
    await capture(tester, 'home-light');
  });

  testWidgets('home page, English, dark', skip: _output == null, (WidgetTester tester) async {
    await launch(
      tester,
      AppSettings(language: LanguagePreference.english, theme: ThemePreference.dark, railExpanded: true),
    );
    await pumpUntil(tester, find.text('Versions in use'));
    await capture(tester, 'home-dark');
  });

  testWidgets('home page, Simplified Chinese, light', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.simplifiedChinese, theme: ThemePreference.light));
    await pumpUntil(tester, find.text('正在使用的版本'));
    await capture(tester, 'home-zh');
  });

  testWidgets('SDK versions, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.english, theme: ThemePreference.light));
    await openNodejs(tester);
    await capture(tester, 'sdks-light');
  });

  testWidgets('version menu, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.english, theme: ThemePreference.light));
    await openNodejs(tester);
    await tester.tap(find.byTooltip('More actions for 22.21.0'));
    await pumpUntil(tester, find.text('Copy installation path'));
    await capture(tester, 'menu-light');
  });

  testWidgets('SDK versions, English, dark', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.english, theme: ThemePreference.dark));
    await openNodejs(tester);
    await openAvailable(tester);
    await capture(tester, 'sdks-dark');
  });

  testWidgets('installation in progress, English, light', skip: _output == null, (WidgetTester tester) async {
    // Every step of the installation takes 4 seconds, long enough to look at it.
    final AppModel model = await launch(
      tester,
      AppSettings(language: LanguagePreference.english, theme: ThemePreference.light),
      environment: const <String, String>{'VFOX_PLUS_FAKE_DELAY': '4'},
    );
    await openNodejs(tester);
    await openAvailable(tester);
    final Finder row = find.ancestor(of: find.text('26.12.0'), matching: find.byType(ListGroupItem));
    await tester.tap(find.descendant(of: row, matching: find.widgetWithText(FilledButton, 'Install')));
    await pumpUntilTrue(
      tester,
      () => (model.tasks.tasks.firstOrNull?.progress ?? 0) >= 0.45,
      reason: 'the download is under way',
    );
    await capture(tester, 'installing-light');
    model.tasks.cancel(model.tasks.tasks.first);
    await pumpFor(tester, const Duration(milliseconds: 500));
  });

  testWidgets('tasks, English, light', skip: _output == null, (WidgetTester tester) async {
    // Every step of an installation takes 3 seconds, so that one installation is
    // under way and another is waiting when the page is shown.
    final AppModel model = await launch(
      tester,
      AppSettings(language: LanguagePreference.english, theme: ThemePreference.light),
      environment: <String, String>{..._activatedShell, 'VFOX_PLUS_FAKE_DELAY': '3'},
    );
    await openNodejs(tester);
    // Select 22.21.0 as the global default, then install three other versions.
    await tester.tap(find.widgetWithText(InkWell, 'Use'));
    await pumpUntil(tester, find.text('Use nodejs@22.21.0: done'));
    await openAvailable(tester);
    Future<void> install(String version) async {
      final Finder row = find.ancestor(of: find.text(version), matching: find.byType(ListGroupItem));
      await tester.tap(find.descendant(of: row, matching: find.widgetWithText(FilledButton, 'Install')));
    }

    await install('26.10.0');
    await pumpUntil(tester, find.text('Install nodejs@26.10.0: done'));
    await install('26.12.0');
    await install('26.11.1');
    await pumpUntilTrue(
      tester,
      () => model.tasks.tasks.any((VfoxTask task) => task.status == TaskStatus.running && (task.progress ?? 0) >= 0.45),
      reason: 'the second installation is under way',
    );
    await tester.tap(railDestination('Tasks'));
    await pumpUntil(tester, find.text('Clear finished'));
    // The output of the installation under way, the second task from the top.
    await tester.tap(find.byTooltip('Show output').at(1));
    await capture(tester, 'tasks-light');
    for (final VfoxTask task in model.tasks.tasks) {
      model.tasks.cancel(task);
    }
    await pumpFor(tester, const Duration(milliseconds: 500));
  });

  testWidgets('plugins, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.english, theme: ThemePreference.light));
    await openPlugins(tester);
    await capture(tester, 'plugins-light');
  });

  testWidgets('add a plugin from source, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.english, theme: ThemePreference.light));
    await openPlugins(tester);
    await tester.tap(find.text('Add from source…'));
    await pumpUntil(tester, find.text('Add a plugin from source'));
    await capture(tester, 'plugin-source-light');
  });

  testWidgets('settings, Simplified Chinese, light', skip: _output == null, (WidgetTester tester) async {
    await launch(tester, AppSettings(language: LanguagePreference.simplifiedChinese, theme: ThemePreference.light));
    await pumpUntil(tester, find.text('正在使用的版本'));
    await tester.tap(railDestination('设置'));
    await pumpUntil(tester, find.text('外观'));
    await capture(tester, 'settings-zh');
  });

  testWidgets('compact home page, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(
      tester,
      AppSettings(language: LanguagePreference.english, theme: ThemePreference.light),
      size: _compact,
    );
    await pumpUntil(tester, find.text('Versions in use'));
    await capture(tester, 'compact-home');
  });

  testWidgets('compact SDK versions, English, light', skip: _output == null, (WidgetTester tester) async {
    await launch(
      tester,
      AppSettings(language: LanguagePreference.english, theme: ThemePreference.light),
      size: _compact,
    );
    await pumpUntil(tester, find.text('Versions in use'));
    // A compact window shows one SDK at a time: the one that the Home page leads to.
    await tester.tap(find.text('nodejs'));
    await pumpUntil(tester, find.text('Node.js runtime environment.'));
    await capture(tester, 'compact-sdks');
  });
}
