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
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"syscall"
)

// madvPageout is MADV_PAGEOUT of Linux 5.4: reclaim the pages of the range now, those that only
// this process maps.
const madvPageout = 21

// pageOutFiles tells the system that the pages of the files that the process maps are not wanted
// for now, and returns how many KB of them it asked it to take back.
//
// The resident set of the application is mostly such pages: a Go executable is mapped whole and
// fault-around maps sixteen pages for every one that runs, the toolkit maps the fonts, and each
// library maps its code. They are clean, the system can take them back whenever it needs the
// memory, and nothing runs most of them while the window is closed. Saying so lets it do that
// first, instead of the memory of other programs. They come back when something runs them.
//
// Only the mappings that cannot hold anything of the process's own are touched: the code and the
// read-only data of files, with no page that was written (the relocations of a library are, and
// writing them to swap and reading them again would only cost time). Pages that other programs map
// too, as those of the libraries of the desktop, are left where they are by the system.
func pageOutFiles() (kb int) {
	file, err := os.Open("/proc/self/smaps")
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }()
	for _, r := range pageableSpans(file) {
		if _, _, errno := syscall.Syscall(syscall.SYS_MADVISE, r.start, r.end-r.start, madvPageout); errno == 0 {
			kb += r.resident
		}
	}
	return kb
}

// pageableSpans finds in the text of smaps the mappings that may be paged out: the code and the
// read-only data of files, with no page of them that was written, and with something in memory.
//
// The text is some 400 KB for a process like this one: it is read line by line and taken apart
// where it lies, so that reading it leaves no garbage for the heap to keep.
func pageableSpans(source io.Reader) []span {
	var (
		reader    = bufio.NewReaderSize(source, 64*1024)
		spans     []span
		current   span
		eligible  bool
		anonymous int
		inMapping bool
	)
	flush := func() {
		if inMapping && eligible && anonymous == 0 && current.resident > 0 {
			spans = append(spans, current)
		}
		inMapping = false
	}
	for {
		line, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			// A line longer than the buffer is no line of smaps: skip what is left of it.
			for errors.Is(err, bufio.ErrBufferFull) {
				_, err = reader.ReadSlice('\n')
			}
			continue
		}
		if len(line) > 0 {
			switch {
			case isHex(line[0]):
				// A mapping: "start-end perms offset device inode path".
				flush()
				var ok bool
				if current.start, current.end, eligible, ok = parseMapping(line); ok {
					inMapping, anonymous, current.resident = true, 0, 0
				}
			case !inMapping:
			case bytes.HasPrefix(line, []byte("Anonymous:")):
				anonymous = kilobytes(line)
			case bytes.HasPrefix(line, []byte("Rss:")):
				current.resident = kilobytes(line)
			}
		}
		if err != nil {
			break
		}
	}
	flush()
	return spans
}

// span is a range of the address space that is to be paged out, with what of it is resident in KB.
type span struct {
	start, end uintptr
	resident   int
}

// isHex tells whether c starts the address of a mapping in smaps, which is written in lower case:
// the other lines start with the name of a measure, in capitals.
func isHex(c byte) bool { return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' }

// parseMapping reads the line of a mapping: its bounds, and whether it is the code or the
// read-only data of a file, which is what may be paged out.
func parseMapping(line []byte) (start, end uintptr, eligible, ok bool) {
	rest := line
	next := func() []byte {
		rest = bytes.TrimLeft(rest, " ")
		i := bytes.IndexAny(rest, " \n")
		if i < 0 {
			i = len(rest)
		}
		field := rest[:i]
		rest = rest[i:]
		return field
	}
	bounds, perms := next(), next()
	next() // The offset,
	next() // the device
	next() // and the inode.
	path := bytes.TrimRight(bytes.TrimLeft(rest, " "), "\n")
	dash := bytes.IndexByte(bounds, '-')
	if dash < 0 {
		return 0, 0, false, false
	}
	low, errLow := parseHex(bounds[:dash])
	high, errHigh := parseHex(bounds[dash+1:])
	if errLow || errHigh {
		return 0, 0, false, false
	}
	eligible = len(path) > 0 && path[0] == '/' && (string(perms) == "r-xp" || string(perms) == "r--p")
	return uintptr(low), uintptr(high), eligible, true
}

// parseHex reads an address; failed is true when it is none.
func parseHex(digits []byte) (value uint64, failed bool) {
	if len(digits) == 0 {
		return 0, true
	}
	for _, c := range digits {
		switch {
		case c >= '0' && c <= '9':
			value = value<<4 | uint64(c-'0')
		case c >= 'a' && c <= 'f':
			value = value<<4 | uint64(c-'a'+10)
		default:
			return 0, true
		}
	}
	return value, false
}

// kilobytes reads the number of a line such as "Rss:        1234 kB".
func kilobytes(line []byte) int {
	n := 0
	for _, c := range line[bytes.IndexByte(line, ':')+1:] {
		switch {
		case c >= '0' && c <= '9':
			n = n*10 + int(c-'0')
		case n > 0:
			return n
		}
	}
	return n
}
