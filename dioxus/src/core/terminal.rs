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

//! Terminal output: ANSI escape sequences and lines, which progress bars
//! redraw with carriage returns.

const ESC: char = '\u{1b}';
const BEL: char = '\u{7}';

/// Removes ANSI escape sequences from `text`: CSI sequences such as colours
/// and cursor movement, OSC sequences such as hyperlinks, and lone two-byte
/// escapes. vfox honours `NO_COLOR`, but its download progress bar does not,
/// so output is always sanitised before it is parsed or shown.
pub fn strip_ansi(text: &str) -> String {
    let chars: Vec<char> = text.chars().collect();
    let mut result = String::with_capacity(text.len());
    let mut i = 0;
    while i < chars.len() {
        if chars[i] == ESC
            && let Some(end) = escape_end(&chars, i)
        {
            i = end;
            continue;
        }
        result.push(chars[i]);
        i += 1;
    }
    result
}

/// The index after the escape sequence that starts at `start`, if one does.
fn escape_end(chars: &[char], start: usize) -> Option<usize> {
    let next = *chars.get(start + 1)?;
    match next {
        // CSI: parameter bytes, intermediate bytes and a final byte.
        '[' => {
            let mut i = start + 2;
            while chars.get(i).is_some_and(|c| ('0'..='?').contains(c)) {
                i += 1;
            }
            while chars.get(i).is_some_and(|c| (' '..='/').contains(c)) {
                i += 1;
            }
            chars.get(i).filter(|c| ('@'..='~').contains(*c)).map(|_| i + 1)
        }
        // OSC, terminated by BEL or ESC \; unterminated, it is a lone escape.
        ']' => {
            let mut i = start + 2;
            while let Some(&c) = chars.get(i) {
                if c == BEL {
                    return Some(i + 1);
                }
                if c == ESC {
                    return if chars.get(i + 1) == Some(&'\\') { Some(i + 2) } else { Some(start + 2) };
                }
                i += 1;
            }
            Some(start + 2)
        }
        '@'..='Z' | '\\'..='_' => Some(start + 2),
        _ => None,
    }
}

/// Splits terminal output into lines. Both `\n` and a lone `\r` end a line:
/// progress bars redraw themselves by returning the carriage, and every redraw
/// is treated as a new line.
pub fn split_terminal_lines(text: &str) -> Vec<String> {
    text.replace("\r\n", "\n").split(['\r', '\n']).map(str::to_owned).collect()
}

/// Turns chunks of raw process output into plain-text lines while the process
/// runs: bytes are decoded as UTF-8 (malformed bytes are replaced, and a
/// character split between two chunks is kept together), lines end at `\n`
/// and at `\r`, ANSI escape sequences are removed and blank lines are
/// dropped. The trailing partial line is returned by [`TerminalLineSplitter::close`].
#[derive(Debug, Default)]
pub struct TerminalLineSplitter {
    leftover: Vec<u8>,
    pending: String,
}

impl TerminalLineSplitter {
    /// A splitter without output.
    pub fn new() -> Self {
        Self::default()
    }

    /// Feeds a chunk of output and returns the lines it completed.
    pub fn push(&mut self, chunk: &[u8]) -> Vec<String> {
        let mut bytes = std::mem::take(&mut self.leftover);
        bytes.extend_from_slice(chunk);
        let mut text = String::new();
        let mut rest: &[u8] = &bytes;
        loop {
            match std::str::from_utf8(rest) {
                Ok(valid) => {
                    text.push_str(valid);
                    break;
                }
                Err(error) => {
                    let (valid, after) = rest.split_at(error.valid_up_to());
                    // Valid by construction.
                    text.push_str(std::str::from_utf8(valid).unwrap_or_default());
                    // Without an error length, a character continues in the next chunk.
                    let Some(length) = error.error_len() else {
                        self.leftover = after.to_vec();
                        break;
                    };
                    text.push('\u{fffd}');
                    rest = &after[length..];
                }
            }
        }
        self.split(&text)
    }

    /// Ends the output and returns its last, unterminated line, if any.
    pub fn close(&mut self) -> Vec<String> {
        let leftover = std::mem::take(&mut self.leftover);
        let mut lines = self.split(&String::from_utf8_lossy(&leftover));
        let rest = std::mem::take(&mut self.pending);
        lines.extend(clean(&rest));
        lines
    }

    fn split(&mut self, text: &str) -> Vec<String> {
        let mut lines = Vec::new();
        for c in text.chars() {
            if c == '\n' || c == '\r' {
                let line = std::mem::take(&mut self.pending);
                lines.extend(clean(&line));
            } else {
                self.pending.push(c);
            }
        }
        lines
    }
}

fn clean(line: &str) -> Option<String> {
    let text = strip_ansi(line);
    if text.trim().is_empty() { None } else { Some(text) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn strips_colours_hyperlinks_and_lone_escapes() {
        assert_eq!(
            strip_ansi("\u{1b}[1;32mok\u{1b}[0m \u{1b}]8;;https://x\u{7}link\u{1b}]8;;\u{7} \u{1b}Mx"),
            "ok link x"
        );
        assert_eq!(strip_ansi("a\u{1b}]8;;https://x\u{1b}\\b"), "ab");
        assert_eq!(strip_ansi("\u{1b}]unterminated"), "unterminated");
        assert_eq!(strip_ansi("\u{1b}"), "\u{1b}");
        assert_eq!(strip_ansi("x\u{1b}[12"), "x\u{1b}[12");
    }

    #[test]
    fn splits_at_carriage_returns_and_new_lines() {
        assert_eq!(split_terminal_lines("a\r\nb\rc\nd"), vec!["a", "b", "c", "d"]);
    }

    #[test]
    fn keeps_characters_split_between_chunks_and_replaces_malformed_bytes() {
        let mut splitter = TerminalLineSplitter::new();
        assert_eq!(splitter.push(b"one\r\x1b[32mtw"), vec!["one"]);
        let euro = "€".as_bytes();
        let mut chunk = b"o ".to_vec();
        chunk.push(euro[0]);
        assert_eq!(splitter.push(&chunk), Vec::<String>::new());
        assert_eq!(splitter.push(&[euro[1], euro[2], b'\n', b' ', b'\n']), vec!["two €"]);
        assert_eq!(splitter.push(b"last"), Vec::<String>::new());
        assert_eq!(splitter.close(), vec!["last"]);
        let mut malformed = TerminalLineSplitter::new();
        assert_eq!(malformed.push(&[b'a', 0xff, b'b', b'\n']), vec!["a\u{fffd}b"]);
        let mut cut = TerminalLineSplitter::new();
        assert_eq!(cut.push(&[0xe2]), Vec::<String>::new());
        assert_eq!(cut.close(), vec!["\u{fffd}"]);
    }
}
