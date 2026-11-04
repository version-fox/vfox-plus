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

package ui

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// The size of the windows of the tests: wide enough for the rail and two panes,
// and small, because every frame is drawn on the CPU.
const (
	windowWidth  = 1000
	windowHeight = 700
)

// recorder is the desktop of the tests: it notes what the application asks of
// it, and opens nothing.
type recorder struct {
	copied []string
	opened []string
	chosen string
	asked  []chooseOptions
	// quits counts the times the application was asked to end.
	quits int
	// windows are the windows that the application opened.
	windows []*fakeWindow
	// trays are the icons that the application put in the tray, and trayError is why
	// it cannot: a desktop with no tray.
	trays     []*fakeTray
	trayError error
	// pagedOut counts the times that the application told the system that it does not want the
	// pages of its files for now.
	pagedOut atomic.Int32
}

// fakeTray is an icon in the tray that shows nothing, and notes what the application asks of
// it. Its menu is changed and it is destroyed from goroutines other than the test's.
type fakeTray struct {
	mu        sync.Mutex
	options   trayOptions
	toolTip   string
	entries   []trayEntry
	updates   int
	destroyed bool
}

func (t *fakeTray) Update(toolTip string, entries []trayEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.toolTip, t.entries = toolTip, entries
	t.updates++
}

func (t *fakeTray) Destroy() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.destroyed = true
}

// state is what the icon shows now, and whether it is gone.
func (t *fakeTray) state() (toolTip string, entries []trayEntry, destroyed bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.toolTip, t.entries, t.destroyed
}

// choose chooses the item of the menu with the label, as the user would, and says whether
// there is one.
func (t *fakeTray) choose(label string) bool {
	_, entries, _ := t.state()
	for _, entry := range entries {
		if entry.label == label && entry.click != nil {
			entry.click()
			return true
		}
	}
	return false
}

// fakeWindow is a window that shows nothing, and notes what the application asks of it. The application
// asks it from any goroutine, as it asks the window of the toolkit, which allows that: a change of the
// settings asks for a frame from the goroutine that made it and from the one of the web interface that
// follows it.
type fakeWindow struct {
	mu                        sync.Mutex
	title                     string
	invalidated, shown, focus atomic.Int32
}

func (w *fakeWindow) Invalidate() { w.invalidated.Add(1) }
func (w *fakeWindow) SetTitle(title string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.title = title
}
func (w *fakeWindow) Show()  { w.shown.Add(1) }
func (w *fakeWindow) Focus() { w.focus.Add(1) }

// windowTitle is the title that the application gave the window.
func (w *fakeWindow) windowTitle() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.title
}

// testApp is the application against the captured output of vfox, drawn without
// a window.
type testApp struct {
	t       *testing.T
	app     *App
	runner  *testsupport.FakeProcessRunner
	tester  *mui.Tester
	desktop *recorder
	web     *pair.Host
	// settingsDir holds the certificate that the web interface makes for other devices.
	settingsDir string
}

func newTestApp(t *testing.T, runner *testsupport.FakeProcessRunner, width, height int, store settings.SettingsStore) *testApp {
	t.Helper()
	environment := map[string]string{"HOME": t.TempDir(), "LANG": "en_US.UTF-8"}
	model := state.NewAppModel(state.AppModelOptions{
		Runner: runner, SettingsStore: store, Environment: environment,
		Locator: vfox.NewVfoxLocator(map[string]string{}, func(path string) bool { return path == "/usr/bin/vfox" }),
	})
	settingsDir := t.TempDir()
	web := pair.NewHost(model, pair.HostOptions{SettingsDir: settingsDir})
	a, err := New(Options{Model: model, Environment: environment, Web: web})
	if err != nil {
		t.Fatal(err)
	}
	a.env.Reg.CollectText = true
	recording := &recorder{}
	a.desktop = desktop{
		openURL: func(address string) { recording.opened = append(recording.opened, address) },
		copy:    func(text string) { recording.copied = append(recording.copied, text) },
		choose: func(options chooseOptions) string {
			recording.asked = append(recording.asked, options)
			return recording.chosen
		},
		newWindow: func(mygo.WindowOptions) surface {
			window := &fakeWindow{}
			recording.windows = append(recording.windows, window)
			return window
		},
		quit:    func() { recording.quits++ },
		pageOut: func() { recording.pagedOut.Add(1) },
		newTray: func(options trayOptions) (trayIcon, error) {
			if recording.trayError != nil {
				return nil, recording.trayError
			}
			tray := &fakeTray{options: options, toolTip: options.toolTip, entries: options.entries}
			recording.trays = append(recording.trays, tray)
			return tray, nil
		},
	}
	a.listen()
	t.Cleanup(model.Dispose)
	web.Start()
	t.Cleanup(web.Close)
	tester := mui.NewTester(a.view, width, height)
	// Springs that jump to their targets keep the tests short: a dialog that
	// scales in is drawn on the slow path of the renderer for a second.
	tester.SetPreferences(mui.Preferences{ReduceMotion: true, TextScale: 1})
	return &testApp{t: t, app: a, runner: runner, desktop: recording, tester: tester, web: web, settingsDir: settingsDir}
}

func defaultApp(t *testing.T) *testApp {
	t.Helper()
	app := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	app.start()
	return app
}

// start probes vfox and waits for the installed SDKs.
func (ta *testApp) start() {
	ta.t.Helper()
	ta.app.model.Start()
	ta.until("the installed SDKs", func() bool { return ta.app.model.Installed.HasValue() })
	ta.settle()
}

// frame draws what changed outside the view.
func (ta *testApp) frame() {
	ta.t.Helper()
	ta.tester.Frame()
}

// settle draws frames until the springs and the transitions have ended.
func (ta *testApp) settle() {
	ta.t.Helper()
	ta.tester.Frame()
	ta.tester.Advance(time.Second)
}

// until draws frames until the condition holds.
func (ta *testApp) until(description string, condition func() bool) {
	ta.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			ta.t.Fatalf("timed out waiting for %s (on screen: %q)", description, ta.texts())
		}
		time.Sleep(5 * time.Millisecond)
		ta.frame()
	}
}

func (ta *testApp) nodes() []*kit.Node { return ta.app.env.Reg.Nodes() }

func (ta *testApp) text(text string) bool {
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleText && n.Text == text {
			return true
		}
	}
	return false
}

func (ta *testApp) mustText(text string) {
	ta.t.Helper()
	if !ta.text(text) {
		ta.t.Fatalf("no text %q on screen; there is %q", text, ta.texts())
	}
}

func (ta *testApp) texts() []string {
	var texts []string
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleText && n.Text != "" {
			texts = append(texts, n.Text)
		}
	}
	return texts
}

// find returns the node a user can reach: when a dialog or a menu is open, only
// its nodes.
func (ta *testApp) find(role kit.Role, name string) (*kit.Node, bool) {
	modal := false
	for _, n := range ta.nodes() {
		modal = modal || n.Modal
	}
	for _, n := range ta.nodes() {
		if n.Role == role && n.Modal == modal && (n.Name == name || role == kit.RoleListItem && strings.HasPrefix(n.Name, name)) {
			return n, true
		}
	}
	return nil, false
}

// click does what a click on the widget does, and lets the interface react.
func (ta *testApp) click(role kit.Role, name string) {
	ta.t.Helper()
	n, ok := ta.find(role, name)
	if !ok || n.Click == nil {
		ta.t.Fatalf("nothing to click: %s %q (%q)", role, name, ta.texts())
	}
	n.Click()
	ta.settle()
}

func (ta *testApp) button(name string) { ta.t.Helper(); ta.click(kit.RoleButton, name) }

// snapshot saves what the window shows to the directory that VFOX_PLUS_SNAPSHOTS
// names, if it is set: the way to look at the pages.
func (ta *testApp) snapshot(name string) {
	ta.t.Helper()
	saveSnapshot(ta.t, name, ta.tester.Image())
}

func saveSnapshot(t *testing.T, name string, img image.Image) {
	t.Helper()
	dir := os.Getenv("VFOX_PLUS_SNAPSHOTS")
	if dir == "" {
		return
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // The directory is the choice of whoever looks at the pages.
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".png"), encoded.Bytes(), 0o600); err != nil { //nolint:gosec // Likewise.
		t.Fatal(err)
	}
}
