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
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"

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
}

// desktop is what the application asks of the desktop, which tests replace.
type desktop struct {
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
	win         *mygo.Window
	model       *state.AppModel
	env         *kit.Env
	environment map[string]string
	locales     []string
	desktop     desktop

	themeKey themeKey
	uiTheme  *mui.Theme
	l10n     atomic.Pointer[i18n.Localizations]
	english  i18n.ArbCatalogue

	mu      sync.Mutex
	pending []func()

	watched map[*state.VfoxTask]func()
	shell   shellState
	pages   pageStates
	e2e     e2eHooks //nolint:unused // Used by the build with the tag e2e.
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
	outlined, err := assets.Files.ReadFile("fonts/MaterialSymbolsRounded.ttf")
	if err != nil {
		return nil, err
	}
	filled, err := assets.Files.ReadFile("fonts/MaterialSymbolsRoundedFilled.ttf")
	if err != nil {
		return nil, err
	}
	codes, err := assets.Files.ReadFile("icons.json")
	if err != nil {
		return nil, err
	}
	loadIcons(outlined, filled, codes)
	english, err := readCatalogue(i18n.English)
	if err != nil {
		return nil, err
	}
	a := &App{
		model: options.Model, environment: options.Environment,
		locales: i18n.SystemLocales(options.Environment), english: english,
		watched: make(map[*state.VfoxTask]func()),
	}
	a.desktop = desktop{openURL: openURL, copy: copyText, choose: a.chooseFile}
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

// invalidate asks for another frame, from any goroutine.
func (a *App) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

// post runs f on the main thread, before the next frame. It may be called from
// any goroutine.
func (a *App) post(f func()) {
	a.mu.Lock()
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

// Run opens the window and runs the application until the window is closed.
func (a *App) Run() {
	prepareProcess()
	mygo.App.SetName(a.T("appTitle"))
	mygo.App.SetVersion(Version)
	// The window has no menu bar: the application is the pages, and the text
	// fields bring their own editing commands.
	mygo.App.SetMenu(nil)
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title: a.T("appTitle"), Width: 1280, Height: 820, MinWidth: 360, MinHeight: 480,
			Content: mui.View(a.view),
		})
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

// listen asks for a frame whenever the state changes.
func (a *App) listen() {
	a.model.Settings.AddListener(a.invalidate)
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
	a.afterFrame(k)
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
	if a.win != nil {
		a.win.SetTitle(a.T("appTitle"))
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
