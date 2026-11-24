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
// written from its code at release 1.0.6.

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
				"vfox does not install Erlang/OTP.",
				"vfox 不会安装 Erlang/OTP。",
			),
			text(
				"Source builds take about a minute.",
				"源码构建约需一分钟。",
			),
			text(
				"Source builds refuse spaces in VFOX_HOME.",
				"源码构建会拒绝 VFOX_HOME 中的空格。",
			),
			text(
				"Run elixir with erlang@29.1.2 in exec.",
				"在 exec 中与 erlang@29.1.2 一起运行 elixir。",
			),
			text(
				"Names such as 1.16 are not releases.",
				"1.16 这样的名称不是发行版。",
			),
			text(
				"Pre-built builds are checked against builds.txt.",
				"预编译构建会按 builds.txt 校验。",
			),
			text(
				"Builds without a SHA-256 are not checked.",
				"没有 SHA-256 的构建不做校验。",
			),
			text(
				"Detect and link finds existing installs.",
				"探测并链接会找出已有的安装。",
			),
		},
		Docs: []Link{
			website("vfox-elixir", text("Documentation website", "文档网站")),
			readme("vfox-elixir", text("Plugin README", "插件说明")),
		},
	}
}
