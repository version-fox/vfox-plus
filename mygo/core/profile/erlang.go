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
// written from its code at release 1.2.0, which its main branch shares.

// erlangProfile is the profile of vfox-erlang. With USE_PREBUILT_OTP set to an
// Ubuntu release, the plugin installs a prebuilt build and lists the prebuilt
// builds instead of the source releases. The variable is offered on Linux only:
// the prebuilt builds of macOS were not checked, and the plugin reads no variable
// on Windows. OTP_CONFIGURE_ARGS and MAKEFLAGS take free text, so the tips name
// them instead.
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
			{Name: "latest", Meaning: text("The newest release in the list; with a prebuilt build selected, the master branch", "列表中最新的发行版；选择了预编译构建时为 master 分支")},
		},
		Variables: variables,
		Guide: text(
			"Write the release as the list shows it, such as 29.1.2. With a prebuilt build selected, the names carry the OTP- prefix, such as OTP-29.1.2.",
			"按列表中的样子写版本，如 29.1.2。选择了预编译构建后，名称带 OTP- 前缀，如 OTP-29.1.2。",
		),
		Tips: []Text{
			text(
				"Building from source takes several minutes and needs the tools that the plugin's README lists.",
				"从源码编译需要几分钟，并需要插件 README 中列出的编译工具。"),
			text(
				"OTP_CONFIGURE_ARGS, set in Settings under Environment variables, passes arguments to configure, such as --without-javac.",
				"在“设置”的“环境变量”中设置 OTP_CONFIGURE_ARGS，可把参数传给 configure，如 --without-javac。"),
			text(
				"MAKEFLAGS, set in Settings under Environment variables, runs make in parallel, such as -j8.",
				"在“设置”的“环境变量”中设置 MAKEFLAGS，可让 make 并行编译，如 -j8。"),
			text(
				"The list of prebuilt builds may not show the newest ones.",
				"预编译构建的列表可能不显示最新的构建。"),
			text(
				"An install fails when the path of the home directory has a space.",
				"家目录的路径含空格时，安装会失败。"),
		},
		Docs: []Link{
			website("vfox-erlang", text("Documentation website", "文档网站")),
			readme("vfox-erlang", text("Plugin README", "插件说明")),
		},
	}
}
