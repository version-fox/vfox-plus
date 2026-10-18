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

namespace VfoxPlus.App.Controls;

/// <summary>
/// A surface with rounded corners that may be clicked: the content is cut to the corners, and a state layer in the
/// colour of the content shows the pointer.
/// </summary>
internal class M3Card : Pressable
{
    private readonly Border container = new();
    private readonly Border layer = new() { Opacity = 0, IsHitTestVisible = false };
    private readonly Border ring = new() { BorderThickness = new Thickness(2), Visibility = Visibility.Collapsed, IsHitTestVisible = false };
    private readonly Grid body = new();
    private readonly Animated layerOpacity;
    private bool interactive;

    public M3Card()
    {
        Grid root = new();
        root.Children.Add(container);
        root.Children.Add(layer);
        container.Child = body;
        root.Children.Add(ring);
        Content = root;
        IsTabStop = false;
        HorizontalAlignment = HorizontalAlignment.Stretch;
        layerOpacity = new Animated(MotionScheme.Expressive.FastEffects, value => layer.Opacity = value);
        ring.BorderBrush = Ui.Theme.Secondary;
        Radius = 16;
    }

    /// <summary>Gets or sets the colour of the card.</summary>
    public Brush Color
    {
        get => container.Background;
        set => container.Background = value;
    }

    /// <summary>Gets or sets the colour of the content, which the state layer has.</summary>
    public Brush ContentBrush
    {
        get => layer.Background;
        set => layer.Background = value;
    }

    /// <summary>Gets or sets the corner radius in dp.</summary>
    public double Radius
    {
        get => container.CornerRadius.TopLeft;
        set
        {
            CornerRadius radii = new(value);
            container.CornerRadius = radii;
            layer.CornerRadius = radii;
            ring.CornerRadius = new CornerRadius(value + 3);
            ring.Margin = new Thickness(-3);
        }
    }

    /// <summary>Gets or sets a value indicating whether the card reacts to the pointer and the keyboard.</summary>
    public bool Interactive
    {
        get => interactive;
        set
        {
            interactive = value;
            IsTabStop = value;
        }
    }

    /// <summary>Gets or sets what the card holds.</summary>
    public UIElement? Child
    {
        get => body.Children.FirstOrDefault();
        set
        {
            body.Children.Clear();
            if (value is not null)
            {
                body.Children.Add(value);
            }
        }
    }

    /// <summary>Gets or sets the name of the card for the tests and assistive technologies.</summary>
    public string CardName
    {
        get => AutomationProperties.GetName(this);
        set => AutomationProperties.SetName(this, value);
    }

    /// <inheritdoc/>
    protected override void StateChanged()
    {
        layerOpacity?.To(interactive && IsEnabled ? (IsPressed || HasKeyboardFocus ? 0.10 : IsHovered ? 0.08 : 0) : 0);
        ring.Visibility = interactive && HasKeyboardFocus ? Visibility.Visible : Visibility.Collapsed;
    }
}
