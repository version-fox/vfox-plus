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
import 'package:vfox_plus/src/ui/widgets/navigation_rail.dart';

/// The repository's shared resources, relative to the flutter/ directory.
Directory sharedDirectory() {
  final directory = Directory('../shared').absolute;
  expect(directory.existsSync(), isTrue, reason: 'Run the integration tests from the flutter/ directory.');
  return directory;
}

/// Pumps frames until [finder] matches exactly one widget, or fails after
/// [timeout]. Waiting for a single match also waits for page transitions,
/// during which the old and the new page are both on screen.
Future<void> pumpUntil(WidgetTester tester, Finder finder, {Duration timeout = const Duration(seconds: 30)}) async {
  final DateTime end = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(end)) {
    await tester.pump(const Duration(milliseconds: 100));
    if (finder.evaluate().length == 1) {
      return;
    }
  }
  fail('Timed out waiting for $finder');
}

/// Pumps frames until [condition] holds, or fails after [timeout] with
/// [reason], which completes "Timed out waiting until".
Future<void> pumpUntilTrue(
  WidgetTester tester,
  bool Function() condition, {
  required String reason,
  Duration timeout = const Duration(seconds: 30),
}) async {
  final DateTime end = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(end)) {
    await tester.pump(const Duration(milliseconds: 100));
    if (condition()) {
      return;
    }
  }
  fail('Timed out waiting until $reason');
}

/// Pumps frames for [duration] of real time, letting springs settle.
Future<void> pumpFor(WidgetTester tester, Duration duration) async {
  final DateTime end = DateTime.now().add(duration);
  while (DateTime.now().isBefore(end)) {
    await tester.pump(const Duration(milliseconds: 50));
  }
}

/// The destination [label] of the navigation rail.
Finder railDestination(String label) =>
    find.descendant(of: find.byType(ExpressiveNavigationRail), matching: find.bySemanticsLabel(label));

/// Writes [content] to [path] and makes it executable.
Future<File> writeExecutable(String path, String content) async {
  final file = File(path)..writeAsStringSync(content);
  final ProcessResult result = await Process.run('chmod', <String>['+x', path]);
  expect(result.exitCode, 0, reason: '${result.stderr}');
  return file;
}

/// A stand-in for vfox that replays the captured outputs in [fixtures] and
/// enforces the environment that the CLI contract requires.
String fakeVfoxScript(String fixtures) =>
    '''
#!/usr/bin/env bash
set -u
fixtures='$fixtures'
# The captures escape control characters as \\xHH and backslashes as \\\\;
# printf %b turns them back into the bytes that vfox printed.
show() { local text; text="\$(cat "\$1"; echo .)"; printf '%b' "\${text%.}"; }
reply() { show "\$fixtures/\$1.stdout"; show "\$fixtures/\$1.stderr" >&2; }
# The GUI must always disable colours and must not leak a shell session.
[ "\${NO_COLOR:-}" = 1 ] || { echo "NO_COLOR is not set"; exit 90; }
[ -z "\${__VFOX_PID:-}" ] || { echo "__VFOX_PID leaked"; exit 91; }
case "\$*" in
  --version) reply version ;;
  list) reply list ;;
  current) reply current ;;
  available) reply available ;;
  "config --list") reply config-list ;;
  "search maven") reply search-maven-installed ;;
  "search nodejs") reply search-nodejs ;;
  "info nodejs --format "*) reply info ;;
  "info "*" --format "*)
    # Plugin information for plugins without a captured fixture.
    printf '%s\\037%s\\037%s\\037%s\\037%s' "\$2" 0.1.2 "https://github.com/version-fox/vfox-\$2" /tmp "\$2 runtime environment." ;;
  "install --yes nodejs@"*)
    echo "Preinstalling \${3}..."
    for percent in 10 45 80 100; do
      printf '\\rDownloading... %3d%% [=====>    ]\\r' "\$percent" >&2
      sleep 0.2
    done
    echo "Install \${3} success!" ;;
  "use --global "*)
    # vfox refuses to use a version outside of an activated shell.
    [ -n "\${__VFOX_SHELL:-}" ] || { reply use-without-hook; exit 1; }
    echo "Now using \${3}." ;;
  *) echo "unexpected command: \$*" >&2; exit 2 ;;
esac
''';
