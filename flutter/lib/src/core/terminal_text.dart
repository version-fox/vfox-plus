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

/// Helpers for turning raw terminal output into plain text.
library;

import 'dart:async';
import 'dart:convert';

/// Matches ANSI escape sequences: CSI sequences such as colours and cursor
/// movement, OSC sequences such as hyperlinks, and lone two-byte escapes.
final RegExp _ansiEscape = RegExp(r'\x1B(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1B]*(?:\x07|\x1B\\)|[@-Z\\-_])');

/// Removes ANSI escape sequences from [text].
///
/// vfox honours `NO_COLOR`, but its download progress bar does not, so the
/// output is always sanitised before it is parsed or shown.
String stripAnsi(String text) => text.replaceAll(_ansiEscape, '');

/// Splits terminal output into lines.
///
/// Both `\n` and a lone `\r` end a line: progress bars redraw themselves by
/// returning the carriage, and every redraw is treated as a new line.
List<String> splitTerminalLines(String text) {
  return text.replaceAll('\r\n', '\n').split(RegExp('[\r\n]'));
}

/// Converts a stream of raw process output into a stream of plain-text lines.
///
/// Lines are decoded as UTF-8 (malformed bytes are replaced), stripped of
/// ANSI escape sequences and split on `\n` and `\r`. Empty lines are dropped.
/// A trailing partial line is emitted when the input stream closes.
Stream<String> terminalLines(Stream<List<int>> bytes) {
  return bytes
      .transform(const Utf8Decoder(allowMalformed: true))
      .transform(const _TerminalLineSplitter())
      .map(stripAnsi)
      .where((String line) => line.trim().isNotEmpty);
}

class _TerminalLineSplitter extends StreamTransformerBase<String, String> {
  const _TerminalLineSplitter();

  @override
  Stream<String> bind(Stream<String> stream) {
    return Stream<String>.eventTransformed(stream, _TerminalLineSink.new);
  }
}

class _TerminalLineSink implements EventSink<String> {
  _TerminalLineSink(this._out);

  final EventSink<String> _out;
  final StringBuffer _pending = StringBuffer();

  @override
  void add(String chunk) {
    for (final int unit in chunk.codeUnits) {
      if (unit == 0x0A || unit == 0x0D) {
        _flush();
      } else {
        _pending.writeCharCode(unit);
      }
    }
  }

  void _flush() {
    if (_pending.isNotEmpty) {
      _out.add(_pending.toString());
      _pending.clear();
    }
  }

  @override
  void addError(Object error, [StackTrace? stackTrace]) => _out.addError(error, stackTrace);

  @override
  void close() {
    _flush();
    _out.close();
  }
}
