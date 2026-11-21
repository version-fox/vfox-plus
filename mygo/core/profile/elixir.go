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
// written from its code at release 1.0.0, the version that vfox add installs.

// elixirProfile is the profile of vfox-elixir. The plugin builds Elixir from its
// source release, or installs a pre-built build from hex.pm when
// VFOX_ELIXIR_MIRROR is hex. Both need Erlang/OTP, which the plugin does not
// install, and the source build finds only an erlc that is on PATH. The variable
// is offered on Linux only: the other platforms were not checked, and on Windows
// the installer runs before the hex check.
func elixirProfile(platform Platform) Profile {
	var variables []Variable
	if platform.OS == "linux" {
		variables = []Variable{
			{
				Key:   "VFOX_ELIXIR_MIRROR",
				Label: text("Pre-built builds", "预编译构建"),
				Help: text(
					"Installs Elixir from the pre-built builds of hex.pm instead of compiling its source, and lists those builds. Empty compiles from the source.",
					"用 hex.pm 的预编译构建安装 Elixir，而不是编译源码，并列出这些构建。留空则从源码编译。"),
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
			{Name: "latest", Meaning: text("The first line of the list: 1.20-latest on Linux, which ran as 1.20.5 when checked; with pre-built builds, main, which is not a release", "列表第一行：Linux 上是 1.20-latest，检查时它运行为 1.20.5；使用预编译构建时是 main，那不是发行版")},
		},
		Variables: variables,
		Guide: text(
			"Write the version as the list shows it, such as 1.20.5. Elixir needs Erlang/OTP to build and to run, and vfox does not install it, so install Erlang/OTP first.",
			"按列表中的样子写版本，如 1.20.5。Elixir 编译和运行都需要 Erlang/OTP，vfox 不会安装它，所以请先安装 Erlang/OTP。",
		),
		Tips: []Text{
			text(
				"vfox does not put an installed Erlang/OTP on the PATH of an Elixir build, so a source build can fail with erlc: No such file or directory.",
				"vfox 不会把已安装的 Erlang/OTP 加入 Elixir 构建时的 PATH，所以源码构建可能以 erlc: No such file or directory 失败。"),
			text(
				"A source build takes about a minute and needs a HOME path without spaces. A pre-built build works with such a path.",
				"源码构建约需一分钟，并且要求家目录路径中不能有空格。预编译构建不受此限制。"),
			text(
				"Running elixir alone fails with exec: erl: not found, so run it together with Erlang/OTP.",
				"单独运行 elixir 会以 exec: erl: not found 失败，所以请与 Erlang/OTP 一起运行。"),
			text(
				"Partial names such as 1.16 are not versions the plugin takes.",
				"1.16 这样的部分版本号不是插件接受的版本。"),
		},
		Docs: []Link{
			website("vfox-elixir", text("Documentation website", "文档网站")),
			readme("vfox-elixir", text("Plugin README", "插件说明")),
		},
	}
}
