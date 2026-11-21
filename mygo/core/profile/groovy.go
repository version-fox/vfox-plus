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

package profile

// The profile of vfox-groovy (https://github.com/version-fox/vfox-groovy),
// written from its code at release 0.1.0, which its main branch shares.

// groovyProfile is the profile of vfox-groovy. The plugin sorts a pre-release
// above the release it precedes, and vfox takes latest from the first line of
// the list, so latest would install a release candidate: the profile names no
// shorthand. The plugin reads no variable and keeps no notes.
func groovyProfile() Profile {
	return Profile{
		Plugin: "groovy",
		Guide: text(
			"Write the version as archive.apache.org lists it, such as 4.0.33 or 6.0.0-RC-2. A pre-release keeps its suffix in the name.",
			"按 archive.apache.org 列出的样子写版本，如 4.0.33 或 6.0.0-RC-2。预发布版的后缀保留在名称中。",
		),
		Tips: []Text{
			text(
				"The plugin installs no JDK, and Groovy needs one to run.",
				"插件不安装 JDK，而 Groovy 运行需要 JDK。"),
			text(
				"latest is not offered: the plugin sorts a pre-release above its release, so the first line can be a release candidate.",
				"未提供 latest：插件把预发布版排在它的正式版之上，列表第一行可能是候选版。"),
			text(
				"Some listed versions have no download, such as 2.4.5-incubating.",
				"列表中有些版本没有下载文件，如 2.4.5-incubating。"),
		},
		Docs: []Link{
			readme("vfox-groovy", text("Plugin README", "插件说明")),
		},
	}
}
