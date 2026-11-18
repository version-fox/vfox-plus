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

// The profile of vfox-kotlin (https://github.com/version-fox/vfox-kotlin),
// written from its documentation at release 0.1.2, and checked against the list
// that the release gives and the archives that its PreInstall finds.

// kotlinProfile is the profile of vfox-kotlin. The plugin lists the rows of the
// release table of the Kotlin site, and at release 0.1.2 the first of them is
// 2.4, a line that is no release: latest, which vfox resolves to the first line
// of the list, fails with "Current version information not detected", as 2.4
// does, while 2.4.0 and 2.4.21 find their archives. The profile says so where
// latest is offered.
func kotlinProfile() Profile {
	return Profile{
		Plugin: "kotlin",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The first line of the list, which can be a line such as 2.4 that is no release", "列表的第一行，可能是 2.4 这样不是发行版的行")},
		},
		Guide: text(
			"Write a Kotlin release in full, such as 2.1.0 or 1.9.24. There are no distributions or suffixes: the version you type is the compiler you get. A line such as 2.4 is no release, and latest takes the first line of the list, which can be one: the install then fails, so write the release.",
			"完整写出 Kotlin 的发行版号，如 2.1.0 或 1.9.24。没有发行版，也没有后缀：输入什么版本就得到什么编译器。2.4 这样的行不是发行版，而 latest 取列表的第一行，它可能正是这样的行：此时安装会失败，请改写发行版号。"),
		Tips: []Text{
			text(
				"kotlinc and kotlinc-native are JVM launchers: a Java runtime 17 or newer must be on PATH. The plugin does not install Java.",
				"kotlinc 与 kotlinc-native 都是 JVM 启动器：PATH 上必须有 Java 17 或更高版本的运行时。插件不会安装 Java。"),
			text(
				"Kotlin/Native is bundled on hosts JetBrains publishes a prebuilt compiler for; other hosts, such as Linux arm64, get the JVM compiler only.",
				"在 JetBrains 提供预编译编译器的平台上会一并安装 Kotlin/Native；其他平台（如 Linux arm64）只安装 JVM 编译器。"),
			text(
				"Old releases have no checksum.",
				"1.9.0 之前的版本没有校验和，vfox 会提示跳过。"),
			text(
				"Detect and link adopts compilers from SDKMAN, asdf, mise, Homebrew, IDE bundles, system directories and KOTLIN_HOME, and needs vfox 1.1.0 or later.",
				"探测并链接可纳入 SDKMAN、asdf、mise、Homebrew、IDE 自带、系统目录以及 KOTLIN_HOME 中的编译器，需要 vfox 1.1.0 或更高版本。"),
		},
		Docs: []Link{
			docs("vfox-kotlin", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-kotlin", "link", text("Use a compiler that is already installed", "使用本机已有的编译器")),
			docs("vfox-kotlin", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
