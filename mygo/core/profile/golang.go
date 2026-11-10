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

// The profile of vfox-golang (https://github.com/version-fox/vfox-golang),
// written from its documentation at release 0.4.3.

func golangProfile() Profile {
	return Profile{
		Plugin: "golang",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest release that go.dev lists", "go.dev 上列出的最新版本")},
			{Name: "1.24.0", Meaning: text("A release in full, as vfox search golang lists it", "完整的版本号，如 vfox search golang 列出的那样")},
			{Name: "1.27rc3", Meaning: text("A release candidate or beta, which the list includes too", "候选版或测试版，列表里也有")},
		},
		Variables: []Variable{
			{
				Key:   "VFOX_GOLANG_MIRROR",
				Label: text("Download mirror", "下载镜像"),
				Help: text(
					"A mirror of go.dev/dl: the address that the archives sit under, with a trailing slash. Only the archive comes from the mirror; the version list and the checksums still come from go.dev.",
					"go.dev/dl 的镜像：安装包所在的地址，以 / 结尾。只有安装包走镜像；版本列表和校验和仍来自 go.dev。"),
				Kind:    VariableURL,
				Example: "https://golang.google.cn/dl/",
				Presets: []Preset{
					{Value: "https://golang.google.cn/dl/", Name: plain("golang.google.cn")},
					{Value: "https://mirrors.aliyun.com/golang/", Name: text("Alibaba Cloud", "阿里云")},
				},
			},
		},
		Guide: text(
			"A version is a release as go.dev names it, written in full: 1.24.0, or 1.20 for the first release of a line before 1.21, which has no patch number. latest installs the newest release. 1.24 or 1 alone is no release. Copy the name from the list.",
			"版本就是 go.dev 命名的发行版，要写完整：1.24.0；1.21 之前的版本线，首个版本没有补丁号（如 1.20）。latest 会安装最新版。单独的 1.24 或 1 不是发行版。请从列表里复制名称。"),
		Tips: []Text{
			text(
				"When another tool already put a Go on the machine, adopt it instead of downloading it again: vfox link --detect golang.",
				"如果别的工具已经在机器上装了 Go，不必重新下载：vfox link --detect golang 可以把它纳入管理。"),
			text(
				"When go does not come from vfox after vfox use, vfox doctor names the executable that shadows it and what to do: remove the other Go, or reorder PATH with path.mode and path.preserve.",
				"vfox use 之后 go 仍不是 vfox 的版本时，vfox doctor 会指出盖住它的可执行文件以及做法：删掉另一个 Go，或用 path.mode 与 path.preserve 调整 PATH 顺序。"),
			text(
				"Every release keeps its own GOPATH with its packages by default. Set VFOX_GOLANG_GOPATH before activation to share one across releases.",
				"默认每个版本都有自己的 GOPATH。在激活前设置 VFOX_GOLANG_GOPATH 可以让多个版本共用一个。"),
			text(
				"The list of releases and their checksums always come from go.dev, which vfox has to reach, even with a mirror set.",
				"即使设置了镜像，版本列表和校验和仍来自 go.dev，vfox 必须能访问它。"),
		},
		Docs: []Link{
			docs("vfox-golang", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-golang", "gopath", text("GOPATH and the environment", "GOPATH 与环境变量")),
			docs("vfox-golang", "mirror", text("Mirror", "镜像")),
			docs("vfox-golang", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
