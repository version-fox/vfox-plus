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

using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;

namespace VfoxPlus.App.Controls;

/// <summary>Where an item sits in a segmented list: the outer corners of the list are rounded more than the corners between items.</summary>
internal enum ListPosition
{
    /// <summary>The only item.</summary>
    Single,

    /// <summary>The first item.</summary>
    First,

    /// <summary>An item between others.</summary>
    Middle,

    /// <summary>The last item.</summary>
    Last,
}

/// <summary>
/// An item of a segmented list: 20 dp outer and 4 dp inner corners, the inner ones growing to 12 dp while the item is
/// hovered, pressed or selected. It has a leading element, a title and a subtitle, and a trailing element.
/// </summary>
internal class ListItem : Pressable
{
    private const double WrapWidth = 560;

    private readonly Border container = new();
    private readonly Border layer = new() { Opacity = 0, IsHitTestVisible = false };
    private readonly Border ring = new() { BorderThickness = new Thickness(2), Margin = new Thickness(-3), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
    private readonly Grid body = new();
    private readonly StackPanel textColumn = new() { Orientation = Orientation.Vertical, VerticalAlignment = VerticalAlignment.Center };
    private readonly Animated active;
    private readonly Animated layerOpacity;
    private ListPosition position = ListPosition.Single;
    private bool selected;
    private bool interactive;
    private bool wrapTrailing;
    private bool? wrapped;
    private UIElement? leading;
    private UIElement? trailing;
    private UIElement? titleContent;
    private UIElement? subtitleContent;
    private TextBlock? title;
    private TextBlock? subtitle;
    private TextStyle titleStyle = Typography.Of(TextRole.BodyLarge);
    private string name = string.Empty;

    public ListItem()
    {
        HorizontalAlignment = HorizontalAlignment.Stretch;
        body.Padding = new Thickness(16, 12, 16, 12);
        body.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        body.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        body.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        body.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        body.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        Grid.SetColumn(textColumn, 1);
        body.Children.Add(textColumn);
        Grid root = new();
        root.Children.Add(container);
        root.Children.Add(layer);
        root.Children.Add(ring);
        root.Children.Add(body);
        Content = root;
        IsTabStop = false;
        active = new Animated(MotionScheme.Expressive.FastSpatial, _ => ApplyShape());
        layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
        ring.BorderBrush = Ui.Theme.Secondary;
        SizeChanged += (_, e) => UpdateWrap(e.NewSize.Width);
        UpdateColors();
        ApplyShape();
    }

    /// <inheritdoc/>
    public override string Role => "item";

    /// <summary>Gets or sets where the item sits in its list.</summary>
    public ListPosition Position
    {
        get => position;
        set
        {
            position = value;
            ApplyShape();
        }
    }

    /// <summary>Gets or sets a value indicating whether the item is highlighted.</summary>
    public bool Selected
    {
        get => selected;
        set
        {
            selected = value;
            active.To(Wanted());
            UpdateColors();
            Automation.Select(this, value);
        }
    }

    /// <summary>Gets or sets a value indicating whether the item reacts to the pointer and the keyboard.</summary>
    public bool Interactive
    {
        get => interactive;
        set
        {
            interactive = value;
            IsTabStop = value;
            active.To(Wanted());
        }
    }

    /// <summary>Gets or sets the name of the item for the tests and assistive technologies; its title by default.</summary>
    public string ItemName
    {
        get => name.Length > 0 ? name : title?.Text ?? string.Empty;
        set
        {
            name = value;
            Automation.Name(this, ItemName);
        }
    }

    /// <summary>Gets or sets the padding of the content.</summary>
    public Thickness ContentPadding
    {
        get => body.Padding;
        set => body.Padding = value;
    }

    /// <summary>Gets or sets a value indicating whether the trailing element moves below the text when the item is narrow.</summary>
    public bool WrapTrailing
    {
        get => wrapTrailing;
        set
        {
            wrapTrailing = value;
            UpdateWrap(ActualWidth);
        }
    }

    /// <summary>Gets or sets the element before the text.</summary>
    public UIElement? Leading
    {
        get => leading;
        set
        {
            if (leading is not null)
            {
                body.Children.Remove(leading);
            }

            leading = value;
            if (value is FrameworkElement element)
            {
                element.VerticalAlignment = VerticalAlignment.Center;
                element.Margin = new Thickness(0, 0, 16, 0);
                Grid.SetColumn(element, 0);
                Grid.SetRowSpan(element, 1);
                body.Children.Add(element);
            }
        }
    }

    /// <summary>Gets or sets the element after the text.</summary>
    public UIElement? Trailing
    {
        get => trailing;
        set
        {
            if (trailing is not null)
            {
                body.Children.Remove(trailing);
            }

            trailing = value;
            wrapped = null;
            if (value is not null)
            {
                body.Children.Add(value);
                UpdateWrap(ActualWidth);
            }
        }
    }

    /// <summary>Gets or sets an element that takes the place of the first line, such as a version with its note.</summary>
    public UIElement? TitleContent
    {
        get => titleContent;
        set
        {
            if (titleContent is not null)
            {
                textColumn.Children.Remove(titleContent);
            }

            titleContent = value;
            if (value is not null)
            {
                textColumn.Children.Insert(0, value);
            }

            if (title is not null)
            {
                title.Visibility = value is null && title.Text.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
            }
        }
    }

    /// <summary>Gets or sets an element under the second line, such as a progress bar or the swatches of the colours.</summary>
    public UIElement? SubtitleContent
    {
        get => subtitleContent;
        set
        {
            if (subtitleContent is not null)
            {
                textColumn.Children.Remove(subtitleContent);
            }

            subtitleContent = value;
            if (value is not null)
            {
                textColumn.Children.Add(value);
            }
        }
    }

    /// <summary>Gets or sets the first line.</summary>
    public string Title
    {
        get => title?.Text ?? string.Empty;
        set
        {
            title ??= Add(textColumn, 0, titleStyle);
            title.Text = value;
            title.Visibility = value.Length == 0 || titleContent is not null ? Visibility.Collapsed : Visibility.Visible;
            Automation.Name(this, ItemName);
        }
    }

    /// <summary>Gets or sets the style of the title; body large by default.</summary>
    public TextStyle TitleStyle
    {
        get => titleStyle;
        set
        {
            titleStyle = value;
            title?.Style(value);
        }
    }

    /// <summary>Gets or sets the second line.</summary>
    public string Subtitle
    {
        get => subtitle?.Text ?? string.Empty;
        set
        {
            if (subtitle is null)
            {
                subtitle = Add(textColumn, (title is null ? 0 : 1) + (titleContent is null ? 0 : 1), Typography.Of(TextRole.BodyMedium));
            }

            subtitle.Text = value;
            subtitle.Visibility = value.Length == 0 ? Visibility.Collapsed : Visibility.Visible;
        }
    }

    /// <summary>Gets or sets how many lines the subtitle may take, or 0 for as many as it needs.</summary>
    public int SubtitleLines
    {
        get => subtitle?.MaxLines ?? 0;
        set
        {
            if (subtitle is not null)
            {
                subtitle.MaxLines = value;
                subtitle.TextTrimming = value > 0 ? TextTrimming.CharacterEllipsis : TextTrimming.None;
            }
        }
    }

    /// <summary>Gets the text of the title and the subtitle, which are not the text of the elements that replace them.</summary>
    public StackPanel Text => textColumn;

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        active.To(Wanted());
        layerOpacity.To(interactive && IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
        ring.Visibility = interactive && HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
    }

    private static TextBlock Add(StackPanel column, int index, TextStyle style)
    {
        TextBlock block = new TextBlock { TextWrapping = TextWrapping.Wrap }.Style(style);
        block.Foreground = Ui.Theme.OnSurface;
        column.Children.Insert(index, block);
        return block;
    }

    private double Wanted() => (interactive && (IsHovered || IsPressed)) || selected ? 1 : 0;

    private void ApplyShape()
    {
        double outer = Corners.LargeIncreased;
        double inner = Math.Min(Math.Max(Corners.ExtraSmall + ((Corners.Medium - Corners.ExtraSmall) * active.Value), 0), outer);
        double top = position is ListPosition.First or ListPosition.Single ? outer : inner;
        double bottom = position is ListPosition.Last or ListPosition.Single ? outer : inner;
        CornerRadius radii = new(top, top, bottom, bottom);
        container.CornerRadius = radii;
        layer.CornerRadius = radii;
        ring.CornerRadius = new CornerRadius(top + 3, top + 3, bottom + 3, bottom + 3);
    }

    private void UpdateColors()
    {
        AppTheme theme = Ui.Theme;
        (Brush background, Brush foreground, Brush secondary) = selected
            ? (theme.SecondaryContainer, theme.OnSecondaryContainer, theme.OnSecondaryContainer)
            : (theme.SurfaceContainer, theme.OnSurface, theme.OnSurfaceVariant);
        container.Background = background;
        layer.Background = foreground;
        if (title is not null)
        {
            title.Foreground = foreground;
        }

        if (subtitle is not null)
        {
            subtitle.Foreground = secondary;
        }

        if (leading is M3Icon icon)
        {
            icon.Foreground = secondary;
        }
    }

    // A narrow item puts the trailing element below the text.
    private void UpdateWrap(double width)
    {
        if (trailing is not FrameworkElement element)
        {
            return;
        }

        bool wrap = wrapTrailing && width > 0 && width < WrapWidth;
        if (wrapped == wrap)
        {
            return;
        }

        wrapped = wrap;
        if (wrap)
        {
            Grid.SetColumn(element, 1);
            Grid.SetRow(element, 1);
            element.HorizontalAlignment = HorizontalAlignment.Left;
            element.Margin = new Thickness(0, 8, 0, 0);
        }
        else
        {
            Grid.SetColumn(element, 2);
            Grid.SetRow(element, 0);
            element.HorizontalAlignment = HorizontalAlignment.Right;
            element.Margin = new Thickness(16, 0, 0, 0);
        }

        element.VerticalAlignment = VerticalAlignment.Center;
    }
}

/// <summary>
/// A segmented list: the first and the last item have rounded outer corners, and the items are 2 dp apart.
/// </summary>
internal sealed class ListGroup : StackPanel
{
    public ListGroup()
    {
        Orientation = Orientation.Vertical;
        Spacing = Gaps.Hairline;
    }

    /// <summary>Gets the items.</summary>
    public IReadOnlyList<ListItem> Items => [.. Children.OfType<ListItem>()];

    /// <summary>Makes the items the ones of the list; an item that is already there keeps its place and its animations.</summary>
    /// <param name="items">The items, in order.</param>
    public void SetItems(IReadOnlyList<ListItem> items)
    {
        for (int i = 0; i < items.Count; i++)
        {
            if (i < Children.Count && ReferenceEquals(Children[i], items[i]))
            {
                continue;
            }

            if (Children.Contains(items[i]))
            {
                Children.Remove(items[i]);
            }

            Children.Insert(i, items[i]);
        }

        while (Children.Count > items.Count)
        {
            Children.RemoveAt(Children.Count - 1);
        }

        for (int i = 0; i < items.Count; i++)
        {
            items[i].Position = items.Count <= 1 ? ListPosition.Single : i == 0 ? ListPosition.First : i == items.Count - 1 ? ListPosition.Last : ListPosition.Middle;
        }
    }
}
