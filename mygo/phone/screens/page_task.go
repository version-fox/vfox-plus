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

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// taskPage is the page of a task, opened from the tasks tab: its title and status,
// and the detail that a row shows under it, with more of its output. The page's
// name is the ID of the task.
func (r *Root) taskPage(k kit.Ctx, page app.Page) {
	id, err := strconv.Atoi(page.Name)
	if err != nil {
		kit.StatusMessage(k, kit.StatusSpec{Icon: "error", IsError: true, Title: i18n.Text("errNotFound")})
		return
	}
	cs := k.Colors()
	task, ok := r.service.Task(id).Value()
	ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
		ui.Column(k.Context).Padding(16, 16, 24, 16).Gap(12).Children(func() {
			if ok {
				kit.Label(k, taskTitle(task), kit.Emph(kit.TitleLarge), cs.OnSurface, kit.Breakable())
				statusColor := cs.OnSurfaceVariant
				if task.Status == remote.TaskFailed {
					statusColor = cs.Error
				}
				kit.Label(k, taskStatusText(task), kit.Type(kit.BodyLarge), statusColor, kit.Breakable())
				if taskActive(task.Status) {
					taskProgress(k, task)
				}
			}
			r.taskDetail(k, id, taskPageLines, false)
		})
	})
}
