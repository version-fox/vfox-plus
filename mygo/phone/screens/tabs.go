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
	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// tabPage builds the page of a tab. The page of each tab is in the file of its
// group (see the comment at the top of each file).
func (r *Root) tabPage(k kit.Ctx, tab app.Tab) {
	switch tab {
	case app.TabHome:
		r.homeTab(k)
	case app.TabSDKs:
		r.sdksTab(k)
	case app.TabPlugins:
		r.pluginsTab(k)
	case app.TabTasks:
		r.tasksTab(k)
	case app.TabSettings:
		r.settingsTab(k)
	}
}

// tabTitle is the title of a tab, which is also its name in the messages.
func tabTitle(tab app.Tab) string {
	return i18n.Text(tab.String())
}

// destinations returns the destinations of the bar of the tabs, in their order.
func destinations() []kit.NavDestination {
	tabs := app.Tabs()
	out := make([]kit.NavDestination, 0, len(tabs))
	for _, tab := range tabs {
		out = append(out, kit.NavDestination{Icon: tab.String(), Label: tabTitle(tab)})
	}
	return out
}

// tabDestinations returns the destinations of the bar, with the badge of the tasks
// tab: the tasks that wait or run on the computer, as the web's badge counts them.
// The window reads the tasks on every frame (see Root.sync), so the badge is current
// on any tab, and the bar reads them as well.
func (r *Root) tabDestinations() []kit.NavDestination {
	out := destinations()
	for i, tab := range app.Tabs() {
		if tab == app.TabTasks {
			tasks, _ := r.service.Tasks().Value()
			out[i].Badge = tasksBadge(tasks)
		}
	}
	return out
}
