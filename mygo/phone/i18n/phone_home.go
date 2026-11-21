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

// This file holds the messages of the home tab. Its screens' group owns it, and no
// other group edits it. Add the messages in an init function, with Register:
//
//	func init() {
//		Register(map[string][2]string{
//			"someKey": {"English text.", "中文文本。"},
//		})
//	}
//
// Every key is checked by the tests: a key used in the code must be registered.

package i18n

func init() {
	Register(map[string][2]string{
		"homeVersion": {
			"vfox {version}",
			"vfox {version}",
		},
		"homeShellHint": {
			"Run this in your shell to use vfox.",
			"在终端中运行这一行，以使用 vfox。",
		},
		"copyCommand": {
			"Copy",
			"复制",
		},
	})
}
