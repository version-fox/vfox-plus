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
using VfoxPlus.App.Pages;
using VfoxPlus.App.Services;
using VfoxPlus.Design.Tokens;
using VfoxPlus.State;

namespace VfoxPlus.App.Shell;

/// <summary>
/// The window: the navigation (a rail at the start, or a bar at the bottom on a narrow window) and the page, which the
/// navigation switches with a transition.
/// </summary>
internal sealed class AppShell : Grid
{
    // The width in dp below which the navigation moves to the bottom.
    private const double CompactWidth = 600;

    private readonly Env env = Env.Current;
    private readonly Dictionary<Destination, PageBase> pages = [];
    private readonly UnavailablePage unavailable;
    private readonly NavRail rail;
    private readonly NavBar bar = new();
    private readonly ViewSwitcher switcher = new();
    private readonly Grid content = new();
    private readonly Action refreshNavigation;
    private PageBase? shown;
    private bool compact;
    private bool arranged;

    public AppShell()
    {
        refreshNavigation = RefreshNavigation;
        unavailable = new UnavailablePage();
        rail = new NavRail("add");
        rail.Selecting += (_, index) => env.Nav.Go((Destination)index);
        rail.ExpandedChanging += (_, expanded) => env.Model.Settings.Update(settings => settings with { RailExpanded = expanded });
        rail.FabClick += (_, _) => env.Nav.Go(Destination.Plugins);
        bar.Selecting += (_, index) => env.Nav.Go((Destination)index);

        ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
        RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        Children.Add(rail);
        content.Children.Add(switcher);
        Grid.SetColumn(content, 1);
        Children.Add(content);
        Grid.SetColumnSpan(bar, 2);
        Grid.SetRow(bar, 1);
        Children.Add(bar);
        Background = env.Theme.Surface;
        SizeChanged += (_, e) => Arrange(e.NewSize.Width);

        env.Nav.Changed += () => env.Dispatch.PostOnce(refreshNavigation);
        env.L10n.Changed += () => env.Dispatch.PostOnce(refreshNavigation);
        Notifier.ListenAll([env.Model.Vfox, env.Model.Settings, env.Model.Tasks], () => env.Dispatch.PostOnce(refreshNavigation));
        RefreshNavigation();
    }

    /// <summary>Gets the page that shows, which the tests look into.</summary>
    public UIElement? Shown => shown;

    private static PageBase Make(Destination destination) => destination switch
    {
        Destination.Sdks => new SdksPage(),
        Destination.Plugins => new PluginsPage(),
        Destination.Tasks => new TasksPage(),
        Destination.Settings => new SettingsPage(),
        _ => new HomePage(),
    };

    private void Arrange(double width)
    {
        bool nowCompact = width < CompactWidth;
        if (nowCompact == compact && arranged)
        {
            return;
        }

        arranged = true;
        compact = nowCompact;
        rail.Visibility = nowCompact ? Visibility.Collapsed : Visibility.Visible;
        bar.Visibility = nowCompact ? Visibility.Visible : Visibility.Collapsed;

        // Beside the rail the page keeps 16 dp from the edges of the window; a narrow window gives it all the space.
        content.Margin = nowCompact ? new Thickness(0) : new Thickness(0, Gaps.L, Gaps.L, Gaps.L);
        if (env.Overlays is { } overlays)
        {
            overlays.BottomInset = nowCompact ? NavBar.BarHeight : 0;
        }
    }

    private void RefreshNavigation()
    {
        Destination current = env.Nav.Current;
        VfoxAvailability availability = env.Model.Vfox.Availability;
        bool showUnavailable = availability != VfoxAvailability.Ready && current != Destination.Settings;
        PageBase page = showUnavailable ? unavailable : GetPage(current);
        IReadOnlyList<NavDestination> destinations =
        [
            new NavDestination("home", env.T("navHome")),
            new NavDestination("sdks", env.T("navSdks")),
            new NavDestination("plugins", env.T("navPlugins")),
            new NavDestination("tasks", env.T("navTasks"), env.Model.Tasks.ActiveCount),
            new NavDestination("settings", env.T("navSettings")),
        ];
        rail.SetDestinations(destinations);
        rail.Selected = (int)current;
        rail.ExpandTooltip = env.T("navExpand");
        rail.CollapseTooltip = env.T("navCollapse");
        rail.Expanded = env.Model.Settings.Settings.RailExpanded;
        rail.FabLabel = env.T("navAddPlugin");
        rail.FabButton.Visibility = availability == VfoxAvailability.Ready ? Visibility.Visible : Visibility.Collapsed;
        bar.SetDestinations(destinations);
        bar.Selected = (int)current;

        if (!ReferenceEquals(page, shown))
        {
            if (shown is not null)
            {
                shown.Active = false;
            }

            shown = page;
            page.Active = true;
            switcher.Show(page);
        }
    }

    private PageBase GetPage(Destination destination)
    {
        if (!pages.TryGetValue(destination, out PageBase? page))
        {
            page = Make(destination);
            pages[destination] = page;
        }

        return page;
    }
}
