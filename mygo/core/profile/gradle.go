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

// The profile of vfox-gradle (https://github.com/version-fox/vfox-gradle),
// written from its documentation at release 0.1.3.

func gradleProfile() Profile {
	return Profile{
		Plugin: "gradle",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest release that gradle.org lists", "gradle.org 上列出的最新版本")},
		},
		Variables: []Variable{
			{
				Key:   "VFOX_GRADLE_MIRRORS",
				Label: text("Download mirror", "下载镜像"),
				Help: text(
					"A mirror of services.gradle.org/distributions: the address that gradle-<version>-bin.zip sits under. The version list itself always comes from gradle.org.",
					"services.gradle.org/distributions 的镜像：gradle-<版本>-bin.zip 所在的地址。版本列表始终来自 gradle.org。"),
				Kind:    VariableURL,
				Example: "https://repo.huaweicloud.com/gradle",
				Presets: []Preset{
					{Value: "https://repo.huaweicloud.com/gradle", Name: text("Huawei Cloud", "华为云")},
					{Value: "https://mirror.nju.edu.cn/gradle", Name: text("Nanjing University", "南京大学")},
					{Value: "https://mirrors.cernet.edu.cn/gradle", Name: text("CERNET", "中国教育和科研计算机网（CERNET）")},
				},
			},
		},
		LegacyFiles: []string{"gradle/wrapper/gradle-wrapper.properties"},
		Guide: text(
			"A version is a release as vfox search gradle lists it: 8.7, 8.14.3, 9.0.0. A bare major such as 8 is not a release and does not install; latest installs the newest release. Copy the name from the list.",
			"版本就是 vfox search gradle 列出的发行版名称：8.7、8.14.3、9.0.0。单独的主版本号（如 8）不是发行版，无法安装；latest 会安装最新版。请从列表里复制名称。"),
		Tips: []Text{
			text(
				"Gradle runs on a JVM, so install a JDK first: vfox add java, then java@21-tem.",
				"Gradle 需要在 JVM 上运行，请先装 JDK：vfox add java，再装 java@21-tem。"),
			text(
				"Release candidates install by name (9.0.0-rc-1) but are never listed and never latest.",
				"候选版可以通过名称安装（如 9.0.0-rc-1），但不会出现在列表里，也不会成为 latest。"),
			text(
				"A mirror without a .sha256 still installs: the checksum is then read from services.gradle.org.",
				"没有 .sha256 文件的镜像仍可安装：此时校验和从 services.gradle.org 读取。"),
			text(
				"When a Gradle is already on the machine, adopt it with vfox link --detect gradle; wrapper distributions in ~/.gradle are found as well.",
				"如果机器上已经有 Gradle，用 vfox link --detect gradle 纳入管理；~/.gradle 下的 wrapper 发行版也能找到。"),
			text(
				"If another Gradle shadows the one of vfox, vfox doctor explains which one it is; a project wrapper still wins through gradle/wrapper/gradle-wrapper.properties.",
				"如果另一个 Gradle 盖住了 vfox 的版本，vfox doctor 会说明是哪一个；项目的 wrapper 仍通过 gradle/wrapper/gradle-wrapper.properties 生效。"),
		},
		Docs: []Link{
			docs("vfox-gradle", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-gradle", "mirror", text("Mirror", "镜像")),
			docs("vfox-gradle", "java-toolchains", plain("Java toolchains")),
			docs("vfox-gradle", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
