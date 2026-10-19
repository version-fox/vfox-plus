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
	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// activationCommand is the line that makes a shell use the versions that vfox+
// chooses.
const activationCommand = `eval "$(vfox activate bash)"`

// The links that the user interface opens.
const (
	linkVfoxInstall = "https://vfox.dev/guides/quick-start.html"
	linkVfoxDocs    = "https://vfox.dev/"
	linkRepository  = "https://github.com/version-fox/vfox-plus"
)

// homePage is the overview: what is installed, what is in use, and shortcuts.
func (a *App) homePage(k kit.Ctx) {
	kit.Pane(k, func() {
		sdks, ok := a.model.Installed.Value()
		if !ok {
			a.loadingOrError(k, a.T("sdksLoadError"), a.model.Installed.Error(), func() { a.model.Installed.Load() })
			return
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
		mui.Scroll(k.Context).Grow(1).Children(func() {
			kit.PageHeader(k, a.T("appTitle"), subtitle, nil, func() {
				a.refreshButton(k, "refresh", a.model.Installed.IsLoading(), func() { a.model.Installed.Load() })
			})
			kit.ContentWidth(k, 1080, func() {
				mui.Column(k.Context).Padding(0, gapXL, gapXL, gapXL).Children(func() {
					a.stats(k, len(sdks), versions, len(active))
					kit.GroupHeader(k, a.T("homeActiveTitle"), nil)
					a.activeList(k, active)
					a.shortcuts(k).Margin(gapXL, 0, 0, 0)
				})
			})
		})
	}).Grow(1)
}

// activeList is the segmented list of the plugins that have a version in use.
func (a *App) activeList(k kit.Ctx, active []core.InstalledSdk) {
	cs := k.Colors()
	if len(active) == 0 {
		kit.ListGroup(k, kit.ListItemSpec{
			ID: "empty", Title: a.T("homeActiveEmpty"), OnClick: func() { a.showSdk("") },
			Leading: func(foreground mui.Color) { kit.Icon(k, "info", 24, foreground, 0) },
		})
		return
	}
	items := make([]kit.ListItemSpec, len(active))
	for i, sdk := range active {
		items[i] = kit.ListItemSpec{
			ID: "sdk:" + sdk.Name, Title: sdk.Name, Subtitle: sdk.Current,
			Leading:  func(mui.Color) { avatar(k, sdk.Name, 40) },
			Trailing: func() { kit.Icon(k, "chevronRight", 24, cs.OnSurfaceVariant, 0) },
			OnClick:  func() { a.showSdk(sdk.Name) },
		}
	}
	kit.ListGroup(k, items...)
}

// stats are the three tiles with the numbers of plugins, versions and versions
// in use.
func (a *App) stats(k kit.Ctx, plugins, versions, active int) {
	compact := a.pageWidth(k) < 560+2*gapXL
	cs := k.Colors()
	gap := gapM
	if compact {
		gap = gapS
	}
	mui.Row(k.Context).Gap(gap).AlignItems(mui.Stretch).Children(func() {
		a.statTile(k, "plugins", compact, plugins, a.T("homeStatPlugins", "count", plugins), "plugins",
			shapes.ShapeCookie9, cs.PrimaryContainer, cs.OnPrimaryContainer)
		a.statTile(k, "versions", compact, versions, a.T("homeStatVersions", "count", versions), "versions",
			shapes.ShapeClover4, cs.SecondaryContainer, cs.OnSecondaryContainer)
		a.statTile(k, "active", compact, active, a.T("homeStatActive"), "current",
			shapes.ShapeSunny, cs.TertiaryContainer, cs.OnTertiaryContainer)
	})
}

func (a *App) statTile(k kit.Ctx, id string, compact bool, value int, label, icon string, shape *shapes.Polygon,
	background, foreground mui.Color) {
	pad := gapXL
	numberStyle := kit.Emph(kit.DisplaySmall)
	if compact {
		pad = gapL
		numberStyle = kit.Emph(kit.HeadlineLarge)
	}
	numberStyle.Line = numberStyle.Size
	kit.Card(k, kit.CardSpec{
		ID: "stat:" + id, Color: background, Radius: kit.CornerExtraLarge, Content: foreground, Name: label,
		OnClick: func() { a.showSdk("") },
		Children: func() {
			// A large, partly hidden shape gives the tile its character.
			kit.Shape(k, shape, 140, foreground.Alpha(0.08), nil).Absolute().Right(-28).Bottom(-36)
			mui.Column(k.Context).Height(136).Padding(pad).Children(func() {
				kit.Icon(k, icon, 24, foreground, 1)
				mui.Spacer(k.Context)
				kit.Line(k, itoa(value), numberStyle, foreground)
				kit.Label(k, label, kit.Emph(kit.LabelLarge), foreground, kit.Lines(2)).Margin(gapXS, 0, 0, 0)
			})
		},
	}).Grow(1).Basis(0).Shrink(1)
}

// shortcuts are the cards under the list: how to use the versions in a terminal,
// and where to find more.
func (a *App) shortcuts(k kit.Ctx) mui.Element {
	if a.pageWidth(k) < 760+2*gapXL {
		return mui.Column(k.Context).Gap(gapM).Children(func() {
			a.shellCard(k)
			a.exploreCard(k)
			a.docsCard(k)
		})
	}
	return mui.Row(k.Context).Gap(gapM).AlignItems(mui.Stretch).Children(func() {
		mui.Column(k.Context).Grow(3).Basis(0).Children(func() { a.shellCard(k).Grow(1) })
		mui.Column(k.Context).Grow(2).Basis(0).Gap(gapM).Children(func() {
			a.exploreCard(k).Grow(1).Basis(0)
			a.docsCard(k).Grow(1).Basis(0)
		})
	})
}

func (a *App) exploreCard(k kit.Ctx) mui.Element {
	return a.shortcutCard(k, "explore", "explore", a.T("homeExploreTitle"), a.T("homeExploreBody"), "chevronRight",
		func() { a.navigate(destPlugins) })
}

func (a *App) docsCard(k kit.Ctx) mui.Element {
	return a.shortcutCard(k, "docs", "docs", a.T("homeDocsTitle"), a.T("homeDocsBody"), "openInNew",
		func() { a.openURL(linkVfoxDocs) })
}

func (a *App) shellCard(k kit.Ctx) mui.Element {
	cs := k.Colors()
	return kit.Card(k, kit.CardSpec{
		ID: "shell", Color: cs.SurfaceContainerHighest, Radius: kit.CornerLarge,
		Children: func() {
			mui.Column(k.Context).Padding(gapXL).Children(func() {
				mui.Row(k.Context).Gap(gapM).Children(func() {
					kit.Icon(k, "terminal", 24, cs.Primary, 0)
					kit.Line(k, a.T("homeShellTitle"), kit.Emph(kit.TitleMedium), cs.OnSurface).Grow(1)
				})
				kit.Label(k, a.T("homeShellBody"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant).Margin(gapS, 0, 0, 0)
				kit.Card(k, kit.CardSpec{
					ID: "command", Color: cs.SurfaceContainerLowest, Radius: kit.CornerMedium,
					Children: func() {
						mui.Row(k.Context).Padding(0, gapXS, 0, gapL).Children(func() {
							kit.Label(k, activationCommand, kit.Type(kit.BodyMedium), cs.OnSurface, kit.Selectable(), kit.Monospace()).
								Grow(1).Shrink(1)
							kit.IconButton(k, kit.IconButtonSpec{ID: "copy-activation", Icon: "copy", Tooltip: a.T("actionCopy"), OnClick: func() {
								a.copy(activationCommand)
								a.notify(a.T("actionCopied"))
							}})
						})
					},
				}).Margin(gapM, 0, 0, 0)
			})
		},
	})
}

func (a *App) shortcutCard(k kit.Ctx, id, icon, title, body, trailing string, onClick func()) mui.Element {
	cs := k.Colors()
	return kit.Card(k, kit.CardSpec{
		ID: "shortcut:" + id, Color: cs.SurfaceContainer, Radius: kit.CornerLarge, Content: cs.OnSurface,
		Name: title, OnClick: onClick,
		Children: func() {
			mui.Row(k.Context).Gap(gapL).Padding(gapL).Children(func() {
				kit.Shape(k, shapes.ShapeCookie6, 48, cs.SecondaryContainer, func() {
					kit.Icon(k, icon, 24, cs.OnSecondaryContainer, 0)
				})
				mui.Column(k.Context).Grow(1).Shrink(1).Justify(mui.Center).Children(func() {
					kit.Label(k, title, kit.Emph(kit.TitleSmall), cs.OnSurface)
					kit.Label(k, body, kit.Type(kit.BodySmall), cs.OnSurfaceVariant).Margin(gapXS, 0, 0, 0)
				})
				kit.Icon(k, trailing, 24, cs.OnSurfaceVariant, 0)
			})
		},
	})
}
