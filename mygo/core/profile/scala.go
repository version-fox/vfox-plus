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

// The profile of vfox-scala (https://github.com/version-fox/vfox-scala), written
// from its code at release 0.1.2, which its main branch shares.

// scalaProfile is the profile of vfox-scala. Scala 3 and Scala 2.10.5 and later
// come from GitHub, and the releases up to 2.10.4 come from scala-lang.org; the
// tips say so. The install reads no variable, and link detection reads
// SCALA_HOME, which the profile does not offer.
func scalaProfile() Profile {
	return Profile{
		Plugin: "scala",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The first version in the list, the newest release", "列表中的第一个版本，即最新的发行版")},
		},
		Guide: text(
			"Write the version as scala-lang.org lists it, such as 3.10.1. latest installs the first version in the list; a partial name such as 3 or 2.13 is not a version the plugin takes.",
			"按 scala-lang.org 列出的样子写版本，如 3.10.1。latest 安装列表中的第一个版本；3 或 2.13 这样的部分版本号不是插件接受的版本。",
		),
		Tips: []Text{
			text(
				"Scala 2.10.4 and older download from scala-lang.org; Scala 2.10.5 and later, and all of Scala 3, download from GitHub.",
				"Scala 2.10.4 及更早的版本从 scala-lang.org 下载；Scala 2.10.5 及之后的版本，以及全部 Scala 3，从 GitHub 下载。"),
			text(
				"The LTS release 3.9.0 installs as 3.9.0, not under the name the list shows.",
				"LTS 版本 3.9.0 以 3.9.0 安装，而不是列表中显示的名称。"),
			text(
				"The plugin installs no JDK, and Scala needs one to run. Scala 2.10.4 did not run on the JDK 25 that was checked.",
				"插件不安装 JDK，而 Scala 运行需要 JDK。检查时，Scala 2.10.4 无法在所检查的 JDK 25 上运行。"),
		},
		Docs: []Link{
			readme("vfox-scala", text("Plugin README", "插件说明")),
		},
	}
}
