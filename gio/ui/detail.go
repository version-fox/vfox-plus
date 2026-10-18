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
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/core/vfox"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

// The tabs of the details of a plugin.
const (
	tabInstalled = "installed"
	tabAvailable = "available"
)

// narrowHeader is the width in dp below which the header of the details puts
// its buttons on a row of their own.
const narrowHeader = 520

// detailState is the state of the details of one plugin.
type detailState struct {
	name       string
	controller *state.SdkDetailController
	tab        string
	stops      []func()
}

func (d *detailState) dispose() {
	for _, stop := range d.stops {
		stop()
	}
	d.controller.Dispose()
}

// detailFor returns the state of the details of the plugin name. It is
// created when the user selects the plugin, and replaced when they select
// another.
func (a *App) detailFor(name string) *detailState {
	if current := a.pages.detail; current != nil && current.name == name {
		return current
	}
	if a.pages.detail != nil {
		a.pages.detail.dispose()
	}
	controller := a.model.SdkDetail(name)
	d := &detailState{name: name, controller: controller, tab: tabInstalled}
	controller.Info.Load()
	invalidate := a.env.Services.Invalidate
	d.stops = append(d.stops,
		controller.AddListener(invalidate),
		controller.Info.AddListener(invalidate),
		a.model.Tasks.OnFinished(func(task *state.VfoxTask) {
			if task.Command.Sdk == name && task.Command.Operation == vfox.OperationUpdatePlugin {
				controller.Info.Load()
			}
		}),
	)
	a.pages.detail = d
	return d
}

func (a *App) showTab(d *detailState, tab string) {
	d.tab = tab
	if tab == tabAvailable {
		d.controller.Versions.EnsureLoaded()
	}
	a.env.Services.Invalidate()
}

// detailBody is what the details show under the tabs: a number of items,
// laid out lazily.
type detailBody struct {
	count int
	key   func(i int) any
	item  func(c kit.Ctx, i int) layout.Dimensions
}

func single(key string, w kit.Widget) detailBody {
	return detailBody{
		count: 1,
		key:   func(int) any { return key },
		item:  func(c kit.Ctx, _ int) layout.Dimensions { return w(c) },
	}
}

// detailView shows the versions of an installed plugin, and installs more.
// onBack, when it is not nil, adds a button that leaves the details.
func (a *App) detailView(name string, onBack func()) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		d := a.detailFor(name)
		sdk, found := a.model.InstalledSdk(name)
		if !found {
			sdk = core.InstalledSdk{Name: name}
		}
		info, hasInfo := d.controller.Info.Value()
		installed := make(map[string]bool, len(sdk.Versions))
		for _, version := range sdk.Versions {
			installed[version] = true
		}

		header := a.detailHeader(sdk, info, hasInfo, onBack)
		tabs := kit.Pad(kit.Inset{Left: gapXL, Right: gapXL, Bottom: gapL}, kit.Align(layout.W, kit.ButtonGroup{
			ID: "tabs", Selected: d.tab, OnSelect: func(tab string) { a.showTab(d, tab) },
			Segments: []kit.Segment{
				{Value: tabInstalled, Label: fmt.Sprintf("%s (%d)", a.T("sdkTabInstalled"), len(sdk.Versions)), Icon: "versions"},
				{Value: tabAvailable, Label: a.T("sdkTabAvailable"), Icon: "install"},
			},
		}.W()))
		head := []kit.Widget{header, tabs}

		var body detailBody
		if d.tab == tabInstalled {
			body = a.installedBody(d, sdk)
		} else {
			body = a.availableBody(c, d, sdk, installed)
		}
		return kit.List("detail:"+name, kit.ListSource{
			Count: len(head) + body.count + 1,
			Key: func(i int) any {
				switch {
				case i < len(head):
					return i
				case i-len(head) < body.count:
					return body.key(i - len(head))
				default:
					return "end"
				}
			},
			Item: func(c kit.Ctx, i int) layout.Dimensions {
				switch {
				case i < len(head):
					return head[i](c)
				case i-len(head) < body.count:
					return body.item(c, i-len(head))
				default:
					return kit.Space(0, gapXL)(c)
				}
			},
		})(c)
	}
}

func (a *App) installedBody(d *detailState, sdk core.InstalledSdk) detailBody {
	if len(sdk.Versions) == 0 {
		return single("empty", kit.StatusMessage{
			Icon: "install", Shape: shapes.ShapePill, Title: a.T("sdkInstalledEmpty", "name", sdk.Name),
			Actions: []kit.Widget{kit.Button{
				ID: "browse", Kind: kit.FilledButton, Icon: "install", Label: a.T("sdkBrowseVersions"),
				OnClick: func() { a.showTab(d, tabAvailable) },
			}.W()},
		}.W())
	}
	return detailBody{
		count: len(sdk.Versions),
		key:   func(i int) any { return "iv:" + sdk.Versions[i] },
		item: func(c kit.Ctx, i int) layout.Dimensions {
			row := a.installedVersionRow(sdk, sdk.Versions[i], kit.PositionOf(i, len(sdk.Versions)))
			return kit.Pad(kit.Inset{Left: gapXL, Right: gapXL, Bottom: hairline}, row)(c)
		},
	}
}

func (a *App) availableBody(c kit.Ctx, d *detailState, sdk core.InstalledSdk, installed map[string]bool) detailBody {
	controller := d.controller
	if !controller.Versions.HasValue() {
		if err := controller.Versions.Error(); err != nil {
			return single("status", a.errorView(a.T("availableError", "name", sdk.Name), err, func() { controller.Versions.Load() }))
		}
		return single("status", a.loadingView(a.T("availableLoading")))
	}
	visible := controller.VisibleVersions(installed)
	cs := colors(c)
	toolbar := kit.Pad(kit.Inset{Left: gapXL, Right: gapXL, Bottom: gapM}, kit.Wrap{Spacing: gapM, RunSpacing: gapS, CrossCenter: true}.Layout(
		kit.Sized(280, 0, kit.TextField{
			ID: "filter", Hint: a.T("availableSearchHint"), LeadingIcon: "search", Dense: true,
			Value: controller.Query(), OnChange: controller.SetQuery,
		}.W()),
		kit.ButtonGroup{
			ID: "filters", Selected: string(controller.Filter()),
			OnSelect: func(value string) { controller.SetFilter(state.VersionFilter(value)) },
			Segments: []kit.Segment{
				{Value: string(state.VersionFilterAll), Label: a.T("availableFilterAll")},
				{Value: string(state.VersionFilterWithNote), Label: a.T("availableFilterTagged")},
				{Value: string(state.VersionFilterInstalled), Label: a.T("availableFilterInstalled")},
			},
		}.W(),
		kit.Label(a.T("availableCount", "count", len(visible)), kit.Type(kit.LabelLarge), cs.OnSurfaceVariant),
	))
	if len(visible) == 0 {
		empty := kit.StatusMessage{Icon: "notFound", Title: a.T("availableEmpty")}.W()
		return detailBody{
			count: 2,
			key:   func(i int) any { return [2]string{"toolbar", "empty"}[i] },
			item: func(c kit.Ctx, i int) layout.Dimensions {
				return [2]kit.Widget{toolbar, empty}[i](c)
			},
		}
	}
	return detailBody{
		count: len(visible) + 1,
		key: func(i int) any {
			if i == 0 {
				return "toolbar"
			}
			return "av:" + visible[i-1].Version
		},
		item: func(c kit.Ctx, i int) layout.Dimensions {
			if i == 0 {
				return toolbar(c)
			}
			version := visible[i-1]
			row := a.availableVersionRow(sdk.Name, version, installed[version.Version], kit.PositionOf(i-1, len(visible)))
			return kit.Pad(kit.Inset{Left: gapXL, Right: gapXL, Bottom: hairline}, row)(c)
		},
	}
}

// detailHeader shows the name of the plugin, what it is and the buttons that
// act on the plugin as a whole.
func (a *App) detailHeader(sdk core.InstalledSdk, info core.PluginInfo, hasInfo bool, onBack func()) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		updating := a.model.Tasks.ActiveTask(vfox.OperationUpdatePlugin, sdk.Name, "")

		current := kit.Tag{Label: a.T("sdksNoCurrent")}
		if sdk.HasCurrent {
			current = kit.Tag{Label: a.T("sdksCurrent", "version", sdk.Current), Icon: "current", Emphasized: true}
		}
		tags := []kit.Widget{current.W()}
		if hasInfo && info.Version != "" {
			tags = append(tags, kit.Tag{Label: a.T("sdkPluginVersion", "version", info.Version)}.W())
		}
		text := []kit.Widget{
			kit.Line(sdk.Name, kit.Emph(kit.HeadlineMedium), cs.OnSurface),
			kit.Space(0, gapXS),
			kit.Wrap{Spacing: gapS, RunSpacing: gapXS}.Layout(tags...),
		}
		if hasInfo && info.Description != "" {
			text = append(text, kit.Space(0, gapS), kit.Label(info.Description, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Lines(3)))
		}
		textColumn := kit.Column(kit.FlexOpts{Cross: kit.CrossStart, Min: true}, widgetsToChildren(text)...)

		var update kit.Widget
		if updating != nil {
			update = kit.Pad(kit.All(gapXS), kit.LoadingIndicator{ID: "updating", Size: 40}.W())
		} else {
			update = kit.IconButton{
				ID: "update", Icon: "update", Tooltip: a.T("sdkUpdatePlugin"),
				OnClick: func() { a.run(a.model.Cli.UpdatePlugin(sdk.Name)) },
			}.W()
		}
		var menu []kit.MenuItem
		if hasInfo && info.Homepage != "" {
			homepage := info.Homepage
			menu = append(menu, kit.MenuItem{Label: a.T("actionOpenHomepage"), Icon: "openInNew", OnClick: func() { a.openURL(homepage) }})
		}
		menu = append(menu, kit.MenuItem{Label: a.T("sdkRemovePlugin"), Icon: "removePlugin", Danger: true, OnClick: func() { a.confirmRemovePlugin(sdk) }})
		more := kit.MenuButton{ID: "more", Icon: "more", Tooltip: a.T("sdkMoreActions"), Items: menu, AlignEnd: true}.W()
		actions := []kit.FlexChild{kit.Rigid(update), kit.Rigid(more)}

		var back kit.Widget
		if onBack != nil {
			back = kit.IconButton{ID: "back", Icon: "back", Tooltip: a.T("actionClose"), OnClick: onBack}.W()
		}
		var layoutWidget kit.Widget
		if c.Constraints.Max.X < c.DP(narrowHeader) {
			// Narrow panes put the buttons on their own row above the name.
			top := []kit.FlexChild{}
			if back != nil {
				top = append(top, kit.Rigid(back))
			}
			top = append(top, kit.Expanded(1, kit.Space(0, 0)))
			top = append(top, actions...)
			layoutWidget = kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true},
				kit.Rigid(kit.Row(kit.FlexOpts{Cross: kit.CrossCenter}, top...)),
				kit.Rigid(kit.Space(0, gapS)),
				kit.Rigid(kit.Row(kit.FlexOpts{Gap: gapL, Cross: kit.CrossStart},
					kit.Rigid(kit.Pad(kit.Inset{Left: gapS}, avatar(sdk.Name, 56))),
					kit.Expanded(1, textColumn))),
			)
		} else {
			row := []kit.FlexChild{}
			if back != nil {
				row = append(row, kit.Rigid(back), kit.Rigid(kit.Space(gapXS, 0)))
			} else {
				row = append(row, kit.Rigid(kit.Space(gapS, 0)))
			}
			row = append(row, kit.Rigid(avatar(sdk.Name, 72)), kit.Rigid(kit.Space(gapL, 0)), kit.Expanded(1, textColumn), kit.Rigid(kit.Space(gapS, 0)))
			row = append(row, actions...)
			layoutWidget = kit.Row(kit.FlexOpts{Cross: kit.CrossStart}, row...)
		}
		return kit.Pad(kit.Inset{Left: gapL, Top: gapXL, Right: gapL, Bottom: gapXL}, layoutWidget)(c)
	}
}

func widgetsToChildren(widgets []kit.Widget) []kit.FlexChild {
	children := make([]kit.FlexChild, len(widgets))
	for i, w := range widgets {
		children[i] = kit.Rigid(w)
	}
	return children
}

func (a *App) confirmRemovePlugin(sdk core.InstalledSdk) {
	a.confirm("remove-plugin", confirmation{
		Title: a.T("sdkRemoveTitle", "name", sdk.Name), Body: a.T("sdkRemoveBody", "count", len(sdk.Versions)),
		ConfirmLabel: a.T("sdkRemoveConfirm"), Icon: "removePlugin", Destructive: true,
		OnConfirm: func() {
			a.run(a.model.Cli.RemovePlugin(sdk.Name))
			a.showSdk("")
		},
	})
}

// installedVersionRow is an installed version, with the buttons that use it,
// copy its path and uninstall it.
func (a *App) installedVersionRow(sdk core.InstalledSdk, version string, position kit.Position) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		c = c.Scope("iv:" + version)
		current := sdk.HasCurrent && sdk.Current == version
		task := a.model.Tasks.ActiveTask("", sdk.Name, version)
		useGlobally := func() { a.run(a.model.Cli.Use(sdk.Name, version, core.UseScopeGlobal, "")) }
		var actions kit.Widget
		if task != nil {
			actions = kit.LoadingIndicator{ID: "busy", Size: 40}.W()
		} else {
			var menu []kit.MenuItem
			if current {
				menu = append(menu,
					kit.MenuItem{Label: a.T("versionUseInProject"), Icon: "projectScope", OnClick: func() { a.askProjectDirectory(sdk.Name, version) }},
					kit.MenuItem{Label: a.T("versionUnuse"), Icon: "cancel", OnClick: func() { a.run(a.model.Cli.Unuse(sdk.Name, core.UseScopeGlobal, "")) }},
				)
			}
			menu = append(menu,
				kit.MenuItem{Label: a.T("versionCopyPath"), Icon: "copy", OnClick: func() { a.copyInstallPath(sdk.Name, version) }},
				kit.MenuItem{Label: a.T("versionUninstall"), Icon: "uninstall", Danger: true, OnClick: func() { a.confirmUninstall(sdk, version, current) }},
			)
			var primary kit.Widget
			if current {
				primary = kit.Tag{Label: a.T("versionInUse"), Icon: "current", Emphasized: true}.W()
			} else {
				primary = kit.SplitButton{
					ID: "use", Label: a.T("versionUse"), Icon: "check", MenuTooltip: a.T("versionUseOptions"), OnClick: useGlobally,
					Items: []kit.MenuItem{
						{Label: a.T("versionUseGlobally"), Icon: "globalScope", OnClick: useGlobally},
						{Label: a.T("versionUseInProject"), Icon: "projectScope", OnClick: func() { a.askProjectDirectory(sdk.Name, version) }},
					},
				}.W()
			}
			actions = kit.Row(kit.FlexOpts{Gap: gapXS, Cross: kit.CrossCenter, Min: true},
				kit.Rigid(primary),
				kit.Rigid(kit.MenuButton{ID: "more", Icon: "more", Tooltip: a.T("versionMoreActions", "version", version), Items: menu, AlignEnd: true}.W()),
			)
		}
		titleStyle := kit.Type(kit.BodyLarge)
		leadingIcon, leadingFill, leadingColor := "versions", float32(0), cs.OnSurfaceVariant
		if current {
			titleStyle = kit.Emph(kit.BodyLarge)
			leadingIcon, leadingFill, leadingColor = "current", 1, cs.Primary
		}
		return kit.ListItem{
			ID: "item", Title: version, TitleStyle: &titleStyle, Position: position, WrapTrailing: true, Trailing: actions,
			Leading: func(color.NRGBA) kit.Widget { return kit.Icon(leadingIcon, 24, leadingColor, leadingFill) },
		}.W()(c)
	}
}

// availableVersionRow is a version that can be installed.
func (a *App) availableVersionRow(sdk string, version core.AvailableVersion, installed bool, position kit.Position) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		c = c.Scope("av:" + version.Version)
		task := a.model.Tasks.ActiveTask(vfox.OperationInstall, sdk, version.Version)
		additions := make([]string, len(version.Additions))
		for i, addition := range version.Additions {
			additions[i] = addition.String()
		}
		var trailing kit.Widget
		switch {
		case task != nil:
			trailing = a.installProgress(task)
		case installed:
			trailing = kit.Button{ID: "installed", Kind: kit.TextButton, Icon: "check", Label: a.T("versionInstalled"), Disabled: true}.W()
		default:
			trailing = kit.Button{
				ID: "install", Kind: kit.TonalButton, Icon: "install", Label: a.T("versionInstall"),
				OnClick: func() { a.run(a.model.Cli.Install(sdk, version.Version)) },
			}.W()
		}
		padding := kit.Sym(gapL, gapS)
		return kit.ListItem{
			ID: "item", Position: position, WrapTrailing: true, Padding: &padding, Trailing: trailing,
			Name: version.Version, Subtitle: strings.Join(additions, " · "),
			TitleWidget: func(foreground color.NRGBA) kit.Widget {
				parts := []kit.Widget{kit.Label(version.Version, kit.Type(kit.BodyLarge), foreground)}
				if version.HasNote {
					parts = append(parts, kit.Tag{Label: version.Note, Emphasized: true}.W())
				}
				return kit.Wrap{Spacing: gapS, CrossCenter: true}.Layout(parts...)
			},
		}.W()(c)
	}
}

// installProgress shows how far an installation has come, with a button that
// cancels it.
func (a *App) installProgress(task *state.VfoxTask) kit.Widget {
	return kit.Sized(220, 0, func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		progress, hasProgress := task.Progress()
		status := a.T("taskStatusRunning")
		if task.Status() == state.TaskQueued {
			status = a.T("taskStatusQueued")
		}
		if hasProgress {
			status = a.T("taskDownloading", "percent", int(math.Round(progress*100)))
		}
		return kit.Row(kit.FlexOpts{Cross: kit.CrossCenter},
			kit.Expanded(1, kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true},
				kit.Rigid(kit.WavyProgress{ID: "progress", Value: progress, Indeterminate: !hasProgress, Label: a.taskTitle(task.Command)}.W()),
				kit.Rigid(kit.Space(0, gapXS)),
				kit.Rigid(kit.Label(status, kit.Type(kit.LabelSmall), cs.OnSurfaceVariant)),
			)),
			kit.Rigid(kit.IconButton{ID: "cancel", Icon: "cancel", Tooltip: a.T("actionCancel"), OnClick: func() { a.model.Tasks.Cancel(task) }}.W()),
		)(c)
	})
}

func (a *App) confirmUninstall(sdk core.InstalledSdk, version string, current bool) {
	body := a.T("versionUninstallBody")
	if current {
		body = a.T("versionUninstallCurrentBody")
	}
	a.confirm("uninstall", confirmation{
		Title: a.T("versionUninstallTitle", "name", sdk.Name, "version", version), Body: body,
		ConfirmLabel: a.T("versionUninstall"), Icon: "uninstall", Destructive: true,
		OnConfirm: func() { a.run(a.model.Cli.Uninstall(sdk.Name, version)) },
	})
}

// copyInstallPath puts the directory a version is installed in on the
// clipboard.
func (a *App) copyInstallPath(name, version string) {
	go func() {
		path, found, err := a.model.Cli.InstallPath(name, version)
		a.post(func() {
			switch {
			case err != nil:
				a.notify(describeError(err))
			case found:
				a.env.Services.Copy(path)
				a.notify(a.T("actionCopied"))
			}
		})
	}()
}

// isDirectory reports whether path is an existing directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
