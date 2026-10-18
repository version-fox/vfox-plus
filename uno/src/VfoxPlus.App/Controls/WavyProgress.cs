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
using Windows.Foundation;

namespace VfoxPlus.App.Controls;

/// <summary>
/// The wavy linear progress indicator of Material 3 Expressive (see
/// <c>docs/design-system.md#wavy-progress-indicator</c>), determinate or indeterminate.
/// </summary>
internal sealed class WavyProgress : Grid
{
    private const double Stroke = 4;
    private const double AmplitudeDp = 3;
    private const double Gap = 4;

    private readonly Microsoft.UI.Xaml.Shapes.Path track = new() { StrokeThickness = Stroke, StrokeStartLineCap = PenLineCap.Round, StrokeEndLineCap = PenLineCap.Round, IsHitTestVisible = false };
    private readonly Microsoft.UI.Xaml.Shapes.Path active = new() { StrokeThickness = Stroke, StrokeStartLineCap = PenLineCap.Round, StrokeEndLineCap = PenLineCap.Round, IsHitTestVisible = false };
    private readonly Microsoft.UI.Xaml.Shapes.Path track2 = new() { StrokeThickness = Stroke, StrokeStartLineCap = PenLineCap.Round, StrokeEndLineCap = PenLineCap.Round, IsHitTestVisible = false };
    private readonly Microsoft.UI.Xaml.Shapes.Path active2 = new() { StrokeThickness = Stroke, StrokeStartLineCap = PenLineCap.Round, StrokeEndLineCap = PenLineCap.Round, IsHitTestVisible = false };
    private readonly Ellipse stop = new() { Width = Stroke / 2, Height = Stroke / 2, HorizontalAlignment = HorizontalAlignment.Right, VerticalAlignment = VerticalAlignment.Center, Margin = new Thickness(0, 0, Stroke / 2, 0), IsHitTestVisible = false };
    private readonly Animated amplitude;
    private readonly Animated shown;
    private readonly TimeSpan started = FrameClock.Instance.Now;
    private bool attached;
    private bool indeterminate;
    private double value;

    public WavyProgress()
    {
        AppTheme theme = Ui.Theme;
        Height = Stroke + (AmplitudeDp * 2);
        HorizontalAlignment = HorizontalAlignment.Stretch;
        track.Stroke = theme.SecondaryContainer;
        track2.Stroke = theme.SecondaryContainer;
        active.Stroke = theme.Primary;
        active2.Stroke = theme.Primary;
        stop.Fill = theme.Primary;
        foreach (UIElement element in new UIElement[] { track, track2, active, active2, stop })
        {
            Children.Add(element);
        }

        amplitude = new Animated(MotionScheme.Expressive.DefaultEffects, _ => Redraw(), 1);
        shown = new Animated(MotionScheme.Expressive.DefaultSpatial, _ => Redraw());
        SizeChanged += (_, _) => Redraw();
        Loaded += (_, _) =>
        {
            if (!attached)
            {
                attached = true;
                FrameClock.Instance.Run(OnFrame);
            }
        };
        Unloaded += (_, _) => attached = false;
    }

    /// <summary>Gets or sets the progress from 0 to 1; ignored while the indicator is indeterminate.</summary>
    public double Value
    {
        get => value;
        set
        {
            this.value = Math.Min(1, Math.Max(0, value));
            amplitude.To(!indeterminate && (this.value <= 0.1 || this.value >= 0.95) ? 0 : 1);
            shown.To(this.value);
        }
    }

    /// <summary>Gets or sets a value indicating whether the indicator cycles without a value.</summary>
    public bool Indeterminate
    {
        get => indeterminate;
        set
        {
            indeterminate = value;
            amplitude.To(1);
            Redraw();
        }
    }

    /// <summary>Gets or sets the name of the indicator for assistive technologies.</summary>
    public string Label
    {
        get => AutomationProperties.GetName(this);
        set => AutomationProperties.SetName(this, value);
    }

    private bool OnFrame(double seconds)
    {
        if (!attached)
        {
            return false;
        }

        if (!FrameClock.Instance.ReducedMotion)
        {
            Redraw();
        }

        return true;
    }

    private static void SetLine(Microsoft.UI.Xaml.Shapes.Path path, double from, double to, double y)
    {
        // A track is a line with round ends; one that is shorter than its caps is not drawn.
        if (to - from <= Stroke)
        {
            path.Data = null;
            return;
        }

        path.Data = new PathGeometry
        {
            Figures = { new PathFigure { StartPoint = new Point(from + (Stroke / 2), y), Segments = { new LineSegment { Point = new Point(to - (Stroke / 2), y) } } } },
        };
    }

    private static void SetWave(Microsoft.UI.Xaml.Shapes.Path path, double from, double to, double y, double amplitudeValue, double wavelength, double phase)
    {
        if (to - from <= 0)
        {
            path.Data = null;
            return;
        }

        double start = from + (Stroke / 2);
        double end = Math.Max(start, to - (Stroke / 2));
        IReadOnlyList<(double X, double Y)> points = Progress.Wave(start, end, y, amplitudeValue, wavelength, phase);
        PolyLineSegment line = new();
        foreach ((double px, double py) in points.Skip(1))
        {
            line.Points.Add(new Point(px, py));
        }

        path.Data = new PathGeometry { Figures = { new PathFigure { StartPoint = new Point(points[0].X, points[0].Y), Segments = { line } } } };
    }

    private void Redraw()
    {
        double width = ActualWidth;
        if (width <= 0)
        {
            return;
        }

        double cy = Height / 2;
        double amp = AmplitudeDp * Math.Min(1, Math.Max(0, amplitude.Value));
        double wavelength = indeterminate ? 20 : 40;
        double clock = FrameClock.Instance.ReducedMotion ? 0 : (FrameClock.Instance.Now - started).TotalSeconds % Progress.WavyCycle.TotalSeconds / Progress.WavyCycle.TotalSeconds;

        // The wave travels one wavelength per second.
        double phase = clock * 2 * 2 * Math.PI;
        if (!indeterminate)
        {
            double progress = Math.Min(1, Math.Max(0, shown.Value));
            double activeEnd = width * progress;
            SetWave(active, 0, activeEnd, cy, amp, wavelength, phase);
            SetLine(track, progress > 0 ? activeEnd + Gap : 0, width, cy);
            track2.Data = null;
            active2.Data = null;
            stop.Visibility = progress < 1 ? Visibility.Visible : Visibility.Collapsed;
            return;
        }

        stop.Visibility = Visibility.Collapsed;
        (double h1, double t1, double h2, double t2) = Progress.Indeterminate(clock);
        h1 *= width;
        t1 *= width;
        h2 *= width;
        t2 *= width;
        double cursor = 0;
        active2.Data = null;
        track2.Data = null;
        if (h2 > t2)
        {
            SetLine(track2, cursor, t2 - Gap, cy);
            SetWave(active2, t2, h2, cy, amp, wavelength, phase);
            cursor = h2 + Gap;
        }

        if (h1 > t1)
        {
            SetLine(track, cursor, t1 - Gap, cy);
            SetWave(active, t1, h1, cy, amp, wavelength, phase);
            cursor = h1 + Gap;
        }
        else
        {
            active.Data = null;
            track.Data = null;
        }

        // What is left of the track after the bars.
        stop.Visibility = Visibility.Collapsed;
        if (cursor < width)
        {
            Microsoft.UI.Xaml.Shapes.Path rest = h1 > t1 ? track : track2;
            if (h1 <= t1 && h2 <= t2)
            {
                rest = track;
            }

            SetLine(rest == track && h1 > t1 ? track2 : rest, cursor, width, cy);
        }
    }
}
