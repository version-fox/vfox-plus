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

// The profile of vfox-maven (https://github.com/version-fox/vfox-maven),
// written from its documentation at release 0.1.3.

func mavenProfile() Profile {
	return Profile{
		Plugin: "maven",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest stable release in the list; a pre-release is installed by its name", "列表中最新的稳定版；预发布版要按名称安装")},
		},
		Guide: text(
			"Write the release as vfox search maven lists it: 3.10.1, 3.0.5, 4.0.0-rc-7. The list holds Maven 1 to 4, the old releases too. latest installs the newest stable release, and the pre-releases come after the stable ones.",
			"写 vfox search maven 列出的版本，如 3.10.1、3.0.5、4.0.0-rc-7。列表包含 Maven 1 到 4，也包括旧版本。latest 安装最新的稳定版；预发布版排在稳定版之后。"),
		Tips: []Text{
			text(
				"Maven needs a JDK: put one on PATH or set JAVA_HOME.",
				"Maven 需要 JDK：把 JDK 放进 PATH 或设置 JAVA_HOME。"),
			text(
				"Install one, such as java@21-tem.",
				"可安装一个，如 java@21-tem。"),
			text(
				"Maven 1.1 needs JDK 8, not JRE.",
				"Maven 1.1 需要 JDK 8，不能用 JRE。"),
			text(
				"vfox use maven@4 picks the newest installed 4.x release.",
				"vfox use maven@4 会选中已安装的最新 4.x 版本。"),
			text(
				"Detect and link adopts Mavens already installed.",
				"探测并链接可纳入已安装的 Maven。"),
		},
		Docs: []Link{
			docs("vfox-maven", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-maven", "link", text("Use a Maven that is already installed", "使用本机已有的 Maven")),
		},
	}
}
