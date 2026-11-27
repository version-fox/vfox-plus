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
// from its code and its documents at release 0.2.3, which its main branch shares.

// denoProfile is the profile of vfox-deno. The plugin lists only the releases
// that have a download for the platform, so the list starts at 0.36.0 on Linux
// with x86-64, and the first line is the newest stable release, which latest
// installs. The plugin reads no variable, keeps no notes and takes no search
// argument. The plugin sets the execute bit after it unpacks an archive, so
// release 0.36.0 runs; a 0.36.0 installed by an older plugin needs a reinstall,
// and the tips say so.
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
				"The list starts at 0.36.0, which is the first release with a download for Linux on x86-64.",
				"列表从 0.36.0 开始，它是 Linux x86-64 上第一个有下载文件的发行版。"),
			text(
				"The plugin checks a release against its .sha256sum file when Deno publishes one, and says so when it does not.",
				"发行版有 .sha256sum 文件时，插件会据此校验；没有时，插件会说明未校验。"),
			text(
				"The plugin sets the execute bit on Deno after it unpacks an archive, so 0.36.0 runs. A 0.36.0 installed by a plugin before 0.2.3 has no execute bit: reinstall it with vfox uninstall and vfox install.",
				"插件在解压后为 Deno 设置可执行位，所以 0.36.0 可以运行。0.2.3 之前的插件安装的 0.36.0 没有可执行位：请用 vfox uninstall 与 vfox install 重新安装。"),
		},
		Docs: []Link{
			docs("vfox-deno", "versions", text("Versions", "版本")),
			docs("vfox-deno", "platforms", text("Platforms", "平台")),
			docs("vfox-deno", "link", text("Use a Deno that is already installed", "使用本机已有的 Deno")),
			docs("vfox-deno", "troubleshooting", text("Troubleshooting", "问题排查")),
			readme("vfox-deno", text("Plugin README", "插件说明")),
		},
	}
}
