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

using VfoxPlus.Core.Vfox;
using Xunit;

namespace VfoxPlus.Tests.Vfox;

public sealed class PluginSourceTests : IDisposable
{
    private readonly string root = Directory.CreateTempSubdirectory("vfox-plus-test-").FullName;

    public void Dispose() => Directory.Delete(root, recursive: true);

    [Fact]
    public void OnlyLocalLuaFilesAreMoved()
    {
        Assert.True(PluginSources.VfoxMoves("/home/u/plugin.lua"));
        Assert.True(PluginSources.VfoxMoves("/home/u/PLUGIN.LUA"));
        Assert.False(PluginSources.VfoxMoves("/home/u/plugin.zip"));
        Assert.False(PluginSources.VfoxMoves("https://example.com/plugin.lua"));
        Assert.True(PluginSources.IsRemote("http://127.0.0.1:8080/p.zip"));
    }

    [Fact]
    public void TheTemporaryDirectoryFollowsTheLegacyDirectoryOfVfox()
    {
        Dictionary<string, string> environment = new() { ["HOME"] = root };
        Assert.Equal(root + "/.vfox/tmp", PluginSources.VfoxTemporaryDirectory(environment));
        Directory.CreateDirectory(root + "/.version-fox");
        Assert.Equal(root + "/.version-fox/tmp", PluginSources.VfoxTemporaryDirectory(environment));
        Assert.True(Path.IsPathRooted(PluginSources.VfoxTemporaryDirectory(new Dictionary<string, string>())));
    }

    [Fact]
    public void UrlsAndArchivesArePassedThrough()
    {
        foreach (string source in new[] { "https://example.com/p.lua", Path.Combine(root, "p.zip") })
        {
            using PreparedPluginSource prepared = PluginSources.Prepare(source, Path.Combine(root, "tmp"));
            Assert.Equal(source, prepared.Path);
            Assert.False(prepared.IsCopy);
        }
    }

    [Fact]
    public void ALocalLuaFileIsCopiedAndTheCopyIsDeleted()
    {
        string original = Path.Combine(root, "mine.lua");
        File.WriteAllText(original, "PLUGIN = {}");
        string temporary = Path.Combine(root, ".vfox", "tmp");
        PreparedPluginSource prepared = PluginSources.Prepare(original, temporary);
        Assert.True(prepared.IsCopy);
        Assert.NotEqual(original, prepared.Path);
        Assert.EndsWith("/mine.lua", prepared.Path, StringComparison.Ordinal);
        Assert.Equal("PLUGIN = {}", File.ReadAllText(prepared.Path));

        prepared.Dispose();
        Assert.False(File.Exists(prepared.Path));
        Assert.True(File.Exists(original));
        Assert.Empty(Directory.GetFileSystemEntries(temporary));
    }

    [Fact]
    public void AFileThatCannotBeCopiedIsLeftForVfoxToReport()
    {
        string temporary = Path.Combine(root, "tmp");
        string source = Path.Combine(root, "missing.lua");
        using PreparedPluginSource prepared = PluginSources.Prepare(source, temporary);
        Assert.Equal(source, prepared.Path);
        Assert.False(prepared.IsCopy);
        Assert.True(!Directory.Exists(temporary) || Directory.GetFileSystemEntries(temporary).Length == 0);
    }
}
