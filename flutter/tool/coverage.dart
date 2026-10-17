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

// Summarises coverage/lcov.info and fails below a minimum line coverage.
//
//   flutter test --coverage && dart run tool/coverage.dart [--min=80]
//
// Generated code (translations and icons) is excluded.

import 'dart:io';

const double _defaultMinimum = 80;

bool _isGenerated(String path) => path.contains('/l10n/app_localizations') || path.endsWith('.g.dart');

void main(List<String> arguments) {
  final double minimum =
      arguments
          .where((String argument) => argument.startsWith('--min='))
          .map((String argument) => double.parse(argument.substring('--min='.length)))
          .lastOrNull ??
      _defaultMinimum;
  final lcov = File('coverage/lcov.info');
  if (!lcov.existsSync()) {
    stderr.writeln('coverage/lcov.info not found. Run `flutter test --coverage` first.');
    exit(2);
  }

  final files = <String, (int found, int hit)>{};
  String? current;
  var found = 0;
  var hit = 0;
  for (final String line in lcov.readAsLinesSync()) {
    if (line.startsWith('SF:')) {
      current = line.substring(3);
      found = 0;
      hit = 0;
    } else if (line.startsWith('LF:')) {
      found = int.parse(line.substring(3));
    } else if (line.startsWith('LH:')) {
      hit = int.parse(line.substring(3));
    } else if (line == 'end_of_record' && current != null) {
      if (!_isGenerated(current)) {
        files[current] = (found, hit);
      }
      current = null;
    }
  }

  final List<String> paths = files.keys.toList()..sort();
  var totalFound = 0;
  var totalHit = 0;
  for (final path in paths) {
    final (int found, int hit) = files[path]!;
    totalFound += found;
    totalHit += hit;
    stdout.writeln('${_percent(hit, found).padLeft(6)}  ${'$hit/$found'.padLeft(9)}  $path');
  }
  final double total = totalFound == 0 ? 100 : 100 * totalHit / totalFound;
  stdout.writeln('${_percent(totalHit, totalFound).padLeft(6)}  ${'$totalHit/$totalFound'.padLeft(9)}  TOTAL');
  if (total < minimum) {
    stderr.writeln('Line coverage ${total.toStringAsFixed(1)}% is below the minimum of $minimum%.');
    exit(1);
  }
}

String _percent(int hit, int found) => found == 0 ? '100.0%' : '${(100 * hit / found).toStringAsFixed(1)}%';
