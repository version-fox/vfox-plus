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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// tasksWindow returns the window of a phone paired with f, on the tasks tab.
func tasksWindow(t *testing.T, f *tsFake) (*Root, *ui.Tester) {
	t.Helper()
	root := tsRoot(t, f)
	root.App().SelectTab(app.TabTasks)
	return root, ui.NewTester(root.View, 480, 900)
}

func TestTasksTabShowsEachStatusWithItsTitle(t *testing.T) {
	f := tsNewFake(t)
	half := 0.5
	running := tsTask(4, "installUse", "go", "1.24", remote.TaskRunning)
	running.Progress = &half
	alias := tsTask(7, "aliasSet", "java", "21", remote.TaskCancelled)
	alias.Alias = "lts"
	f.tsSetTasks(
		tsTask(5, "install", "nodejs", "24.11.0", remote.TaskQueued),
		running,
		tsTask(3, "uninstall", "java", "21", remote.TaskRunning),
		tsTask(2, "addPlugin", "nodejs", "", remote.TaskSucceeded),
		tsTask(1, "link", "node", "20", remote.TaskFailed),
		alias,
	)
	_, tt := tasksWindow(t, f)
	tsWaitText(t, tt, "Install nodejs@24.11.0")
	for _, want := range []string{
		"Queued", "Install and use go@1.24", "Running · 50%", "Uninstall java@21", "Running",
		"Add a plugin nodejs", "Done", "Link node@20", "Failed", "Name alias lts for java@21", "Cancelled",
		"Clear finished",
	} {
		if !tsHas(tt, want) {
			t.Errorf("the tasks tab has no %q; the frame shows %q", want, tt.Texts())
		}
	}
	tsShot(t, tt, "tasks-list")
	if err := tt.Click("Clear finished"); err != nil {
		t.Fatalf("Click(Clear finished): %v", err)
	}
	tsWaitCall(t, tt, f, "DELETE /api/tasks/finished")
	tsWaitUntil(t, tt, "the finished tasks to go", func() bool { return !tsHas(tt, "Done") })
}

func TestTasksTabSaysSoWhenThereAreNoTasks(t *testing.T) {
	f := tsNewFake(t)
	_, tt := tasksWindow(t, f)
	tsWaitText(t, tt, "No tasks yet.")
	tsShot(t, tt, "tasks-empty")
}

func TestATaskOpensItsOutputAndCanBeCancelled(t *testing.T) {
	f := tsNewFake(t)
	half := 0.5
	task := tsTask(4, "installUse", "go", "1.24", remote.TaskRunning)
	task.Progress = &half
	task.DroppedLines = 3
	task.Log = []string{"downloading go1.24", "unpacking go1.24"}
	f.tsSetTasks(task)
	_, tt := tasksWindow(t, f)
	tsWaitText(t, tt, "Install and use go@1.24")
	if err := tt.Click("Install and use go@1.24"); err != nil {
		t.Fatalf("Click(the row): %v", err)
	}
	tsWaitText(t, tt, "vfox installUse go@1.24")
	if !tsHas(tt, "3 earlier lines dropped") {
		t.Errorf("the open row does not say that lines were dropped; the frame shows %q", tt.Texts())
	}
	if !tt.HasText("unpacking go1.24") {
		t.Errorf("the open row does not show the newest output; the frame shows %q", tt.Texts())
	}
	tsShot(t, tt, "task-row-open")
	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("Click(Cancel): %v", err)
	}
	tsWaitCall(t, tt, f, "POST /api/tasks/4/cancel")
	tsWaitText(t, tt, "Retry")
	if !tsHas(tt, "Cancelled · 50%") {
		t.Errorf("the cancelled task is not shown as cancelled, with its progress; the frame shows %q", tt.Texts())
	}
}

func TestTheTaskPageShowsTheNewestLinesAndTheFullLog(t *testing.T) {
	f := tsNewFake(t)
	task := tsTask(4, "installUse", "go", "1.24", remote.TaskFailed)
	task.ErrorMessage = "disk full"
	task.HasFullLog = true
	f.tsSetTasks(task)
	var lines []string
	for i := 1; i <= 700; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	f.logs[4] = strings.Join(lines, "\n") + "\n"
	root, tt := tasksWindow(t, f)
	root.App().Push(app.Page{Kind: app.PageTask, Name: "4"})
	tsWaitText(t, tt, "Install and use go@1.24")
	tsWaitText(t, tt, "Failed")
	tsShot(t, tt, "task-page")
	if err := tt.Click("Full log"); err != nil {
		t.Fatalf("Click(Full log): %v", err)
	}
	tsWaitText(t, tt, "Showing the last 500 of 700 lines.")
	if !tt.HasText("line 700") {
		t.Errorf("the full log does not show its last line; the frame shows %q", tt.Texts())
	}
	if !tt.HasText("line 201") || tt.HasText("line 200") {
		t.Error("the full log does not start at the 201st line, which is the first of the newest 500")
	}
	tsShot(t, tt, "log-page")
}

func TestAMissingFullLogSaysThatTheComputerKeptNone(t *testing.T) {
	f := tsNewFake(t)
	task := tsTask(3, "uninstall", "java", "21", remote.TaskFailed)
	task.HasFullLog = true
	f.tsSetTasks(task)
	root, tt := tasksWindow(t, f)
	root.App().Push(app.Page{Kind: app.PageLog, Name: "3"})
	tsWaitText(t, tt, "The computer kept no complete log of this task.")
}

// TestAFinishedTaskIsAnnouncedOnAnyTab checks that the message of a task that ends is
// shown on the home tab too: the web shows it on every page.
func TestAFinishedTaskIsAnnouncedOnAnyTab(t *testing.T) {
	f := tsNewFake(t)
	f.tsSetTasks(tsTask(9, "install", "node", "20", remote.TaskRunning))
	root := tsRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	// The first look only records the statuses, so the change comes after it.
	tsWaitUntil(t, tt, "the window to record the running task", func() bool { return root.tasks.looked })
	f.tsSetStatus(9, remote.TaskFailed)
	root.Service().Refresh()
	tsWaitText(t, tt, "Task failed.")
	if tab := root.App().Tab(); tab != app.TabHome {
		t.Errorf("the message was shown with the window on tab %v, want the home tab", tab)
	}
}

func TestAFinishedTaskIsAnnouncedOnce(t *testing.T) {
	f := tsNewFake(t)
	f.tsSetTasks(tsTask(9, "install", "node", "20", remote.TaskRunning))
	root, tt := tasksWindow(t, f)
	tsWaitText(t, tt, "Running")
	f.tsSetStatus(9, remote.TaskSucceeded)
	root.Service().Refresh()
	tsWaitText(t, tt, "Task succeeded.")
}

func TestTheBadgeOfTheTasksTabCountsWhatWaitsOrRuns(t *testing.T) {
	f := tsNewFake(t)
	f.tsSetTasks(
		tsTask(1, "install", "node", "20", remote.TaskQueued),
		tsTask(2, "install", "go", "1.24", remote.TaskRunning),
		tsTask(3, "install", "java", "21", remote.TaskSucceeded),
	)
	root := tsRoot(t, f)
	deadline := time.Now().Add(5 * time.Second)
	for {
		badge := -1
		for _, destination := range root.tabDestinations() {
			if destination.Label == "Tasks" {
				badge = destination.Badge
			}
		}
		if badge == 2 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the badge of the tasks tab is %d, want 2 (one queued, one running)", badge)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
