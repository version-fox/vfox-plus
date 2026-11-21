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

package screens

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// The tabs of the page of an SDK.
const (
	sdkInstalled = "installed"
	sdkAvailable = "available"
	sdkOptions   = "options"
)

// sdkPage is the page of an SDK, with its three tabs: the installed versions, the
// versions that can be installed, and the options of the plugin, which only a plugin
// with a profile has. This file holds the page and its header; the tabs are in
// page_sdk_installed.go, page_sdk_options.go and page_sdk_dialogs.go, and the
// available tab is availableTab, of the installable group. The name and the way back
// are the bar of the page (pageScreen in pages.go), which the window draws, or the
// system draws in its own stack of screens.
func (r *Root) sdkPage(k kit.Ctx, page app.Page) {
	column := ui.Column(k.Context).Fill()
	tab := ui.Local(column, "sdkTab", func() string { return sdkInstalled })
	column.Children(func() {
		profile := r.service.Profile(page.Name)
		options := sdkHasOptions(profile)
		if *tab == sdkOptions && !options {
			*tab = sdkInstalled
		}
		r.sdkHeader(k, page.Name)
		r.sdkTabs(k, page.Name, tab, options)
		ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
			switch *tab {
			case sdkAvailable:
				r.availableTab(k, page.Name)
			case sdkOptions:
				r.sdkOptionsTab(k, page.Name, profile)
			default:
				r.sdkInstalledTab(k, page.Name, tab)
			}
		})
	})
}

// sdkHasOptions reports whether the page has an options tab, which a plugin with a
// profile has. While the profile loads the tab is not there yet. When the computer
// could not answer, the tab is there, so that it can say so and ask again.
func sdkHasOptions(profile app.Resource[*remote.Profile]) bool {
	value, loaded := profile.Value()
	if loaded {
		return value != nil
	}
	return profile.Err() != nil
}

// sdkHeader is the top of the page: the version in use, the homepage of the plugin
// when it has one, and the menu, which holds the homepage and the removal of the
// plugin. The menu's removal asks first.
func (r *Root) sdkHeader(k kit.Ctx, name string) {
	cs := k.Colors()
	info, _ := r.service.Info(name).Value()
	inv, loaded := r.service.Inventory().Value()
	sdk, installed := sdkFindInstalled(inv.Installed.Value, name)
	services := k.Services()
	homepage := info.Homepage
	openHomepage := func() { r.sdkOpenLink(services, homepage) }
	var menu []kit.MenuItem
	if homepage != "" {
		menu = append(menu, kit.MenuItem{Label: i18n.Text("homepageOpen"), Icon: "openInNew", OnClick: openHomepage})
	}
	menu = append(menu, kit.MenuItem{
		Label:   i18n.Text("remove"),
		Icon:    "removePlugin",
		Danger:  true,
		OnClick: func() { r.sdkConfirmRemove(name) },
	})
	ui.Row(k.Context).Padding(12, 16, 4, 16).AlignItems(ui.Center).Gap(12).Children(func() {
		sdkAvatar(k, name, 44)
		ui.Column(k.Context).Grow(1).Shrink(1).MinWidth(0).Gap(4).Children(func() {
			if !loaded {
				return
			}
			if installed && sdk.HasCurrent {
				kit.Label(k, i18n.Text("active"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
				// A row keeps the tag as wide as its text; a column would stretch it.
				ui.Row(k.Context).Children(func() {
					kit.Tag(k, kit.TagSpec{Label: sdk.Current, Emphasized: true})
				})
				return
			}
			kit.Label(k, i18n.Text("activeEmpty"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Breakable())
		})
		if homepage != "" {
			kit.IconButton(k, kit.IconButtonSpec{
				ID:      "sdkHomepage",
				Icon:    "openInNew",
				Tooltip: i18n.Text("homepage"),
				OnClick: openHomepage,
			})
		}
		kit.IconButton(k, kit.IconButtonSpec{
			ID:      "sdkMenu",
			Icon:    "more",
			Tooltip: i18n.Text("actions"),
			Menu:    menu,
		})
	})
}

// sdkTabs is the choice of the tab. The installed tab counts the versions that are
// installed, once the computer has answered; the options tab is there when the plugin
// has options.
func (r *Root) sdkTabs(k kit.Ctx, name string, tab *string, options bool) {
	label := i18n.Text("installed")
	if inv, loaded := r.service.Inventory().Value(); loaded {
		sdk, _ := sdkFindInstalled(inv.Installed.Value, name)
		label += " (" + strconv.Itoa(len(sdk.Versions)) + ")"
	}
	segments := []kit.Segment{
		{Value: sdkInstalled, Label: label, Icon: "storage"},
		{Value: sdkAvailable, Label: i18n.Text("availableTab"), Icon: "install"},
	}
	if options {
		segments = append(segments, kit.Segment{Value: sdkOptions, Label: i18n.Text("optionsTab"), Icon: "options"})
	}
	width, _ := k.Size()
	ui.Row(k.Context).Padding(4, 16, 8, 16).Children(func() {
		kit.ButtonGroup(k, kit.ButtonGroupSpec{
			ID:       "sdkTabs",
			Segments: segments,
			Selected: *tab,
			OnSelect: func(value string) { *tab = value },
			MaxWidth: width - 32,
			Collapse: true,
		})
	})
}
