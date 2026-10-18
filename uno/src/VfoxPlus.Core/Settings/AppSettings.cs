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

using System.Collections.Immutable;
using System.Text;
using System.Text.Encodings.Web;
using System.Text.Json;
using System.Text.Json.Nodes;

namespace VfoxPlus.Core.Settings;

/// <summary>The light or dark appearance; <see cref="System"/> follows the desktop.</summary>
public enum ThemePreference
{
    /// <summary>Follows the desktop.</summary>
    System,

    /// <summary>The light appearance.</summary>
    Light,

    /// <summary>The dark appearance.</summary>
    Dark,
}

/// <summary>The language choice of the user; <see cref="System"/> follows the desktop locale.</summary>
public enum LanguagePreference
{
    /// <summary>Follows the desktop locale.</summary>
    System,

    /// <summary>English.</summary>
    English,

    /// <summary>Simplified Chinese.</summary>
    SimplifiedChinese,
}

/// <summary>
/// The preferences of the application itself (not of vfox), persisted as JSON.
/// </summary>
/// <remarks>
/// The JSON keys are part of the shared specification in <c>docs/architecture.md#settings</c>, so that every
/// implementation of the GUI can read the same file.
/// </remarks>
public sealed record AppSettings
{
    /// <summary>
    /// The brand colour used when the desktop accent colour is not used or not available: the orange of the
    /// vfox logo.
    /// </summary>
    public const uint DefaultSeedColor = 0xFFF28C1B;

    private static readonly ImmutableDictionary<string, string> NoEnvironment = ImmutableDictionary<string, string>.Empty;
    private static readonly ImmutableDictionary<string, JsonNode?> NoUnknown = ImmutableDictionary<string, JsonNode?>.Empty;

    /// <summary>Gets the light or dark appearance.</summary>
    public ThemePreference Theme { get; init; } = ThemePreference.System;

    /// <summary>Gets a value indicating whether the colour scheme is derived from the desktop accent colour.</summary>
    public bool UseSystemColor { get; init; } = true;

    /// <summary>Gets the ARGB seed of the colour scheme when <see cref="UseSystemColor"/> is off.</summary>
    public uint SeedColor { get; init; } = DefaultSeedColor;

    /// <summary>Gets the user interface language.</summary>
    public LanguagePreference Language { get; init; } = LanguagePreference.System;

    /// <summary>Gets the path of the vfox executable chosen by the user, or an empty string to detect it.</summary>
    public string VfoxExecutable { get; init; } = string.Empty;

    /// <summary>Gets the additional environment variables for vfox, such as mirror settings.</summary>
    public IReadOnlyDictionary<string, string> ExtraEnvironment { get; init; } = NoEnvironment;

    /// <summary>Gets a value indicating whether the navigation rail shows labels next to the icons.</summary>
    public bool RailExpanded { get; init; }

    /// <summary>
    /// Gets the keys of the settings file that this version does not know, kept as they are so that settings
    /// written by another implementation survive.
    /// </summary>
    public IReadOnlyDictionary<string, JsonNode?> Unknown { get; init; } = NoUnknown;

    /// <summary>Determines whether two settings are equal, comparing the environments by value.</summary>
    /// <param name="other">The other settings.</param>
    /// <returns><see langword="true"/> if both are equal.</returns>
    public bool Equals(AppSettings? other) =>
        other is not null
        && Theme == other.Theme
        && UseSystemColor == other.UseSystemColor
        && SeedColor == other.SeedColor
        && Language == other.Language
        && VfoxExecutable == other.VfoxExecutable
        && RailExpanded == other.RailExpanded
        && ExtraEnvironment.Count == other.ExtraEnvironment.Count
        && ExtraEnvironment.All(pair => other.ExtraEnvironment.TryGetValue(pair.Key, out string? value) && value == pair.Value)
        && UnknownJson(Unknown) == UnknownJson(other.Unknown);

    /// <summary>Gets a hash code of the settings.</summary>
    /// <returns>The hash code.</returns>
    public override int GetHashCode() => HashCode.Combine(Theme, UseSystemColor, SeedColor, Language, VfoxExecutable, RailExpanded);

    /// <summary>Restores settings from JSON text, ignoring malformed values.</summary>
    /// <param name="json">The text.</param>
    /// <returns>The settings; the defaults if the text is not a JSON object.</returns>
    public static AppSettings FromJson(string json)
    {
        JsonObject? root;
        try
        {
            root = JsonNode.Parse(json) as JsonObject;
        }
        catch (JsonException)
        {
            return new AppSettings();
        }

        return root is null ? new AppSettings() : FromObject(root);
    }

    /// <summary>
    /// Serialises the settings, with the keys in the order of the specification and the unknown keys
    /// last.
    /// </summary>
    /// <returns>The JSON text, indented by two spaces.</returns>
    public string ToJson()
    {
        using MemoryStream stream = new();
        using (Utf8JsonWriter writer = new(stream, new JsonWriterOptions { Indented = true, Encoder = JavaScriptEncoder.UnsafeRelaxedJsonEscaping }))
        {
            writer.WriteStartObject();
            writer.WriteString("theme", ThemeName(Theme));
            writer.WriteBoolean("useSystemColor", UseSystemColor);
            writer.WriteNumber("seedColor", SeedColor);
            writer.WriteString("language", LanguageName(Language));
            if (VfoxExecutable.Length > 0)
            {
                writer.WriteString("vfoxExecutable", VfoxExecutable);
            }

            writer.WriteStartObject("extraEnvironment");
            foreach ((string key, string value) in ExtraEnvironment.OrderBy(pair => pair.Key, StringComparer.Ordinal))
            {
                writer.WriteString(key, value);
            }

            writer.WriteEndObject();
            writer.WriteBoolean("railExpanded", RailExpanded);
            foreach ((string key, JsonNode? value) in Unknown.OrderBy(pair => pair.Key, StringComparer.Ordinal))
            {
                writer.WritePropertyName(key);
                if (value is null)
                {
                    writer.WriteNullValue();
                }
                else
                {
                    value.WriteTo(writer);
                }
            }

            writer.WriteEndObject();
        }

        return Encoding.UTF8.GetString(stream.ToArray()) + "\n";
    }

    private static string UnknownJson(IReadOnlyDictionary<string, JsonNode?> unknown) =>
        string.Join(',', unknown.OrderBy(pair => pair.Key, StringComparer.Ordinal).Select(pair => $"{pair.Key}:{pair.Value?.ToJsonString()}"));

    private static string ThemeName(ThemePreference theme) => theme switch
    {
        ThemePreference.Light => "light",
        ThemePreference.Dark => "dark",
        _ => "system",
    };

    private static string LanguageName(LanguagePreference language) => language switch
    {
        LanguagePreference.English => "english",
        LanguagePreference.SimplifiedChinese => "simplifiedChinese",
        _ => "system",
    };

    private static AppSettings FromObject(JsonObject root)
    {
        AppSettings settings = new();
        if (Text(root, "theme") is { } theme)
        {
            settings = settings with
            {
                Theme = theme switch
                {
                    "light" => ThemePreference.Light,
                    "dark" => ThemePreference.Dark,
                    "system" => ThemePreference.System,
                    _ => settings.Theme,
                },
            };
        }

        if (Boolean(root, "useSystemColor") is { } useSystemColor)
        {
            settings = settings with { UseSystemColor = useSystemColor };
        }

        // A boolean is not a colour, nor is a number that no ARGB value can be.
        if (root["seedColor"] is JsonValue seed && seed.TryGetValue(out double color)
            && color >= 0 && color <= uint.MaxValue && color == Math.Floor(color))
        {
            settings = settings with { SeedColor = (uint)color };
        }

        if (Text(root, "language") is { } language)
        {
            settings = settings with
            {
                Language = language switch
                {
                    "english" => LanguagePreference.English,
                    "simplifiedChinese" => LanguagePreference.SimplifiedChinese,
                    "system" => LanguagePreference.System,
                    _ => settings.Language,
                },
            };
        }

        if (Text(root, "vfoxExecutable") is { Length: > 0 } executable)
        {
            settings = settings with { VfoxExecutable = executable };
        }

        if (root["extraEnvironment"] is JsonObject environment)
        {
            Dictionary<string, string> variables = [];
            foreach ((string key, JsonNode? value) in environment)
            {
                if (value is JsonValue text && text.TryGetValue(out string? content))
                {
                    variables[key] = content;
                }
            }

            settings = settings with { ExtraEnvironment = variables.ToImmutableDictionary() };
        }

        if (Boolean(root, "railExpanded") is true)
        {
            settings = settings with { RailExpanded = true };
        }

        string[] known = ["theme", "useSystemColor", "seedColor", "language", "vfoxExecutable", "extraEnvironment", "railExpanded"];
        Dictionary<string, JsonNode?> unknown = [];
        foreach ((string key, JsonNode? value) in root)
        {
            if (!known.Contains(key))
            {
                unknown[key] = value?.DeepClone();
            }
        }

        return settings with { Unknown = unknown.ToImmutableDictionary() };
    }

    private static string? Text(JsonObject root, string key) =>
        root[key] is JsonValue value && value.TryGetValue(out string? text) ? text : null;

    private static bool? Boolean(JsonObject root, string key) =>
        root[key] is JsonValue value && value.TryGetValue(out bool flag) ? flag : null;
}

/// <summary>
/// The extra environment variables of vfox as the text the user edits: one <c>KEY=VALUE</c> per line.
/// </summary>
public static class EnvironmentLines
{
    /// <summary>
    /// Parses <c>KEY=VALUE</c> lines into environment variables. Blank lines and lines starting with
    /// <c>#</c> are ignored. Keys must be valid POSIX variable names.
    /// </summary>
    /// <param name="text">The text.</param>
    /// <returns>The variables of the valid lines, and the 1-based numbers of the lines that could not be parsed.</returns>
    public static (IReadOnlyDictionary<string, string> Variables, IReadOnlyList<int> InvalidLines) Parse(string text)
    {
        Dictionary<string, string> variables = [];
        List<int> invalid = [];
        string[] lines = text.Split('\n');
        for (int index = 0; index < lines.Length; index++)
        {
            string line = lines[index].Trim();
            if (line.Length == 0 || line.StartsWith('#'))
            {
                continue;
            }

            int equals = line.IndexOf('=', StringComparison.Ordinal);
            string key = equals < 0 ? string.Empty : line[..equals].Trim();
            if (equals < 0 || !IsKey(key))
            {
                invalid.Add(index + 1);
                continue;
            }

            variables[key] = line[(equals + 1)..].Trim();
        }

        return (variables, invalid);
    }

    /// <summary>Formats environment variables as <c>KEY=VALUE</c> lines, sorted by key.</summary>
    /// <param name="variables">The variables.</param>
    /// <returns>The text.</returns>
    public static string Format(IReadOnlyDictionary<string, string> variables) =>
        string.Join('\n', variables.OrderBy(pair => pair.Key, StringComparer.Ordinal).Select(pair => $"{pair.Key}={pair.Value}"));

    private static bool IsKey(string key)
    {
        if (key.Length == 0)
        {
            return false;
        }

        for (int i = 0; i < key.Length; i++)
        {
            char c = key[i];
            bool valid = c == '_' || c is >= 'a' and <= 'z' || c is >= 'A' and <= 'Z' || (i > 0 && c is >= '0' and <= '9');
            if (!valid)
            {
                return false;
            }
        }

        return true;
    }
}
