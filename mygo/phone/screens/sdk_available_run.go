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

// Starting work from the installable versions and the plugin tab. The web shows the
// tasks once a task has started, and not before: the call runs in the background, and
// the window is changed only on the main thread, so the call leaves a flag that the
// frame that follows reads.

import (
	"context"
	"sync/atomic"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// availFollowTasks shows the tasks tab when a call that landed says so. It runs on
// the main thread, in the frame of the tab that made the call, and asks for another
// frame so that the change is drawn.
func (r *Root) availFollowTasks(k kit.Ctx, landed *atomic.Bool) {
	if landed.Swap(false) {
		r.App().SelectTab(app.TabTasks)
		k.Invalidate()
	}
}

// availThen runs call in the background, with Root.Do. When call succeeds, the frame
// that follows shows the tasks tab (see availFollowTasks); a call that fails shows its
// error instead, and the tasks tab stays where it is, as the web's does.
func (r *Root) availThen(landed *atomic.Bool, call func(ctx context.Context) error) {
	r.Do(func(ctx context.Context) error {
		if err := call(ctx); err != nil {
			return err
		}
		landed.Store(true)
		r.invalidate()
		return nil
	}, "")
}

// availStart queues the task req on the computer, and then shows the tasks tab.
func (r *Root) availStart(landed *atomic.Bool, req remote.TaskRequest) {
	r.availThen(landed, func(ctx context.Context) error {
		_, err := r.Service().StartTask(ctx, req)
		return err
	})
}
