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
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// settingsTab is the settings tab: this phone, the connection of other devices to
// the computer, the settings of the application, and vfox's configuration. The
// order is the web's. Each section reads its data from the service as it is drawn,
// and changes it through Do.
func (r *Root) settingsTab(k kit.Ctx, bar bool) {
	ui.Column(k.Context).Fill().Children(func() {
		if bar {
			kit.PageHeader(k, tabTitle(app.TabSettings), "", nil)
		}
		ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
			ui.Column(k.Context).Padding(0, 16, 24, 16).Gap(16).Children(func() {
				r.phoneCard(k)
				r.settingsConnectionSection(k)
				r.settingsAppSection(k)
				r.settingsConfigSection(k)
			})
		})
	})
}

// settingsDraft is the text of a field of the settings. It follows the value that the
// computer has, and keeps what the user typed while that value does not change, so
// that a refresh does not overwrite an edit in progress. It is kept on the element of
// its section with ui.Local.
type settingsDraft struct {
	text string
	last string
	set  bool
}

// follow sets the text to value when value is new: the first one that the computer
// gave, or one that differs from the last.
func (d *settingsDraft) follow(value string) {
	if !d.set || value != d.last {
		d.text, d.last, d.set = value, value, true
	}
}

// settingsCopy puts text on the clipboard. It is called from a click, which is
// outside of a frame, so it uses the clipboard of mygo, as the desktop does.
func settingsCopy(text string) {
	mygo.Clipboard.WriteText(text)
}
