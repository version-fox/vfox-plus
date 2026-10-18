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

/// <summary>The style of an icon button.</summary>
internal enum IconButtonKind
{
    /// <summary>No container.</summary>
    Standard,

    /// <summary>A primary container.</summary>
    Filled,

    /// <summary>A secondary container.</summary>
    Tonal,
}

/// <summary>
/// A button that shows only an icon. Its tooltip is its accessible name. The button takes at least 48 dp for the
/// pointer, round at rest and squarer while it is pressed.
/// </summary>
internal sealed class M3IconButton : Pressable
{
    private readonly Border container = new() { IsHitTestVisible = false };
    private readonly Border layer = new() { Opacity = 0, IsHitTestVisible = false };
    private readonly Border ring = new() { BorderThickness = new Thickness(2), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
    private readonly M3Icon icon = new("add");
    private readonly Animated shape;
    private readonly Animated layerOpacity;
    private IconButtonKind kind;
    private double diameter = 40;
    private bool selected;
    private string tooltip = string.Empty;
    private Brush? iconBrush;

    public M3IconButton()
    {
        HorizontalAlignment = HorizontalAlignment.Left;
        VerticalAlignment = VerticalAlignment.Center;
        container.HorizontalAlignment = HorizontalAlignment.Center;
        container.VerticalAlignment = VerticalAlignment.Center;
        layer.HorizontalAlignment = HorizontalAlignment.Center;
        layer.VerticalAlignment = VerticalAlignment.Center;
        ring.HorizontalAlignment = HorizontalAlignment.Center;
        ring.VerticalAlignment = VerticalAlignment.Center;
        icon.HorizontalAlignment = HorizontalAlignment.Center;
        icon.VerticalAlignment = VerticalAlignment.Center;
        icon.RenderTransformOrigin = new Windows.Foundation.Point(0.5, 0.5);
        Grid root = new();
        root.Children.Add(container);
        root.Children.Add(layer);
        root.Children.Add(ring);
        root.Children.Add(icon);
        Content = root;
        shape = new Animated(MotionScheme.Expressive.FastSpatial, ApplyShape);
        layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
        ring.BorderBrush = Ui.Theme.Secondary;
        Resize();
        shape.Apply();
        UpdateColors();
    }

    /// <summary>Gets or sets the style of the button.</summary>
    public IconButtonKind Kind
    {
        get => kind;
        set
        {
            kind = value;
            UpdateColors();
        }
    }

    /// <summary>Gets or sets what the icon means, as in <c>shared/icons.json</c>.</summary>
    public string Symbol
    {
        get => icon.Symbol;
        set => icon.Symbol = value;
    }

    /// <summary>Gets or sets the text that names the button, and shows when the pointer rests on it.</summary>
    public string Tooltip
    {
        get => tooltip;
        set
        {
            tooltip = value;
            AutomationProperties.SetName(this, value);
            Tip.SetText(this, value);
        }
    }

    /// <summary>Gets or sets a value indicating whether the button is a toggle that is on: its icon is filled.</summary>
    public bool Selected
    {
        get => selected;
        set
        {
            selected = value;
            icon.FillTo(value ? 1 : 0);
            AutomationProperties.SetItemStatus(this, value ? "selected" : string.Empty);
        }
    }

    /// <summary>Gets or sets the diameter of the container in dp; the button takes at least 48 dp for the pointer.</summary>
    public double Diameter
    {
        get => diameter;
        set
        {
            diameter = value;
            Resize();
            ApplyShape(shape.Value);
        }
    }

    /// <summary>Gets or sets the colour of the icon; the default depends on the style.</summary>
    public Brush? IconBrush
    {
        get => iconBrush;
        set
        {
            iconBrush = value;
            UpdateColors();
        }
    }

    /// <summary>Gets or sets the angle in degrees that the icon is turned by: the chevron of a button that expands something.</summary>
    public double IconAngle
    {
        get => (icon.RenderTransform as RotateTransform)?.Angle ?? 0;
        set => icon.RenderTransform = new RotateTransform { Angle = value };
    }

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        shape.To(IsPressed ? 1 : 0);
        layerOpacity.To(IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
        ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
        UpdateColors();
    }

    private void Resize()
    {
        double box = Math.Max(48, diameter);
        Width = box;
        Height = box;
        container.Width = diameter;
        container.Height = diameter;
        layer.Width = diameter;
        layer.Height = diameter;
        ring.Width = diameter + 6;
        ring.Height = diameter + 6;
    }

    private void ApplyShape(double pressed)
    {
        double radius = Math.Max(0, diameter / 2 + ((Corners.Medium - (diameter / 2)) * pressed));
        container.CornerRadius = new CornerRadius(radius);
        layer.CornerRadius = new CornerRadius(radius);
        ring.CornerRadius = new CornerRadius(radius + 3);
    }

    private void UpdateColors()
    {
        AppTheme theme = Ui.Theme;
        (Brush containerBrush, Brush content) = kind switch
        {
            IconButtonKind.Filled => (theme.Primary, theme.OnPrimary),
            IconButtonKind.Tonal => (theme.SecondaryContainer, theme.OnSecondaryContainer),
            _ => (theme.Transparent, theme.OnSurfaceVariant),
        };
        content = iconBrush ?? content;
        if (!IsEnabled)
        {
            content = theme.OnSurface38;
            containerBrush = kind == IconButtonKind.Standard ? theme.Transparent : theme.OnSurface12;
        }

        container.Background = containerBrush;
        layer.Background = content;
        icon.Foreground = content;
    }
}
