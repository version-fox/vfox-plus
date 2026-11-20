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

// Package screens draws the phone app with the widgets of mygo's kit: the
// pairing page, the tabs and the pages above them. It shows the state of package
// app and of its Service, and it is the only package of the phone that depends on
// the user interface of mygo.
//
// Each kind of screen is in its own file: the tabs in the files named after them,
// the pages above the tabs in page_<kind>.go, and the connection in pairing.go.
// Every screen is a method of Root, which holds the Service the screen reads.
package screens

import (
	"context"
	"io/fs"
	"sync"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/assets"
	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// compactWidth is the width in dp below which the bar of the tabs is at the
// bottom of the window, as it is on a phone.
const compactWidth = 600

// Root is the window of the phone app. It builds the screens for the state of the
// app and of its service, and keeps the state of the widgets that the model does
// not hold, such as the text of the pairing link.
type Root struct {
	app      *app.App
	service  *app.Service
	router   *ui.Router
	env      *kit.Env
	uiTheme  *ui.Theme
	look     look
	linkText string

	// seen is the session of the service that the App last copied; see sync.
	seen app.Session

	// mu guards redraw and done, which the goroutines of the calls reach (see
	// Attach and Do).
	mu     sync.Mutex
	redraw func()
	// done holds the messages of the calls that succeeded, which sync shows once.
	done []string

	// tasks is the watch of the tasks that end, which sync announces on every screen.
	// Only the frames use it, on the main thread.
	tasks taskWatch

	// pickFile has the user choose a plugin file, and returns the path of a copy of it, or ""
	// when the user chose nothing. It blocks until the user answers, so it runs in the
	// background. It is a field so that the tests answer for the user; see pickPluginFile.
	pickFile func(ctx context.Context) (string, error)
}

// New loads the icons of the interface, and assembles the app: a router, which
// shows the pages, the model, whose pages the history of the router follows, and
// the service of the computer, which reads the saved computer from the app's
// configuration directory. It does not connect: Service.Run does that. It returns
// an error when the icons cannot be loaded, or the configuration directory cannot
// be found.
func New() (*Root, error) {
	table, err := assets.Files.ReadFile("icons.json")
	if err != nil {
		return nil, err
	}
	icons, err := fs.Sub(assets.Files, "icons")
	if err != nil {
		return nil, err
	}
	if err := kit.LoadIcons(table, icons); err != nil {
		return nil, err
	}
	service, err := newService()
	if err != nil {
		return nil, err
	}
	router := ui.NewRouter(app.RootPath)
	// On a phone the system shows the pages that open above the tabs as its own
	// stack of screens, with its bar, its back button and its back gesture;
	// elsewhere SystemStack changes nothing. The tabs themselves draw their heading.
	router.SystemStack = true
	theme := kit.NewTheme(seedColor, false, false, 1)
	root := &Root{router: router, env: kit.NewEnv(theme), uiTheme: theme.UI(), service: service, pickFile: pickPluginFile}
	root.app = app.New(routerHistory{router: router})
	root.seen = service.Status().Session
	root.app.SetSession(root.seen)
	service.SetNotify(root.invalidate)
	return root, nil
}

// App returns the state of the app, which the window shows.
func (r *Root) App() *app.App {
	return r.app
}

// Service returns the connection to the computer, and the data read from it. The
// window reads it on the main thread, every frame; the calls that change the
// computer must not be made there: they run with Service.Do.
func (r *Root) Service() *app.Service {
	return r.service
}

// Attach connects the window. Redraw asks the window for a frame; it must be safe
// from any goroutine, as the window's Invalidate is. The service calls it when its
// state changes, from the goroutines that make the change.
func (r *Root) Attach(redraw func()) {
	r.mu.Lock()
	r.redraw = redraw
	r.mu.Unlock()
}

// invalidate asks the attached window for a frame, when there is one.
func (r *Root) invalidate() {
	r.mu.Lock()
	redraw := r.redraw
	r.mu.Unlock()
	if redraw != nil {
		redraw()
	}
}

// Do runs fn in the background, where it may call the service's methods that
// wait for the computer: fn must not touch the window. When fn succeeds, the message
// of done, a key of the i18n catalogs, is shown as a snack bar, unless done is "";
// when it fails, the error is shown as a notice of the service. Call Do from the
// main thread, in a click or a key handler, or in the build; it returns at once.
func (r *Root) Do(fn func(ctx context.Context) error, done string) {
	r.service.Do(func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		if done != "" {
			r.mu.Lock()
			r.done = append(r.done, done)
			r.mu.Unlock()
			r.invalidate()
		}
		return nil
	})
}

// takeDone returns the messages of the calls that succeeded, oldest first, and
// forgets them.
func (r *Root) takeDone() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	done := r.done
	r.done = nil
	return done
}

// View builds the window for a frame.
func (r *Root) View(c *ui.Context) {
	r.applyTheme(c)
	k := r.env.Begin(c)
	width, _ := c.Size()
	r.env.Compact = width < compactWidth
	r.sync(k)
	if r.app.Screen() == app.ScreenPairing {
		r.pairingScreen(k)
	} else {
		// The system's back gesture may have gone back a page since the last
		// frame; the stack follows the history before it is drawn.
		r.app.Reconcile()
		r.router.View(c, func(route *ui.Route) {
			r.route(k, route)
		})
	}
	r.env.LayoutDialogs(k)
	r.env.LayoutSnackbars(k)
}

// sync brings the window up to the service, before a frame is drawn. The session
// of the App is copied from the service when the service's session changes, so a
// session that the App was given otherwise is kept until then. The notices of the
// calls that ran in the background are shown as snack bars, once each, and so are the
// messages of the tasks that end. The system's
// back key is taken where the App takes the back gesture: on a tab other than the
// home tab, with no page above the tabs (see app.App.TakesBack). Pages take it
// through the router, and on the home tab it is left to the system.
func (r *Root) sync(k kit.Ctx) {
	if session := r.service.Status().Session; session != r.seen {
		r.seen = session
		r.app.SetSession(session)
	}
	for _, notice := range r.service.TakeNotices() {
		if text := noticeText(notice.Kind); text != "" {
			k.Env.ShowSnackbar(kit.Snack{Text: text})
		}
	}
	for _, key := range r.takeDone() {
		k.Env.ShowSnackbar(kit.Snack{Text: i18n.Text(key)})
	}
	// The tasks that ended are announced on every screen, as the web announces them on
	// every page, so the watch is read here and not on the tasks tab only.
	r.announceTasks(k, &r.tasks)
	if r.app.TakesBack() && k.Shortcut(0, ui.KeyBack) {
		r.app.Back()
	}
}

// noticeText is the message of a notice of the service, or "" for a notice that
// shows nothing: a call that the app's suspension cancelled.
func noticeText(kind app.Kind) string {
	switch kind {
	case app.KindNotLink:
		return i18n.Text("errNotLink")
	case app.KindNotPluginFile:
		return i18n.Text("errNotPluginFile")
	case app.KindFileTooLarge:
		return i18n.Text("errFileTooLarge")
	case app.KindFileUnreadable:
		return i18n.Text("errFileUnreadable")
	case app.KindPairingRefused:
		return i18n.Text("errPairingRefused")
	case app.KindUnpaired:
		return i18n.Text("errUnpaired")
	case app.KindCertificateChanged:
		return i18n.Text("errCertificateChanged")
	case app.KindUnreachable:
		return i18n.Text("errUnreachable")
	case app.KindUnavailable:
		return i18n.Text("errUnavailable")
	case app.KindNotFound:
		return i18n.Text("errNotFound")
	case app.KindRejected:
		return i18n.Text("errRejected")
	case app.KindServer:
		return i18n.Text("errServer")
	case app.KindOther:
		return i18n.Text("errOther")
	}
	return ""
}

// route builds the page of one route of the router: the tabs at the root, and
// the page of its path above them.
func (r *Root) route(k kit.Ctx, route *ui.Route) {
	page, ok := app.ParsePage(route.Path())
	if !ok {
		// The tabs draw their own heading, in every window, and give the bar of the system no title. On
		// Android the bar is a toolbar, and with no way back beside it (the root has none) it is as high as
		// its title is, with the room that the font gives the line but not what its letters need: the tail of
		// the g of Plugins and of Settings was cut. It also kept the colour that its title had in the light
		// theme when the system went dark while the app ran. The bar is then no more than the place of the
		// status bar.
		route.Title("")
		r.tabShell(k)
		return
	}
	// Where the system shows the bar of the window, with the title and the way
	// back of each page above the tabs, the window draws none of its own.
	bar := !r.router.InSystemStack(k.Context)
	route.Title(pageTitle(page))
	r.pageScreen(k, page, bar)
}

// tabShell is the root of the router: the page of the tab shown, above the bar of
// the tabs. The page has its name, which is the tab's, for a screen reader: the title
// of the route is empty (see route).
func (r *Root) tabShell(k kit.Ctx) {
	tab := r.app.Tab()
	ui.Column(k.Context).Fill().Label(tabTitle(tab)).Children(func() {
		// The connection's state (checking, or not answering) is shown above the tabs.
		r.sessionStrip(k)
		kit.SwitchView(k, kit.FadeThrough, 0, tab, func() {
			r.tabPage(k, tab)
		}).Grow(1).Shrink(1)
		kit.NavBar(k, kit.NavBarSpec{
			ID:           "tabs",
			Destinations: r.tabDestinations(),
			Selected:     int(tab),
			OnSelect: func(index int) {
				r.app.SelectTab(app.Tabs()[index])
			},
		})
	})
}

// seedColor is the seed of the colour scheme: the brand colour of the desktop and
// of the web interface, from which the web's light primary colour, #88521C, is
// generated.
const seedColor uint32 = 0xFFF28C1B

// look is what the theme depends on; the theme is built again when it changes.
type look struct {
	dark      bool
	reduced   bool
	textScale float32
}

// applyTheme sets the theme of the frame. Its colours follow the system's light or
// dark appearance, and its text size and motion follow the system's preferences.
func (r *Root) applyTheme(c *ui.Context) {
	prefs := c.Preferences()
	current := look{dark: c.Theme().Dark, reduced: prefs.ReduceMotion, textScale: prefs.TextScale}
	if current != r.look {
		theme := kit.NewTheme(seedColor, current.dark, current.reduced, current.textScale)
		r.env.Theme = theme
		r.uiTheme = theme.UI()
		r.look = current
	}
	c.SetTheme(r.uiTheme)
	if prefs.ReduceMotion {
		c.DisableAnimations()
	}
}
