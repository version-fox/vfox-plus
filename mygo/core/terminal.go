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

package core

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// ansiEscape matches ANSI escape sequences: CSI sequences such as colours
// and cursor movement, OSC sequences such as hyperlinks, and lone two-byte
// escapes.
var ansiEscape = regexp.MustCompile("\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\x07\x1b]*(?:\x07|\x1b\\\\)|[@-Z\\\\-_])")

// StripANSI removes ANSI escape sequences from text.
//
// vfox honours `NO_COLOR`, but its download progress bar does not, so the
// output is always sanitised before it is parsed or shown.
func StripANSI(text string) string {
	return ansiEscape.ReplaceAllString(text, "")
}

// SplitTerminalLines splits terminal output into lines.
//
// Both `\n` and a lone `\r` end a line: progress bars redraw themselves by
// returning the carriage, and every redraw is treated as a new line.
func SplitTerminalLines(text string) []string {
	return strings.FieldsFunc(strings.ReplaceAll(text, "\r\n", "\n"), func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}

// TerminalLineSplitter turns chunks of raw process output into plain-text
// lines while the process runs.
//
// Bytes are decoded as UTF-8 (malformed bytes are replaced, and a character
// split between two chunks is kept together), lines end at `\n` and at `\r`,
// ANSI escape sequences are removed and blank lines are dropped. The trailing
// partial line is emitted by Close.
type TerminalLineSplitter struct {
	pending []byte
	text    strings.Builder
}

// Push feeds a chunk of output and returns the lines it completed.
func (s *TerminalLineSplitter) Push(chunk []byte) []string {
	s.pending = append(s.pending, chunk...)
	s.text.WriteString(decodePending(&s.pending, false))
	return s.split()
}

// Close ends the output and returns its last, unterminated line, if any.
// A trailing partial character is replaced, as a decoder does at the end of
// the stream.
func (s *TerminalLineSplitter) Close() []string {
	lines := s.split()
	rest := cleanLine(s.text.String() + decodePending(&s.pending, true))
	s.pending = nil
	s.text.Reset()
	return append(lines, rest...)
}

func (s *TerminalLineSplitter) split() []string {
	text := s.text.String()
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' || text[i] == '\r' {
			lines = append(lines, cleanLine(text[start:i])...)
			start = i + 1
		}
	}
	rest := text[start:]
	s.text.Reset()
	s.text.WriteString(rest)
	return lines
}

// decodePending decodes the pending bytes, keeping a trailing incomplete
// character for the next chunk unless final ends the stream.
func decodePending(pending *[]byte, final bool) string {
	keep := 0
	if !final {
		keep = trailingIncomplete(*pending)
	}
	data := (*pending)[:len(*pending)-keep]
	out := strings.Builder{}
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size <= 1 {
			// A malformed sequence is replaced once as a whole, as decoders
			// of other languages do, rather than once per byte.
			size = malformedLength(data)
		}
		out.WriteRune(r)
		data = data[size:]
	}
	*pending = append([]byte(nil), (*pending)[len(*pending)-keep:]...)
	return out.String()
}

// trailingIncomplete returns how many trailing bytes could still become a
// valid rune once more bytes arrive (0 when the tail is complete, or
// malformed and decoded right away).
func trailingIncomplete(data []byte) int {
	n := len(data)
	if n == 0 {
		return 0
	}
	// Step back to the start of the trailing run: continuation bytes belong
	// to it, and so does a lead byte that begins it.
	i := n - 1
	for i > 0 && isContinuationByte(data[i]) {
		i--
	}
	c := data[i]
	if c < 0x80 || !isLeadByte(c) {
		// A complete character, or stray and malformed bytes, which decode
		// to U+FFFD immediately.
		return 0
	}
	var want int
	switch {
	case c < 0xE0:
		want = 2
	case c < 0xF0:
		want = 3
	default:
		want = 4
	}
	if n-i < want {
		return n - i
	}
	return 0
}

func isContinuationByte(c byte) bool { return c >= 0x80 && c < 0xC0 }

func isLeadByte(c byte) bool { return c >= 0xC2 && c <= 0xF4 }

func cleanLine(line string) []string {
	text := strings.TrimSpace(StripANSI(line))
	if text == "" {
		return nil
	}
	return []string{text}
}

// malformedLength is the length of the longest start of a valid UTF-8 sequence
// at the beginning of data, which is malformed: at least one byte.
func malformedLength(data []byte) int {
	lead := data[0]
	var want int
	switch {
	case lead >= 0xC2 && lead < 0xE0:
		want = 2
	case lead >= 0xE0 && lead < 0xF0:
		want = 3
	case lead >= 0xF0 && lead <= 0xF4:
		want = 4
	default:
		return 1
	}
	length := 1
	for ; length < want && length < len(data); length++ {
		low, high := byte(0x80), byte(0xBF)
		if length == 1 {
			switch lead {
			case 0xE0:
				low = 0xA0
			case 0xED:
				high = 0x9F
			case 0xF0:
				low = 0x90
			case 0xF4:
				high = 0x8F
			}
		}
		if data[length] < low || data[length] > high {
			break
		}
	}
	return length
}
