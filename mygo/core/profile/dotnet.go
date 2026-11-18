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

// The profile of vfox-dotnet (https://github.com/version-fox/vfox-dotnet),
// written from its documentation at release 0.3.3.

func dotnetProfile() Profile {
	return Profile{
		Plugin: "dotnet",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest release in Microsoft's metadata", "Microsoft 元数据中最新的发行版")},
			{Name: "8.0.34-runtime", Meaning: text("A release with -runtime installs .NET Runtime only; -aspnetcore installs ASP.NET Core Runtime", "带 -runtime 的版本只安装 .NET Runtime；-aspnetcore 安装 ASP.NET Core Runtime")},
		},
		Facets: []Facet{
			{ID: "distribution", Label: text("Distribution", "发行版"), Kind: FacetSuffix, Values: []string{"runtime", "aspnetcore"}},
		},
		Guide: text(
			"Write a channel or a release: 8, 8.0 or 8.0.34, with -runtime or -aspnetcore for a smaller distribution. vfox search lists every installable variant and puts the distribution and SDK version in the note. Use an exact version for a preview such as 11.0.0-rc.3: a channel picks the newest stable release.",
			"写通道或发行版：8、8.0 或 8.0.34，加 -runtime 或 -aspnetcore 可安装较小的发行版。vfox search 列出每一个可安装的变体，并在注记里写出发行版和 SDK 版本。预览版如 11.0.0-rc.3 要用完整版本：通道选的是最新的稳定版。"),
		Tips: []Text{
			text(
				"The plugin sets DOTNET_ROOT and puts the SDK directory on PATH; activate vfox in a script before running dotnet.",
				"插件会设置 DOTNET_ROOT 并把 SDK 目录加入 PATH；在脚本里运行 dotnet 之前要先激活 vfox。"),
			text(
				"dotnet --version prints the SDK number, such as 8.0.428, which can be newer than the release vfox selected, such as 8.0.34.",
				"dotnet --version 显示的是 SDK 号，如 8.0.428，可能比 vfox 选中的发行版（如 8.0.34）更新。"),
			text(
				"Linux needs ICU, libssl and zlib.",
				"Linux 上需要 ICU、libssl 和 zlib。"),
			text(
				"Detect and link adopts .NET SDKs already on the machine, including those under DOTNET_ROOT and DOTNET_HOME, and needs vfox 1.1.0 or later.",
				"探测并链接可纳入机器上已有的 .NET SDK，包括 DOTNET_ROOT 与 DOTNET_HOME 指向的，需要 vfox 1.1.0 或更高版本。"),
		},
		Docs: []Link{
			docs("vfox-dotnet", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-dotnet", "detect", text("Use an SDK that is already installed", "使用本机已有的 SDK")),
		},
	}
}
