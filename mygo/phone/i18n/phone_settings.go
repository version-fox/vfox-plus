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

// The messages that only the settings tab shows, which the phone adds to the web's
// keys. Their keys start with "set".
func init() {
	Register(map[string][2]string{
		"setLangEnglish": {
			"English",
			"English",
		},
		"setLangChinese": {
			// The web names this language in its own script in both languages, so the segment is the same.
			"简体中文",
			"简体中文",
		},
		"setUILanguage": {
			"Interface language",
			"界面语言",
		},
		"setUILanguageHelp": {
			"Changes only the screens of this phone.",
			"只改变这台手机的界面。",
		},
		"setUIPrefFailed": {
			"The interface language could not be saved.",
			"无法保存界面语言。",
		},
		"setEnvHelp": {
			"One KEY=value per line. Saving replaces all the variables.",
			"每行一个 KEY=value。保存会整体替换全部变量。",
		},
		"setPortRange": {
			"Use a port from 1024 to 65535.",
			"端口须在 1024 到 65535 之间。",
		},
		"setPortHint": {
			"1024 to 65535",
			"1024 到 65535",
		},
		"setRemoteOffTitle": {
			"Stop other devices?",
			"停止其他设备的访问？",
		},
		"setRemoteOffBody": {
			"Other devices lose their connection, and this phone with them. This phone must pair again to come back.",
			"其他设备会断开连接，这台手机也一样。这台手机需要重新配对才能再次连接。",
		},
		"setRemoteOffOk": {
			"Stop",
			"停止",
		},
		"setPairOnce": {
			"Each link works once, for five minutes.",
			"每个链接只能使用一次，有效期五分钟。",
		},
		"setConfigEdit": {
			"Change the value. An empty value is saved as it is.",
			"修改这个值。空值会原样保存。",
		},
	})
}
