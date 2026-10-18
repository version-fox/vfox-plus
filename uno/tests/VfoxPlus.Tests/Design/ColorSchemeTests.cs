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
using System.Text.Json;
using VfoxPlus.Design.Color;
using Xunit;

namespace VfoxPlus.Tests.Design;

public class ColorSchemeTests
{
    private static string Hex(uint argb) => "#" + (argb & 0xFFFFFF).ToString("x6", CultureInfo.InvariantCulture);

    [Fact]
    public void SchemesMatchTheJavaScriptImplementation()
    {
        // The golden file is the output of testdata/generate.mjs: for every seed, the roles of the light and the
        // dark scheme, as the JavaScript implementation computes them.
        using JsonDocument golden = JsonDocument.Parse(File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "Design", "testdata", "golden.json")));
        int checkedColours = 0;
        List<string> mismatches = [];
        foreach (JsonProperty seedEntry in golden.RootElement.EnumerateObject())
        {
            uint seed = 0xFF000000 | uint.Parse(seedEntry.Name.AsSpan(1), NumberStyles.HexNumber, CultureInfo.InvariantCulture);
            foreach (bool dark in new[] { false, true })
            {
                JsonElement expected = seedEntry.Value.GetProperty(dark ? "dark" : "light");
                foreach ((string name, uint argb) in ColorScheme.Generate(seed, dark).Tokens())
                {
                    string want = expected.GetProperty(name).GetString()!;
                    checkedColours++;
                    if (Hex(argb) != want)
                    {
                        mismatches.Add($"seed {seedEntry.Name} {(dark ? "dark" : "light")} {name}: got {Hex(argb)}, want {want}");
                    }
                }
            }
        }

        Assert.True(mismatches.Count == 0, $"{mismatches.Count} of {checkedColours} colours differ:\n{string.Join('\n', mismatches.Take(20))}");
        Assert.True(checkedColours >= 46 * 2 * 48, $"only {checkedColours} colours were checked");
    }

    [Fact]
    public void TheDefaultSeedGivesTheOrangeScheme()
    {
        ColorScheme light = ColorScheme.Generate(AppSettingsDefaults.Seed, dark: false);
        ColorScheme dark = ColorScheme.Generate(AppSettingsDefaults.Seed, dark: true);
        Assert.Equal("#fff8f5", Hex(light.Surface));
        Assert.Equal("#88521c", Hex(light.Primary));
        Assert.Equal("#ffdcc1", Hex(light.PrimaryContainer));
        Assert.Equal("#221a14", Hex(light.OnSurface));
        Assert.Equal("#ffb779", Hex(dark.Primary));
        Assert.Equal(0xFFu, light.Surface >> 24);
    }

    [Fact]
    public void HctRoundTrips()
    {
        foreach (uint argb in new uint[] { 0xFF000000, 0xFFFFFFFF, 0xFFF28C1B, 0xFF0061A4, 0xFF808080 })
        {
            Hct hct = Hct.FromArgb(argb);
            Assert.Equal(Hex(argb), Hex(hct.Argb));
            Hct again = Hct.From(hct.Hue, hct.Chroma, hct.Tone);
            Assert.True(Hex(again.Argb) == Hex(argb), $"{Hex(argb)}: solving for {hct.Hue:F1}, {hct.Chroma:F1}, {hct.Tone:F1} gave {Hex(again.Argb)}");
        }

        // A chroma of 0 is a grey; 8 bits per channel leave a little chroma.
        Assert.True(Hct.From(10, 0, 50).Chroma <= 3);
    }

    [Fact]
    public void TonalPalettesHaveStableTonesAndAverageTheNeighboursOfAYellow99()
    {
        TonalPalette palette = new(60, 36);
        Assert.Equal("#000000", Hex(palette.Tone(0)));
        Assert.Equal("#ffffff", Hex(palette.Tone(100)));
        Assert.Equal(palette.Tone(40), palette.Tone(40));
        Assert.InRange(palette.HctAt(40).Tone, 39, 41);
        Assert.Equal(60, palette.Hue);
        Assert.Equal(36, palette.Chroma);

        // Tone 99 of a yellow is the average of its neighbours.
        TonalPalette yellow = new(110, 36);
        uint a = yellow.Tone(98);
        uint b = yellow.Tone(99);
        uint c = yellow.Tone(100);
        Assert.False(b == a && b == c, $"{Hex(a)} {Hex(b)} {Hex(c)}");
        Assert.InRange(b >> 16 & 255, Math.Min(a >> 16 & 255, c >> 16 & 255), Math.Max(a >> 16 & 255, c >> 16 & 255));
    }

    [Fact]
    public void TheTokensListEveryRoleOnceWithTheNamesOfTheWebFrontEnd()
    {
        IReadOnlyList<(string Name, uint Argb)> tokens = ColorScheme.Generate(0xFFF28C1B, dark: false).Tokens();
        Assert.Equal(tokens.Count, tokens.Select(token => token.Name).Distinct().Count());
        Assert.Contains(tokens, token => token.Name == "surface-container-low");
        Assert.Contains(tokens, token => token.Name == "on-tertiary-fixed-variant");
        Assert.Equal(48, tokens.Count);
    }
}

internal static class AppSettingsDefaults
{
    public const uint Seed = VfoxPlus.Core.Settings.AppSettings.DefaultSeedColor;
}
