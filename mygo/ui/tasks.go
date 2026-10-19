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

package ui

import (
	"math"
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// tasksPage lists what vfox did and is doing.
func (a *App) tasksPage(k kit.Ctx) {
	kit.Pane(k, func() {
		tasks := a.model.Tasks.Tasks()
		header := func() {
			kit.PageHeader(k, a.T("navTasks"), "", nil, func() {
				kit.Button(k, kit.ButtonSpec{
					ID: "clear", Kind: kit.TextButton, Icon: "succeeded", Label: a.T("tasksClearFinished"),
					Disabled: !a.model.Tasks.HasFinished(), OnClick: a.model.Tasks.ClearFinished,
				})
			})
		}
		if len(tasks) == 0 {
			headerAndBody(k, header, func() {
				kit.StatusMessage(k, kit.StatusSpec{
					Icon: "tasks", Shape: shapes.ShapeSunny, Title: a.T("tasksEmptyTitle"), Body: a.T("tasksEmptyBody"),
				})
			})
			return
		}
		list := &a.pages.tasksList
		list.Key = func(i int) any {
			if i == 0 {
				return "header"
			}
			return tasks[i-1].ID
		}
		mui.List(k.Context, list, len(tasks)+1, func(i int) {
			if i == 0 {
				header()
				return
			}
			a.taskRow(k, tasks[i-1], kit.PositionOf(i-1, len(tasks))).Margin(0, gapXL, hairline, gapXL)
		}).Grow(1)
	}).Grow(1)
}

// taskRow is a task: what it does, how far it is, and its output.
func (a *App) taskRow(k kit.Ctx, task *state.VfoxTask, position kit.Position) mui.Element {
	cs := k.Colors()
	column := mui.Column(k.Key("task:" + itoa(task.ID)))
	expanded := mui.Local(column, "expanded", func() bool { return false })
	logScroll := mui.Local(column, "log", func() mui.ScrollState { return mui.ScrollState{} })
	status := task.Status()
	progress, hasProgress := task.Progress()

	leading := func() {
		switch status {
		case state.TaskQueued:
			kit.Icon(k, "queued", 24, cs.OnSurfaceVariant, 0)
		case state.TaskRunning:
			kit.LoadingIndicator(k, kit.LoadingSpec{ID: "running", Size: 32})
		case state.TaskSucceeded:
			kit.Icon(k, "succeeded", 24, cs.Primary, 1)
		case state.TaskFailed:
			kit.Icon(k, "error", 24, cs.Error, 1)
		default:
			kit.Icon(k, "cancel", 24, cs.OnSurfaceVariant, 0)
		}
	}
	var statusText string
	switch status {
	case state.TaskQueued:
		statusText = a.T("taskStatusQueued")
	case state.TaskRunning:
		statusText = a.T("taskStatusRunning")
		if hasProgress {
			statusText = a.T("taskDownloading", "percent", int(math.Round(progress*100)))
		}
	case state.TaskSucceeded:
		statusText = a.T("taskStatusSucceeded")
	case state.TaskFailed:
		message, _ := task.ErrorMessage()
		statusText = a.T("taskStatusFailed") + " · " + message
	default:
		statusText = a.T("taskStatusCancelled")
	}
	statusColor := cs.OnSurfaceVariant
	if status == state.TaskFailed {
		statusColor = cs.Error
	}
	toggle := func() { *expanded = !*expanded }
	tooltip := a.T("taskShowOutput")
	if *expanded {
		tooltip = a.T("taskHideOutput")
	}
	turn := column.AnimateTo("turn", kit.Flag(*expanded), k.Theme().Motion.FastSpatial.Anim())
	trailing := func() {
		mui.Row(k.Context).Children(func() {
			switch {
			case status.IsActive():
				kit.IconButton(k, kit.IconButtonSpec{
					ID: "cancel", Icon: "cancel", Tooltip: a.T("actionCancel"), OnClick: func() { a.model.Tasks.Cancel(task) },
				})
			case status == state.TaskFailed || status == state.TaskCancelled:
				kit.IconButton(k, kit.IconButtonSpec{
					ID: "retry", Icon: "retry", Tooltip: a.T("actionRetry"), OnClick: func() { a.model.Tasks.Retry(task) },
				})
			}
			kit.IconButton(k, kit.IconButtonSpec{
				ID: "toggle", Icon: "expandMore", Tooltip: tooltip, OnClick: toggle, Rotation: 180 * turn,
			})
		})
	}
	itemPosition := position
	switch {
	case *expanded && position == kit.PosLast:
		itemPosition = kit.PosMiddle
	case *expanded && position == kit.PosSingle:
		itemPosition = kit.PosFirst
	}
	titleStyle := kit.Emph(kit.BodyLarge)
	isLast := position == kit.PosLast || position == kit.PosSingle
	return column.Children(func() {
		kit.ListItem(k, kit.ListItemSpec{
			ID: "item", Position: itemPosition, Title: a.taskTitle(task.Command), TitleStyle: &titleStyle,
			Leading: func(mui.Color) { mui.Box(k.Context).Size(32, 32).Shrink(0).Center().Children(leading) },
			SubtitleWidget: func(mui.Color) {
				mui.Column(k.Context).Children(func() {
					kit.Label(k, statusText, kit.Type(kit.BodyMedium), statusColor, kit.Lines(2))
					if status == state.TaskRunning && task.Command.Operation == vfox.OperationInstall {
						kit.WavyProgress(k, kit.WavySpec{
							ID: "progress", Value: progress, Indeterminate: !hasProgress, Label: a.taskTitle(task.Command),
						}).MaxWidth(480).Margin(gapS, 0, 0, 0)
					}
				})
			},
			Trailing: trailing,
			OnClick:  toggle,
		})
		kit.Collapsible(k, "output", *expanded, func() {
			a.taskOutput(k, task, isLast, logScroll).Margin(hairline, 0, 0, 0)
		})
	})
}

// taskOutput is what vfox printed, under the command, which can be copied.
func (a *App) taskOutput(k kit.Ctx, task *state.VfoxTask, isLast bool, scroll *mui.ScrollState) mui.Element {
	cs := k.Colors()
	lines := task.Log()
	command := "$ " + task.Command.CommandLine()
	text := command + "\n" + a.T("taskNoOutput")
	if len(lines) > 0 {
		text = command + "\n" + strings.Join(lines, "\n")
	}
	style := kit.Type(kit.BodySmall)
	style.Line = style.Size * 1.5
	bottom := kit.CornerExtraSmall
	if isLast {
		bottom = kit.CornerLargeInc
	}
	// The log follows its end, unless the user scrolled up from it.
	if scroll.Y >= scroll.MaxY {
		scroll.Y = math.MaxFloat32
	}
	return kit.Panel(k, cs.SurfaceContainerHighest, kit.CornerExtraSmall, bottom, func() {
		mui.Row(k.Context).Padding(gapS, gapXS, gapM, gapL).AlignItems(mui.Start).Children(func() {
			mui.Scroll(k.Context).TrackScroll(scroll).Grow(1).Shrink(1).MaxHeight(320).Children(func() {
				kit.Label(k, text, style, cs.OnSurface, kit.Selectable(), kit.Monospace())
			})
			kit.IconButton(k, kit.IconButtonSpec{ID: "copy", Icon: "copy", Tooltip: a.T("taskCopyOutput"), OnClick: func() {
				a.copy(text)
				a.notify(a.T("actionCopied"))
			}})
		})
	})
}
