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

// The profile of vfox-python (https://github.com/version-fox/vfox-python),
// written from its documentation at release 0.6.4. The commits after that tag
// change only the documents.

// pythonProfile is the profile of vfox-python. A variable that only one
// platform reads is offered on that platform alone: the C library is detected
// on Linux, and the Python launcher is registered on Windows.
func pythonProfile(platform Platform) Profile {
	variables := []Variable{
		{
			Key:   "VFOX_PYTHON_MIRROR",
			Label: text("python.org mirror", "python.org 镜像"),
			Help: text(
				"A mirror of www.python.org/ftp/python, for the Windows installers and for the source archives of a source build. The list of versions still comes from the pyenv index, or from python.org on Windows. On Linux and macOS, a mirror keeps CPython on a source build unless the uv-build mirror is set too.",
				"www.python.org/ftp/python 的镜像，用于 Windows 安装包，以及源码构建所用的源码包。版本列表仍来自 pyenv 索引，Windows 上则来自 python.org。在 Linux 和 macOS 上，设置镜像后 CPython 会走源码构建，除非同时设置 uv-build 镜像。"),
			Kind:    VariableURL,
			Example: "https://mirrors.huaweicloud.com/python/",
			Presets: []Preset{
				{Value: "https://mirrors.huaweicloud.com/python/", Name: text("Huawei Cloud", "华为云")},
				{Value: "https://registry.npmmirror.com/-/binary/python/", Name: plain("npmmirror")},
			},
		},
		{
			Key:   "VFOX_PYTHON_USE_UV_BUILD",
			Label: text("Install method", "安装方式"),
			Help: text(
				"By default, on Linux and macOS, a CPython release that the uv-build index lists installs as a prebuilt archive, and the others are built from source. Windows keeps the python.org installers. 0: build from source on Linux and macOS. 1: uv-build only, on every platform; a release the index does not list is an error.",
				"默认情况下，在 Linux 和 macOS 上，uv-build 索引列出的 CPython 版本安装预编译的压缩包，其余版本从源码构建；Windows 保持使用 python.org 安装包。0：在 Linux 和 macOS 上从源码构建。1：所有平台都只使用 uv-build，索引没有列出的版本会报错。"),
			Kind: VariableChoice,
			Presets: []Preset{
				{Value: "0", Name: text("Source build on Linux and macOS", "Linux 和 macOS 上从源码构建")},
				{Value: "1", Name: text("uv-build only", "仅 uv-build")},
			},
		},
		{
			Key:   "VFOX_PYTHON_UV_BUILD_MIRROR",
			Label: text("uv-build mirror", "uv-build 镜像"),
			Help: text(
				"A mirror of the GitHub releases of python-build-standalone, where the uv-build archives come from. The index and its checksums still come from vault.vfox.dev.",
				"python-build-standalone 在 GitHub 发布页的镜像，uv-build 压缩包从这里下载。索引和校验和仍来自 vault.vfox.dev。"),
			Kind:    VariableURL,
			Example: "https://registry.npmmirror.com/-/binary/python-build-standalone/",
			Presets: []Preset{
				{Value: "https://registry.npmmirror.com/-/binary/python-build-standalone/", Name: plain("npmmirror")},
			},
		},
	}
	switch platform.OS {
	case "linux":
		variables = append(variables, Variable{
			Key:   "VFOX_PYTHON_UV_LIBC",
			Label: text("C library", "C 库"),
			Help: text(
				"On Linux the plugin detects glibc and musl itself. Set this only where detection is wrong or fails: uv-build then takes the build for gnu or musl. Without a known C library, the default install method builds from source.",
				"在 Linux 上，插件会自行识别 glibc 与 musl。只有识别错误或失败时才需要设置：uv-build 会改用 gnu 或 musl 版本。无法识别 C 库时，默认安装方式会改为源码构建。"),
			Kind:     VariableChoice,
			Advanced: true,
			Presets: []Preset{
				{Value: "gnu", Name: plain("gnu (glibc)")},
				{Value: "musl", Name: plain("musl")},
			},
		})
	case "windows":
		variables = append(variables, Variable{
			Key:   "VFOX_PYTHON_REGISTER_PY_LAUNCHER",
			Label: text("Python launcher", "Python 启动器"),
			Help: text(
				"Installs are registered with the Python launcher, so that py -0p lists them. Set it to 0 to turn the registration off.",
				"安装的 Python 会注册到 Python 启动器，使 py -0p 能够列出它们。设为 0 则关闭注册。"),
			Kind: VariableChoice,
			Presets: []Preset{
				{Value: "1", Name: text("On", "开启")},
				{Value: "0", Name: text("Off", "关闭")},
			},
		})
	}

	return Profile{
		Plugin: "python",
		Shorthands: []Shorthand{
			{Name: "latest", Meaning: text("The newest CPython release in the list that is not free-threaded; on Linux and macOS it can be a pre-release", "列表中最新的非自由线程版 CPython 版本；在 Linux 和 macOS 上可能是预发布版")},
			{Name: "3.13", Meaning: text("The newest 3.13.x release, never a free-threaded or pre-release build", "最新的 3.13.x 版本，不包括自由线程版和预发布版")},
			{Name: "3.13t", Meaning: text("The newest free-threaded 3.13.x release", "最新的 3.13.x 自由线程版")},
		},
		Variables:      variables,
		LegacyFiles:    []string{".python-version"},
		LegacyStrategy: true,
		Facets: []Facet{
			{ID: "implementation", Label: text("Implementation", "实现"), Kind: FacetNote, Values: []string{"GraalPy", "GraalPy Community"}},
		},
		Guide: text(
			"A version is a CPython release as the list shows it, such as 3.12.2, 3.14.0a4 or 3.14.0a4t. A partial name such as 3.12 installs the newest release of that line; GraalPy is named in full, such as graalpy-24.2.0.",
			"版本就是列表里显示的名称，如 3.12.2、3.14.0a4 或 3.14.0a4t。3.12 这样的部分版本号会安装该版本线上最新的版本；GraalPy 要写完整名称，如 graalpy-24.2.0。"),
		Tips: []Text{
			text(
				"On Linux and macOS, uv-build installs the CPython releases its index lists; others are built from source.",
				"在 Linux 和 macOS 上，uv-build 直接安装索引列出的 CPython 版本，其他版本从源码构建。"),
			text(
				"A partial name without t never selects a free-threaded build or a pre-release.",
				"不带 t 的部分版本号既不会选中自由线程版，也不会选中预发布版。"),
			text(
				"With uv-build only, GraalPy cannot be installed.",
				"选择“仅 uv-build”时无法安装 GraalPy。"),
			text(
				"On Windows, installers come from python.org.",
				"在 Windows 上，安装包来自 python.org。"),
			text(
				"A linked Python keeps its source in its name, such as 3.12.3-system.",
				"链接的 Python 会在名称中保留来源，如 3.12.3-system。"),
			text(
				"GITHUB_TOKEN raises the GitHub rate limit.",
				"设置 GITHUB_TOKEN 或 GH_TOKEN 可以提高 GitHub 的访问速率限制。"),
		},
		Docs: []Link{
			docs("vfox-python", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-python", "uv", text("uv and uv-build", "uv 与 uv-build")),
			docs("vfox-python", "mirror", text("Mirror", "镜像")),
			docs("vfox-python", "graalpy", plain("GraalPy")),
			docs("vfox-python", "link", text("Use a Python that is already installed", "使用本机已有的 Python")),
			docs("vfox-python", "platforms", text("Platforms", "平台")),
			docs("vfox-python", "windows-launcher", text("Windows Python launcher", "Windows Python 启动器")),
		},
	}
}
