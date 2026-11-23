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

// The profile of vfox-erlang (https://github.com/version-fox/vfox-erlang),
// written from its code at release 1.2.5, which its main branch shares.

// erlangProfile is the profile of vfox-erlang. With USE_PREBUILT_OTP set to an
// Ubuntu release, the plugin installs a prebuilt build and lists the prebuilt
// builds instead of the source releases; the names are the release numbers, and
// OTP-29.1.2 also installs the prebuilt build of that release. The variable is
// offered on Linux only: the prebuilt builds of macOS were not checked, and the
// plugin ignores the variable on Windows. OTP_CONFIGURE_ARGS and MAKEFLAGS take
// free text, so the tips name them instead.
func erlangProfile(platform Platform) Profile {
	var variables []Variable
	if platform.OS == "linux" {
		variables = []Variable{
			{
				Key:   "USE_PREBUILT_OTP",
				Label: text("Prebuilt build", "预编译构建"),
				Help: text(
					"Installs Erlang/OTP from the prebuilt build of this Ubuntu release, instead of compiling it, and lists the prebuilt builds. Empty compiles from the source.",
					"用这个 Ubuntu 版本的预编译构建安装 Erlang/OTP，而不是编译源码，版本列表也改为预编译构建。留空则从源码编译。"),
				Kind: VariableChoice,
				Presets: []Preset{
					{Value: "ubuntu-20.04", Name: plain("Ubuntu 20.04")},
					{Value: "ubuntu-22.04", Name: plain("Ubuntu 22.04")},
					{Value: "ubuntu-24.04", Name: plain("Ubuntu 24.04")},
				},
			},
		}
	}
	return Profile{
		Plugin: "erlang",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest release in the list", "列表中最新的发行版")},
		},
		Variables: variables,
		Guide: text(
			"Write the release as the list shows it, such as 29.1.2. With a prebuilt build selected, OTP-29.1.2 also installs the prebuilt build of that release.",
			"按列表中的样子写版本，如 29.1.2。选择了预编译构建后，OTP-29.1.2 也会安装该发行版的预编译构建。",
		),
		Tips: []Text{
			text(
				"Building from source takes a long time and needs the tools that the plugin's source-build page lists.",
				"从源码编译需要很长时间，并需要插件源码编译页面中列出的编译工具。"),
			text(
				"OTP_CONFIGURE_ARGS, set in Settings under Environment variables, passes arguments to configure, such as --without-javac.",
				"在“设置”的“环境变量”中设置 OTP_CONFIGURE_ARGS，可把参数传给 configure，如 --without-javac。"),
			text(
				"MAKEFLAGS, set in Settings under Environment variables, runs make in parallel, such as -j8.",
				"在“设置”的“环境变量”中设置 MAKEFLAGS，可让 make 并行编译，如 -j8。"),
			text(
				"vfox keeps one search cache per plugin, and the cache does not depend on USE_PREBUILT_OTP. Turn the search cache off before you use prebuilt releases, as the plugin's prebuilt page says.",
				"vfox 为每个插件保留一份搜索缓存，而且缓存与 USE_PREBUILT_OTP 无关。使用预编译构建前请关闭搜索缓存，见插件的预编译页面。"),
			text(
				"A source build stops before configure, once it has downloaded the source, when the path of VFOX_HOME contains an apostrophe, a double quote, a dollar sign, a backtick, a space, a tab or a newline, and the install says which character. A prebuilt build works with a space in the path.",
				"源码编译在下载源码之后、configure 之前停止：VFOX_HOME 的路径含单引号、双引号、美元符号、反引号、空格、制表符或换行符时，安装会说明是哪个字符。预编译构建可以在含空格的路径下安装。"),
			text(
				"A prebuilt install is recorded under its release number, such as 29.1.2, so run it as erlang@29.1.2. vfox exec does not map an OTP- name to that install: erlang@OTP-29.1.2 is not found there, and without USE_PREBUILT_OTP vfox starts a source build of it.",
				"预编译安装按发行版号记录，如 29.1.2，所以请用 erlang@29.1.2 运行它。vfox exec 不会把 OTP- 名称映射到该安装：在 vfox exec 中写 erlang@OTP-29.1.2 会找不到它，未设置 USE_PREBUILT_OTP 时 vfox 会开始一次源码构建。"),
		},
		Docs: []Link{
			website("vfox-erlang", text("Documentation website", "文档网站")),
			docs("vfox-erlang", "prebuilt", text("Prebuilt builds", "预编译构建")),
			docs("vfox-erlang", "source-build", text("Building from source", "从源码编译")),
			readme("vfox-erlang", text("Plugin README", "插件说明")),
		},
	}
}
