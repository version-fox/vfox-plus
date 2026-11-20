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
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// logPage is the complete output of a task, as the computer kept it. A long log is
// shown by its newest lines (see logPageLines and logPageBytes), and the page says
// how many lines it left out; the copy button copies what is shown. A task whose
// complete output the computer did not keep says so.
func (r *Root) logPage(k kit.Ctx, page app.Page) {
	cs := k.Colors()
	id, err := strconv.Atoi(page.Name)
	if err != nil {
		kit.StatusMessage(k, kit.StatusSpec{Icon: "error", IsError: true, Title: i18n.Text("errNotFound")})
		return
	}
	res := r.service.TaskLog(id)
	text, ok := res.Value()
	switch {
	case ok:
		tail, total, shown := logTail(text, logPageLines, logPageBytes)
		column := ui.Column(k.Context).Fill().Gap(8)
		scroll := ui.Local(column, "logScroll", func() ui.ScrollState { return ui.ScrollState{} })
		column.Children(func() {
			ui.Row(k.Context).Gap(8).Padding(8, 16, 0, 16).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				if shown < total {
					note := strings.NewReplacer("{shown}", strconv.Itoa(shown), "{total}", strconv.Itoa(total)).Replace(i18n.Text("taskLogTail"))
					kit.Label(k, note, kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Lines(2), kit.Breakable()).Grow(1).Shrink(1)
				} else {
					ui.Spacer(k.Context).Grow(1)
				}
				kit.Button(k, kit.ButtonSpec{
					ID: "copy", Kind: kit.TonalButton, Label: i18n.Text("taskCopyLog"), Icon: "copy",
					OnClick: func() {
						taskCopy(tail)
						k.Env.ShowSnackbar(kit.Snack{Text: i18n.Text("copied")})
					},
				})
			})
			if tail == "" {
				ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
					kit.StatusMessage(k, kit.StatusSpec{Icon: "docs", Title: i18n.Text("taskNoOutput")})
				})
				return
			}
			// The log follows its end, unless the user has scrolled up from it.
			if scroll.Y >= scroll.MaxY {
				scroll.Y = math.MaxFloat32
			}
			ui.Scroll(k.Context).TrackScroll(scroll).Grow(1).Shrink(1).Padding(0, 16, 16, 16).Children(func() {
				kit.Label(k, tail, kit.Type(kit.BodySmall), cs.OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable())
			})
		})
	case res.Phase() == app.Failed:
		if app.KindOf(res.Err()) == app.KindNotFound {
			ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
				kit.StatusMessage(k, kit.StatusSpec{Icon: "docs", Title: i18n.Text("taskLogNone")})
			})
			return
		}
		ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
			kit.StatusMessage(k, kit.StatusSpec{
				Icon: "error", IsError: true, Title: i18n.Text("failed"),
				Body: noticeText(app.KindOf(res.Err())),
			})
		})
	default:
		ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
			kit.LoadingMessage(k, "", i18n.Text("loading"))
		})
	}
}
