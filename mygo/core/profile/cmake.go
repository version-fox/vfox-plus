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

// The profile of vfox-cmake (https://github.com/version-fox/vfox-cmake),
// written from its documentation at release 0.1.8 and checked against what that
// release lists and installs, on every platform that its data has.

// cmakeCandidates are the release candidates that the list of CMake has had, as
// the last part of a version: 4.5.0-rc1 is the first candidate of 4.5.0.
var cmakeCandidates = []string{"rc1", "rc2", "rc3", "rc4", "rc5", "rc6"}

// cmakeProfile is the profile of vfox-cmake. The plugin gives no note to a
// version, and takes no variable: it adds the bin directory to PATH, and its
// Detect hook reads CMAKE_HOME to find a CMake that is already installed.
//
// At release 0.1.8 its sort does not tell a release candidate from its
// release: the comparison drops the part of a version that is not a number, so
// 4.5.0-rc3, 4.5.0-rc2 and 4.5.0 are equal to it, and they are listed in no
// fixed order (on Linux for amd64 141 of the 142 candidates come before their
// release, on Linux for x86 14 of 35). vfox resolves latest to the first line of
// the list, which is 4.5.0-rc3 on every platform that has CMake 4.5. The
// profile says so where latest is offered.
func cmakeProfile() Profile {
	return Profile{
		Plugin: "cmake",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The first line of the list, which can be a release candidate", "列表的第一行，可能是候选版")},
		},
		Facets: []Facet{
			{ID: "candidate", Label: text("Release candidate", "候选版"), Kind: FacetSuffix, Values: cmakeCandidates},
		},
		Guide: text(
			"Write the release in full as vfox search cmake lists it, such as 3.31.6, or 4.5.0-rc1 for a release candidate. A line such as 3.31 is no release. latest installs the first line of the list, and a release candidate can come before its release there: write the release to be sure of it.",
			"按 vfox search cmake 的列表完整写出版本，如 3.31.6；候选版写成 4.5.0-rc1。3.31 这样的版本线不是发行版。latest 安装列表的第一行，而候选版可能排在它的正式版之前：要装正式版，请写出它的版本号。"),
		Tips: []Text{
			text(
				"latest is the first line of the list, and a release candidate can come before its release.",
				"latest 是列表的第一行，而候选版可能排在它的正式版之前。"),
			text(
				"Detect and link finds CMake on PATH and in CMAKE_HOME, which you set in Settings, under Environment variables.",
				"探测并链接会在 PATH 和 CMAKE_HOME 中找已有的 CMake；CMAKE_HOME 在“设置”的“环境变量”中设置。"),
			text(
				"Needs vfox 1.1.0 or later.",
				"需要 vfox 1.1.0 或更高版本。"),
			text(
				"Needs access to jsDelivr and cmake.org.",
				"需要能访问 jsDelivr 和 cmake.org。"),
		},
		Docs: []Link{
			docs("vfox-cmake", "platforms", text("Platforms and versions", "平台与版本")),
			docs("vfox-cmake", "detect", text("Use a CMake that is already installed", "使用本机已有的 CMake")),
			docs("vfox-cmake", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}
