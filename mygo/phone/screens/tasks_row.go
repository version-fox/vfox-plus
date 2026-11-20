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

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// taskRow is a task of the list: its title, its status with its progress, and the
// button that opens its output under the row. The output is read from the computer
// only while it is open, so a closed row costs no request.
func (r *Root) taskRow(k kit.Ctx, task remote.Task, position kit.Position) ui.Element {
	cs := k.Colors()
	column := ui.Column(k.Key(task.ID))
	open := ui.Local(column, "open", func() bool { return false })
	turn := column.AnimateTo("turn", kit.Flag(*open), k.Theme().Motion.FastSpatial.Anim())
	title := taskTitle(task)
	status := taskStatusText(task)
	statusColor := cs.OnSurfaceVariant
	if task.Status == remote.TaskFailed {
		statusColor = cs.Error
	}
	toggle := func() { *open = !*open }
	tooltip := i18n.Text("taskShowOutput")
	if *open {
		tooltip = i18n.Text("taskHideOutput")
	}
	// An open row has its output under it, so its corners at the bottom are square.
	itemPosition := position
	switch {
	case *open && position == kit.PosLast:
		itemPosition = kit.PosMiddle
	case *open && position == kit.PosSingle:
		itemPosition = kit.PosFirst
	}
	last := position == kit.PosLast || position == kit.PosSingle
	titleStyle := kit.Emph(kit.BodyLarge)
	return column.Children(func() {
		kit.ListItem(k, kit.ListItemSpec{
			ID: "item", Position: itemPosition, Title: title, TitleStyle: &titleStyle,
			// The status is part of the name of the row, for assistive technology, as it is of what the row shows.
			Name: title + ", " + status,
			Leading: func(ui.Color) {
				ui.Box(k.Context).Size(32, 32).Shrink(0).Center().Children(func() {
					r.taskStatusIcon(k, task.Status)
				})
			},
			SubtitleWidget: func(ui.Color) {
				ui.Column(k.Context).Children(func() {
					kit.Label(k, status, kit.Type(kit.BodyMedium), statusColor, kit.Lines(2), kit.Breakable())
					if taskActive(task.Status) {
						taskProgress(k, task)
					}
				})
			},
			Trailing: func() {
				kit.IconButton(k, kit.IconButtonSpec{
					ID: "toggle", Icon: "expandMore", Tooltip: tooltip, OnClick: toggle, Rotation: 180 * turn,
				})
			},
			OnClick: toggle,
		})
		kit.Collapsible(k, "output", *open, func() {
			r.taskPanel(k, task.ID, last)
		})
	})
}

// taskStatusIcon is the icon of a status, as the web has it: an hourglass for a
// queued task, a spinner for a running one, a tick, an error and a cross.
func (r *Root) taskStatusIcon(k kit.Ctx, status string) {
	cs := k.Colors()
	switch status {
	case remote.TaskQueued:
		kit.Icon(k, "queued", 24, cs.OnSurfaceVariant, 0)
	case remote.TaskRunning:
		kit.LoadingIndicator(k, kit.LoadingSpec{ID: "running", Size: 32})
	case remote.TaskSucceeded:
		kit.Icon(k, "succeeded", 24, cs.Primary, 1)
	case remote.TaskFailed:
		kit.Icon(k, "error", 24, cs.Error, 1)
	default:
		kit.Icon(k, "cancel", 24, cs.OnSurfaceVariant, 0)
	}
}

// taskProgress is the bar of a task that waits or runs. It is determinate when the
// computer reports a progress, and it moves without a value when it reports none.
func taskProgress(k kit.Ctx, task remote.Task) {
	value, indeterminate := 0.0, task.Progress == nil
	if !indeterminate {
		value = *task.Progress
	}
	kit.WavyProgress(k, kit.WavySpec{
		ID: "progress", Value: value, Indeterminate: indeterminate, Label: taskTitle(task),
	}).MaxWidth(480).Margin(4, 0, 0, 0)
}

// taskPanel is the output of a row, under it: the detail of the task, with the
// newest lines of its output. last says whether the row is the last of the list,
// which decides its bottom corners.
func (r *Root) taskPanel(k kit.Ctx, id int, last bool) {
	bottom := kit.CornerExtraSmall
	if last {
		bottom = kit.CornerLargeInc
	}
	kit.Panel(k, k.Colors().SurfaceContainerHighest, kit.CornerExtraSmall, bottom, func() {
		ui.Column(k.Context).Padding(12, 16, 12, 16).Gap(8).Children(func() {
			r.taskDetail(k, id, taskRowLines, true)
		})
	})
}
