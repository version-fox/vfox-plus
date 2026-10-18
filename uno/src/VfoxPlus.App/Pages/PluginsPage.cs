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
using VfoxPlus.App.Theming;
using VfoxPlus.Core;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Design.Tokens;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>Lists the plugins of the registry, to add them.</summary>
internal sealed class PluginsPage : PageBase
{
    // The filters of the registry.
    private const string FilterAll = "all";
    private const string FilterOfficial = "official";
    private const string FilterCommunity = "community";

    private readonly ScrollPage page = new();
    private readonly Border pane;
    private readonly TextBlock title;
    private readonly TextBlock subtitle;
    private readonly Grid updateSlot = new();
    private readonly M3IconButton updateAll = new() { Symbol = "update" };
    private readonly LoadingIndicator updating = new(40);
    private readonly M3IconButton refresh;
    private readonly M3Button custom = new() { Kind = ButtonKind.Tonal, Symbol = "add" };
    private readonly M3WrapPanel toolbar = new() { Spacing = Gaps.M, RunSpacing = Gaps.M, CenterCross = true, Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, Gaps.L) };
    private readonly M3SearchBar search = new() { HorizontalAlignment = HorizontalAlignment.Left };
    private readonly ButtonGroup filter = new();
    private readonly CardGrid grid = new() { Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, Gaps.Xl) };
    private readonly Dictionary<string, PluginCard> cards = [];
    private List<string> shown = [];
    private string query = string.Empty;
    private string filterValue = FilterAll;
    private bool requested;

    public PluginsPage()
    {
        refresh = Common.RefreshButton(() => _ = Env.Model.Registry.LoadAsync());
        updateAll.Click += (_, _) => Env.Run(VfoxCommands.UpdateAllPlugins);
        custom.Click += (_, _) => Controls.Overlays.Dialogs.AskCustomPlugin();
        updateSlot.Children.Add(updateAll);
        (Grid header, title, subtitle) = PageParts.Header(string.Empty, string.Empty, updateSlot, refresh, custom);
        search.TextEdited += (_, text) =>
        {
            query = text;
            Refresh();
        };
        filter.Selecting += (_, value) =>
        {
            filterValue = value;
            Refresh();
        };
        filter.Selected = FilterAll;
        toolbar.Children.Add(search);
        toolbar.Children.Add(filter);
        toolbar.SizeChanged += (_, e) => search.Width = Math.Max(0, Math.Min(420, e.NewSize.Width));
        page.Top.Children.Add(header);
        page.Top.Children.Add(toolbar);
        pane = PageParts.Pane(page);
        Children.Add(pane);
        Watch(Env.Model.Registry, Env.Model.Installed, Env.Model.Tasks);
        WatchTaskProgress();
    }

    /// <summary>Gets the cards of the plugins that show, which the tests look into.</summary>
    public IReadOnlyList<PluginCard> Cards => [.. grid.Children.OfType<PluginCard>()];

    /// <inheritdoc/>
    public override void Refresh()
    {
        var registry = Env.Model.Registry;
        if (!requested)
        {
            requested = true;
            _ = registry.EnsureLoadedAsync();
        }

        HashSet<string> installed = [.. (Env.Model.Installed.Value ?? []).Select(sdk => sdk.Name)];
        PageParts.SetTitle(title, T("pluginsTitle"));
        subtitle.SetText(T("pluginsSubtitle"));
        bool isUpdatingAll = Env.Model.Tasks.ActiveTask(VfoxOperation.UpdateAllPlugins) is not null;
        updateSlot.Children.Clear();
        updateSlot.Children.Add(isUpdatingAll ? updating : updateAll);
        updateAll.Tooltip = T("pluginsUpdateAll");
        updateAll.IsEnabled = installed.Count > 0;
        refresh.Tooltip = T("actionRefresh");
        refresh.IsEnabled = !registry.IsLoading;
        custom.Label = T("pluginsAddCustom");
        search.Hint = T("pluginsSearchHint");
        search.ClearTooltip = T("actionClose");
        search.Text = query;
        filter.SetSegments(
        [
            new Segment(FilterAll, T("pluginsFilterAll")),
            new Segment(FilterOfficial, T("pluginsFilterOfficial"), "verified"),
            new Segment(FilterCommunity, T("pluginsFilterCommunity")),
        ]);
        filter.Selected = filterValue;

        if (!registry.HasValue)
        {
            page.ShowMessage(registry.Error is { } error
                ? Common.ErrorView(T("pluginsLoadError"), error, () => _ = registry.LoadAsync())
                : Common.LoadingView());
            return;
        }

        string text = query.Trim();
        List<AvailablePlugin> visible =
        [
            .. registry.Value!.Where(plugin =>
                (filterValue == FilterAll || (filterValue == FilterOfficial && plugin.Official) || (filterValue == FilterCommunity && !plugin.Official))
                && (text.Length == 0
                    || plugin.Name.Contains(text, StringComparison.OrdinalIgnoreCase)
                    || plugin.Homepage.Contains(text, StringComparison.OrdinalIgnoreCase))),
        ];
        if (visible.Count == 0)
        {
            page.ShowMessage(new StatusMessage("notFound", T("pluginsNoMatch"), string.Empty));
            return;
        }

        foreach (AvailablePlugin plugin in visible)
        {
            if (!cards.TryGetValue(plugin.Name, out PluginCard? card))
            {
                card = new PluginCard(plugin.Name);
                cards[plugin.Name] = card;
            }

            card.Bind(plugin, installed.Contains(plugin.Name), Env.Model.Tasks.ActiveTask(VfoxOperation.AddPlugin, plugin.Name));
        }

        // The grid is laid out again only when the plugins that show are not the same, in the same order.
        List<string> names = [.. visible.Select(plugin => plugin.Name)];
        if (!names.SequenceEqual(shown))
        {
            shown = names;
            grid.Children.Clear();
            foreach (string name in names)
            {
                grid.Children.Add(cards[name]);
            }
        }

        page.ShowContent(grid);
    }
}

/// <summary>A plugin of the registry, with the button that adds it.</summary>
internal sealed class PluginCard : M3Card
{
    private readonly Env env = Env.Current;
    private readonly string name;
    private readonly TextBlock source = Ui.Text(string.Empty, TextRole.BodySmall, Ui.Theme.OnSurfaceVariant);
    private readonly Border official = new() { Background = Ui.Theme.Transparent, Margin = new Thickness(Gaps.Xs, 0, 0, 0), VerticalAlignment = VerticalAlignment.Center };
    private readonly M3IconButton homepage = new() { Symbol = "openInNew" };
    private readonly Grid actionSlot = new();
    private readonly M3Button add = new() { Kind = ButtonKind.Tonal, Symbol = "add" };
    private readonly M3Button added = new() { Kind = ButtonKind.Text, Symbol = "check" };
    private readonly LoadingIndicator adding = new(40);
    private string homepageUrl = string.Empty;

    public PluginCard(string name)
    {
        this.name = name;
        AppTheme theme = Ui.Theme;
        Radius = Corners.LargeIncreased;
        ContentBrush = theme.OnSurface;
        TextBlock title = Ui.Text(name, Typography.Emphasized(TextRole.TitleMedium), theme.OnSurface);
        title.TextTrimming = TextTrimming.CharacterEllipsis;
        title.VerticalAlignment = VerticalAlignment.Center;
        source.TextTrimming = TextTrimming.CharacterEllipsis;
        M3Icon verified = new("verified", 18, theme.Primary);
        verified.FillTo(1, animate: false);
        official.Child = verified;

        StackPanel titleRow = new() { Orientation = Orientation.Horizontal };
        titleRow.Children.Add(title);
        titleRow.Children.Add(official);
        StackPanel text = new() { Orientation = Orientation.Vertical, VerticalAlignment = VerticalAlignment.Center };
        text.Children.Add(titleRow);
        text.Children.Add(source);
        Grid heading = new() { ColumnSpacing = Gaps.M };
        heading.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        heading.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        Avatar avatar = new(name, 44);
        heading.Children.Add(avatar);
        Grid.SetColumn(text, 1);
        heading.Children.Add(text);

        Grid footer = new();
        footer.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        footer.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        footer.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        footer.Children.Add(homepage);
        Grid.SetColumn(actionSlot, 2);
        footer.Children.Add(actionSlot);

        Grid body = new() { Padding = new Thickness(Gaps.L) };
        body.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        body.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
        body.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        body.Children.Add(heading);
        Grid.SetRow(footer, 2);
        body.Children.Add(footer);
        Child = body;
        CardName = name;

        homepage.Click += (_, _) => env.OpenUrl(homepageUrl);
        add.Click += (_, _) => env.Run(() => VfoxCommands.AddPlugin(name));
        added.Click += (_, _) => env.Nav.ShowSdk(name);
    }

    /// <summary>Gets the name of the plugin.</summary>
    public string PluginName => name;

    /// <summary>Shows a plugin.</summary>
    /// <param name="plugin">The plugin.</param>
    /// <param name="installed">Whether it is installed.</param>
    /// <param name="task">The task that adds it, or <see langword="null"/>.</param>
    public void Bind(AvailablePlugin plugin, bool installed, VfoxTask? task)
    {
        AppTheme theme = Ui.Theme;
        homepageUrl = plugin.Homepage;
        Color = installed ? theme.SurfaceContainerHighest : theme.SurfaceContainer;
        source.Text = SourceOf(plugin.Homepage);
        official.Visibility = plugin.Official ? Visibility.Visible : Visibility.Collapsed;
        Tip.SetText(official, plugin.Official ? env.T("pluginsOfficial") : string.Empty);
        homepage.Tooltip = env.T("actionOpenHomepage");
        homepage.IsEnabled = plugin.Homepage.Length > 0;
        actionSlot.Children.Clear();
        if (task is not null)
        {
            actionSlot.Children.Add(adding);
        }
        else if (installed)
        {
            added.Label = env.T("pluginsAdded");
            actionSlot.Children.Add(added);
        }
        else
        {
            add.Label = env.T("pluginsAdd");
            actionSlot.Children.Add(add);
        }
    }

    // The repository of a plugin as owner/name, from its homepage.
    private static string SourceOf(string homepage)
    {
        if (!Uri.TryCreate(homepage, UriKind.Absolute, out Uri? address))
        {
            return homepage;
        }

        string path = string.Join('/', address.AbsolutePath.Split('/', StringSplitOptions.RemoveEmptyEntries));
        return path.Length == 0 ? homepage : path;
    }
}
