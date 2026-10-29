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

package webheap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/memoryreport"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeChromeEnv); mode != "" {
		runFakeChrome(mode)
		return
	}
	os.Exit(m.Run())
}

// trial is one run of the tool against the fake browser.
type trial struct {
	output   string
	stdout   []byte
	commands []string // "<command>\t<detail>" in the order that the browser got them
	err      error
}

// runFake runs the tool with this test binary as the browser.
func runFake(t *testing.T, mode string, extra ...string) trial {
	t.Helper()
	directory := t.TempDir()
	logPath := filepath.Join(directory, "commands.log")
	t.Setenv(fakeChromeEnv, mode)
	t.Setenv(fakeChromeLogEnv, logPath)
	args := append([]string{
		"-server", "http://127.0.0.1:9/",
		"-pairing", "http://127.0.0.1:9/auth/connect/abc",
		"-output", filepath.Join(directory, "out", "memory.txt"),
		"-chrome", os.Args[0],
		"-settle", "0", "-idle", "0", "-timeout", "30",
	}, extra...)
	var stdout bytes.Buffer
	var result trial
	result.err = run(args, &stdout)
	result.stdout = stdout.Bytes()
	if written, err := os.ReadFile(filepath.Join(directory, "out", "memory.txt")); err == nil {
		result.output = string(written)
	}
	if log, err := os.ReadFile(logPath); err == nil {
		result.commands = strings.Split(strings.TrimSpace(string(log)), "\n")
	}
	return result
}

// has reports whether the browser got a command that starts with kind and
// whose detail contains text.
func (r trial) has(kind, text string) bool {
	for _, line := range r.commands {
		if strings.HasPrefix(line, kind+"\t") && strings.Contains(line, text) {
			return true
		}
	}
	return false
}

// index is the position of the first command of the kind with the text in its
// detail, or -1.
func (r trial) index(kind, text string) int {
	for i, line := range r.commands {
		if strings.HasPrefix(line, kind+"\t") && strings.Contains(line, text) {
			return i
		}
	}
	return -1
}

func TestRunMeasures(t *testing.T) {
	got := runFake(t, "ok")
	if got.err != nil {
		t.Fatal(got.err)
	}
	want := "webheap (app): 2000 bytes\nwebheap (app-settled): 5000 bytes\n"
	if got.output != want {
		t.Errorf("output %q, want %q", got.output, want)
	}
	// What the memory report reads.
	heap := memoryreport.ParseWebHeap(got.output)
	if heap == nil || heap.App != 2000 || heap.Settled != 5000 {
		t.Errorf("the memory report reads %+v", heap)
	}
	var summary measurement
	if err := json.Unmarshal(got.stdout, &summary); err != nil {
		t.Fatalf("stdout %q: %v", got.stdout, err)
	}
	if summary != (measurement{Browser: "FakeChrome/1.2.3", DOMNodes: 321, Heap: 2000, Settled: 5000}) {
		t.Errorf("summary %+v", summary)
	}
	if !strings.Contains(string(got.stdout), `"heapUsedBytes":2000`) {
		t.Errorf("the keys of the summary changed: %s", got.stdout)
	}
	// It pairs first and opens the app after: the pairing page sets the cookie.
	pairing := got.index("Page.navigate", "/auth/connect/abc")
	app := got.index("Page.navigate", "http://127.0.0.1:9/app")
	if pairing < 0 || app < pairing {
		t.Errorf("the order of the navigation: pairing %d, app %d in %q", pairing, app, got.commands)
	}
	for _, kind := range []string{"Page.captureScreenshot", "Emulation.setEmulatedMedia"} {
		if got.has(kind, "") {
			t.Errorf("%s without a screenshot", kind)
		}
	}
	if got.has("Runtime.evaluate", "localStorage") {
		t.Error("the language was pinned without -lang")
	}
	if !got.has("start", "--window-size=1280,800") || !got.has("start", "--headless=new") || !got.has("start", "--no-sandbox") {
		t.Errorf("the browser got the wrong command line: %q", got.commands[0])
	}
}

func TestRunCleansUp(t *testing.T) {
	got := runFake(t, "ok")
	if got.err != nil {
		t.Fatal(got.err)
	}
	var profile string
	var pid int
	for _, line := range got.commands {
		if args, ok := strings.CutPrefix(line, "start\t"); ok {
			for _, arg := range strings.Fields(args) {
				if value, ok := strings.CutPrefix(arg, "--user-data-dir="); ok {
					profile = value
				}
			}
		}
		if value, ok := strings.CutPrefix(line, "pid\t"); ok {
			pid, _ = strconv.Atoi(value)
		}
	}
	if profile == "" || pid == 0 {
		t.Fatalf("no profile or pid in %q", got.commands)
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Errorf("the profile %s is still there: %v", profile, err)
	}
	if runtime.GOOS != "windows" {
		if process, err := os.FindProcess(pid); err == nil && process.Signal(syscall.Signal(0)) == nil {
			t.Errorf("the browser %d is still running", pid)
		}
	}
}

func TestRunScreenshot(t *testing.T) {
	directory := t.TempDir()
	shot := filepath.Join(directory, "shots", "webapp.png")
	got := runFake(t, "ok", "-screenshot", shot, "-lang", "zh", "-tab", "plugins", "-dark", "-window", "412,915")
	if got.err != nil {
		t.Fatal(got.err)
	}
	image, err := os.ReadFile(shot)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(image, fakeScreenshot) {
		t.Errorf("the screenshot has %d bytes, want %d", len(image), len(fakeScreenshot))
	}
	if !got.has("start", "--window-size=412,915") {
		t.Errorf("the window: %q", got.commands[0])
	}
	// The language is pinned right after the app opens, before anything is read.
	app := got.index("Page.navigate", "/app")
	pin := got.index("Runtime.evaluate", `localStorage.setItem("vfox-plus-lang", "zh")`)
	list := got.index("Runtime.evaluate", "#sdkList")
	if !(app < pin && pin < list) {
		t.Errorf("language %d is not between the app %d and the SDK list %d", pin, app, list)
	}
	// Dark first, then the page, then the picture.
	dark := got.index("Emulation.setEmulatedMedia", `"prefers-color-scheme"`)
	tab := got.index("Runtime.evaluate", `[data-tab="plugins"]`)
	capture := got.index("Page.captureScreenshot", `"png"`)
	if !(0 < dark && dark < tab && tab < capture) {
		t.Errorf("dark %d, tab %d, capture %d", dark, tab, capture)
	}
}

func TestRunReadsAFragmentedScreenshot(t *testing.T) {
	shot := filepath.Join(t.TempDir(), "webapp.png")
	got := runFake(t, "fragmented", "-screenshot", shot)
	if got.err != nil {
		t.Fatal(got.err)
	}
	if image, err := os.ReadFile(shot); err != nil || !bytes.Equal(image, fakeScreenshot) {
		t.Errorf("the screenshot: %d bytes, %v", len(image), err)
	}
}

func TestRunWaitsForTheFirstPage(t *testing.T) {
	if got := runFake(t, "late-page"); got.err != nil {
		t.Fatal(got.err)
	}
}

func TestRunFailures(t *testing.T) {
	for _, tc := range []struct {
		mode string
		args []string
		want string
	}{
		{"exit", nil, "exited before it listened: fake chrome: cannot open a display"},
		{"navigate-error", nil, "cannot open the app: net::ERR_CONNECTION_REFUSED"},
		{"redirect", nil, "the app did not open: the browser ended up on http://127.0.0.1:1/"},
		{"never-ready", []string{"-timeout", "2"}, "timed out waiting for the SDK list"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			started := time.Now()
			got := runFake(t, tc.mode, tc.args...)
			if got.err == nil || !strings.Contains(got.err.Error(), tc.want) {
				t.Fatalf("got %v, want an error with %q", got.err, tc.want)
			}
			if got.output != "" {
				t.Errorf("a failed run wrote %q", got.output)
			}
			if elapsed := time.Since(started); elapsed > 20*time.Second {
				t.Errorf("took %s", elapsed)
			}
		})
	}
}

func TestRunWithoutAChrome(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CHROME_BIN", "")
	err := run([]string{"-server", "http://127.0.0.1:9", "-pairing", "http://127.0.0.1:9/auth/connect/abc", "-output", filepath.Join(t.TempDir(), "memory.txt")}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no Chrome found") {
		t.Fatalf("got %v", err)
	}
}

func TestParseOptions(t *testing.T) {
	base := []string{"-server", "http://127.0.0.1:1", "-pairing", "http://127.0.0.1:1/auth/connect/x", "-output", "memory.txt"}
	with := func(extra ...string) []string { return append(append([]string{}, base...), extra...) }

	got, err := parseOptions(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.width != 1280 || got.height != 800 || got.settle != 5*time.Second || got.idle != 20*time.Second || got.timeout != 180*time.Second || got.dark {
		t.Errorf("defaults: %+v", got)
	}
	got, err = parseOptions(with("--settle", "10", "--window", "412,915", "--lang", "en", "--tab", "tasks", "--dark", "--timeout", "7.5", "--idle", "0"))
	if err != nil {
		t.Fatal(err)
	}
	if got.settle != 10*time.Second || got.width != 412 || got.height != 915 || got.lang != "en" || got.tab != "tasks" || !got.dark ||
		got.timeout != 7500*time.Millisecond || got.idle != 0 {
		t.Errorf("parsed: %+v", got)
	}

	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"no flags":         {nil, "Usage: webheap"},
		"no output":        {base[:4], "Usage: webheap"},
		"negative settle":  {with("-settle", "-1"), "bad -settle"},
		"negative idle":    {with("-idle", "-1"), "bad -idle"},
		"settle not a num": {with("-settle", "NaN"), "bad -settle"},
		"zero timeout":     {with("-timeout", "0"), "bad -timeout"},
		"window":           {with("-window", "412x915"), "bad -window"},
		"tiny window":      {with("-window", "4,9"), "bad -window"},
		"language":         {with("-lang", "de"), "bad -lang"},
		"tab":              {with("-tab", "x\"]"), "bad -tab"},
		"stray argument":   {with("extra"), "unexpected argument"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseOptions(tc.args); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("got %v, want an error with %q", err, tc.want)
			}
		})
	}
}

func TestFindChrome(t *testing.T) {
	t.Setenv("CHROME_BIN", "/from/env")
	if got, err := findChrome("/named"); err != nil || got != "/named" {
		t.Errorf("a named browser: %q, %v", got, err)
	}
	if got, err := findChrome(""); err != nil || got != "/from/env" {
		t.Errorf("$CHROME_BIN: %q, %v", got, err)
	}
	t.Setenv("CHROME_BIN", "")
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	if _, err := findChrome(""); err == nil && runtime.GOOS != "darwin" {
		t.Error("found a Chrome in an empty PATH")
	}
	if err := os.WriteFile(filepath.Join(bin, "chromium"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if got, err := findChrome(""); err != nil || got != filepath.Join(bin, "chromium") {
			t.Errorf("chromium in PATH: %q, %v", got, err)
		}
	}
}

func TestStderrWatcher(t *testing.T) {
	watcher := &stderrWatcher{endpoint: make(chan string, 1)}
	// The line comes in pieces, behind other output.
	for _, piece := range []string{"[0101/000000:ERROR] noise\nDevTools lis", "tening on ws://127.0.0.1:41", "234/devtools/browser/abc-def\nmore\n"} {
		if n, err := watcher.Write([]byte(piece)); n != len(piece) || err != nil {
			t.Fatalf("write: %d, %v", n, err)
		}
	}
	select {
	case got := <-watcher.endpoint:
		if got != "ws://127.0.0.1:41234/devtools/browser/abc-def" {
			t.Errorf("got %q", got)
		}
	default:
		t.Fatal("no endpoint")
	}
	if tail := watcher.lastOutput(); !strings.HasSuffix(tail, "more") || len(tail) > tailSize {
		t.Errorf("tail %q", tail)
	}
	// Only the last bytes are kept.
	_, _ = watcher.Write([]byte(strings.Repeat("x", 3*tailSize) + "\nend\n"))
	if tail := watcher.lastOutput(); len(tail) > tailSize || !strings.HasSuffix(tail, "end") {
		t.Errorf("tail of %d bytes: %q", len(tail), tail)
	}
}

func TestChoosePage(t *testing.T) {
	blank := devToolsTarget{Type: "page", URL: "about:blank", WebSocketDebuggerURL: "ws://blank"}
	shown := devToolsTarget{Type: "page", URL: "http://example.test/", WebSocketDebuggerURL: "ws://shown"}
	worker := devToolsTarget{Type: "service_worker", URL: "http://example.test/sw.js", WebSocketDebuggerURL: "ws://worker"}
	unreachable := devToolsTarget{Type: "page", URL: "http://example.test/"}
	for name, tc := range map[string]struct {
		targets []devToolsTarget
		want    string
	}{
		"none":              {nil, ""},
		"only a worker":     {[]devToolsTarget{worker}, ""},
		"only a blank page": {[]devToolsTarget{worker, blank}, "ws://blank"},
		"a page that shows": {[]devToolsTarget{blank, shown}, "ws://shown"},
		"no debugger":       {[]devToolsTarget{unreachable, blank}, "ws://blank"},
	} {
		if got := choosePage(tc.targets); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestMedianAndSeconds(t *testing.T) {
	if got := median([]int64{3, 1, 2}); got != 2 {
		t.Errorf("median %d", got)
	}
	samples := []int64{9, 4, 7}
	_ = median(samples)
	if samples[0] != 9 {
		t.Error("median sorted its argument")
	}
	if _, err := seconds("x", 1e9, true); err == nil {
		t.Error("a billion seconds is not a timeout")
	}
}

func TestSleepStopsWithTheContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := sleep(ctx, time.Hour); err == nil {
		t.Error("slept an hour")
	}
	if err := sleep(context.Background(), 0); err != nil {
		t.Error(err)
	}
}
