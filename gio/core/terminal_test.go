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

package core_test

import (
	"reflect"
	"testing"

	core "github.com/version-fox/vfox-plus/gio/core"
)

func TestStripANSIRemovesColoursLinksAndCursorMoves(t *testing.T) {
	for name, tc := range map[string]struct{ input, want string }{
		"colour":     {"\x1b[31mred\x1b[0m text", "red text"},
		"cursor":     {"a\x1b[2K\x1b[1Gb", "ab"},
		"hyperlink":  {"\x1b]8;;https://vfox.dev\x07vfox\x1b]8;;\x07", "vfox"},
		"hyperlink2": {"\x1b]8;;https://vfox.dev\x1b\\vfox\x1b]8;;\x1b\\", "vfox"},
		"plain":      {"nothing to remove", "nothing to remove"},
	} {
		if got := core.StripANSI(tc.input); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestSplitTerminalLinesEndsLinesAtNewlinesAndCarriageReturns(t *testing.T) {
	got := core.SplitTerminalLines("one\r\ntwo\rthree\n\nfour")
	if want := []string{"one", "two", "three", "four"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLineSplitterJoinsChunksAndCharactersSplitBetweenThem(t *testing.T) {
	var splitter core.TerminalLineSplitter
	// "é" is two bytes and "你" three; each is cut between two chunks.
	chunks := [][]byte{[]byte("caf\xc3"), []byte("\xa9 \xe4\xbd"), []byte("\xa0\nnext"), []byte(" line\r"), []byte("\x1b[32mgreen\x1b[0m\n")}
	var got []string
	for _, chunk := range chunks {
		got = append(got, splitter.Push(chunk)...)
	}
	got = append(got, splitter.Close()...)
	if want := []string{"café 你", "next line", "green"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLineSplitterRedrawsOfAProgressBarAreLines(t *testing.T) {
	var splitter core.TerminalLineSplitter
	got := splitter.Push([]byte("\rDownloading...  10%\rDownloading...  20%\r"))
	if len(got) != 2 || got[1] != "Downloading...  20%" {
		t.Errorf("got %q", got)
	}
}

func TestLineSplitterEmitsTheUnterminatedLineOnClose(t *testing.T) {
	var splitter core.TerminalLineSplitter
	if got := splitter.Push([]byte("partial")); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
	if got := splitter.Close(); !reflect.DeepEqual(got, []string{"partial"}) {
		t.Errorf("got %q", got)
	}
	if got := splitter.Close(); len(got) != 0 {
		t.Errorf("a closed splitter has nothing left, got %q", got)
	}
}

func TestLineSplitterReplacesMalformedBytes(t *testing.T) {
	var splitter core.TerminalLineSplitter
	got := splitter.Push([]byte("a\xffb\xe4\xbd\n"))
	// A character that is cut short at the end of a stream becomes U+FFFD.
	got = append(got, splitter.Close()...)
	if want := []string{"a�b�"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	var truncated core.TerminalLineSplitter
	truncated.Push([]byte("end \xe4\xbd"))
	if got := truncated.Close(); !reflect.DeepEqual(got, []string{"end �"}) {
		t.Errorf("got %q", got)
	}
}

func TestVersionAdditionIsShownWithItsVersion(t *testing.T) {
	if got := (core.VersionAddition{Name: "npm", Version: "11.19.1"}).String(); got != "npm 11.19.1" {
		t.Errorf("got %q", got)
	}
	if got := (core.VersionAddition{Name: "npm"}).String(); got != "npm" {
		t.Errorf("got %q", got)
	}
}
