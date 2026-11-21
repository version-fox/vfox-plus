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

// The installable versions of an SDK, the tab of the SDK page that the web calls
// "Available". The rules of what the list keeps are in sdk_available_rules.go; the
// choice of the search source is in sdk_available_choice.go; the other ways to add a
// version and their dialogs are in sdk_available_install.go.

import (
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// availState is what the installable versions keep from frame to frame: the search,
// the filters that are on, how many versions the list shows, and the flag that a call
// which landed sets (see availThen).
type availState struct {
	query  string
	limit  int
	chosen availChosen
	// search is the choice of search that the filters and the limit were set for.
	search string
	landed *atomic.Bool
}

// availableTab is the tab of the versions that an SDK can install, which the SDK page
// shows as its second tab. The name is the plugin's.
func (r *Root) availableTab(k kit.Ctx, name string) {
	column := ui.Column(k.Context).Fill()
	st := ui.Local(column, "availState", func() availState {
		return availState{limit: availPageSize, chosen: availChosen{}, landed: new(atomic.Bool)}
	})
	column.Children(func() {
		r.availFollowTasks(k, st.landed)
		r.availBody(k, name, st)
	})
}

// availBody builds the tab: the choice of the search and the other ways to add a
// version, above, and the versions below. The versions wait while the plugin's profile
// and the versions load, and show their failure, with a retry, when they fail.
func (r *Root) availBody(k kit.Ctx, name string, st *availState) {
	svc := r.Service()
	profile := svc.Profile(name)
	prof, loaded := profile.Value()
	inventory, _ := svc.Inventory().Value()
	installed := availInstalled(inventory, name)
	var search *remote.ProfileSearch
	if prof != nil {
		search = prof.Search
	}

	ui.Row(k.Context).Padding(8, 16, 0, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		if search != nil {
			r.availChoiceButton(k, name, prof)
		}
		kit.Spacer(k)
		r.availMoreButton(k, name, prof, st.landed)
	})
	if loaded {
		// A new choice of search lists other versions, which the filters of the last
		// ones may match none of: as the web does, the filters and the page are cleared.
		if value := availSearchValue(prof); value != st.search {
			st.search = value
			st.chosen = availChosen{}
			st.limit = availPageSize
		}
	}
	ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
		if !loaded {
			availStatus(k, svc, profile, i18n.Text("loading"))
			return
		}
		versions := svc.Versions(name, availSearchValue(prof))
		list, ok := versions.Value()
		if !ok {
			availStatus(k, svc, versions, i18n.Text("fetching"))
			return
		}
		r.availList(k, name, st, search, prof, list, installed)
	})
}

// availStatus shows what a read that has no value yet shows: its failure, or the label
// of its loading.
func availStatus[T any](k kit.Ctx, svc *app.Service, res app.Resource[T], loading string) {
	if res.Phase() == app.Failed {
		availFailed(k, svc, noticeText(app.KindOf(res.Err())))
		return
	}
	kit.LoadingMessage(k, loading, i18n.Text("loading"))
}

// availFailed shows a failure of a read, with the text of its cause, and a retry that
// reads everything again.
func availFailed(k kit.Ctx, svc *app.Service, cause string) {
	kit.StatusMessage(k, kit.StatusSpec{
		Icon:    "error",
		IsError: true,
		Title:   i18n.Text("failed"),
		Body:    cause,
		Actions: []func(){func() {
			kit.Button(k, kit.ButtonSpec{
				ID: "retry", Kind: kit.TonalButton, Label: i18n.Text("retry"), Icon: "retry", OnClick: svc.Refresh,
			})
		}},
	})
}

// availList builds the list of the versions: the search, the filters and the count
// above it, and below them the first versions of the list that match, which the "show
// more" adds to. A version that the computer has installed is marked, and has no
// install action.
func (r *Root) availList(k kit.Ctx, name string, st *availState, search *remote.ProfileSearch, prof *remote.Profile, versions []remote.AvailableVersion, installed map[string]bool) {
	cs := k.Colors()
	current, _ := availChoiceOf(search, availSearchValue(prof))
	showSource := current.Everything

	facets := availFacets(prof, versions)
	rows := availRows(versions, strings.ToLower(st.query), st.chosen)
	ui.Column(k.Context).Padding(0, 16, 8, 16).Children(func() {
		kit.SearchBar(k, kit.SearchBarSpec{
			ID:           "search",
			Hint:         i18n.Text("search"),
			Text:         &st.query,
			ClearTooltip: i18n.Text("clearSearch"),
			OnChange:     func(string) { st.limit = availPageSize },
		})
	})
	if len(facets) > 0 {
		ui.Row(k.Context).Wrap().Gap(8).GapY(8).Padding(0, 16, 8, 16).Children(func() {
			for _, facet := range facets {
				for _, value := range facet.Values {
					kit.FilterChip(k, kit.ChipSpec{
						ID:       facet.ID + ":" + value,
						Label:    value,
						Selected: st.chosen[facet.ID][value],
						OnClick: func() {
							availToggle(st.chosen, facet.ID, value)
							st.limit = availPageSize
						},
					})
				}
			}
		})
	}
	ui.Column(k.Context).Padding(0, 16, 8, 16).Children(func() {
		kit.Label(k, availCount("nVersions", len(rows)), kit.Type(kit.LabelLarge), cs.OnSurfaceVariant)
	})
	if len(rows) == 0 {
		kit.StatusMessage(k, kit.StatusSpec{Icon: "notFound", Title: i18n.Text("noMatch")})
		return
	}
	shown := rows
	if len(shown) > st.limit {
		shown = shown[:st.limit]
	}
	ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
		ui.Column(k.Context).Padding(0, 16, 16, 16).Gap(12).Children(func() {
			items := make([]kit.ListItemSpec, 0, len(shown))
			for _, version := range shown {
				items = append(items, r.availRowItem(k, name, version, showSource, installed[version.Version], st.landed))
			}
			kit.ListGroup(k, items...)
			if rest := len(rows) - len(shown); rest > 0 {
				kit.Button(k, kit.ButtonSpec{
					ID:      "showMore",
					Kind:    kit.TonalButton,
					Label:   availCount("showMore", rest),
					OnClick: func() { st.limit += availPageStep },
				})
			}
		})
	})
}

// availRowItem is the row of one version: the version, with the tags of its source,
// its note and its installation, and its additions below. A version that is not
// installed has the install button, and the menu of installing and using it.
func (r *Root) availRowItem(k kit.Ctx, name string, version remote.AvailableVersion, showSource, installed bool, landed *atomic.Bool) kit.ListItemSpec {
	item := kit.ListItemSpec{
		ID: version.Version,
		TitleWidget: func(foreground ui.Color) {
			ui.Row(k.Context).Wrap().Gap(8).GapY(4).AlignItems(ui.Center).Children(func() {
				kit.Label(k, version.Version, kit.Type(kit.BodyLarge), foreground, kit.Monospace(), kit.Breakable())
				if source := availSourceName(version); showSource && source != "" {
					kit.Tag(k, kit.TagSpec{Label: source})
				}
				if version.HasNote {
					kit.Tag(k, kit.TagSpec{Label: version.Note})
				}
				if installed {
					kit.Tag(k, kit.TagSpec{Label: i18n.Text("installed"), Icon: "check"})
				}
			})
		},
		Subtitle: availAdditions(version.Additions),
	}
	if installed {
		return item
	}
	item.Trailing = func() {
		kit.SplitButton(k, kit.SplitButtonSpec{
			ID:    "install",
			Label: i18n.Text("install"),
			Icon:  "install",
			OnClick: func() {
				r.availStart(landed, remote.TaskRequest{Operation: remote.RequestInstall, Sdk: name, Version: version.Version})
			},
			MenuTooltip: i18n.Text("actions"),
			Items: []kit.MenuItem{{
				Label: i18n.Text("installUse"),
				Icon:  "check",
				OnClick: func() {
					r.availStart(landed, remote.TaskRequest{Operation: remote.RequestInstall, Sdk: name, Version: version.Version, Use: true})
				},
			}},
		})
	}
	return item
}
