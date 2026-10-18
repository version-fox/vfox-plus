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
using VfoxPlus.Core;
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Pages;

/// <summary>Lists the installed plugins and shows the versions of the one that is selected.</summary>
internal sealed class SdksPage : PageBase
{
    // The width in dp from which the page shows the list and the details of the selected plugin next to each other.
    private const double TwoPaneWidth = 840;

    private readonly SdkListView list = new();
    private readonly Border listPane;
    private readonly Border detailPane;
    private readonly ViewSwitcher detailSwitcher = new();
    private readonly ViewSwitcher narrowSwitcher = new();
    private readonly Grid wide = new();
    private readonly Border messagePane = new();
    private SdkDetailView? detail;
    private StatusMessage? prompt;
    private string promptLanguage = string.Empty;
    private bool isWide;
    private bool arranged;

    public SdksPage()
    {
        listPane = PageParts.Pane(list);
        detailPane = PageParts.Pane(detailSwitcher);
        messagePane = PageParts.Pane(new Grid());
        wide.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(360) });
        wide.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(Gaps.Xl) });
        wide.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        Grid.SetColumn(detailPane, 2);
        Children.Add(wide);
        Children.Add(narrowSwitcher);
        Children.Add(messagePane);
        SizeChanged += (_, e) =>
        {
            bool nowWide = e.NewSize.Width >= TwoPaneWidth;
            if (nowWide != isWide || !arranged)
            {
                isWide = nowWide;
                Arrange(first: !arranged);
                arranged = true;
                Refresh();
            }
        };
        Watch(Env.Model.Installed, Env.Model.Tasks);
        WatchTaskProgress();
        Env.Nav.Changed += () => Env.Dispatch.PostOnce(Refresh);
    }

    /// <summary>Gets the list of the plugins.</summary>
    public SdkListView List => list;

    /// <summary>Gets the details of the selected plugin, or <see langword="null"/>.</summary>
    public SdkDetailView? Detail => detail;

    /// <inheritdoc/>
    public override void Refresh()
    {
        if (!arranged)
        {
            return;
        }

        var installed = Env.Model.Installed;
        if (!installed.HasValue)
        {
            ShowMessage(installed.Error is { } error
                ? Common.ErrorView(T("sdksLoadError"), error, () => _ = installed.LoadAsync())
                : Common.LoadingView());
            return;
        }

        IReadOnlyList<InstalledSdk> sdks = installed.Value!;
        string selected = sdks.Any(sdk => sdk.Name == Env.Nav.SelectedSdk) ? Env.Nav.SelectedSdk : string.Empty;
        UseDetail(selected);
        list.Show(sdks, selected, withChevrons: !isWide);
        messagePane.Visibility = Visibility.Collapsed;
        if (isWide)
        {
            wide.Visibility = Visibility.Visible;
            narrowSwitcher.Visibility = Visibility.Collapsed;
            UIElement view = detail is null ? Prompt() : detail;
            detailSwitcher.Show(view, Transition.FadeThrough);
            return;
        }

        wide.Visibility = Visibility.Collapsed;
        narrowSwitcher.Visibility = Visibility.Visible;
        if (detail is null)
        {
            narrowSwitcher.Show(listPane, Transition.SharedAxisX, goingBack: true);
        }
        else
        {
            detailSwitcher.Show(detail, Transition.FadeThrough);
            narrowSwitcher.Show(detailPane, Transition.SharedAxisX);
        }
    }

    // The details of the selected plugin: made when the plugin is selected, and replaced when another is.
    private void UseDetail(string selected)
    {
        if (detail is not null && detail.SdkName == selected)
        {
            detail.ShowBack = !isWide;
            return;
        }

        SdkDetailView? old = detail;
        detail = selected.Length == 0 ? null : new SdkDetailView(selected) { ShowBack = !isWide };
        old?.Release();
    }

    private StatusMessage Prompt()
    {
        if (prompt is null || promptLanguage != Env.L10n.Language)
        {
            prompt = new StatusMessage("sdks", T("sdksSelectPrompt"), string.Empty, shape: MaterialShapes.Gem);
            promptLanguage = Env.L10n.Language;
        }

        return prompt;
    }

    private void ShowMessage(UIElement view)
    {
        messagePane.Child = view;
        messagePane.Visibility = Visibility.Visible;
        wide.Visibility = Visibility.Collapsed;
        narrowSwitcher.Visibility = Visibility.Collapsed;
    }

    // The panes sit in a grid when they are next to each other, and in a switcher when one shows at a time.
    private void Arrange(bool first)
    {
        _ = first;
        wide.Children.Clear();
        narrowSwitcher.Reset();
        detailSwitcher.Reset();
        if (isWide)
        {
            wide.Children.Add(listPane);
            wide.Children.Add(detailPane);
        }
    }
}
