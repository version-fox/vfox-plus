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
using VfoxPlus.Core.I18n;
using VfoxPlus.Core.Settings;
using VfoxPlus.Tests.Support;
using Xunit;

namespace VfoxPlus.Tests.I18n;

public class I18nTests
{
    private static string Directory => Path.Combine(TestPaths.SharedDirectory, "i18n");

    private static IReadOnlyDictionary<string, object?> Args(params (string, object?)[] pairs) =>
        pairs.ToDictionary(pair => pair.Item1, pair => pair.Item2);

    private static string Format(string message, string language, params (string, object?)[] pairs) =>
        MessageFormat.Format(MessageFormat.Parse(message), Args(pairs), language);

    [Fact]
    public void PluralsChooseTheExactCaseThenTheCategory()
    {
        const string message = "{count, plural, =0{none} =1{one {name}} other{{count} of {name}}}!";
        Assert.Equal("none!", Format(message, "en", ("count", 0), ("name", "x")));
        Assert.Equal("one x!", Format(message, "en", ("count", 1), ("name", "x")));
        Assert.Equal("7 of x!", Format(message, "en", ("count", 7), ("name", "x")));
        Assert.Equal("2.5 of x!", Format(message, "en", ("count", 2.5), ("name", "x")));
        Assert.Equal("none!", Format(message, "en", ("count", "oops"), ("name", "x")));
    }

    [Fact]
    public void AnEmptyExactCaseWins()
    {
        Assert.Equal(string.Empty, Format("{n, plural, =0{} other{some}}", "en", ("n", 0)));
    }

    [Fact]
    public void ChineseHasOnlyTheOtherCategory()
    {
        const string message = "{n, plural, one{single} other{many}}";
        Assert.Equal("single", Format(message, "en", ("n", 1)));
        Assert.Equal("single", Format(message, "en", ("n", 1.0)));
        Assert.Equal("many", Format(message, "zh", ("n", 1)));
    }

    [Fact]
    public void UnknownPlaceholdersStayAndApostrophesAreLiteral()
    {
        Assert.Equal("don't {x}", Format("don't {x}", "en"));
        Assert.Equal("2", Format("{x}", "en", ("x", 2.0)));
    }

    [Theory]
    [InlineData("{a, select, x{y} other{z}}")]
    [InlineData("{a, plural, =1{y}}")]
    [InlineData("a } b")]
    [InlineData("{1}")]
    [InlineData("{a, plural x}")]
    public void MalformedMessagesAreRejected(string source)
    {
        Assert.Throws<MessageFormatException>(() => MessageFormat.Parse(source));
    }

    [Fact]
    public void TheCataloguesHaveTheSameDescribedMessagesWithTheSamePlaceholders()
    {
        Assert.Equal(["app_en.arb", "app_zh.arb"], System.IO.Directory.GetFiles(Directory).Select(Path.GetFileName).Order(StringComparer.Ordinal));
        using JsonDocument english = JsonDocument.Parse(File.ReadAllText(Path.Combine(Directory, "app_en.arb")));
        using JsonDocument chinese = JsonDocument.Parse(File.ReadAllText(Path.Combine(Directory, "app_zh.arb")));
        Assert.Equal("en", english.RootElement.GetProperty("@@locale").GetString());
        Assert.Equal("zh", chinese.RootElement.GetProperty("@@locale").GetString());

        string[] englishKeys = MessageKeys(english);
        Assert.Equal(englishKeys, MessageKeys(chinese));
        foreach (string key in englishKeys)
        {
            Assert.True(english.RootElement.TryGetProperty("@" + key, out JsonElement meta) && meta.TryGetProperty("description", out _), $"{key} is not described");
            HashSet<string> template = Placeholders(MessageFormat.Parse(english.RootElement.GetProperty(key).GetString()!));
            HashSet<string> translated = Placeholders(MessageFormat.Parse(chinese.RootElement.GetProperty(key).GetString()!));
            Assert.True(template.IsSubsetOf(translated), $"{key} lacks a placeholder in app_zh.arb");
            HashSet<string> declared = meta.TryGetProperty("placeholders", out JsonElement placeholders)
                ? [.. placeholders.EnumerateObject().Select(property => property.Name)]
                : [];
            Assert.True(template.SetEquals(declared), $"{key}: placeholders {string.Join(',', template)}, declared {string.Join(',', declared)}");
        }
    }

    [Fact]
    public void LocalizationsFormatInTheirLanguageAndFallBackToEnglish()
    {
        ArbCatalogue english = ArbCatalogue.Parse(File.ReadAllText(Path.Combine(Directory, "app_en.arb")));
        ArbCatalogue chinese = ArbCatalogue.Parse(File.ReadAllText(Path.Combine(Directory, "app_zh.arb")));
        Localizations en = new(Languages.English, english, english);
        Localizations zh = new(Languages.SimplifiedChinese, chinese, english);
        Assert.Equal("2 versions installed", en.T("sdksVersionCount", ("count", 2)));
        Assert.Equal("1 version installed", en.T("sdksVersionCount", ("count", 1)));
        Assert.Equal("已安装 2 个版本", zh.T("sdksVersionCount", ("count", 2)));
        Assert.Equal("安装 maven@3.9.16", zh.T("taskInstall", ("subject", "maven@3.9.16")));

        Localizations fallback = new(Languages.SimplifiedChinese, ArbCatalogue.Parse("""{"navHome":"首页"}"""), english);
        Assert.Equal("首页", fallback.T("navHome"));
        Assert.Equal("SDKs", fallback.T("navSdks"));
        Assert.Equal("noSuchKey", en.T("noSuchKey"));
        Assert.Equal("en", en.LanguageTag);
        Assert.Equal("zh-Hans", zh.LanguageTag);
        Assert.Contains("navHome", english.Keys);
    }

    [Fact]
    public void ABrokenMessageShowsItsKey()
    {
        ArbCatalogue broken = ArbCatalogue.Parse("""{"bad":"{x, plural, =1{y}}","empty":""}""");
        Localizations localizations = new(Languages.English, broken, broken);
        Assert.Equal("bad", localizations.T("bad"));
        Assert.Equal("empty", localizations.T("empty"));
    }

    [Theory]
    [InlineData(LanguagePreference.System, "zh-TW", Languages.SimplifiedChinese)]
    [InlineData(LanguagePreference.System, "zh_CN.UTF-8", Languages.SimplifiedChinese)]
    [InlineData(LanguagePreference.System, "de-DE,en-US", Languages.English)]
    [InlineData(LanguagePreference.System, "fr", Languages.English)]
    [InlineData(LanguagePreference.English, "zh-CN", Languages.English)]
    [InlineData(LanguagePreference.SimplifiedChinese, "en", Languages.SimplifiedChinese)]
    public void TheLanguageFollowsThePreferenceThenTheDesktop(LanguagePreference preference, string locales, string expected)
    {
        Assert.Equal(expected, Languages.Resolve(preference, locales.Split(',')));
    }

    [Fact]
    public void TheSystemLocalesAreReadLikeGLibDoes()
    {
        Assert.Equal(
            ["zh_CN", "en", "de_DE.UTF-8"],
            Languages.SystemLocales(new Dictionary<string, string> { ["LANGUAGE"] = "zh_CN:en", ["LC_ALL"] = string.Empty, ["LC_MESSAGES"] = "C.UTF-8", ["LANG"] = "de_DE.UTF-8" }));
        Assert.Empty(Languages.SystemLocales(new Dictionary<string, string> { ["LANG"] = "C" }));
        Assert.Empty(Languages.SystemLocales(new Dictionary<string, string> { ["LANG"] = "POSIX" }));
    }

    private static string[] MessageKeys(JsonDocument catalogue) =>
        [.. catalogue.RootElement.EnumerateObject().Select(property => property.Name).Where(name => !name.StartsWith('@')).Order(StringComparer.Ordinal)];

    private static HashSet<string> Placeholders(IReadOnlyList<MessagePart> parts)
    {
        HashSet<string> names = [];
        void Walk(IReadOnlyList<MessagePart> items)
        {
            foreach (MessagePart part in items)
            {
                switch (part)
                {
                    case ArgumentPart argument:
                        names.Add(argument.Name);
                        break;
                    case PluralPart plural:
                        names.Add(plural.Name);
                        foreach (IReadOnlyList<MessagePart> nested in plural.Cases.Values)
                        {
                            Walk(nested);
                        }

                        break;
                }
            }
        }

        Walk(parts);
        return names;
    }
}
