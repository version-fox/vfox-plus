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

using System.Globalization;
using System.Text.RegularExpressions;

namespace VfoxPlus.Core.Vfox;

/// <summary>
/// The current version of one plugin, as printed by <c>vfox current</c>.
/// </summary>
/// <param name="Name">The plugin name.</param>
/// <param name="Version">The version in effect, or <see langword="null"/> when none is.</param>
public sealed record CurrentVersion(string Name, string? Version);

/// <summary>
/// The installed versions of one plugin, in output order.
/// </summary>
/// <param name="Name">The plugin name.</param>
/// <param name="Versions">The versions.</param>
public sealed record SdkVersions(string Name, IReadOnlyList<string> Versions);

/// <summary>
/// Parsers for the human-oriented output of the vfox CLI.
/// </summary>
/// <remarks>
/// vfox has no machine-readable output mode for most commands, so every format handled here is documented
/// in <c>docs/vfox-cli-contract.md</c> and covered by the captured outputs in <c>shared/fixtures/</c>. All
/// parsers are pure functions: they receive the decoded standard output of a command and never fail on
/// unexpected lines, which are skipped instead.
/// </remarks>
public static partial class VfoxOutput
{
    /// <summary>The ASCII unit separator used between the fields of <c>vfox info --format</c>.</summary>
    public const string FieldSeparator = "\u001f";

    /// <summary>
    /// The Go template passed to <c>vfox info &lt;name&gt; --format</c>. The fields are joined with
    /// <see cref="FieldSeparator"/>, which cannot appear in any of them, so descriptions may contain new
    /// lines and any punctuation.
    /// </summary>
    public const string PluginInfoTemplate = """{{.Name}}{{"\x1f"}}{{.Version}}{{"\x1f"}}{{.Homepage}}{{"\x1f"}}{{.InstallPath}}{{"\x1f"}}{{.Description}}""";

    /// <summary>Parses <c>vfox --version</c>, for example <c>vfox version 1.0.12</c>.</summary>
    /// <param name="output">The output.</param>
    /// <returns>The version, or an empty string if no version number is found.</returns>
    public static string ParseVersion(string output)
    {
        Match match = VersionPattern().Match(Terminal.StripAnsi(output));
        return match.Success ? match.Groups[1].Value : string.Empty;
    }

    /// <summary>
    /// Parses <c>vfox current</c> without arguments into plugin names and current versions. Each line has
    /// the form <c>&lt;name&gt; -&gt; v&lt;version&gt;</c>, or <c>&lt;name&gt; -&gt; N/A</c> when no version is
    /// in effect. vfox prefixes the version with a <c>v</c>, which is removed again.
    /// </summary>
    /// <param name="output">The output.</param>
    /// <returns>The versions, in the order of the output.</returns>
    public static IReadOnlyList<CurrentVersion> ParseCurrent(string output)
    {
        List<CurrentVersion> result = [];
        foreach (string line in Lines(output))
        {
            int arrow = line.IndexOf(" -> ", StringComparison.Ordinal);
            if (arrow <= 0)
            {
                continue;
            }

            string name = line[..arrow].Trim();
            string value = line[(arrow + 4)..].Trim();
            if (name.Length == 0 || name.Contains(' ', StringComparison.Ordinal))
            {
                continue;
            }

            result.Add(new CurrentVersion(name, value.Length > 0 && value != "N/A" ? StripVPrefix(value) : null));
        }

        return result;
    }

    /// <summary>
    /// Parses the tree printed by <c>vfox list</c> without arguments into the installed versions of every
    /// plugin, in output order. A plugin without installed versions is drawn as a leaf.
    /// </summary>
    /// <param name="output">The output.</param>
    /// <returns>The plugins with their versions.</returns>
    public static IReadOnlyList<SdkVersions> ParseInstalledTree(string output)
    {
        Dictionary<string, List<string>> byName = [];
        List<string> order = [];
        List<string>? current = null;
        foreach (string line in Lines(output))
        {
            Match match = TreeItemPattern().Match(line);
            if (!match.Success)
            {
                continue;
            }

            // Each level of the tree indents by two characters.
            int level = match.Groups[1].Length / 2;
            string text = match.Groups[2].Value.Trim();
            if (text.Length == 0)
            {
                continue;
            }

            if (level == 0)
            {
                if (!byName.TryGetValue(text, out current))
                {
                    current = [];
                    byName[text] = current;
                    order.Add(text);
                }
            }
            else
            {
                current?.Add(text);
            }
        }

        return [.. order.Select(name => new SdkVersions(name, byName[name]))];
    }

    /// <summary>
    /// Combines the outputs of <c>vfox list</c> and <c>vfox current</c> into SDKs. Plugins that only appear
    /// in one of the outputs are still included, so that the result degrades gracefully if one command fails.
    /// </summary>
    /// <param name="tree">The installed versions.</param>
    /// <param name="current">The current versions.</param>
    /// <returns>The installed SDKs.</returns>
    public static IReadOnlyList<InstalledSdk> MergeInstalledSdks(IReadOnlyList<SdkVersions> tree, IReadOnlyList<CurrentVersion> current)
    {
        Dictionary<string, IReadOnlyList<string>> versionsByName = [];
        List<string> order = [];
        HashSet<string> seen = [];
        foreach (SdkVersions item in tree)
        {
            versionsByName[item.Name] = item.Versions;
            if (seen.Add(item.Name))
            {
                order.Add(item.Name);
            }
        }

        Dictionary<string, CurrentVersion> currentByName = [];
        foreach (CurrentVersion item in current)
        {
            currentByName[item.Name] = item;
            if (seen.Add(item.Name))
            {
                order.Add(item.Name);
            }
        }

        return
        [
            .. order.Select(name => new InstalledSdk(
                name,
                versionsByName.GetValueOrDefault(name) ?? [],
                currentByName.GetValueOrDefault(name)?.Version)),
        ];
    }

    /// <summary>
    /// Parses <c>vfox search &lt;name&gt;</c> in a non-interactive terminal. Every entry is
    /// <c>&lt;version&gt;[ (&lt;note&gt;)][ [&lt;tool&gt; &lt;version&gt;,...]][ (installed)]</c>.
    /// </summary>
    /// <param name="output">The output.</param>
    /// <returns>The versions that can be installed.</returns>
    public static IReadOnlyList<AvailableVersion> ParseSearch(string output)
    {
        List<AvailableVersion> result = [];
        foreach (string rawLine in Lines(output))
        {
            string line = rawLine.TrimEnd(' ', '\t');
            if (line.StartsWith(" - ", StringComparison.Ordinal) && ParseSearchEntry(line[3..].Trim()) is { } version)
            {
                result.Add(version);
            }
        }

        return result;
    }

    /// <summary>Parses <c>vfox available</c>.</summary>
    /// <param name="output">The output.</param>
    /// <returns>The plugins of the registry.</returns>
    public static IReadOnlyList<AvailablePlugin> ParseAvailablePlugins(string output)
    {
        List<AvailablePlugin> result = [];
        foreach (string line in Lines(output))
        {
            Match match = AvailablePluginPattern().Match(line);
            if (match.Success)
            {
                result.Add(new AvailablePlugin(match.Groups[1].Value, match.Groups[3].Value, match.Groups[2].Value == "✓"));
            }
        }

        return result;
    }

    /// <summary>
    /// Parses the output of <c>vfox info &lt;name&gt; --format</c> with <see cref="PluginInfoTemplate"/>.
    /// </summary>
    /// <param name="output">The output.</param>
    /// <returns>The information, or <see langword="null"/> if the output does not have the expected number of fields.</returns>
    public static PluginInfo? ParsePluginInfo(string output)
    {
        string[] fields = Terminal.StripAnsi(output).Split(FieldSeparator);
        if (fields.Length < 5)
        {
            return null;
        }

        // The description is the last field and may itself contain separators.
        return new PluginInfo(
            fields[0].Trim(),
            fields[1].Trim(),
            fields[2].Trim(),
            fields[3].Trim(),
            string.Join(FieldSeparator, fields[4..]).Trim());
    }

    /// <summary>
    /// Parses <c>vfox info &lt;name&gt;@&lt;version&gt;</c>, which prints the directory of an installed
    /// version, or <c>notfound</c>.
    /// </summary>
    /// <param name="output">The output.</param>
    /// <returns>The directory, or <see langword="null"/> if the version is not installed.</returns>
    public static string? ParseInstallPath(string output)
    {
        string path = Terminal.StripAnsi(output).Trim();
        return path.Length == 0 || path == "notfound" ? null : path;
    }

    /// <summary>Parses <c>vfox config --list</c>, which prints one <c>&lt;key&gt; = &lt;value&gt;</c> per line.</summary>
    /// <param name="output">The output.</param>
    /// <returns>The entries.</returns>
    public static IReadOnlyList<ConfigEntry> ParseConfigList(string output)
    {
        List<ConfigEntry> result = [];
        foreach (string line in Lines(output))
        {
            int equals = line.IndexOf(" = ", StringComparison.Ordinal);
            if (equals <= 0)
            {
                // `key =` with an empty value loses its trailing space when trimmed.
                string trimmed = line.TrimEnd(' ', '\t');
                if (trimmed.EndsWith(" =", StringComparison.Ordinal))
                {
                    result.Add(new ConfigEntry(trimmed[..^2].Trim()));
                }

                continue;
            }

            string key = line[..equals].Trim();
            if (key.Length > 0 && !key.Contains(' ', StringComparison.Ordinal))
            {
                result.Add(new ConfigEntry(key, line[(equals + 3)..].Trim()));
            }
        }

        return result;
    }

    /// <summary>
    /// Extracts the download progress from a line of <c>vfox install</c> output. The progress bar is
    /// written to standard error and redrawn with carriage returns:
    /// <c>Downloading...  42% [=====&gt;    ] (1.2 MB/s) [3s:4s]</c>.
    /// </summary>
    /// <param name="line">The line.</param>
    /// <returns>The fraction between 0 and 1, or <see langword="null"/> if the line is not a progress line.</returns>
    public static double? ParseDownloadProgress(string line)
    {
        Match match = DownloadProgressPattern().Match(line);
        if (!match.Success || !int.TryParse(match.Groups[1].Value, NumberStyles.None, CultureInfo.InvariantCulture, out int percent))
        {
            return null;
        }

        return Math.Clamp(percent, 0, 100) / 100.0;
    }

    /// <summary>
    /// Picks a short, human readable error message from a failed command.
    /// </summary>
    /// <remarks>
    /// Usage errors (<c>cli.Exit</c>) go to standard error. The error a command returns goes to standard
    /// output up to vfox 1.0.12 and to standard error from 1.0.13, while what a command prints before
    /// failing, such as the reason of a failed installation, stays on standard output. So the lines of
    /// standard output are read first and those of standard error last. Progress bar redraws and blank
    /// lines are ignored, and the last meaningful line wins, unless it is the generic
    /// <c>failed to install &lt;name&gt;</c> trailer, in which case the more specific line printed before it
    /// is used.
    /// </remarks>
    /// <param name="stdout">The standard output.</param>
    /// <param name="stderr">The standard error.</param>
    /// <returns>The message, or an empty string if there is none.</returns>
    public static string SummarizeError(string stdout, string stderr)
    {
        List<string> lines = [.. MeaningfulLines(stdout), .. MeaningfulLines(stderr)];
        if (lines.Count == 0)
        {
            return string.Empty;
        }

        return lines.Count > 1 && FailedToInstallPattern().IsMatch(lines[^1]) ? lines[^2] : lines[^1];
    }

    private static AvailableVersion? ParseSearchEntry(string entry)
    {
        if (entry.Length == 0)
        {
            return null;
        }

        int firstSpace = entry.IndexOf(' ', StringComparison.Ordinal);
        if (firstSpace < 0)
        {
            return new AvailableVersion(entry);
        }

        string version = entry[..firstSpace];
        string rest = entry[(firstSpace + 1)..].Trim();
        bool installed = false;
        IReadOnlyList<VersionAddition>? additions = null;
        string? note = null;

        const string installedSuffix = "(installed)";
        if (rest.EndsWith(installedSuffix, StringComparison.Ordinal))
        {
            installed = true;
            rest = rest[..^installedSuffix.Length].Trim();
        }

        if (rest.EndsWith(']') && rest.LastIndexOf('[') is var open and >= 0)
        {
            additions = ParseAdditions(rest[(open + 1)..^1]);
            rest = rest[..open].Trim();
        }

        if (rest.StartsWith('(') && rest.EndsWith(')'))
        {
            string inner = rest[1..^1].Trim();
            if (inner.Length > 0)
            {
                note = inner;
            }
        }
        else if (rest.Length > 0)
        {
            note = rest;
        }

        return new AvailableVersion(version, note, additions, installed);
    }

    private static List<VersionAddition> ParseAdditions(string text)
    {
        List<VersionAddition> result = [];
        foreach (string part in text.Split(','))
        {
            string item = part.Trim();
            if (item.Length == 0)
            {
                continue;
            }

            int space = item.IndexOf(' ', StringComparison.Ordinal);
            result.Add(space < 0 ? new VersionAddition(item, string.Empty) : new VersionAddition(item[..space], item[(space + 1)..].Trim()));
        }

        return result;
    }

    private static List<string> MeaningfulLines(string text)
    {
        List<string> result = [];
        foreach (string line in Lines(text))
        {
            string trimmed = line.Trim();
            if (trimmed.Length > 0 && ParseDownloadProgress(trimmed) is null)
            {
                result.Add(trimmed);
            }
        }

        return result;
    }

    private static string StripVPrefix(string version) =>
        version.Length > 1 && version[0] == 'v' ? version[1..] : version;

    private static string[] Lines(string output) => Terminal.SplitLines(Terminal.StripAnsi(output));

    [GeneratedRegex("([0-9]+\\.[0-9]+\\.[0-9]+[0-9A-Za-z.+-]*)", RegexOptions.CultureInvariant)]
    private static partial Regex VersionPattern();

    [GeneratedRegex("^([│ ]*)[├└]─[─┬](.+)$", RegexOptions.CultureInvariant)]
    private static partial Regex TreeItemPattern();

    [GeneratedRegex("^\\s+(\\S+)\\s+([✓✗])\\s+(\\S+)\\s*$", RegexOptions.CultureInvariant)]
    private static partial Regex AvailablePluginPattern();

    [GeneratedRegex("Downloading\\.\\.\\.\\s+([0-9]{1,3})%", RegexOptions.CultureInvariant)]
    private static partial Regex DownloadProgressPattern();

    [GeneratedRegex("^failed to install\\b", RegexOptions.CultureInvariant)]
    private static partial Regex FailedToInstallPattern();
}
