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

package app

import (
	"fmt"
	"reflect"
	"testing"
)

// fakeHistory is a History that behaves as the router of a window does: a push
// drops the entries after the one shown and is refused for the page shown, and a
// move stops at the first and the last entry.
type fakeHistory struct {
	entries []string
	at      int
}

func newFakeHistory() *fakeHistory {
	return &fakeHistory{entries: []string{RootPath}}
}

func (h *fakeHistory) Path() string { return h.entries[h.at] }

func (h *fakeHistory) Push(path string) {
	if h.entries[h.at] == path {
		return
	}
	h.entries = append(h.entries[:h.at+1], path)
	h.at = len(h.entries) - 1
}

func (h *fakeHistory) Go(delta int) {
	h.at = max(0, min(h.at+delta, len(h.entries)-1))
}

// newPairedApp returns a paired app over a fresh history.
func newPairedApp(t *testing.T) (*App, *fakeHistory) {
	t.Helper()
	history := newFakeHistory()
	a := New(history)
	a.SetSession(Paired)
	return a, history
}

func TestANewAppIsUnpairedOnTheHomeTab(t *testing.T) {
	a := New(newFakeHistory())
	if a.Session() != Unpaired || a.Screen() != ScreenPairing {
		t.Errorf("session %v, screen %v; want the pairing page of an unpaired phone", a.Session(), a.Screen())
	}
	if a.Tab() != TabHome {
		t.Errorf("tab %v, want the home tab", a.Tab())
	}
	if pages := a.Pages(); len(pages) != 0 {
		t.Errorf("pages %v, want none", pages)
	}
	if _, ok := a.Top(); ok {
		t.Error("Top reports a page above the tabs")
	}
}

func TestAPairedAppShowsTheTabs(t *testing.T) {
	a, _ := newPairedApp(t)
	if a.Screen() != ScreenTabs {
		t.Errorf("screen %v, want the tabs", a.Screen())
	}
}

func TestPushAndPopFollowTheHistory(t *testing.T) {
	a, history := newPairedApp(t)
	sdk := Page{Kind: PageSDK, Name: "go"}
	task := Page{Kind: PageTask, Name: "7"}
	if !a.Push(sdk) || !a.Push(task) {
		t.Fatal("Push refused a page")
	}
	if got := history.Path(); got != "/tasks/7" {
		t.Errorf("history at %q, want /tasks/7", got)
	}
	if top, ok := a.Top(); !ok || top != task {
		t.Errorf("Top = %v, %v; want %v", top, ok, task)
	}
	if !a.Pop() || history.Path() != "/sdk/go" {
		t.Errorf("Pop did not go back to /sdk/go, history at %q", history.Path())
	}
	if !a.Pop() || history.Path() != RootPath {
		t.Errorf("Pop did not go back to the tabs, history at %q", history.Path())
	}
	if a.Pop() {
		t.Error("Pop reported a page above the tabs, which are shown")
	}
}

func TestPushRefusesWhatIsNotAPageOrIsShownAlready(t *testing.T) {
	a, history := newPairedApp(t)
	for _, page := range []Page{
		{},
		{Kind: PageSDK},
		{Kind: PagePair, Name: "x"},
		{Kind: PageKind(99), Name: "go"},
	} {
		if a.Push(page) {
			t.Errorf("Push(%+v) accepted a page that is not valid", page)
		}
	}
	if !a.Push(Page{Kind: PageSDK, Name: "go"}) {
		t.Fatal("Push refused a valid page")
	}
	if a.Push(Page{Kind: PageSDK, Name: "go"}) {
		t.Error("Push accepted the page shown already")
	}
	if want := []string{RootPath, "/sdk/go"}; !reflect.DeepEqual(history.entries, want) {
		t.Errorf("history entries %v, want %v", history.entries, want)
	}
}

func TestPushStopsAtMaxDepth(t *testing.T) {
	a, _ := newPairedApp(t)
	for i := range MaxDepth {
		if !a.Push(Page{Kind: PageSDK, Name: fmt.Sprintf("sdk%d", i)}) {
			t.Fatalf("Push refused page %d of %d", i+1, MaxDepth)
		}
	}
	if a.Push(Page{Kind: PagePair}) {
		t.Errorf("Push went beyond MaxDepth, %d pages", MaxDepth)
	}
	if got := len(a.Pages()); got != MaxDepth {
		t.Errorf("%d pages, want %d", got, MaxDepth)
	}
}

func TestSelectTabClosesThePages(t *testing.T) {
	a, history := newPairedApp(t)
	a.Push(Page{Kind: PageSDK, Name: "go"})
	a.Push(Page{Kind: PageTask, Name: "7"})
	a.SelectTab(TabTasks)
	if a.Tab() != TabTasks || len(a.Pages()) != 0 || history.Path() != RootPath {
		t.Errorf("tab %v, pages %v, history at %q; want the tasks tab with no pages", a.Tab(), a.Pages(), history.Path())
	}
	a.SelectTab(Tab(99))
	if a.Tab() != TabTasks {
		t.Errorf("a value that is not a tab changed the tab to %v", a.Tab())
	}
}

func TestPopFollowsABackThatTheHistoryMadeItself(t *testing.T) {
	a, history := newPairedApp(t)
	a.Push(Page{Kind: PageSDK, Name: "go"})
	a.Push(Page{Kind: PageTask, Name: "7"})

	// The system's back gesture goes back a page of the router.
	history.Go(-1)
	a.Reconcile()
	if pages := a.Pages(); !reflect.DeepEqual(pages, []Page{{Kind: PageSDK, Name: "go"}}) {
		t.Fatalf("pages after the system's back %v, want the SDK page only", pages)
	}

	history.Go(-1)
	a.Reconcile()
	if len(a.Pages()) != 0 {
		t.Errorf("pages after the second back %v, want none", a.Pages())
	}
}

func TestAPushAfterABackReplacesThePagesAfterIt(t *testing.T) {
	a, history := newPairedApp(t)
	a.Push(Page{Kind: PageSDK, Name: "go"})
	a.Push(Page{Kind: PageSDK, Name: "node"})
	history.Go(-1) // the system goes back to /sdk/go
	if !a.Push(Page{Kind: PagePair}) {
		t.Fatal("Push refused a page after a back")
	}
	want := []Page{{Kind: PageSDK, Name: "go"}, {Kind: PagePair}}
	if !reflect.DeepEqual(a.Pages(), want) {
		t.Errorf("pages %v, want %v", a.Pages(), want)
	}
	if history.Path() != "/pair" {
		t.Errorf("history at %q, want /pair", history.Path())
	}
}

func TestUnpairingClosesThePagesAndSelectsTheHomeTab(t *testing.T) {
	a, history := newPairedApp(t)
	a.SelectTab(TabSDKs)
	a.Push(Page{Kind: PageSDK, Name: "go"})
	a.SetSession(Unpaired)
	if a.Screen() != ScreenPairing || a.Tab() != TabHome || len(a.Pages()) != 0 || history.Path() != RootPath {
		t.Errorf("after unpairing: screen %v, tab %v, pages %v, history %q", a.Screen(), a.Tab(), a.Pages(), history.Path())
	}
	a.SetSession(Paired)
	if a.Screen() != ScreenTabs || a.Tab() != TabHome {
		t.Errorf("after pairing again: screen %v, tab %v", a.Screen(), a.Tab())
	}
}

func TestTheTabsAreFiveInTheOrderOfTheBar(t *testing.T) {
	want := []string{"home", "sdks", "plugins", "tasks", "settings"}
	var got []string
	for _, tab := range Tabs() {
		got = append(got, tab.String())
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tabs %v, want %v", got, want)
	}
	if Tab(99).String() != "" {
		t.Error("a value that is not a tab has a name")
	}
}
