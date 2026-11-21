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
// from its code at release 0.2.0, which its main branch shares.

// etcdProfile is the profile of vfox-etcd. The plugin reads BUILD_WITH_FAILPOINT,
// which builds the main branch of etcd whatever version is chosen, so it is not
// offered as a variable, and the tips name it. The plugin takes no shorthand:
// latest is not a name it resolves.
func etcdProfile() Profile {
	return Profile{
		Plugin: "etcd",
		Guide: text(
			"Write the version as the release names it, such as 3.7.3, without a v.",
			"按发行版的名称写版本，如 3.7.3，不带 v。",
		),
		Tips: []Text{
			text(
				"Some listed versions have no download for Linux on amd64, such as 0.1.0, and fail to install.",
				"列表中有些版本没有 Linux amd64 的下载文件，如 0.1.0，安装会失败。"),
			text(
				"A release without a SHA256SUMS file installs without a checksum check, such as 0.3.0.",
				"没有 SHA256SUMS 文件的发行版（如 0.3.0）安装时不校验校验和。"),
			text(
				"BUILD_WITH_FAILPOINT, set in Settings under Environment variables, builds the main branch of etcd whatever version is chosen.",
				"在“设置”的“环境变量”中设置 BUILD_WITH_FAILPOINT，会构建 etcd 的 main 分支，与所选版本无关。"),
		},
		Docs: []Link{
			website("vfox-etcd", text("Documentation website", "文档网站")),
			readme("vfox-etcd", text("Plugin README", "插件说明")),
		},
	}
}
