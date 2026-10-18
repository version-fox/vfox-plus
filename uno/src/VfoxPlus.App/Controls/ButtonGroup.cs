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

/// <summary>A button of a <see cref="ButtonGroup"/>.</summary>
/// <param name="Value">Identifies the segment.</param>
/// <param name="Label">The text of the button.</param>
/// <param name="Symbol">The icon before the label, if any.</param>
/// <param name="Tooltip">Describes the segment.</param>
internal sealed record Segment(string Value, string Label, string? Symbol = null, string? Tooltip = null);

/// <summary>
/// A connected button group with a single selection: the outer corners are fully round, the inner ones 8 dp; the
/// selected button becomes fully round, and a button that is pressed pinches its inner corners to 4 dp.
/// </summary>
internal sealed class ButtonGroup : UserControl
{
    private readonly Grid row = new() { ColumnSpacing = Gaps.Hairline, HorizontalAlignment = HorizontalAlignment.Left };
    private readonly List<SegmentButton> buttons = [];
    private bool expand;
    private string selected = string.Empty;

    public ButtonGroup()
    {
        Content = row;
        IsTabStop = false;
        HorizontalAlignment = HorizontalAlignment.Left;
    }

    /// <summary>Raised with the value of a segment that the user chose.</summary>
    public event EventHandler<string>? Selecting;

    /// <summary>Gets or sets the value of the selected segment.</summary>
    public string Selected
    {
        get => selected;
        set
        {
            selected = value;
            foreach (SegmentButton button in buttons)
            {
                button.SetSelected(button.Segment.Value == value);
            }
        }
    }

    /// <summary>Gets or sets a value indicating whether the group is as wide as its space.</summary>
    public bool Expand
    {
        get => expand;
        set
        {
            expand = value;
            HorizontalAlignment = value ? HorizontalAlignment.Stretch : HorizontalAlignment.Left;
            row.HorizontalAlignment = HorizontalAlignment;
            UpdateColumns();
        }
    }

    /// <summary>Gets the buttons, which the tests choose.</summary>
    public IReadOnlyList<Pressable> Buttons => buttons;

    /// <summary>Sets the segments.</summary>
    /// <param name="segments">The segments, which replace the ones there are.</param>
    public void SetSegments(IReadOnlyList<Segment> segments)
    {
        // The buttons keep their springs while the labels change, as they do when the language does.
        if (buttons.Count == segments.Count && buttons.Select(button => button.Segment.Value).SequenceEqual(segments.Select(segment => segment.Value)))
        {
            for (int i = 0; i < segments.Count; i++)
            {
                buttons[i].Update(segments[i]);
            }

            return;
        }

        row.Children.Clear();
        buttons.Clear();
        for (int i = 0; i < segments.Count; i++)
        {
            SegmentButton button = new(segments[i], first: i == 0, last: i == segments.Count - 1);
            button.SetSelected(segments[i].Value == selected, animate: false);
            button.Click += (_, _) => Selecting?.Invoke(this, button.Segment.Value);
            buttons.Add(button);
            row.Children.Add(button);
        }

        UpdateColumns();
    }

    private void UpdateColumns()
    {
        row.ColumnDefinitions.Clear();
        for (int i = 0; i < buttons.Count; i++)
        {
            row.ColumnDefinitions.Add(new ColumnDefinition { Width = expand ? new GridLength(1, GridUnitType.Star) : GridLength.Auto });
            Grid.SetColumn(buttons[i], i);
        }
    }

    private sealed class SegmentButton : Pressable
    {
        private const double Height40 = 40;

        private readonly Border container = new();
        private readonly Border layer = new() { Opacity = 0, IsHitTestVisible = false };
        private readonly Border ring = new() { BorderThickness = new Thickness(2), Margin = new Thickness(-3), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
        private readonly TextBlock text = Ui.Text(string.Empty, TextRole.LabelLarge);
        private readonly StackPanel content = new() { Orientation = Orientation.Horizontal, Spacing = 8, Padding = new Thickness(16, 0, 16, 0), HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center };
        private readonly Animated selection;
        private readonly Animated press;
        private readonly Animated layerOpacity;
        private readonly bool first;
        private readonly bool last;
        private M3Icon? icon;
        private bool isSelected;

        public SegmentButton(Segment segment, bool first, bool last)
        {
            Segment = segment;
            this.first = first;
            this.last = last;
            Height = Height40;
            MinWidth = 48;
            text.VerticalAlignment = VerticalAlignment.Center;
            content.Children.Add(text);
            Grid root = new();
            root.Children.Add(container);
            root.Children.Add(layer);
            root.Children.Add(ring);
            root.Children.Add(content);
            Content = root;
            selection = new Animated(MotionScheme.Expressive.FastSpatial, _ => ApplyShape());
            press = new Animated(MotionScheme.Expressive.FastSpatial, _ => ApplyShape());
            layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
            ring.BorderBrush = Ui.Theme.Secondary;
            Update(segment);
            UpdateColors();
            ApplyShape();
        }

        public Segment Segment { get; private set; }

        public void Update(Segment segment)
        {
            Segment = segment;
            text.Text = segment.Label;
            AutomationProperties.SetName(this, segment.Label);
            Tip.SetText(this, segment.Tooltip ?? string.Empty);
            if (segment.Symbol is { } symbol)
            {
                if (icon is null)
                {
                    icon = new M3Icon(symbol, 20);
                    content.Children.Insert(0, icon);
                }
                else
                {
                    icon.Symbol = symbol;
                }
            }
            else if (icon is not null)
            {
                content.Children.Remove(icon);
                icon = null;
            }

            UpdateColors();
        }

        public void SetSelected(bool value, bool animate = true)
        {
            isSelected = value;
            AutomationProperties.SetItemStatus(this, value ? "selected" : string.Empty);
            if (animate)
            {
                selection.To(value ? 1 : 0);
            }
            else
            {
                selection.Snap(value ? 1 : 0);
            }

            UpdateColors();
        }

        protected override void StateChanged()
        {
            press.To(IsPressed ? 1 : 0);
            layerOpacity.To(IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
            ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
        }

        private void ApplyShape()
        {
            double full = Height40 / 2;
            double inner = Lerp(Lerp(Corners.Small, full, selection.Value), Corners.ExtraSmall, press.Value);
            inner = Math.Min(Math.Max(inner, 0), full);
            double left = first ? full : inner;
            double right = last ? full : inner;
            CornerRadius radii = new(left, right, right, left);
            container.CornerRadius = radii;
            layer.CornerRadius = radii;
            ring.CornerRadius = new CornerRadius(left + 3, right + 3, right + 3, left + 3);
        }

        private static double Lerp(double a, double b, double t) => a + ((b - a) * t);

        private void UpdateColors()
        {
            AppTheme theme = Ui.Theme;
            (Brush containerBrush, Brush contentBrush) = isSelected ? (theme.Primary, theme.OnPrimary) : (theme.SecondaryContainer, theme.OnSecondaryContainer);
            container.Background = containerBrush;
            layer.Background = contentBrush;
            text.Foreground = contentBrush;
            if (icon is not null)
            {
                icon.Foreground = contentBrush;
            }
        }
    }
}
