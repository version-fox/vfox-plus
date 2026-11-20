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

package screens

import (
	"strings"
	"unicode/utf8"
)

// The bounds of the output that the phone lays out. A log can be large (the
// computer keeps a whole failed install), and a long text is laid out again on
// every frame, so the phone shows only the newest part of it:
//
//   - the full log page shows the newest logPageLines lines, and at most
//     logPageBytes bytes of them;
//   - the task page shows the newest taskPageLines lines of the output the
//     computer sends, and the task's row in the list shows the newest
//     taskRowLines lines.
//
// The newest lines are the ones that tell why a task failed, so the cut is always
// at the start. The page says how much it cut.
const (
	logPageLines  = 500
	logPageBytes  = 64 << 10
	taskPageLines = 200
	taskRowLines  = 40
	// taskOutputBytes bounds the output of the task page and of a row alike.
	taskOutputBytes = 32 << 10
)

// logTail returns the newest part of text that a page shows: at most maxLines lines,
// and at most maxBytes bytes of them, where a line that does not fit is left out
// whole. A single line that is longer than maxBytes keeps its newest bytes, cut at a
// character. It returns the part, the number of lines of text, and the number of
// lines of the part. The text may end with a newline, which is not a line of its own.
func logTail(text string, maxLines, maxBytes int) (tail string, total, shown int) {
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return "", 0, 0
	}
	total = strings.Count(text, "\n") + 1
	if maxLines <= 0 {
		return "", total, 0
	}
	limit := len(text) // the line being taken ends here, before a newline
	start, used := len(text), 0
	for shown < maxLines {
		begin := strings.LastIndexByte(text[:limit], '\n') + 1
		length := limit - begin
		if shown > 0 && used+length+1 > maxBytes {
			break
		}
		start = begin
		used += length + 1
		shown++
		if begin == 0 {
			break
		}
		limit = begin - 1
	}
	tail = text[start:]
	if len(tail) > maxBytes {
		cut := len(tail) - maxBytes
		for cut < len(tail) && !utf8.RuneStart(tail[cut]) {
			cut++
		}
		tail = tail[cut:]
	}
	return tail, total, shown
}
