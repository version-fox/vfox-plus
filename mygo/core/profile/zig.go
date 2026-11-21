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

// The profile of vfox-zig (https://github.com/version-fox/vfox-zig), written
// from its code at release 0.1.1, which its main branch shares.

// zigProfile is the profile of vfox-zig. The list merges ziglang.org with
// machengine.org, and the plugin marks latest on a line that is not the newest,
// so latest is not offered: vfox takes the first line of the list for it, and
// that line can be a development build. The plugin reads no variable.
func zigProfile() Profile {
	return Profile{
		Plugin: "zig",
		Shorthands: []Shorthand{
			{Name: "nightly", Meaning: text("The development build of the master branch of Zig", "Zig master 分支的开发版")},
		},
		Guide: text(
			"Write the release as ziglang.org lists it, such as 0.17.1. nightly installs the newest development build; a build of machengine.org installs by the name in its note, such as mach-latest.",
			"按 ziglang.org 列出的样子写版本，如 0.17.1。nightly 安装最新的开发版；machengine.org 的构建按其备注中的名称安装，如 mach-latest。",
		),
		Tips: []Text{
			text(
				"The list needs both ziglang.org and machengine.org: when machengine.org cannot be reached, the whole list fails.",
				"列表同时需要 ziglang.org 与 machengine.org：machengine.org 无法访问时，整个列表都会失败。"),
			text(
				"latest is not offered: vfox takes the first line of the list for it, and that line can be a development build.",
				"未提供 latest：vfox 取列表第一行作为 latest，而那一行可能是开发版。"),
			text(
				"A name that the list does not show can report success without installing anything.",
				"列表中没有的名称可能报告安装成功，却什么也没有安装。"),
		},
		Docs: []Link{
			readme("vfox-zig", text("Plugin README", "插件说明")),
		},
	}
}
