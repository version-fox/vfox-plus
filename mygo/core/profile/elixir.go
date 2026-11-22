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

// The profile of vfox-elixir (https://github.com/version-fox/vfox-elixir),
// written from its code at release 1.0.3.

// elixirProfile is the profile of vfox-elixir. The plugin builds Elixir from its
// source release, or installs a pre-built build from hex.pm when
// VFOX_ELIXIR_MIRROR is hex. Both need Erlang/OTP, which the plugin does not
// install. A source build uses the erlc of an Erlang/OTP that vfox installed when
// erlc is not on PATH, and refuses a VFOX_HOME path with a space or another shell
// character. With hex, latest stops before the download, and the list has only
// names with an -otp-N suffix: a name without one may not run under another
// Erlang/OTP. The variable is offered on Linux only: the other platforms were not
// checked.
func elixirProfile(platform Platform) Profile {
	var variables []Variable
	if platform.OS == "linux" {
		variables = []Variable{
			{
				Key:   "VFOX_ELIXIR_MIRROR",
				Label: text("Pre-built builds", "预编译构建"),
				Help: text(
					"Installs Elixir from the pre-built builds of hex.pm instead of compiling its source. Empty compiles from the source. The builds are listed by vfox search elixir hex, and their names carry the Erlang/OTP they need, such as 1.20.5-otp-29.",
					"用 hex.pm 的预编译构建安装 Elixir，而不是编译源码。留空则从源码编译。预编译构建可用 vfox search elixir hex 列出，名称中带有它需要的 Erlang/OTP 版本，如 1.20.5-otp-29。"),
				Kind: VariableChoice,
				Presets: []Preset{
					{Value: "hex", Name: text("Pre-built builds from hex.pm", "来自 hex.pm 的预编译构建")},
				},
			},
		}
	}
	return Profile{
		Plugin: "elixir",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The first line of the list: 1.20-latest on Linux, which ran as 1.20.5 when checked. Latest stops before the download with pre-built builds", "列表第一行：Linux 上是 1.20-latest，检查时它运行为 1.20.5。使用预编译构建时 latest 会在下载前停止")},
		},
		Variables: variables,
		Guide: text(
			"Write the version as the list shows it, such as 1.20.5, or with pre-built builds a name such as 1.20.5-otp-29. Elixir needs Erlang/OTP to build and to run, and vfox does not install it, so install Erlang/OTP first, for example with the erlang plugin.",
			"按列表中的样子写版本，如 1.20.5；使用预编译构建时写 1.20.5-otp-29 这样的名称。Elixir 编译和运行都需要 Erlang/OTP，vfox 不会安装它，所以请先安装 Erlang/OTP，例如用 erlang 插件安装。",
		),
		Tips: []Text{
			text(
				"A source build uses the erlc of an Erlang/OTP that vfox installed when erlc is not on PATH. Without any Erlang/OTP, the install stops before the build and says so.",
				"源码构建在 PATH 上没有 erlc 时，会使用 vfox 已安装的 Erlang/OTP 的 erlc。没有任何 Erlang/OTP 时，安装会在构建前停止并说明。"),
			text(
				"A source build takes about a minute. The path of VFOX_HOME must not hold a space or another shell character, such as a quote, a dollar sign or an ampersand; the install then stops before make and says so. A pre-built build works with such a path.",
				"源码构建约需一分钟。VFOX_HOME 的路径中不能有空格，也不能有引号、美元符号或 & 这样的 shell 字符；否则安装会在 make 之前停止并说明。预编译构建不受此限制。"),
			text(
				"Running elixir alone fails with exec: erl: not found, so run it together with Erlang/OTP, for example vfox exec erlang@29.1.2 elixir@1.20.5 -- elixir --version.",
				"单独运行 elixir 会以 exec: erl: not found 失败，所以请与 Erlang/OTP 一起运行，例如 vfox exec erlang@29.1.2 elixir@1.20.5 -- elixir --version。"),
			text(
				"For a source build, a name such as 1.16 is not a release, because its tag does not exist. The pre-built list shows only names with an -otp-N suffix, such as 1.20.5-otp-29. A name without one, such as 1.20.5, is not listed; if you type it, the install warns that the build may not run with your Erlang/OTP, so prefer a name with its suffix.",
				"对源码构建而言，1.16 这样的名称不是发行版，因为它的标签不存在。预编译列表只显示带 -otp-N 后缀的名称，如 1.20.5-otp-29；不带后缀的名称（如 1.20.5）不会列出，如果手动输入，安装会警告这个构建可能无法用你的 Erlang/OTP 运行，所以请优先使用带后缀的名称。"),
		},
		Docs: []Link{
			website("vfox-elixir", text("Documentation website", "文档网站")),
			readme("vfox-elixir", text("Plugin README", "插件说明")),
		},
	}
}
