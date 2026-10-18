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

using VfoxPlus.App.Controls;
using VfoxPlus.App.Services;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Platform;

namespace VfoxPlus.App.Pages;

/// <summary>What the rows of the configuration of vfox share: they save a key in the background, and say how that went.</summary>
internal static class ConfigSaving
{
    /// <summary>Saves a value of the configuration of vfox in the background, and says whether that worked.</summary>
    /// <param name="save">Saves the value.</param>
    public static void Run(Func<Task> save)
    {
        Env env = Env.Current;
        _ = Task.Run(async () =>
        {
            try
            {
                await save().ConfigureAwait(false);
                env.Dispatch.Post(() => env.Notify(env.T("settingsSaved")));
            }
            catch (Exception error) when (error is VfoxException or VfoxNotFoundException or IOException)
            {
                env.Dispatch.Post(() => env.Notify(Presentation.Describe(error)));
            }
        });
    }

    /// <summary>Makes the space that an icon takes, for the rows that have none.</summary>
    /// <returns>The space.</returns>
    public static UIElement Blank() => new Border { Width = 24, Height = 24 };
}

/// <summary>A boolean key of the configuration of vfox, as a switch.</summary>
internal sealed class ConfigSwitchRow : ListItem
{
    private readonly Env env = Env.Current;
    private readonly string key;
    private readonly string titleKey;
    private readonly string? helpKey;
    private readonly M3Switch toggle = new();
    private bool on;

    public ConfigSwitchRow(string key, string? icon, string titleKey, string? helpKey)
    {
        this.key = key;
        this.titleKey = titleKey;
        this.helpKey = helpKey;
        Leading = icon is null ? ConfigSaving.Blank() : new M3Icon(icon, 24);
        Trailing = toggle;
        Click += (_, _) => Toggle();
        toggle.Toggled += (_, _) => Toggle();
    }

    /// <summary>Gets the key of the configuration.</summary>
    public string Key => key;

    /// <summary>Gets the switch, which the tests choose.</summary>
    public M3Switch Switch => toggle;

    /// <summary>Shows the value of the key.</summary>
    public void Bind()
    {
        on = env.Model.VfoxConfig.ValueFor(key) == "true";
        bool saving = env.Model.VfoxConfig.IsSaving(key);
        Title = env.T(titleKey);
        Subtitle = helpKey is null ? string.Empty : env.T(helpKey);
        toggle.Label = Title;
        toggle.IsOn = on;
        toggle.IsEnabled = !saving;
        Interactive = !saving;
    }

    private void Toggle() => ConfigSaving.Run(() => env.Model.VfoxConfig.SetAsync(key, on ? "false" : "true"));
}

/// <summary>A key of the configuration of vfox with a few values, as a dropdown.</summary>
internal sealed class ConfigChoiceRow : ListItem
{
    private readonly Env env = Env.Current;
    private readonly string key;
    private readonly string titleKey;
    private readonly (string Value, string LabelKey)[] choices;
    private readonly Dropdown dropdown = new();

    public ConfigChoiceRow(string key, string titleKey, (string Value, string LabelKey)[] choices)
    {
        this.key = key;
        this.titleKey = titleKey;
        this.choices = choices;
        WrapTrailing = true;
        Leading = ConfigSaving.Blank();
        Trailing = dropdown;
        dropdown.Selecting += (_, selection) => ConfigSaving.Run(() => env.Model.VfoxConfig.SetAsync(key, selection));
    }

    /// <summary>Gets the key of the configuration.</summary>
    public string Key => key;

    /// <summary>Gets the dropdown, which the tests choose.</summary>
    public Dropdown Choices => dropdown;

    /// <summary>Shows the value of the key.</summary>
    public void Bind()
    {
        Title = env.T(titleKey);
        dropdown.Choices = [.. choices.Select(choice => new Choice(choice.Value, env.T(choice.LabelKey)))];
        dropdown.Value = env.Model.VfoxConfig.ValueFor(key) ?? string.Empty;
        dropdown.IsEnabled = !env.Model.VfoxConfig.IsSaving(key);
    }
}

/// <summary>
/// A key of the configuration of vfox with a free value, as a text field: it is saved when the user presses Enter or
/// leaves the field, and an empty field resets the key.
/// </summary>
internal sealed class ConfigTextRow : ListItem
{
    private readonly Env env = Env.Current;
    private readonly string key;
    private readonly string? titleKey;
    private readonly string? helpKey;
    private readonly string hint;
    private readonly bool pickDirectory;
    private readonly M3TextField input = new() { Dense = true, Width = 300 };
    private readonly M3IconButton action = new();
    private readonly Grid host = new();
    private string stored = string.Empty;
    private string saved = string.Empty;
    private bool known;

    public ConfigTextRow(string key, string? titleKey, string? icon, string? helpKey, string hint, bool pickDirectory)
    {
        this.key = key;
        this.titleKey = titleKey;
        this.helpKey = helpKey;
        this.hint = hint;
        this.pickDirectory = pickDirectory;
        WrapTrailing = true;
        Leading = icon is null ? ConfigSaving.Blank() : new M3Icon(icon, 24);
        input.Trailing = action;
        host.Children.Add(input);
        Trailing = host;
        input.Submitted += (_, text) => Save(text);
        input.Blurred += (_, text) => Save(text);
        action.Click += (_, _) =>
        {
            if (pickDirectory)
            {
                env.Choose(new ChooseOptions(Title, stored, Directory: true), path =>
                {
                    input.Text = path;
                    Save(path);
                });
            }
            else
            {
                input.Text = string.Empty;
                Save(string.Empty);
            }
        };
    }

    /// <summary>Gets the key of the configuration.</summary>
    public string Key => key;

    /// <summary>Gets the field, which the tests fill.</summary>
    public M3TextField Field => input;

    /// <summary>Shows the value of the key. What the user is typing stays, unless the value changed in vfox.</summary>
    public void Bind()
    {
        string now = env.Model.VfoxConfig.ValueFor(key) ?? string.Empty;
        if (!known || now != stored)
        {
            stored = now;
            saved = now;
            known = true;
            input.Text = now;
        }

        Title = titleKey is null ? key : env.T(titleKey);
        Subtitle = helpKey is null ? string.Empty : env.T(helpKey);
        input.Hint = hint;
        input.IsEnabled = !env.Model.VfoxConfig.IsSaving(key);
        action.Symbol = pickDirectory ? "folder" : "close";
        action.Tooltip = env.T(pickDirectory ? "actionBrowse" : "actionReset");
        action.Visibility = pickDirectory || stored.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
    }

    private void Save(string text)
    {
        string value = text.Trim();

        // Submitting and then leaving the field must not save twice. If vfox rejects the value, the reload shows the
        // stored value again.
        if (value == saved)
        {
            return;
        }

        saved = value;
        ConfigSaving.Run(() => value.Length == 0 ? env.Model.VfoxConfig.ResetAsync(key) : env.Model.VfoxConfig.SetAsync(key, value));
    }
}
