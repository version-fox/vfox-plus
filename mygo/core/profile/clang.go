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

// The profile of vfox-clang (https://github.com/version-fox/vfox-clang),
// written from its code at release 0.5.3, which its main branch shares.

// clangProfile is the profile of vfox-clang. The plugin installs clang from
// conda-forge with pixi, and reads two variables: Conda_Forge, the channel that
// pixi takes clang from, and GITHUB_URL, the GitHub address of its version list
// and of the pixi download. Only Conda_Forge is offered, because only its
// mirrors were checked; GITHUB_URL is named in a tip.
func clangProfile() Profile {
	return Profile{
		Plugin: "clang",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest version in the list, which the list marks as latest", "列表中最新的版本，列表中以 latest 标出")},
		},
		Variables: []Variable{
			{
				Key:   "Conda_Forge",
				Label: text("Conda channel", "Conda 频道"),
				Help: text(
					"The address of the conda channel that pixi installs clang from. It can be a mirror of conda-forge. Empty means conda-forge.",
					"pixi 安装 clang 所用的 conda 频道地址，可以是 conda-forge 的镜像。留空即 conda-forge。"),
				Kind:    VariableURL,
				Example: "https://prefix.dev/conda-forge",
				Presets: []Preset{
					{Value: "https://prefix.dev/conda-forge", Name: plain("prefix.dev")},
					{Value: "https://mirrors.tuna.tsinghua.edu.cn/anaconda/cloud/conda-forge", Name: text("Tsinghua mirror", "清华镜像")},
				},
			},
		},
		Guide: text(
			"Write the version as vfox search clang lists it, such as 22.1.3. latest installs the version that the list marks as latest. A partial name such as 22.1 is not a version the plugin takes.",
			"按 vfox search clang 列出的样子写版本，如 22.1.3。latest 安装列表中标为 latest 的版本。22.1 这样的部分版本号不是插件接受的版本。",
		),
		Tips: []Text{
			text(
				"Each install downloads pixi, then clang from the channel, which is large: an install took 7 to 14 minutes on a network of 250 to 850 kB/s.",
				"每次安装先下载 pixi，再从频道下载 clang。clang 的包较大：在 250 至 850 kB/s 的网络上，一次安装用了 7 至 14 分钟。"),
			text(
				"GITHUB_URL, set in Settings under Environment variables, moves the GitHub downloads to a mirror that keeps the GitHub paths.",
				"在“设置”的“环境变量”中设置 GITHUB_URL，可把 GitHub 下载改到保持相同路径的镜像。"),
			text(
				"Detect and link finds the clang installations that are already on the machine, such as those of conda, Homebrew and the system llvm directories, and links them instead of installing another copy. It needs vfox 1.1.0 or later, and it was checked on Linux only.",
				"探测并链接会找出机器上已有的 clang，如 conda、Homebrew 与系统 llvm 目录中的安装，并链接它们，而不再安装一份。需要 vfox 1.1.0 或更高版本，只在 Linux 上检查过。"),
		},
		Docs: []Link{
			readme("vfox-clang", text("Plugin README", "插件说明")),
		},
	}
}
