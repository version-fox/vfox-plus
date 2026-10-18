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
/// A point, or a vector, in the plane.
/// </summary>
/// <param name="X">The horizontal coordinate.</param>
/// <param name="Y">The vertical coordinate, which grows downwards.</param>
public readonly record struct Vec2(double X, double Y)
{
    // Computed values are methods: the compiler prints the properties of a record, and a property that is a Vec2
    // would print itself for ever.

    /// <summary>Returns the length of the vector.</summary>
    /// <returns>The length.</returns>
    public double Length() => Math.Sqrt((X * X) + (Y * Y));

    /// <summary>Returns the squared length of the vector.</summary>
    /// <returns>The squared length.</returns>
    public double LengthSquared() => (X * X) + (Y * Y);

    /// <summary>Returns the unit vector in the direction of this one.</summary>
    /// <returns>The direction.</returns>
    public Vec2 Direction() => this / Length();

    /// <summary>Returns the angle of the vector from the X axis, in radians.</summary>
    /// <returns>The angle.</returns>
    public double AngleRadians() => Math.Atan2(Y, X);

    /// <summary>Returns this vector turned by 90 degrees.</summary>
    /// <returns>The turned vector.</returns>
    public Vec2 Rotate90() => new(-Y, X);

    /// <summary>Adds two vectors.</summary>
    /// <param name="a">The first.</param>
    /// <param name="b">The second.</param>
    /// <returns>The sum.</returns>
    public static Vec2 operator +(Vec2 a, Vec2 b) => new(a.X + b.X, a.Y + b.Y);

    /// <summary>Subtracts a vector from another.</summary>
    /// <param name="a">The first.</param>
    /// <param name="b">The second.</param>
    /// <returns>The difference.</returns>
    public static Vec2 operator -(Vec2 a, Vec2 b) => new(a.X - b.X, a.Y - b.Y);

    /// <summary>Scales a vector.</summary>
    /// <param name="a">The vector.</param>
    /// <param name="factor">The factor.</param>
    /// <returns>The scaled vector.</returns>
    public static Vec2 operator *(Vec2 a, double factor) => new(a.X * factor, a.Y * factor);

    /// <summary>Divides a vector.</summary>
    /// <param name="a">The vector.</param>
    /// <param name="divisor">The divisor.</param>
    /// <returns>The divided vector.</returns>
    public static Vec2 operator /(Vec2 a, double divisor) => new(a.X / divisor, a.Y / divisor);

    /// <summary>Returns the dot product with another vector.</summary>
    /// <param name="other">The other vector.</param>
    /// <returns>The product.</returns>
    public double Dot(Vec2 other) => (X * other.X) + (Y * other.Y);

    /// <summary>Determines whether another vector is clockwise of this one.</summary>
    /// <param name="other">The other vector.</param>
    /// <returns><see langword="true"/> if it is.</returns>
    public bool IsClockwise(Vec2 other) => (X * other.Y) - (Y * other.X) > 0;

    /// <summary>Turns the point around a centre.</summary>
    /// <param name="degrees">The angle.</param>
    /// <param name="center">The centre.</param>
    /// <returns>The turned point.</returns>
    public Vec2 Rotate(double degrees, Vec2 center)
    {
        double radians = degrees * Math.PI / 180;
        Vec2 offset = this - center;
        double cos = Math.Cos(radians);
        double sin = Math.Sin(radians);
        return new Vec2((offset.X * cos) - (offset.Y * sin), (offset.X * sin) + (offset.Y * cos)) + center;
    }
}

/// <summary>Tolerances and helpers of the shape maths, the same as those of the library this is ported from.</summary>
internal static class Maths
{
    internal const double DistanceEpsilon = 1e-5;
    internal const double AngleEpsilon = 1e-6;

    internal static double Distance(double x, double y) => Math.Sqrt((x * x) + (y * y));

    internal static double DistanceSquared(double x, double y) => (x * x) + (y * y);

    /// <summary>Returns the unit vector in the direction of (x, y).</summary>
    internal static Vec2 DirectionVector(double x, double y)
    {
        double d = Distance(x, y);
        return new Vec2(x / d, y / d);
    }

    /// <summary>Returns the point at a radius and an angle (radians) from a centre.</summary>
    internal static Vec2 RadialToCartesian(double radius, double angleRadians, Vec2 center) =>
        (new Vec2(Math.Cos(angleRadians), Math.Sin(angleRadians)) * radius) + center;

    internal static double Square(double x) => x * x;

    internal static double Lerp(double start, double stop, double fraction) => (start * (1 - fraction)) + (stop * fraction);

    internal static Vec2 Interpolate(Vec2 start, Vec2 stop, double fraction) =>
        new(Lerp(start.X, stop.X, fraction), Lerp(start.Y, stop.Y, fraction));

    /// <summary>Returns a number modulo 1, never negative.</summary>
    internal static double PositiveModulo(double number) => (((number % 1) + 1) % 1);

    /// <summary>Determines whether the corner at <paramref name="current"/>, between two points, is convex.</summary>
    internal static bool IsConvex(Vec2 previous, Vec2 current, Vec2 next) => (current - previous).IsClockwise(next - current);

    internal static Func<double, double, (double X, double Y)> Rotation(double degrees)
    {
        double radians = degrees * Math.PI / 180;
        double cos = Math.Cos(radians);
        double sin = Math.Sin(radians);
        return (x, y) => ((x * cos) - (y * sin), (x * sin) + (y * cos));
    }
}
