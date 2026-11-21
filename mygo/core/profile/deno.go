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

// The profile of vfox-deno (https://github.com/version-fox/vfox-deno), written
// from its code at release 0.2.1, which its main branch shares.

// denoProfile is the profile of vfox-deno. The plugin reads no variable, keeps
// no notes and takes no search argument: the profile is the name latest, the
// guide and the tips.
func denoProfile() Profile {
	return Profile{
		Plugin: "deno",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The first release in the list, which is the newest stable release", "列表中的第一个版本，即最新的稳定版")},
		},
		Guide: text(
			"Write the release as the list shows it, such as 2.9.8. A line such as 2 or 2.1 installs the newest release of that line.",
			"按列表中的样子写版本，如 2.9.8。2 或 2.1 这样的版本线会安装该线上最新的发行版。",
		),
		Tips: []Text{
			text(
				"Releases before 0.36.0 have no download for this plugin, and they fail to install.",
				"0.36.0 之前的发行版没有本插件可下载的文件，安装会失败。"),
			text(
				"The plugin gives no checksum, so vfox installs the archive without checking it.",
				"插件不提供校验和，vfox 安装时不校验安装包。"),
		},
		Docs: []Link{
			readme("vfox-deno", text("Plugin README", "插件说明")),
		},
	}
}
