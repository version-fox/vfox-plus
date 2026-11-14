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
			"Write the release as vfox search gradle lists it: 8.7, 8.14.3, 9.0.0. A bare major such as 8 does not install; latest installs the newest. Release candidates install by name (9.0.0-rc-1) but are never listed.",
			"写 vfox search gradle 列出的发行版名称：8.7、8.14.3、9.0.0。单独的主版本号（如 8）无法安装；latest 安装最新版。候选版可以按名称安装（如 9.0.0-rc-1），但不会出现在列表里。"),
		Tips: []Text{
			text(
				"Gradle runs on a JVM: install a JDK first (java@21-tem).",
				"Gradle 需要在 JVM 上运行，请先装 JDK（如 java@21-tem）。"),
		},
		Docs: []Link{
			docs("vfox-gradle", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-gradle", "mirror", text("Mirror", "镜像")),
			docs("vfox-gradle", "java-toolchains", plain("Java toolchains")),
			docs("vfox-gradle", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
