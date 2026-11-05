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

//go:build linux

package ui

import (
	"os"
	"strings"
	"testing"
)

// smapsSample is what /proc/<pid>/smaps shows of a process, cut down to the lines that matter and
// a few that do not: the executable, a library with its relocated part, the heap, an anonymous
// mapping, a font, a mapping that nothing has loaded and the special ones of the kernel.
const smapsSample = `00400000-01400000 r--p 00000000 08:30 1001                               /usr/bin/vfox-plus
Size:              16384 kB
Rss:               12000 kB
Anonymous:             0 kB
VmFlags: rd mr mw me
01400000-02400000 r-xp 01000000 08:30 1001                               /usr/bin/vfox-plus
Size:              16384 kB
Rss:                8000 kB
Anonymous:             0 kB
02400000-02500000 rw-p 02000000 08:30 1001                               /usr/bin/vfox-plus
Size:               1024 kB
Rss:                 900 kB
Anonymous:           900 kB
7f0000000000-7f0000100000 r--p 00000000 08:30 2002                       /usr/lib/x86_64-linux-gnu/libgtk-3.so.0.2420.32
Size:                1024 kB
Rss:                 500 kB
Anonymous:             0 kB
7f0000100000-7f0000200000 r--p 00100000 08:30 2002                       /usr/lib/x86_64-linux-gnu/libgtk-3.so.0.2420.32 (deleted)
Size:                1024 kB
Rss:                 100 kB
Anonymous:           100 kB
7f0000300000-7f0000400000 r--p 00000000 08:30 3003                       /usr/share/fonts/truetype/ubuntu/UbuntuSans[wdth,wght].ttf
Size:                1024 kB
Rss:                 700 kB
Anonymous:             0 kB
7f0000500000-7f0000600000 r--p 00000000 08:30 4004                       /usr/lib/locale/locale-archive
Size:                1024 kB
Rss:                   0 kB
Anonymous:             0 kB
7f0000700000-7f0000800000 rw-p 00000000 00:00 0
Size:                1024 kB
Rss:                 800 kB
Anonymous:           800 kB
7f0000900000-7f0000a00000 r--p 00000000 00:00 0                          [anon: Go: heap]
Size:                1024 kB
Rss:                 800 kB
Anonymous:           800 kB
7f0000b00000-7f0000c00000 rw-s 00000000 00:01 5005                       /dev/shm/buffer
Size:                1024 kB
Rss:                 300 kB
Anonymous:             0 kB
7ffc00000000-7ffc00021000 rw-p 00000000 00:00 0                          [stack]
Size:                 132 kB
Rss:                  40 kB
Anonymous:            40 kB
7ffc10000000-7ffc10004000 r-xp 00000000 00:00 0                          [vdso]
Size:                  16 kB
Rss:                   8 kB
Anonymous:             0 kB
`

func TestOnlyTheCodeAndTheReadOnlyDataOfFilesArePagedOut(t *testing.T) {
	spans := pageableSpans(strings.NewReader(smapsSample))
	type want struct {
		start, end uintptr
		resident   int
	}
	expected := []want{
		{0x00400000, 0x01400000, 12000}, // The executable, read-only.
		{0x01400000, 0x02400000, 8000},  // Its code.
		{0x7f0000000000, 0x7f0000100000, 500},
		{0x7f0000300000, 0x7f0000400000, 700}, // A font that the toolkit mapped.
	}
	if len(spans) != len(expected) {
		t.Fatalf("%d spans, want %d: %+v", len(spans), len(expected), spans)
	}
	for i, w := range expected {
		if got := spans[i]; got.start != w.start || got.end != w.end || got.resident != w.resident {
			t.Errorf("span %d = %+v, want %+v", i, got, w)
		}
	}
	// What is not there: the data of the executable that the process wrote (rw-p), the relocated
	// part of a library (a page of it is the process's own), what no file backs, the heap, the
	// buffer that the window shares with the display (a file, but written), the stack, the
	// vdso, and what nothing has loaded.
}

func TestParsingTheLinesOfSmaps(t *testing.T) {
	start, end, eligible, ok := parseMapping([]byte("7f0000300000-7f0000400000 r--p 00000000 08:30 3003                       /usr/share/fonts/UbuntuSans[wdth,wght].ttf\n"))
	if !ok || start != 0x7f0000300000 || end != 0x7f0000400000 || !eligible {
		t.Errorf("a font = %x %x %t %t", start, end, eligible, ok)
	}
	for line, want := range map[string]bool{
		"00400000-00401000 r-xp 00000000 08:30 1 /bin/vfox-plus\n":                    true,
		"00400000-00401000 r--p 00000000 08:30 1 /bin/vfox-plus\n":                    true,
		"00400000-00401000 rw-p 00000000 08:30 1 /bin/vfox-plus\n":                    false,
		"00400000-00401000 r--s 00000000 08:30 1 /bin/vfox-plus\n":                    false,
		"00400000-00401000 r--p 00000000 00:00 0 [vvar]\n":                            false,
		"00400000-00401000 r--p 00000000 00:00 0\n":                                   false,
		"00400000-00401000 r-xp 00000000 08:30 1 /tmp/a file with spaces (deleted)\n": true,
		"00400000-00401000 r--p 00000000 08:30 1 /usr/lib/locale/locale-archive":      true, // The last line may have no newline.
	} {
		if _, _, eligible, ok := parseMapping([]byte(line)); !ok || eligible != want {
			t.Errorf("parseMapping(%q) = eligible %t, ok %t, want eligible %t", line, eligible, ok, want)
		}
	}
	for _, bad := range []string{"no-dash 1 2 3 4\n", "zz-yy r--p 0 0 0\n", "-1000 r--p 0 0 0\n", "1000- r--p 0 0 0\n"} {
		if _, _, _, ok := parseMapping([]byte(bad)); ok {
			t.Errorf("parseMapping(%q) succeeded", bad)
		}
	}
	for line, want := range map[string]int{
		"Rss:               12000 kB\n": 12000,
		"Anonymous:             0 kB\n": 0,
		"Rss:                   7 kB":   7,
		"Rss:\n":                        0,
	} {
		if got := kilobytes([]byte(line)); got != want {
			t.Errorf("kilobytes(%q) = %d, want %d", line, got, want)
		}
	}
	for c, want := range map[byte]bool{'0': true, '9': true, 'a': true, 'f': true, 'g': false, 'A': false, 'R': false, 'V': false, ' ': false} {
		if got := isHex(c); got != want {
			t.Errorf("isHex(%q) = %t, want %t", c, got, want)
		}
	}
}

func TestALongLineIsNoMappingAndDoesNotStopTheReading(t *testing.T) {
	long := strings.Repeat("x", 200*1024)
	sample := "00400000-00401000 r--p 00000000 08:30 1 /bin/a\nRss: 4 kB\nAnonymous: 0 kB\n" +
		"VmFlags: " + long + "\n" +
		"00500000-00501000 r--p 00000000 08:30 2 /bin/b\nRss: 8 kB\nAnonymous: 0 kB\n"
	spans := pageableSpans(strings.NewReader(sample))
	if len(spans) != 2 || spans[0].resident != 4 || spans[1].resident != 8 {
		t.Fatalf("spans = %+v, want the two mappings around the long line", spans)
	}
}

func TestPagingOutTheFilesOfThisProcessLeavesItWorking(t *testing.T) {
	if _, err := os.Stat("/proc/self/smaps"); err != nil {
		t.Skip("no /proc/self/smaps")
	}
	// The test binary is a Go executable like the application: its code is mapped and some of
	// it is resident. Paging it out must leave the process as it was, and the code comes back
	// when it runs.
	if kb := pageOutFiles(); kb <= 0 {
		t.Errorf("pageOutFiles = %d KB: a running program has code in memory", kb)
	}
	total := 0
	for i := range 1000 {
		total += i
	}
	if total != 499500 {
		t.Errorf("the arithmetic of the process went wrong after the paging: %d", total)
	}
	if kb := pageOutFiles(); kb < 0 {
		t.Errorf("a second paging = %d", kb)
	}
}
