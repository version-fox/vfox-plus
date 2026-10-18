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

namespace VfoxPlus.Design.Shapes;

/// <summary>
/// A cubic Bézier curve: the first anchor, the two control points and the second anchor.
/// </summary>
/// <param name="X0">The horizontal coordinate of the start.</param>
/// <param name="Y0">The vertical coordinate of the start.</param>
/// <param name="X1">The horizontal coordinate of the control point of the start.</param>
/// <param name="Y1">The vertical coordinate of the control point of the start.</param>
/// <param name="X2">The horizontal coordinate of the control point of the end.</param>
/// <param name="Y2">The vertical coordinate of the control point of the end.</param>
/// <param name="X3">The horizontal coordinate of the end.</param>
/// <param name="Y3">The vertical coordinate of the end.</param>
public readonly record struct Cubic(double X0, double Y0, double X1, double Y1, double X2, double Y2, double X3, double Y3)
{
    /// <summary>Returns the start of the curve.</summary>
    /// <returns>The point.</returns>
    public Vec2 Anchor0() => new(X0, Y0);

    /// <summary>Returns the control point of the start.</summary>
    /// <returns>The point.</returns>
    public Vec2 Control0() => new(X1, Y1);

    /// <summary>Returns the control point of the end.</summary>
    /// <returns>The point.</returns>
    public Vec2 Control1() => new(X2, Y2);

    /// <summary>Returns the end of the curve.</summary>
    /// <returns>The point.</returns>
    public Vec2 Anchor1() => new(X3, Y3);

    /// <summary>Creates a curve from its points.</summary>
    /// <param name="anchor0">The start.</param>
    /// <param name="control0">The control point of the start.</param>
    /// <param name="control1">The control point of the end.</param>
    /// <param name="anchor1">The end.</param>
    /// <returns>The curve.</returns>
    public static Cubic From(Vec2 anchor0, Vec2 control0, Vec2 control1, Vec2 anchor1) =>
        new(anchor0.X, anchor0.Y, control0.X, control0.Y, control1.X, control1.Y, anchor1.X, anchor1.Y);

    /// <summary>Returns the point of the curve at <paramref name="t"/>, from 0 to 1.</summary>
    /// <param name="t">The parameter.</param>
    /// <returns>The point.</returns>
    public Vec2 PointOnCurve(double t)
    {
        double u = 1 - t;
        return new Vec2(
            (X0 * (u * u * u)) + (X1 * (3 * t * u * u)) + (X2 * (3 * t * t * u)) + (X3 * (t * t * t)),
            (Y0 * (u * u * u)) + (Y1 * (3 * t * u * u)) + (Y2 * (3 * t * t * u)) + (Y3 * (t * t * t)));
    }

    /// <summary>Applies a function to the four points of the curve.</summary>
    /// <param name="transform">The function.</param>
    /// <returns>The transformed curve.</returns>
    public Cubic Transformed(Func<double, double, (double X, double Y)> transform)
    {
        (double x0, double y0) = transform(X0, Y0);
        (double x1, double y1) = transform(X1, Y1);
        (double x2, double y2) = transform(X2, Y2);
        (double x3, double y3) = transform(X3, Y3);
        return new Cubic(x0, y0, x1, y1, x2, y2, x3, y3);
    }

    /// <summary>Returns the curve between two curves, at a progress from 0 to 1.</summary>
    internal static Cubic Interpolate(Cubic a, Cubic b, double progress) => new(
        Maths.Lerp(a.X0, b.X0, progress), Maths.Lerp(a.Y0, b.Y0, progress),
        Maths.Lerp(a.X1, b.X1, progress), Maths.Lerp(a.Y1, b.Y1, progress),
        Maths.Lerp(a.X2, b.X2, progress), Maths.Lerp(a.Y2, b.Y2, progress),
        Maths.Lerp(a.X3, b.X3, progress), Maths.Lerp(a.Y3, b.Y3, progress));

    /// <summary>Returns the curve that draws the line from one point to another.</summary>
    internal static Cubic StraightLine(double x0, double y0, double x1, double y1) => new(
        x0, y0,
        Maths.Lerp(x0, x1, 1.0 / 3), Maths.Lerp(y0, y1, 1.0 / 3),
        Maths.Lerp(x0, x1, 2.0 / 3), Maths.Lerp(y0, y1, 2.0 / 3),
        x1, y1);

    /// <summary>Returns the curve that approximates the arc of the circle around a centre from one point to another.</summary>
    internal static Cubic CircularArc(double centerX, double centerY, double x0, double y0, double x1, double y1)
    {
        Vec2 p0d = Maths.DirectionVector(x0 - centerX, y0 - centerY);
        Vec2 p1d = Maths.DirectionVector(x1 - centerX, y1 - centerY);
        Vec2 rotatedP0 = p0d.Rotate90();
        Vec2 rotatedP1 = p1d.Rotate90();
        bool clockwise = rotatedP0.Dot(new Vec2(x1 - centerX, y1 - centerY)) >= 0;
        double cosa = p0d.Dot(p1d);
        if (cosa > 0.999)
        {
            // The arc is so short that a line is as good.
            return StraightLine(x0, y0, x1, y1);
        }

        double direction = clockwise ? 1 : -1;
        double k = Maths.Distance(x0 - centerX, y0 - centerY) * 4 / 3
            * (Math.Sqrt(2 * (1 - cosa)) - Math.Sqrt(1 - (cosa * cosa))) / (1 - cosa) * direction;
        return new Cubic(
            x0, y0,
            x0 + (rotatedP0.X * k), y0 + (rotatedP0.Y * k),
            x1 - (rotatedP1.X * k), y1 - (rotatedP1.Y * k),
            x1, y1);
    }

    /// <summary>Determines whether the curve starts and ends at the same point.</summary>
    /// <returns><see langword="true"/> if it does.</returns>
    internal bool ZeroLength() =>
        Math.Abs(X0 - X3) < Maths.DistanceEpsilon && Math.Abs(Y0 - Y3) < Maths.DistanceEpsilon;

    /// <summary>
    /// Returns the bounding box (left, top, right, bottom) of the curve. With <paramref name="approximate"/>, the
    /// control points count as part of it, which is faster and never too small.
    /// </summary>
    internal (double Left, double Top, double Right, double Bottom) CalculateBounds(bool approximate)
    {
        if (ZeroLength())
        {
            return (X0, Y0, X0, Y0);
        }

        double minX = Math.Min(X0, X3);
        double minY = Math.Min(Y0, Y3);
        double maxX = Math.Max(X0, X3);
        double maxY = Math.Max(Y0, Y3);
        if (approximate)
        {
            return (
                Math.Min(minX, Math.Min(X1, X2)), Math.Min(minY, Math.Min(Y1, Y2)),
                Math.Max(maxX, Math.Max(X1, X2)), Math.Max(maxY, Math.Max(Y1, Y2)));
        }

        // The extrema are where the derivative is zero, a quadratic per axis.
        Cubic self = this;
        void Extend(double a0, double c0, double c1, double a1, bool horizontal, ref double lo, ref double hi)
        {
            double a = -a0 + (3 * c0) - (3 * c1) + a1;
            double b = (2 * a0) - (4 * c0) + (2 * c1);
            double cc = -a0 + c0;
            double low = lo;
            double high = hi;
            void Consider(double t)
            {
                if (t is >= 0 and <= 1)
                {
                    Vec2 point = self.PointOnCurve(t);
                    double value = horizontal ? point.X : point.Y;
                    low = Math.Min(low, value);
                    high = Math.Max(high, value);
                }
            }

            if (Math.Abs(a) < Maths.DistanceEpsilon)
            {
                if (b != 0)
                {
                    Consider(2 * cc / (-2 * b));
                }
            }
            else
            {
                double discriminant = (b * b) - (4 * a * cc);
                if (discriminant >= 0)
                {
                    double root = Math.Sqrt(discriminant);
                    Consider((-b + root) / (2 * a));
                    Consider((-b - root) / (2 * a));
                }
            }

            lo = low;
            hi = high;
        }

        Extend(X0, X1, X2, X3, horizontal: true, ref minX, ref maxX);
        Extend(Y0, Y1, Y2, Y3, horizontal: false, ref minY, ref maxY);
        return (minX, minY, maxX, maxY);
    }

    /// <summary>Cuts the curve at <paramref name="t"/> into two curves that draw the same.</summary>
    internal (Cubic First, Cubic Second) Split(double t)
    {
        double u = 1 - t;
        Vec2 point = PointOnCurve(t);
        return (
            new Cubic(
                X0, Y0,
                (X0 * u) + (X1 * t), (Y0 * u) + (Y1 * t),
                (X0 * (u * u)) + (X1 * (2 * u * t)) + (X2 * (t * t)), (Y0 * (u * u)) + (Y1 * (2 * u * t)) + (Y2 * (t * t)),
                point.X, point.Y),
            new Cubic(
                point.X, point.Y,
                (X1 * (u * u)) + (X2 * (2 * u * t)) + (X3 * (t * t)), (Y1 * (u * u)) + (Y2 * (2 * u * t)) + (Y3 * (t * t)),
                (X2 * u) + (X3 * t), (Y2 * u) + (Y3 * t),
                X3, Y3));
    }

    /// <summary>Returns the curve drawn backwards.</summary>
    internal Cubic Reverse() => new(X3, Y3, X2, Y2, X1, Y1, X0, Y0);
}
