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

// Package ui is the user interface of the mygo implementation of vfox-plus: the
// window, the shell with its navigation, and the pages, which follow
// docs/en/dev/ui-spec.md. Its widgets are those of the toolkit in ui/kit, built on the
// native user interface of mygo.
package ui

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/assets"
	"github.com/version-fox/vfox-plus/mygo/core/i18n"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// ApplicationID is the identifier of the application on the desktop: the Wayland
// application id and the X11 window class, which the desktop entry refers to.
const ApplicationID = "dev.vfox.VfoxPlus"

// Version is the version of the application.
const Version = "0.1.0"

// Options say what the application is made of.
type Options struct {
	// Model is the state of the application.
	Model *state.AppModel
	// Environment is the environment variables of the process.
	Environment map[string]string
	// Web is the web interface, which the settings switch on and off.
	Web Web
}

// surface is the window as the application uses it.
type surface interface {
	// Invalidate asks for another frame.
	Invalidate()
	// SetTitle names the window.
	SetTitle(title string)
	// Show makes the window visible, and Focus gives it the keyboard.
	Show()
	Focus()
}

// desktop is what the application asks of the desktop, which tests replace.
type desktop struct {
	// newWindow opens the window, which draws the application.
	newWindow func(options mygo.WindowOptions) surface
	// quit ends the application.
	quit func()
	// newTray puts the icon of the application in the tray of the desktop, which it
	// fails to do where there is none.
	newTray func(options trayOptions) (trayIcon, error)
	// pageOut tells the system that the pages of the files that the process maps are not wanted
	// for now: it is what the application does when it runs without a window.
	pageOut func()
	// openURL opens a web address in the browser of the user.
	openURL func(address string)
	// copy puts text on the clipboard.
	copy func(text string)
	// choose lets the user choose a file or a directory; it blocks, and returns
	// an empty path when the user gives up.
	choose func(options chooseOptions) string
}

// App is the user interface: it owns the window and builds the pages.
type App struct {
	// win is the window, or nil while there is none: before the application is ready,
	// and in the lightweight mode after the window was closed. Goroutines of any kind
	// ask it for a frame, so it is atomic.
	win         atomic.Pointer[surface]
	model       *state.AppModel
	web         Web
	env         *kit.Env
	environment map[string]string
	locales     []string
	desktop     desktop

	themeKey themeKey
	uiTheme  *mui.Theme
	l10n     atomic.Pointer[i18n.Localizations]
	english  i18n.ArbCatalogue

	// tray is the icon in the tray while the application runs without its window, and
	// nil otherwise. It is changed from the main thread, and read from goroutines of any
	// kind, which wait for the main thread while they do: trayMu guards the pointer and
	// is never held across a call to the icon.
	trayMu sync.Mutex
	tray   trayIcon

	mu      sync.Mutex
	pending []func()
	// trim is the timer that pages the files out once the window has stayed closed.
	trim *time.Timer
	// released is true while the application runs without its window, when the work that
	// is meant for the window is dropped.
	released bool

	watched map[*state.VfoxTask]func()
	shell   shellState
	pages   pageStates
}

// themeKey is what the theme depends on: the theme is built again when it
// changes.
type themeKey struct {
	seed      uint32
	dark      bool
	reduced   bool
	textScale float32
}

// New creates the application.
func New(options Options) (*App, error) {
	if options.Web == nil {
		return nil, fmt.Errorf("ui: the application needs a web interface to switch on and off")
	}
	table, err := assets.Files.ReadFile("icons.json")
	if err != nil {
		return nil, err
	}
	icons, err := fs.Sub(assets.Files, "icons")
	if err != nil {
		return nil, err
	}
	loadIcons(table, icons)
	english, err := readCatalogue(i18n.English)
	if err != nil {
		return nil, err
	}
	a := &App{
		model: options.Model, web: options.Web, environment: options.Environment,
		locales: i18n.SystemLocales(options.Environment), english: english,
		watched: make(map[*state.VfoxTask]func()),
	}
	a.desktop = desktop{
		openURL: openURL, copy: copyText, choose: a.chooseFile,
		newWindow: func(options mygo.WindowOptions) surface { return mygo.NewWindow(options) },
		quit:      mygo.App.Quit,
		newTray:   newNativeTray,
		pageOut:   func() { pageOutFiles() },
	}
	a.env = kit.NewEnv(kit.NewTheme(settings.DefaultSeedColor, false, false, 1))
	a.uiTheme = a.env.Theme.UI()
	a.syncLanguage()
	return a, nil
}

func readCatalogue(language i18n.SupportedLanguage) (i18n.ArbCatalogue, error) {
	content, err := assets.Files.ReadFile("i18n/app_" + string(language) + ".arb")
	if err != nil {
		return nil, err
	}
	return i18n.ParseCatalogue(content)
}

// window is the window of the application, or nil when there is none.
func (a *App) window() surface {
	if win := a.win.Load(); win != nil {
		return *win
	}
	return nil
}

// invalidate asks for another frame, from any goroutine.
func (a *App) invalidate() {
	if win := a.window(); win != nil {
		win.Invalidate()
	}
}

// post runs f on the main thread, before the next frame. It may be called from
// any goroutine. Without a window it does nothing: what f would show has no
// place to be shown in.
func (a *App) post(f func()) {
	a.mu.Lock()
	if a.released {
		a.mu.Unlock()
		return
	}
	a.pending = append(a.pending, f)
	a.mu.Unlock()
	a.invalidate()
}

func (a *App) runPending() {
	a.mu.Lock()
	pending := a.pending
	a.pending = nil
	a.mu.Unlock()
	for _, f := range pending {
		f()
	}
}

// Run opens the window and runs the application until it quits: when the window
// is closed, unless the lightweight mode keeps the application running without it.
// There is one application per user: a second one asks the first to show its
// window, and ends without a word.
func (a *App) Run() {
	prepareProcess()
	mygo.App.SetName(a.T("appTitle"))
	mygo.App.SetVersion(Version)
	// The window has no menu bar: the application is the pages, and the text
	// fields bring their own editing commands.
	mygo.App.SetMenu(nil)
	if !a.allowsSeveralInstances() && !mygo.App.RequestSingleInstanceLock() {
		return
	}
	// Only the application that runs serves the web interface.
	a.web.Start()
	defer a.web.Close()
	mygo.App.OnWindowAllClosed(a.windowsClosed)
	mygo.App.OnSecondInstance(func([]string, string) { a.showWindow() })
	// However the application is ended, the icon goes before it does.
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.removeTray() })
	mygo.App.WhenReady(func() {
		a.useSystemLanguages(mygo.App.PreferredLanguages())
		a.openWindow()
		a.listen()
		go a.model.Start()
		a.startE2E()
	})
	err := mygo.App.Run()
	a.model.Dispose()
	if err != nil {
		log.Printf("the window failed: %v", err)
		os.Exit(1)
	}
}

// openWindow opens the window of the application.
func (a *App) openWindow() {
	a.mu.Lock()
	a.released = false
	a.mu.Unlock()
	width, height := a.windowSize()
	win := a.desktop.newWindow(mygo.WindowOptions{
		Title: a.T("appTitle"), Width: width, Height: height, MinWidth: 360, MinHeight: 480,
		Content: mui.View(a.view),
	})
	a.win.Store(&win)
	a.cancelTrim()
	a.hideTray()
}

// showWindow brings the window of the application to the front, and opens it
// again when the application runs without one.
func (a *App) showWindow() {
	win := a.window()
	if win == nil {
		a.openWindow()
		return
	}
	win.Show()
	win.Focus()
}

// windowsClosed runs when the last window has been closed. The application
// quits, unless the lightweight mode is on and the web interface is up, which
// is what it would keep running for: then the application goes on without its
// window, and the web interface and the tasks with it.
func (a *App) windowsClosed() {
	if !a.keepsRunning() {
		a.desktop.quit()
		return
	}
	a.releaseWindow()
}

// keepsRunning tells whether closing the window leaves the application running.
func (a *App) keepsRunning() bool {
	return a.model.Settings.Settings().Lightweight && a.web.Status().Running
}

// releaseWindow lets go of everything that the window held: its widgets, the
// page the user was on, what the lists remembered, and the memory they took.
// A window that opens later starts from a clean state, as the application does.
func (a *App) releaseWindow() {
	a.win.Store(nil)
	a.mu.Lock()
	a.released = true
	a.pending = nil
	a.mu.Unlock()
	for task, stop := range a.watched {
		stop()
		delete(a.watched, task)
	}
	a.env = kit.NewEnv(kit.NewTheme(settings.DefaultSeedColor, false, false, 1))
	a.uiTheme = a.env.Theme.UI()
	a.themeKey = themeKey{}
	a.shell = shellState{}
	a.pages = pageStates{}
	// The icon in the tray is the way back to the window.
	a.showTray()
	// What the window held is garbage now; the runtime would return it to the
	// system in its own time, and the lightweight mode is about not waiting for that.
	go debug.FreeOSMemory()
	a.scheduleTrim()
}

// pageOutDelay is how long the window has to stay closed before the files that the process maps
// are paged out: the window and the icon are torn down and made in that time, which runs code
// that would only bring the pages back, and a window that opens again in it needs them.
var pageOutDelay = 2 * time.Second

// scheduleTrim pages the files that the process maps out once the window has stayed closed
// for pageOutDelay, and returns what that took to the runtime. Most of the resident memory of the
// application without its window is clean pages of its code, its fonts and its libraries, which
// nothing runs while there is no window: the system takes them back when it needs the memory, and
// being told that it may do so at once keeps the resident set at what the application uses.
func (a *App) scheduleTrim() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.trim != nil {
		a.trim.Stop()
	}
	a.trim = time.AfterFunc(pageOutDelay, func() {
		if a.window() != nil {
			return
		}
		a.desktop.pageOut()
		debug.FreeOSMemory() // What the paging itself allocated.
	})
}

// cancelTrim lets the files be: a window is open, and runs what they hold.
func (a *App) cancelTrim() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.trim != nil {
		a.trim.Stop()
		a.trim = nil
	}
}

// listen asks for a frame whenever the state changes.
func (a *App) listen() {
	a.web.AddListener(a.invalidate)
	a.model.Settings.AddListener(a.invalidate)
	a.model.Settings.AddListener(a.refreshTray)
	a.model.Vfox.AddListener(a.invalidate)
	a.model.Installed.AddListener(a.invalidate)
	a.model.Registry.AddListener(a.invalidate)
	a.model.VfoxConfig.AddListener(a.invalidate)
	a.model.Tasks.AddListener(a.invalidate)
	a.model.Tasks.OnFinished(a.announce)
}

// view builds the window: the shell, the dialogs and the snack bar.
func (a *App) view(c *mui.Context) {
	a.runPending()
	// The theme of the frame follows the desktop until the application sets its
	// own: the dark appearance of the desktop is what it starts from.
	desktopDark := c.Theme().Dark
	a.syncTheme(desktopDark, c.Preferences())
	a.syncLanguage()
	c.SetTheme(a.uiTheme)
	if os.Getenv("VFOX_PLUS_REDUCED_MOTION") != "" {
		c.DisableAnimations()
	}
	k := a.env.Begin(c)
	width, _ := c.Size()
	a.env.Compact = width < compactWidth
	a.layoutShell(k)
	a.env.LayoutDialogs(k)
	a.env.LayoutSnackbars(k)
	a.watchTasks()
}

// watchTasks asks for a frame whenever a task changes, such as when it reports
// its progress, and forgets the tasks that are gone.
func (a *App) watchTasks() {
	tasks := a.model.Tasks.Tasks()
	present := make(map[*state.VfoxTask]bool, len(tasks))
	for _, task := range tasks {
		present[task] = true
		if _, ok := a.watched[task]; !ok {
			a.watched[task] = task.AddListener(a.invalidate)
		}
	}
	for task, stop := range a.watched {
		if !present[task] {
			stop()
			delete(a.watched, task)
		}
	}
}

// syncTheme builds the theme again when the settings or the desktop changed
// what it depends on.
func (a *App) syncTheme(desktopDark bool, preferences mui.Preferences) {
	current := a.model.Settings.Settings()
	seed := current.SeedColor
	if current.UseSystemColor && preferences.Accent.A != 0 {
		seed = 0xFF000000 | uint32(preferences.Accent.R)<<16 | uint32(preferences.Accent.G)<<8 | uint32(preferences.Accent.B)
	}
	dark := false
	switch current.Theme {
	case settings.ThemeDark:
		dark = true
	case settings.ThemeSystem:
		dark = desktopDark
	}
	reduced := preferences.ReduceMotion || os.Getenv("VFOX_PLUS_REDUCED_MOTION") != ""
	key := themeKey{seed: seed, dark: dark, reduced: reduced, textScale: preferences.TextScale}
	if a.themeKey == key {
		return
	}
	a.themeKey = key
	a.env.Theme = kit.NewTheme(seed, dark, reduced, preferences.TextScale)
	a.uiTheme = a.env.Theme.UI()
}

// useSystemLanguages puts the languages that the system says the user reads before the
// locales of the environment, and loads the messages again. The environment is all that
// there is before the application runs, and Windows has no variable for the display
// language, which mygo can tell only once the application is ready. On Linux it reads
// the same variables, so nothing changes there.
func (a *App) useSystemLanguages(preferred []string) {
	a.locales = append(slices.Clone(preferred), a.locales...)
	a.syncLanguage()
}

// syncLanguage loads the messages of the language the settings ask for.
func (a *App) syncLanguage() {
	language := i18n.ResolveLanguage(a.model.Settings.Settings().Language, a.locales)
	if current := a.l10n.Load(); current != nil && current.Language == language {
		return
	}
	messages := a.english
	if language != i18n.English {
		if loaded, err := readCatalogue(language); err == nil {
			messages = loaded
		}
	}
	a.l10n.Store(i18n.NewLocalizations(language, messages, a.english))
	if win := a.window(); win != nil {
		win.SetTitle(a.T("appTitle"))
	}
}

// T returns the message key in the language of the user interface. The
// arguments are names and values in turns: T("homeSubtitle", "version", "1.0").
func (a *App) T(key string, pairs ...any) string {
	return a.l10n.Load().T(key, i18n.Arguments(pairs...))
}

// announce tells the user that a task has finished, in a snack bar.
func (a *App) announce(task *state.VfoxTask) {
	if task.Status() == state.TaskCancelled {
		return
	}
	a.post(func() {
		title := a.taskTitle(task.Command)
		failed := task.Status() == state.TaskFailed
		if !failed && a.shell.destination == destTasks {
			return
		}
		snack := kit.Snack{CloseLabel: a.T("actionClose")}
		if failed {
			message, _ := task.ErrorMessage()
			snack.Text = strings.TrimSpace(fmt.Sprintf("%s\n%s", a.T("taskFailed", "title", title), message))
			// Failures stay until the user dismisses them, so that they can be
			// read in full.
			snack.Persist, snack.Closable = true, true
			snack.Action = a.T("taskView")
			snack.OnAction = func() { a.navigate(destTasks) }
		} else {
			snack.Text = a.T("taskSucceeded", "title", title)
		}
		a.env.ShowSnackbar(snack)
	})
}

// openURL opens a web address in the browser of the user.
func (a *App) openURL(address string) { a.desktop.openURL(address) }

// copy puts text on the clipboard.
func (a *App) copy(text string) { a.desktop.copy(text) }

// notify shows a short message in a snack bar.
func (a *App) notify(text string) {
	a.env.ShowSnackbar(kit.Snack{Text: text})
	a.invalidate()
}
