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

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/models.dart';
import 'package:vfox_plus/src/core/process_runner.dart';
import 'package:vfox_plus/src/core/settings/app_settings.dart';
import 'package:vfox_plus/src/core/settings/settings_store.dart';
import 'package:vfox_plus/src/core/vfox/vfox_cli.dart';
import 'package:vfox_plus/src/core/vfox/vfox_command.dart';
import 'package:vfox_plus/src/core/vfox/vfox_locator.dart';
import 'package:vfox_plus/src/state/app_model.dart';
import 'package:vfox_plus/src/state/async_resource.dart';
import 'package:vfox_plus/src/state/sdk_detail_controller.dart';
import 'package:vfox_plus/src/state/settings_controller.dart';
import 'package:vfox_plus/src/state/task_queue.dart';
import 'package:vfox_plus/src/state/vfox_config_controller.dart';
import 'package:vfox_plus/src/state/vfox_controller.dart';

import '../support/fakes.dart';

void main() {
  group('AsyncResource', () {
    test('keeps the previous value while reloading and records errors', () async {
      var calls = 0;
      final resource = AsyncResource<int>(() async {
        calls++;
        if (calls == 2) {
          throw StateError('boom');
        }
        return calls;
      });
      await resource.load();
      expect(resource.value, 1);
      expect(resource.hasValue, isTrue);

      final Future<void> reload = resource.load();
      expect(resource.isLoading, isTrue);
      expect(resource.value, 1);
      await reload;
      expect(resource.error, isA<StateError>());
      expect(resource.value, 1);
      expect(resource.isLoading, isFalse);
    });

    test('lets the latest of overlapping loads win', () async {
      final completers = <Completer<String>>[];
      final resource = AsyncResource<String>(() {
        final completer = Completer<String>();
        completers.add(completer);
        return completer.future;
      });
      final Future<void> first = resource.load();
      final Future<void> second = resource.load();
      completers[1].complete('new');
      completers[0].complete('old');
      await Future.wait(<Future<void>>[first, second]);
      expect(resource.value, 'new');
    });

    test('ensureLoaded loads once', () async {
      var calls = 0;
      final resource = AsyncResource<int>(() async => ++calls);
      await resource.ensureLoaded();
      await resource.ensureLoaded();
      expect(calls, 1);
    });
  });

  group('SettingsController', () {
    test('persists changes and skips no-ops', () async {
      final store = MemorySettingsStore(AppSettings(theme: ThemePreference.dark));
      final controller = SettingsController(store);
      await controller.load();
      expect(controller.settings.theme, ThemePreference.dark);
      await controller.update((s) => s.copyWith(theme: ThemePreference.dark));
      expect(store.saveCount, 0);
      await controller.update((s) => s.copyWith(theme: ThemePreference.light));
      expect(store.saveCount, 1);
      expect((await store.load()).theme, ThemePreference.light);
    });
  });

  group('VfoxController', () {
    late FakeProcessRunner runner;
    late VfoxCli cli;

    setUp(() {
      runner = fixtureRunner();
      cli = VfoxCli(
        runner: runner,
        launchConfig: const VfoxLaunchConfig(executable: 'vfox'),
      );
    });

    VfoxController controller({String? found}) => VfoxController(
      cli: cli,
      locator: VfoxLocator(environment: const <String, String>{}, isExecutable: (String path) => path == found),
      environment: const <String, String>{'HOME': '/home/u'},
    );

    test('uses the detected executable and reads the version', () async {
      final VfoxController vfox = controller(found: '/usr/local/bin/vfox');
      await vfox.apply(AppSettings(extraEnvironment: const <String, String>{'A': '1'}));
      expect(vfox.availability, VfoxAvailability.ready);
      expect(vfox.version, '1.0.12');
      expect(vfox.executable, '/usr/local/bin/vfox');
      expect(vfox.isCustomExecutable, isFalse);
      expect(cli.launchConfig.extraEnvironment, <String, String>{'A': '1'});
      expect(runner.invocations.single.executable, '/usr/local/bin/vfox');
    });

    test('prefers the executable chosen by the user', () async {
      final VfoxController vfox = controller(found: '/usr/local/bin/vfox');
      await vfox.apply(AppSettings(vfoxExecutable: '/opt/vfox'));
      expect(vfox.executable, '/opt/vfox');
      expect(vfox.isCustomExecutable, isTrue);
    });

    test('reports a missing and a broken executable', () async {
      final VfoxController vfox = controller();
      runner.missingExecutable = true;
      await vfox.apply(AppSettings());
      expect(vfox.availability, VfoxAvailability.notFound);

      runner
        ..missingExecutable = false
        ..on('--version', const ProcessOutput(exitCode: 0, stdout: 'hello'));
      await vfox.check();
      expect(vfox.availability, VfoxAvailability.broken);
    });
  });

  group('SdkDetailController', () {
    test('filters available versions by text, note and installation', () async {
      final runner = fixtureRunner();
      final controller = SdkDetailController(
        cli: VfoxCli(
          runner: runner,
          launchConfig: const VfoxLaunchConfig(executable: 'vfox'),
        ),
        name: 'nodejs',
      );
      addTearDown(controller.dispose);
      await controller.versions.load();
      const installed = <String>{'24.11.1'};
      expect(controller.visibleVersions(installed: installed), hasLength(872));

      controller.filter = VersionFilter.withNote;
      expect(controller.visibleVersions(installed: installed).every((AvailableVersion v) => v.note == 'LTS'), isTrue);

      controller
        ..filter = VersionFilter.all
        ..query = '24.11.';
      expect(controller.visibleVersions(installed: installed).map((AvailableVersion v) => v.version), <String>[
        '24.11.1',
        '24.11.0',
      ]);

      controller.filter = VersionFilter.installed;
      expect(controller.visibleVersions(installed: installed).single.version, '24.11.1');
    });
  });

  group('VfoxConfigController', () {
    test('saves and reloads, and exposes values by key', () async {
      final runner = fixtureRunner()..on('config gitignore.enable false', const ProcessOutput(exitCode: 0));
      final controller = VfoxConfigController(
        VfoxCli(
          runner: runner,
          launchConfig: const VfoxLaunchConfig(executable: 'vfox'),
        ),
      );
      await controller.load();
      expect(controller.valueOf('gitignore.enable'), 'true');
      expect(controller.valueOf('missing'), isNull);

      final Future<void> saving = controller.set('gitignore.enable', 'false');
      expect(controller.isSaving('gitignore.enable'), isTrue);
      await saving;
      expect(controller.isSaving('gitignore.enable'), isFalse);
      expect(runner.calls('config --list'), hasLength(2));
    });

    test('rethrows rejected values after reloading', () async {
      final runner = fixtureRunner()
        ..on('config cache.availableHookDuration x', fixtureOutput('config-set-invalid-duration', exitCode: 1));
      final controller = VfoxConfigController(
        VfoxCli(
          runner: runner,
          launchConfig: const VfoxLaunchConfig(executable: 'vfox'),
        ),
      );
      await expectLater(controller.set('cache.availableHookDuration', 'x'), throwsA(isA<VfoxException>()));
      expect(controller.hasValue, isTrue);
    });
  });

  group('AppModel', () {
    late FakeProcessRunner runner;
    late AppModel model;

    setUp(() {
      runner = fixtureRunner();
      model = AppModel(
        runner: runner,
        settingsStore: MemorySettingsStore(),
        environment: const <String, String>{'HOME': '/home/u'},
        locator: VfoxLocator(
          environment: const <String, String>{},
          isExecutable: (String path) => path == '/usr/bin/vfox',
        ),
      );
    });

    tearDown(() => model.dispose());

    test('starts by probing vfox and loading the installed SDKs', () async {
      await model.start();
      await pumpEventQueue();
      expect(model.vfox.availability, VfoxAvailability.ready);
      expect(model.installed.value?.map((InstalledSdk sdk) => sdk.name), <String>['maven', 'mvn2', 'nodejs']);
      expect(model.installedSdk('nodejs')?.current, '24.11.1');
      expect(model.installedSdk('python'), isNull);
    });

    test('reloads the installed SDKs after a task', () async {
      await model.start();
      await pumpEventQueue();
      final int before = runner.calls('list').length;
      model.run(model.cli.use('nodejs', '22.21.0'));
      await pumpEventQueue();
      expect(runner.calls('list').length, before + 1);
      expect(model.tasks.tasks.single.status, TaskStatus.succeeded);
    });

    test('hands vfox a copy of a local .lua plugin and deletes it afterwards', () async {
      final Directory home = Directory.systemTemp.createTempSync('vfox-plus-home');
      addTearDown(() => home.deleteSync(recursive: true));
      final original = File('${home.path}/mine.lua')..writeAsStringSync('PLUGIN = {}');
      final local = AppModel(
        runner: runner,
        settingsStore: MemorySettingsStore(),
        environment: <String, String>{'HOME': home.path},
        locator: VfoxLocator(
          environment: const <String, String>{},
          isExecutable: (String path) => path == '/usr/bin/vfox',
        ),
      );
      addTearDown(local.dispose);
      await local.start();
      await pumpEventQueue();

      final VfoxTask task = await local.addPluginFromSource(original.path, alias: 'mine');
      final List<String> arguments = runner.calls('add').single.arguments;
      final String copy = arguments[arguments.indexOf('--source') + 1];
      // vfox renames the file into its temporary directory, which only works
      // on the same filesystem.
      expect(copy, startsWith('${home.path}/.vfox/tmp/vfox-plus-plugin-'));
      expect(arguments.sublist(arguments.length - 2), <String>['--alias', 'mine']);

      await pumpEventQueue();
      expect(task.status, TaskStatus.succeeded);
      expect(original.existsSync(), isTrue);
      expect(File(copy).existsSync(), isFalse);
    });

    test('passes URLs to vfox as they are', () async {
      await model.start();
      await model.addPluginFromSource('https://example.com/p.zip');
      expect(runner.calls('add').single.arguments, <String>['add', '--source', 'https://example.com/p.zip']);
    });

    test('re-applies the settings when the executable changes', () async {
      await model.start();
      await pumpEventQueue();
      await model.settings.update((s) => s.copyWith(vfoxExecutable: '/opt/vfox'));
      await pumpEventQueue();
      expect(model.vfox.executable, '/opt/vfox');
      expect(runner.invocations.last.executable, '/opt/vfox');
    });
  });
}
