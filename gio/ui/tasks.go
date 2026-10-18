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
	"image/color"
	"math"
	"strings"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/core/vfox"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

// tasksPage lists what vfox did and is doing.
func (a *App) tasksPage() kit.Widget {
	return kit.Pane(func(c kit.Ctx) layout.Dimensions {
		tasks := a.model.Tasks.Tasks()
		header := kit.PageHeader(a.T("navTasks"), "", nil, kit.Button{
			ID: "clear", Kind: kit.TextButton, Icon: "succeeded", Label: a.T("tasksClearFinished"),
			Disabled: !a.model.Tasks.HasFinished(), OnClick: a.model.Tasks.ClearFinished,
		}.W())
		if len(tasks) == 0 {
			return headerAndBody(header, kit.StatusMessage{
				Icon: "tasks", Shape: shapes.ShapeSunny, Title: a.T("tasksEmptyTitle"), Body: a.T("tasksEmptyBody"),
			}.W())(c)
		}
		return kit.List("tasks", kit.ListSource{
			Count: len(tasks) + 1,
			Key: func(i int) any {
				if i == 0 {
					return "header"
				}
				return tasks[i-1].ID
			},
			Item: func(c kit.Ctx, i int) layout.Dimensions {
				if i == 0 {
					return header(c)
				}
				row := a.taskRow(tasks[i-1], kit.PositionOf(i-1, len(tasks)))
				return kit.Pad(kit.Inset{Left: gapXL, Right: gapXL, Bottom: hairline}, row)(c)
			},
		})(c)
	})
}

type taskRowState struct {
	expanded bool
	turn     kit.Anim
}

// taskRow is a task: what it does, how far it is, and its output.
func (a *App) taskRow(task *state.VfoxTask, position kit.Position) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		c = c.Scope("task:" + itoa(task.ID))
		cs := colors(c)
		ui := kit.UseID[taskRowState](c, "state")
		status := task.Status()
		progress, hasProgress := task.Progress()

		var leading kit.Widget
		switch status {
		case state.TaskQueued:
			leading = kit.Icon("queued", 24, cs.OnSurfaceVariant, 0)
		case state.TaskRunning:
			leading = kit.LoadingIndicator{ID: "running", Size: 32}.W()
		case state.TaskSucceeded:
			leading = kit.Icon("succeeded", 24, cs.Primary, 1)
		case state.TaskFailed:
			leading = kit.Icon("error", 24, cs.Error, 1)
		default:
			leading = kit.Icon("cancel", 24, cs.OnSurfaceVariant, 0)
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
		subtitle := []kit.Widget{kit.Label(statusText, kit.Type(kit.BodyMedium), statusColor, kit.Lines(2))}
		if status == state.TaskRunning && task.Command.Operation == vfox.OperationInstall {
			subtitle = append(subtitle, kit.Space(0, gapS), kit.MaxWidth(480, kit.WavyProgress{
				ID: "progress", Value: progress, Indeterminate: !hasProgress, Label: a.taskTitle(task.Command),
			}.W()))
		}

		var actions []kit.FlexChild
		switch {
		case status.IsActive():
			actions = append(actions, kit.Rigid(kit.IconButton{
				ID: "cancel", Icon: "cancel", Tooltip: a.T("actionCancel"), OnClick: func() { a.model.Tasks.Cancel(task) },
			}.W()))
		case status == state.TaskFailed || status == state.TaskCancelled:
			actions = append(actions, kit.Rigid(kit.IconButton{
				ID: "retry", Icon: "retry", Tooltip: a.T("actionRetry"), OnClick: func() { a.model.Tasks.Retry(task) },
			}.W()))
		}
		tooltip := a.T("taskShowOutput")
		if ui.expanded {
			tooltip = a.T("taskHideOutput")
		}
		toggle := func() { ui.expanded = !ui.expanded }
		turn := ui.turn.To(c, boolTo(ui.expanded)*math.Pi, c.Env.Theme.Motion.FastSpatial)
		actions = append(actions, kit.Rigid(kit.IconButton{
			ID: "toggle", Icon: "expandMore", Tooltip: tooltip, OnClick: toggle, Rotation: turn,
		}.W()))

		itemPosition := position
		switch {
		case ui.expanded && position == kit.PosLast:
			itemPosition = kit.PosMiddle
		case ui.expanded && position == kit.PosSingle:
			itemPosition = kit.PosFirst
		}
		titleStyle := kit.Emph(kit.BodyLarge)
		item := kit.ListItem{
			ID: "item", Position: itemPosition, Title: a.taskTitle(task.Command), TitleStyle: &titleStyle,
			Leading: func(color.NRGBA) kit.Widget { return kit.Sized(32, 32, kit.Center(leading)) },
			SubtitleWidget: func(color.NRGBA) kit.Widget {
				return kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true}, widgetsToChildren(subtitle)...)
			},
			Trailing: kit.Row(kit.FlexOpts{Cross: kit.CrossCenter, Min: true}, actions...),
			OnClick:  toggle,
		}
		isLast := position == kit.PosLast || position == kit.PosSingle
		output := kit.Collapsible("output", ui.expanded, kit.Pad(kit.Inset{Top: hairline}, a.taskOutput(task, isLast)))
		return kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true}, kit.Rigid(item.W()), kit.Rigid(output))(c)
	}
}

func boolTo(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// taskOutput is what vfox printed, under the command, which can be copied.
func (a *App) taskOutput(task *state.VfoxTask, isLast bool) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		lines := task.Log()
		command := "$ " + task.Command.CommandLine()
		text := command + "\n" + a.T("taskNoOutput")
		if len(lines) > 0 {
			text = command + "\n" + strings.Join(lines, "\n")
		}
		style := kit.Type(kit.BodySmall)
		style.Line = style.Size * 1.5
		log := kit.Limit(0, 0, 0, 320, kit.List("log", kit.ListSource{
			Count: 1, Shrink: true, StickToEnd: true,
			Item: func(c kit.Ctx, _ int) layout.Dimensions {
				return kit.SelectableText("text", text, style, cs.OnSurface, kit.Monospace())(c)
			},
		}))
		bottom := kit.CornerExtraSmall
		if isLast {
			bottom = kit.CornerLargeInc
		}
		return kit.Panel{
			Color: cs.SurfaceContainerHighest, Top: kit.CornerExtraSmall, Bottom: bottom,
			Child: kit.Pad(kit.Inset{Left: gapL, Top: gapS, Right: gapXS, Bottom: gapM},
				kit.Row(kit.FlexOpts{Cross: kit.CrossStart},
					kit.Expanded(1, log),
					kit.Rigid(kit.IconButton{ID: "copy", Icon: "copy", Tooltip: a.T("taskCopyOutput"), OnClick: func() {
						c.Env.Services.Copy(text)
						a.notify(a.T("actionCopied"))
					}}.W()),
				)),
		}.W()(c)
	}
}
