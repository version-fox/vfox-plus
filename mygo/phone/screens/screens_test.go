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
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// newRoot returns the window in English, which the tests read. The language is set
// back when the test ends. The configuration directory is a directory of the test,
// so that no test reads or writes the files of a user.
func newRoot(t *testing.T) *Root {
	t.Helper()
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root, err := New()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// settle draws the frame once the transitions of the router are over: a page that
// is pushed slides in from the side, and its elements are where they belong only
// at the end of the slide.
func settle(tt *ui.Tester) {
	tt.Advance(2 * time.Second)
	tt.Frame()
}

// hasText reports whether the frame shows a text that is exactly s. Texts that
// merely contain s, such as the explanation of the pairing page, do not count.
func hasText(tt *ui.Tester, s string) bool {
	for _, text := range tt.Texts() {
		if text == s {
			return true
		}
	}
	return false
}

// backKey is the key that the router takes as the way back, which is the system's
// back gesture on a phone: Alt+Left on Linux and Windows, Cmd+[ on macOS.
func backKey() (ui.Modifiers, ui.Key) {
	if runtime.GOOS == "darwin" {
		return ui.Cmd, ui.KeyBracketLeft
	}
	return ui.Alt, ui.KeyLeft
}

// newTLSServer starts a TLS server for a fake computer, and stops it when the test
// ends. Its log of failed handshakes is discarded: the tests that refuse a certificate
// make the handshake fail on purpose, and the log would fill the output of every test.
func newTLSServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

// bodyTexts returns the texts of a frame other than the title of a page and its way
// back: what the page shows under its bar. A page that is loading shows "Loading…",
// which is part of its body.
func bodyTexts(tt *ui.Tester, title string) []string {
	var body []string
	for _, text := range tt.Texts() {
		if text != title && text != "Back" {
			body = append(body, text)
		}
	}
	return body
}

func TestAnUnpairedPhoneShowsOnlyThePairingPage(t *testing.T) {
	tt := ui.NewTester(newRoot(t).View, 480, 900)
	for _, text := range []string{"This phone", "Connect"} {
		if !hasText(tt, text) {
			t.Errorf("the pairing page has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	for _, tab := range []string{"Home", "SDKs", "Plugins", "Tasks", "Settings"} {
		if hasText(tt, tab) {
			t.Errorf("an unpaired phone shows the tab %q", tab)
		}
	}
}

func TestAPairedPhoneShowsTheFiveTabs(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	for _, tab := range []string{"Home", "SDKs", "Plugins", "Tasks", "Settings"} {
		if !hasText(tt, tab) {
			t.Errorf("the bar of the tabs has no %q; the frame shows %q", tab, tt.Texts())
		}
	}
	// The home tab shows its own sections, which the answers of the computer fill in.
	for _, section := range []string{"In use", "Health"} {
		if !hasText(tt, section) {
			t.Errorf("the home tab has no %q section; the frame shows %q", section, tt.Texts())
		}
	}
}

func TestTheBarOfTabsSelectsTheTab(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	if err := tt.Click("Tasks"); err != nil {
		t.Fatalf("Click(Tasks): %v", err)
	}
	tt.Frame()
	if root.App().Tab() != app.TabTasks {
		t.Errorf("tab %v after a click on Tasks, want %v", root.App().Tab(), app.TabTasks)
	}
}

func TestAPageAboveTheTabsHasItsWayBack(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	if !root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"}) {
		t.Fatal("Push refused the page of the SDK go")
	}
	settle(tt)
	if !hasText(tt, "go") {
		t.Fatalf("the page of the SDK go is not shown; the frame shows %q", tt.Texts())
	}
	if hasText(tt, "Settings") {
		t.Error("the bar of the tabs is shown above a page")
	}
	if err := tt.Click("Back"); err != nil {
		t.Fatalf("Click(Back): %v", err)
	}
	settle(tt)
	if pages := root.App().Pages(); len(pages) != 0 {
		t.Errorf("pages after Back: %v, want none", pages)
	}
	if !hasText(tt, "Settings") {
		t.Errorf("the tabs are not shown after Back; the frame shows %q", tt.Texts())
	}
}

func TestTheSystemsBackGoesBackOnePage(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	sdk := app.Page{Kind: app.PageSDK, Name: "go"}
	root.App().Push(sdk)
	root.App().Push(app.Page{Kind: app.PageTask, Name: "7"})
	settle(tt)

	mods, key := backKey()
	tt.Key(mods, key)
	settle(tt)
	if pages := root.App().Pages(); len(pages) != 1 || pages[0] != sdk {
		t.Fatalf("pages after the system's back: %v, want the SDK page only", pages)
	}
	if !hasText(tt, "go") {
		t.Errorf("the SDK page is not shown after the back; the frame shows %q", tt.Texts())
	}

	tt.Key(mods, key)
	settle(tt)
	if pages := root.App().Pages(); len(pages) != 0 {
		t.Errorf("pages after the second back: %v, want none", pages)
	}
	if !hasText(tt, "Settings") {
		t.Errorf("the tabs are not shown after the second back; the frame shows %q", tt.Texts())
	}
}

func TestTheSystemsBackGoesFromAnotherTabToTheHomeTab(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	root.App().SelectTab(app.TabTasks)
	tt.Frame() // the frame takes the back key for the tabs

	tt.Key(0, ui.KeyBack)
	settle(tt)
	if root.App().Tab() != app.TabHome {
		t.Errorf("tab %v after the system's back on the tasks tab, want the home tab", root.App().Tab())
	}
}

func TestTheSystemsBackLeavesTheHomeTabAlone(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	tt.Frame()
	tt.Key(0, ui.KeyBack)
	settle(tt)
	if root.App().Tab() != app.TabHome || root.App().Screen() != app.ScreenTabs {
		t.Errorf("the home tab changed on the back key: tab %v, screen %v", root.App().Tab(), root.App().Screen())
	}
}

func TestUnpairingReturnsToThePairingPage(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	root.App().Push(app.Page{Kind: app.PagePair})
	root.App().SetSession(app.Unpaired)
	tt.Frame()
	if !hasText(tt, "This phone") || hasText(tt, "Settings") {
		t.Errorf("an unpaired phone shows %q; want the pairing page only", tt.Texts())
	}

	root.App().SetSession(app.Paired)
	tt.Frame()
	if !hasText(tt, "Home") || !hasText(tt, "Settings") {
		t.Errorf("the tabs after pairing again show %q", tt.Texts())
	}
}

func TestEveryKindOfPageHasItsBody(t *testing.T) {
	pages := []app.Page{
		{Kind: app.PageSDK, Name: "go"},
		{Kind: app.PageTask, Name: "7"},
		{Kind: app.PageLog, Name: "7"},
		{Kind: app.PagePair},
	}
	for _, page := range pages {
		root := newRoot(t)
		root.App().SetSession(app.Paired)
		tt := ui.NewTester(root.View, 480, 900)
		if !root.App().Push(page) {
			t.Fatalf("Push refused %+v", page)
		}
		settle(tt)
		title := pageTitle(page)
		if !hasText(tt, title) {
			t.Errorf("the page %+v has no title %q; the frame shows %q", page, title, tt.Texts())
		}
		if len(bodyTexts(tt, title)) == 0 {
			t.Errorf("the page %+v has no text under its bar; the frame shows %q", page, tt.Texts())
		}
	}
}

func TestSessionsThatNeedPairingShowThePairingPage(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	for _, session := range []app.Session{app.NeedsPairing, app.CertificateChanged} {
		root.App().SetSession(session)
		tt.Frame()
		if !hasText(tt, "This phone") || hasText(tt, "Settings") {
			t.Errorf("%v shows %q; want the pairing page", session, tt.Texts())
		}
	}
	root.App().SetSession(app.Unreachable)
	tt.Frame()
	if !hasText(tt, "Settings") {
		t.Errorf("an unreachable computer shows %q; want the tabs, whose pages say so", tt.Texts())
	}
}

func TestNoticesOfTheServiceAreShownAsSnackBars(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	root.Service().Do(func(ctx context.Context) error { return nil })
	root.Service().Do(func(ctx context.Context) error { return remote.ErrNotFound })
	want := i18n.TextIn(i18n.English, "errNotFound")
	deadline := time.Now().Add(5 * time.Second)
	for !hasText(tt, want) {
		if time.Now().After(deadline) {
			t.Fatalf("the notice %q is not shown; the frame shows %q", want, tt.Texts())
		}
		time.Sleep(5 * time.Millisecond)
		tt.Frame()
	}
}

func TestASucceededCallShowsItsMessageOnce(t *testing.T) {
	root := newRoot(t)
	root.App().SetSession(app.Paired)
	tt := ui.NewTester(root.View, 480, 900)
	// "connect" is a text that only the pairing page shows, so the frame can only
	// show it once the message is queued and drawn.
	root.Do(func(ctx context.Context) error { return nil }, "connect")
	want := i18n.TextIn(i18n.English, "connect")
	deadline := time.Now().Add(5 * time.Second)
	for {
		tt.Frame()
		if hasText(tt, want) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the message of the call is not shown; the frame shows %q", tt.Texts())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if left := root.takeDone(); len(left) != 0 {
		t.Errorf("the message was kept after it was shown: %v", left)
	}
}

func TestTheServiceAsksTheWindowForAFrame(t *testing.T) {
	root := newRoot(t)
	frames := make(chan struct{}, 4)
	root.Attach(func() {
		select {
		case frames <- struct{}{}:
		default:
		}
	})
	root.Service().Do(func(ctx context.Context) error { return errors.New("a change") })
	select {
	case <-frames:
	case <-time.After(5 * time.Second):
		t.Fatal("the service did not ask the window for a frame after a change")
	}
}
