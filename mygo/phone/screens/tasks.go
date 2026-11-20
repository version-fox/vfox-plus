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
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// tasksTab is the tasks tab: what vfox has done and is doing, the newest first. A
// row opens its output under it, and the task page and the full log open from
// there. The list is a virtual list, so that only the rows on the screen are built.
func (r *Root) tasksTab(k kit.Ctx, bar bool) {
	column := ui.Column(k.Context).Fill()
	list := ui.Local(column, "list", func() ui.ListState { return ui.ListState{} })
	res := r.service.Tasks()
	values, loaded := res.Value()
	tasks, cut := newestTasks(values)
	column.Children(func() {
		if bar {
			kit.PageHeader(k, i18n.Text("tasksH"), "", nil, func() { r.taskClearButton(k) })
		} else {
			ui.Row(k.Context).Justify(ui.End).Padding(8, 16, 0, 16).Children(func() { r.taskClearButton(k) })
		}
		switch {
		case !loaded && res.Phase() == app.Failed:
			ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
				kit.StatusMessage(k, kit.StatusSpec{
					Icon: "error", IsError: true, Title: i18n.Text("failed"),
					Body: noticeText(app.KindOf(res.Err())),
					Actions: []func(){func() {
						kit.Button(k, kit.ButtonSpec{ID: "refresh", Kind: kit.TonalButton, Label: i18n.Text("refresh"), Icon: "refresh", OnClick: r.service.Refresh})
					}},
				})
			})
		case !loaded:
			ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
				kit.LoadingMessage(k, "", i18n.Text("loading"))
			})
		case len(tasks) == 0:
			ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
				kit.StatusMessage(k, kit.StatusSpec{Icon: "tasks", Title: i18n.Text("noTasks")})
			})
		default:
			ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
				if cut > 0 {
					note := strings.NewReplacer("{count}", strconv.Itoa(len(tasks))).Replace(i18n.Text("taskListCapped"))
					kit.Label(k, note, kit.Type(kit.BodySmall), k.Colors().OnSurfaceVariant, kit.Lines(2)).Margin(0, 16, 8, 16)
				}
				list.Key = func(row int) any { return tasks[row].ID }
				ui.List(k.Context, list, len(tasks), func(i int) {
					r.taskRow(k, tasks[i], kit.PositionOf(i, len(tasks))).Margin(0, 16, 2, 16)
				}).Grow(1)
			})
		}
	})
}

// announceTasks shows a snack bar for each task that succeeded, failed or was
// cancelled since the last call with the same watch. The first call only records
// the statuses. The window calls it on every frame with its one watch (see
// Root.sync), so the message of a task that ends is shown once, on whatever tab is
// shown. It reads the tasks of the computer, which loads them when they are out of
// date, and it does nothing until they have loaded.
func (r *Root) announceTasks(k kit.Ctx, watch *taskWatch) {
	values, loaded := r.service.Tasks().Value()
	if !loaded {
		return
	}
	for _, message := range watch.observe(values) {
		k.Env.ShowSnackbar(kit.Snack{
			Text: message, Action: i18n.Text("view"),
			OnAction: func() { r.app.SelectTab(app.TabTasks) },
		})
	}
}

// taskClearButton forgets the finished tasks of the computer.
func (r *Root) taskClearButton(k kit.Ctx) {
	kit.Button(k, kit.ButtonSpec{
		ID: "clear", Kind: kit.TextButton, Icon: "succeeded", Label: i18n.Text("clear"),
		OnClick: r.taskClearFinished,
	})
}
