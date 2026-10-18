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
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls;

/// <summary>The parts that pages are made of.</summary>
internal static class PageParts
{
    /// <summary>The header of a page: a title, a line under it, and actions at the end.</summary>
    /// <param name="title">The title.</param>
    /// <param name="subtitle">The line under it, or an empty text.</param>
    /// <param name="actions">The buttons at the end.</param>
    /// <returns>The header, and the text blocks that hold its title and subtitle.</returns>
    public static (Grid Header, TextBlock Title, TextBlock Subtitle) Header(string title, string subtitle, params UIElement[] actions)
    {
        AppTheme theme = Ui.Theme;
        TextBlock titleBlock = Ui.Text(title, TextRole.HeadlineMedium, emphasized: true);
        Automation.Name(titleBlock, title);
        Microsoft.UI.Xaml.Automation.AutomationProperties.SetHeadingLevel(titleBlock, Microsoft.UI.Xaml.Automation.Peers.AutomationHeadingLevel.Level1);
        TextBlock subtitleBlock = Ui.Text(subtitle, TextRole.BodyMedium, theme.OnSurfaceVariant);
        subtitleBlock.MaxLines = 2;
        subtitleBlock.TextTrimming = TextTrimming.CharacterEllipsis;
        subtitleBlock.Margin = new Thickness(0, 4, 0, 0);
        subtitleBlock.Visibility = subtitle.Length == 0 ? Visibility.Collapsed : Visibility.Visible;
        StackPanel text = new() { Orientation = Orientation.Vertical, VerticalAlignment = VerticalAlignment.Center };
        text.Children.Add(titleBlock);
        text.Children.Add(subtitleBlock);
        Grid header = new() { Padding = new Thickness(24, 24, 16, 16) };
        header.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        header.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        header.Children.Add(text);
        if (actions.Length > 0)
        {
            StackPanel row = new() { Orientation = Orientation.Horizontal, Spacing = 8, Margin = new Thickness(16, 0, 0, 0), VerticalAlignment = VerticalAlignment.Center };
            foreach (UIElement action in actions)
            {
                row.Children.Add(action);
            }

            Grid.SetColumn(row, 1);
            header.Children.Add(row);
        }

        return (header, titleBlock, subtitleBlock);
    }

    /// <summary>Sets the text of the title of a page, which is also its name for assistive technologies and the tests.</summary>
    /// <param name="title">The text block of the title.</param>
    /// <param name="text">The title.</param>
    public static void SetTitle(TextBlock title, string text)
    {
        title.Text = text;
        Automation.Name(title, text);
    }

    /// <summary>The title of a group of a page.</summary>
    /// <param name="title">The title.</param>
    /// <param name="trailing">An element at the end, or <see langword="null"/>.</param>
    /// <returns>The header, and the text block that holds the title.</returns>
    public static (Grid Header, TextBlock Title) GroupHeader(string title, UIElement? trailing = null)
    {
        TextBlock block = Ui.Text(title, TextRole.TitleSmall, Ui.Theme.Primary);
        Microsoft.UI.Xaml.Automation.AutomationProperties.SetHeadingLevel(block, Microsoft.UI.Xaml.Automation.Peers.AutomationHeadingLevel.Level2);
        Grid row = new() { Padding = new Thickness(16, 24, 4, 8) };
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        block.VerticalAlignment = VerticalAlignment.Center;
        row.Children.Add(block);
        if (trailing is not null)
        {
            Grid.SetColumn(trailing, 1);
            row.Children.Add(trailing);
        }

        return (row, block);
    }

    /// <summary>A rounded surface that a page sits on: 28 dp corners, the content cut to them.</summary>
    /// <param name="child">The page.</param>
    /// <returns>The pane.</returns>
    public static Border Pane(UIElement child) => new()
    {
        Background = Ui.Theme.SurfaceContainerLow,
        CornerRadius = new CornerRadius(Corners.ExtraLarge),
        Child = child,
    };

    /// <summary>Limits the width of a page and puts it at the top, in the middle.</summary>
    /// <param name="maxWidth">The width in dp.</param>
    /// <param name="child">The content.</param>
    /// <returns>The element that holds the content.</returns>
    public static Grid ContentWidth(double maxWidth, UIElement child)
    {
        // Stretched up to the limit, and centred when the space is wider.
        Grid host = new() { MaxWidth = maxWidth, HorizontalAlignment = HorizontalAlignment.Stretch, VerticalAlignment = VerticalAlignment.Top };
        host.Children.Add(child);
        return host;
    }

    /// <summary>Puts elements one under the other, as wide as the space.</summary>
    /// <param name="spacing">The space between them.</param>
    /// <param name="children">The elements.</param>
    /// <returns>The column.</returns>
    public static StackPanel Column(double spacing, params UIElement[] children)
    {
        StackPanel column = new() { Orientation = Orientation.Vertical, Spacing = spacing };
        foreach (UIElement child in children)
        {
            column.Children.Add(child);
        }

        return column;
    }

    /// <summary>Puts elements next to each other.</summary>
    /// <param name="spacing">The space between them.</param>
    /// <param name="children">The elements.</param>
    /// <returns>The row.</returns>
    public static StackPanel Row(double spacing, params UIElement[] children)
    {
        StackPanel row = new() { Orientation = Orientation.Horizontal, Spacing = spacing };
        foreach (UIElement child in children)
        {
            row.Children.Add(child);
        }

        return row;
    }
}
