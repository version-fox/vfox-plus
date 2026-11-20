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
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// tasksInEnglish sets the English messages for a test, which reads them.
func tasksInEnglish(t *testing.T) {
	t.Helper()
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
}

func TestTasksTitleFollowsTheWebRules(t *testing.T) {
	tasksInEnglish(t)
	cases := []struct {
		name string
		task remote.Task
		want string
	}{
		{"install with its version", remote.Task{Operation: remote.OperationInstall, Sdk: "nodejs", Version: "24.11.0"}, "Install nodejs@24.11.0"},
		{"install and use", remote.Task{Operation: remote.OperationInstallUse, Sdk: "go", Version: "1.24"}, "Install and use go@1.24"},
		{"uninstall without a version", remote.Task{Operation: remote.OperationUninstall, Sdk: "java"}, "Uninstall java"},
		{"alias set names the alias", remote.Task{Operation: remote.OperationAliasSet, Sdk: "java", Version: "21", Alias: "lts"}, "Name alias lts for java@21"},
		{"alias unset", remote.Task{Operation: remote.OperationAliasUnset, Sdk: "java", Alias: "lts"}, "Remove alias lts of java"},
		{"a plugin added by its name", remote.Task{Operation: remote.OperationAddPlugin, Sdk: "nodejs"}, "Add a plugin nodejs"},
		{"update of every plugin", remote.Task{Operation: remote.OperationUpdateAllPlugins}, "Update all"},
		{"an operation the phone does not know", remote.Task{Operation: "frobnicate", Sdk: "x"}, "Frobnicate x"},
	}
	for _, tc := range cases {
		if got := taskTitle(tc.task); got != tc.want {
			t.Errorf("%s: taskTitle = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestTheMessagesOfTasksAreRegistered(t *testing.T) {
	for op, key := range taskOperationKeys {
		for _, lang := range []i18n.Language{i18n.English, i18n.Chinese} {
			if i18n.TextIn(lang, key) == key {
				t.Errorf("operation %s: the message %q is missing in %s", op, key, lang)
			}
		}
	}
	for status, key := range taskStatusKeys {
		for _, lang := range []i18n.Language{i18n.English, i18n.Chinese} {
			if i18n.TextIn(lang, key) == key {
				t.Errorf("status %s: the message %q is missing in %s", status, key, lang)
			}
		}
	}
}

func TestTaskStatusTextShowsTheProgress(t *testing.T) {
	tasksInEnglish(t)
	half := 0.4
	cases := []struct {
		task remote.Task
		want string
	}{
		{remote.Task{Status: remote.TaskQueued}, "Queued"},
		{remote.Task{Status: remote.TaskRunning, Progress: &half}, "Running · 40%"},
		{remote.Task{Status: remote.TaskRunning}, "Running"},
		{remote.Task{Status: remote.TaskSucceeded}, "Done"},
		{remote.Task{Status: remote.TaskFailed}, "Failed"},
		{remote.Task{Status: remote.TaskCancelled}, "Cancelled"},
	}
	for _, tc := range cases {
		if got := taskStatusText(tc.task); got != tc.want {
			t.Errorf("taskStatusText(%s) = %q, want %q", tc.task.Status, got, tc.want)
		}
	}
}

func TestTheBadgeCountsTheTasksThatWaitOrRun(t *testing.T) {
	tasks := []remote.Task{
		{ID: 1, Status: remote.TaskQueued},
		{ID: 2, Status: remote.TaskRunning},
		{ID: 3, Status: remote.TaskSucceeded},
		{ID: 4, Status: remote.TaskFailed},
		{ID: 5, Status: remote.TaskCancelled},
	}
	if got := tasksBadge(tasks); got != 2 {
		t.Errorf("tasksBadge = %d, want 2 (one queued, one running)", got)
	}
	if got := tasksBadge(nil); got != 0 {
		t.Errorf("tasksBadge(nil) = %d, want 0", got)
	}
}

func TestTheListIsNewestFirstAndBounded(t *testing.T) {
	var tasks []remote.Task
	for id := 1; id <= taskListLimit+5; id++ {
		tasks = append(tasks, remote.Task{ID: id})
	}
	before := tasks[0].ID
	out, cut := newestTasks(tasks)
	if len(out) != taskListLimit || cut != 5 {
		t.Fatalf("newestTasks kept %d and cut %d, want %d and 5", len(out), cut, taskListLimit)
	}
	if out[0].ID != taskListLimit+5 || out[len(out)-1].ID != 6 {
		t.Errorf("the list runs from %d to %d, want from %d to 6", out[0].ID, out[len(out)-1].ID, taskListLimit+5)
	}
	if tasks[0].ID != before || tasks[0].ID != 1 {
		t.Error("newestTasks changed the tasks it was given")
	}
}

func TestAFailureSaysItsErrorOrTaskFailed(t *testing.T) {
	tasksInEnglish(t)
	cases := []struct {
		task remote.Task
		want string
	}{
		{remote.Task{Status: remote.TaskFailed, ErrorMessage: "disk full"}, "disk full"},
		{remote.Task{Status: remote.TaskFailed}, "Task failed."},
	}
	for _, tc := range cases {
		if got := taskFailureText(tc.task); got != tc.want {
			t.Errorf("taskFailureText(%s, %q) = %q, want %q", tc.task.Status, tc.task.ErrorMessage, got, tc.want)
		}
	}
}

func TestTheWatchReportsOnlyTasksThatEndedSinceTheLastLook(t *testing.T) {
	tasksInEnglish(t)
	var watch taskWatch
	first := []remote.Task{
		{ID: 1, Status: remote.TaskRunning},
		{ID: 2, Status: remote.TaskQueued},
		{ID: 3, Status: remote.TaskRunning},
	}
	if got := watch.observe(first); len(got) != 0 {
		t.Fatalf("the first look reported %v; the tasks that ended before it are not news", got)
	}
	second := []remote.Task{
		{ID: 1, Status: remote.TaskSucceeded},
		{ID: 2, Status: remote.TaskFailed, ErrorMessage: "disk full"},
		{ID: 3, Status: remote.TaskCancelled},
		{ID: 4, Status: remote.TaskSucceeded},
	}
	got := watch.observe(second)
	want := []string{"Task succeeded.", "disk full", "Task cancelled."}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("the look reported %q, want %q (task 4 is new and ended, so it is not news)", got, want)
	}
	if again := watch.observe(second); len(again) != 0 {
		t.Errorf("the same tasks again reported %q", again)
	}
}

func TestAForgottenTaskDoesNotComeBackAsNews(t *testing.T) {
	tasksInEnglish(t)
	var watch taskWatch
	watch.observe([]remote.Task{{ID: 7, Status: remote.TaskRunning}})
	watch.observe(nil)
	if got := watch.observe([]remote.Task{{ID: 7, Status: remote.TaskSucceeded}}); len(got) != 0 {
		t.Errorf("a task the list had dropped reported %q", got)
	}
}
