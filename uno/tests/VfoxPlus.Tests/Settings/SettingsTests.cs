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

using System.Text.Json.Nodes;
using VfoxPlus.Core.Settings;
using Xunit;

namespace VfoxPlus.Tests.Settings;

public sealed class SettingsTests : IDisposable
{
    private readonly string root = Directory.CreateTempSubdirectory("vfox-plus-test-").FullName;

    public void Dispose() => Directory.Delete(root, recursive: true);

    [Fact]
    public void SettingsSurviveARoundTrip()
    {
        AppSettings original = new()
        {
            Theme = ThemePreference.Dark,
            UseSystemColor = false,
            SeedColor = 0xFF123456,
            Language = LanguagePreference.SimplifiedChinese,
            VfoxExecutable = "/opt/vfox",
            ExtraEnvironment = new Dictionary<string, string> { ["A"] = "1" },
            RailExpanded = true,
            Unknown = new Dictionary<string, JsonNode?> { ["future"] = JsonValue.Create(true) },
        };
        Assert.Equal(original, AppSettings.FromJson(original.ToJson()));
    }

    [Fact]
    public void TheKeysAreWrittenInTheOrderOfTheSpecification()
    {
        string text = new AppSettings().ToJson();
        int previous = -1;
        foreach (string key in new[] { "\"theme\"", "\"useSystemColor\"", "\"seedColor\"", "\"language\"", "\"extraEnvironment\"", "\"railExpanded\"" })
        {
            int index = text.IndexOf(key, StringComparison.Ordinal);
            Assert.True(index > previous, $"keys out of order in {text}");
            previous = index;
        }

        Assert.DoesNotContain("vfoxExecutable", text, StringComparison.Ordinal);
        Assert.Contains("\"seedColor\": 4294085659", text, StringComparison.Ordinal);
        Assert.EndsWith("}\n", text, StringComparison.Ordinal);
    }

    [Fact]
    public void MalformedValuesAreIgnoredAndUnknownKeysSurvive()
    {
        AppSettings parsed = AppSettings.FromJson(
            """{"theme":"purple","useSystemColor":"yes","seedColor":"#fff","language":42,"vfoxExecutable":"","extraEnvironment":{"A":"1","B":2},"railExpanded":1,"future":true}""");
        Assert.Equal(ThemePreference.System, parsed.Theme);
        Assert.True(parsed.UseSystemColor);
        Assert.Equal(AppSettings.DefaultSeedColor, parsed.SeedColor);
        Assert.Equal(LanguagePreference.System, parsed.Language);
        Assert.Equal(string.Empty, parsed.VfoxExecutable);
        Assert.Equal(new Dictionary<string, string> { ["A"] = "1" }, parsed.ExtraEnvironment);
        Assert.False(parsed.RailExpanded);
        Assert.Contains("\"future\": true", parsed.ToJson(), StringComparison.Ordinal);

        Assert.Equal(new AppSettings(), AppSettings.FromJson("[1,2]"));
        Assert.Equal(new AppSettings(), AppSettings.FromJson("{not json"));
        Assert.Empty(AppSettings.FromJson("""{"extraEnvironment":["x"]}""").ExtraEnvironment);
        Assert.Equal(AppSettings.DefaultSeedColor, AppSettings.FromJson("""{"seedColor":true}""").SeedColor);
        Assert.Equal(AppSettings.DefaultSeedColor, AppSettings.FromJson("""{"seedColor":-1}""").SeedColor);
        Assert.Equal(AppSettings.DefaultSeedColor, AppSettings.FromJson("""{"seedColor":4294967296}""").SeedColor);
        Assert.Equal(0xFF3366CCu, AppSettings.FromJson("""{"seedColor":4281558732}""").SeedColor);
    }

    [Fact]
    public void SettingsCompareByValue()
    {
        AppSettings baseline = new();
        Assert.Equal(baseline, baseline with { ExtraEnvironment = new Dictionary<string, string>() });
        Assert.NotEqual(baseline, baseline with { RailExpanded = true });
        Assert.NotEqual(baseline, baseline with { ExtraEnvironment = new Dictionary<string, string> { ["A"] = "1" } });
        Assert.NotEqual(baseline, baseline with { Unknown = new Dictionary<string, JsonNode?> { ["x"] = JsonValue.Create(1) } });
        Assert.Equal(baseline.GetHashCode(), new AppSettings().GetHashCode());
    }

    [Fact]
    public void EnvironmentLinesAreParsedAndFormatted()
    {
        (IReadOnlyDictionary<string, string> variables, IReadOnlyList<int> invalid) = EnvironmentLines.Parse("# mirrors\nA=1\n\n B = x=y \nnot valid\n1X=2\n");
        Assert.Equal(new Dictionary<string, string> { ["A"] = "1", ["B"] = "x=y" }, variables);
        Assert.Equal([5, 6], invalid);
        Assert.Equal("A=1\nB=2", EnvironmentLines.Format(new Dictionary<string, string> { ["B"] = "2", ["A"] = "1" }));
    }

    [Theory]
    [InlineData("/cfg", "/h", "/cfg/vfox-plus/settings.json")]
    [InlineData("cfg", "/h", "/h/.config/vfox-plus/settings.json")]
    [InlineData("", "/home/u", "/home/u/.config/vfox-plus/settings.json")]
    [InlineData("", "", "/fallback/.config/vfox-plus/settings.json")]
    public void TheSettingsFileFollowsTheXdgDirectories(string xdg, string home, string expected)
    {
        Dictionary<string, string> environment = [];
        if (xdg.Length > 0)
        {
            environment["XDG_CONFIG_HOME"] = xdg;
        }

        if (home.Length > 0)
        {
            environment["HOME"] = home;
        }

        Assert.Equal(expected, JsonFileSettingsStore.SettingsPath(environment, "/fallback"));
    }

    [Fact]
    public void TheFileStoreSavesAtomicallyAndLoadsWhatItSaved()
    {
        string path = Path.Combine(root, "config", "vfox-plus", "settings.json");
        JsonFileSettingsStore store = new(path);
        Assert.Equal(new AppSettings(), store.Load());

        AppSettings changed = new()
        {
            Theme = ThemePreference.Dark,
            RailExpanded = true,
            ExtraEnvironment = new Dictionary<string, string> { ["K"] = "é" },
            Unknown = new Dictionary<string, JsonNode?> { ["future"] = JsonValue.Create(true) },
        };
        store.Save(changed);
        Assert.Equal(changed, store.Load());
        Assert.Contains("\"K\": \"é\"", File.ReadAllText(path), StringComparison.Ordinal);
        Assert.Single(Directory.GetFileSystemEntries(Path.GetDirectoryName(path)!));
    }

    [Fact]
    public void ACorruptedFileGivesTheDefaults()
    {
        string path = Path.Combine(root, "settings.json");
        File.WriteAllText(path, "{not json");
        Assert.Equal(new AppSettings(), new JsonFileSettingsStore(path).Load());
    }

    [Fact]
    public void TheFileStoreFailsWhereItCannotWrite()
    {
        string blocker = Path.Combine(root, "file");
        File.WriteAllText(blocker, string.Empty);

        // A directory cannot be made inside a file.
        JsonFileSettingsStore store = new(Path.Combine(blocker, "settings.json"));
        Assert.ThrowsAny<IOException>(() => store.Save(new AppSettings()));
    }

    [Fact]
    public void TheMemoryStoreCountsItsSaves()
    {
        MemorySettingsStore store = new(new AppSettings());
        store.Save(new AppSettings { RailExpanded = true });
        Assert.Equal(1, store.SaveCount);
        Assert.True(store.Load().RailExpanded);
    }
}
