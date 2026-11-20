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
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// taskListLimit bounds the tasks that the list shows: the newest ones.
const taskListLimit = 200

// taskOperationKeys are the messages of the operations of a task, by the name the
// computer gives the task. The web shows the plugin operations by their own names;
// the phone uses the messages of the operations they are (add, remove, update).
var taskOperationKeys = map[string]string{
	remote.OperationInstall:          "install",
	remote.OperationInstallUse:       "installUse",
	remote.OperationUninstall:        "uninstall",
	remote.OperationUse:              "use",
	remote.OperationUnuse:            "unuse",
	remote.OperationAddPlugin:        "add",
	remote.OperationRemovePlugin:     "remove",
	remote.OperationUpdatePlugin:     "update",
	remote.OperationUpdateAllPlugins: "updateAll",
	remote.OperationLink:             "link",
	remote.OperationLinkDetect:       "linkDetect",
	remote.OperationDoctorAdopt:      "doctorAdopt",
}

// taskStatusKeys are the messages of the statuses of a task.
var taskStatusKeys = map[string]string{
	remote.TaskQueued:    "statusQueued",
	remote.TaskRunning:   "statusRunning",
	remote.TaskSucceeded: "statusDone",
	remote.TaskFailed:    "statusFailed",
	remote.TaskCancelled: "statusCancelled",
}

// taskTitle is what a task does, as the web writes it: the operation and the SDK
// with its version, such as "Install nodejs@24.11.0". The alias operations have
// their own messages, which name the alias and the SDK.
func taskTitle(task remote.Task) string {
	subject := task.Sdk
	if task.Version != "" {
		subject += "@" + task.Version
	}
	switch task.Operation {
	case remote.OperationAliasSet:
		return strings.NewReplacer("{alias}", task.Alias, "{subject}", subject).Replace(i18n.Text("taskAliasSet"))
	case remote.OperationAliasUnset:
		return strings.NewReplacer("{alias}", task.Alias, "{subject}", subject).Replace(i18n.Text("taskAliasUnset"))
	}
	label := task.Operation
	if key, ok := taskOperationKeys[task.Operation]; ok {
		label = i18n.Text(key)
	}
	label = upperFirst(label)
	if subject != "" {
		label += " " + subject
	}
	return strings.TrimSpace(label)
}

// upperFirst capitalises the first letter of s, as the web does with the names of
// the operations.
func upperFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if size == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

// taskStatusText is the status of a task with its progress, as the web writes it:
// "Running · 40%". A task without a progress has no percentage.
func taskStatusText(task remote.Task) string {
	text := task.Status
	if key, ok := taskStatusKeys[task.Status]; ok {
		text = i18n.Text(key)
	}
	if task.Progress != nil {
		text += " · " + taskPercent(*task.Progress)
	}
	return text
}

// taskPercent writes a progress, from 0 to 1, as a whole percentage.
func taskPercent(progress float64) string {
	return strconv.Itoa(int(math.Round(progress*100))) + "%"
}

// taskActive reports whether a task waits for its turn or runs, so that it can be
// cancelled.
func taskActive(status string) bool {
	return status == remote.TaskQueued || status == remote.TaskRunning
}

// taskRetryable reports whether a task stopped without success, so that it can run
// again.
func taskRetryable(status string) bool {
	return status == remote.TaskFailed || status == remote.TaskCancelled
}

// tasksBadge is the number on the tasks tab: the tasks that wait or run.
func tasksBadge(tasks []remote.Task) int {
	n := 0
	for _, task := range tasks {
		if taskActive(task.Status) {
			n++
		}
	}
	return n
}

// newestTasks returns the tasks that the list shows, the newest first: at most
// taskListLimit of them, and the number of the older ones that were left out. It
// returns a new slice, and does not change tasks.
func newestTasks(tasks []remote.Task) ([]remote.Task, int) {
	out := slices.Clone(tasks)
	slices.SortFunc(out, func(a, b remote.Task) int { return b.ID - a.ID })
	if len(out) > taskListLimit {
		return out[:taskListLimit], len(out) - taskListLimit
	}
	return out, 0
}

// taskFailureText is what a failed task says: the error of the computer, or
// "Task failed." when there is none. A cancelled task is not a failure: it says
// "Task cancelled." instead, where the web shows a failure.
func taskFailureText(task remote.Task) string {
	if task.ErrorMessage != "" {
		return task.ErrorMessage
	}
	return i18n.Text("taskFailed")
}

// taskWatch finds the tasks that ended since the last look, which the window
// reports as messages, as the web does. The first look only records the statuses,
// since the tasks that ended before it are not news. A task that is new since the
// last look, and has ended by then, is not reported either.
type taskWatch struct {
	seen   map[int]string
	looked bool
}

// observe takes the tasks of a look, and returns the messages of the tasks that
// succeeded, failed or were cancelled since the last look. It keeps the statuses
// of tasks; a task that the computer no longer lists is forgotten.
func (w *taskWatch) observe(tasks []remote.Task) []string {
	var messages []string
	next := make(map[int]string, len(tasks))
	for _, task := range tasks {
		next[task.ID] = task.Status
		if !w.looked {
			continue
		}
		if prev := w.seen[task.ID]; prev != remote.TaskQueued && prev != remote.TaskRunning {
			continue
		}
		switch task.Status {
		case remote.TaskSucceeded:
			messages = append(messages, i18n.Text("taskSucceeded"))
		case remote.TaskFailed:
			messages = append(messages, taskFailureText(task))
		case remote.TaskCancelled:
			messages = append(messages, i18n.Text("taskCancelled"))
		}
	}
	w.seen, w.looked = next, true
	return messages
}
