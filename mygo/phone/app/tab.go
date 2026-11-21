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

package app

// Tab is one of the five destinations of the bottom bar, which are the pages of
// the web interface.
type Tab int

// The tabs, in the order of the bottom bar.
const (
	TabHome Tab = iota
	TabSDKs
	TabPlugins
	TabTasks
	TabSettings
)

// Tabs returns the tabs in the order of the bottom bar.
func Tabs() []Tab {
	return []Tab{TabHome, TabSDKs, TabPlugins, TabTasks, TabSettings}
}

// valid reports whether t is one of the tabs.
func (t Tab) valid() bool {
	return t >= TabHome && t <= TabSettings
}

// String names the tab: home, sdks, plugins, tasks or settings. The name is also
// the key of the tab's message and the meaning of its icon, as in the web
// interface, and it is "" for a value that is not a tab.
func (t Tab) String() string {
	switch t {
	case TabHome:
		return "home"
	case TabSDKs:
		return "sdks"
	case TabPlugins:
		return "plugins"
	case TabTasks:
		return "tasks"
	case TabSettings:
		return "settings"
	}
	return ""
}
