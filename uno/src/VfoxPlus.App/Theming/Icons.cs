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

using System.Text.Json;

namespace VfoxPlus.App.Theming;

/// <summary>
/// The icons of the application, by meaning: <c>shared/icons.json</c> says which glyph of Material Symbols Rounded
/// each meaning is, and the build copies the code points to <c>Assets/Generated/icons.json</c>.
/// </summary>
internal static class Icons
{
    private static readonly Lazy<Dictionary<string, string>> Glyphs = new(Load);

    /// <summary>Returns the glyph of an icon in the icon font.</summary>
    /// <param name="meaning">What the icon means, such as <c>sdks</c> or <c>add</c>.</param>
    /// <returns>The character to draw.</returns>
    public static string Glyph(string meaning) =>
        Glyphs.Value.TryGetValue(meaning, out string? glyph) ? glyph : throw new ArgumentException($"There is no icon for {meaning}.", nameof(meaning));

    private static Dictionary<string, string> Load()
    {
        using JsonDocument document = JsonDocument.Parse(File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "Assets", "Generated", "icons.json")));
        Dictionary<string, string> glyphs = [];
        foreach (JsonProperty icon in document.RootElement.GetProperty("icons").EnumerateObject())
        {
            glyphs[icon.Name] = char.ConvertFromUtf32(icon.Value.GetProperty("codepoint").GetInt32());
        }

        return glyphs;
    }
}
