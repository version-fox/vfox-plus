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
// from its code at release 0.2.5.

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
				"Older releases use pinned checksums.",
				"更早的发行版使用插件固定的校验和。",
			),
			text(
				"Releases from 3.4.0 need a checksum file.",
				"3.4.0 及以后的发行版需要校验和文件。",
			),
			text(
				"The list comes from the jsDelivr CDN.",
				"版本列表来自 jsDelivr CDN。",
			),
			text(
				"BUILD_WITH_FAILPOINT=yes builds with failpoints.",
				"BUILD_WITH_FAILPOINT=yes 会从源码构建带 failpoint 的版本。",
			),
			text(
				"Only yes turns the build on.",
				"只有 yes 才会开启该构建。",
			),
			text(
				"The build needs git, go and make.",
				"该构建需要 git、go 与 make。",
			),
			text(
				"Releases without a gofail target are refused.",
				"没有 gofail 目标的发行版会被拒绝。",
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
