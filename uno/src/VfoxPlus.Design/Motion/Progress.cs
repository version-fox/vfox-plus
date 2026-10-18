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

namespace VfoxPlus.Design.Motion;

/// <summary>
/// The motion of the progress indicators of Material 3 Expressive: where the loading indicator is in its cycle of
/// shapes, and where the active parts of the wavy linear indicator are while it has no value (see
/// <c>docs/design-system.md#wavy-progress-indicator</c>).
/// </summary>
public static class Progress
{
    /// <summary>How long the loading indicator holds a shape before it morphs into the next one.</summary>
    public static readonly TimeSpan LoadingMorphInterval = TimeSpan.FromMilliseconds(650);

    /// <summary>How long the loading indicator takes to turn once.</summary>
    public static readonly TimeSpan LoadingRotationPeriod = TimeSpan.FromMilliseconds(4666);

    /// <summary>How big the shape of the loading indicator is, in the container it is drawn in.</summary>
    public const double LoadingActiveScale = 38.0 / 48.0;

    /// <summary>How long the indeterminate wavy indicator takes for one cycle of its two bars.</summary>
    public static readonly TimeSpan WavyCycle = TimeSpan.FromSeconds(2);

    private static readonly Spring LoadingMorphSpring = new(0.6, 200);

    /// <summary>
    /// Returns the state of the loading indicator after a time: the index of the morph, how far the morph is (it
    /// overshoots, as its spring does) and the angle the shape is turned by.
    /// </summary>
    /// <param name="elapsed">The time since the indicator started.</param>
    /// <returns>The step, the progress of the morph, and the rotation in radians.</returns>
    public static (int Step, double Progress, double Rotation) Loading(TimeSpan elapsed)
    {
        double millis = elapsed.TotalMilliseconds;
        double interval = LoadingMorphInterval.TotalMilliseconds;
        int step = (int)(millis / interval);
        double progress = LoadingMorphSpring.Position((millis - (step * interval)) / 1000);

        // A constant rotation, and a quarter turn per morph that follows the (overshooting) spring.
        double rotation = (2 * Math.PI * (millis / LoadingRotationPeriod.TotalMilliseconds)) + ((step + progress) * Math.PI / 2);
        return (step, progress, rotation);
    }

    /// <summary>
    /// Returns the easing curve of CSS and Flutter's <c>Cubic</c>: the y of the point of the Bézier curve through
    /// (0,0), (a,b), (c,d), (1,1) whose x is <paramref name="t"/>.
    /// </summary>
    /// <param name="a">The x of the first control point.</param>
    /// <param name="b">The y of the first control point.</param>
    /// <param name="c">The x of the second control point.</param>
    /// <param name="d">The y of the second control point.</param>
    /// <param name="t">The progress of the time, from 0 to 1.</param>
    /// <returns>The progress of the animation.</returns>
    public static double CubicBezier(double a, double b, double c, double d, double t)
    {
        static double Bezier(double p1, double p2, double s)
        {
            double u = 1 - s;
            return (3 * u * u * s * p1) + (3 * u * s * s * p2) + (s * s * s);
        }

        double lo = 0;
        double hi = 1;
        double s = t;
        for (int i = 0; i < 24; i++)
        {
            double x = Bezier(a, c, s);
            if (Math.Abs(x - t) < 0.001)
            {
                break;
            }

            if (x < t)
            {
                lo = s;
            }
            else
            {
                hi = s;
            }

            s = (lo + hi) / 2;
        }

        return Bezier(b, d, s);
    }

    /// <summary>
    /// Returns where the two bars of the indeterminate wavy indicator are: each bar is a tail and a head, as fractions of
    /// the width of the indicator, which follow Flutter's curves of the indeterminate linear progress indicator.
    /// </summary>
    /// <param name="cycle">The position in the cycle of <see cref="WavyCycle"/>, from 0 to 1.</param>
    /// <returns>The head and the tail of the first bar, and of the second.</returns>
    public static (double Head1, double Tail1, double Head2, double Tail2) Indeterminate(double cycle) => (
        Interval(cycle, 0, 0.375, t => CubicBezier(0.2, 0, 0.8, 1, t)),
        Interval(cycle, 0.1665, 0.5415, t => CubicBezier(0.4, 0, 1, 1, t)),
        Interval(cycle, 0.5, 0.7835, t => CubicBezier(0, 0, 0.65, 1, t)),
        Interval(cycle, 0.6335, 0.9, t => CubicBezier(0.1, 0, 0.45, 1, t)));

    /// <summary>
    /// Maps <paramref name="t"/>, from 0 to 1, to the progress of an animation that begins at <paramref name="begin"/>
    /// and ends at <paramref name="end"/>, eased by a curve.
    /// </summary>
    /// <param name="t">The progress of the time.</param>
    /// <param name="begin">When the animation begins.</param>
    /// <param name="end">When the animation ends.</param>
    /// <param name="curve">The easing curve.</param>
    /// <returns>The progress of the animation.</returns>
    public static double Interval(double t, double begin, double end, Func<double, double> curve)
    {
        double x = Math.Min(1, Math.Max(0, (t - begin) / (end - begin)));
        return x is 0 or 1 ? x : curve(x);
    }

    /// <summary>
    /// Returns the points of the wave of the active part of the wavy indicator: a sine of an amplitude and a wavelength
    /// that travels, from the start of the part to its end, one point per pixel.
    /// </summary>
    /// <param name="start">The x where the wave starts.</param>
    /// <param name="end">The x where the wave ends.</param>
    /// <param name="centerY">The y of the middle of the wave.</param>
    /// <param name="amplitude">The height of the wave.</param>
    /// <param name="wavelength">The length of one wave.</param>
    /// <param name="phase">How far the wave has travelled, in radians.</param>
    /// <returns>The points.</returns>
    public static IReadOnlyList<(double X, double Y)> Wave(double start, double end, double centerY, double amplitude, double wavelength, double phase)
    {
        List<(double X, double Y)> points = [];
        double last = Math.Max(start, end);
        for (double x = start; x <= last; x++)
        {
            points.Add((x, centerY + (amplitude * Math.Sin((2 * Math.PI * x / wavelength) - phase))));
        }

        if (points.Count == 1)
        {
            points.Add((last + 0.01, centerY));
        }

        return points;
    }
}
