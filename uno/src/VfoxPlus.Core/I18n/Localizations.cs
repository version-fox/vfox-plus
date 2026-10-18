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

using System.Collections.Concurrent;
using System.Text.Json;
using VfoxPlus.Core.Settings;

namespace VfoxPlus.Core.I18n;

/// <summary>
/// The messages of an ARB catalogue: the <c>app_&lt;language&gt;.arb</c> files of <c>shared/i18n</c>,
/// which are bundled as they are, without their <c>@</c>-prefixed metadata.
/// </summary>
public sealed class ArbCatalogue
{
    private readonly Dictionary<string, string> messages;

    private ArbCatalogue(Dictionary<string, string> messages) => this.messages = messages;

    /// <summary>Gets the message keys.</summary>
    public IReadOnlyCollection<string> Keys => messages.Keys;

    /// <summary>Parses the content of an ARB file.</summary>
    /// <param name="json">The JSON text.</param>
    /// <returns>The catalogue.</returns>
    public static ArbCatalogue Parse(string json)
    {
        Dictionary<string, string> messages = [];
        using JsonDocument document = JsonDocument.Parse(json);
        foreach (JsonProperty property in document.RootElement.EnumerateObject())
        {
            if (!property.Name.StartsWith('@') && property.Value.ValueKind == JsonValueKind.String)
            {
                messages[property.Name] = property.Value.GetString()!;
            }
        }

        return new ArbCatalogue(messages);
    }

    /// <summary>Looks up the source of a message.</summary>
    /// <param name="key">The key.</param>
    /// <param name="message">The message in ICU MessageFormat.</param>
    /// <returns><see langword="true"/> if the catalogue has the key.</returns>
    public bool TryGetMessage(string key, out string message)
    {
        bool found = messages.TryGetValue(key, out string? value) && value.Length > 0;
        message = found ? value! : string.Empty;
        return found;
    }
}

/// <summary>
/// The languages of the user interface, and how one is chosen.
/// </summary>
public static class Languages
{
    /// <summary>The code of English, the template and the fallback of the catalogues.</summary>
    public const string English = "en";

    /// <summary>The code of Simplified Chinese.</summary>
    public const string SimplifiedChinese = "zh";

    /// <summary>
    /// Returns the locales of the desktop, most preferred first, as GLib reads them from the environment.
    /// </summary>
    /// <param name="environment">The environment.</param>
    /// <returns>The locales, without <c>C</c> and <c>POSIX</c>.</returns>
    public static IReadOnlyList<string> SystemLocales(IReadOnlyDictionary<string, string> environment)
    {
        List<string> locales = [];
        void Add(string? value)
        {
            foreach (string locale in (value ?? string.Empty).Split(':'))
            {
                if (locale.Length > 0 && locale is not ("C" or "POSIX") && !locale.StartsWith("C.", StringComparison.Ordinal))
                {
                    locales.Add(locale);
                }
            }
        }

        Add(environment.GetValueOrDefault("LANGUAGE"));
        foreach (string key in new[] { "LC_ALL", "LC_MESSAGES", "LANG" })
        {
            Add(environment.GetValueOrDefault(key));
        }

        return locales;
    }

    /// <summary>
    /// Picks the catalogue for a preference. With <see cref="LanguagePreference.System"/>, the desktop locales
    /// are matched against the catalogues; every Chinese locale gets Simplified Chinese and English is the
    /// fallback.
    /// </summary>
    /// <param name="preference">The preference of the user.</param>
    /// <param name="locales">The locales of the desktop.</param>
    /// <returns>The language code.</returns>
    public static string Resolve(LanguagePreference preference, IReadOnlyList<string> locales)
    {
        switch (preference)
        {
            case LanguagePreference.SimplifiedChinese:
                return SimplifiedChinese;
            case LanguagePreference.English:
                return English;
        }

        foreach (string locale in locales)
        {
            string language = locale.ToLowerInvariant();
            int end = language.AsSpan().IndexOfAny("-_.@");
            if (end >= 0)
            {
                language = language[..end];
            }

            if (language == "zh")
            {
                return SimplifiedChinese;
            }

            if (language == "en")
            {
                return English;
            }
        }

        return English;
    }
}

/// <summary>
/// The messages of one language. English is the fallback of a key that the language lacks. It is safe for
/// use from several threads.
/// </summary>
public sealed class Localizations
{
    private readonly ArbCatalogue messages;
    private readonly ArbCatalogue fallback;
    private readonly ConcurrentDictionary<string, IReadOnlyList<MessagePart>> cache = new();

    /// <summary>Initializes a new instance of the <see cref="Localizations"/> class.</summary>
    /// <param name="language">The language code of the messages.</param>
    /// <param name="messages">The catalogue of the language.</param>
    /// <param name="fallback">The catalogue to use for the keys the language lacks.</param>
    public Localizations(string language, ArbCatalogue messages, ArbCatalogue fallback)
    {
        Language = language;
        this.messages = messages;
        this.fallback = fallback;
    }

    /// <summary>Gets the language code of the messages.</summary>
    public string Language { get; }

    /// <summary>Gets the BCP 47 tag of the language.</summary>
    public string LanguageTag => Language == Languages.SimplifiedChinese ? "zh-Hans" : "en";

    /// <summary>
    /// Formats a message. A key that the language lacks falls back to English; a key that no catalogue has
    /// is returned as it is, which the tests catch.
    /// </summary>
    /// <param name="key">The key of the message.</param>
    /// <param name="args">The arguments of its placeholders: names and values.</param>
    /// <returns>The text.</returns>
    public string T(string key, params (string Name, object? Value)[] args)
    {
        IReadOnlyList<MessagePart> parts = cache.GetOrAdd(key, Parse);
        return MessageFormat.Format(parts, args.ToDictionary(pair => pair.Name, pair => pair.Value), Language);
    }

    private IReadOnlyList<MessagePart> Parse(string key)
    {
        if (!messages.TryGetMessage(key, out string source) && !fallback.TryGetMessage(key, out source))
        {
            return [new TextPart(key)];
        }

        try
        {
            return MessageFormat.Parse(source);
        }
        catch (MessageFormatException)
        {
            return [new TextPart(key)];
        }
    }
}
