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

package i18n

// The messages that only the tasks tab, the task page and the full log of a task
// show. The screens use the web's keys as they are; the keys here are the ones the
// phone adds, and they start with "task".
func init() {
	Register(map[string][2]string{
		"taskDetails": {
			"Details",
			"详情",
		},
		"taskShowOutput": {
			"Show the output",
			"展开输出",
		},
		"taskHideOutput": {
			"Hide the output",
			"收起输出",
		},
		"taskFullLog": {
			"Full log",
			"完整日志",
		},
		"taskCopyLog": {
			"Copy log",
			"复制日志",
		},
		"taskCancelled": {
			"Task cancelled.",
			"任务已取消。",
		},
		"taskNoOutput": {
			"No output yet.",
			"还没有输出。",
		},
		"taskListCapped": {
			"Only the newest {count} tasks are listed.",
			"只列出最新的 {count} 个任务。",
		},
		"taskLogTail": {
			"Showing the last {shown} of {total} lines.",
			"只显示最后 {shown} 行，共 {total} 行。",
		},
		"taskLogNone": {
			"The computer kept no complete log of this task.",
			"电脑没有保存这个任务的完整日志。",
		},
	})
}
