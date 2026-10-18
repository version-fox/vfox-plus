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
using Microsoft.UI.Xaml.Input;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Layout;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;
using Windows.Foundation;
using Windows.System;

namespace VfoxPlus.App.Controls.Overlays;

/// <summary>An entry of a menu.</summary>
/// <param name="Label">The text of the entry.</param>
/// <param name="OnClick">Runs when the entry is chosen, after the menu has closed.</param>
/// <param name="Symbol">The icon before the text, if any.</param>
/// <param name="Danger">Draws the entry in the colour of errors, for destructive actions.</param>
/// <param name="Disabled">Makes the entry inert.</param>
/// <param name="Selected">Marks the entry that holds the current value of a dropdown.</param>
internal sealed record MenuItem(string Label, Action? OnClick = null, string? Symbol = null, bool Danger = false, bool Disabled = false, bool Selected = false);

/// <summary>How a menu is placed.</summary>
/// <param name="AlignEnd">Aligns the right edge of the menu with the right edge of its anchor, for anchors at the end of a row.</param>
/// <param name="MatchWidth">Makes the menu at least as wide as its anchor, for dropdowns.</param>
internal sealed record MenuOptions(bool AlignEnd = false, bool MatchWidth = false);

/// <summary>
/// A menu: a surface below its anchor that opens from the corner next to the anchor on a spring, and closes when an
/// entry is chosen, when the user presses elsewhere, and on Escape.
/// </summary>
internal sealed class MenuPopup
{
    private readonly OverlayHost host;
    private readonly FrameworkElement anchor;
    private readonly MenuOptions options;
    private readonly List<MenuRow> rows = [];
    private readonly ScaleTransform scale = new();
    private readonly Animated opening;
    private bool closed;

    public MenuPopup(OverlayHost host, FrameworkElement anchor, IReadOnlyList<MenuItem> items, MenuOptions options)
    {
        this.host = host;
        this.anchor = anchor;
        this.options = options;
        AppTheme theme = Ui.Theme;
        StackPanel column = new() { Padding = new Thickness(0, Gaps.Xs, 0, Gaps.Xs), Orientation = Orientation.Vertical };
        foreach (MenuItem item in items)
        {
            MenuRow row = new(item, this);
            row.KeyDown += OnRowKeyDown;
            rows.Add(row);
            column.Children.Add(row);
        }

        Surface = new Border
        {
            Background = theme.SurfaceContainer,
            CornerRadius = new CornerRadius(Corners.Large),
            Child = column,
            MinWidth = 112,
            MaxWidth = 280,
            RenderTransform = scale,
            Opacity = 0,
        };
        Elevation.Set(Surface, 3);
        Scrim = new Border { Background = theme.Transparent };
        Scrim.PointerPressed += (_, e) =>
        {
            e.Handled = true;
            Close();
        };
        opening = new Animated(MotionScheme.Expressive.DefaultSpatial, Apply);
    }

    /// <summary>Raised when the menu has closed, however it did.</summary>
    public event EventHandler? Dismissed;

    /// <summary>Gets what covers the window behind the menu, which a press on closes.</summary>
    public Border Scrim { get; }

    /// <summary>Gets the menu itself.</summary>
    public Border Surface { get; }

    /// <summary>Gets the entries, which the tests choose.</summary>
    public IReadOnlyList<Pressable> Rows => rows;

    /// <summary>Opens the menu at its anchor.</summary>
    public void Open()
    {
        Scrim.Width = host.ActualWidth;
        Scrim.Height = host.ActualHeight;
        double window = host.ActualWidth;
        Box at = host.BoundsOf(anchor);
        Surface.MaxWidth = Math.Min(280, window - 16);
        if (options.MatchWidth)
        {
            Surface.MinWidth = Math.Min(Math.Max(112, at.Width), window - 16);
            Surface.MaxWidth = Math.Max(Surface.MaxWidth, Surface.MinWidth);
        }

        Surface.Measure(new Size(Surface.MaxWidth, host.ActualHeight));
        Size size = Surface.DesiredSize;
        (double x, double y) = Placement.Place(window, host.ActualHeight, at, size.Width, size.Height, options.AlignEnd, 8);
        Canvas.SetLeft(Surface, x);
        Canvas.SetTop(Surface, y);

        // The menu opens from the corner next to the anchor.
        Surface.RenderTransformOrigin = new Point(options.AlignEnd ? 1 : 0, 0);
        opening.Snap(0);
        opening.To(1);
        rows.FirstOrDefault(row => row.IsEnabled)?.Focus(FocusState.Programmatic);
    }

    /// <summary>Closes the menu.</summary>
    public void Close()
    {
        if (closed)
        {
            return;
        }

        closed = true;
        host.Closed(this);
        (anchor as Control)?.Focus(FocusState.Programmatic);
        Dismissed?.Invoke(this, EventArgs.Empty);
    }

    private void Apply(double open)
    {
        Surface.Opacity = Math.Min(1, Math.Max(0, open * 1.5));
        double factor = 0.8 + (0.2 * Math.Min(open, 1.1));
        scale.ScaleX = factor;
        scale.ScaleY = factor;
    }

    private void OnRowKeyDown(object sender, KeyRoutedEventArgs e)
    {
        int step = e.Key switch
        {
            VirtualKey.Down => 1,
            VirtualKey.Up => -1,
            _ => 0,
        };
        if (step == 0)
        {
            return;
        }

        e.Handled = true;
        int index = rows.IndexOf((MenuRow)sender);
        for (int i = index + step; i >= 0 && i < rows.Count; i += step)
        {
            if (rows[i].IsEnabled)
            {
                rows[i].Focus(FocusState.Keyboard);
                return;
            }
        }
    }

    private sealed class MenuRow : Pressable
    {
        private readonly Border layer = new() { Opacity = 0, IsHitTestVisible = false };
        private readonly Animated layerOpacity;
        private readonly Brush foreground;

        public MenuRow(MenuItem item, MenuPopup menu)
        {
            AppTheme theme = Ui.Theme;
            foreground = item.Danger ? theme.Error : item.Selected ? theme.OnSecondaryContainer : theme.OnSurface;
            Brush iconBrush = item.Danger ? theme.Error : item.Selected ? theme.OnSecondaryContainer : theme.OnSurfaceVariant;
            if (item.Disabled)
            {
                foreground = theme.OnSurface38;
                iconBrush = foreground;
            }

            StackPanel row = new()
            {
                Orientation = Orientation.Horizontal,
                Spacing = 12,
                Padding = new Thickness(12, 0, 12, 0),
                VerticalAlignment = VerticalAlignment.Center,
            };
            if (item.Symbol is { } symbol)
            {
                row.Children.Add(new M3Icon(symbol, 24, iconBrush));
            }

            TextBlock label = Ui.Text(item.Label, TextRole.BodyLarge, foreground);
            label.VerticalAlignment = VerticalAlignment.Center;
            row.Children.Add(label);
            layer.Background = foreground;
            Grid root = new() { MinHeight = 48, Background = item.Selected ? theme.SecondaryContainer : theme.Transparent };
            root.Children.Add(layer);
            root.Children.Add(row);
            Content = root;
            HorizontalAlignment = HorizontalAlignment.Stretch;
            AutomationProperties.SetName(this, item.Label);
            layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
            IsEnabled = !item.Disabled;
            Click += (_, _) =>
            {
                menu.Close();
                item.OnClick?.Invoke();
            };
        }

        public override string Role => "menuitem";

        protected override void StateChanged() =>
            layerOpacity?.To(IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
    }
}
