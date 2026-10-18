# Copyright 2026 Han Li and contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Helpers for turning raw terminal output into plain text."""

import codecs
import re

# ANSI escape sequences: CSI sequences such as colours and cursor movement,
# OSC sequences such as hyperlinks, and lone two-byte escapes.
_ANSI_ESCAPE = re.compile(r"\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[@-Z\\-_])")
_LINE_BREAK = re.compile(r"[\r\n]")


def strip_ansi(text: str) -> str:
    """Removes ANSI escape sequences from `text`.

    vfox honours `NO_COLOR`, but its download progress bar does not, so the
    output is always sanitised before it is parsed or shown.
    """
    return _ANSI_ESCAPE.sub("", text)


def split_terminal_lines(text: str) -> list[str]:
    """Splits terminal output into lines.

    Both a line feed and a lone carriage return end a line: progress bars
    redraw themselves by returning the carriage, and every redraw is treated
    as a new line.
    """
    return _LINE_BREAK.split(text.replace("\r\n", "\n"))


class TerminalLineSplitter:
    """Turns chunks of raw process output into plain-text lines while the
    process runs.

    Bytes are decoded as UTF-8 (malformed bytes are replaced, and a character
    split between two chunks is kept together), lines end at line feeds and
    carriage returns, ANSI escape sequences are removed and blank lines are
    dropped. The trailing partial line is emitted by :meth:`close`.
    """

    def __init__(self) -> None:
        self._decoder = codecs.getincrementaldecoder("utf-8")(errors="replace")
        self._pending = ""

    def push(self, chunk: bytes) -> list[str]:
        """Feeds a chunk of output and returns the lines it completed."""
        return self._split(self._decoder.decode(chunk))

    def close(self) -> list[str]:
        """Ends the output and returns its last, unterminated line, if any."""
        lines = self._split(self._decoder.decode(b"", final=True))
        rest = self._pending
        self._pending = ""
        return [*lines, *_clean(rest)]

    def _split(self, text: str) -> list[str]:
        lines: list[str] = []
        start = 0
        for index, char in enumerate(text):
            if char in "\n\r":
                lines.extend(_clean(self._pending + text[start:index]))
                self._pending = ""
                start = index + 1
        self._pending += text[start:]
        return lines


def _clean(line: str) -> list[str]:
    text = strip_ansi(line)
    return [] if text.strip() == "" else [text]
