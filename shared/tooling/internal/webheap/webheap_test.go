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
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/memoryreport"
)

// The tests that run the tool drive a real Chrome (the one that findChrome
// finds: -chrome, $CHROME_BIN, or a name in PATH) against a small page that
// does what the web interface does for the tool: it sets the session cookie at
// the pairing link, lists SDKs after a moment, and reports to the server what
// the tool does to it. They are skipped on a machine without a Chrome.

// testPage is the web interface of the tests.
const testPage = `<!doctype html><meta charset="utf-8"><title>vfox-plus</title>
<nav>
  <button data-tab="home">Home</button><button data-tab="sdks">SDKs</button><button data-tab="plugins">Plugins</button>
  <button data-tab="tasks">Tasks</button><button data-tab="settings">Settings</button>
</nav>
<div id="sdkList"><div class="loader"></div></div>
<div id="detail"></div>
<script>
const note = (what) => navigator.sendBeacon("/note?" + encodeURIComponent(what));
const dark = matchMedia("(prefers-color-scheme: dark)");
note("dark-at-start=" + dark.matches);
dark.addEventListener("change", (e) => note("dark=" + e.matches));
const lang = localStorage.getItem("vfox-plus-lang");
if (lang) note("lang=" + lang);
for (const button of document.querySelectorAll("nav [data-tab]")) button.onclick = () => note("tab=" + button.dataset.tab);
const showList = LISTED;
setTimeout(() => {
  if (!showList) return;
  const list = document.getElementById("sdkList");
  list.innerHTML = "";
  for (const name of ["nodejs", "java"]) {
    const row = document.createElement("div");
    row.dataset.sdk = name;
    row.textContent = name;
    row.onclick = () => openDetail(name);
    list.append(row);
  }
}, 200);
function openDetail(name) {
  note("sdk=" + name);
  document.getElementById("detail").innerHTML =
    '<div id="detailTabs"><button data-tab="installed">Installed</button><button data-tab="options">Options</button></div><div id="body"></div>';
  for (const button of document.querySelectorAll("#detailTabs [data-tab]")) {
    button.onclick = () => { note("detail=" + button.dataset.tab); if (button.dataset.tab === "options") showOptions(); };
  }
}
function showOptions() {
  document.getElementById("body").innerHTML = '<button data-var="VFOX_NODEJS_MIRROR">mirror</button><div id="varChips"></div>';
  document.querySelector("[data-var]").onclick = () => {
    note("var=VFOX_NODEJS_MIRROR");
    document.getElementById("varChips").innerHTML = '<button data-value="https://npmmirror.com/mirrors/node">npmmirror</button>';
    document.querySelector("#varChips [data-value]").onclick = () => note("preset=npmmirror");
  };
}
</script>`

// server is a fake vfox-plus: its pairing link, /app, and a place where the page
// reports what happens to it.
type server struct {
	*httptest.Server
	mu       sync.Mutex
	requests []string // "GET /auth/connect/abc" and the like, in order
	notes    []string // what the page reported
}

type serverMode int

const (
	serving   serverMode = iota // the way it should work
	redirects                   // sends /app to the intro page, as for an unpaired browser
	neverList                   // shows no SDK list, ever
	noScenes                    // lists no SDK to click
)

func newServer(t *testing.T, mode serverMode) *server {
	t.Helper()
	s := &server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/connect/abc", func(w http.ResponseWriter, r *http.Request) {
		s.record("GET " + r.URL.Path)
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "paired", Path: "/"})
		_, _ = fmt.Fprint(w, "<!doctype html><title>paired</title>paired")
	})
	mux.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		s.record("GET " + r.URL.Path)
		if _, err := r.Cookie("session"); err != nil || mode == redirects {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		listed := "true"
		if mode == neverList || mode == noScenes {
			listed = "false"
		}
		if mode == noScenes {
			// The list has rows, but not the ones that the scenes click.
			_, _ = fmt.Fprint(w, strings.Replace(testPage, `<div class="loader"></div>`, `<div class="row">only this</div>`, 1))
			return
		}
		_, _ = fmt.Fprint(w, strings.Replace(testPage, "LISTED", listed, 1))
	})
	mux.HandleFunc("/note", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.notes = append(s.notes, r.URL.RawQuery)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.record("GET " + r.URL.Path)
		_, _ = fmt.Fprint(w, "<!doctype html><title>intro</title>Open a pairing link.")
	})
	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

func (s *server) record(request string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, request)
}

// seen are the notes of the page, decoded, after waiting a moment for the last
// ones, which the page sends in the background.
func (s *server) seen(t *testing.T, want ...string) []string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		s.mu.Lock()
		notes := make([]string, len(s.notes))
		for i, note := range s.notes {
			notes[i], _ = unescape(note)
		}
		s.mu.Unlock()
		missing := false
		for _, w := range want {
			if !contains(notes, w) {
				missing = true
			}
		}
		if !missing || time.Now().After(deadline) {
			return notes
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func unescape(note string) (string, error) {
	return strings.NewReplacer("%3D", "=", "%3A", ":", "%2F", "/").Replace(note), nil
}

// chrome is the browser of the tests, or the end of the test.
func chrome(t *testing.T) string {
	t.Helper()
	binary, err := findChrome("")
	if err != nil {
		t.Skip("no Chrome to drive: set CHROME_BIN")
	}
	return binary
}

// trial is one run of the tool.
type trial struct {
	output string
	stdout []byte
	err    error
}

func runTool(t *testing.T, s *server, extra ...string) trial {
	t.Helper()
	directory := t.TempDir()
	args := append([]string{
		"-server", s.URL, "-pairing", s.URL + "/auth/connect/abc",
		"-output", filepath.Join(directory, "out", "memory.txt"),
		"-chrome", chrome(t),
		"-settle", "0", "-idle", "0.5", "-timeout", "60",
	}, extra...)
	var stdout bytes.Buffer
	var result trial
	result.err = run(args, &stdout)
	result.stdout = stdout.Bytes()
	if written, err := os.ReadFile(filepath.Join(directory, "out", "memory.txt")); err == nil {
		result.output = string(written)
	}
	return result
}

func TestRunMeasures(t *testing.T) {
	s := newServer(t, serving)
	got := runTool(t, s)
	if got.err != nil {
		t.Fatal(got.err)
	}
	// What the memory report reads.
	heap := memoryreport.ParseWebHeap(got.output)
	if heap == nil || heap.App <= 0 || heap.Settled <= 0 {
		t.Errorf("the memory report reads %+v from %q", heap, got.output)
	}
	var summary measurement
	if err := json.Unmarshal(got.stdout, &summary); err != nil {
		t.Fatalf("stdout %q: %v", got.stdout, err)
	}
	if !strings.Contains(summary.Browser, "hrome") || summary.DOMNodes < 10 || summary.Heap != heap.App || summary.Settled != heap.Settled {
		t.Errorf("summary %+v, heap %+v", summary, heap)
	}
	if !strings.Contains(string(got.stdout), `"heapUsedBytes":`) {
		t.Errorf("the keys of the summary changed: %s", got.stdout)
	}
	// It pairs first and opens the app after: the pairing page sets the cookie.
	// (The browser asks for the icon of each page, too.)
	pairing, app := indexOf(s.requests, "GET /auth/connect/abc"), indexOf(s.requests, "GET /app")
	if pairing < 0 || app < pairing {
		t.Errorf("the order of the requests: %q", s.requests)
	}
	for _, note := range s.seen(t) {
		if strings.HasPrefix(note, "lang=") {
			t.Errorf("the language was pinned without -lang: %q", note)
		}
	}
}

func TestRunCleansUp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the temporary directory is TMP there")
	}
	// The browser makes its profile below the temporary directory. The name is
	// short, because Chrome makes a socket in the profile, and the path of a socket
	// may not be longer than about 100 bytes (the directory of t.TempDir is too long
	// for that where $TMPDIR is, as on macOS).
	temporary, err := os.MkdirTemp("", "wh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(temporary) })
	t.Setenv("TMPDIR", temporary)
	got := runTool(t, newServer(t, serving))
	if got.err != nil {
		t.Fatal(got.err)
	}
	// (A browser that is killed leaves the directory of its shared memory, which is not the profile.)
	profiles, err := filepath.Glob(filepath.Join(temporary, "chromedp-runner*"))
	if err != nil || len(profiles) != 0 {
		t.Errorf("the profile is left behind: %v (%v)", profiles, err)
	}
}

func TestRunScreenshot(t *testing.T) {
	s := newServer(t, serving)
	shot := filepath.Join(t.TempDir(), "shots", "webapp.png")
	got := runTool(t, s, "-screenshot", shot, "-lang", "zh", "-tab", "plugins", "-dark", "-window", "412,915")
	if got.err != nil {
		t.Fatal(got.err)
	}
	file, err := os.Open(shot)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	image, err := png.Decode(file)
	if err != nil || image.Bounds().Dx() == 0 || image.Bounds().Dy() == 0 {
		t.Fatalf("the screenshot is not a picture: %v", err)
	}
	// The language is pinned, the colour scheme is dark and the page of the tab is open.
	notes := s.seen(t, "lang=zh", "dark=true", "tab=plugins")
	for _, want := range []string{"dark-at-start=false", "lang=zh", "dark=true", "tab=plugins"} {
		if !contains(notes, want) {
			t.Errorf("the page did not hear %q: %q", want, notes)
		}
	}
}

func TestRunPlaysTheSceneBeforeTheScreenshot(t *testing.T) {
	s := newServer(t, serving)
	shot := filepath.Join(t.TempDir(), "options-mirror.png")
	got := runTool(t, s, "-screenshot", shot, "-scene", "options-mirror", "-lang", "zh")
	if got.err != nil {
		t.Fatal(got.err)
	}
	if info, err := os.Stat(shot); err != nil || info.Size() == 0 {
		t.Fatalf("the screenshot: %v", err)
	}
	// The plugin, its tab and the dialog, in the order of a person.
	want := []string{"sdk=nodejs", "detail=options", "var=VFOX_NODEJS_MIRROR", "preset=npmmirror"}
	notes := s.seen(t, want...)
	last := -1
	for _, w := range want {
		at := indexOf(notes, w)
		if at < 0 || at < last {
			t.Fatalf("the scene did not do %q in order: %q", w, notes)
		}
		last = at
	}
}

func TestRunFailsWhenTheSceneFindsNothingToClick(t *testing.T) {
	shot := filepath.Join(t.TempDir(), "options.png")
	got := runTool(t, newServer(t, noScenes), "-screenshot", shot, "-scene", "options", "-timeout", "5")
	if got.err == nil || !strings.Contains(got.err.Error(), "the scene options (step 1") {
		t.Errorf("got %v, want the step that found nothing", got.err)
	}
	if _, err := os.Stat(shot); err == nil {
		t.Error("no picture of a page that is not the scene")
	}
}

func TestRunFailures(t *testing.T) {
	for _, tc := range []struct {
		label string
		mode  serverMode
		args  []string
		want  string
	}{
		{"the app sends an unpaired browser away", redirects, nil, "the app did not open: the browser ended up on " + "http://127.0.0.1:"},
		{"the SDK list never comes", neverList, []string{"-timeout", "4"}, "timed out waiting for the SDK list"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			started := time.Now()
			got := runTool(t, newServer(t, tc.mode), tc.args...)
			if got.err == nil || !strings.Contains(got.err.Error(), tc.want) {
				t.Fatalf("got %v, want an error with %q", got.err, tc.want)
			}
			if got.output != "" {
				t.Errorf("a failed run wrote %q", got.output)
			}
			if elapsed := time.Since(started); elapsed > 30*time.Second {
				t.Errorf("took %s", elapsed)
			}
		})
	}
}

func TestRunFailsWhenTheServerIsGone(t *testing.T) {
	s := newServer(t, serving)
	s.Close()
	got := runTool(t, s)
	if got.err == nil || !strings.Contains(got.err.Error(), "cannot open the pairing page") || !strings.Contains(got.err.Error(), "ERR_CONNECTION_REFUSED") {
		t.Errorf("got %v", got.err)
	}
}

func TestRunFailsWhenTheBrowserDoesNotStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake browser is a shell script")
	}
	fake := filepath.Join(t.TempDir(), "chrome")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho 'fake chrome: cannot open a display' >&2\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"-server", "http://127.0.0.1:9", "-pairing", "http://127.0.0.1:9/auth/connect/abc",
		"-output", filepath.Join(t.TempDir(), "memory.txt"), "-chrome", fake, "-timeout", "30"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "cannot open a display") {
		t.Errorf("got %v, want the message of the browser", err)
	}
}

func indexOf(list []string, item string) int {
	for i, entry := range list {
		if entry == item {
			return i
		}
	}
	return -1
}

func TestScenesUseWhatThePageHas(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "mygo", "pair", "webapp.html"))
	if err != nil {
		t.Fatal(err)
	}
	// The identifiers, classes and attributes that the steps name, as the page spells them.
	// The values of the data (the name of a plugin, an address) are not the page's.
	token := regexp.MustCompile(`#([A-Za-z][A-Za-z0-9]*)|getElementById\("([A-Za-z0-9]+)"\)|[\s"']\.([a-z]+)\b|\[(data-[a-z]+)`)
	for _, name := range sceneNames() {
		if len(scenes[name]) == 0 {
			t.Errorf("scene %s has no steps", name)
		}
		for i, move := range scenes[name] {
			if (move.click == "") == (move.wait == "") {
				t.Errorf("step %d of %s must click or wait", i+1, name)
			}
			for _, found := range token.FindAllStringSubmatch(move.click+move.wait, -1) {
				for _, word := range found[1:] {
					if word != "" && !bytes.Contains(page, []byte(word)) {
						t.Errorf("step %d of %s uses %q, which webapp.html does not have", i+1, name, word)
					}
				}
			}
		}
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

	got, err = parseOptions(with("-screenshot", "x.png", "-scene", "java-temurin"))
	if err != nil || got.scene != "java-temurin" {
		t.Errorf("a scene: %+v, %v", got, err)
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
		"scene":            {with("-screenshot", "x.png", "-scene", "nope"), "bad -scene"},
		"scene alone":      {with("-scene", "options"), "goes with a -screenshot"},
		"scene and tab":    {with("-screenshot", "x.png", "-scene", "options", "-tab", "tasks"), "instead of a -tab"},
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
