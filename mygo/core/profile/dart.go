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

// The profile of vfox-dart (https://github.com/version-fox/vfox-dart),
// written from its documentation at release 0.1.3.

func dartProfile() Profile {
	return Profile{
		Plugin: "dart",
		Shorthands: []Shorthand{
			{Name: "stable", Meaning: text("The newest version published on the stable channel", "stable 通道上发布的最新版本")},
			{Name: "dev", Meaning: text("The newest version published on the dev channel", "dev 通道上发布的最新版本")},
			{Name: "beta", Meaning: text("The newest version published on the beta channel", "beta 通道上发布的最新版本")},
			{Name: "latest", Meaning: text("The highest version in the list, which may be a dev build; use stable for the newest stable SDK", "列表中版本号最大的版本，可能是 dev 构建；最新的稳定版请用 stable")},
		},
		Facets: []Facet{
			{ID: "channel", Label: text("Channel", "通道"), Kind: FacetNote, Values: []string{"stable", "dev", "beta"}},
		},
		Guide: text(
			"Write the release as vfox search dart lists it, such as 3.14.0. The note on an entry is the channel it came from, and a version published on several channels is listed once for each of them. A channel name instead installs the newest version on that channel, resolved when you install, and the SDK is stored under that version number.",
			"写 vfox search dart 列出的版本，如 3.14.0。每条记录后面的注记是它所属的通道；同时发布在多个通道上的版本会为每个通道各列一次。写通道名则安装该通道上最新的版本，在安装时解析，SDK 以解析出的版本号存储。"),
		Tips: []Text{
			text(
				"A channel is resolved at install time: installing stable again later can pick up a newer version.",
				"通道在安装时解析：以后再次安装 stable 可能得到更新的版本。"),
			text(
				"The plugin does not install arm64 builds, although the official Dart archive has arm64 SDKs for Linux, macOS and Windows; it errors instead of installing a mismatched SDK.",
				"插件不安装 arm64 构建，尽管 Dart 的官方归档里有 Linux、macOS 和 Windows 的 arm64 SDK；插件会报错，而不安装不匹配的 SDK。"),
			text(
				"Detect and link finds Flutter's bundled SDK.",
				"探测并链接可找到 Flutter 自带的 SDK。"),
			text(
				"Detect and link needs vfox 1.1.0.",
				"探测并链接需要 vfox 1.1.0。"),
			text(
				"Only PATH is set: no mirrors or version files.",
				"只设置 PATH：没有镜像，也没有版本文件。"),
		},
		Docs: []Link{
			docs("vfox-dart", "versions", text("Versions and channels", "版本与通道")),
			docs("vfox-dart", "link", text("Dart SDKs already installed", "本机已有的 Dart SDK")),
			docs("vfox-dart", "platforms", text("Platform support", "平台支持")),
		},
	}
}
