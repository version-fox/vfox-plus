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

// Package fakevfox is a stand-in for vfox that replays the captured outputs
// of shared/fixtures, replacing shared/fixtures/fake-vfox.sh.
//
// It answers the read-only commands with the captures of vfox-1.0.13 (or of
// $VFOX_PLUS_FIXTURES), pretends to install versions with a progress bar on
// standard error, and fails like vfox when `vfox use` runs without the hook
// environment. It also checks what the CLI contract requires of the caller:
// NO_COLOR=1, no __VFOX_PID of an activated shell and the working directory /
// for commands that are not about a project. It keeps no state: installing a
// version does not add it to the output of `vfox list`.
//
// VFOX_PLUS_FIXTURES is the fixture directory, by default the vfox-1.0.13
// directory next to the running executable.
package fakevfox

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// exit codes for the caller contract violations, matching fake-vfox.sh.
const (
	exitNoColor    = 90
	exitPIDLeaked  = 91
	exitNotAtRoot  = 92
	exitUnexpected = 2
)

// Run replays one fake vfox invocation. It exits the process with the same
// status the shell script would produce (90, 91, 92, 1 or 2 on failure)
// because the vfxtool entry point maps every returned error to exit 1.
func Run(args []string) error {
	if code := runCommand(args, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
	return nil
}

// runCommand executes the fake vfox command with args, writing standard
// output to out and standard error to errOut, and returns the exit code the
// invocation produces.
func runCommand(args []string, out, errOut io.Writer) int {
	if os.Getenv("NO_COLOR") != "1" {
		fmt.Fprintln(out, "NO_COLOR is not set")
		return exitNoColor
	}
	if os.Getenv("__VFOX_PID") != "" {
		fmt.Fprintln(out, "__VFOX_PID leaked")
		return exitPIDLeaked
	}
	// Commands that are not about a project run in the filesystem root.
	// Only an exact "/" in PWD is trusted: like bash, which overwrites an
	// invalid PWD with the working directory on startup, anything else that
	// leaked in through the environment is replaced with the ground truth.
	// Go sets no PWD itself, so without this the value that the launcher
	// happened to have would decide.
	first := ""
	if len(args) > 0 {
		first = args[0]
	}
	wd := os.Getenv("PWD")
	if wd != "/" {
		if cwd, err := os.Getwd(); err == nil {
			wd = cwd
		}
	}
	if wd != "/" && first != "use" && first != "unuse" {
		fmt.Fprintln(out, "not run in /: "+wd)
		return exitNotAtRoot
	}

	fixtures := fixturesDir()
	reply := func(name string) int {
		showFile(filepath.Join(fixtures, name+".stdout"), out)
		showFile(filepath.Join(fixtures, name+".stderr"), errOut)
		return 0
	}

	// The dispatch table mirrors the case statement of fake-vfox.sh, which
	// matches on "$*" with the arguments joined by spaces.
	joined := strings.Join(args, " ")
	switch {
	case joined == "--version":
		return reply("version")
	case joined == "list":
		return reply("list")
	case joined == "current":
		return reply("current")
	case joined == "available":
		return reply("available")
	case joined == "config --list":
		return reply("config-list")
	case strings.HasPrefix(joined, "config "):
		return 0
	case joined == "search maven":
		return reply("search-maven-installed")
	case joined == "search nodejs":
		return reply("search-nodejs")
	case strings.HasPrefix(joined, "search "):
		fmt.Fprint(out, "Available versions:\n - 2.1.0 (LTS) [tool 2.1]\n - 2.0.0\n - 1.0.0\n")
		return 0
	case strings.HasPrefix(joined, "info nodejs --format "):
		return reply("info")
	case isInfoFormat(joined):
		// Plugin information for plugins without a captured fixture.
		sdk := args[1]
		fmt.Fprintf(out, "%s\x1f%s\x1f%s\x1f%s\x1f%s",
			sdk, "0.1.2", "https://github.com/version-fox/vfox-"+sdk, "/tmp", sdk+" runtime environment.")
		return 0
	case strings.HasPrefix(joined, "info ") && strings.Contains(joined[len("info "):], "@"):
		return reply("info-version-path")
	case strings.HasPrefix(joined, "install --yes ") && len(args) >= 3:
		what := args[2]
		fmt.Fprintf(out, "Preinstalling %s...\n", what)
		for _, percent := range []int{10, 45, 80, 100} {
			fmt.Fprintf(errOut, "\rDownloading... %3d%% [=====>    ]\r", percent)
			time.Sleep(fakeDelay())
		}
		fmt.Fprintf(out, "Install %s success!\n", what)
		return 0
	case strings.HasPrefix(joined, "uninstall ") && len(args) >= 2:
		fmt.Fprintf(out, "Uninstalled %s successfully!\n", args[1])
		return 0
	case (strings.HasPrefix(joined, "use --global ") || strings.HasPrefix(joined, "use --project ")) && len(args) >= 3:
		// vfox refuses to use a version outside of an activated shell.
		if os.Getenv("__VFOX_SHELL") == "" {
			reply("use-without-hook")
			return 1
		}
		fmt.Fprintf(out, "Now using %s.\n", args[2])
		return 0
	case strings.HasPrefix(joined, "unuse ") && len(args) >= 3:
		fmt.Fprintf(out, "Unset %s successfully.\n", args[2])
		return 0
	case strings.HasPrefix(joined, "add ") && len(args) >= 2:
		fmt.Fprintf(out, "Plugin %s added successfully!\n", args[len(args)-1])
		return 0
	case strings.HasPrefix(joined, "remove --yes ") && len(args) >= 3:
		fmt.Fprintf(out, "Remove %s plugin successfully!\n", args[2])
		return 0
	case strings.HasPrefix(joined, "update ") && len(args) >= 2:
		fmt.Fprintf(out, "Update %s plugin successfully!\n", args[1])
		return 0
	default:
		fmt.Fprintf(errOut, "unexpected command: %s\n", joined)
		return exitUnexpected
	}
}

// isInfoFormat reports whether joined matches the shell pattern
// `info * --format *`: the info command with a --format option and a value.
func isInfoFormat(joined string) bool {
	if !strings.HasPrefix(joined, "info ") {
		return false
	}
	rest := joined[len("info "):]
	i := strings.Index(rest, " --format ")
	return i >= 0 && i+len(" --format ") <= len(rest)
}

// fixturesDir returns the fixture directory: $VFOX_PLUS_FIXTURES, or the
// vfox-1.0.13 directory next to the running executable.
func fixturesDir() string {
	if dir := os.Getenv("VFOX_PLUS_FIXTURES"); dir != "" {
		return dir
	}
	exe, err := os.Executable()
	if err != nil {
		return "vfox-1.0.13"
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Join(filepath.Dir(exe), "vfox-1.0.13")
}

// showFile writes the captured file to w, decoding the \xHH and \\
// escapes back into the bytes vfox printed, like `printf %b` does in
// fake-vfox.sh.
func showFile(path string, w io.Writer) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_, _ = w.Write(DecodePercentB(string(data)))
}

// fakeDelay returns how long to pause between install progress updates,
// from $VFOX_PLUS_FAKE_DELAY in seconds (default 0.2).
func fakeDelay() time.Duration {
	s := os.Getenv("VFOX_PLUS_FAKE_DELAY")
	if s == "" {
		s = "0.2"
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		f = 0.2
	}
	return time.Duration(f * float64(time.Second))
}

// DecodePercentB expands backslash escapes the way printf %b does: \a, \b,
// \e, \f, \n, \r, \t, \v, \\, \ooo octal, \xHH, \uHHHH and \UHHHHHHHH become
// the characters they denote, \c ends the output, and any other escape keeps
// both characters. Decoding runs in a single pass from left to right, so \\n
// decodes to a backslash followed by n rather than a newline.
func DecodePercentB(s string) []byte {
	out := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		c := s[i]
		if c != '\\' {
			out = append(out, c)
			i++
			continue
		}
		i++
		if i >= len(s) {
			out = append(out, '\\')
			break
		}
		e := s[i]
		i++
		switch e {
		case 'a':
			out = append(out, '\a')
		case 'b':
			out = append(out, '\b')
		case 'c':
			return out
		case 'e', 'E':
			out = append(out, 0x1b)
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'v':
			out = append(out, '\v')
		case '\\':
			out = append(out, '\\')
		case '\'', '"':
			out = append(out, e)
		case 'x':
			n, v := hexDigits(s, i, 2)
			if n == 0 {
				out = append(out, '\\', 'x')
			} else {
				out = append(out, byte(v))
				i += n
			}
		case 'u', 'U':
			width := 4
			if e == 'U' {
				width = 8
			}
			n, v := hexDigits(s, i, width)
			if n == 0 {
				out = append(out, '\\', e)
			} else {
				out = append(out, string(rune(v))...)
				i += n
			}
		default:
			if e >= '0' && e <= '7' {
				v := int(e - '0')
				for k := 0; k < 2 && i < len(s) && s[i] >= '0' && s[i] <= '7'; k++ {
					v = v*8 + int(s[i]-'0')
					i++
				}
				out = append(out, byte(v&0xff))
			} else {
				out = append(out, '\\', e)
			}
		}
	}
	return out
}

// hexDigits parses up to max hexadecimal digits at s[start:], returning how
// many were consumed and their value.
func hexDigits(s string, start, max int) (int, int) {
	n, v := 0, 0
	for n < max && start+n < len(s) {
		d := hexVal(s[start+n])
		if d < 0 {
			break
		}
		v = v*16 + d
		n++
	}
	return n, v
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}
