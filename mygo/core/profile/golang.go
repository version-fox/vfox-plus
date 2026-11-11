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
			"Write the release as go.dev names it, in full: 1.24.0, or 1.20 for the first release of a line before 1.21. latest installs the newest; 1.24 or 1 alone is no release.",
			"写 go.dev 的发行版名称，要写完整：1.24.0；1.21 之前各系列的首个版本没有补丁号（如 1.20）。latest 安装最新版；单独的 1.24 或 1 不是发行版。"),
		Tips: []Text{
			text(
				"The releases and their checksums always come from go.dev, which vfox has to reach even with a mirror set.",
				"即使设置了镜像，版本列表和校验和仍来自 go.dev，vfox 必须能访问它。"),
			text(
				"Each release keeps its own GOPATH; set VFOX_GOLANG_GOPATH before activation to share one.",
				"每个版本默认有自己的 GOPATH；在激活前设置 VFOX_GOLANG_GOPATH 可以共用一个。"),
		},
		Docs: []Link{
			docs("vfox-golang", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-golang", "gopath", text("GOPATH and the environment", "GOPATH 与环境变量")),
			docs("vfox-golang", "mirror", text("Mirror", "镜像")),
			docs("vfox-golang", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
