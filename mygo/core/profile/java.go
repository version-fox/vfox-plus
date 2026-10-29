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

// The profile of vfox-java (https://github.com/version-fox/vfox-java), written
// from its documentation at release 0.6.3.

// javaOpenJDK is the distribution that `vfox search java` lists without an
// argument, and that a version name without a suffix belongs to.
const javaOpenJDK = "open"

// text is a text with a translation, and plain is one without.
func text(en, zh string) Text { return Text{EN: en, ZH: zh} }
func plain(en string) Text    { return Text{EN: en} }

// distribution makes a choice for a distribution of Foojay. The value is the
// short name the plugin lists (`tem`), the alias its long name (`temurin`).
func distribution(value, long string, name, maker Text) Choice {
	return Choice{Value: value, Aliases: []string{long}, Name: name, Maker: maker}
}

// javaDistributions are the 31 distributions of Foojay that the plugin lists,
// the popular ones first, then the others by name.
var javaDistributions = func() []Choice {
	popular := func(c Choice) Choice { c.Popular = true; return c }
	return []Choice{
		popular(distribution("open", "openjdk", plain("OpenJDK"), text("Builds of jdk.java.net, the default", "jdk.java.net 上的构建，默认"))),
		popular(distribution("tem", "temurin", plain("Temurin"), plain("Eclipse Adoptium"))),
		popular(distribution("zulu", "zulu", plain("Zulu"), plain("Azul"))),
		popular(distribution("librca", "liberica", plain("Liberica"), plain("BellSoft"))),
		popular(distribution("amzn", "corretto", plain("Corretto"), text("Amazon", "亚马逊"))),
		popular(distribution("ms", "microsoft", plain("Microsoft Build of OpenJDK"), text("Microsoft", "微软"))),
		popular(distribution("oracle", "oracle", plain("Oracle JDK"), plain("Oracle"))),
		popular(distribution("graal", "graalvm", plain("Oracle GraalVM"), plain("Oracle"))),
		distribution("adpt", "aoj", text("AdoptOpenJDK with HotSpot", "AdoptOpenJDK（HotSpot）"), plain("AdoptOpenJDK")),
		distribution("adptj9", "aoj_openj9", text("AdoptOpenJDK with OpenJ9", "AdoptOpenJDK（OpenJ9）"), plain("AdoptOpenJDK")),
		distribution("albba", "dragonwell", plain("Dragonwell"), text("Alibaba", "阿里巴巴")),
		distribution("bsg", "bisheng", text("BiSheng JDK", "毕昇 JDK"), text("Huawei", "华为")),
		distribution("eliya", "eliya", plain("Eliya JDK"), plain("Asymm Systems")),
		distribution("gln", "gluon_graalvm", plain("Gluon GraalVM"), plain("Gluon")),
		distribution("graalce", "graalvm_community", text("GraalVM Community Edition", "GraalVM 社区版"), text("GraalVM project", "GraalVM 项目")),
		distribution("grl8", "graalvm_ce8", text("GraalVM CE for JDK 8", "GraalVM CE（JDK 8）"), text("GraalVM project", "GraalVM 项目")),
		distribution("grl11", "graalvm_ce11", text("GraalVM CE for JDK 11", "GraalVM CE（JDK 11）"), text("GraalVM project", "GraalVM 项目")),
		distribution("grl16", "graalvm_ce16", text("GraalVM CE for JDK 16", "GraalVM CE（JDK 16）"), text("GraalVM project", "GraalVM 项目")),
		distribution("grl17", "graalvm_ce17", text("GraalVM CE for JDK 17", "GraalVM CE（JDK 17）"), text("GraalVM project", "GraalVM 项目")),
		distribution("grl19", "graalvm_ce19", text("GraalVM CE for JDK 19", "GraalVM CE（JDK 19）"), text("GraalVM project", "GraalVM 项目")),
		distribution("jb", "jetbrains", plain("JetBrains Runtime"), plain("JetBrains")),
		distribution("kona", "kona", plain("Kona JDK"), text("Tencent", "腾讯")),
		distribution("nik", "liberica_native", plain("Liberica Native Image Kit"), plain("BellSoft")),
		distribution("mandrel", "mandrel", plain("Mandrel"), plain("Red Hat")),
		distribution("ojdkbld", "ojdk_build", plain("OJDKBuild"), text("Community", "社区")),
		distribution("opnlgc", "openlogic", plain("OpenLogic OpenJDK"), plain("OpenLogic")),
		distribution("rdht", "redhat", plain("Red Hat build of OpenJDK"), plain("Red Hat")),
		distribution("sapmchn", "sap_machine", plain("SapMachine"), plain("SAP")),
		distribution("sem", "semeru", text("Semeru Runtimes, open edition", "Semeru Runtimes 开放版"), plain("IBM")),
		distribution("semcrt", "semeru_certified", text("Semeru Runtimes, certified edition", "Semeru Runtimes 认证版"), plain("IBM")),
		distribution("trava", "trava", plain("TravaOpenJDK"), text("Community", "社区")),
	}
}()

// javaEverything lists the versions of every distribution at once.
var javaEverything = Choice{
	Value: "all", Name: text("All distributions", "全部发行版"), Everything: true, Popular: true,
	Maker: text("Every distribution at once, about 2,500 versions", "一次列出所有发行版，约 2500 个版本"),
}

// javaLoongnix are the JDKs of the Loongnix archive, the only source on Linux
// for loong64: Foojay does not know that architecture.
var javaLoongnix = []Choice{
	{
		Value: "loongnix", Aliases: []string{"all"}, Name: text("Loongnix JDKs", "龙芯 Loongnix JDK"), Everything: true,
		Maker: text("Both ABIs of Loongson's archive", "龙芯归档的两种 ABI"), Popular: true,
	},
	{
		Value: "lnxabi1", Aliases: []string{"loongnix_abi1"}, Name: plain("Loongnix ABI1"), Popular: true,
		Maker: text("Linux 4.19 UAPI: Loongnix 20, Loongnix Server 8.4", "Linux 4.19 UAPI：Loongnix 20、Loongnix Server 8.4"),
	},
	{
		Value: "lnxabi2", Aliases: []string{"loongnix_abi2"}, Name: plain("Loongnix ABI2"), Popular: true,
		Maker: text("Linux 5.10 UAPI: Debian 12, openEuler", "Linux 5.10 UAPI：Debian 12、openEuler"),
	},
}

// javaDistribution returns the distribution with the short name, among those of
// Foojay and of Loongnix.
func javaDistribution(value string) (Choice, bool) {
	for _, group := range [][]Choice{javaDistributions, javaLoongnix} {
		for _, choice := range group {
			if choice.Value == value {
				return choice, true
			}
		}
	}
	return Choice{}, false
}

func javaProfile(platform Platform) Profile {
	search := &Search{Label: text("Distribution", "发行版"), Default: javaOpenJDK}
	if platform.OS == "linux" && platform.Arch == "loong64" {
		search.Default = "loongnix"
		search.Choices = append(search.Choices, javaLoongnix...)
	} else {
		search.Choices = append(search.Choices, javaDistributions[:8]...)
		search.Choices = append(search.Choices, javaEverything)
		search.Choices = append(search.Choices, javaDistributions[8:]...)
	}
	return Profile{
		Plugin: "java",
		Naming: NamingJava,
		Search: search,
		Shorthands: []Shorthand{
			{Name: "21-tem", Meaning: text("The newest Temurin 21", "最新的 Temurin 21")},
			{Name: "21.0.8-tem", Meaning: text("The newest build of Temurin 21.0.8", "Temurin 21.0.8 的最新构建")},
			{Name: "temurin-21", Meaning: text("The same as 21-tem, with the distribution first", "与 21-tem 相同，发行版写在前面")},
			{Name: "21-zulu-fx", Meaning: text("The newest Zulu 21 with JavaFX", "带 JavaFX 的最新 Zulu 21")},
			{Name: "28.ea-open", Meaning: text("The newest early-access build of OpenJDK 28; searches list releases only", "OpenJDK 28 的最新抢先体验版；搜索只列出正式版")},
			{Name: "21", Meaning: text("OpenJDK from jdk.java.net, which stops at 21.0.2", "jdk.java.net 上的 OpenJDK，21 只到 21.0.2")},
		},
		Variables: []Variable{{
			Key:   "VFOX_JAVA_GITHUB_MIRROR",
			Label: text("GitHub mirror", "GitHub 镜像"),
			Help: text(
				"Most distributions download from GitHub. A mirror address goes in front of the GitHub address, so it must take the full address after its own, as gh-proxy does.",
				"大多数发行版从 GitHub 下载。镜像地址会加在 GitHub 地址前面，所以镜像必须能接受完整的 GitHub 地址，如 gh-proxy。"),
			Kind:    VariableURL,
			Example: "https://gh-proxy.org/",
			Presets: []Preset{{Value: "https://gh-proxy.org/", Name: plain("gh-proxy")}},
		}},
		LegacyFiles: []string{".sdkmanrc"},
		Facets: []Facet{
			{ID: "support", Label: text("Support", "支持"), Kind: FacetNote, Values: []string{"LTS"}},
			{ID: "flavor", Label: text("Flavor", "类型"), Kind: FacetNote, Values: []string{"JavaFX"}},
		},
		Guide: text(
			"A version is written <version>-<distribution>, such as 21.0.8-tem, with -fx after it for a build with JavaFX. The distribution is its short name (tem) or its long name (temurin). Without one, the version means OpenJDK from jdk.java.net.",
			"版本写成 <版本>-<发行版>，例如 21.0.8-tem，后面加 -fx 表示带 JavaFX 的构建。发行版可以写简称（tem）或全称（temurin）。不写发行版时指 jdk.java.net 上的 OpenJDK。"),
		Tips: []Text{
			text(
				"OpenJDK from jdk.java.net keeps only the newest releases and the last build of older ones: 17 stops at 17.0.2 and 21 at 21.0.2. Name a distribution for a release that is still maintained.",
				"jdk.java.net 只保留最新的版本和旧版本的最后一次构建：17 只到 17.0.2，21 只到 21.0.2。要装仍在维护的版本，请写上发行版。"),
			text(
				"An installed JDK keeps its full name, such as 21.0.12.1+1-tem. Using and uninstalling take that name; vfox use also maps 21-tem to the newest installed build.",
				"已安装的 JDK 使用完整名称，例如 21.0.12.1+1-tem。使用和卸载都要用这个名称；vfox use 还能把 21-tem 对应到已安装的最新构建。"),
			text(
				"-fx builds and plain builds are never mixed: java@21-zulu installs a build without JavaFX, and java@21-tem-fx fails because Temurin has none.",
				"带 -fx 的构建和普通构建不会混用：java@21-zulu 安装不带 JavaFX 的构建，而 java@21-tem-fx 会失败，因为 Temurin 没有这种构建。"),
			text(
				"A search lists what exists for this machine, and vfox keeps the answer for 12 hours per distribution.",
				"搜索列出的是适用于本机的版本，vfox 会按发行版把结果缓存 12 小时。"),
		},
		Docs: []Link{
			docs("vfox-java", "versions", text("Versions and names", "版本与命名")),
			docs("vfox-java", "distributions", text("Distributions", "发行版")),
			docs("vfox-java", "javafx", plain("JavaFX")),
			docs("vfox-java", "sdkmanrc", plain(".sdkmanrc")),
			docs("vfox-java", "mirror", text("GitHub mirror", "GitHub 镜像")),
			docs("vfox-java", "platforms", text("Platforms", "平台")),
			docs("vfox-java", "troubleshooting", text("Troubleshooting", "问题排查")),
		},
	}
}

// docs links a page of the documentation of a plugin of the version-fox
// organisation, in English and in Simplified Chinese.
func docs(repository, page string, title Text) Link {
	base := "https://github.com/version-fox/" + repository + "/blob/main/docs/"
	return Link{Title: title, URL: Text{EN: base + page + ".md", ZH: base + "zh-CN/" + page + ".md"}}
}
