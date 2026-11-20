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

// This file holds the messages of the SDK page: its tabs, installed versions and options. Its screens' group owns it, and
// no other group edits it. Add the messages in an init function, with Register:
//
//	func init() {
//		Register(map[string][2]string{
//			"someKey": {"English text.", "中文文本。"},
//		})
//	}
//
// Every key is checked by the tests: a key used in the code must be registered.

package i18n

// The messages that only the phone's SDK screens show. The web interface's keys
// are used as they are, and are not registered again.
func init() {
	Register(map[string][2]string{
		"sdkQueued": {
			"Queued. See the Tasks tab for its progress.",
			"已加入任务队列，进度见任务页。",
		},
		"sdkClearSearch": {
			"Clear search",
			"清除搜索",
		},
	})
}
