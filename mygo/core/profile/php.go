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

// The profile of vfox-php (https://github.com/version-fox/vfox-php), written
// from its documentation at release 0.3.3.

// phpProfile is the profile of vfox-php. Its platforms differ in what they
// build. On Linux and macOS PHP is compiled from the source release, so the
// build switches are read by the install and are offered there. On Windows the
// plugin unpacks prebuilt builds, and the Windows list marks each release as TS
// or NTS, which is the filter offered there.
func phpProfile(platform Platform) Profile {
	var variables []Variable
	var facets []Facet
	var tips []Text
	switch platform.OS {
	case "linux", "darwin":
		variables = []Variable{
			{
				Key:   "PHP_WITHOUT_PEAR",
				Label: text("PEAR", "PEAR"),
				Help: text(
					"Whether a PHP built from source includes PEAR, which provides the pear and pecl commands. Unset or no includes it; yes builds without it.",
					"从源码构建的 PHP 是否包含 PEAR，即 pear 与 pecl 命令。未设置或为 no 时包含；为 yes 时不包含。"),
				Kind: VariableChoice,
				Presets: []Preset{
					{Value: "no", Name: text("With PEAR", "包含 PEAR")},
					{Value: "yes", Name: text("Without PEAR", "不包含 PEAR")},
				},
			},
			{
				Key:   "PHP_WITHOUT_PCRE_JIT",
				Label: text("PCRE JIT", "PCRE JIT"),
				Help: text(
					"Whether a PHP built from source includes the PCRE JIT, which PHP has on by default. Unset or no keeps it on; yes builds without it.",
					"从源码构建的 PHP 是否包含 PCRE JIT（PHP 默认开启）。未设置或为 no 时保持开启；为 yes 时不包含。"),
				Kind: VariableChoice,
				Presets: []Preset{
					{Value: "no", Name: text("JIT on, the default", "开启 JIT，即默认值")},
					{Value: "yes", Name: text("JIT off", "关闭 JIT")},
				},
			},
		}
		tips = []Text{
			text(
				"On Linux and macOS PHP is compiled from source, which takes a while.",
				"在 Linux 和 macOS 上 PHP 从源码编译，需要一段时间。"),
			text(
				"Missing packages are named before compiling.",
				"编译前会列出缺少的软件包。"),
			text(
				"Set PHP_CONFIGURE_OPTIONS and MAKEFLAGS in Settings, under Environment variables.",
				"PHP_CONFIGURE_OPTIONS 与 MAKEFLAGS 接受任意文本：在“设置”的“环境变量”中以 KEY=VALUE 的形式设置。"),
		}
	case "windows":
		// The Linux and macOS lists say nothing about the build of a release,
		// so there is nothing to filter by there.
		facets = []Facet{{ID: "build", Label: text("Build", "构建"), Kind: FacetNote, Values: []string{"TS", "NTS"}}}
		tips = []Text{
			text(
				"On Windows the install adds the Visual C++ 2015-2022 runtime when it is missing, after Windows asks for permission.",
				"在 Windows 上，缺少 Visual C++ 2015-2022 运行库时，安装会在 Windows 请求授权后将其添加。"),
		}
	}
	tips = append(tips,
		text(
			"php@8.4 skips -zts and -nts builds.",
			"php@8.4 不会选中 -zts 或 -nts 构建。"),
		text(
			"Detect and link needs vfox 1.1.0 or later.",
			"探测并链接需要 vfox 1.1.0 或更高版本。"),
		text(
			"Composer comes with every install.",
			"每次安装都自带 Composer。"),
	)
	return Profile{
		Plugin: "php",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest PHP release in the list; on Windows its thread-safe build", "列表中最新的 PHP 发行版；在 Windows 上为线程安全版")},
		},
		Variables: variables,
		Facets:    facets,
		Guide: text(
			"Write the release as vfox search php lists it, such as 8.5.12. A line such as 8.4 installs its newest release. On Linux and macOS, -zts names the thread-safe build of a release. On Windows a plain version is the thread-safe build, and -nts names the non-thread-safe one.",
			"写 vfox search php 列出的发行版，如 8.5.12。8.4 这样的版本线会安装其中最新的版本。在 Linux 和 macOS 上，-zts 指某个发行版的线程安全构建。在 Windows 上，不带后缀的版本是线程安全构建，-nts 指非线程安全构建。"),
		Tips: tips,
		Docs: []Link{
			docs("vfox-php", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-php", "configuration", text("Build options", "构建选项")),
			docs("vfox-php", "platforms", text("Platforms", "平台")),
			docs("vfox-php", "link", text("Use a PHP that is already installed", "使用本机已有的 PHP")),
			docs("vfox-php", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
