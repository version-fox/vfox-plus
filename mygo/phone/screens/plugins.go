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

// The plugin tab: the registry of the plugins that vfox can add, with the search and
// the filter over it, the actions that add or update them, and the dialog that adds a
// plugin from its source. The registry's list is in plugins_list.go; the dialog is in
// plugins_add.go; the rules are in plugins_rules.go.

import (
	"sync/atomic"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pluginsState is what the plugin tab keeps from frame to frame: the search, the
// filter that is on, how many plugins the list shows, and the flag that a call which
// landed sets (see availThen).
type pluginsState struct {
	query  string
	filter string
	limit  int
	landed *atomic.Bool
}

// pluginsTab is the plugin tab. With the bar of the window it shows the title and the
// actions in its header; without it, which is when the system shows the title, it shows
// the actions above the list.
func (r *Root) pluginsTab(k kit.Ctx, bar bool) {
	column := ui.Column(k.Context).Fill()
	st := ui.Local(column, "pluginsState", func() pluginsState {
		return pluginsState{filter: pluginFilterAll, limit: availPageSize, landed: new(atomic.Bool)}
	})
	column.Children(func() {
		r.availFollowTasks(k, st.landed)
		actions := r.pluginsActions(k, st)
		if bar {
			kit.PageHeader(k, tabTitle(app.TabPlugins), "", nil, actions...)
		} else {
			ui.Row(k.Context).Padding(8, 16, 0, 16).Gap(8).Wrap().Justify(ui.End).Children(func() {
				for _, action := range actions {
					action()
				}
			})
		}
		ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
			r.pluginsBody(k, st)
		})
	})
}

// pluginsActions returns the actions of the tab: updating every plugin, refreshing
// the registry, and adding a plugin from its source.
func (r *Root) pluginsActions(k kit.Ctx, st *pluginsState) []func() {
	env := k.Env
	svc := r.Service()
	return []func(){
		func() {
			kit.Button(k, kit.ButtonSpec{
				ID:    "updateAll",
				Kind:  kit.TextButton,
				Label: i18n.Text("updateAll"),
				Icon:  "update",
				OnClick: func() {
					r.availStart(st.landed, remote.TaskRequest{Operation: remote.RequestUpdate})
				},
			})
		},
		func() {
			kit.IconButton(k, kit.IconButtonSpec{
				ID:      "refresh",
				Icon:    "refresh",
				Tooltip: i18n.Text("refresh"),
				OnClick: svc.Refresh,
			})
		},
		func() {
			kit.Button(k, kit.ButtonSpec{
				ID:    "addFromSource",
				Kind:  kit.TonalButton,
				Label: i18n.Text("addFromSource"),
				Icon:  "add",
				OnClick: func() {
					r.pluginsOpenAdd(env, st.landed)
				},
			})
		},
	}
}
