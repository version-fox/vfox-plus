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
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls;

/// <summary>The style of a button.</summary>
internal enum ButtonKind
{
    /// <summary>The high emphasis button.</summary>
    Filled,

    /// <summary>The medium emphasis button.</summary>
    Tonal,

    /// <summary>A button with a border.</summary>
    Outlined,

    /// <summary>The low emphasis button.</summary>
    Text,
}

/// <summary>
/// A Material 3 Expressive button: round at rest, and squarer while it is pressed (see
/// <c>docs/design-system.md#shape-morphing-on-interaction</c>).
/// </summary>
internal sealed class M3Button : Pressable
{
    private const double Height40 = 40;
    private const double IconSize = 20;

    private readonly Border container = new() { BorderThickness = new Thickness(1) };
    private readonly Border layer = new() { Opacity = 0, IsHitTestVisible = false };
    private readonly Border ring = new() { BorderThickness = new Thickness(2), Margin = new Thickness(-3), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
    private readonly StackPanel row = new()
    {
        Orientation = Orientation.Horizontal,
        Spacing = 8,
        Padding = new Thickness(Gaps.L, 0, Gaps.L, 0),
        HorizontalAlignment = HorizontalAlignment.Center,
        VerticalAlignment = VerticalAlignment.Center,
    };

    private readonly TextBlock text = Ui.Text(string.Empty, TextRole.LabelLarge).Style(Typography.Of(TextRole.LabelLarge));
    private readonly Animated shape;
    private readonly Animated layerOpacity;
    private M3Icon? icon;
    private ButtonKind kind;
    private bool danger;
    private string label = string.Empty;
    private string? symbol;
    private Brush? contentBrush;

    public M3Button()
    {
        Height = Height40;
        MinWidth = 48;
        HorizontalAlignment = HorizontalAlignment.Left;
        text.VerticalAlignment = VerticalAlignment.Center;
        row.Children.Add(text);
        Grid root = new();
        root.Children.Add(container);
        root.Children.Add(layer);
        root.Children.Add(ring);
        root.Children.Add(row);
        Content = root;
        shape = new Animated(MotionScheme.Expressive.FastSpatial, ApplyShape);
        layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
        ring.BorderBrush = Ui.Theme.Secondary;
        shape.Apply();
        UpdateColors();
    }

    /// <summary>Gets or sets the style of the button.</summary>
    public ButtonKind Kind
    {
        get => kind;
        set
        {
            kind = value;
            UpdateColors();
        }
    }

    /// <summary>Gets or sets a value indicating whether a filled button has the colour of errors, for destructive actions.</summary>
    public bool Danger
    {
        get => danger;
        set
        {
            danger = value;
            UpdateColors();
        }
    }

    /// <summary>Gets or sets the colour of the label and the icon, in place of the one of the style; a disabled button ignores it.</summary>
    public Brush? ContentBrush
    {
        get => contentBrush;
        set
        {
            contentBrush = value;
            UpdateColors();
        }
    }

    /// <summary>Gets or sets the text of the button.</summary>
    public string Label
    {
        get => label;
        set
        {
            label = value;
            text.Text = value;
            text.Visibility = value.Length == 0 ? Visibility.Collapsed : Visibility.Visible;
            AutomationProperties.SetName(this, value);
        }
    }

    /// <summary>Gets or sets the icon before the label, if any.</summary>
    public string? Symbol
    {
        get => symbol;
        set
        {
            symbol = value;
            if (value is null)
            {
                if (icon is not null)
                {
                    row.Children.Remove(icon);
                    icon = null;
                }

                return;
            }

            if (icon is null)
            {
                icon = new M3Icon(value, IconSize);
                row.Children.Insert(0, icon);
            }
            else
            {
                icon.Symbol = value;
            }

            UpdateColors();
        }
    }

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        shape.To(IsPressed ? 1 : 0);
        layerOpacity.To(IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
        ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
        UpdateColors();
    }

    private void ApplyShape(double pressed)
    {
        double radius = Math.Max(0, Lerp(Height40 / 2, Corners.Small, pressed));
        container.CornerRadius = new CornerRadius(radius);
        layer.CornerRadius = new CornerRadius(radius);
        ring.CornerRadius = new CornerRadius(radius + 3);
    }

    private static double Lerp(double a, double b, double t) => a + ((b - a) * t);

    private void UpdateColors()
    {
        AppTheme theme = Ui.Theme;
        Brush containerBrush;
        Brush content;
        Brush border = theme.Transparent;
        switch (kind)
        {
            case ButtonKind.Tonal:
                (containerBrush, content) = (theme.SecondaryContainer, theme.OnSecondaryContainer);
                break;
            case ButtonKind.Outlined:
                (containerBrush, content, border) = (theme.Transparent, theme.Primary, theme.Outline);
                break;
            case ButtonKind.Text:
                (containerBrush, content) = (theme.Transparent, theme.Primary);
                break;
            default:
                (containerBrush, content) = danger ? (theme.Error, theme.OnError) : (theme.Primary, theme.OnPrimary);
                break;
        }

        content = contentBrush ?? content;
        if (!IsEnabled)
        {
            content = theme.OnSurface38;
            containerBrush = kind is ButtonKind.Filled or ButtonKind.Tonal ? theme.OnSurface12 : theme.Transparent;
            border = kind == ButtonKind.Outlined ? theme.OnSurface12 : theme.Transparent;
        }

        container.Background = containerBrush;
        container.BorderBrush = border;
        layer.Background = content;
        text.Foreground = content;
        if (icon is not null)
        {
            icon.Foreground = content;
        }
    }
}
