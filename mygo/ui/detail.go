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
	"math"
	"os"
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// The tabs of the details of a plugin.
const (
	tabInstalled = "installed"
	tabAvailable = "available"
)

// narrowHeader is the width in dp below which the header of the details puts its
// buttons on a row of their own.
const narrowHeader = 520

// wrapWidth is the width in dp of a list item below which the buttons at its end
// move under its text.
const wrapWidth = 560

// detailState is the state of the details of one plugin.
type detailState struct {
	name       string
	controller *state.SdkDetailController
	tab        string
	stops      []func()
	// installed and available keep the place of the two lists.
	installed, available mui.ListState
}

func (d *detailState) dispose() {
	for _, stop := range d.stops {
		stop()
	}
	d.controller.Dispose()
}

// detailFor returns the state of the details of the plugin name. It is created
// when the user selects the plugin, and replaced when they select another.
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
	d.stops = append(d.stops,
		controller.AddListener(a.invalidate),
		controller.Info.AddListener(a.invalidate),
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
	a.invalidate()
}

// detailBody is what the details show under the tabs: a number of items, built
// lazily.
type detailBody struct {
	count int
	key   func(i int) any
	item  func(i int)
}

func single(key string, build func()) detailBody {
	return detailBody{
		count: 1,
		key:   func(int) any { return key },
		item:  func(int) { build() },
	}
}

// detailView shows the versions of an installed plugin, and installs more.
// onBack, when it is not nil, adds a button that leaves the details.
func (a *App) detailView(k kit.Ctx, name string, onBack func()) {
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
	tabs := func() {
		kit.ButtonGroup(k, kit.ButtonGroupSpec{
			ID: "tabs", Selected: d.tab, OnSelect: func(tab string) { a.showTab(d, tab) },
			Segments: []kit.Segment{
				{Value: tabInstalled, Label: fmt.Sprintf("%s (%d)", a.T("sdkTabInstalled"), len(sdk.Versions)), Icon: "versions"},
				{Value: tabAvailable, Label: a.T("sdkTabAvailable"), Icon: "install"},
			},
		}).AlignSelf(mui.Start).Margin(0, gapXL, gapL, gapXL)
	}
	head := []func(){
		func() { a.detailHeader(k, sdk, info, hasInfo, onBack) },
		tabs,
	}
	var body detailBody
	list := &d.installed
	if d.tab == tabInstalled {
		body = a.installedBody(k, d, sdk)
	} else {
		body = a.availableBody(k, d, sdk, installed)
		list = &d.available
	}
	list.Key = func(i int) any {
		switch {
		case i < len(head):
			return i
		case i-len(head) < body.count:
			return body.key(i - len(head))
		default:
			return "end"
		}
	}
	mui.List(k.Context, list, len(head)+body.count+1, func(i int) {
		switch {
		case i < len(head):
			head[i]()
		case i-len(head) < body.count:
			body.item(i - len(head))
		default:
			kit.Gap(k, 0, gapXL)
		}
	}).Grow(1)
}

func (a *App) installedBody(k kit.Ctx, d *detailState, sdk core.InstalledSdk) detailBody {
	if len(sdk.Versions) == 0 {
		return single("empty", func() {
			kit.StatusMessage(k, kit.StatusSpec{
				Icon: "install", Shape: shapes.ShapePill, Title: a.T("sdkInstalledEmpty", "name", sdk.Name),
				Actions: []func(){func() {
					kit.Button(k, kit.ButtonSpec{
						ID: "browse", Kind: kit.FilledButton, Icon: "install", Label: a.T("sdkBrowseVersions"),
						OnClick: func() { a.showTab(d, tabAvailable) },
					})
				}},
			}).Height(360)
		})
	}
	return detailBody{
		count: len(sdk.Versions),
		key:   func(i int) any { return "iv:" + sdk.Versions[i] },
		item: func(i int) {
			a.installedVersionRow(k, sdk, sdk.Versions[i], kit.PositionOf(i, len(sdk.Versions))).
				Margin(0, gapXL, hairline, gapXL)
		},
	}
}

func (a *App) availableBody(k kit.Ctx, d *detailState, sdk core.InstalledSdk, installed map[string]bool) detailBody {
	controller := d.controller
	if !controller.Versions.HasValue() {
		if err := controller.Versions.Error(); err != nil {
			return single("status", func() {
				a.errorView(k, a.T("availableError", "name", sdk.Name), err, func() { controller.Versions.Load() })
			})
		}
		return single("status", func() { a.loadingView(k, a.T("availableLoading")) })
	}
	visible := controller.VisibleVersions(installed)
	cs := k.Colors()
	toolbar := func() {
		mui.Row(k.Context).Wrap().Gap(gapM).GapY(gapS).Margin(0, gapXL, gapM, gapXL).Children(func() {
			kit.TextField(k, kit.TextFieldSpec{
				ID: "filter", Hint: a.T("availableSearchHint"), LeadingIcon: "search", Dense: true,
				Value: controller.Query(), OnChange: controller.SetQuery,
			}).Width(280)
			kit.ButtonGroup(k, kit.ButtonGroupSpec{
				ID: "filters", Selected: string(controller.Filter()),
				OnSelect: func(value string) { controller.SetFilter(state.VersionFilter(value)) },
				Segments: []kit.Segment{
					{Value: string(state.VersionFilterAll), Label: a.T("availableFilterAll")},
					{Value: string(state.VersionFilterWithNote), Label: a.T("availableFilterTagged")},
					{Value: string(state.VersionFilterInstalled), Label: a.T("availableFilterInstalled")},
				},
			})
			kit.Label(k, a.T("availableCount", "count", len(visible)), kit.Type(kit.LabelLarge), cs.OnSurfaceVariant)
		})
	}
	if len(visible) == 0 {
		return detailBody{
			count: 2,
			key:   func(i int) any { return [2]string{"toolbar", "empty"}[i] },
			item: func(i int) {
				if i == 0 {
					toolbar()
					return
				}
				kit.StatusMessage(k, kit.StatusSpec{Icon: "notFound", Title: a.T("availableEmpty")}).Height(320)
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
		item: func(i int) {
			if i == 0 {
				toolbar()
				return
			}
			version := visible[i-1]
			a.availableVersionRow(k, sdk.Name, version, installed[version.Version], kit.PositionOf(i-1, len(visible))).
				Margin(0, gapXL, hairline, gapXL)
		},
	}
}

// detailHeader shows the name of the plugin, what it is and the buttons that act
// on the plugin as a whole.
func (a *App) detailHeader(k kit.Ctx, sdk core.InstalledSdk, info core.PluginInfo, hasInfo bool, onBack func()) {
	cs := k.Colors()
	updating := a.model.Tasks.ActiveTask(vfox.OperationUpdatePlugin, sdk.Name, "")

	textColumn := func() {
		mui.Column(k.Context).Grow(1).Shrink(1).AlignItems(mui.Start).Children(func() {
			kit.Line(k, sdk.Name, kit.Emph(kit.HeadlineMedium), cs.OnSurface)
			mui.Row(k.Context).Wrap().Gap(gapS).GapY(gapXS).Margin(gapXS, 0, 0, 0).Children(func() {
				if sdk.HasCurrent {
					kit.Tag(k, kit.TagSpec{Label: a.T("sdksCurrent", "version", sdk.Current), Icon: "current", Emphasized: true})
				} else {
					kit.Tag(k, kit.TagSpec{Label: a.T("sdksNoCurrent")})
				}
				if hasInfo && info.Version != "" {
					kit.Tag(k, kit.TagSpec{Label: a.T("sdkPluginVersion", "version", info.Version)})
				}
			})
			if hasInfo && info.Description != "" {
				kit.Label(k, info.Description, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Lines(3)).Margin(gapS, 0, 0, 0)
			}
		})
	}
	actions := func() {
		if updating != nil {
			kit.LoadingIndicator(k, kit.LoadingSpec{ID: "updating", Size: 40}).Margin(gapXS)
		} else {
			kit.IconButton(k, kit.IconButtonSpec{
				ID: "update", Icon: "update", Tooltip: a.T("sdkUpdatePlugin"),
				OnClick: func() { a.run(a.model.Cli.UpdatePlugin(sdk.Name)) },
			})
		}
		var menu []kit.MenuItem
		if hasInfo && info.Homepage != "" {
			homepage := info.Homepage
			menu = append(menu, kit.MenuItem{Label: a.T("actionOpenHomepage"), Icon: "openInNew", OnClick: func() { a.openURL(homepage) }})
		}
		menu = append(menu, kit.MenuItem{
			Label: a.T("sdkRemovePlugin"), Icon: "removePlugin", Danger: true, OnClick: func() { a.confirmRemovePlugin(sdk) },
		})
		kit.IconButton(k, kit.IconButtonSpec{
			ID: "more", Icon: "more", Tooltip: a.T("sdkMoreActions"), Menu: menu, MenuAlignEnd: true,
		})
	}
	back := func() {
		kit.IconButton(k, kit.IconButtonSpec{ID: "back", Icon: "back", Tooltip: a.T("actionClose"), OnClick: onBack})
	}
	if a.pages.detailWidth < narrowHeader {
		// Narrow panes put the buttons on their own row above the name.
		mui.Column(k.Context).Padding(gapXL, gapL, gapXL, gapL).Gap(gapS).Children(func() {
			mui.Row(k.Context).Children(func() {
				if onBack != nil {
					back()
				}
				mui.Spacer(k.Context)
				actions()
			})
			mui.Row(k.Context).Gap(gapL).AlignItems(mui.Start).Children(func() {
				avatar(k, sdk.Name, 56)
				textColumn()
			})
		})
		return
	}
	mui.Row(k.Context).Padding(gapXL, gapL, gapXL, gapL).AlignItems(mui.Start).Children(func() {
		if onBack != nil {
			back()
			kit.Gap(k, gapXS, 0)
		} else {
			kit.Gap(k, gapS, 0)
		}
		avatar(k, sdk.Name, 72)
		kit.Gap(k, gapL, 0)
		textColumn()
		kit.Gap(k, gapS, 0)
		actions()
	})
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

// itemWraps tells whether the buttons at the end of the items of the details
// move under their text.
func (a *App) itemWraps() bool { return a.pages.detailWidth-2*gapXL < wrapWidth }

// installedVersionRow is an installed version, with the buttons that use it,
// copy its path and uninstall it.
func (a *App) installedVersionRow(k kit.Ctx, sdk core.InstalledSdk, version string, position kit.Position) mui.Element {
	cs := k.Colors()
	current := sdk.HasCurrent && sdk.Current == version
	task := a.model.Tasks.ActiveTask("", sdk.Name, version)
	useGlobally := func() { a.run(a.model.Cli.Use(sdk.Name, version, core.UseScopeGlobal, "")) }
	trailing := func() {
		if task != nil {
			kit.LoadingIndicator(k, kit.LoadingSpec{ID: "busy", Size: 40})
			return
		}
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
		mui.Row(k.Context).Gap(gapXS).Children(func() {
			if current {
				kit.Tag(k, kit.TagSpec{Label: a.T("versionInUse"), Icon: "current", Emphasized: true})
			} else {
				kit.SplitButton(k, kit.SplitButtonSpec{
					ID: "use", Label: a.T("versionUse"), Icon: "check", MenuTooltip: a.T("versionUseOptions"), OnClick: useGlobally,
					Items: []kit.MenuItem{
						{Label: a.T("versionUseGlobally"), Icon: "globalScope", OnClick: useGlobally},
						{Label: a.T("versionUseInProject"), Icon: "projectScope", OnClick: func() { a.askProjectDirectory(sdk.Name, version) }},
					},
				})
			}
			kit.IconButton(k, kit.IconButtonSpec{
				ID: "more", Icon: "more", Tooltip: a.T("versionMoreActions", "version", version), Menu: menu, MenuAlignEnd: true,
			})
		})
	}
	titleStyle := kit.Type(kit.BodyLarge)
	leadingIcon, leadingFill, leadingColor := "versions", float32(0), cs.OnSurfaceVariant
	if current {
		titleStyle = kit.Emph(kit.BodyLarge)
		leadingIcon, leadingFill, leadingColor = "current", 1, cs.Primary
	}
	return kit.ListItem(k, kit.ListItemSpec{
		ID: "iv:" + version, Title: version, TitleStyle: &titleStyle, Position: position, Wrap: a.itemWraps(), Trailing: trailing,
		Leading: func(mui.Color) { kit.Icon(k, leadingIcon, 24, leadingColor, leadingFill) },
	})
}

// availableVersionRow is a version that can be installed.
func (a *App) availableVersionRow(k kit.Ctx, sdk string, version core.AvailableVersion, installed bool, position kit.Position) mui.Element {
	task := a.model.Tasks.ActiveTask(vfox.OperationInstall, sdk, version.Version)
	additions := make([]string, len(version.Additions))
	for i, addition := range version.Additions {
		additions[i] = addition.String()
	}
	trailing := func() {
		switch {
		case task != nil:
			a.installProgress(k, task)
		case installed:
			kit.Button(k, kit.ButtonSpec{ID: "installed", Kind: kit.TextButton, Icon: "check", Label: a.T("versionInstalled"), Disabled: true})
		default:
			kit.Button(k, kit.ButtonSpec{
				ID: "install", Kind: kit.TonalButton, Icon: "install", Label: a.T("versionInstall"),
				OnClick: func() { a.run(a.model.Cli.Install(sdk, version.Version)) },
			})
		}
	}
	return kit.ListItem(k, kit.ListItemSpec{
		ID: "av:" + version.Version, Position: position, Wrap: a.itemWraps(), Padding: []float32{gapS, gapL}, Trailing: trailing,
		Name: version.Version, Subtitle: strings.Join(additions, " · "),
		TitleWidget: func(foreground mui.Color) {
			mui.Row(k.Context).Wrap().Gap(gapS).Children(func() {
				kit.Label(k, version.Version, kit.Type(kit.BodyLarge), foreground)
				if version.HasNote {
					kit.Tag(k, kit.TagSpec{Label: version.Note, Emphasized: true})
				}
			})
		},
	})
}

// installProgress shows how far an installation has come, with a button that
// cancels it.
func (a *App) installProgress(k kit.Ctx, task *state.VfoxTask) {
	cs := k.Colors()
	progress, hasProgress := task.Progress()
	status := a.T("taskStatusRunning")
	if task.Status() == state.TaskQueued {
		status = a.T("taskStatusQueued")
	}
	if hasProgress {
		status = a.T("taskDownloading", "percent", int(math.Round(progress*100)))
	}
	mui.Row(k.Context).Width(220).Shrink(0).Children(func() {
		mui.Column(k.Context).Grow(1).Shrink(1).Gap(gapXS).Children(func() {
			kit.WavyProgress(k, kit.WavySpec{ID: "progress", Value: progress, Indeterminate: !hasProgress, Label: a.taskTitle(task.Command)})
			kit.Label(k, status, kit.Type(kit.LabelSmall), cs.OnSurfaceVariant)
		})
		kit.IconButton(k, kit.IconButtonSpec{
			ID: "cancel", Icon: "cancel", Tooltip: a.T("actionCancel"), OnClick: func() { a.model.Tasks.Cancel(task) },
		})
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

// copyInstallPath puts the directory a version is installed in on the clipboard.
func (a *App) copyInstallPath(name, version string) {
	go func() {
		path, found, err := a.model.Cli.InstallPath(name, version)
		a.post(func() {
			switch {
			case err != nil:
				a.notify(describeError(err))
			case found:
				a.copy(path)
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
