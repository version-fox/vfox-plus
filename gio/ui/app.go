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

// Package ui is the user interface of the Gio implementation of vfox-plus: the
// window, the shell with its navigation, and the pages, which follow
// docs/ui-spec.md. Its widgets are those of the toolkit in ui/kit.
package ui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/version-fox/vfox-plus/gio/assets"
	"github.com/version-fox/vfox-plus/gio/core/i18n"
	"github.com/version-fox/vfox-plus/gio/core/settings"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/platform"
)

// ApplicationID is the identifier of the application on the desktop: the
// Wayland application id and the X11 window class, which the desktop entry
// refers to.
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

// App is the user interface: it owns the window and lays out the pages.
type App struct {
	win         *app.Window
	model       *state.AppModel
	env         *kit.Env
	fonts       *kit.Fonts
	portal      atomic.Pointer[platform.Portal]
	environment map[string]string
	locales     []string

	themeKey themeKey
	l10n     atomic.Pointer[i18n.Localizations]
	english  i18n.ArbCatalogue

	mu         sync.Mutex
	pending    []func()
	appearance platform.Appearance
	clipboard  *string

	watched map[*state.VfoxTask]func()
	shell   shellState
	pages   pageStates
	window  windowState
	// windowAction asks the window system to do something with the window.
	windowAction func(system.Action)
	e2e          e2eHooks //nolint:unused // Used by the build with the tag e2e.
}

// themeKey is what the theme depends on: the theme is built again when it
// changes.
type themeKey struct {
	seed    uint32
	dark    bool
	reduced bool
}

// New creates the application.
func New(options Options) (*App, error) {
	iconFont, err := assets.Files.ReadFile("fonts/MaterialSymbolsRounded.ttf")
	if err != nil {
		return nil, err
	}
	iconCodes, err := assets.Files.ReadFile("icons.json")
	if err != nil {
		return nil, err
	}
	fonts, err := kit.LoadFonts(iconFont, iconCodes)
	if err != nil {
		return nil, err
	}
	english, err := readCatalogue(i18n.English)
	if err != nil {
		return nil, err
	}
	a := &App{
		model: options.Model, fonts: fonts, environment: options.Environment,
		locales: i18n.SystemLocales(options.Environment), english: english,
		watched: make(map[*state.VfoxTask]func()),
	}
	a.env = kit.NewEnv(kit.NewTheme(settings.DefaultSeedColor, false, false, fonts), services{a})
	a.syncTheme()
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

// services is what the window system offers the toolkit.
type services struct{ a *App }

func (s services) Invalidate() {
	if s.a.win != nil {
		s.a.win.Invalidate()
	}
}

func (s services) Copy(text string) {
	s.a.mu.Lock()
	s.a.clipboard = &text
	s.a.mu.Unlock()
	s.Invalidate()
}

func (s services) OpenURL(address string) { platform.OpenURL(address) }

// post runs f on the goroutine of the window, before the next frame. It may be
// called from any goroutine.
func (a *App) post(f func()) {
	a.mu.Lock()
	a.pending = append(a.pending, f)
	a.mu.Unlock()
	a.env.Services.Invalidate()
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
	app.ID = ApplicationID
	go func() {
		window := new(app.Window)
		a.win = window
		window.Option(
			app.Title(a.T("appTitle")),
			app.Size(unit.Dp(1280), unit.Dp(820)),
			app.MinSize(unit.Dp(360), unit.Dp(480)),
		)
		a.windowAction = window.Perform
		a.listen()
		go a.model.Start()
		go a.connectPortal()
		a.startE2E()
		if err := a.loop(window); err != nil {
			log.Printf("the window failed: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

// listen asks for a frame whenever the state changes.
func (a *App) listen() {
	invalidate := a.env.Services.Invalidate
	a.model.Settings.AddListener(invalidate)
	a.model.Vfox.AddListener(invalidate)
	a.model.Installed.AddListener(invalidate)
	a.model.Registry.AddListener(invalidate)
	a.model.VfoxConfig.AddListener(invalidate)
	a.model.Tasks.AddListener(invalidate)
	a.model.Tasks.OnFinished(a.announce)
}

// connectPortal connects to the desktop, which tells how applications should
// look, and follows it. There is no session bus in a container, and the
// application then keeps to its settings.
func (a *App) connectPortal() {
	portal, err := platform.Connect()
	if err != nil {
		return
	}
	a.portal.Store(portal)
	a.refreshAppearance(portal)
	portal.Watch(func() { a.refreshAppearance(portal) })
}

func (a *App) refreshAppearance(portal *platform.Portal) {
	appearance := portal.Appearance()
	a.mu.Lock()
	a.appearance = appearance
	a.mu.Unlock()
	a.env.Services.Invalidate()
}

func (a *App) loop(window *app.Window) error {
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			a.model.Dispose()
			return e.Err
		case app.WaylandViewEvent:
			a.onWaylandView(window, e)
		case app.ConfigEvent:
			a.onConfig(e.Config)
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			a.frame(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// frame lays out the window.
func (a *App) frame(gtx layout.Context) {
	a.runPending()
	a.syncTheme()
	a.syncLanguage()
	a.env.BeginFrame(gtx.Constraints.Max)
	c := kit.Ctx{Context: gtx, Env: a.env}
	if c.Constraints.Max.X < c.DP(compactWidth) {
		a.env.BottomInset = c.DP(kit.NavBarHeight)
	} else {
		a.env.BottomInset = 0
	}
	paint.Fill(gtx.Ops, a.env.Theme.Colors.Surface)
	if a.needsCaption() {
		kit.Column(kit.FlexOpts{Cross: kit.CrossStretch},
			kit.Rigid(a.captionBar()),
			kit.Expanded(1, func(c kit.Ctx) layout.Dimensions { return a.layoutShell(c) }),
		)(c)
	} else {
		a.layoutShell(c)
	}
	a.env.LayoutOverlays(c)
	a.watchTasks()
	a.flushClipboard(gtx)
	a.afterFrame()
}

func (a *App) flushClipboard(gtx layout.Context) {
	a.mu.Lock()
	text := a.clipboard
	a.clipboard = nil
	a.mu.Unlock()
	if text != nil {
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(*text))})
	}
}

// watchTasks asks for a frame whenever a task changes, such as when it
// reports its progress, and forgets the tasks that are gone.
func (a *App) watchTasks() {
	tasks := a.model.Tasks.Tasks()
	present := make(map[*state.VfoxTask]bool, len(tasks))
	for _, task := range tasks {
		present[task] = true
		if _, ok := a.watched[task]; !ok {
			a.watched[task] = task.AddListener(a.env.Services.Invalidate)
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
func (a *App) syncTheme() {
	current := a.model.Settings.Settings()
	a.mu.Lock()
	appearance := a.appearance
	a.mu.Unlock()
	seed := current.SeedColor
	if current.UseSystemColor && appearance.Accent != nil {
		seed = *appearance.Accent
	}
	dark := false
	switch current.Theme {
	case settings.ThemeDark:
		dark = true
	case settings.ThemeSystem:
		dark = appearance.Dark != nil && *appearance.Dark
	}
	reduced := appearance.ReducedMotion != nil && *appearance.ReducedMotion || os.Getenv("VFOX_PLUS_REDUCED_MOTION") != ""
	key := themeKey{seed: seed, dark: dark, reduced: reduced}
	if a.env.Theme != nil && key == a.themeKey && a.themeKey != (themeKey{}) {
		return
	}
	a.themeKey = key
	a.env.Theme = kit.NewTheme(seed, dark, reduced, a.fonts)
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
		a.win.Option(app.Title(a.T("appTitle")))
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
			// Failures stay until the user dismisses them, so that they can
			// be read in full.
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
func (a *App) openURL(address string) { a.env.Services.OpenURL(address) }

// notify shows a short message in a snack bar.
func (a *App) notify(text string) { a.env.ShowSnackbar(kit.Snack{Text: text}) }
