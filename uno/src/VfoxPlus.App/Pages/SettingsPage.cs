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
using VfoxPlus.App.Controls.Overlays;
using VfoxPlus.App.Services;
using VfoxPlus.App.Theming;
using VfoxPlus.Core;
using VfoxPlus.Core.Settings;
using VfoxPlus.Design.Tokens;
using VfoxPlus.Platform;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>The preferences of the application and the configuration of vfox.</summary>
internal sealed class SettingsPage : PageBase
{
    // The accent colours to choose from.
    private static readonly uint[] SeedColors =
    [
        AppSettings.DefaultSeedColor, 0xFFD9473B, 0xFFC2407A, 0xFF7E57C2, 0xFF3F6FD8, 0xFF0B8A9A, 0xFF3E8E41, 0xFF8D7B1E,
    ];

    // The keys of the configuration of vfox that the page has a row for.
    private static readonly string[] KnownKeys =
    [
        "proxy.enable", "proxy.url", "registry.address", "storage.sdkPath", "legacyVersionFile.enable",
        "legacyVersionFile.strategy", "cache.availableHookDuration", "plugin.http.timeout",
        "plugin.http.downloadTimeout", "gitignore.enable",
    ];

    private readonly ScrollPage page = new();
    private readonly Border pane;
    private readonly TextBlock title;

    // Appearance.
    private readonly TextBlock appearanceTitle;
    private readonly ListItem themeItem = new() { Leading = new M3Icon("darkMode", 24), WrapTrailing = true };
    private readonly ButtonGroup themeGroup = new();
    private readonly ListItem systemColorItem = new() { Leading = new M3Icon("palette", 24), Interactive = true };
    private readonly M3Switch systemColorSwitch = new();
    private readonly ListItem colorItem = new() { Leading = ConfigSaving.Blank() };
    private readonly List<ColorSwatch> swatches = [];
    private readonly ListItem languageItem = new() { Leading = new M3Icon("language", 24), WrapTrailing = true };
    private readonly ButtonGroup languageGroup = new();

    // vfox.
    private readonly TextBlock vfoxTitle;
    private readonly ListItem executableItem = new() { Leading = new M3Icon("terminal", 24), WrapTrailing = true };
    private readonly StackPanel executableActions = new() { Orientation = Orientation.Horizontal, Spacing = Gaps.Xs, VerticalAlignment = VerticalAlignment.Center };
    private readonly LoadingIndicator checking = new(36);
    private readonly M3IconButton detect = new() { Symbol = "refresh" };
    private readonly M3Button browse = new() { Kind = ButtonKind.Tonal };
    private readonly ListItem environmentItem = new() { Leading = new M3Icon("environment", 24) };
    private readonly TextBlock environmentHelp = Ui.Text(string.Empty, TextRole.BodyMedium, Ui.Theme.OnSurfaceVariant);
    private readonly M3TextField environment = new() { Lines = 3, MaxLines = 8, Monospace = true, Hint = "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node" };
    private readonly M3Button saveEnvironment = new() { Kind = ButtonKind.Filled, HorizontalAlignment = HorizontalAlignment.Right };

    // The configuration of vfox.
    private readonly TextBlock configTitle;
    private readonly TextBlock configHelp = Ui.Text(string.Empty, TextRole.BodySmall, Ui.Theme.OnSurfaceVariant);
    private readonly StackPanel configBody = new() { Orientation = Orientation.Vertical };
    private readonly ConfigSwitchRow proxyEnable = new("proxy.enable", "proxy", "configProxyEnable", null);
    private readonly ConfigTextRow proxyUrl = new("proxy.url", "configProxyUrl", null, null, "http://127.0.0.1:7890", false);
    private readonly ConfigTextRow registry = new("registry.address", "configRegistry", "registry", null, string.Empty, false);
    private readonly ConfigTextRow storage = new("storage.sdkPath", "configStorage", "storage", null, string.Empty, true);
    private readonly ConfigSwitchRow legacyEnable = new("legacyVersionFile.enable", "legacyFiles", "configLegacyEnable", "configLegacyEnableHelp");
    private readonly ConfigChoiceRow legacyStrategy = new("legacyVersionFile.strategy", "configLegacyStrategy", [("specified", "strategySpecified"), ("latest_installed", "strategyLatestInstalled"), ("latest_available", "strategyLatestAvailable")]);
    private readonly ConfigSwitchRow gitignore = new("gitignore.enable", "gitignore", "configGitignore", "configGitignoreHelp");
    private readonly ConfigTextRow cache = new("cache.availableHookDuration", "configCache", "cache", "configCacheHelp", "12h", false);
    private readonly ConfigTextRow httpTimeout = new("plugin.http.timeout", "configHttpTimeout", "timeout", "configTimeoutHelp", "30s", false);
    private readonly ConfigTextRow downloadTimeout = new("plugin.http.downloadTimeout", "configDownloadTimeout", null, "configTimeoutHelp", "30m", false);
    private readonly Dictionary<string, ConfigTextRow> others = [];
    private readonly ListGroup proxyGroup = new();
    private readonly ListGroup legacyGroup = new() { Margin = new Thickness(0, Gaps.L, 0, 0) };
    private readonly ListGroup timeoutGroup = new() { Margin = new Thickness(0, Gaps.L, 0, 0) };
    private readonly ListGroup otherGroup = new();
    private readonly TextBlock otherTitle;
    private UIElement? configMessage;
    private string configMessageKey = string.Empty;

    // About.
    private readonly TextBlock aboutTitle;
    private readonly ListItem aboutItem = new() { Leading = new Avatar("vfox+", 40, "sdks") };
    private readonly ListItem sourceItem = new() { Leading = new M3Icon("sourceCode", 24), Trailing = new M3Icon("openInNew", 24), Interactive = true };
    private readonly ListItem docsItem = new() { Leading = new M3Icon("docs", 24), Trailing = new M3Icon("openInNew", 24), Interactive = true };
    private readonly ListItem licensesItem = new() { Leading = new M3Icon("license", 24), Trailing = new M3Icon("chevronRight", 24), Interactive = true };

    private bool configRequested;
    private bool environmentReady;

    public SettingsPage()
    {
        (Grid header, title, _) = PageParts.Header(string.Empty, string.Empty);
        page.Top.Children.Add(header);

        StackPanel column = new() { Orientation = Orientation.Vertical };
        ListGroup appearance = new();
        appearance.SetItems([themeItem, systemColorItem, colorItem, languageItem]);
        (Grid appearanceHeader, appearanceTitle) = PageParts.GroupHeader(string.Empty);
        column.Children.Add(appearanceHeader);
        column.Children.Add(appearance);

        ListGroup vfox = new();
        vfox.SetItems([executableItem, environmentItem]);
        (Grid vfoxHeader, vfoxTitle) = PageParts.GroupHeader(string.Empty);
        column.Children.Add(vfoxHeader);
        column.Children.Add(vfox);

        (Grid configHeader, configTitle) = PageParts.GroupHeader(string.Empty);
        configHelp.TextWrapping = TextWrapping.Wrap;
        configHelp.Margin = new Thickness(Gaps.L, 0, 0, Gaps.M);
        column.Children.Add(configHeader);
        column.Children.Add(configHelp);
        column.Children.Add(configBody);

        ListGroup about = new();
        about.SetItems([aboutItem, sourceItem, docsItem, licensesItem]);
        (Grid aboutHeader, aboutTitle) = PageParts.GroupHeader(string.Empty);
        column.Children.Add(aboutHeader);
        column.Children.Add(about);
        (_, otherTitle) = PageParts.GroupHeader(string.Empty);

        BuildAppearance();
        BuildVfox();
        proxyGroup.SetItems([proxyEnable, proxyUrl, registry, storage]);
        legacyGroup.SetItems([legacyEnable, legacyStrategy, gitignore]);
        timeoutGroup.SetItems([cache, httpTimeout, downloadTimeout]);

        sourceItem.Click += (_, _) => Env.OpenUrl(Common.Repository);
        docsItem.Click += (_, _) => Env.OpenUrl(Common.VfoxDocs);
        licensesItem.Click += (_, _) => Dialogs.ShowLicenses();

        Grid width = PageParts.ContentWidth(840, column);
        width.Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, Gaps.Xxl);
        page.ShowContent(width);
        pane = PageParts.Pane(page);
        Children.Add(pane);
        Watch(Env.Model.Settings, Env.Model.Vfox, Env.Model.VfoxConfig);
    }

    /// <summary>Gets the swatches of the accent colours, which the tests choose.</summary>
    public IReadOnlyList<ColorSwatch> Swatches => swatches;

    /// <summary>Gets the group of buttons that choose the theme, which the tests choose.</summary>
    public ButtonGroup ThemeGroup => themeGroup;

    /// <summary>Gets the group of buttons that choose the language, which the tests choose.</summary>
    public ButtonGroup LanguageGroup => languageGroup;

    /// <summary>Gets the switch that makes the application follow the accent colour of the desktop.</summary>
    public M3Switch SystemColorSwitch => systemColorSwitch;

    /// <summary>Gets the field that holds the environment variables of vfox.</summary>
    public M3TextField EnvironmentField => environment;

    /// <inheritdoc/>
    public override void Refresh()
    {
        AppSettings settings = Env.Model.Settings.Settings;
        if (!configRequested && Env.Model.Vfox.Availability == VfoxAvailability.Ready)
        {
            configRequested = true;
            _ = Env.Model.VfoxConfig.Entries.LoadAsync();
        }

        PageParts.SetTitle(title, T("navSettings"));
        RefreshAppearance(settings);
        RefreshVfox();
        RefreshConfig();
        RefreshAbout();
    }

    private void BuildAppearance()
    {
        themeItem.Trailing = themeGroup;
        themeGroup.Selecting += (_, value) =>
            Env.Model.Settings.Update(settings => settings with { Theme = Enum.Parse<ThemePreference>(value) });
        systemColorItem.Trailing = systemColorSwitch;
        systemColorSwitch.Toggled += (_, on) => Env.Model.Settings.Update(settings => settings with { UseSystemColor = on });
        systemColorItem.Click += (_, _) =>
            Env.Model.Settings.Update(settings => settings with { UseSystemColor = !settings.UseSystemColor });
        M3WrapPanel colors = new() { Spacing = Gaps.S, RunSpacing = Gaps.S, Margin = new Thickness(0, Gaps.S, 0, 0) };
        foreach (uint seed in SeedColors)
        {
            ColorSwatch swatch = new() { Seed = seed };
            uint chosen = seed;
            swatch.Click += (_, _) =>
                Env.Model.Settings.Update(settings => settings with { SeedColor = chosen, UseSystemColor = false });
            swatches.Add(swatch);
            colors.Children.Add(swatch);
        }

        colorItem.SubtitleContent = colors;
        languageItem.Trailing = languageGroup;
        languageGroup.Selecting += (_, value) =>
            Env.Model.Settings.Update(settings => settings with { Language = Enum.Parse<LanguagePreference>(value) });
    }

    private void RefreshAppearance(AppSettings settings)
    {
        appearanceTitle.Text = T("settingsAppearance");
        themeItem.Title = T("settingsTheme");
        themeGroup.SetSegments(
        [
            new Segment(nameof(ThemePreference.System), T("themeSystem"), "systemMode"),
            new Segment(nameof(ThemePreference.Light), T("themeLight"), "lightMode"),
            new Segment(nameof(ThemePreference.Dark), T("themeDark"), "darkMode"),
        ]);
        themeGroup.Selected = settings.Theme.ToString();
        systemColorItem.Title = T("settingsSystemColor");
        systemColorSwitch.Label = systemColorItem.Title;
        systemColorSwitch.IsOn = settings.UseSystemColor;
        colorItem.Title = T("settingsColor");
        for (int i = 0; i < swatches.Count; i++)
        {
            swatches[i].Label = T("settingsColorOption", ("index", i + 1));
            swatches[i].Selected = !settings.UseSystemColor && settings.SeedColor == SeedColors[i];
            swatches[i].Refresh();
        }

        languageItem.Title = T("settingsLanguage");

        // The names of the languages are always in their own language.
        languageGroup.SetSegments(
        [
            new Segment(nameof(LanguagePreference.System), T("languageSystem")),
            new Segment(nameof(LanguagePreference.English), "English"),
            new Segment(nameof(LanguagePreference.SimplifiedChinese), "简体中文"),
        ]);
        languageGroup.Selected = settings.Language.ToString();
    }

    private void BuildVfox()
    {
        executableActions.Children.Add(browse);
        executableItem.Trailing = executableActions;
        detect.Click += (_, _) => Env.Model.Settings.Update(settings => settings with { VfoxExecutable = string.Empty });
        browse.Click += (_, _) => Env.Choose(
            new ChooseOptions(T("vfoxLocate"), "/usr/local/bin"),
            path => Env.Model.Settings.Update(settings => settings with { VfoxExecutable = path }));

        environmentHelp.TextWrapping = TextWrapping.Wrap;
        environment.Margin = new Thickness(0, Gaps.M, 0, 0);
        saveEnvironment.Margin = new Thickness(0, Gaps.S, 0, 0);
        saveEnvironment.Click += (_, _) => SaveEnvironment();
        StackPanel editor = new() { Orientation = Orientation.Vertical, Margin = new Thickness(0, Gaps.S, 0, 0) };
        editor.Children.Add(environmentHelp);
        editor.Children.Add(environment);
        editor.Children.Add(saveEnvironment);
        environmentItem.SubtitleContent = editor;
    }

    private void RefreshVfox()
    {
        VfoxController vfox = Env.Model.Vfox;
        vfoxTitle.Text = T("settingsVfox");
        string executable = vfox.Executable ?? "vfox";
        string source = vfox.Executable is null
            ? T("settingsExecutableMissing")
            : T(vfox.IsCustomExecutable ? "settingsExecutableCustom" : "settingsExecutableDetected");
        List<string> parts = [executable, source];
        if (vfox.Version is { } version)
        {
            parts.Add(T("settingsVfoxVersion", ("version", version)));
        }

        executableItem.Title = T("settingsExecutable");
        executableItem.Subtitle = string.Join(" · ", parts);
        executableActions.Children.Clear();
        if (vfox.Availability == VfoxAvailability.Checking)
        {
            executableActions.Children.Add(checking);
        }
        else if (vfox.IsCustomExecutable)
        {
            detect.Tooltip = T("settingsExecutableDetect");
            executableActions.Children.Add(detect);
        }

        browse.Label = T("actionBrowse");
        executableActions.Children.Add(browse);

        environmentItem.Title = T("settingsEnvironment");
        environmentHelp.Text = T("settingsEnvironmentHelp");
        saveEnvironment.Label = T("actionSave");
        if (!environmentReady)
        {
            environmentReady = true;
            environment.Text = EnvironmentLines.Format(Env.Model.Settings.Settings.ExtraEnvironment);
        }
    }

    // Stores the environment variables that vfox runs with.
    private void SaveEnvironment()
    {
        (IReadOnlyDictionary<string, string> variables, IReadOnlyList<int> invalid) = EnvironmentLines.Parse(environment.Text);
        if (invalid.Count > 0)
        {
            string lines = string.Join(", ", invalid.Select(line => line.ToString(System.Globalization.CultureInfo.InvariantCulture)));
            environment.Error = T("settingsEnvironmentInvalid", ("count", invalid.Count), ("lines", lines));
            return;
        }

        environment.Error = string.Empty;
        Env.Model.Settings.Update(settings => settings with { ExtraEnvironment = variables });
        Env.Notify(T("settingsSaved"));
    }

    private void RefreshConfig()
    {
        VfoxConfigController config = Env.Model.VfoxConfig;
        configTitle.Text = T("settingsVfoxConfig");
        configHelp.Text = T("settingsVfoxConfigHelp");
        if (Env.Model.Vfox.Availability != VfoxAvailability.Ready)
        {
            configBody.Children.Clear();
            return;
        }

        if (!config.Entries.HasValue)
        {
            string key = (config.Entries.Error?.Message ?? "loading") + "|" + Env.L10n.Language;
            if (configMessageKey != key || configMessage is null)
            {
                configMessageKey = key;
                configMessage = config.Entries.Error is { } error
                    ? Common.ErrorView(T("settingsVfoxConfigError"), error, () => _ = config.Entries.LoadAsync())
                    : Common.LoadingView();
            }

            ShowConfigBody(configMessage);
            return;
        }

        foreach (ConfigSwitchRow row in new[] { proxyEnable, legacyEnable, gitignore })
        {
            row.Bind();
        }

        foreach (ConfigTextRow row in new[] { proxyUrl, registry, storage, cache, httpTimeout, downloadTimeout })
        {
            row.Bind();
        }

        registry.Field.Hint = T("configRegistryHint");
        storage.Field.Hint = T("configStorageHint");
        legacyStrategy.Bind();

        // The keys of vfox that the page has no row for.
        List<ListItem> extra = [];
        foreach (ConfigEntry entry in config.Entries.Value!.Where(entry => !KnownKeys.Contains(entry.Key)))
        {
            if (!others.TryGetValue(entry.Key, out ConfigTextRow? row))
            {
                row = new ConfigTextRow(entry.Key, null, null, null, string.Empty, false);
                others[entry.Key] = row;
            }

            row.Bind();
            extra.Add(row);
        }

        otherTitle.Text = T("configOther");
        otherGroup.SetItems(extra);
        ShowConfigBody(null);
    }

    // Either a message, or the groups of the keys.
    private void ShowConfigBody(UIElement? message)
    {
        configBody.Children.Clear();
        if (message is not null)
        {
            configBody.Children.Add(message);
            return;
        }

        configBody.Children.Add(proxyGroup);
        configBody.Children.Add(legacyGroup);
        configBody.Children.Add(timeoutGroup);
        if (otherGroup.Items.Count > 0)
        {
            configBody.Children.Add(PageParts.GroupHeader(otherTitle.Text).Header);
            configBody.Children.Add(otherGroup);
        }
    }

    private void RefreshAbout()
    {
        aboutTitle.Text = T("settingsAbout");
        aboutItem.Title = T("appTitle");
        aboutItem.Subtitle = T("aboutVersion", ("version", ApplicationInfo.Version)) + " · " + T("aboutLicense");
        sourceItem.Title = T("aboutSource");
        sourceItem.Subtitle = Common.Repository;
        docsItem.Title = T("aboutDocs");
        docsItem.Subtitle = Common.VfoxDocs;
        licensesItem.Title = T("aboutLicenses");
    }
}
