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
using VfoxPlus.Design.Motion;
using Windows.Foundation;

namespace VfoxPlus.App.Controls;

/// <summary>
/// Shows its content under a height that follows a spring between 0 and the height of the content, so that content
/// appears and disappears by growing.
/// </summary>
internal sealed class Collapsible : Grid
{
    private readonly Animated open;
    private bool isOpen;

    public Collapsible()
    {
        open = new Animated(MotionScheme.Expressive.DefaultSpatial, Apply);
        Visibility = Visibility.Collapsed;
        SizeChanged += (_, e) => Clip = new RectangleGeometry { Rect = new Rect(0, 0, e.NewSize.Width, e.NewSize.Height) };
    }

    /// <summary>Gets a value indicating whether the content is shown, or on its way.</summary>
    public bool IsOpen => isOpen;

    /// <summary>Opens or closes the content.</summary>
    /// <param name="value">Whether to show it.</param>
    /// <param name="animate">Whether to grow or shrink; otherwise it appears at once.</param>
    public void SetOpen(bool value, bool animate = true)
    {
        isOpen = value;
        if (animate)
        {
            open.To(value ? 1 : 0);
        }
        else
        {
            open.Snap(value ? 1 : 0);
        }
    }

    private void Apply(double t)
    {
        double progress = Math.Min(Math.Max(t, 0), 1);
        if (progress <= 0.001 && !isOpen)
        {
            Visibility = Visibility.Collapsed;
            Height = double.NaN;
            return;
        }

        Visibility = Visibility.Visible;
        if (progress >= 0.999 && isOpen)
        {
            Height = double.NaN;
            return;
        }

        double natural = 0;
        foreach (UIElement child in Children)
        {
            child.Measure(new Size(ActualWidth > 0 ? ActualWidth : double.PositiveInfinity, double.PositiveInfinity));
            natural = Math.Max(natural, child.DesiredSize.Height);
        }

        Height = natural * progress;
    }
}
