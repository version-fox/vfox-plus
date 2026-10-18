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
using VfoxPlus.Core;
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Pages;

/// <summary>The list of the installed plugins, with a field that searches it.</summary>
internal sealed class SdkListView : Grid
{
    private readonly Env env = Env.Current;
    private readonly ScrollPage page = new();
    private readonly TextBlock title;
    private readonly M3IconButton refresh;
    private readonly M3SearchBar search = new();
    private readonly ListGroup group = new() { Margin = new Thickness(Gaps.L, 0, Gaps.L, Gaps.L) };
    private readonly Dictionary<string, ListItem> items = [];
    private UIElement? empty;
    private string emptyLanguage = string.Empty;
    private IReadOnlyList<InstalledSdk> sdks = [];
    private string selected = string.Empty;
    private bool chevrons;
    private string query = string.Empty;

    public SdkListView()
    {
        refresh = Common.RefreshButton(() => _ = env.Model.Installed.LoadAsync());
        (Grid header, title, _) = PageParts.Header(string.Empty, string.Empty, refresh);
        search.Margin = new Thickness(Gaps.L, 0, Gaps.L, Gaps.M);
        search.TextEdited += (_, text) =>
        {
            query = text;
            Rebuild();
        };
        page.Top.Children.Add(header);
        page.Top.Children.Add(search);
        Children.Add(page);
    }

    /// <summary>Gets the rows of the plugins, which the tests look into.</summary>
    public IReadOnlyList<ListItem> Items => group.Items;

    /// <summary>Shows the plugins.</summary>
    /// <param name="installed">The installed plugins.</param>
    /// <param name="selectedName">The plugin that is selected, or an empty text.</param>
    /// <param name="withChevrons">Whether the rows have a chevron, which tells that they open the plugin: when the details are not next to the list.</param>
    public void Show(IReadOnlyList<InstalledSdk> installed, string selectedName, bool withChevrons)
    {
        sdks = installed;
        selected = selectedName;
        chevrons = withChevrons;
        PageParts.SetTitle(title, env.T("navSdks"));
        refresh.IsEnabled = !env.Model.Installed.IsLoading;
        refresh.Tooltip = env.T("actionRefresh");
        search.Hint = env.T("sdksSearchHint");
        search.ClearTooltip = env.T("actionClose");
        Rebuild();
    }

    private void Rebuild()
    {
        if (sdks.Count == 0)
        {
            page.ShowMessage(Empty());
            return;
        }

        string text = query.Trim();
        List<InstalledSdk> visible = [.. sdks.Where(sdk => text.Length == 0 || sdk.Name.Contains(text, StringComparison.OrdinalIgnoreCase))];
        if (visible.Count == 0)
        {
            page.ShowMessage(new StatusMessage("notFound", env.T("sdksNoMatch", ("query", text)), string.Empty));
            return;
        }

        List<ListItem> shown = [];
        foreach (InstalledSdk sdk in visible)
        {
            if (!items.TryGetValue(sdk.Name, out ListItem? item))
            {
                string name = sdk.Name;
                item = new ListItem { Title = name, Leading = new Avatar(name, 40), Interactive = true, SubtitleLines = 2 };
                item.Click += (_, _) => env.Nav.ShowSdk(name);
                items[name] = item;
            }

            string current = sdk.HasCurrent ? env.T("sdksCurrent", ("version", sdk.Current)) : env.T("sdksNoCurrent");
            item.Subtitle = current + "\n" + env.T("sdksVersionCount", ("count", sdk.Versions.Count));
            item.SubtitleLines = 2;
            item.Selected = sdk.Name == selected;
            item.Trailing = chevrons ? new M3Icon("chevronRight", 24) : null;
            shown.Add(item);
        }

        group.SetItems(shown);
        page.ShowContent(group);
    }

    // Nothing is installed yet: the pane says so, and leads to the plugins.
    private UIElement Empty()
    {
        if (empty is null || emptyLanguage != env.L10n.Language)
        {
            M3Button add = new() { Kind = ButtonKind.Filled, Symbol = "add", Label = env.T("navAddPlugin") };
            add.Click += (_, _) => env.Nav.Go(Destination.Plugins);
            empty = new StatusMessage("empty", env.T("sdksEmptyTitle"), env.T("sdksEmptyBody"), shape: MaterialShapes.Flower, actions: [add]);
            emptyLanguage = env.L10n.Language;
        }

        return empty;
    }
}
