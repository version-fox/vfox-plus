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

/// <summary>
/// A button with an action and a menu next to it. The trailing button, which opens the menu, becomes a circle while the
/// menu is open.
/// </summary>
internal sealed class M3SplitButton : UserControl
{
    private const double Height40 = 40;

    private readonly Part leading;
    private readonly Part trailing;
    private IReadOnlyList<MenuItem> items = [];
    private MenuPopup? menu;

    public M3SplitButton()
    {
        leading = new Part(isLeading: true);
        trailing = new Part(isLeading: false);
        trailing.Click += (_, _) => ToggleMenu();
        leading.Click += (_, _) => Click?.Invoke(this, EventArgs.Empty);
        StackPanel row = new() { Orientation = Orientation.Horizontal, Spacing = Gaps.Hairline, VerticalAlignment = VerticalAlignment.Center };
        row.Children.Add(leading);
        row.Children.Add(trailing);
        Content = row;
        IsTabStop = false;
        HorizontalAlignment = HorizontalAlignment.Left;
        IsEnabledChanged += (_, _) =>
        {
            leading.IsEnabled = IsEnabled;
            trailing.IsEnabled = IsEnabled;
        };
    }

    /// <summary>Raised when the action is run.</summary>
    public event EventHandler? Click;

    /// <summary>Gets or sets the text of the action.</summary>
    public string Label
    {
        get => leading.Label;
        set => leading.Label = value;
    }

    /// <summary>Gets or sets the icon of the action, if any.</summary>
    public string? Symbol
    {
        get => leading.Symbol;
        set => leading.Symbol = value;
    }

    /// <summary>Gets or sets the name of the trailing button.</summary>
    public string MenuTooltip
    {
        get => trailing.Label;
        set => trailing.Label = value;
    }

    /// <summary>Gets or sets the entries of the menu.</summary>
    public IReadOnlyList<MenuItem> Items
    {
        get => items;
        set => items = value;
    }

    /// <summary>Gets the button of the action, which the tests choose.</summary>
    public Pressable Action => leading;

    /// <summary>Gets the button that opens the menu, which the tests choose.</summary>
    public Pressable MenuButton => trailing;

    private void ToggleMenu()
    {
        if (menu is not null || OverlayHost.Current is not { } host)
        {
            return;
        }

        trailing.SetOpen(true);
        menu = host.ShowMenu(trailing, items, new MenuOptions());

        // Closing the menu, whatever closes it, turns the trailing button back.
        menu.Dismissed += (_, _) =>
        {
            menu = null;
            trailing.SetOpen(false);
        };
    }

    private sealed class Part : Pressable
    {
        private readonly bool isLeading;
        private readonly Border container = new();
        private readonly Border layer = new() { Opacity = 0, IsHitTestVisible = false };
        private readonly Border ring = new() { BorderThickness = new Thickness(2), Margin = new Thickness(-3), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
        private readonly StackPanel content = new() { Orientation = Orientation.Horizontal, Spacing = 8, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center };
        private readonly TextBlock text = Ui.Text(string.Empty, TextRole.LabelLarge);
        private readonly M3Icon chevron = new("expandMore", 22) { RenderTransformOrigin = new Windows.Foundation.Point(0.5, 0.5) };
        private readonly Animated press;
        private readonly Animated open;
        private readonly Animated layerOpacity;
        private M3Icon? icon;
        private string symbol = string.Empty;

        public Part(bool isLeading)
        {
            this.isLeading = isLeading;
            Height = Height40;
            Grid root = new();
            root.Children.Add(container);
            root.Children.Add(layer);
            root.Children.Add(ring);
            if (isLeading)
            {
                content.Margin = new Thickness(16, 0, 12, 0);
                text.VerticalAlignment = VerticalAlignment.Center;
                content.Children.Add(text);
                root.Children.Add(content);
            }
            else
            {
                Width = 44;
                chevron.HorizontalAlignment = HorizontalAlignment.Center;
                chevron.VerticalAlignment = VerticalAlignment.Center;
                root.Children.Add(chevron);
            }

            Content = root;
            press = new Animated(MotionScheme.Expressive.FastSpatial, _ => ApplyShape());
            open = new Animated(MotionScheme.Expressive.FastSpatial, _ => ApplyShape());
            layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
            ring.BorderBrush = Ui.Theme.Secondary;
            ApplyShape();
            UpdateColors();
        }

        public string Label
        {
            get => text.Text;
            set
            {
                text.Text = value;
                AutomationProperties.SetName(this, value);
                if (!isLeading)
                {
                    Tip.SetText(this, value);
                }
            }
        }

        public string? Symbol
        {
            get => symbol.Length == 0 ? null : symbol;
            set
            {
                symbol = value ?? string.Empty;
                if (value is null)
                {
                    if (icon is not null)
                    {
                        content.Children.Remove(icon);
                        icon = null;
                    }

                    return;
                }

                if (icon is null)
                {
                    icon = new M3Icon(value, 20);
                    content.Children.Insert(0, icon);
                }
                else
                {
                    icon.Symbol = value;
                }

                UpdateColors();
            }
        }

        public void SetOpen(bool value)
        {
            open.To(value ? 1 : 0);
            chevron.RenderTransform = new RotateTransform { Angle = value ? 180 : 0 };
            AutomationProperties.SetItemStatus(this, value ? "expanded" : string.Empty);
        }

        protected override void StateChanged()
        {
            press.To(IsPressed ? 1 : 0);
            layerOpacity.To(IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
            ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
            UpdateColors();
        }

        private void ApplyShape()
        {
            double full = Height40 / 2;
            double inner = Corners.ExtraSmall + ((Corners.Small - Corners.ExtraSmall) * press.Value);
            CornerRadius radii;
            if (isLeading)
            {
                radii = new CornerRadius(full, inner, inner, full);
            }
            else
            {
                double radius = inner + ((full - inner) * open.Value);
                radii = new CornerRadius(radius, full, full, radius);
            }

            container.CornerRadius = radii;
            layer.CornerRadius = radii;
            ring.CornerRadius = new CornerRadius(radii.TopLeft + 3, radii.TopRight + 3, radii.BottomRight + 3, radii.BottomLeft + 3);
        }

        private void UpdateColors()
        {
            AppTheme theme = Ui.Theme;
            Brush containerBrush = IsEnabled ? theme.SecondaryContainer : theme.OnSurface12;
            Brush contentBrush = IsEnabled ? theme.OnSecondaryContainer : theme.OnSurface38;
            container.Background = containerBrush;
            layer.Background = contentBrush;
            text.Foreground = contentBrush;
            chevron.Foreground = contentBrush;
            if (icon is not null)
            {
                icon.Foreground = contentBrush;
            }
        }
    }
}
