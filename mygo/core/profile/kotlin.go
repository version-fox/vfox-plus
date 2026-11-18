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
// written from its documentation at release 0.1.4, and checked against the list
// that the release gives and the archives that its PreInstall finds.

// kotlinProfile is the profile of vfox-kotlin. Its list holds releases only, so
// latest, which vfox resolves to the first line of the list, is the newest
// release.
func kotlinProfile() Profile {
	return Profile{
		Plugin: "kotlin",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest release on the Kotlin release page", "Kotlin 发布页面上最新的版本")},
		},
		Guide: text(
			"Write a Kotlin release in full, such as 2.1.0 or 1.9.24. There are no distributions or suffixes: the version you type is the compiler you get. latest installs the newest release on the Kotlin release page; if that page cannot be read, search and install say why instead of guessing a version.",
			"完整写出 Kotlin 的发行版号，如 2.1.0 或 1.9.24。没有发行版，也没有后缀：输入什么版本就得到什么编译器。latest 安装 Kotlin 发布页面上最新的版本；若该页面无法读取，搜索与安装会报出原因，而不会猜测版本。"),
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
