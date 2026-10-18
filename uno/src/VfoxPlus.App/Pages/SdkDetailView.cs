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
using VfoxPlus.Core.Vfox;
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>
/// The details of an installed plugin: what it is, the versions that are installed, and the versions that can be
/// installed. It lives as long as the plugin is selected.
/// </summary>
internal sealed class SdkDetailView : Grid
{
    private const string TabInstalled = "installed";
    private const string TabAvailable = "available";

    // The width in dp below which the header puts its buttons on a row of their own.
    private const double NarrowHeader = 520;

    private readonly Env env = Env.Current;
    private readonly string name;
    private readonly SdkDetailController controller;
    private readonly List<IDisposable> subscriptions = [];
    private readonly Action refresh;
    private readonly Action changed;

    private readonly ScrollViewer scroll = new() { VerticalScrollBarVisibility = ScrollBarVisibility.Auto, HorizontalScrollBarVisibility = ScrollBarVisibility.Disabled };
    private readonly Grid header = new() { Padding = new Thickness(Gaps.L, Gaps.Xl, Gaps.L, Gaps.Xl) };
    private readonly M3IconButton back = new() { Symbol = "back", VerticalAlignment = VerticalAlignment.Top };
    private readonly Avatar largeAvatar;
    private readonly Avatar smallAvatar;
    private readonly StackPanel text = new() { Orientation = Orientation.Vertical };
    private readonly TextBlock heading;
    private readonly M3WrapPanel tags = new() { Spacing = Gaps.S, RunSpacing = Gaps.Xs };
    private readonly Tag withCurrent = new(string.Empty, "current", emphasized: true);
    private readonly Tag withoutCurrent = new(string.Empty);
    private readonly Tag pluginVersion = new(string.Empty);
    private readonly TextBlock description;
    private readonly StackPanel actions = new() { Orientation = Orientation.Horizontal, VerticalAlignment = VerticalAlignment.Top };
    private readonly M3IconButton update = new() { Symbol = "update" };
    private readonly M3IconButton more = new() { Symbol = "more" };
    private readonly LoadingIndicator updating = new(40) { Margin = new Thickness(Gaps.Xs) };
    private readonly ButtonGroup tabs = new() { Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, Gaps.L) };
    private readonly Grid bodyHost = new();

    private readonly ListGroup installedList = new() { Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, 0) };
    private readonly Dictionary<string, InstalledVersionRow> installedRows = [];

    private readonly StackPanel availablePanel = new() { Orientation = Orientation.Vertical };
    private readonly M3WrapPanel toolbar = new() { Spacing = Gaps.M, RunSpacing = Gaps.S, CenterCross = true, Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, Gaps.M) };
    private readonly M3TextField filter = new() { Dense = true, LeadingSymbol = "search", Width = 280 };
    private readonly ButtonGroup filters = new();
    private readonly TextBlock count;
    private readonly VirtualRows<AvailableVersion> rows;
    private readonly Border noVersions = new();
    private HashSet<string> installedNames = [];

    private string tab = TabInstalled;
    private bool showBack;
    private bool? narrow;
    private string messageKey = string.Empty;
    private UIElement? messageView;
    private bool disposed;

    public SdkDetailView(string name)
    {
        this.name = name;
        controller = env.Model.SdkDetail(name);
        refresh = Refresh;
        changed = () => env.Dispatch.PostOnce(refresh);
        AppTheme theme = Ui.Theme;

        largeAvatar = new Avatar(name, 72);
        smallAvatar = new Avatar(name, 56);
        heading = Ui.Text(name, Typography.Emphasized(TextRole.HeadlineMedium), theme.OnSurface);
        heading.TextWrapping = TextWrapping.Wrap;
        description = Ui.Text(string.Empty, TextRole.BodyMedium, theme.OnSurfaceVariant);
        description.MaxLines = 3;
        description.TextWrapping = TextWrapping.Wrap;
        description.TextTrimming = TextTrimming.CharacterEllipsis;
        description.Margin = new Thickness(0, Gaps.S, 0, 0);
        tags.Margin = new Thickness(0, Gaps.Xs, 0, 0);
        text.Children.Add(heading);
        text.Children.Add(tags);
        text.Children.Add(description);
        actions.Children.Add(update);
        actions.Children.Add(more);
        back.Click += (_, _) => env.Nav.ShowSdk(string.Empty);
        update.Click += (_, _) => env.Run(() => VfoxCommands.UpdatePlugin(name));
        Menus.Attach(more, BuildPluginMenu);
        header.SizeChanged += (_, e) => ArrangeHeader(e.NewSize.Width < NarrowHeader);

        tabs.Selecting += (_, value) => ShowTab(value);
        tabs.Selected = tab;

        count = Ui.Text(string.Empty, TextRole.LabelLarge, theme.OnSurfaceVariant);
        filter.TextEdited += (_, value) => controller.SetQuery(value);
        filters.Selecting += (_, value) => controller.SetFilter(Enum.Parse<VersionFilter>(value));
        toolbar.Children.Add(filter);
        toolbar.Children.Add(filters);
        toolbar.Children.Add(count);
        rows = new VirtualRows<AvailableVersion>(
            () => new AvailableVersionRow(),
            (row, version, position) => ((AvailableVersionRow)row).Bind(name, version, installedNames.Contains(version.Version), position),
            version => version.Version,
            Gaps.Hairline);
        rows.Repeater.Margin = new Thickness(Gaps.Xl, 0, Gaps.Xl, 0);
        availablePanel.Children.Add(toolbar);
        availablePanel.Children.Add(rows.Repeater);
        availablePanel.Children.Add(noVersions);

        StackPanel column = new() { Orientation = Orientation.Vertical };
        column.Children.Add(header);
        column.Children.Add(tabs);
        column.Children.Add(bodyHost);
        column.Children.Add(new Border { Height = Gaps.Xl });
        scroll.Content = column;
        Children.Add(scroll);

        subscriptions.Add(Notifier.ListenAll([controller, controller.Info, controller.Versions, env.Model.Installed, env.Model.Tasks], changed));
        subscriptions.Add(env.Model.Tasks.OnFinished(task =>
        {
            if (task.Command.Sdk == name && task.Command.Operation == VfoxOperation.UpdatePlugin)
            {
                _ = controller.Info.LoadAsync();
            }
        }));
        env.TasksChanged += changed;
        env.L10n.Changed += changed;
        env.ThemeChanged += changed;
        _ = controller.Info.LoadAsync();
        Refresh();
    }

    /// <summary>Gets the plugin whose details these are.</summary>
    public string SdkName => name;

    /// <summary>Gets or sets a value indicating whether the details have a button that leaves them: when the list of plugins is not next to them.</summary>
    public bool ShowBack
    {
        get => showBack;
        set
        {
            if (showBack == value)
            {
                return;
            }

            showBack = value;
            narrow = null;
            ArrangeHeader(header.ActualWidth is > 0 and < NarrowHeader);
        }
    }

    /// <summary>Gets the tab that shows: <c>installed</c> or <c>available</c>.</summary>
    public string Tab => tab;

    /// <summary>Gets the rows of the available versions that exist now, which the tests look into.</summary>
    public IReadOnlyCollection<FrameworkElement> AvailableRows => rows.Rows;

    /// <summary>Stops listening to the state; the details are not shown any more.</summary>
    public void Release()
    {
        if (disposed)
        {
            return;
        }

        disposed = true;
        env.TasksChanged -= changed;
        env.L10n.Changed -= changed;
        env.ThemeChanged -= changed;
        foreach (IDisposable subscription in subscriptions)
        {
            subscription.Dispose();
        }

        controller.Dispose();
    }

    /// <summary>Shows the state again.</summary>
    public void Refresh()
    {
        if (disposed)
        {
            return;
        }

        InstalledSdk sdk = env.Model.InstalledSdk(name) ?? new InstalledSdk(name, []);
        PluginInfo? info = controller.Info.Value;
        RefreshHeader(sdk, info);
        tabs.SetSegments(
        [
            new Segment(TabInstalled, $"{env.T("sdkTabInstalled")} ({sdk.Versions.Count})", "versions"),
            new Segment(TabAvailable, env.T("sdkTabAvailable"), "install"),
        ]);
        tabs.Selected = tab;
        if (tab == TabInstalled)
        {
            ShowInstalled(sdk);
        }
        else
        {
            ShowAvailable(sdk);
        }
    }

    private void ShowTab(string value)
    {
        tab = value;
        if (value == TabAvailable)
        {
            _ = controller.Versions.EnsureLoadedAsync();
        }

        Refresh();
    }

    private void RefreshHeader(InstalledSdk sdk, PluginInfo? info)
    {
        withCurrent.Text = sdk.HasCurrent ? env.T("sdksCurrent", ("version", sdk.Current)) : string.Empty;
        withoutCurrent.Text = env.T("sdksNoCurrent");
        tags.Children.Clear();
        tags.Children.Add(sdk.HasCurrent ? withCurrent : withoutCurrent);
        if (info is { Version.Length: > 0 })
        {
            pluginVersion.Text = env.T("sdkPluginVersion", ("version", info.Version));
            tags.Children.Add(pluginVersion);
        }

        description.SetText(info?.Description ?? string.Empty);
        description.Margin = new Thickness(0, Gaps.S, 0, 0);
        update.Tooltip = env.T("sdkUpdatePlugin");
        more.Tooltip = env.T("sdkMoreActions");
        back.Tooltip = env.T("actionClose");
        bool isUpdating = env.Model.Tasks.ActiveTask(VfoxOperation.UpdatePlugin, name) is not null;
        actions.Children.Clear();
        actions.Children.Add(isUpdating ? updating : update);
        actions.Children.Add(more);
        if (narrow is null)
        {
            ArrangeHeader(header.ActualWidth is > 0 and < NarrowHeader);
        }
    }

    // The header is a row of the back button, the avatar, the text and the buttons; a narrow one puts the buttons on a
    // row above the avatar and the text.
    private void ArrangeHeader(bool isNarrow)
    {
        if (narrow == isNarrow)
        {
            return;
        }

        narrow = isNarrow;
        header.Children.Clear();
        header.ColumnDefinitions.Clear();
        header.RowDefinitions.Clear();
        if (isNarrow)
        {
            header.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
            header.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
            header.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
            header.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
            header.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
            if (showBack)
            {
                header.Children.Add(back);
            }

            Grid.SetColumn(actions, 2);
            Grid.SetRow(actions, 0);
            actions.Margin = new Thickness(0);
            header.Children.Add(actions);
            smallAvatar.Margin = new Thickness(Gaps.S, Gaps.S, 0, 0);
            smallAvatar.VerticalAlignment = VerticalAlignment.Top;
            Grid.SetRow(smallAvatar, 1);
            Grid.SetColumn(smallAvatar, 0);
            header.Children.Add(smallAvatar);
            Grid.SetRow(text, 1);
            Grid.SetColumn(text, 1);
            Grid.SetColumnSpan(text, 2);
            text.Margin = new Thickness(Gaps.L, Gaps.S, 0, 0);
            header.Children.Add(text);
            return;
        }

        header.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        header.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        header.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        header.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        header.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        if (showBack)
        {
            Grid.SetColumn(back, 0);
            Grid.SetRow(back, 0);
            header.Children.Add(back);
        }

        largeAvatar.Margin = new Thickness(showBack ? Gaps.Xs : Gaps.S, 0, 0, 0);
        largeAvatar.VerticalAlignment = VerticalAlignment.Top;
        Grid.SetColumn(largeAvatar, 1);
        Grid.SetRow(largeAvatar, 0);
        header.Children.Add(largeAvatar);
        Grid.SetColumn(text, 2);
        Grid.SetRow(text, 0);
        Grid.SetColumnSpan(text, 1);
        text.Margin = new Thickness(Gaps.L, 0, Gaps.S, 0);
        header.Children.Add(text);
        Grid.SetColumn(actions, 3);
        Grid.SetRow(actions, 0);
        header.Children.Add(actions);
    }

    private List<MenuItem> BuildPluginMenu()
    {
        List<MenuItem> menu = [];
        if (controller.Info.Value is { Homepage.Length: > 0 } info)
        {
            string homepage = info.Homepage;
            menu.Add(new MenuItem(env.T("actionOpenHomepage"), () => env.OpenUrl(homepage), Symbol: "openInNew"));
        }

        menu.Add(new MenuItem(env.T("sdkRemovePlugin"), ConfirmRemovePlugin, Symbol: "removePlugin", Danger: true));
        return menu;
    }

    private void ConfirmRemovePlugin()
    {
        int versions = env.Model.InstalledSdk(name)?.Versions.Count ?? 0;
        Dialogs.Confirm(
            "remove-plugin",
            env.T("sdkRemoveTitle", ("name", name)),
            env.T("sdkRemoveBody", ("count", versions)),
            env.T("sdkRemoveConfirm"),
            "removePlugin",
            destructive: true,
            onConfirm: () =>
            {
                env.Run(() => VfoxCommands.RemovePlugin(name));
                env.Nav.ShowSdk(string.Empty);
            });
    }

    private void ShowInstalled(InstalledSdk sdk)
    {
        if (sdk.Versions.Count == 0)
        {
            Show(Message("installed-empty", () =>
            {
                M3Button browse = new() { Kind = ButtonKind.Filled, Symbol = "install", Label = env.T("sdkBrowseVersions") };
                browse.Click += (_, _) => ShowTab(TabAvailable);
                return new StatusMessage("install", env.T("sdkInstalledEmpty", ("name", name)), string.Empty, shape: MaterialShapes.Pill, actions: [browse]);
            }));
            return;
        }

        List<ListItem> items = [];
        foreach (string version in sdk.Versions)
        {
            if (!installedRows.TryGetValue(version, out InstalledVersionRow? row))
            {
                row = new InstalledVersionRow();
                installedRows[version] = row;
            }

            row.Bind(sdk, version, ListPosition.Single);
            items.Add(row);
        }

        foreach (string gone in installedRows.Keys.Except(sdk.Versions).ToArray())
        {
            installedRows.Remove(gone);
        }

        installedList.SetItems(items);
        Show(installedList);
    }

    private void ShowAvailable(InstalledSdk sdk)
    {
        if (!controller.Versions.HasValue)
        {
            if (controller.Versions.Error is { } error)
            {
                Show(Message("available-error:" + error.Message, () => Common.ErrorView(env.T("availableError", ("name", name)), error, () => _ = controller.Versions.LoadAsync())));
            }
            else
            {
                Show(Message("available-loading", () => Common.LoadingView(env.T("availableLoading"))));
            }

            return;
        }

        installedNames = [.. sdk.Versions];
        IReadOnlyList<AvailableVersion> visible = controller.VisibleVersions(installedNames);
        filter.Hint = env.T("availableSearchHint");
        filter.Text = controller.Query;
        filters.SetSegments(
        [
            new Segment(nameof(VersionFilter.All), env.T("availableFilterAll")),
            new Segment(nameof(VersionFilter.WithNote), env.T("availableFilterTagged")),
            new Segment(nameof(VersionFilter.Installed), env.T("availableFilterInstalled")),
        ]);
        filters.Selected = controller.Filter.ToString();
        count.Text = env.T("availableCount", ("count", visible.Count));
        rows.SetItems(visible);
        rows.Repeater.Visibility = visible.Count == 0 ? Visibility.Collapsed : Visibility.Visible;
        noVersions.Visibility = visible.Count == 0 ? Visibility.Visible : Visibility.Collapsed;
        if (visible.Count == 0)
        {
            noVersions.Child = Message("available-empty", () => new StatusMessage("notFound", env.T("availableEmpty"), string.Empty));
        }

        Show(availablePanel);
    }

    // The message of a state, made again when its words change.
    private UIElement Message(string key, Func<UIElement> make)
    {
        string full = key + "|" + env.L10n.Language;
        if (messageKey != full || messageView is null)
        {
            messageKey = full;
            messageView = make();
        }

        return messageView;
    }

    private void Show(UIElement view)
    {
        if (bodyHost.Children.Count == 1 && ReferenceEquals(bodyHost.Children[0], view))
        {
            return;
        }

        bodyHost.Children.Clear();
        bodyHost.Children.Add(view);
    }
}
