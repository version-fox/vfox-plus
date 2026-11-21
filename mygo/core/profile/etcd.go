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
// from its code at release 0.2.1, which its main branch shares.

// etcdProfile is the profile of vfox-etcd. The plugin takes full release names,
// with or without a v, and no partial name. latest is not offered: the version
// list is read from the main branch through a CDN, which can keep an older copy
// that lists a release without a download. BUILD_WITH_FAILPOINT is a tip, not a
// variable, because only some releases build with failpoints.
func etcdProfile() Profile {
	return Profile{
		Plugin: "etcd",
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
				"The version list comes from the plugin's main branch through a CDN, which can keep an older copy for a while. An older copy can list a release that has no linux amd64 download, such as 0.1.0, and installing that release then fails.",
				"版本列表经 CDN 从插件的 main 分支读取，CDN 可能会保留旧副本一段时间。旧副本可能列出没有 Linux amd64 下载文件的发行版（如 0.1.0），安装它会失败。",
			),
			text(
				"BUILD_WITH_FAILPOINT=yes, set in Settings under Environment variables, builds the version you ask for from source with failpoints. It needs git, go and GNU make on the PATH, and it takes a few minutes. It was checked on Linux only, and only 3.4.32 and 3.5.12 were checked with failpoints: 3.5.5 fails, and 3.6.0 finishes without failpoints. Check the result with GOFAIL_HTTP.",
				"在“设置”的“环境变量”中设置 BUILD_WITH_FAILPOINT=yes，会按所选版本从源码构建，并启用 failpoint。需要 PATH 上有 git、go 和 GNU make，构建需要几分钟。仅在 Linux 上检查过，且只检查过 3.4.32 和 3.5.12 的 failpoint 构建：3.5.5 会失败，3.6.0 构建完成但没有 failpoint。请用 GOFAIL_HTTP 检查结果。",
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
