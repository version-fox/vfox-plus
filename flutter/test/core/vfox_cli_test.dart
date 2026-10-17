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

import 'package:flutter_test/flutter_test.dart';
import 'package:vfox_plus/src/core/models.dart';
import 'package:vfox_plus/src/core/process_runner.dart';
import 'package:vfox_plus/src/core/vfox/vfox_cli.dart';
import 'package:vfox_plus/src/core/vfox/vfox_command.dart';

import '../support/fakes.dart';

void main() {
  late FakeProcessRunner runner;
  late VfoxCli cli;

  setUp(() {
    runner = fixtureRunner();
    cli = VfoxCli(
      runner: runner,
      launchConfig: const VfoxLaunchConfig(
        executable: '/usr/local/bin/vfox',
        baseEnvironment: <String, String>{
          'HOME': '/home/user',
          'PATH': '/usr/bin',
          '__VFOX_SHELL': 'bash',
          '__VFOX_PID': '42',
          '__VFOX_CURTMPPATH': '/home/user/.vfox/tmp/x',
        },
        extraEnvironment: <String, String>{'VFOX_NODEJS_MIRROR': 'https://mirror', 'PATH': '/opt/bin'},
      ),
    );
  });

  group('launch configuration', () {
    test('removes the variables of an activated shell and disables colours', () {
      final Map<String, String> env = cli.launchConfig.environment();
      expect(env, isNot(contains('__VFOX_SHELL')));
      expect(env, isNot(contains('__VFOX_PID')));
      expect(env, isNot(contains('__VFOX_CURTMPPATH')));
      expect(env['NO_COLOR'], '1');
      expect(env['HOME'], '/home/user');
    });

    test('lets user variables override inherited ones', () {
      final Map<String, String> env = cli.launchConfig.environment();
      expect(env['PATH'], '/opt/bin');
      expect(env['VFOX_NODEJS_MIRROR'], 'https://mirror');
    });

    test('marks the hook environment only when asked to', () {
      expect(cli.launchConfig.environment(hookShell: true)['__VFOX_SHELL'], 'bash');
    });
  });

  group('queries', () {
    test('run in the neutral directory with the configured executable', () async {
      await cli.version();
      final ProcessCall call = runner.invocations.single;
      expect(call.executable, '/usr/local/bin/vfox');
      expect(call.workingDirectory, '/');
      expect(call.environment['NO_COLOR'], '1');
    });

    test('version', () async {
      expect(await cli.version(), '1.0.12');
    });

    test('installedSdks merges list and current', () async {
      expect(await cli.installedSdks(), <InstalledSdk>[
        InstalledSdk(name: 'maven', versions: const <String>['3.9.16', '3.8.9'], current: '3.9.16'),
        InstalledSdk(name: 'mvn2', versions: const <String>[]),
        InstalledSdk(name: 'nodejs', versions: const <String>['24.11.1', '22.21.0'], current: '24.11.1'),
      ]);
    });

    test('installedSdks is empty when no plugin is installed', () async {
      runner
        ..on('list', fixtureOutput('list-empty', exitCode: 1))
        ..on('current', fixtureOutput('current-empty'));
      expect(await cli.installedSdks(), isEmpty);
    });

    test('installedSdks reports a failing list', () async {
      runner.on('list', const ProcessOutput(exitCode: 1, stdout: 'boom\n'));
      expect(cli.installedSdks(), throwsA(isA<VfoxException>().having((e) => e.message, 'message', 'boom')));
    });

    test('pluginInfo uses the format template', () async {
      final PluginInfo info = await cli.pluginInfo('nodejs');
      expect(info.version, '0.5.0');
      expect(runner.invocations.single.arguments.take(3), <String>['info', 'nodejs', '--format']);
    });

    test('searchVersions', () async {
      final List<AvailableVersion> versions = await cli.searchVersions('maven');
      expect(versions.map((AvailableVersion v) => v.version), contains('3.9.16'));
    });

    test('availablePlugins', () async {
      expect(await cli.availablePlugins(), hasLength(43));
    });

    test('configuration is listed, set and reset', () async {
      runner
        ..on('config proxy.url http://proxy:8080', const ProcessOutput(exitCode: 0))
        ..on('config --unset proxy.url', const ProcessOutput(exitCode: 0));
      expect(await cli.configEntries(), hasLength(10));
      await cli.setConfig('proxy.url', 'http://proxy:8080');
      await cli.unsetConfig('proxy.url');
      expect(runner.calls('config').map((ProcessCall c) => c.arguments.join(' ')), <String>[
        'config --list',
        'config proxy.url http://proxy:8080',
        'config --unset proxy.url',
      ]);
    });

    test('a rejected value becomes a VfoxException with vfox message', () async {
      runner.on(
        'config cache.availableHookDuration notaduration',
        fixtureOutput('config-set-invalid-duration', exitCode: 1),
      );
      expect(
        cli.setConfig('cache.availableHookDuration', 'notaduration'),
        throwsA(isA<VfoxException>().having((e) => e.message, 'message', 'time: invalid duration "notaduration"')),
      );
    });

    test('installPath', () async {
      runner
        ..on('info nodejs@24.11.1', fixtureOutput('info-version-path'))
        ..on('info nodejs@1.0.0', fixtureOutput('info-version-not-installed'));
      expect(await cli.installPath('nodejs', '24.11.1'), '/tmp/vfox-sb/home/.vfox/cache/nodejs/v-24.11.1');
      expect(await cli.installPath('nodejs', '1.0.0'), isNull);
    });

    test('a missing executable becomes a VfoxNotFoundException', () async {
      runner.missingExecutable = true;
      expect(cli.version(), throwsA(isA<VfoxNotFoundException>()));
    });
  });

  group('commands', () {
    test('install answers prompts with --yes', () {
      final VfoxCommand command = cli.install('nodejs', '24.11.1');
      expect(command.arguments, <String>['install', '--yes', 'nodejs@24.11.1']);
      expect(command.operation, VfoxOperation.install);
      expect(command.subject, 'nodejs@24.11.1');
      expect(command.workingDirectory, isNull);
    });

    test('use in the global scope needs the hook environment', () {
      final VfoxCommand command = cli.use('nodejs', '24.11.1');
      expect(command.arguments, <String>['use', '--global', 'nodejs@24.11.1']);
      expect(command.requiresHookEnvironment, isTrue);
    });

    test('use in a project runs in the project directory', () {
      final VfoxCommand command = cli.use('java', '21', scope: UseScope.project, projectDirectory: '/work/app');
      expect(command.arguments, <String>['use', '--project', 'java@21']);
      expect(command.workingDirectory, '/work/app');
    });

    test('the project scope requires a directory', () {
      expect(() => cli.use('java', '21', scope: UseScope.project), throwsArgumentError);
    });

    test('unuse, uninstall, remove and update', () {
      expect(cli.unuse('java').arguments, <String>['unuse', '--global', 'java']);
      expect(cli.uninstall('java', '21').arguments, <String>['uninstall', 'java@21']);
      expect(cli.removePlugin('java').arguments, <String>['remove', '--yes', 'java']);
      expect(cli.updatePlugin('java').arguments, <String>['update', 'java']);
      expect(cli.updateAllPlugins().arguments, <String>['update', '--all']);
    });

    test('addPlugin from the registry or from a source', () {
      expect(cli.addPlugin('nodejs').arguments, <String>['add', 'nodejs']);
      final VfoxCommand custom = cli.addPlugin('', source: ' https://x/p.zip ', alias: 'mine');
      expect(custom.arguments, <String>['add', '--source', 'https://x/p.zip', '--alias', 'mine']);
      expect(custom.sdk, 'mine');
      expect(() => cli.addPlugin('  '), throwsArgumentError);
    });

    test('rejects names and versions that look like options', () {
      expect(() => cli.install('--all', '1'), throwsArgumentError);
      expect(() => cli.install('java', '-1'), throwsArgumentError);
      expect(() => cli.install('java', '1 2'), throwsArgumentError);
      expect(() => cli.install('java', 'a@b'), throwsArgumentError);
      expect(() => cli.setConfig('proxy..url', 'x'), throwsArgumentError);
    });

    test('start launches the command with the right environment', () async {
      final VfoxCommand command = cli.use('java', '21', scope: UseScope.project, projectDirectory: '/work/app');
      await cli.start(command);
      final ProcessCall call = runner.invocations.single;
      expect(call.streamed, isTrue);
      expect(call.workingDirectory, '/work/app');
      expect(call.environment['__VFOX_SHELL'], 'bash');
    });

    test('start reports a missing executable', () async {
      runner.missingExecutable = true;
      expect(cli.start(cli.updateAllPlugins()), throwsA(isA<VfoxNotFoundException>()));
    });
  });
}
