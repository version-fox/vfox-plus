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
	"testing"
	"unicode/utf8"
)

func TestLogTailKeepsTheNewestLines(t *testing.T) {
	cases := []struct {
		name                 string
		text                 string
		maxLines, maxBytes   int
		wantTail             string
		wantTotal, wantShown int
	}{
		{"an empty log has no lines", "", 10, 100, "", 0, 0},
		{"the newline that ends the text is not a line", "a\nb\n", 10, 100, "a\nb", 2, 2},
		{"the lines are cut from the start", "a\nb\nc\nd", 2, 100, "c\nd", 4, 2},
		{"a line is kept when it fits the bytes", "aaaa\nbb\ncc", 10, 6, "bb\ncc", 3, 2},
		{"a line that does not fit the bytes is left out whole", "aaaa\nbb\ncc", 10, 5, "cc", 3, 1},
		{"a single long line keeps its newest bytes", "xxxxxxxxxx", 10, 4, "xxxx", 1, 1},
		{"no line is asked for", "a\nb", 0, 100, "", 2, 0},
		{"a long line is cut at a character", "我我我我", 10, 4, "我", 1, 1},
	}
	for _, tc := range cases {
		tail, total, shown := logTail(tc.text, tc.maxLines, tc.maxBytes)
		if tail != tc.wantTail || total != tc.wantTotal || shown != tc.wantShown {
			t.Errorf("%s: logTail(%q, %d, %d) = (%q, %d, %d), want (%q, %d, %d)", tc.name, tc.text, tc.maxLines, tc.maxBytes,
				tail, total, shown, tc.wantTail, tc.wantTotal, tc.wantShown)
		}
	}
}

func TestLogTailCutsOnlyAtCharacters(t *testing.T) {
	text := "一二三四五六七八九十\n" + "甲乙丙丁戊己庚辛壬癸"
	for maxBytes := 1; maxBytes <= len(text); maxBytes++ {
		tail, _, _ := logTail(text, 5, maxBytes)
		if !utf8.ValidString(tail) {
			t.Fatalf("logTail with %d bytes cut a character: %q", maxBytes, tail)
		}
		if len(tail) > maxBytes {
			t.Fatalf("logTail with %d bytes kept %d bytes", maxBytes, len(tail))
		}
	}
}

func TestTheBoundsOfThePagesAreSane(t *testing.T) {
	if taskRowLines > taskPageLines || taskPageLines > logPageLines {
		t.Errorf("the lines shown grow from the list (%d) to the page (%d) to the full log (%d); they should not shrink", taskRowLines, taskPageLines, logPageLines)
	}
	if taskOutputBytes > logPageBytes {
		t.Errorf("the output of a task (%d bytes) is larger than the full log allows (%d)", taskOutputBytes, logPageBytes)
	}
}
