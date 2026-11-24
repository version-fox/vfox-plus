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
// from its code at release 0.1.4, which its main branch shares.

// zigProfile is the profile of vfox-zig. The list needs ziglang.org, and
// machengine.org is optional: without it the list shows the official releases,
// with a warning. latest is the newest stable release, and nightly is the
// development build of the master branch. vfox 1.1.0 exec takes a version name
// and not these names, so the tips say which name to give it. The plugin reads
// no variable of its own; its link detection reads VFOX_HOME, PATH and HOME.
func zigProfile() Profile {
	return Profile{
		Plugin: "zig",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest stable release", "最新的稳定版")},
			{Name: "nightly", Meaning: text("The development build of the master branch of Zig", "Zig master 分支的开发版")},
		},
		Guide: text(
			"Write the release as ziglang.org lists it, such as 0.17.1. latest installs the newest stable release, and nightly the newest development build. A Mach build installs by the name in its note, such as mach-latest.",
			"按 ziglang.org 列出的样子写版本，如 0.17.1。latest 安装最新的稳定版，nightly 安装最新的开发版。Mach 的构建按其备注中的名称安装，如 mach-latest。",
		),
		Tips: []Text{
			text(
				"The list needs ziglang.org.",
				"列表需要 ziglang.org。",
			),
			text(
				"machengine.org is optional.",
				"machengine.org 可选。",
			),
			text(
				"Unlisted names are refused.",
				"列表中没有的名称会被拒绝。",
			),
			text(
				"vfox 1.1.0 exec takes version names only.",
				"vfox 1.1.0 的 exec 只接受版本名。",
			),
			text(
				"Mach builds may come from pkg.hexops.org.",
				"Mach 构建可能来自 pkg.hexops.org。",
			),
			text(
				"Detect and link finds existing zig installs.",
				"探测并链接会找出已有的 zig 安装。",
			),
		},
		Docs: []Link{
			docs("vfox-zig", "versions", text("Versions", "版本")),
			docs("vfox-zig", "link", text("Linking installed versions", "链接已安装的版本")),
			readme("vfox-zig", text("Plugin README", "插件说明")),
		},
	}
}
