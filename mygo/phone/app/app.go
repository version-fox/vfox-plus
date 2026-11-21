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

// Package app is the state of the phone app: whether the phone is paired with a
// computer, the tab that is shown, and the pages above the tab. The connection to
// the computer and the data read from it are in Service, which is safe for use
// from any goroutine. The rest of this package has no dependency on the user
// interface, so that its logic is tested without a window. The window shows this
// state, and the history of the window follows the pages through History, which
// is how the system's back gesture reaches them.
package app

import "slices"

// Session says whether the phone is paired with a computer, and what the phone
// can do about it. The Service decides it, and the window copies it to the App on
// the main thread (App.SetSession).
type Session int

// The sessions of the phone.
const (
	// Unpaired is a phone with no saved computer: it shows the pairing page only.
	Unpaired Session = iota
	// Connecting is a phone whose saved computer is being checked. It shows the
	// tabs, and their loads wait for the answer.
	Connecting
	// Paired is a phone that the saved computer accepts: it shows the tabs.
	Paired
	// Unreachable is a paired phone whose computer did not answer the last try.
	// It shows the tabs, and the loads say that they failed.
	Unreachable
	// NeedsPairing is a phone whose saved computer no longer accepts its session
	// (HTTP 401). The computer stays saved, for the next pairing, and the pairing
	// page says that the phone must pair again.
	NeedsPairing
	// CertificateChanged is a phone whose saved computer presents a certificate
	// other than the one that was paired. Nothing is trusted automatically: the
	// pairing page says so, and the user decides.
	CertificateChanged
)

// String names the session, for the messages of the tests and the logs.
func (s Session) String() string {
	switch s {
	case Unpaired:
		return "unpaired"
	case Connecting:
		return "connecting"
	case Paired:
		return "paired"
	case Unreachable:
		return "unreachable"
	case NeedsPairing:
		return "needs pairing"
	case CertificateChanged:
		return "certificate changed"
	}
	return "unknown session"
}

// hasTabs reports whether a session shows the tabs. A phone with a saved computer
// shows them even while the computer cannot be reached, and the pages say so.
func (s Session) hasTabs() bool {
	return s == Connecting || s == Paired || s == Unreachable
}

// Screen is what the window shows for the state of the app.
type Screen int

// The screens of the window.
const (
	// ScreenPairing is the pairing page, which a phone without a usable computer shows.
	ScreenPairing Screen = iota
	// ScreenTabs are the tabs, with the pages above them.
	ScreenTabs
)

// MaxDepth is how many pages can stand above the tabs. The history of a window
// keeps a limited number of entries, and the pages stay well within it.
const MaxDepth = 32

// History is where the pages are shown: the history of the window's router. Its
// entries are the root and the pages of the stack, in order, and the stack
// mirrors them.
type History interface {
	// Path returns the path of the page shown.
	Path() string
	// Push shows the page at path after the one shown, and forgets the pages
	// that followed it.
	Push(path string)
	// Go moves delta entries through the history; a negative delta goes back.
	Go(delta int)
}

// App is the state of the phone app. Its methods are called on the main thread.
type App struct {
	history History
	session Session
	tab     Tab
	pages   []Page
}

// New returns an unpaired app on the home tab, with no pages above it. Its
// pages are shown in history, which must not be nil.
func New(history History) *App {
	return &App{history: history, session: Unpaired, tab: TabHome}
}

// Session returns the session of the phone.
func (a *App) Session() Session { return a.session }

// SetSession changes the session of the phone. A session that shows the pairing
// page closes the pages and selects the home tab, where a phone that is paired
// again starts.
func (a *App) SetSession(session Session) {
	if session == a.session {
		return
	}
	a.session = session
	if a.Screen() == ScreenPairing {
		a.closePages()
		a.tab = TabHome
	}
}

// Screen returns what the window shows: the pairing page while the phone has no
// usable computer, and the tabs otherwise.
func (a *App) Screen() Screen {
	if a.session.hasTabs() {
		return ScreenTabs
	}
	return ScreenPairing
}

// Tab returns the tab under the pages.
func (a *App) Tab() Tab { return a.tab }

// SelectTab shows a tab, and closes the pages above the tabs. A value that is
// not a tab changes nothing.
func (a *App) SelectTab(tab Tab) {
	if !tab.valid() {
		return
	}
	a.closePages()
	a.tab = tab
}

// Pages returns the pages above the tabs, the first opened first.
func (a *App) Pages() []Page { return slices.Clone(a.pages) }

// Top returns the page shown above the tabs, and false when the tabs are shown.
func (a *App) Top() (Page, bool) {
	if len(a.pages) == 0 {
		return Page{}, false
	}
	return a.pages[len(a.pages)-1], true
}

// Push shows a page above the one shown. It reports false, and changes nothing,
// when the page is not valid, when it is the page shown already, and when the
// pages are at MaxDepth.
func (a *App) Push(page Page) bool {
	a.Reconcile()
	if !page.valid() || len(a.pages) >= MaxDepth {
		return false
	}
	path := page.Path()
	if a.history.Path() == path {
		return false
	}
	a.history.Push(path)
	a.pages = append(a.pages, page)
	return true
}

// Pop goes back from the page shown, which closes it, and reports false, and
// changes nothing, when the tabs are shown.
func (a *App) Pop() bool {
	a.Reconcile()
	if len(a.pages) == 0 {
		return false
	}
	a.history.Go(-1)
	a.pages = a.pages[:len(a.pages)-1]
	return true
}

// Back is the system's back gesture, in two steps: it closes the page shown, and
// on a tab other than the home tab it goes to the home tab. On the home tab it
// changes nothing and reports false, and the system then leaves the app. The
// window takes the gesture itself only where TakesBack says so.
func (a *App) Back() bool {
	a.Reconcile()
	switch {
	case len(a.pages) > 0:
		return a.Pop()
	case a.Screen() == ScreenTabs && a.tab != TabHome:
		a.SelectTab(TabHome)
		return true
	}
	return false
}

// TakesBack reports whether the app takes the back gesture now, which is the case
// when the tabs show a tab other than the home tab, with no page above them. A
// page is taken by the router, which goes back by itself; the home tab is left to
// the system, which leaves the app.
func (a *App) TakesBack() bool {
	a.Reconcile()
	return len(a.pages) == 0 && a.Screen() == ScreenTabs && a.tab != TabHome
}

// Reconcile closes the pages that the history no longer shows. The history goes
// back by itself when the system's back gesture goes back a page, and the stack
// follows it here, before the window is drawn.
func (a *App) Reconcile() {
	path := a.history.Path()
	for len(a.pages) > 0 && a.pages[len(a.pages)-1].Path() != path {
		a.pages = a.pages[:len(a.pages)-1]
	}
}

// closePages goes back to the tabs, and forgets the pages above them.
func (a *App) closePages() {
	a.Reconcile()
	if n := len(a.pages); n > 0 {
		a.history.Go(-n)
		a.pages = nil
	}
}
