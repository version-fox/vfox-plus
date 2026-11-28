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

package measureapk

import (
	"fmt"
	"regexp"
	"strings"
)

// logEntry is one line of `logcat -v threadtime`: the process that wrote it, its level (V, D, I, W,
// E, F or S), its tag and its message.
type logEntry struct {
	process, level, tag, message string
}

// logLine matches one line of `logcat -v threadtime`: the date, the time, the process and the
// thread, then the level, the tag, and the message.
var logLine = regexp.MustCompile(`^\S+\s+\S+\s+(\d+)\s+\d+\s+([VDIWEFS])\s+([^:]+?)\s*:\s?(.*)$`)

// uiautomatorFrame is in the stack of a crash of uiautomator, the command of the device that reads the
// screens for the tool. On a slow emulator it fails now and then, with a timeout while it connects
// (`Timeout while connecting UiAutomation`), and the next read succeeds; its crash is not the app's.
const uiautomatorFrame = "com.android.commands.uiautomator."

// emulatorGLNoise matches the messages that the software GL of the emulator writes into the log of
// the app at the error level under the tag mygo: it reports the bindings of the attributes of its
// shaders, and nothing failed (docs/en/dev/memory.md, the first screen of the emulator).
var emulatorGLNoise = regexp.MustCompile(`s_glBindAttribLocation: bind attrib`)

// parseLog reads the entries of a log. Lines that are not entries, such as the markers of the
// buffers, are left out.
func parseLog(text string) []logEntry {
	var entries []logEntry
	for _, line := range strings.Split(text, "\n") {
		if match := logLine.FindStringSubmatch(strings.TrimRight(line, "\r")); match != nil {
			entries = append(entries, logEntry{process: match[1], level: match[2], tag: match[3], message: match[4]})
		}
	}
	return entries
}

// appErrors returns the errors of the application that the log shows, one line each: a fatal
// message of any tag, an error of the Android runtime (a crash) that is not uiautomator's, and an
// error that the app writes to its standard error, which the system logs at the error level under the
// tag mygo (the standard output is logged at the info level). An empty result means the app logged no
// error.
func appErrors(text string) []string {
	entries := parseLog(text)
	uiautomator := map[string]bool{}
	for _, e := range entries {
		if e.tag == "AndroidRuntime" && strings.Contains(e.message, uiautomatorFrame) {
			uiautomator[e.process] = true
		}
	}
	var problems []string
	for _, e := range entries {
		switch {
		case e.level == "F",
			e.tag == "AndroidRuntime" && e.level == "E" && !uiautomator[e.process],
			e.tag == "mygo" && e.level == "E" && !emulatorGLNoise.MatchString(e.message):
			problems = append(problems, fmt.Sprintf("%s/%s: %s", e.level, e.tag, e.message))
		}
	}
	return problems
}

// anrs returns the lines of the log that the system wrote for an application of the package pkg
// that did not answer. They are reported and are not failures: the emulator draws its first
// frames slowly, and an ANR there does not mean that the pairing failed.
func anrs(text, pkg string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "ANR in "+pkg) {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}
