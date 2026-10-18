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
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/platform"
)

// captionHeight is the height in dp of the title bar that the application draws.
const captionHeight = 40

// windowState is what the window system said about the window.
type windowState struct {
	// wayland says that the window is a Wayland window. Wayland compositors
	// either draw the title bar of a window (KDE, Sway) or leave that to the
	// application (GNOME, whose Mutter has no server-side decorations); Gio
	// draws a title bar of its own in the colours of its sample theme then,
	// which does not belong to this interface, so the application draws it.
	wayland bool
	// known says that the system has reported the configuration of the window.
	known bool
	// decorated says that the system draws the title bar.
	decorated bool
	mode      app.WindowMode
}

// onWaylandView takes over the title bar when the window is a Wayland window:
// without it, Gio draws one itself where the compositor does not.
func (a *App) onWaylandView(window *app.Window, e app.WaylandViewEvent) {
	a.window.wayland = e.Valid()
	if a.window.wayland {
		window.Option(app.Decorated(false))
	}
}

// onConfig notes the configuration of the window that the system reported.
func (a *App) onConfig(config app.Config) {
	a.window.known = true
	a.window.decorated = config.Decorated
	a.window.mode = config.Mode
}

// needsCaption says whether the application draws the title bar: when it is a
// Wayland window that the compositor does not decorate, and not in full screen.
func (a *App) needsCaption() bool {
	w := a.window
	return w.wayland && w.known && !w.decorated && w.mode != app.Fullscreen
}

// act asks the window system to do something with the window.
func (a *App) act(action system.Action) {
	if a.windowAction != nil {
		a.windowAction(action)
	}
}

// windowButtons is how the title bar lays its buttons out: the one of the
// desktop, or else the one of Ubuntu.
func (a *App) windowButtons() platform.WindowButtons {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.appearance.WindowButtons != nil {
		return *a.appearance.WindowButtons
	}
	return platform.DefaultWindowButtons()
}

// captionButton is a button of the title bar, by the name it has in the layout
// of the desktop.
func (a *App) captionButton(name string) kit.Widget {
	switch name {
	case "minimize":
		return kit.WindowButton{
			ID: "window-minimize", Glyph: kit.GlyphMinimize, Name: name,
			OnClick: func() { a.act(system.ActionMinimize) },
		}.W()
	case "maximize":
		button := kit.WindowButton{ID: "window-maximize", Glyph: kit.GlyphMaximize, Name: name}
		button.OnClick = func() { a.act(system.ActionMaximize) }
		if a.window.mode == app.Maximized {
			button.ID, button.Glyph, button.Name = "window-restore", kit.GlyphRestore, "restore"
			button.OnClick = func() { a.act(system.ActionUnmaximize) }
		}
		return button.W()
	default:
		return kit.WindowButton{
			ID: "window-close", Glyph: kit.GlyphClose, Name: name,
			OnClick: func() { a.act(system.ActionClose) },
		}.W()
	}
}

// captionBar is the title bar: the title, which the user drags to move the
// window, between the window buttons.
func (a *App) captionBar() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		buttons := a.windowButtons()
		var left, right []kit.FlexChild
		for _, name := range buttons.Left {
			left = append(left, kit.Rigid(a.captionButton(name)), kit.Rigid(kit.Space(gapXS, 0)))
		}
		for _, name := range buttons.Right {
			right = append(right, kit.Rigid(kit.Space(gapXS, 0)), kit.Rigid(a.captionButton(name)))
		}
		title := kit.WindowDrag(kit.Fill(kit.Pad(kit.Inset{Left: gapL, Right: gapL},
			kit.Align(layout.W, kit.Line(a.T("appTitle"), kit.Emph(kit.TitleSmall), colors(c).OnSurfaceVariant)))))
		children := append(left, kit.Expanded(1, title))
		children = append(children, right...)
		row := kit.Row(kit.FlexOpts{Cross: kit.CrossCenter}, children...)
		return kit.Sized(0, captionHeight, kit.Pad(kit.Inset{Left: gapS, Right: gapS}, row))(c)
	}
}
