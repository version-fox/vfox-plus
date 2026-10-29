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

// The profile of vfox-nodejs (https://github.com/version-fox/vfox-nodejs),
// written from its documentation at release 0.5.3.

func nodejsProfile() Profile {
	return Profile{
		Plugin: "nodejs",
		Shorthands: []Shorthand{
			{Name: "24", Meaning: text("The newest 24.x release", "最新的 24.x 版本")},
			{Name: "24.21", Meaning: text("The newest 24.21.x release", "最新的 24.21.x 版本")},
			{Name: "latest", Meaning: text("The newest release of any line, LTS or not", "所有版本线中最新的版本，不论是否 LTS")},
		},
		Variables: []Variable{
			{
				Key:   "VFOX_NODEJS_MIRROR",
				Label: text("Download mirror", "下载镜像"),
				Help: text(
					"A mirror of nodejs.org/dist: it serves the version list, the checksums and the packages. A mirror that lags behind lists fewer releases.",
					"nodejs.org/dist 的镜像，提供版本列表、校验和与安装包。落后的镜像会少列出一些版本。"),
				Kind:    VariableURL,
				Example: "https://npmmirror.com/mirrors/node",
				Presets: []Preset{
					{Value: "https://npmmirror.com/mirrors/node", Name: plain("npmmirror")},
					{Value: "https://mirrors.huaweicloud.com/nodejs", Name: text("Huawei Cloud", "华为云")},
					{Value: "https://mirrors.tuna.tsinghua.edu.cn/nodejs-release", Name: text("Tsinghua University (TUNA)", "清华大学 TUNA")},
					{Value: "https://mirrors.ustc.edu.cn/node", Name: text("USTC", "中国科学技术大学")},
					{Value: "https://mirrors.aliyun.com/nodejs-release", Name: text("Alibaba Cloud", "阿里云")},
					{Value: "https://mirrors.cloud.tencent.com/nodejs-release", Name: text("Tencent Cloud", "腾讯云")},
					{Value: "https://mirror.nju.edu.cn/nodejs-release", Name: text("Nanjing University", "南京大学")},
					{Value: "https://mirrors.bfsu.edu.cn/nodejs-release", Name: text("Beijing Foreign Studies University", "北京外国语大学")},
				},
			},
			{
				Key:   "VFOX_NODEJS_LIBC",
				Label: text("C library", "C 库"),
				Help: text(
					"The plugin detects musl systems such as Alpine itself. Set this only on a system it gets wrong: musl forces the musl builds, glibc the others.",
					"插件会自行识别 Alpine 等 musl 系统。只有识别错误时才需要设置：musl 强制使用 musl 构建，glibc 强制使用其他构建。"),
				Kind: VariableChoice, Advanced: true,
				Presets: []Preset{
					{Value: "musl", Name: plain("musl")},
					{Value: "glibc", Name: plain("glibc")},
				},
			},
		},
		LegacyFiles:    []string{".node-version", ".nvmrc"},
		LegacyStrategy: true,
		Facets: []Facet{
			{ID: "support", Label: text("Support", "支持"), Kind: FacetNote, Values: []string{"LTS"}},
		},
		Guide: text(
			"A version is a release number, or the start of one: 24.21.0, 24.21 or 24 install the newest release of that line, and latest the newest release of any line. There is no lts alias: name the line.",
			"版本是一个发行版本号，或者它的开头：24.21.0、24.21 或 24 会安装该版本线上最新的版本，latest 则是所有版本线中最新的。没有 lts 别名，请写版本线。"),
		Tips: []Text{
			text(
				"An installed release keeps its release number, such as 24.22.0. Uninstalling needs the whole number: 22 or 22.23 answer that nothing is installed.",
				"已安装的版本使用完整的版本号，例如 24.22.0。卸载必须写完整版本号：写 22 或 22.23 会提示未安装。"),
			text(
				"Packages installed with npm install -g belong to the release in use. Switching to another release leaves them behind: install them again.",
				"用 npm install -g 安装的包属于当时使用的版本。切换到别的版本后要重新安装。"),
			text(
				"Node.js 25 and later does not ship Corepack. A node that pnpm installs itself can shadow the one of vfox.",
				"Node.js 25 及更高版本不再自带 Corepack。pnpm 自己安装的 node 可能会盖住 vfox 的。"),
			text(
				"The version list is the same on every platform; a release without a package for this machine fails when it is installed.",
				"版本列表在所有平台上都一样；没有适用于本机的安装包的版本，安装时会失败。"),
		},
		Docs: []Link{
			docs("vfox-nodejs", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-nodejs", "lts", plain("LTS")),
			docs("vfox-nodejs", "nvmrc", plain(".nvmrc, .node-version")),
			docs("vfox-nodejs", "mirror", text("Mirror", "镜像")),
			docs("vfox-nodejs", "npm", text("npm and global packages", "npm 与全局包")),
			docs("vfox-nodejs", "pnpm", plain("pnpm")),
			docs("vfox-nodejs", "musl", plain("musl (Alpine)")),
			docs("vfox-nodejs", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
