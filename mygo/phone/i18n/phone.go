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

// phoneEnglish holds the messages that only the phone shows: the name of the app,
// the placeholders of the screens that are not built yet, and the pairing page.
// They are kept out of the web's catalogs, which keep the web's keys.
var phoneEnglish = map[string]string{
	"appName":       "vfox+",
	"back":          "Back",
	"connect":       "Connect",
	"pairLead":      "Paste the pairing link that vfox+ shows on your computer, under Other devices in Settings.",
	"pairLinkHint":  "https://…",
	"pairLinkLabel": "Pairing link",
	"phonePending":  "Not built yet.",
}

// phoneChinese is phoneEnglish in Simplified Chinese; it has the same keys.
var phoneChinese = map[string]string{
	"appName":       "vfox+",
	"back":          "返回",
	"connect":       "连接",
	"pairLead":      "粘贴电脑上 vfox+ 设置页“其他设备”中显示的配对链接。",
	"pairLinkHint":  "https://…",
	"pairLinkLabel": "配对链接",
	"phonePending":  "待实现。",
}
