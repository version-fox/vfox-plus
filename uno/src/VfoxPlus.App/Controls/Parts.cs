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

using VfoxPlus.App.Theming;
using VfoxPlus.Design.Shapes;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls;

/// <summary>A small label in a pill: the current version, a note such as LTS.</summary>
internal sealed class Tag : Border
{
    private readonly TextBlock label;

    public Tag(string text, string? symbol = null, bool emphasized = false)
    {
        AppTheme theme = Ui.Theme;
        (Brush background, Brush foreground) = emphasized
            ? (theme.TertiaryContainer, theme.OnTertiaryContainer)
            : (theme.SurfaceContainerHighest, theme.OnSurfaceVariant);
        StackPanel row = new() { Orientation = Orientation.Horizontal, Spacing = 4, VerticalAlignment = VerticalAlignment.Center };
        if (symbol is not null)
        {
            M3Icon icon = new(symbol, 16, foreground);
            icon.FillTo(1, animate: false);
            row.Children.Add(icon);
        }

        label = Ui.Text(text, TextRole.LabelMedium, foreground);
        label.VerticalAlignment = VerticalAlignment.Center;
        row.Children.Add(label);
        Background = background;
        CornerRadius = new CornerRadius(100);
        Padding = new Thickness(10, 3, 10, 3);
        Child = row;
        HorizontalAlignment = HorizontalAlignment.Left;
        VerticalAlignment = VerticalAlignment.Center;
        Automation.Name(this, text);
    }

    /// <summary>Gets or sets the text of the tag.</summary>
    public string Text
    {
        get => label.Text;
        set
        {
            label.Text = value;
            Automation.Name(this, value);
        }
    }
}

/// <summary>
/// The centred message of an empty, failed or unavailable view: an icon on a large shape, a title, an explanation and
/// actions.
/// </summary>
internal sealed class StatusMessage : Grid
{
    public StatusMessage(string symbol, string title, string body, bool isError = false, RoundedPolygon? shape = null, IReadOnlyList<UIElement>? actions = null)
    {
        AppTheme theme = Ui.Theme;
        (Brush background, Brush foreground) = isError ? (theme.ErrorContainer, theme.OnErrorContainer) : (theme.SecondaryContainer, theme.OnSecondaryContainer);
        StackPanel column = new()
        {
            Orientation = Orientation.Vertical,
            MaxWidth = 504,
            Padding = new Thickness(32),
            HorizontalAlignment = HorizontalAlignment.Center,
            VerticalAlignment = VerticalAlignment.Center,
        };
        Grid badge = new() { Width = 112, Height = 112, HorizontalAlignment = HorizontalAlignment.Center };
        badge.Children.Add(new ShapeView(shape ?? MaterialShapes.Cookie9, 112, background));
        M3Icon icon = new(symbol, 48, foreground) { HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center };
        icon.FillTo(1, animate: false);
        badge.Children.Add(icon);
        column.Children.Add(badge);
        TextBlock heading = Ui.Text(title, TextRole.TitleLarge, emphasized: true);
        heading.TextAlignment = TextAlignment.Center;
        heading.TextWrapping = TextWrapping.Wrap;
        heading.Margin = new Thickness(0, 24, 0, 0);
        column.Children.Add(heading);
        if (body.Length > 0)
        {
            TextBlock text = Ui.Text(body, TextRole.BodyMedium, theme.OnSurfaceVariant);
            text.TextAlignment = TextAlignment.Center;
            text.TextWrapping = TextWrapping.Wrap;
            text.Margin = new Thickness(0, 8, 0, 0);
            column.Children.Add(text);
        }

        if (actions is { Count: > 0 })
        {
            StackPanel row = new() { Orientation = Orientation.Horizontal, Spacing = 8, HorizontalAlignment = HorizontalAlignment.Center, Margin = new Thickness(0, 24, 0, 0) };
            foreach (UIElement action in actions)
            {
                row.Children.Add(action);
            }

            column.Children.Add(row);
        }

        Children.Add(column);
    }
}
