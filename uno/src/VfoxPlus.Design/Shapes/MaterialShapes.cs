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
/// The shapes of the Material 3 Expressive shape library, in the unit square (<c>MaterialShapes</c> of
/// <c>androidx.compose.material3</c>). Only the ones that vfox-plus draws are here.
/// </summary>
public static class MaterialShapes
{
    private static readonly Vec2 UnitCenter = new(0.5, 0.5);
    private static readonly Rounding CornerRound15 = new(0.15);
    private static readonly Rounding CornerRound20 = new(0.2);
    private static readonly Rounding CornerRound50 = new(0.5);
    private static readonly Rounding CornerRound100 = new(1);

    /// <summary>Gets a circle.</summary>
    public static RoundedPolygon Circle { get; } = RoundedPolygon.Circle(10, 0.5, UnitCenter);

    /// <summary>Gets a tilted ellipse.</summary>
    public static RoundedPolygon Oval { get; } = RoundedPolygon.Circle(8, 1, Origin).Transformed(ScaleThenRotate(1, 0.64, -45)).Normalized();

    /// <summary>Gets a pill, with the long side horizontal.</summary>
    public static RoundedPolygon Pill { get; } = CustomPolygon([Round(0.961, 0.039, 0.426), Sharp(1.001, 0.428), Round(1, 0.609, 1)], 2, mirroring: true).Normalized();

    /// <summary>Gets a pentagon with rounded corners.</summary>
    public static RoundedPolygon Pentagon { get; } = CustomPolygon([Round(0.5, -0.009, 0.172), Round(1.03, 0.365, 0.164), Round(0.828, 0.97, 0.169)], 1, mirroring: true).Normalized();

    /// <summary>Gets a gem.</summary>
    public static RoundedPolygon Gem { get; } = CustomPolygon(
        [Smooth(0.499, 1.023, 0.241, 0.778), Round(-0.005, 0.792, 0.208), Round(0.073, 0.258, 0.228), Round(0.433, -0, 0.491)],
        1,
        mirroring: true).Normalized();

    /// <summary>Gets a rounded arch.</summary>
    public static RoundedPolygon Arch { get; } = RoundedPolygon.Regular(
        4,
        1,
        Origin,
        Rounding.Unrounded,
        [CornerRound100, CornerRound100, CornerRound20, CornerRound20]).Rotated(-135).Normalized();

    /// <summary>Gets a sun with eight rays.</summary>
    public static RoundedPolygon Sunny { get; } = Star(8, 0.8, CornerRound15, 0);

    /// <summary>Gets a cookie with four sides.</summary>
    public static RoundedPolygon Cookie4 { get; } = CustomPolygon([Round(1.237, 1.236, 0.258), Round(0.5, 0.918, 0.233)], 4, mirroring: false).Normalized();

    /// <summary>Gets a cookie with six sides.</summary>
    public static RoundedPolygon Cookie6 { get; } = CustomPolygon([Round(0.723, 0.884, 0.394), Round(0.5, 1.099, 0.398)], 6, mirroring: false).Normalized();

    /// <summary>Gets a cookie with nine sides.</summary>
    public static RoundedPolygon Cookie9 { get; } = Star(9, 0.8, CornerRound50, -90);

    /// <summary>Gets a cookie with twelve sides, the shape of the application icon.</summary>
    public static RoundedPolygon Cookie12 { get; } = Star(12, 0.8, CornerRound50, -90);

    /// <summary>Gets a clover with four leaves.</summary>
    public static RoundedPolygon Clover4 { get; } = CustomPolygon([Sharp(0.5, 0.074), Round(0.725, -0.099, 0.476)], 4, mirroring: true).Normalized();

    /// <summary>Gets a clover with eight leaves.</summary>
    public static RoundedPolygon Clover8 { get; } = CustomPolygon([Sharp(0.5, 0.036), Round(0.758, -0.101, 0.209)], 8, mirroring: false).Normalized();

    /// <summary>Gets a burst with ten rounded points.</summary>
    public static RoundedPolygon SoftBurst { get; } = CustomPolygon([Round(0.193, 0.277, 0.053), Round(0.176, 0.055, 0.053)], 10, mirroring: false).Normalized();

    /// <summary>Gets a flower with eight petals.</summary>
    public static RoundedPolygon Flower { get; } = CustomPolygon([Sharp(0.370, 0.187), Round(0.416, 0.049, 0.381), Round(0.479, 0.001, 0.095)], 8, mirroring: true).Normalized();

    /// <summary>Gets a diamond with puffy sides.</summary>
    public static RoundedPolygon PuffyDiamond { get; } = CustomPolygon([Round(0.87, 0.13, 0.146), Sharp(0.818, 0.357), Round(1, 0.332, 0.853)], 4, mirroring: true).Normalized();

    /// <summary>
    /// Gets the shapes of the avatars of plugins, in the order of the design system: a stable hash of the name picks
    /// one.
    /// </summary>
    public static IReadOnlyList<RoundedPolygon> AvatarShapes { get; } =
    [
        Cookie9, Clover4, Sunny, SoftBurst, Cookie6, Gem, Pentagon, Flower, PuffyDiamond, Cookie4, Arch, Clover8,
    ];

    private static Vec2 Origin => default;

    private static (Vec2 Point, Rounding Rounding) Sharp(double x, double y) => (new Vec2(x, y), Rounding.Unrounded);

    private static (Vec2 Point, Rounding Rounding) Round(double x, double y, double radius) => (new Vec2(x, y), new Rounding(radius));

    private static (Vec2 Point, Rounding Rounding) Smooth(double x, double y, double radius, double smoothing) => (new Vec2(x, y), new Rounding(radius, smoothing));

    // Repeats the vertices reps times around the centre of the unit square, turning them, or with mirroring turning
    // and mirroring them.
    private static RoundedPolygon CustomPolygon(IReadOnlyList<(Vec2 Point, Rounding Rounding)> points, int reps, bool mirroring)
    {
        List<(Vec2 Point, Rounding Rounding)> repeated = Repeat(points, reps, UnitCenter, mirroring);
        List<double> vertices = new(repeated.Count * 2);
        List<Rounding> perVertex = new(repeated.Count);
        foreach ((Vec2 point, Rounding rounding) in repeated)
        {
            vertices.Add(point.X);
            vertices.Add(point.Y);
            perVertex.Add(rounding);
        }

        return RoundedPolygon.FromVertices(vertices, Rounding.Unrounded, perVertex, UnitCenter);
    }

    private static List<(Vec2 Point, Rounding Rounding)> Repeat(IReadOnlyList<(Vec2 Point, Rounding Rounding)> points, int reps, Vec2 center, bool mirroring)
    {
        List<(Vec2 Point, Rounding Rounding)> result = [];
        if (!mirroring)
        {
            for (int i = 0; i < points.Count * reps; i++)
            {
                (Vec2 point, Rounding rounding) = points[i % points.Count];
                result.Add((point.Rotate((i / points.Count) * 360.0 / reps, center), rounding));
            }

            return result;
        }

        (double Angle, double Distance)[] measures = [.. points.Select(point =>
        {
            Vec2 offset = point.Point - center;
            return (offset.AngleRadians(), offset.Length());
        })];
        int actualReps = reps * 2;
        double sectionAngle = Math.PI * 2 / actualReps;
        for (int r = 0; r < actualReps; r++)
        {
            for (int index = 0; index < points.Count; index++)
            {
                int i = r % 2 != 0 ? points.Count - 1 - index : index;
                if (i > 0 || r % 2 == 0)
                {
                    double angle = sectionAngle * r;
                    angle += r % 2 == 0 ? measures[i].Angle : sectionAngle - measures[i].Angle + (2 * measures[0].Angle);
                    Vec2 point = (new Vec2(Math.Cos(angle), Math.Sin(angle)) * measures[i].Distance) + center;
                    result.Add((point, points[i].Rounding));
                }
            }
        }

        return result;
    }

    // Scales and then turns, by degrees.
    private static Func<double, double, (double X, double Y)> ScaleThenRotate(double scaleX, double scaleY, double degrees)
    {
        Func<double, double, (double X, double Y)> turn = Maths.Rotation(degrees);
        return (x, y) => turn(x * scaleX, y * scaleY);
    }

    private static RoundedPolygon Star(int vertices, double innerRadius, Rounding rounding, double degrees)
    {
        RoundedPolygon polygon = RoundedPolygon.Star(vertices, 1, innerRadius, rounding, null, Origin);
        if (degrees != 0)
        {
            polygon = polygon.Rotated(degrees);
        }

        return polygon.Normalized();
    }
}
