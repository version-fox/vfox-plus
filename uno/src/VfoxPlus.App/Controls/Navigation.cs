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

using Microsoft.UI.Xaml.Shapes;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;
using VfoxPlus.Design.Tokens;
using Windows.Foundation;

namespace VfoxPlus.App.Controls;

/// <summary>A destination of the navigation rail or bar.</summary>
/// <param name="Symbol">The icon, filled while the destination is selected.</param>
/// <param name="Label">Names the destination.</param>
/// <param name="Badge">Shows a count on the icon when it is above zero.</param>
internal sealed record NavDestination(string Symbol, string Label, int Badge = 0);

/// <summary>The count on an icon: a small pill in the colour of errors.</summary>
internal sealed class Badge : Border
{
    private readonly TextBlock text;

    public Badge()
    {
        AppTheme theme = Ui.Theme;
        text = Ui.Text(string.Empty, TextRole.LabelSmall, theme.OnError);
        text.HorizontalAlignment = HorizontalAlignment.Center;
        text.VerticalAlignment = VerticalAlignment.Center;
        Background = theme.Error;
        CornerRadius = new CornerRadius(8);
        Height = 16;
        MinWidth = 16;
        Padding = new Thickness(4, 0, 4, 0);
        Child = text;
        IsHitTestVisible = false;
        Visibility = Visibility.Collapsed;
    }

    public void Set(int count)
    {
        text.Text = count > 999 ? "999+" : count.ToString(System.Globalization.CultureInfo.InvariantCulture);
        Visibility = count > 0 ? Visibility.Visible : Visibility.Collapsed;
    }
}

/// <summary>
/// An extended floating action button: a primary container with an icon, and a label as much of which as the
/// expansion reveals.
/// </summary>
internal sealed class Fab : Pressable
{
    private readonly Border container = new() { CornerRadius = new CornerRadius(Corners.Large), Height = 56 };
    private readonly Border layer = new() { CornerRadius = new CornerRadius(Corners.Large), Opacity = 0, IsHitTestVisible = false };
    private readonly Border ring = new() { BorderThickness = new Thickness(2), CornerRadius = new CornerRadius(Corners.Large + 3), Margin = new Thickness(-3), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
    private readonly Canvas canvas = new() { Height = 56 };
    private readonly M3Icon icon;
    private readonly TextBlock label;
    private readonly Animated layerOpacity;
    private double expansion;
    private double labelWidth;

    public Fab(string symbol)
    {
        AppTheme theme = Ui.Theme;
        Height = 56;
        icon = new M3Icon(symbol, 24, theme.OnPrimaryContainer);
        label = Ui.Text(string.Empty, TextRole.TitleSmall, theme.OnPrimaryContainer);
        label.IsHitTestVisible = false;
        container.Background = theme.PrimaryContainer;
        layer.Background = theme.OnPrimaryContainer;
        ring.BorderBrush = theme.Secondary;
        canvas.Children.Add(icon);
        canvas.Children.Add(label);
        Canvas.SetLeft(icon, 16);
        Canvas.SetTop(icon, 16);
        Canvas.SetTop(label, 16);
        Canvas.SetLeft(label, 16 + 24 + 12);
        Grid root = new() { Height = 56 };
        root.Children.Add(container);
        root.Children.Add(layer);
        root.Children.Add(ring);
        root.Children.Add(canvas);
        Content = root;
        HorizontalAlignment = HorizontalAlignment.Left;
        layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
        Elevation.Set(container, 1);
        Apply();
    }

    /// <summary>Gets or sets the text next to the icon of an extended button.</summary>
    public string Label
    {
        get => label.Text;
        set
        {
            label.Text = value;
            labelWidth = MeasureLabel();
            Automation.Name(this, value);
            Apply();
        }
    }

    /// <summary>Gets or sets how far the button is extended, from 0 (the icon alone) to 1.</summary>
    public double Expansion
    {
        get => expansion;
        set
        {
            expansion = Math.Min(1, Math.Max(0, value));
            Apply();
        }
    }

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        layerOpacity?.To(IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
        ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
    }

    // How wide the label is: an element that is not shown has no size, so it is measured as if it were.
    private double MeasureLabel()
    {
        Visibility before = label.Visibility;
        label.Visibility = Visibility.Visible;
        label.Measure(new Size(double.PositiveInfinity, 56));
        double width = label.DesiredSize.Width;
        label.Visibility = before;
        return width;
    }

    private void Apply()
    {
        double width = (16 * 2) + 24 + ((labelWidth + 12 + 4) * expansion);
        Width = width;
        container.Width = width;
        layer.Width = width;
        canvas.Width = width;
        label.Opacity = Math.Min(1, Math.Max(0, (expansion - 0.5) * 2));
        label.Visibility = expansion > 0.5 ? Visibility.Visible : Visibility.Collapsed;
        Tip.SetText(this, expansion < 0.5 ? label.Text : string.Empty);
    }
}

/// <summary>
/// The navigation rail of Material 3 Expressive: 96 dp wide with the labels under the icons, or 240 dp wide with the
/// labels next to them, with a menu button that switches between the two, and a floating action button.
/// </summary>
internal sealed class NavRail : Grid
{
    public const double CollapsedWidth = 96;
    public const double ExpandedWidth = 240;

    private readonly Canvas canvas = new();
    private readonly M3IconButton menu = new() { Symbol = "menu" };
    private readonly Fab fab;
    private readonly List<RailItem> items = [];
    private readonly Animated expand;
    private int selected;
    private bool expanded;

    public NavRail(string fabSymbol)
    {
        fab = new Fab(fabSymbol);
        canvas.Children.Add(menu);
        canvas.Children.Add(fab);
        Children.Add(canvas);
        HorizontalAlignment = HorizontalAlignment.Left;
        expand = new Animated(MotionScheme.Expressive.DefaultSpatial, Place);
        menu.Click += (_, _) => ExpandedChanging?.Invoke(this, !expanded);
        fab.Click += (_, _) => FabClick?.Invoke(this, EventArgs.Empty);
        Place(0);
    }

    /// <summary>Raised with the index of the destination the user chose.</summary>
    public event EventHandler<int>? Selecting;

    /// <summary>Raised when the user presses the menu button, with the state it asks for.</summary>
    public event EventHandler<bool>? ExpandedChanging;

    /// <summary>Raised when the user presses the floating action button.</summary>
    public event EventHandler? FabClick;

    /// <summary>Gets or sets the index of the selected destination.</summary>
    public int Selected
    {
        get => selected;
        set
        {
            selected = value;
            for (int i = 0; i < items.Count; i++)
            {
                items[i].SetSelected(i == value);
            }
        }
    }

    /// <summary>Gets or sets a value indicating whether the labels are next to the icons.</summary>
    public bool Expanded
    {
        get => expanded;
        set
        {
            expanded = value;
            expand.To(value ? 1 : 0);
            UpdateMenu();
        }
    }

    /// <summary>Gets or sets the name of the menu button of the collapsed rail.</summary>
    public string ExpandTooltip { get; set; } = string.Empty;

    /// <summary>Gets or sets the name of the menu button of the expanded rail.</summary>
    public string CollapseTooltip { get; set; } = string.Empty;

    /// <summary>Gets or sets the text of the floating action button.</summary>
    public string FabLabel
    {
        get => fab.Label;
        set => fab.Label = value;
    }

    /// <summary>Gets the items, which the tests choose.</summary>
    public IReadOnlyList<Pressable> Items => items;

    /// <summary>Gets the menu button, which the tests choose.</summary>
    public Pressable MenuButton => menu;

    /// <summary>Gets the floating action button, which the tests choose.</summary>
    public Pressable FabButton => fab;

    /// <summary>Sets the destinations.</summary>
    /// <param name="destinations">The destinations, which replace the ones there are.</param>
    public void SetDestinations(IReadOnlyList<NavDestination> destinations)
    {
        if (items.Count == destinations.Count)
        {
            for (int i = 0; i < items.Count; i++)
            {
                items[i].Update(destinations[i]);
            }

            return;
        }

        foreach (RailItem item in items)
        {
            canvas.Children.Remove(item);
        }

        items.Clear();
        for (int i = 0; i < destinations.Count; i++)
        {
            int index = i;
            RailItem item = new(destinations[i]);
            item.Click += (_, _) => Selecting?.Invoke(this, index);
            item.SetSelected(i == selected, animate: false);
            items.Add(item);
            canvas.Children.Add(item);
        }

        Place(expand.Value);
    }

    private void UpdateMenu()
    {
        menu.Symbol = expanded ? "menuOpen" : "menu";
        menu.Tooltip = expanded ? CollapseTooltip : ExpandTooltip;
    }

    private void Place(double t)
    {
        double width = CollapsedWidth + ((ExpandedWidth - CollapsedWidth) * t);
        Width = width;
        canvas.Width = width;
        double y = 16;

        // The menu button is centred over the collapsed rail and aligned with the icons of the items when expanded.
        Canvas.SetLeft(menu, ((CollapsedWidth - 48) / 2) + ((16 - ((CollapsedWidth - 48) / 2)) * t));
        Canvas.SetTop(menu, y);
        y += 48 + 8;
        Canvas.SetLeft(fab, ((CollapsedWidth - 56) / 2) + ((12 - ((CollapsedWidth - 56) / 2)) * t));
        Canvas.SetTop(fab, y);
        fab.Expansion = t;
        y += 56 + 24;
        double itemHeight = (64 + ((56 - 64) * t)) + 4;
        foreach (RailItem item in items)
        {
            Canvas.SetLeft(item, 0);
            Canvas.SetTop(item, y);
            item.Place(width, itemHeight, t);
            y += itemHeight;
        }
    }

    private sealed class RailItem : Pressable
    {
        private readonly Canvas surface = new();
        private readonly Border indicator = new() { Background = Ui.Theme.SecondaryContainer, Opacity = 0, IsHitTestVisible = false };
        private readonly Border layer = new() { Background = Ui.Theme.OnSurface, Opacity = 0, IsHitTestVisible = false };
        private readonly Border ring = new() { BorderThickness = new Thickness(2), BorderBrush = Ui.Theme.Secondary, Visibility = Visibility.Collapsed, IsHitTestVisible = false };
        private readonly M3Icon icon;
        private readonly Badge badge = new();
        private readonly TextBlock collapsedLabel;
        private readonly TextBlock expandedLabel;
        private readonly Animated selection;
        private readonly Animated layerOpacity;
        private double width = CollapsedWidth;
        private double height = 68;
        private double t;
        private bool isSelected;

        public RailItem(NavDestination destination)
        {
            AppTheme theme = Ui.Theme;
            icon = new M3Icon(destination.Symbol, 24, theme.OnSurfaceVariant);
            collapsedLabel = Ui.Text(destination.Label, Typography.Of(TextRole.LabelMedium), theme.OnSurfaceVariant);
            expandedLabel = Ui.Text(destination.Label, Typography.Of(TextRole.LabelLarge), theme.OnSurfaceVariant);
            collapsedLabel.TextAlignment = TextAlignment.Center;
            collapsedLabel.IsHitTestVisible = false;
            expandedLabel.IsHitTestVisible = false;
            surface.Background = theme.Transparent;
            foreach (UIElement element in new UIElement[] { indicator, layer, ring, icon, badge, collapsedLabel, expandedLabel })
            {
                surface.Children.Add(element);
            }

            Content = surface;
            selection = new Animated(MotionScheme.Expressive.FastSpatial, _ => Apply());
            layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
            Update(destination);
        }

        public void Update(NavDestination destination)
        {
            collapsedLabel.Text = destination.Label;
            expandedLabel.Text = destination.Label;
            icon.Symbol = destination.Symbol;
            badge.Set(destination.Badge);
            Automation.Name(this, destination.Label);
            Apply();
        }

        public void SetSelected(bool value, bool animate = true)
        {
            isSelected = value;
            Automation.Select(this, value);
            icon.FillTo(value ? 1 : 0, animate);
            if (animate)
            {
                selection.To(value ? 1 : 0);
            }
            else
            {
                selection.Snap(value ? 1 : 0);
            }

            Apply();
        }

        public void Place(double newWidth, double newHeight, double expansion)
        {
            width = newWidth;
            height = newHeight;
            t = expansion;
            Apply();
        }

        protected override void StateChanged()
        {
            layerOpacity?.To(IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0);
            ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
        }

        private void Apply()
        {
            AppTheme theme = Ui.Theme;
            Width = width;
            Height = height;
            surface.Width = width;
            surface.Height = height;

            // Collapsed: a 56 x 32 indicator with the label below it. Expanded: a 56 dp high indicator around the icon
            // and the label.
            Rect collapsed = new((CollapsedWidth - 56) / 2, 4, 56, 32);
            Rect expandedRect = new(12, 0, width - 24, 56);
            Rect rect = new(
                collapsed.X + ((expandedRect.X - collapsed.X) * t),
                collapsed.Y + ((expandedRect.Y - collapsed.Y) * t),
                collapsed.Width + ((expandedRect.Width - collapsed.Width) * t),
                collapsed.Height + ((expandedRect.Height - collapsed.Height) * t));
            double s = Math.Min(Math.Max(selection.Value, 0), 1.2);

            // The indicator grows out of the centre of the icon.
            double scale = 0.5 + (0.5 * s);
            double grown = rect.Width * scale;
            double left = rect.X + ((rect.Width - grown) / 2);
            foreach (Border border in new[] { indicator, layer })
            {
                border.Width = grown;
                border.Height = rect.Height;
                border.CornerRadius = new CornerRadius(rect.Height / 2);
                Canvas.SetLeft(border, left);
                Canvas.SetTop(border, rect.Y);
            }

            indicator.Opacity = Math.Min(Math.Max(selection.Value, 0), 1);
            ring.Width = rect.Width + 6;
            ring.Height = rect.Height + 6;
            ring.CornerRadius = new CornerRadius((rect.Height / 2) + 3);
            Canvas.SetLeft(ring, rect.X - 3);
            Canvas.SetTop(ring, rect.Y - 3);

            double iconCenterX = (rect.X + (rect.Width / 2)) + (((rect.X + 16 + 12) - (rect.X + (rect.Width / 2))) * t);
            double iconCenterY = rect.Y + (rect.Height / 2);
            Canvas.SetLeft(icon, iconCenterX - 12);
            Canvas.SetTop(icon, iconCenterY - 12);
            Canvas.SetLeft(badge, iconCenterX + 12 - 8);
            Canvas.SetTop(badge, iconCenterY - 12 - 4);
            icon.Foreground = isSelected ? theme.OnSecondaryContainer : theme.OnSurfaceVariant;

            double collapsedOpacity = 1 - (t * 2);
            collapsedLabel.Opacity = Math.Max(0, collapsedOpacity);
            collapsedLabel.Visibility = collapsedOpacity > 0 ? Visibility.Visible : Visibility.Collapsed;
            collapsedLabel.Width = CollapsedWidth;
            collapsedLabel.FontWeight = new Windows.UI.Text.FontWeight((ushort)(isSelected ? 700 : 500));
            collapsedLabel.Foreground = isSelected ? theme.Secondary : theme.OnSurfaceVariant;
            Canvas.SetLeft(collapsedLabel, 0);
            Canvas.SetTop(collapsedLabel, rect.Y + rect.Height + 4);

            double expandedOpacity = (t - 0.5) * 2;
            expandedLabel.Opacity = Math.Max(0, expandedOpacity);
            expandedLabel.Visibility = expandedOpacity > 0 ? Visibility.Visible : Visibility.Collapsed;
            expandedLabel.FontWeight = new Windows.UI.Text.FontWeight((ushort)(isSelected ? 700 : 500));
            expandedLabel.Foreground = isSelected ? theme.OnSecondaryContainer : theme.OnSurfaceVariant;
            expandedLabel.Width = Math.Max(0, width - (rect.X + 16 + 24 + 12) - 24);
            Canvas.SetLeft(expandedLabel, rect.X + 16 + 24 + 12);
            Canvas.SetTop(expandedLabel, rect.Y + (rect.Height / 2) - 10);
        }
    }
}

/// <summary>
/// The navigation bar of Material 3 for compact windows: the destinations at the bottom, the selected one on an
/// indicator.
/// </summary>
internal sealed class NavBar : Grid
{
    /// <summary>The height of the navigation bar in dp.</summary>
    public const double BarHeight = 64;

    private readonly List<BarItem> items = [];
    private int selected;

    public NavBar()
    {
        Height = BarHeight;
        Background = Ui.Theme.SurfaceContainer;
    }

    /// <summary>Raised with the index of the destination the user chose.</summary>
    public event EventHandler<int>? Selecting;

    /// <summary>Gets or sets the index of the selected destination.</summary>
    public int Selected
    {
        get => selected;
        set
        {
            selected = value;
            for (int i = 0; i < items.Count; i++)
            {
                items[i].SetSelected(i == value);
            }
        }
    }

    /// <summary>Gets the items, which the tests choose.</summary>
    public IReadOnlyList<Pressable> Items => items;

    /// <summary>Sets the destinations.</summary>
    /// <param name="destinations">The destinations, which replace the ones there are.</param>
    public void SetDestinations(IReadOnlyList<NavDestination> destinations)
    {
        if (items.Count == destinations.Count)
        {
            for (int i = 0; i < items.Count; i++)
            {
                items[i].Update(destinations[i]);
            }

            return;
        }

        Children.Clear();
        ColumnDefinitions.Clear();
        items.Clear();
        for (int i = 0; i < destinations.Count; i++)
        {
            int index = i;
            BarItem item = new(destinations[i]);
            item.Click += (_, _) => Selecting?.Invoke(this, index);
            item.SetSelected(i == selected, animate: false);
            ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
            Grid.SetColumn(item, i);
            items.Add(item);
            Children.Add(item);
        }
    }

    private sealed class BarItem : Pressable
    {
        private readonly Border indicator = new() { Width = 64, Height = 32, CornerRadius = new CornerRadius(16), Background = Ui.Theme.SecondaryContainer, Opacity = 0, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Top, Margin = new Thickness(0, 12, 0, 0), IsHitTestVisible = false };
        private readonly Border layer = new() { Width = 64, Height = 32, CornerRadius = new CornerRadius(16), Background = Ui.Theme.OnSurface, Opacity = 0, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Top, Margin = new Thickness(0, 12, 0, 0), IsHitTestVisible = false };
        private readonly Border ring = new() { Width = 70, Height = 38, CornerRadius = new CornerRadius(19), BorderThickness = new Thickness(2), BorderBrush = Ui.Theme.Secondary, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Top, Margin = new Thickness(0, 9, 0, 0), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
        private readonly M3Icon icon;
        private readonly Badge badge = new() { HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Top, Margin = new Thickness(24, 12, 0, 0) };
        private readonly TextBlock label;
        private readonly ScaleTransform grow = new() { ScaleX = 0.5, ScaleY = 1 };
        private readonly Animated selection;
        private readonly Animated layerOpacity;
        private bool isSelected;

        public BarItem(NavDestination destination)
        {
            AppTheme theme = Ui.Theme;
            icon = new M3Icon(destination.Symbol, 24, theme.OnSurfaceVariant) { HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Top, Margin = new Thickness(0, 16, 0, 0) };
            label = Ui.Text(destination.Label, Typography.Of(TextRole.LabelMedium), theme.OnSurfaceVariant);
            label.HorizontalAlignment = HorizontalAlignment.Center;
            label.VerticalAlignment = VerticalAlignment.Top;
            label.Margin = new Thickness(0, 48, 0, 0);
            label.IsHitTestVisible = false;
            indicator.RenderTransform = grow;
            indicator.RenderTransformOrigin = new Point(0.5, 0.5);
            Grid root = new() { Background = theme.Transparent };
            foreach (UIElement element in new UIElement[] { indicator, layer, ring, icon, badge, label })
            {
                root.Children.Add(element);
            }

            Content = root;
            selection = new Animated(MotionScheme.Expressive.FastSpatial, _ => Apply());
            layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
            Update(destination);
        }

        public void Update(NavDestination destination)
        {
            label.Text = destination.Label;
            icon.Symbol = destination.Symbol;
            badge.Set(destination.Badge);
            Automation.Name(this, destination.Label);
        }

        public void SetSelected(bool value, bool animate = true)
        {
            isSelected = value;
            Automation.Select(this, value);
            icon.FillTo(value ? 1 : 0, animate);
            if (animate)
            {
                selection.To(value ? 1 : 0);
            }
            else
            {
                selection.Snap(value ? 1 : 0);
            }

            Apply();
        }

        protected override void StateChanged()
        {
            layerOpacity?.To(IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0);
            ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
        }

        private void Apply()
        {
            AppTheme theme = Ui.Theme;
            double s = Math.Min(Math.Max(selection.Value, 0), 1.2);
            grow.ScaleX = 0.5 + (0.5 * s);
            indicator.Opacity = Math.Min(Math.Max(selection.Value, 0), 1);
            icon.Foreground = isSelected ? theme.OnSecondaryContainer : theme.OnSurfaceVariant;
            label.Foreground = isSelected ? theme.OnSurface : theme.OnSurfaceVariant;
            label.FontWeight = new Windows.UI.Text.FontWeight((ushort)(isSelected ? 700 : 500));
        }
    }
}

/// <summary>
/// A round sample of an accent colour; the one that is selected morphs into a flower while it turns, with a check mark
/// in it.
/// </summary>
internal sealed class ColorSwatch : Pressable
{
    private static readonly Lazy<Design.Shapes.Morph> Flower = new(() => new Design.Shapes.Morph(Design.Shapes.MaterialShapes.Circle, Design.Shapes.MaterialShapes.Flower));

    private readonly ShapeView shape;
    private readonly M3Icon check;
    private readonly Ellipse layer = new() { Width = 48, Height = 48, Opacity = 0, IsHitTestVisible = false };
    private readonly Border ring = new() { Width = 46, Height = 46, CornerRadius = new CornerRadius(23), BorderThickness = new Thickness(2), BorderBrush = Ui.Theme.Secondary, Visibility = Visibility.Collapsed, IsHitTestVisible = false };
    private readonly Animated morph;
    private readonly Animated layerOpacity;
    private readonly SolidColorBrush primary = new();
    private readonly SolidColorBrush onPrimary = new();
    private uint seed;

    public ColorSwatch()
    {
        Width = 48;
        Height = 48;
        shape = new ShapeView(Design.Shapes.MaterialShapes.Circle, 40, primary) { HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center };
        check = new M3Icon("check", 20, onPrimary) { HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center, Opacity = 0 };
        layer.Fill = primary;
        layer.HorizontalAlignment = HorizontalAlignment.Center;
        layer.VerticalAlignment = VerticalAlignment.Center;
        ring.HorizontalAlignment = HorizontalAlignment.Center;
        ring.VerticalAlignment = VerticalAlignment.Center;
        Grid root = new() { Background = Ui.Theme.Transparent };
        root.Children.Add(layer);
        root.Children.Add(shape);
        root.Children.Add(check);
        root.Children.Add(ring);
        Content = root;
        morph = new Animated(MotionScheme.Expressive.FastSpatial, Apply);
        layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
    }

    /// <summary>Gets or sets the accent colour the swatch shows, as ARGB.</summary>
    public uint Seed
    {
        get => seed;
        set
        {
            seed = value;
            Refresh();
        }
    }

    /// <summary>Gets or sets a value indicating whether the swatch shows the colour in use.</summary>
    public bool Selected
    {
        get => morph.Target > 0.5;
        set
        {
            morph.To(value ? 1 : 0);
            Automation.Select(this, value);
        }
    }

    /// <summary>Gets or sets the name of the swatch.</summary>
    public string Label
    {
        get => Automation.NameOf(this);
        set
        {
            Automation.Name(this, value);
            Tip.SetText(this, value);
        }
    }

    /// <summary>Takes the colours of the swatch from the scheme of its seed, in the light or the dark theme.</summary>
    public void Refresh()
    {
        Design.Color.ColorScheme scheme = Design.Color.ColorScheme.Generate(seed, Ui.Theme.Dark);
        primary.Color = AppTheme.ToColor(scheme.Primary);
        onPrimary.Color = AppTheme.ToColor(scheme.OnPrimary);
    }

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        layerOpacity?.To(IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0);
        ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
    }

    private void Apply(double t)
    {
        double progress = Math.Min(1, Math.Max(0, t));
        shape.SetOutline(Flower.Value.AsCubics(progress));
        shape.Angle = t * 0.6 * 180 / Math.PI;
        check.Opacity = progress;
    }
}
