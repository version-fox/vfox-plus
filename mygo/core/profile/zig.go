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
				"The list needs ziglang.org: when it cannot be reached, the whole list fails. machengine.org is optional: without it the list shows the official releases, with a warning.",
				"列表需要 ziglang.org：它无法访问时，整个列表都会失败。machengine.org 是可选的：无法访问时，列表只显示官方发行版，并给出警告。",
			),
			text(
				"A name that the list does not show is refused, and nothing is installed.",
				"列表中没有的名称会被拒绝，什么也不会安装。",
			),
			text(
				"vfox 1.1.0 exec does not take latest, nightly or a Mach name. Give the version name that vfox list shows, such as 0.18.0-dev.1272+2ff397f75. vfox use takes these names.",
				"vfox 1.1.0 的 exec 不接受 latest、nightly 或 Mach 名称。请用 vfox list 显示的版本名，如 0.18.0-dev.1272+2ff397f75。vfox use 可以接受这些名称。",
			),
			text(
				"A Mach build that ziglang.org does not publish, such as 2024.5.0-mach, is downloaded from pkg.hexops.org, and its checksum from the index is still checked.",
				"ziglang.org 没有发布的 Mach 构建（如 2024.5.0-mach）从 pkg.hexops.org 下载，仍然按索引中的校验和检查。",
			),
			text(
				"Detect and link finds the Zig installations that are already on the machine, from asdf, mise, zvm, Scoop, the system directories and PATH, and links them instead of installing another copy. A zig that cannot run is not listed, and neither are the installs that vfox manages or anything under VFOX_HOME. It needs vfox 1.1.0 or later, and it was checked on Linux only.",
				"探测并链接会找出机器上已有的 Zig 安装，来自 asdf、mise、zvm、Scoop、系统目录与 PATH，并链接它们，而不再安装一份。无法运行的 zig 不会被列出，vfox 管理的安装与 VFOX_HOME 下的任何内容也不会被列出。需要 vfox 1.1.0 或更高版本，只在 Linux 上检查过。",
			),
		},
		Docs: []Link{
			docs("vfox-zig", "versions", text("Versions", "版本")),
			docs("vfox-zig", "link", text("Linking installed versions", "链接已安装的版本")),
			readme("vfox-zig", text("Plugin README", "插件说明")),
		},
	}
}
