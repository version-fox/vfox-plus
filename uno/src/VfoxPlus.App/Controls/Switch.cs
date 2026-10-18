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
using Microsoft.UI.Xaml.Shapes;
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;

namespace VfoxPlus.App.Controls;

/// <summary>
/// A Material 3 switch: a track with a thumb that slides, and grows while it is pressed. The colours change with the
/// position of the thumb, which the opacity of two layers does.
/// </summary>
internal sealed class M3Switch : Pressable
{
    private const double TrackWidth = 52;
    private const double TrackHeight = 32;

    private readonly Border onTrack = new() { CornerRadius = new CornerRadius(TrackHeight / 2) };
    private readonly Border outline = new() { CornerRadius = new CornerRadius(TrackHeight / 2), BorderThickness = new Thickness(2) };
    private readonly Ellipse offThumb = new();
    private readonly Ellipse onThumb = new();
    private readonly Ellipse offState = new() { Width = 40, Height = 40, Opacity = 0, IsHitTestVisible = false };
    private readonly Ellipse onState = new() { Width = 40, Height = 40, Opacity = 0, IsHitTestVisible = false };
    private readonly M3Icon check = new("check", 16, Ui.Theme.OnPrimaryContainer);
    private readonly Border ring = new() { BorderThickness = new Thickness(2), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
    private readonly Canvas canvas = new() { Width = TrackWidth, Height = TrackHeight };
    private readonly Animated slide;
    private readonly Animated grow;
    private readonly Animated state;
    private bool isOn;
    private string label = string.Empty;

    public M3Switch()
    {
        AppTheme theme = Ui.Theme;
        HorizontalAlignment = HorizontalAlignment.Left;
        VerticalAlignment = VerticalAlignment.Center;
        Grid root = new() { Width = TrackWidth, Height = TrackHeight };
        Border offTrack = new() { CornerRadius = new CornerRadius(TrackHeight / 2), Background = theme.SurfaceContainerHighest };
        root.Children.Add(offTrack);
        root.Children.Add(onTrack);
        root.Children.Add(outline);
        root.Children.Add(canvas);
        root.Children.Add(ring);
        canvas.Children.Add(offState);
        canvas.Children.Add(onState);
        canvas.Children.Add(offThumb);
        canvas.Children.Add(onThumb);
        canvas.Children.Add(check);
        ring.CornerRadius = new CornerRadius((TrackHeight / 2) + 3);
        ring.Margin = new Thickness(-3);
        ring.BorderBrush = theme.Secondary;
        Content = root;
        slide = new Animated(MotionScheme.Expressive.FastSpatial, _ => Place());
        grow = new Animated(MotionScheme.Expressive.FastSpatial, _ => Place());
        state = new Animated(MotionScheme.Expressive.FastEffects, _ => Place());
        UpdateColors();
        Place();
    }

    /// <summary>Raised when the user flips the switch.</summary>
    public event EventHandler<bool>? Toggled;

    /// <summary>Gets or sets a value indicating whether the switch is on.</summary>
    public bool IsOn
    {
        get => isOn;
        set
        {
            isOn = value;
            slide.To(value ? 1 : 0);
            AutomationProperties.SetItemStatus(this, value ? "on" : "off");
        }
    }

    /// <summary>Gets or sets what the switch is for, which names it.</summary>
    public string Label
    {
        get => label;
        set
        {
            label = value;
            AutomationProperties.SetName(this, value);
        }
    }

    /// <inheritdoc/>
    public override string Role => "switch";

    /// <summary>Sets the state without animating it, as when a page is first built.</summary>
    /// <param name="on">Whether the switch is on.</param>
    public void Set(bool on)
    {
        isOn = on;
        slide.Snap(on ? 1 : 0);
        AutomationProperties.SetItemStatus(this, on ? "on" : "off");
    }

    /// <inheritdoc/>
    protected override void OnClicked() => Toggled?.Invoke(this, !isOn);

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        grow.To(IsPressed ? 1 : 0);
        state.To(IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
        ring.Visibility = HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
        UpdateColors();
    }

    private void Place()
    {
        double position = slide.Value;
        double diameter = ((16 + (8 * position)) * (1 - grow.Value)) + (28 * grow.Value);
        double travel = TrackWidth - 8 - 24;
        double centerX = 4 + 12 + (travel * position);
        double centerY = TrackHeight / 2;
        onTrack.Opacity = position;
        outline.Opacity = 1 - position;
        foreach (Ellipse thumb in new[] { offThumb, onThumb })
        {
            thumb.Width = diameter;
            thumb.Height = diameter;
            Canvas.SetLeft(thumb, centerX - (diameter / 2));
            Canvas.SetTop(thumb, centerY - (diameter / 2));
        }

        offThumb.Opacity = 1 - position;
        onThumb.Opacity = position;
        Canvas.SetLeft(offState, centerX - 20);
        Canvas.SetTop(offState, centerY - 20);
        Canvas.SetLeft(onState, centerX - 20);
        Canvas.SetTop(onState, centerY - 20);
        offState.Opacity = state.Value * (1 - position);
        onState.Opacity = state.Value * position;
        Canvas.SetLeft(check, centerX - 8);
        Canvas.SetTop(check, centerY - 8);
        check.Opacity = position;
    }

    private void UpdateColors()
    {
        AppTheme theme = Ui.Theme;
        if (IsEnabled)
        {
            onTrack.Background = theme.Primary;
            outline.BorderBrush = theme.Outline;
            offThumb.Fill = theme.Outline;
            onThumb.Fill = theme.OnPrimary;
        }
        else
        {
            onTrack.Background = theme.OnSurface12;
            outline.BorderBrush = theme.OnSurface12;
            offThumb.Fill = theme.OnSurface38;
            onThumb.Fill = theme.Surface;
        }

        offState.Fill = theme.OnSurface;
        onState.Fill = theme.Primary;
        check.Foreground = theme.OnPrimaryContainer;
    }
}
