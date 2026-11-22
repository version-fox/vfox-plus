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
// written from its code and its documents at release 0.1.2, which its main
// branch shares.

// groovyProfile is the profile of vfox-groovy. The plugin sorts a pre-release
// below the release that it precedes, so the first line of the list is the
// newest release and latest installs it. The plugin reads one variable,
// VFOX_GROOVY_MIRROR, for a mirror of the Apache archive. The profile offers
// dlcdn.apache.org only: the Tsinghua mirror lists the releases, but it refused
// the SDK downloads of vfox's client with HTTP 403. The notes of the plugin mark
// the pre-releases and the incubator releases, which the stage filter picks.
func groovyProfile() Profile {
	return Profile{
		Plugin: "groovy",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest release in the list. A pre-release sorts below the release it precedes", "列表中最新的正式版。预发布版排在它所属的正式版之后")},
		},
		Variables: []Variable{
			{
				Key:   "VFOX_GROOVY_MIRROR",
				Label: text("Archive mirror", "归档镜像"),
				Help: text(
					"Reads the archive from dlcdn.apache.org, another copy of the Apache archive that keeps the same layout, for when archive.apache.org is slow. Empty reads archive.apache.org.",
					"从 dlcdn.apache.org 读取，它是保持相同布局的 Apache 归档副本，适用于 archive.apache.org 较慢的情况。留空则读取 archive.apache.org。"),
				Kind:    VariableURL,
				Example: "https://dlcdn.apache.org/groovy",
				Presets: []Preset{
					{Value: "https://dlcdn.apache.org/groovy", Name: plain("dlcdn.apache.org")},
				},
			},
		},
		Facets: []Facet{
			{ID: "stage", Label: text("Stage", "阶段"), Kind: FacetNote, Values: []string{"pre-release", "incubating"}},
		},
		Guide: text(
			"Write the version as the list shows it, such as 4.0.33 or 6.0.0-RC-2. latest installs the newest release. A partial name such as 4 or 6 is not a version the plugin takes.",
			"按列表中的样子写版本，如 4.0.33 或 6.0.0-RC-2。latest 安装最新的正式版。4 或 6 这样的部分版本号不是插件接受的版本。",
		),
		Tips: []Text{
			text(
				"The plugin installs no JDK, and Groovy needs one to run.",
				"插件不安装 JDK，而 Groovy 运行需要 JDK。"),
			text(
				"With dlcdn.apache.org, the list holds only the current releases, so older versions do not show.",
				"使用 dlcdn.apache.org 时，列表只有当前的发行版，较旧的版本不会出现。"),
			text(
				"A release without a .sha256 file installs without a checksum check, and vfox says so.",
				"没有 .sha256 文件的发行版安装时不做校验和检查，vfox 会说明这一点。"),
		},
		Docs: []Link{
			docs("vfox-groovy", "versions", text("Versions", "版本")),
			docs("vfox-groovy", "mirror", text("Mirror", "镜像")),
			docs("vfox-groovy", "platforms", text("Platforms", "平台")),
			docs("vfox-groovy", "link", text("Use a Groovy that is already installed", "使用本机已有的 Groovy")),
			docs("vfox-groovy", "troubleshooting", text("Troubleshooting", "问题排查")),
			readme("vfox-groovy", text("Plugin README", "插件说明")),
		},
	}
}
