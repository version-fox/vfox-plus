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

using VfoxPlus.Core.I18n;
using VfoxPlus.Core.Settings;

namespace VfoxPlus.App.Services;

/// <summary>
/// The messages of the user interface in the language of the user: the catalogues of <c>shared/i18n</c>, which the
/// build copies to <c>Assets/Generated/i18n</c>. A change of language is announced, and what shows text sets it again.
/// </summary>
internal sealed class L10n
{
    private readonly string directory;
    private readonly ArbCatalogue english;
    private Localizations current;

    public L10n(string directory)
    {
        this.directory = directory;
        english = Read(Languages.English);
        current = new Localizations(Languages.English, english, english);
    }

    /// <summary>Raised when the language changed.</summary>
    public event Action? Changed;

    /// <summary>Gets the code of the language of the messages.</summary>
    public string Language => current.Language;

    /// <summary>Returns a message in the language of the user interface.</summary>
    /// <param name="key">The key of the message.</param>
    /// <param name="args">The arguments of its placeholders: names and values.</param>
    /// <returns>The text.</returns>
    public string T(string key, params (string Name, object? Value)[] args) => current.T(key, args);

    /// <summary>Loads the messages of the language the user asks for, and announces it if it changed.</summary>
    /// <param name="preference">The choice of the user.</param>
    /// <param name="locales">The locales of the desktop.</param>
    public void Select(LanguagePreference preference, IReadOnlyList<string> locales)
    {
        string language = Languages.Resolve(preference, locales);
        if (language == current.Language)
        {
            return;
        }

        ArbCatalogue messages = language == Languages.English ? english : Read(language);
        current = new Localizations(language, messages, english);
        Changed?.Invoke();
    }

    private ArbCatalogue Read(string language) =>
        ArbCatalogue.Parse(File.ReadAllText(Path.Combine(directory, $"app_{language}.arb")));
}
