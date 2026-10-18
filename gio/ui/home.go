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
	"image"
	"image/color"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

// activationCommand is the line that makes a shell use the versions that
// vfox+ chooses.
const activationCommand = `eval "$(vfox activate bash)"`

// The links that the user interface opens.
const (
	linkVfoxInstall = "https://vfox.dev/guides/quick-start.html"
	linkVfoxDocs    = "https://vfox.dev/"
	linkRepository  = "https://github.com/version-fox/vfox-plus"
)

// homePage is the overview: what is installed, what is in use, and shortcuts.
func (a *App) homePage() kit.Widget {
	return kit.Pane(func(c kit.Ctx) layout.Dimensions {
		sdks, ok := a.model.Installed.Value()
		if !ok {
			return a.loadingOrError(a.T("sdksLoadError"), a.model.Installed.Error(), func() { a.model.Installed.Load() })(c)
		}
		var active []core.InstalledSdk
		versions := 0
		for _, sdk := range sdks {
			versions += len(sdk.Versions)
			if sdk.HasCurrent {
				active = append(active, sdk)
			}
		}
		subtitle := a.T("homeSubtitle", "version", orQuestionMark(a.model.Vfox.Version))
		if executable, ok := a.model.Vfox.Executable(); ok {
			subtitle += " · " + executable
		}
		header := kit.PageHeader(a.T("appTitle"), subtitle, nil,
			a.refreshButton("refresh", a.model.Installed.IsLoading(), func() { a.model.Installed.Load() }))
		var activeList kit.Widget
		if len(active) == 0 {
			activeList = kit.ListGroup(kit.ListItem{
				ID: "empty", Title: a.T("homeActiveEmpty"), OnClick: func() { a.showSdk("") },
				Leading: func(col color.NRGBA) kit.Widget { return kit.Icon("info", 24, col, 0) },
			})
		} else {
			items := make([]kit.ListItem, len(active))
			for i, sdk := range active {
				items[i] = kit.ListItem{
					ID: "sdk:" + sdk.Name, Title: sdk.Name, Subtitle: sdk.Current,
					Leading:  func(color.NRGBA) kit.Widget { return avatar(sdk.Name, 40) },
					Trailing: kit.Icon("chevronRight", 24, colors(c).OnSurfaceVariant, 0),
					OnClick:  func() { a.showSdk(sdk.Name) },
				}
			}
			activeList = kit.ListGroup(items...)
		}
		content := kit.Pad(kit.Inset{Left: gapXL, Right: gapXL, Bottom: gapXL}, kit.ContentWidth(1080, column(0,
			a.stats(len(sdks), versions, len(active)),
			kit.GroupHeader(a.T("homeActiveTitle"), nil),
			activeList,
			kit.Space(0, gapXL),
			a.shortcuts(),
		)))
		return scrollPage("home", header, content)(c)
	})
}

func orQuestionMark(value func() (string, bool)) string {
	if text, ok := value(); ok {
		return text
	}
	return "?"
}

// stats are the three tiles with the numbers of plugins, versions and versions
// in use.
func (a *App) stats(plugins, versions, active int) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		compact := c.Constraints.Max.X < c.DP(560)
		cs := colors(c)
		tile := func(id string, value int, label, icon string, shape *shapes.Polygon, background, foreground color.NRGBA) kit.FlexChild {
			return kit.Expanded(1, a.statTile(id, compact, value, label, icon, shape, background, foreground))
		}
		gap := gapM
		if compact {
			gap = gapS
		}
		return kit.Row(kit.FlexOpts{Gap: gap, Cross: kit.CrossStretch},
			tile("plugins", plugins, a.T("homeStatPlugins", "count", plugins), "plugins", shapes.ShapeCookie9, cs.PrimaryContainer, cs.OnPrimaryContainer),
			tile("versions", versions, a.T("homeStatVersions", "count", versions), "versions", shapes.ShapeClover4, cs.SecondaryContainer, cs.OnSecondaryContainer),
			tile("active", active, a.T("homeStatActive"), "current", shapes.ShapeSunny, cs.TertiaryContainer, cs.OnTertiaryContainer),
		)(c)
	}
}

func (a *App) statTile(id string, compact bool, value int, label, icon string, shape *shapes.Polygon, background, foreground color.NRGBA) kit.Widget {
	return kit.Card{
		ID: "stat:" + id, Color: background, Radius: kit.CornerExtraLarge, Content: foreground, Name: label,
		OnClick: func() { a.showSdk("") },
		Child: kit.Sized(0, 136, func(c kit.Ctx) layout.Dimensions {
			pad := gapXL
			numberStyle := kit.Emph(kit.DisplaySmall)
			if compact {
				pad = gapL
				numberStyle = kit.Emph(kit.HeadlineLarge)
			}
			numberStyle.Line = numberStyle.Size
			// A large, partly hidden shape gives the tile its character.
			decoration := kit.ShapeBox{Polygon: shape, Size: 140, Color: kit.Alpha(foreground, 0.08)}.W()
			place := func(stack, child image.Point) image.Point {
				return image.Pt(stack.X-child.X+c.DP(28), stack.Y-child.Y+c.DP(36))
			}
			text := kit.Pad(kit.All(pad), kit.Column(kit.FlexOpts{Cross: kit.CrossStart},
				kit.Rigid(kit.Icon(icon, 24, foreground, 1)),
				kit.Expanded(1, kit.Space(0, 0)),
				kit.Rigid(kit.Line(itoa(value), numberStyle, foreground)),
				kit.Rigid(kit.Space(0, gapXS)),
				kit.Rigid(kit.Label(label, kit.Emph(kit.LabelLarge), foreground, kit.Lines(2))),
			))
			return kit.Stack(layout.NW,
				kit.PlacedAt(place, decoration),
				kit.Expand(text),
			)(c)
		}),
	}.W()
}

// shortcuts are the cards under the list: how to use the versions in a
// terminal, and where to find more.
func (a *App) shortcuts() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		shell := a.shellCard()
		explore := a.shortcutCard("explore", "explore", a.T("homeExploreTitle"), a.T("homeExploreBody"), "chevronRight",
			func() { a.navigate(destPlugins) })
		docs := a.shortcutCard("docs", "docs", a.T("homeDocsTitle"), a.T("homeDocsBody"), "openInNew",
			func() { a.openURL(linkVfoxDocs) })
		if c.Constraints.Max.X < c.DP(760) {
			return column(gapM, shell, explore, docs)(c)
		}
		return kit.Row(kit.FlexOpts{Gap: gapM, Cross: kit.CrossStretch},
			kit.Expanded(3, shell),
			kit.Expanded(2, kit.Column(kit.FlexOpts{Gap: gapM, Cross: kit.CrossStretch},
				kit.Expanded(1, explore), kit.Expanded(1, docs))),
		)(c)
	}
}

func (a *App) shellCard() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		command := kit.Pad(kit.Inset{Left: gapL, Right: gapXS},
			kit.Row(kit.FlexOpts{Cross: kit.CrossCenter},
				kit.Expanded(1, kit.SelectableText("activation", activationCommand, kit.Type(kit.BodyMedium), cs.OnSurface, kit.Monospace())),
				kit.Rigid(kit.IconButton{ID: "copy-activation", Icon: "copy", Tooltip: a.T("actionCopy"), OnClick: func() {
					c.Env.Services.Copy(activationCommand)
					a.notify(a.T("actionCopied"))
				}}.W()),
			))
		body := kit.Pad(kit.All(gapXL), column(0,
			kit.Row(kit.FlexOpts{Gap: gapM, Cross: kit.CrossCenter},
				kit.Rigid(kit.Icon("terminal", 24, cs.Primary, 0)),
				kit.Expanded(1, kit.Label(a.T("homeShellTitle"), kit.Emph(kit.TitleMedium), cs.OnSurface)),
			),
			kit.Space(0, gapS),
			kit.Label(a.T("homeShellBody"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant),
			kit.Space(0, gapM),
			kit.Card{Color: cs.SurfaceContainerLowest, Radius: kit.CornerMedium, Child: command}.W(),
		))
		return kit.Card{Color: cs.SurfaceContainerHighest, Radius: kit.CornerLarge, Child: body}.W()(c)
	}
}

func (a *App) shortcutCard(id, icon, title, body, trailing string, onClick func()) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		leading := kit.ShapeBox{
			Polygon: shapes.ShapeCookie6, Size: 48, Color: cs.SecondaryContainer,
			Child: kit.Center(kit.Icon(icon, 24, cs.OnSecondaryContainer, 0)),
		}.W()
		text := kit.Column(kit.FlexOpts{Main: kit.MainCenter, Cross: kit.CrossStart, Min: true},
			kit.Rigid(kit.Label(title, kit.Emph(kit.TitleSmall), cs.OnSurface)),
			kit.Rigid(kit.Space(0, gapXS)),
			kit.Rigid(kit.Label(body, kit.Type(kit.BodySmall), cs.OnSurfaceVariant)),
		)
		row := kit.Pad(kit.All(gapL), kit.Row(kit.FlexOpts{Gap: gapL, Cross: kit.CrossCenter},
			kit.Rigid(leading), kit.Expanded(1, text), kit.Rigid(kit.Icon(trailing, 24, cs.OnSurfaceVariant, 0)),
		))
		return kit.Card{
			ID: "shortcut:" + id, Color: cs.SurfaceContainer, Radius: kit.CornerLarge, Content: cs.OnSurface,
			Name: title, OnClick: onClick, Child: row,
		}.W()(c)
	}
}
