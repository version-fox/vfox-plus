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
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>The overview: what is installed, what is in use, and shortcuts.</summary>
internal sealed class HomePage : PageBase
{
    // The line that makes a shell use the versions that vfox+ chooses.
    private const string ActivationCommand = """eval "$(vfox activate bash)" """;

    private readonly Border pane;
    private readonly ScrollViewer scroll = new() { VerticalScrollBarVisibility = ScrollBarVisibility.Auto, HorizontalScrollBarVisibility = ScrollBarVisibility.Disabled };
    private readonly Grid host = new();
    private readonly StackPanel content = new() { Orientation = Orientation.Vertical };
    private readonly TextBlock title;
    private readonly TextBlock subtitle;
    private readonly M3IconButton refreshButton;
    private readonly StatTile plugins;
    private readonly StatTile versions;
    private readonly StatTile inUse;
    private readonly Grid stats = new();
    private readonly TextBlock activeTitle;
    private readonly ListGroup activeList = new();
    private readonly Dictionary<string, ListItem> activeItems = [];
    private readonly ListItem emptyItem = new();
    private readonly Grid shortcuts = new();
    private readonly TextBlock shellTitle;
    private readonly TextBlock shellBody;
    private readonly M3IconButton copyButton;
    private readonly ShortcutCard explore;
    private readonly ShortcutCard docs;
    private UIElement? message;

    public HomePage()
    {
        AppTheme theme = Ui.Theme;
        refreshButton = Common.RefreshButton(() => _ = Env.Model.Installed.LoadAsync());
        (Grid header, title, subtitle) = PageParts.Header(string.Empty, string.Empty, refreshButton);

        plugins = new StatTile("plugins", "plugins", MaterialShapes.Cookie9, theme.PrimaryContainer, theme.OnPrimaryContainer);
        versions = new StatTile("versions", "versions", MaterialShapes.Clover4, theme.SecondaryContainer, theme.OnSecondaryContainer);
        inUse = new StatTile("current", "active", MaterialShapes.Sunny, theme.TertiaryContainer, theme.OnTertiaryContainer);
        foreach (StatTile tile in new[] { plugins, versions, inUse })
        {
            tile.Click += (_, _) => Env.Nav.ShowSdk(string.Empty);
            stats.Children.Add(tile);
        }

        (Grid activeHeader, activeTitle) = PageParts.GroupHeader(string.Empty);
        emptyItem.Leading = new M3Icon("info", 24);
        emptyItem.Interactive = true;
        emptyItem.Click += (_, _) => Env.Nav.ShowSdk(string.Empty);

        explore = new ShortcutCard("explore", "chevronRight");
        explore.Click += (_, _) => Env.Nav.Go(Destination.Plugins);
        docs = new ShortcutCard("docs", "openInNew");
        docs.Click += (_, _) => Env.OpenUrl(Common.VfoxDocs);
        (M3Card shellCard, shellTitle, shellBody, copyButton) = BuildShellCard();
        copyButton.Click += (_, _) =>
        {
            Env.Platform.Copy(ActivationCommand.Trim());
            Env.Notify(T("actionCopied"));
        };
        shortcuts.Children.Add(shellCard);
        shortcuts.Children.Add(explore);
        shortcuts.Children.Add(docs);

        StackPanel body = PageParts.Column(0, stats, activeHeader, activeList, new Border { Height = 24 }, shortcuts);
        Grid wide = PageParts.ContentWidth(1080, body);
        wide.Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, Gaps.Xl);
        content.Children.Add(header);
        content.Children.Add(wide);
        scroll.Content = content;
        host.Children.Add(scroll);
        pane = PageParts.Pane(host);
        Children.Add(pane);
        wide.SizeChanged += (_, e) => Arrange(e.NewSize.Width);
        Watch(Env.Model.Installed, Env.Model.Vfox, Env.Model.Settings);
    }

    /// <inheritdoc/>
    public override void Refresh()
    {
        SetMessage(null);
        AppModel model = Env.Model;
        title.Text = T("appTitle");
        Automation.Name(title, title.Text);
        if (!model.Installed.HasValue)
        {
            scroll.Visibility = Visibility.Collapsed;
            SetMessage(model.Installed.Error is { } error
                ? Common.ErrorView(T("sdksLoadError"), error, () => _ = model.Installed.LoadAsync())
                : Common.LoadingView());
            return;
        }

        scroll.Visibility = Visibility.Visible;
        IReadOnlyList<InstalledSdk> sdks = model.Installed.Value!;
        List<InstalledSdk> active = [.. sdks.Where(sdk => sdk.HasCurrent)];
        int count = sdks.Sum(sdk => sdk.Versions.Count);
        string version = model.Vfox.Version ?? "?";
        subtitle.SetText(T("homeSubtitle", ("version", version)) + (model.Vfox.Executable is { } executable ? " · " + executable : string.Empty));
        refreshButton.IsEnabled = !model.Installed.IsLoading;
        plugins.Set(sdks.Count, T("homeStatPlugins", ("count", sdks.Count)));
        versions.Set(count, T("homeStatVersions", ("count", count)));
        inUse.Set(active.Count, T("homeStatActive"));
        activeTitle.Text = T("homeActiveTitle");
        emptyItem.Title = T("homeActiveEmpty");
        if (active.Count == 0)
        {
            activeList.SetItems([emptyItem]);
        }
        else
        {
            List<ListItem> items = [];
            foreach (InstalledSdk sdk in active)
            {
                if (!activeItems.TryGetValue(sdk.Name, out ListItem? item))
                {
                    item = new ListItem { Leading = new Avatar(sdk.Name, 40), Interactive = true, Trailing = new M3Icon("chevronRight", 24) };
                    string name = sdk.Name;
                    item.Click += (_, _) => Env.Nav.ShowSdk(name);
                    activeItems[sdk.Name] = item;
                }

                item.Title = sdk.Name;
                item.Subtitle = sdk.Current ?? string.Empty;
                items.Add(item);
            }

            activeList.SetItems(items);
        }

        shellTitle.Text = T("homeShellTitle");
        shellBody.Text = T("homeShellBody");
        copyButton.Tooltip = T("actionCopy");
        explore.Set(T("homeExploreTitle"), T("homeExploreBody"));
        docs.Set(T("homeDocsTitle"), T("homeDocsBody"));
    }

    private static (M3Card M3Card, TextBlock Title, TextBlock Body, M3IconButton Copy) BuildShellCard()
    {
        AppTheme theme = Ui.Theme;
        TextBlock title = Ui.Text(string.Empty, TextRole.TitleMedium, emphasized: true);
        title.VerticalAlignment = VerticalAlignment.Center;
        StackPanel heading = PageParts.Row(12, new M3Icon("terminal", 24, theme.Primary), title);
        TextBlock body = Ui.Text(string.Empty, TextRole.BodyMedium, theme.OnSurfaceVariant);
        body.TextWrapping = TextWrapping.Wrap;
        body.Margin = new Thickness(0, 8, 0, 0);
        TextBox command = Common.SelectableText(ActivationCommand.Trim(), Typography.Of(TextRole.BodyMedium), theme.OnSurface);
        command.VerticalAlignment = VerticalAlignment.Center;
        M3IconButton copy = new() { Symbol = "copy" };
        Grid commandRow = new() { Padding = new Thickness(16, 4, 4, 4) };
        commandRow.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        commandRow.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        commandRow.Children.Add(command);
        Grid.SetColumn(copy, 1);
        commandRow.Children.Add(copy);
        M3Card commandCard = new() { Color = theme.SurfaceContainerLowest, Radius = Corners.Medium, Child = commandRow, Margin = new Thickness(0, 12, 0, 0) };
        StackPanel column = PageParts.Column(0, heading, body, commandCard);
        column.Padding = new Thickness(24);
        M3Card card = new() { Color = theme.SurfaceContainerHighest, Radius = Corners.Large, Child = column };
        return (card, title, body, copy);
    }

    private void SetMessage(UIElement? view)
    {
        if (message is not null)
        {
            host.Children.Remove(message);
        }

        message = view;
        if (view is not null)
        {
            host.Children.Add(view);
        }
    }

    // The tiles share the width, and the cards under the list take two columns or one.
    private void Arrange(double width)
    {
        bool compact = width < 560;
        stats.ColumnDefinitions.Clear();
        StatTile[] tiles = [plugins, versions, inUse];
        stats.ColumnSpacing = compact ? Gaps.S : Gaps.M;
        for (int i = 0; i < tiles.Length; i++)
        {
            stats.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
            Grid.SetColumn(tiles[i], i);
            tiles[i].Compact = compact;
        }

        shortcuts.ColumnDefinitions.Clear();
        shortcuts.RowDefinitions.Clear();
        shortcuts.ColumnSpacing = Gaps.M;
        shortcuts.RowSpacing = Gaps.M;
        M3Card shell = (M3Card)shortcuts.Children[0];
        if (width < 760)
        {
            shortcuts.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
            for (int i = 0; i < 3; i++)
            {
                shortcuts.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
                Grid.SetRow(shortcuts.Children[i], i);
                Grid.SetColumn(shortcuts.Children[i], 0);
                Grid.SetRowSpan(shortcuts.Children[i], 1);
            }
        }
        else
        {
            shortcuts.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(3, GridUnitType.Star) });
            shortcuts.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(2, GridUnitType.Star) });
            shortcuts.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
            shortcuts.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
            Grid.SetColumn(shell, 0);
            Grid.SetRow(shell, 0);
            Grid.SetRowSpan(shell, 2);
            Grid.SetColumn(explore, 1);
            Grid.SetRow(explore, 0);
            Grid.SetRowSpan(explore, 1);
            Grid.SetColumn(docs, 1);
            Grid.SetRow(docs, 1);
            Grid.SetRowSpan(docs, 1);
        }
    }

    /// <summary>A tile with a number, a label and a large shape that is partly hidden behind its edge; it opens the SDKs.</summary>
    private sealed class StatTile : M3Card
    {
        private readonly TextBlock number;
        private readonly TextBlock label;
        private readonly Grid grid = new() { Height = 136 };
        private bool compact;

        public StatTile(string icon, string name, RoundedPolygon shape, Brush background, Brush foreground)
        {
            Color = background;
            ContentBrush = foreground;
            Radius = Corners.ExtraLarge;
            Interactive = true;
            CardName = name;
            number = new TextBlock { VerticalAlignment = VerticalAlignment.Bottom, Foreground = foreground };
            label = new TextBlock { Foreground = foreground, TextWrapping = TextWrapping.Wrap, MaxLines = 2, Margin = new Thickness(0, Gaps.Xs, 0, 0) };
            label.Style(Typography.Emphasized(TextRole.LabelLarge));

            // A large, partly hidden shape gives the tile its character.
            ShapeView decoration = new(shape, 140, foreground) { Opacity = 0.08, HorizontalAlignment = HorizontalAlignment.Right, VerticalAlignment = VerticalAlignment.Bottom, Margin = new Thickness(0, 0, -28, -36) };
            M3Icon glyph = new(icon, 24, foreground) { HorizontalAlignment = HorizontalAlignment.Left, VerticalAlignment = VerticalAlignment.Top };
            glyph.FillTo(1, animate: false);
            StackPanel text = new() { Orientation = Orientation.Vertical, VerticalAlignment = VerticalAlignment.Bottom };
            text.Children.Add(number);
            text.Children.Add(label);
            Grid layout = new();
            layout.Children.Add(glyph);
            layout.Children.Add(text);
            grid.Children.Add(decoration);
            grid.Children.Add(layout);
            Child = grid;
            Compact = false;
        }

        public bool Compact
        {
            get => compact;
            set
            {
                compact = value;
                TextStyle style = Typography.Emphasized(value ? TextRole.HeadlineLarge : TextRole.DisplaySmall);
                number.Style(style with { Line = style.Size });
                ((Grid)grid.Children[1]).Padding = new Thickness(value ? Gaps.L : Gaps.Xl);
            }
        }

        public void Set(int value, string text)
        {
            number.Text = value.ToString(System.Globalization.CultureInfo.InvariantCulture);
            label.Text = text;
            CardName = text;
        }
    }

    /// <summary>A card that opens something: a cookie with an icon, a title and a line, and an icon at the end.</summary>
    private sealed class ShortcutCard : M3Card
    {
        private readonly TextBlock title;
        private readonly TextBlock body;

        public ShortcutCard(string icon, string trailing)
        {
            AppTheme theme = Ui.Theme;
            Color = theme.SurfaceContainer;
            ContentBrush = theme.OnSurface;
            Radius = Corners.Large;
            Interactive = true;
            Grid badge = new() { Width = 48, Height = 48, VerticalAlignment = VerticalAlignment.Center };
            badge.Children.Add(new ShapeView(MaterialShapes.Cookie6, 48, theme.SecondaryContainer));
            badge.Children.Add(new M3Icon(icon, 24, theme.OnSecondaryContainer) { HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center });
            title = Ui.Text(string.Empty, TextRole.TitleSmall, emphasized: true);
            body = Ui.Text(string.Empty, TextRole.BodySmall, theme.OnSurfaceVariant);
            body.TextWrapping = TextWrapping.Wrap;
            body.Margin = new Thickness(0, Gaps.Xs, 0, 0);
            StackPanel text = new() { Orientation = Orientation.Vertical, VerticalAlignment = VerticalAlignment.Center };
            text.Children.Add(title);
            text.Children.Add(body);
            Grid row = new() { Padding = new Thickness(Gaps.L), ColumnSpacing = Gaps.L };
            row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
            row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
            row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
            row.Children.Add(badge);
            Grid.SetColumn(text, 1);
            row.Children.Add(text);
            M3Icon end = new(trailing, 24) { VerticalAlignment = VerticalAlignment.Center };
            Grid.SetColumn(end, 2);
            row.Children.Add(end);
            Child = row;
        }

        public void Set(string titleText, string bodyText)
        {
            title.Text = titleText;
            body.Text = bodyText;
            CardName = titleText;
        }
    }
}
