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

// Package webheap implements `vfxtool webheap`: it measures the JavaScript
// heap of the paired web interface (/app) with the Chrome DevTools Protocol
// and writes the lines of the memory report: `webheap (app)` after the SDK
// list settles, and `webheap (app-settled)` after idling on the open page,
// which catches leaks in the live event stream. It also takes the screenshots
// of the web interface for the documentation.
//
// The caller starts the server and passes its pairing link; the tool only
// drives one throw-away browser profile. Desktop and server memory are not
// measured here, only the web client in front of a human. The tool needs
// nothing but a Chrome that speaks the DevTools Protocol; it replaces a Node.js
// script and speaks the protocol with a WebSocket client of its own (see
// websocket.go).
package webheap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const usage = "Usage: webheap -server URL -pairing URL -output FILE [-settle SECONDS] [-idle SECONDS] [-timeout SECONDS] [-screenshot FILE] [-lang en|zh] [-window WIDTH,HEIGHT] [-tab NAME | -scene NAME] [-dark] [-chrome PATH]"

var windowSize = regexp.MustCompile(`^([0-9]{2,5}),([0-9]{2,5})$`)

// tabs are the pages that -tab can open: the data-tab attributes of the
// navigation of the web interface.
var tabs = []string{"home", "sdks", "plugins", "tasks", "settings"}

// The expressions that run in the page.
const (
	// sdkListReady is true once the SDK list shows its rows and no spinner.
	sdkListReady = `(() => { const list = document.querySelector("#sdkList"); return list && list.children.length > 0 && !list.querySelector(".loader"); })()`
	domNodes     = `document.querySelectorAll("*").length`
)

// options is the command line.
type options struct {
	server     string
	pairing    string
	output     string
	screenshot string
	lang       string
	tab        string
	scene      string
	chrome     string
	width      int
	height     int
	settle     time.Duration
	idle       time.Duration
	timeout    time.Duration
	dark       bool
}

// seconds converts a number of seconds of a flag.
func seconds(name string, value float64, zeroAllowed bool) (time.Duration, error) {
	if math.IsNaN(value) || value < 0 || value > 1e7 || (value == 0 && !zeroAllowed) {
		return 0, fmt.Errorf("bad -%s", name)
	}
	return time.Duration(value * float64(time.Second)), nil
}

func parseOptions(args []string) (options, error) {
	var o options
	flags := flag.NewFlagSet("webheap", flag.ContinueOnError)
	flags.StringVar(&o.server, "server", "", "base `URL` of the running server, like http://127.0.0.1:PORT")
	flags.StringVar(&o.pairing, "pairing", "", "the one-time pairing `URL` that the server printed")
	flags.StringVar(&o.output, "output", "", "`file` that gets the two webheap lines")
	flags.StringVar(&o.screenshot, "screenshot", "", "also write a PNG screenshot of the settled page to this `file`")
	flags.StringVar(&o.lang, "lang", "", "pin the interface language to `en|zh`")
	flags.StringVar(&o.tab, "tab", "", "open this page before the screenshot: home, sdks, plugins, tasks or settings")
	flags.StringVar(&o.scene, "scene", "", "play this scene before the screenshot, which opens a dialog or a tab of a plugin: "+strings.Join(sceneNames(), ", "))
	flags.StringVar(&o.chrome, "chrome", "", "Chrome `executable` (default: $CHROME_BIN, then google-chrome, chromium, chromium-browser)")
	window := flags.String("window", "1280,800", "browser window `WIDTH,HEIGHT`")
	settle := flags.Float64("settle", 5, "`seconds` to settle after the SDK list loads, before the first measurement")
	idle := flags.Float64("idle", 20, "`seconds` to idle on the open page before the second measurement")
	timeout := flags.Float64("timeout", 180, "`seconds` that the whole run may take")
	flags.BoolVar(&o.dark, "dark", false, "emulate the dark colour scheme for the screenshot")
	if err := flags.Parse(args); err != nil {
		return o, err
	}
	if flags.NArg() != 0 {
		return o, fmt.Errorf("unexpected argument %q\n%s", flags.Arg(0), usage)
	}
	if o.server == "" || o.pairing == "" || o.output == "" {
		return o, errors.New(usage)
	}
	var err error
	if o.settle, err = seconds("settle", *settle, true); err != nil {
		return o, err
	}
	if o.idle, err = seconds("idle", *idle, true); err != nil {
		return o, err
	}
	if o.timeout, err = seconds("timeout", *timeout, false); err != nil {
		return o, err
	}
	match := windowSize.FindStringSubmatch(*window)
	if match == nil {
		return o, errors.New("bad -window (want WIDTH,HEIGHT)")
	}
	o.width, _ = strconv.Atoi(match[1])
	o.height, _ = strconv.Atoi(match[2])
	if o.lang != "" && o.lang != "en" && o.lang != "zh" {
		return o, errors.New("bad -lang (want en or zh)")
	}
	if o.tab != "" && !contains(tabs, o.tab) {
		return o, fmt.Errorf("bad -tab (want one of %s)", strings.Join(tabs, ", "))
	}
	if o.scene != "" {
		if _, known := scenes[o.scene]; !known {
			return o, fmt.Errorf("bad -scene (want one of %s)", strings.Join(sceneNames(), ", "))
		}
		if o.tab != "" || o.screenshot == "" {
			return o, errors.New("a -scene goes with a -screenshot, and instead of a -tab")
		}
	}
	return o, nil
}

func contains(list []string, item string) bool {
	for _, entry := range list {
		if entry == item {
			return true
		}
	}
	return false
}

// measurement is what a run found, and the line that it prints.
type measurement struct {
	Browser  string `json:"chrome"`
	DOMNodes int    `json:"domNodes"`
	Heap     int64  `json:"heapUsedBytes"`
	Settled  int64  `json:"heapSettledBytes"`
}

// formatLines is the content of the output file, which
// memoryreport.ParseWebHeap reads.
func formatLines(heap, settled int64) string {
	return fmt.Sprintf("webheap (app): %d bytes\nwebheap (app-settled): %d bytes\n", heap, settled)
}

// Run implements `webheap`. The flags:
//
//	-server URL        base URL of the running server
//	-pairing URL       the one-time pairing link it printed
//	-output FILE       gets `webheap (app): N bytes` and `webheap (app-settled): N bytes`
//	-settle SECONDS    settle time after the SDK list loads (default 5)
//	-idle SECONDS      idle time on the open page before the second reading (default 20)
//	-timeout SECONDS   time that the whole run may take (default 180)
//	-screenshot FILE   also write a PNG of the settled page
//	-lang en|zh        pin the interface language (default: the browser's)
//	-window W,H        browser window (default 1280,800)
//	-tab NAME          open home, sdks, plugins, tasks or settings before the screenshot
//	-scene NAME        or play a scene: the details of a plugin, a dialog (see scenes.go)
//	-dark              emulate the dark colour scheme for the screenshot
//	-chrome PATH       Chrome executable (default: $CHROME_BIN, then PATH)
func Run(args []string) error {
	return run(args, os.Stdout)
}

// run is Run with the place for the summary line.
func run(args []string, stdout io.Writer) error {
	o, err := parseOptions(args)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	found, err := measure(ctx, o)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(o.output), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(o.output, []byte(formatLines(found.Heap, found.Settled)), 0o644); err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(found)
}

// median is the middle one of the sorted samples.
func median(samples []int64) int64 {
	sorted := append([]int64(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}

// sleep waits for d, or fails when the run has run out of time.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// session is the page that is driven.
type session struct {
	page *cdpClient
}

// open navigates to address and waits for the page to load. It then checks
// where the browser ended up: a server that redirects /app to its intro page
// (the pairing link did not work) would otherwise show as a long wait for an
// SDK list that never comes.
func (s *session) open(ctx context.Context, address, label string) error {
	loaded := s.page.expect("Page.loadEventFired")
	var navigation struct {
		ErrorText string `json:"errorText"`
	}
	if err := s.page.call(ctx, "Page.navigate", map[string]string{"url": address}, &navigation); err != nil {
		return fmt.Errorf("opening %s: %w", label, err)
	}
	if navigation.ErrorText != "" {
		return fmt.Errorf("cannot open %s: %s", label, navigation.ErrorText)
	}
	if err := s.page.await(ctx, loaded); err != nil {
		return fmt.Errorf("waiting for %s: %w", label, err)
	}
	var href string
	if err := s.page.evaluate(ctx, "location.href", &href); err != nil {
		return fmt.Errorf("reading the address of %s: %w", label, err)
	}
	if !strings.HasPrefix(href, address) {
		return fmt.Errorf("%s did not open: the browser ended up on %s (was the pairing link used before?)", label, href)
	}
	return nil
}

// waitFor polls a JavaScript expression until it is true. An evaluation that
// fails, as one does while the page is still loading, is tried again.
func (s *session) waitFor(ctx context.Context, expression, label string) error {
	for {
		var ready bool
		if err := s.page.evaluate(ctx, expression, &ready); err == nil && ready {
			return nil
		}
		if err := sleep(ctx, 500*time.Millisecond); err != nil {
			return fmt.Errorf("timed out waiting for %s", label)
		}
	}
}

// heap reads the used JavaScript heap three times and takes the median.
func (s *session) heap(ctx context.Context) (int64, error) {
	samples := make([]int64, 3)
	for i := range samples {
		var usage struct {
			UsedSize float64 `json:"usedSize"`
		}
		if err := s.page.call(ctx, "Runtime.getHeapUsage", nil, &usage); err != nil {
			return 0, err
		}
		samples[i] = int64(math.Round(usage.UsedSize))
	}
	return median(samples), nil
}

// capture writes a PNG of the page, in the dark colour scheme and on the page
// or in the scene that the options name.
func (s *session) capture(ctx context.Context, o options) error {
	if o.dark {
		features := map[string]any{"features": []map[string]string{{"name": "prefers-color-scheme", "value": "dark"}}}
		if err := s.page.call(ctx, "Emulation.setEmulatedMedia", features, nil); err != nil {
			return err
		}
	}
	if o.tab != "" {
		// Open the page through whichever navigation is shown (rail or bar).
		click := fmt.Sprintf(`[...document.querySelectorAll('[data-tab="%s"]')].find((b) => b.offsetParent)?.click()`, o.tab)
		if err := s.page.evaluate(ctx, click, nil); err != nil {
			return err
		}
		if err := sleep(ctx, 1200*time.Millisecond); err != nil {
			return err
		}
	}
	if o.scene != "" {
		if err := s.play(ctx, o.scene); err != nil {
			return err
		}
		if err := sleep(ctx, 900*time.Millisecond); err != nil {
			return err
		}
	}
	var shot struct {
		Data string `json:"data"`
	}
	if err := s.page.call(ctx, "Page.captureScreenshot", map[string]string{"format": "png"}, &shot); err != nil {
		return err
	}
	image, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		return fmt.Errorf("the screenshot is not base64: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(o.screenshot), 0o755); err != nil {
		return err
	}
	return os.WriteFile(o.screenshot, image, 0o644)
}

// measure runs the whole measurement in one browser.
func measure(ctx context.Context, o options) (measurement, error) {
	var found measurement
	binary, err := findChrome(o.chrome)
	if err != nil {
		return found, err
	}
	profile, err := os.MkdirTemp("", "vfox-webheap-")
	if err != nil {
		return found, err
	}
	defer func() { _ = os.RemoveAll(profile) }()
	chrome, err := startBrowser(ctx, binary, chromeArgs(o.width, o.height, profile))
	if err != nil {
		return found, err
	}
	defer chrome.stop()
	socket, err := pageSocket(ctx, chrome.endpoint)
	if err != nil {
		return found, fmt.Errorf("no page to drive: %w", err)
	}
	ws, err := dialWebSocket(ctx, socket)
	if err != nil {
		return found, err
	}
	page := newCDPClient(ws)
	defer page.close()
	s := &session{page: page}

	for _, domain := range []string{"Page.enable", "Runtime.enable"} {
		if err := page.call(ctx, domain, nil, nil); err != nil {
			return found, err
		}
	}
	// Pair first, exactly like a browser: the pairing page sets the session
	// cookie, and /app opens paired.
	if err := s.open(ctx, o.pairing, "the pairing page"); err != nil {
		return found, err
	}
	app := strings.TrimSuffix(o.server, "/") + "/app"
	if err := s.open(ctx, app, "the app"); err != nil {
		return found, err
	}
	if o.lang != "" {
		// Pin the language before settling, so that screenshots are deterministic.
		language, _ := json.Marshal(o.lang)
		reloaded := page.expect("Page.loadEventFired")
		pin := fmt.Sprintf(`localStorage.setItem("vfox-plus-lang", %s); location.reload();`, language)
		if err := page.evaluate(ctx, pin, nil); err != nil {
			return found, err
		}
		if err := page.await(ctx, reloaded); err != nil {
			return found, fmt.Errorf("waiting for the app to reload: %w", err)
		}
	}
	if err := s.waitFor(ctx, sdkListReady, "the SDK list"); err != nil {
		return found, err
	}
	if err := sleep(ctx, o.settle); err != nil {
		return found, err
	}
	if o.screenshot != "" {
		// A screenshot of the settled page, for the documentation; the memory
		// measurement does not need it.
		if err := s.capture(ctx, o); err != nil {
			return found, err
		}
	}
	if found.Heap, err = s.heap(ctx); err != nil {
		return found, err
	}
	if err := page.evaluate(ctx, domNodes, &found.DOMNodes); err != nil {
		return found, err
	}
	// Idle on the open page with live server-sent events: this catches leaks.
	if err := sleep(ctx, o.idle); err != nil {
		return found, err
	}
	if found.Settled, err = s.heap(ctx); err != nil {
		return found, err
	}
	found.Browser = "unknown"
	var version struct {
		Product string `json:"product"`
	}
	if err := page.call(ctx, "Browser.getVersion", nil, &version); err == nil && version.Product != "" {
		found.Browser = version.Product // Best effort: a page target may refuse.
	}
	return found, nil
}
