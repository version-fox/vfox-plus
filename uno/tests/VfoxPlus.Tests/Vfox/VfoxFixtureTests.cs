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
using VfoxPlus.Core;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Tests.Support;
using Xunit;

namespace VfoxPlus.Tests.Vfox;

/// <summary>
/// Runs the parsers against the shared vfox output fixtures. The expected results live in
/// <c>shared/fixtures/vfox-*/cases.json</c> and are shared by every implementation of the GUI; see
/// <c>shared/fixtures/README.md</c>.
/// </summary>
public class VfoxFixtureTests
{
    public static TheoryData<string, string, string> Cases()
    {
        TheoryData<string, string, string> data = [];
        string fixtures = Path.Combine(TestPaths.SharedDirectory, "fixtures");
        foreach (string manifest in Directory.GetFiles(fixtures, "cases.json", SearchOption.AllDirectories).Order(StringComparer.Ordinal))
        {
            JsonObject root = JsonNode.Parse(File.ReadAllText(manifest))!.AsObject();
            string directory = Path.GetDirectoryName(manifest)!;
            foreach (JsonNode? item in root["cases"]!.AsArray())
            {
                JsonObject fixture = item!.AsObject();
                foreach ((string check, JsonNode? expected) in fixture["expect"]!.AsObject())
                {
                    data.Add(directory, fixture["name"]!.GetValue<string>(), check + "|" + (expected?.ToJsonString() ?? "null"));
                }
            }
        }

        return data;
    }

    [Theory]
    [MemberData(nameof(Cases))]
    public void ParsersMatchTheFixtures(string directory, string name, string checkAndExpected)
    {
        int separator = checkAndExpected.IndexOf('|', StringComparison.Ordinal);
        string check = checkAndExpected[..separator];
        JsonNode? expected = JsonNode.Parse(checkAndExpected[(separator + 1)..]);
        string stdout = Fixtures.ReadCapture(Path.Combine(directory, name + ".stdout"));
        string stderr = Fixtures.ReadCapture(Path.Combine(directory, name + ".stderr"));

        JsonNode? actual = check switch
        {
            "version" => JsonValue.Create(VfoxOutput.ParseVersion(stdout)),
            "current" => Current(VfoxOutput.ParseCurrent(stdout)),
            "list" => Tree(VfoxOutput.ParseInstalledTree(stdout)),
            "search" => Selected(Indexed(VfoxOutput.ParseSearch(stdout).Select(Search).ToList()), expected),
            "available" => Selected(Indexed(VfoxOutput.ParseAvailablePlugins(stdout).Select(Available).ToList()), expected),
            "pluginInfo" => VfoxOutput.ParsePluginInfo(stdout) is { } info ? PluginInfoJson(info) : null,
            "installPath" => VfoxOutput.ParseInstallPath(stdout) is { } path ? JsonValue.Create(path) : null,
            "config" => new JsonArray(VfoxOutput.ParseConfigList(stdout).Select(entry => (JsonNode?)new JsonObject { ["key"] = entry.Key, ["value"] = entry.Value }).ToArray()),
            "finalProgress" => FinalProgress(stderr),
            "error" => JsonValue.Create(VfoxOutput.SummarizeError(stdout, stderr)),
            _ => throw new InvalidOperationException($"Unknown parser '{check}' in cases.json"),
        };

        Assert.True(Same(actual, expected), $"{name} {check}: got {actual?.ToJsonString()}, want {expected?.ToJsonString()}");
    }

    private static JsonObject Current(IReadOnlyList<CurrentVersion> versions)
    {
        JsonObject result = [];
        foreach (CurrentVersion version in versions)
        {
            result[version.Name] = version.Version;
        }

        return result;
    }

    private static JsonObject Tree(IReadOnlyList<SdkVersions> tree)
    {
        JsonObject result = [];
        foreach (SdkVersions item in tree)
        {
            result[item.Name] = new JsonArray(item.Versions.Select(version => (JsonNode?)JsonValue.Create(version)).ToArray());
        }

        return result;
    }

    private static JsonObject PluginInfoJson(PluginInfo info) => new()
    {
        ["name"] = info.Name,
        ["version"] = info.Version,
        ["homepage"] = info.Homepage,
        ["installPath"] = info.InstallPath,
        ["description"] = info.Description,
    };

    private static JsonObject Search(AvailableVersion version) => new()
    {
        ["version"] = version.Version,
        ["note"] = version.Note,
        ["additions"] = new JsonArray(version.Tools.Select(tool => (JsonNode?)new JsonObject { ["name"] = tool.Name, ["version"] = tool.Version }).ToArray()),
        ["installed"] = version.Installed,
    };

    private static JsonObject Available(AvailablePlugin plugin) => new()
    {
        ["name"] = plugin.Name,
        ["official"] = plugin.Official,
        ["homepage"] = plugin.Homepage,
    };

    private static JsonObject Indexed(List<JsonObject> items)
    {
        JsonObject entries = [];
        for (int i = 0; i < items.Count; i++)
        {
            entries[i.ToString(System.Globalization.CultureInfo.InvariantCulture)] = items[i];
        }

        return new JsonObject { ["count"] = items.Count, ["items"] = entries };
    }

    // The fixtures list the items that matter by index; the others are not compared.
    private static JsonObject Selected(JsonObject actual, JsonNode? expected)
    {
        JsonObject items = [];
        JsonObject all = actual["items"]!.AsObject();
        foreach ((string index, _) in expected!["items"]!.AsObject())
        {
            items[index] = all[index]?.DeepClone();
        }

        return new JsonObject { ["count"] = actual["count"]!.DeepClone(), ["items"] = items };
    }

    private static JsonNode? FinalProgress(string stderr)
    {
        double? last = null;
        foreach (string line in Terminal.SplitLines(Terminal.StripAnsi(stderr)))
        {
            if (VfoxOutput.ParseDownloadProgress(line) is { } progress)
            {
                last = progress;
            }
        }

        return last is { } value ? JsonValue.Create(value) : null;
    }

    private static bool Same(JsonNode? a, JsonNode? b)
    {
        switch (a, b)
        {
            case (null, null):
                return true;
            case (JsonObject left, JsonObject right):
                return left.Count == right.Count && left.All(pair => right.ContainsKey(pair.Key) && Same(pair.Value, right[pair.Key]));
            case (JsonArray left, JsonArray right):
                return left.Count == right.Count && left.Zip(right).All(pair => Same(pair.First, pair.Second));
            case (JsonValue left, JsonValue right):
                if (left.TryGetValue(out double x) && right.TryGetValue(out double y))
                {
                    return x == y;
                }

                return left.ToJsonString() == right.ToJsonString();
            default:
                return false;
        }
    }
}
