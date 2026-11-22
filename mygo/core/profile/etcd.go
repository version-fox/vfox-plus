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

// The profile of vfox-etcd (https://github.com/version-fox/vfox-etcd), written
// from its code at release 0.2.2.

// etcdProfile is the profile of vfox-etcd. The plugin takes full release names,
// with or without a v, and no partial name. latest is the first line of the list,
// which is read from the release through a CDN. BUILD_WITH_FAILPOINT is a tip, not
// a variable, because only some releases build with failpoints.
func etcdProfile() Profile {
	return Profile{
		Plugin: "etcd",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The first line of the list, the newest release", "列表第一行，即最新的发行版")},
		},
		Guide: text(
			"Write the version as the release names it, such as 3.7.3. A v in front, as in v3.5.12, also works. A partial version such as 3.5 is refused.",
			"按发行版的名称写版本，如 3.7.3。前面加 v（如 v3.5.12）也可以。3.5 这样的部分版本号会被拒绝。",
		),
		Tips: []Text{
			text(
				"Releases before 3.4.0 that publish no checksum file, such as 0.3.0, are checked against checksums that the plugin pins, and the install says so. A release from 3.4.0 on without a checksum file is refused.",
				"没有校验和文件的 3.4.0 之前的发行版（如 0.3.0），会用插件固定的校验和检查，安装时会说明这一点。3.4.0 及以后没有校验和文件的发行版会被拒绝。",
			),
			text(
				"The version list comes from the plugin's release through the jsDelivr CDN. A search fails when the CDN cannot be reached, unless vfox still has the list from an earlier search.",
				"版本列表经 jsDelivr CDN 从插件的发行版读取。CDN 无法访问时搜索会失败，除非 vfox 仍保留着之前搜索得到的列表。",
			),
			text(
				"BUILD_WITH_FAILPOINT=yes, set in Settings under Environment variables, builds the version you ask for from source with failpoints. It needs git, go and GNU make on the PATH, and it takes longer than an install from a release. It was checked on Linux only: 3.5.12 builds with failpoints, and 3.5.5 fails. Check the result with GOFAIL_HTTP. Only yes turns the build on: 1 and YES are refused, and no or an empty value installs the official build.",
				"在“设置”的“环境变量”中设置 BUILD_WITH_FAILPOINT=yes，会按所选版本从源码构建，并启用 failpoint。需要 PATH 上有 git、go 和 GNU make，并且比安装发行版耗时更长。仅在 Linux 上检查过：3.5.12 能构建出带 failpoint 的版本，3.5.5 会失败。请用 GOFAIL_HTTP 检查结果。只有值 yes 会开启该构建：1 与 YES 会被拒绝，no 或留空则安装官方构建。",
			),
		},
		Docs: []Link{
			website("vfox-etcd", text("Documentation website", "文档网站")),
			docs("vfox-etcd", "versions", text("Versions", "版本")),
			docs("vfox-etcd", "platforms", text("Platforms", "平台")),
			docs("vfox-etcd", "failpoint", text("Failpoint builds", "failpoint 构建")),
			docs("vfox-etcd", "link", text("Linking installed versions", "链接已安装的版本")),
			docs("vfox-etcd", "troubleshooting", text("Troubleshooting", "问题排查")),
			readme("vfox-etcd", text("Plugin README", "插件说明")),
		},
	}
}
