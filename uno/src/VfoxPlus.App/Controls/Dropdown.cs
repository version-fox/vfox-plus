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

using Microsoft.UI.Xaml.Automation;
using VfoxPlus.App.Controls.Overlays;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls;

/// <summary>An entry of a dropdown.</summary>
/// <param name="Value">Identifies the entry.</param>
/// <param name="Label">The text of the entry.</param>
internal sealed record Choice(string Value, string Label);

/// <summary>A filled field that shows the chosen entry and opens a menu of all of them.</summary>
internal sealed class Dropdown : Pressable
{
    private readonly Border container = new() { CornerRadius = new CornerRadius(Corners.Medium) };
    private readonly Border outline = new() { CornerRadius = new CornerRadius(Corners.Medium), BorderThickness = new Thickness(2), Opacity = 0, IsHitTestVisible = false };
    private readonly TextBlock text = Ui.Text(string.Empty, TextRole.BodyLarge);
    private readonly M3Icon arrow = new("expandMore", 24) { RenderTransformOrigin = new Windows.Foundation.Point(0.5, 0.5) };
    private readonly Animated turn;
    private IReadOnlyList<Choice> choices = [];
    private string value = string.Empty;
    private MenuPopup? menu;

    public Dropdown()
    {
        Height = 56;
        Width = 240;
        HorizontalAlignment = HorizontalAlignment.Left;
        outline.BorderBrush = Ui.Theme.Primary;
        text.VerticalAlignment = VerticalAlignment.Center;
        text.TextTrimming = TextTrimming.CharacterEllipsis;
        text.Margin = new Thickness(16, 0, 48, 0);
        arrow.HorizontalAlignment = HorizontalAlignment.Right;
        arrow.VerticalAlignment = VerticalAlignment.Center;
        arrow.Margin = new Thickness(0, 0, 16, 0);
        Grid root = new();
        root.Children.Add(container);
        root.Children.Add(outline);
        root.Children.Add(text);
        root.Children.Add(arrow);
        Content = root;
        turn = new Animated(MotionScheme.Expressive.FastSpatial, angle => arrow.RenderTransform = new RotateTransform { Angle = angle * 180 });
        UpdateColors();
    }

    /// <summary>Raised with the value of the entry the user chose.</summary>
    public event EventHandler<string>? Selecting;

    /// <summary>Gets or sets the entries.</summary>
    public IReadOnlyList<Choice> Choices
    {
        get => choices;
        set
        {
            choices = value;
            ShowValue();
        }
    }

    /// <summary>Gets or sets the value of the chosen entry; none is chosen when no entry has it.</summary>
    public string Value
    {
        get => value;
        set
        {
            this.value = value;
            ShowValue();
        }
    }

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        UpdateColors();
        outline.Opacity = menu is not null || HasKeyboardFocus ? 1 : 0;
    }

    /// <inheritdoc/>
    protected override void OnClicked()
    {
        if (menu is not null || OverlayHost.Current is not { } host)
        {
            return;
        }

        List<MenuItem> items = [.. choices.Select(choice => new MenuItem(choice.Label, () =>
        {
            if (choice.Value != value)
            {
                Selecting?.Invoke(this, choice.Value);
            }
        }, Selected: choice.Value == value))];
        menu = host.ShowMenu(this, items, new MenuOptions(MatchWidth: true));
        turn.To(1);
        outline.Opacity = 1;
        menu.Dismissed += (_, _) =>
        {
            menu = null;
            turn.To(0);
            outline.Opacity = HasKeyboardFocus ? 1 : 0;
        };
    }

    private void ShowValue()
    {
        string label = choices.FirstOrDefault(choice => choice.Value == value)?.Label ?? string.Empty;
        text.Text = label;
        AutomationProperties.SetName(this, label);
    }

    private void UpdateColors()
    {
        AppTheme theme = Ui.Theme;
        container.Background = IsEnabled ? theme.SurfaceContainerHighest : theme.OnSurface04;
        text.Foreground = IsEnabled ? theme.OnSurface : theme.OnSurface38;
        arrow.Foreground = IsEnabled ? theme.OnSurfaceVariant : theme.OnSurface38;
    }
}
