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
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pageScreen builds a page above the tabs: the bar with its title and its way back,
// where the window draws the bar, and the body of the page under it.
func (r *Root) pageScreen(k kit.Ctx, page app.Page, bar bool) {
	ui.Column(k.Context).Fill().Children(func() {
		if bar {
			kit.PageHeader(k, pageTitle(page), "", func() {
				r.backButton(k)
			})
		}
		ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
			r.pageBody(k, page)
		})
	})
}

// pageBody builds what a page shows under its bar. Each kind of page is built in
// the file of its group.
func (r *Root) pageBody(k kit.Ctx, page app.Page) {
	switch page.Kind {
	case app.PageSDK:
		r.sdkPage(k, page)
	case app.PageTask:
		r.taskPage(k, page)
	case app.PageLog:
		r.logPage(k, page)
	case app.PagePair:
		r.pairPage(k)
	}
}

// backButton is the way back from the page shown, to the one before it.
func (r *Root) backButton(k kit.Ctx) {
	kit.IconButton(k, kit.IconButtonSpec{
		ID:      "back",
		Icon:    "back",
		Tooltip: i18n.Text("back"),
		OnClick: func() {
			r.app.Pop()
		},
	})
}

// pageTitle is the title of a page above the tabs: the name of an SDK for its
// page, and the title of the other kinds of page in the messages.
func pageTitle(page app.Page) string {
	switch page.Kind {
	case app.PageSDK:
		return page.Name
	case app.PageTask:
		return i18n.Text("tasksH")
	case app.PageLog:
		return i18n.Text("log")
	case app.PagePair:
		return i18n.Text("phonePair")
	}
	return ""
}
