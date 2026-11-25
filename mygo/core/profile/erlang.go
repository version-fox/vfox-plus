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
// written from its code at release 1.2.8, which its main branch shares.

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
				"Source builds take a long time.",
				"源码编译需要很长时间。",
			),
			text(
				"Set OTP_CONFIGURE_ARGS and MAKEFLAGS in Settings.",
				"在“设置”中设置 OTP_CONFIGURE_ARGS 与 MAKEFLAGS。",
			),
			text(
				"Turn the search cache off first.",
				"先关闭搜索缓存，再使用预编译构建。",
			),
			text(
				"Source builds refuse some characters in VFOX_HOME.",
				"源码编译会拒绝 VFOX_HOME 中的某些字符。",
			),
			text(
				"Run a prebuilt install as erlang@29.1.2.",
				"预编译安装请用 erlang@29.1.2 运行。",
			),
			text(
				"vfox exec does not map OTP- names.",
				"vfox exec 不会映射 OTP- 名称。",
			),
		},
		Docs: []Link{
			website("vfox-erlang", text("Documentation website", "文档网站")),
			docs("vfox-erlang", "prebuilt", text("Prebuilt builds", "预编译构建")),
			docs("vfox-erlang", "source-build", text("Building from source", "从源码编译")),
			readme("vfox-erlang", text("Plugin README", "插件说明")),
		},
	}
}
