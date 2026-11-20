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
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// taskCopy puts text on the clipboard. It is called from a click, which is outside
// of a frame, where the context of the window cannot write the clipboard; so it
// uses the clipboard of mygo, as the desktop does.
func taskCopy(text string) {
	mygo.Clipboard.WriteText(text)
}

// taskDetail is what a task ran and printed: the command, how many lines of its
// output the computer dropped, the newest lines of the output, and the actions of
// the task. The row of the list (inList) also opens the task page. The row and the
// page show the same detail, and differ only in how many lines of output they show.
func (r *Root) taskDetail(k kit.Ctx, id int, lines int, inList bool) {
	cs := k.Colors()
	column := ui.Column(k.Context).Gap(8)
	scroll := ui.Local(column, "output", func() ui.ScrollState { return ui.ScrollState{} })
	res := r.service.Task(id)
	task, ok := res.Value()
	column.Children(func() {
		if !ok {
			if res.Phase() == app.Failed {
				kit.Label(k, noticeText(app.KindOf(res.Err())), kit.Type(kit.BodyMedium), cs.Error, kit.Breakable())
			} else {
				kit.LoadingIndicator(k, kit.LoadingSpec{ID: "detail", Size: 24, Label: i18n.Text("loading")})
			}
			return
		}
		kit.Label(k, task.CommandLine, kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Monospace(), kit.Breakable(), kit.Selectable())
		if task.DroppedLines > 0 {
			dropped := i18n.Text("droppedHead") + strconv.Itoa(task.DroppedLines) + i18n.Text("droppedTail")
			kit.Label(k, dropped, kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Breakable())
		}
		text := strings.Join(task.Log, "\n")
		output, total, shown := logTail(text, lines, taskOutputBytes)
		if output == "" {
			kit.Label(k, i18n.Text("taskNoOutput"), kit.Type(kit.BodySmall), cs.OnSurfaceVariant)
		} else {
			if shown < total {
				note := strings.NewReplacer("{shown}", strconv.Itoa(shown), "{total}", strconv.Itoa(total)).Replace(i18n.Text("taskLogTail"))
				kit.Label(k, note, kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Breakable())
			}
			// The output follows its end, unless the user has scrolled up from it.
			if scroll.Y >= scroll.MaxY {
				scroll.Y = math.MaxFloat32
			}
			ui.Scroll(k.Context).TrackScroll(scroll).MaxHeight(260).Children(func() {
				kit.Label(k, output, kit.Type(kit.BodySmall), cs.OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable())
			})
		}
		ui.Row(k.Context).Gap(8).Wrap().Children(func() {
			if taskActive(task.Status) {
				kit.Button(k, kit.ButtonSpec{
					ID: "cancel", Kind: kit.TonalButton, Label: i18n.Text("cancel"), Icon: "cancel",
					OnClick: func() { r.taskCancel(id) },
				})
			}
			if taskRetryable(task.Status) {
				kit.Button(k, kit.ButtonSpec{
					ID: "retry", Kind: kit.TonalButton, Label: i18n.Text("retry"), Icon: "retry",
					OnClick: func() { r.taskRetry(id) },
				})
			}
			if task.HasFullLog {
				kit.Button(k, kit.ButtonSpec{
					ID: "full-log", Kind: kit.TextButton, Label: i18n.Text("taskFullLog"), Icon: "docs",
					OnClick: func() { r.app.Push(app.Page{Kind: app.PageLog, Name: strconv.Itoa(id)}) },
				})
			}
			if inList {
				kit.Button(k, kit.ButtonSpec{
					ID: "details", Kind: kit.TextButton, Label: i18n.Text("taskDetails"), Icon: "openInNew",
					OnClick: func() { r.app.Push(app.Page{Kind: app.PageTask, Name: strconv.Itoa(id)}) },
				})
			}
			kit.IconButton(k, kit.IconButtonSpec{
				ID: "copy", Icon: "copy", Tooltip: i18n.Text("taskCopyLog"),
				OnClick: func() {
					taskCopy(text)
					k.Env.ShowSnackbar(kit.Snack{Text: i18n.Text("copied")})
				},
			})
		})
	})
}

// taskCancel cancels a task, which the computer runs or waits to run.
func (r *Root) taskCancel(id int) {
	r.Do(func(ctx context.Context) error {
		return r.service.CancelTask(ctx, id)
	}, "")
}

// taskRetry runs a task that failed or was cancelled again.
func (r *Root) taskRetry(id int) {
	r.Do(func(ctx context.Context) error {
		return r.service.RetryTask(ctx, id)
	}, "")
}

// taskClearFinished forgets the tasks that ended, on the computer.
func (r *Root) taskClearFinished() {
	r.Do(func(ctx context.Context) error {
		return r.service.ClearFinished(ctx)
	}, "")
}
