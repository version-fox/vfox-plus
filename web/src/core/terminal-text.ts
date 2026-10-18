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

/** Helpers for turning raw terminal output into plain text. */

/**
 * Matches ANSI escape sequences: CSI sequences such as colours and cursor
 * movement, OSC sequences such as hyperlinks, and lone two-byte escapes.
 */
// eslint-disable-next-line no-control-regex -- terminal escapes are control characters.
const ANSI_ESCAPE = /\x1B(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1B]*(?:\x07|\x1B\\)|[@-Z\\-_])/g;

/**
 * Removes ANSI escape sequences from `text`.
 *
 * vfox honours `NO_COLOR`, but its download progress bar does not, so the
 * output is always sanitised before it is parsed or shown.
 */
export function stripAnsi(text: string): string {
  return text.replace(ANSI_ESCAPE, '');
}

/**
 * Splits terminal output into lines.
 *
 * Both `\n` and a lone `\r` end a line: progress bars redraw themselves by
 * returning the carriage, and every redraw is treated as a new line.
 */
export function splitTerminalLines(text: string): string[] {
  return text.replaceAll('\r\n', '\n').split(/[\r\n]/);
}

/**
 * Turns chunks of raw process output into plain-text lines while the process
 * runs.
 *
 * Bytes are decoded as UTF-8 (malformed bytes are replaced, and a character
 * split between two chunks is kept together), lines end at `\n` and at `\r`,
 * ANSI escape sequences are removed and blank lines are dropped. The trailing
 * partial line is emitted by {@link TerminalLineSplitter.close}.
 */
export class TerminalLineSplitter {
  private readonly decoder = new TextDecoder('utf-8', { fatal: false });
  private pending = '';

  /** Feeds a chunk of output and returns the lines it completed. */
  push(chunk: Uint8Array): string[] {
    return this.split(this.decoder.decode(chunk, { stream: true }));
  }

  /** Ends the output and returns its last, unterminated line, if any. */
  close(): string[] {
    const lines = this.split(this.decoder.decode());
    const rest = this.pending;
    this.pending = '';
    return [...lines, ...TerminalLineSplitter.clean(rest)];
  }

  private split(text: string): string[] {
    const lines: string[] = [];
    let start = 0;
    for (let i = 0; i < text.length; i++) {
      const code = text.charCodeAt(i);
      if (code === 0x0a || code === 0x0d) {
        lines.push(...TerminalLineSplitter.clean(this.pending + text.slice(start, i)));
        this.pending = '';
        start = i + 1;
      }
    }
    this.pending += text.slice(start);
    return lines;
  }

  private static clean(line: string): string[] {
    const text = stripAnsi(line);
    return text.trim() === '' ? [] : [text];
  }
}
