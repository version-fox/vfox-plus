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

import "testing"

func TestTheBackGestureClosesThePageFirst(t *testing.T) {
	a, history := newPairedApp(t)
	a.SelectTab(TabTasks)
	a.Push(Page{Kind: PageTask, Name: "7"})
	if a.TakesBack() {
		t.Fatal("the app takes the back gesture above a page; the router does")
	}
	if !a.Back() {
		t.Fatal("Back did not close the page")
	}
	if len(a.Pages()) != 0 || a.Tab() != TabTasks || history.Path() != RootPath {
		t.Errorf("after closing the page: pages %v, tab %v, history %q", a.Pages(), a.Tab(), history.Path())
	}
}

func TestTheBackGestureGoesFromAnotherTabToTheHomeTab(t *testing.T) {
	a, _ := newPairedApp(t)
	a.SelectTab(TabSettings)
	if !a.TakesBack() {
		t.Fatal("the app does not take the back gesture on the settings tab")
	}
	if !a.Back() || a.Tab() != TabHome {
		t.Errorf("Back on the settings tab: tab %v; want the home tab", a.Tab())
	}
	if a.TakesBack() {
		t.Error("the app takes the back gesture on the home tab, which leaves the app")
	}
}

func TestTheBackGestureLeavesTheAppFromTheHomeTab(t *testing.T) {
	a, history := newPairedApp(t)
	if a.Back() {
		t.Error("Back on the home tab reported that it went back; the system must leave the app")
	}
	if a.Tab() != TabHome || history.Path() != RootPath {
		t.Errorf("Back on the home tab changed tab %v or history %q", a.Tab(), history.Path())
	}
}

func TestTheBackGestureTakesNothingWhilePairing(t *testing.T) {
	a := New(newFakeHistory())
	a.SelectTab(TabSettings) // ignored while unpaired: the tabs are not shown
	if a.TakesBack() || a.Back() {
		t.Error("an unpaired phone takes the back gesture")
	}
}

func TestSessionsThatHaveAComputerShowTheTabs(t *testing.T) {
	for _, tt := range []struct {
		session Session
		screen  Screen
		name    string
	}{
		{Unpaired, ScreenPairing, "unpaired"},
		{Connecting, ScreenTabs, "connecting"},
		{Paired, ScreenTabs, "paired"},
		{Unreachable, ScreenTabs, "unreachable"},
		{NeedsPairing, ScreenPairing, "needs pairing"},
		{CertificateChanged, ScreenPairing, "certificate changed"},
	} {
		a := New(newFakeHistory())
		a.SetSession(tt.session)
		if a.Screen() != tt.screen || a.Session() != tt.session || tt.session.String() != tt.name {
			t.Errorf("%v: screen %v, name %q; want screen %v and name %q", tt.session, a.Screen(), tt.session.String(), tt.screen, tt.name)
		}
	}
	if Session(99).String() != "unknown session" {
		t.Error("a value that is not a session has a name")
	}
}

func TestAnUnreachableComputerKeepsThePagesOpen(t *testing.T) {
	a, _ := newPairedApp(t)
	a.Push(Page{Kind: PageSDK, Name: "go"})
	a.SetSession(Unreachable)
	if len(a.Pages()) != 1 || a.Screen() != ScreenTabs {
		t.Errorf("unreachable: pages %v, screen %v; want the page kept under the tabs", a.Pages(), a.Screen())
	}
}

func TestLosingTheComputerClosesThePagesAndShowsThePairingPage(t *testing.T) {
	a, history := newPairedApp(t)
	a.SelectTab(TabSDKs)
	a.Push(Page{Kind: PageSDK, Name: "go"})
	a.SetSession(NeedsPairing)
	if a.Screen() != ScreenPairing || a.Tab() != TabHome || len(a.Pages()) != 0 || history.Path() != RootPath {
		t.Errorf("after needing pairing: screen %v, tab %v, pages %v, history %q", a.Screen(), a.Tab(), a.Pages(), history.Path())
	}
	a.SetSession(Connecting)
	if a.Screen() != ScreenTabs {
		t.Errorf("a saved computer that is checked again shows %v, want the tabs", a.Screen())
	}
}
