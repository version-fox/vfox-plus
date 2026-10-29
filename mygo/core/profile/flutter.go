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

// The profile of vfox-flutter (https://github.com/version-fox/vfox-flutter),
// written from its documentation at release 0.2.5.

// flutterArchitectures are the architectures that a release can be asked for
// with a suffix.
var flutterArchitectures = []string{"x64", "arm64"}

func flutterProfile() Profile {
	return Profile{
		Plugin: "flutter",
		Naming: NamingFlutter,
		Shorthands: []Shorthand{
			{Name: "stable", Meaning: text("The release the stable channel points to today", "stable 通道当前指向的版本")},
			{Name: "beta", Meaning: text("The release the beta channel points to today", "beta 通道当前指向的版本")},
			{Name: "stable-x64", Meaning: text("The x64 build of the stable release; -arm64 likewise", "stable 版本的 x64 构建；-arm64 同理")},
			{Name: "latest", Meaning: text("The first line of the list: the newest stable release, with its architecture suffix", "列表的第一行：最新的 stable 版本，带架构后缀")},
		},
		Variables: []Variable{
			{
				Key:   "FLUTTER_STORAGE_BASE_URL",
				Label: text("Flutter storage mirror", "Flutter 存储镜像"),
				Help: text(
					"Where the release index and the archives come from, and where the flutter tool itself downloads from. It has to stay set for the flutter command as well.",
					"发行版索引和压缩包的来源，flutter 工具自己的下载也走这里。运行 flutter 命令时它也需要保持设置。"),
				Kind:    VariableURL,
				Example: "https://storage.flutter-io.cn",
				Presets: []Preset{
					{Value: "https://storage.flutter-io.cn", Name: text("CFUG (China Flutter User Group)", "Flutter 中文社区（CFUG）")},
					{Value: "https://mirrors.cernet.edu.cn/flutter", Name: text("CERNET", "中国教育和科研计算机网（CERNET）")},
				},
			},
			{
				Key:   "VFOX_FLUTTER_GITHUB_MIRROR",
				Label: text("GitHub mirror", "GitHub 镜像"),
				Help: text(
					"For the installs from source, on Linux and Windows for ARM64: the address goes in front of the GitHub address of the clone.",
					"用于源码安装（Linux 和 Windows 的 ARM64）：地址会加在克隆所用的 GitHub 地址前面。"),
				Kind:    VariableURL,
				Example: "https://gh-proxy.org/",
				Presets: []Preset{{Value: "https://gh-proxy.org/", Name: plain("gh-proxy")}},
			},
		},
		Facets: []Facet{
			{ID: "channel", Label: text("Channel", "通道"), Kind: FacetNote, Values: []string{"stable", "beta", "dev", "OpenHarmony"}},
			{ID: "architecture", Label: text("Architecture", "架构"), Kind: FacetSuffix, Values: flutterArchitectures},
		},
		Guide: text(
			"A version is a release as the list shows it: 3.47.4, or a beta such as 3.50.0-0.1.pre. -x64 or -arm64 after it asks for that architecture. stable and beta stand for the release the channel points to. Versions are matched whole: 3.47 does not select the newest 3.47.x.",
			"版本就是列表里的发行版名称：3.47.4，或者 3.50.0-0.1.pre 这样的 beta 版。后面加 -x64 或 -arm64 可指定架构。stable 和 beta 代表通道当前指向的版本。版本必须完整匹配：3.47 不会选中最新的 3.47.x。"),
		Tips: []Text{
			text(
				"A channel is replaced by the release it points to while it installs, so afterwards use that version: flutter@stable is not an installed name, and vfox use flutter@stable fails.",
				"安装时通道会被替换成它指向的版本，之后请使用那个版本：flutter@stable 不是已安装的名称，vfox use flutter@stable 会失败。"),
			text(
				"Do not run flutter upgrade inside an SDK that vfox manages: it moves the SDK in place. Install the newer version instead.",
				"不要在 vfox 管理的 SDK 里运行 flutter upgrade：它会原地改动 SDK。请安装更新的版本。"),
			text(
				"3.47.4 and 3.47.4-x64 are two installs of the same Flutter on an x64 machine. Write the suffix always or never.",
				"在 x64 机器上，3.47.4 和 3.47.4-x64 是同一个 Flutter 的两份安装。要么总是写后缀，要么总是不写。"),
			text(
				"Linux and Windows for ARM64, and the OpenHarmony builds, install as git checkouts and need git.",
				"Linux 和 Windows 的 ARM64 版本以及 OpenHarmony 构建是以 git 检出的方式安装的，需要装有 git。"),
		},
		Docs: []Link{
			docs("vfox-flutter", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-flutter", "channels", text("Channels", "通道")),
			docs("vfox-flutter", "arm64", text("Architecture selection", "架构选择")),
			docs("vfox-flutter", "ohos", plain("OpenHarmony")),
			docs("vfox-flutter", "mirror", text("Mirror", "镜像")),
			docs("vfox-flutter", "flutter-upgrade", plain("flutter upgrade")),
			docs("vfox-flutter", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
