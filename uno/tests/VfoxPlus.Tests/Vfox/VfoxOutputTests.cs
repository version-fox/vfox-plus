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

using VfoxPlus.Core;
using VfoxPlus.Core.Vfox;
using Xunit;

namespace VfoxPlus.Tests.Vfox;

public class VfoxOutputTests
{
    [Fact]
    public void ParseVersionFindsTheNumber()
    {
        Assert.Equal("1.0.12", VfoxOutput.ParseVersion("vfox version 1.0.12\n"));
        Assert.Equal("2.0.0-beta.1", VfoxOutput.ParseVersion("vfox version 2.0.0-beta.1"));
        Assert.Equal(string.Empty, VfoxOutput.ParseVersion("command not found"));
    }

    [Fact]
    public void ParseCurrentReadsNamesAndVersions()
    {
        Assert.Equal([new CurrentVersion("go", "v1.2")], VfoxOutput.ParseCurrent("go -> vv1.2\n"));
        IReadOnlyList<CurrentVersion> got = VfoxOutput.ParseCurrent("Warning: something\njava -> \u001b[92mv21\u001b[0m\nmaven -> N/A \nbad name -> v1\n");
        Assert.Equal([new CurrentVersion("java", "21"), new CurrentVersion("maven", null)], got);
    }

    [Fact]
    public void ParseInstalledTreeReadsPluginsAndTheirVersions()
    {
        IReadOnlyList<SdkVersions> leaf = VfoxOutput.ParseInstalledTree("All installed sdk versions\n└──maven\n");
        Assert.Equal("maven", Assert.Single(leaf).Name);
        Assert.Empty(leaf[0].Versions);

        IReadOnlyList<SdkVersions> tree = VfoxOutput.ParseInstalledTree("All installed sdk versions\n\n└─┬nodejs\n  └──24.11.1\n\n");
        Assert.Equal(["24.11.1"], Assert.Single(tree).Versions);

        Assert.Empty(VfoxOutput.ParseInstalledTree("  └──24.11.1\n"));
    }

    [Fact]
    public void MergeInstalledSdksKeepsThePluginsOfBothOutputs()
    {
        IReadOnlyList<InstalledSdk> got = VfoxOutput.MergeInstalledSdks(
            [new SdkVersions("java", ["21"])],
            [new CurrentVersion("java", "21"), new CurrentVersion("go", null)]);
        Assert.Equal(2, got.Count);
        Assert.Equal("java", got[0].Name);
        Assert.Equal(["21"], got[0].Versions);
        Assert.Equal("21", got[0].Current);
        Assert.Equal("go", got[1].Name);
        Assert.Empty(got[1].Versions);
        Assert.False(got[1].HasCurrent);
    }

    [Fact]
    public void ParseSearchReadsNotesToolsAndTheInstalledMark()
    {
        IReadOnlyList<AvailableVersion> got = VfoxOutput.ParseSearch(
            "Available versions:\n"
            + " - 24.22.0 (LTS) [npm 11.19.1,corepack 0.34.0] (installed)\n"
            + " - 1.0 (release candidate)\n"
            + " - 0.1.14 [npm ]\n"
            + " - 2.0\n"
            + " - 3.0 ()\n");
        Assert.Equal(5, got.Count);
        Assert.Equal("24.22.0", got[0].Version);
        Assert.Equal("LTS", got[0].Note);
        Assert.True(got[0].Installed);
        Assert.Equal(["npm 11.19.1", "corepack 0.34.0"], got[0].Tools.Select(tool => tool.ToString()));
        Assert.Equal("release candidate", got[1].Note);
        Assert.False(got[1].Installed);
        Assert.Equal(["npm"], got[2].Tools.Select(tool => tool.ToString()));
        Assert.False(got[3].HasNote);
        Assert.False(got[4].HasNote);

        Assert.Equal("preview", Assert.Single(VfoxOutput.ParseSearch(" - 1.0 preview\n")).Note);
        Assert.Empty(VfoxOutput.ParseSearch("Please select a version\n -\n - \n"));
    }

    [Fact]
    public void ParsePluginInfoKeepsSeparatorsOfTheDescription()
    {
        PluginInfo? info = VfoxOutput.ParsePluginInfo("a\u001f1.0\u001fhttps://x\u001f/p\u001fone\u001ftwo");
        Assert.NotNull(info);
        Assert.Equal("one\u001ftwo", info.Description);
        Assert.Equal("https://x", info.Homepage);
        Assert.Null(VfoxOutput.ParsePluginInfo("nodejs not supported"));
    }

    [Fact]
    public void ParseInstallPathRecognisesNotFound()
    {
        Assert.Null(VfoxOutput.ParseInstallPath("notfound\n"));
        Assert.Equal("/opt/sdk/v-1", VfoxOutput.ParseInstallPath("/opt/sdk/v-1\n"));
    }

    [Fact]
    public void ParseConfigListReadsKeysWithEmptyValues()
    {
        Assert.Equal(
            [new ConfigEntry("proxy.url", string.Empty), new ConfigEntry("foo.bar", "a = b")],
            VfoxOutput.ParseConfigList("proxy.url = \nfoo.bar = a = b\nnot a setting\n = x\n"));
        Assert.Equal([new ConfigEntry("registry.address")], VfoxOutput.ParseConfigList("registry.address =\n"));
    }

    [Fact]
    public void ParseDownloadProgressReadsThePercentage()
    {
        Assert.Equal(0.42, VfoxOutput.ParseDownloadProgress("Downloading...  42% [===>  ] (1 MB/s) [1s:2s]"));
        Assert.Equal(1.0, VfoxOutput.ParseDownloadProgress("Downloading... 100% [======]"));
        Assert.Equal(1.0, VfoxOutput.ParseDownloadProgress("Downloading... 250% [======]"));
        Assert.Null(VfoxOutput.ParseDownloadProgress("Preinstalling maven@3.9.16..."));
    }

    [Fact]
    public void SummarizeErrorPicksTheMeaningfulLine()
    {
        Assert.Equal("Use the -y flag", VfoxOutput.SummarizeError("Removing...\n", "Use the -y flag\n"));
        Assert.Equal(
            "failed to download x@1 file, err:EOF",
            VfoxOutput.SummarizeError(
                "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\nfailed to install x\n",
                "\rDownloading...  3% [>   ]\r      \r"));

        // vfox 1.0.13 prints the error a command returns to standard error.
        Assert.Equal(
            "failed to download x@1 file, err:EOF",
            VfoxOutput.SummarizeError(
                "Preinstalling x@1...\nfailed to download x@1 file, err:EOF\n",
                "\rDownloading...  3% [>   ]\r      \rfailed to install x\n"));
        Assert.Equal(string.Empty, VfoxOutput.SummarizeError(string.Empty, "  \n"));
        Assert.Equal("failed to install x", VfoxOutput.SummarizeError(string.Empty, "failed to install x\n"));
    }
}
