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

using System.Globalization;
using System.Text.Json;
using VfoxPlus.Design.Shapes;
using Xunit;

namespace VfoxPlus.Tests.Design;

public class ShapesTests
{
    // Samples the outline, n points per curve.
    private static List<Vec2> Sample(IReadOnlyList<Cubic> cubics, int n)
    {
        List<Vec2> points = [];
        foreach (Cubic cubic in cubics)
        {
            for (int i = 0; i < n; i++)
            {
                points.Add(cubic.PointOnCurve((double)i / n));
            }
        }

        return points;
    }

    // Samples an SVG path of absolute M, L, H, V, C and Z commands.
    private static List<Vec2> ParsePath(string path, int n)
    {
        string[] fields = path
            .Replace("M", " M ", StringComparison.Ordinal).Replace("L", " L ", StringComparison.Ordinal)
            .Replace("H", " H ", StringComparison.Ordinal).Replace("V", " V ", StringComparison.Ordinal)
            .Replace("C", " C ", StringComparison.Ordinal).Replace("Z", " Z ", StringComparison.Ordinal)
            .Replace(",", " ", StringComparison.Ordinal)
            .Split((char[]?)null, StringSplitOptions.RemoveEmptyEntries);
        List<Vec2> points = [];
        Vec2 current = default;
        Vec2 start = default;
        double Number(int i) => double.Parse(fields[i], CultureInfo.InvariantCulture);
        for (int i = 0; i < fields.Length;)
        {
            switch (fields[i])
            {
                case "M":
                    current = new Vec2(Number(i + 1), Number(i + 2));
                    start = current;
                    points.Add(current);
                    i += 3;
                    break;
                case "L":
                    current = new Vec2(Number(i + 1), Number(i + 2));
                    points.Add(current);
                    i += 3;
                    break;
                case "H":
                    current = new Vec2(Number(i + 1), current.Y);
                    points.Add(current);
                    i += 2;
                    break;
                case "V":
                    current = new Vec2(current.X, Number(i + 1));
                    points.Add(current);
                    i += 2;
                    break;
                case "C":
                    // Several curves may follow one command letter.
                    i++;
                    while (i + 5 < fields.Length && "MLHVCZ".All(letter => !fields[i].Contains(letter, StringComparison.Ordinal)))
                    {
                        Cubic curve = new(current.X, current.Y, Number(i), Number(i + 1), Number(i + 2), Number(i + 3), Number(i + 4), Number(i + 5));
                        for (int k = 1; k <= n; k++)
                        {
                            points.Add(curve.PointOnCurve((double)k / n));
                        }

                        current = new Vec2(Number(i + 4), Number(i + 5));
                        i += 6;
                    }

                    break;
                case "Z":
                    points.Add(start);
                    i++;
                    break;
                default:
                    throw new InvalidOperationException($"unsupported path command {fields[i]}");
            }
        }

        return points;
    }

    // Scales the points into the unit square, keeping the aspect ratio and centring them.
    private static List<Vec2> Normalize(List<Vec2> points)
    {
        double minX = points.Min(p => p.X);
        double minY = points.Min(p => p.Y);
        double maxX = points.Max(p => p.X);
        double maxY = points.Max(p => p.Y);
        double side = Math.Max(maxX - minX, maxY - minY);
        return [.. points.Select(p => new Vec2(((p.X - minX) / side) + ((1 - ((maxX - minX) / side)) / 2), ((p.Y - minY) / side) + ((1 - ((maxY - minY) / side)) / 2)))];
    }

    // The distance of a point to the closed polyline.
    private static double DistanceToPolyline(Vec2 p, List<Vec2> line)
    {
        double best = double.MaxValue;
        for (int i = 0; i < line.Count; i++)
        {
            Vec2 a = line[i];
            Vec2 b = line[(i + 1) % line.Count];
            double dx = b.X - a.X;
            double dy = b.Y - a.Y;
            double length = (dx * dx) + (dy * dy);
            double t = length > 0 ? Math.Max(0, Math.Min(1, (((p.X - a.X) * dx) + ((p.Y - a.Y) * dy)) / length)) : 0;
            best = Math.Min(best, Math.Sqrt(Math.Pow(p.X - (a.X + (t * dx)), 2) + Math.Pow(p.Y - (a.Y + (t * dy)), 2)));
        }

        return best;
    }

    // The largest distance of a point of one outline to the other.
    private static double Hausdorff(List<Vec2> a, List<Vec2> b)
    {
        static double Directed(List<Vec2> from, List<Vec2> to) => from.Max(p => DistanceToPolyline(p, to));
        return Math.Max(Directed(a, b), Directed(b, a));
    }

    public static TheoryData<string> ShapeNames() =>
        [
            "4-leaf-clover", "4-sided-cookie", "6-sided-cookie", "8-leaf-clover", "9-sided-cookie", "12-sided-cookie", "oval",
            "pentagon", "pill", "soft-burst", "sunny", "flower", "gem", "arch", "puffy-diamond", "circle",
        ];

    private static RoundedPolygon Shape(string name) => name switch
    {
        "4-leaf-clover" => MaterialShapes.Clover4,
        "4-sided-cookie" => MaterialShapes.Cookie4,
        "6-sided-cookie" => MaterialShapes.Cookie6,
        "8-leaf-clover" => MaterialShapes.Clover8,
        "9-sided-cookie" => MaterialShapes.Cookie9,
        "12-sided-cookie" => MaterialShapes.Cookie12,
        "oval" => MaterialShapes.Oval,
        "pentagon" => MaterialShapes.Pentagon,
        "pill" => MaterialShapes.Pill,
        "soft-burst" => MaterialShapes.SoftBurst,
        "sunny" => MaterialShapes.Sunny,
        "flower" => MaterialShapes.Flower,
        "gem" => MaterialShapes.Gem,
        "arch" => MaterialShapes.Arch,
        "puffy-diamond" => MaterialShapes.PuffyDiamond,
        "circle" => MaterialShapes.Circle,
        _ => throw new ArgumentOutOfRangeException(nameof(name)),
    };

    // The outlines of the Material shapes, from @m3e/web: the shapes drawn here must not be further than 2 % of
    // their size from them.
    [Theory]
    [MemberData(nameof(ShapeNames))]
    public void ShapesMatchTheMaterialOutlines(string name)
    {
        using JsonDocument paths = JsonDocument.Parse(File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "Design", "testdata", "shapes.json")));
        string reference = paths.RootElement.GetProperty(name).GetString()!;
        List<Vec2> expected = Normalize(ParsePath(reference, 40));
        List<Vec2> actual = Normalize(Sample(Shape(name).Cubics, 40));
        double distance = Hausdorff(expected, actual);
        Assert.True(distance <= 0.02, $"{name}: {distance:F4} away from the Material outline");
    }

    [Fact]
    public void OutlinesAreClosedLoopsInTheUnitSquare()
    {
        List<RoundedPolygon> all = [MaterialShapes.Circle, MaterialShapes.Oval, MaterialShapes.Pill, .. MaterialShapes.AvatarShapes];
        for (int index = 0; index < all.Count; index++)
        {
            IReadOnlyList<Cubic> cubics = all[index].Cubics;
            Assert.True(cubics.Count >= 3, $"shape {index} has {cubics.Count} curves");
            Cubic previous = cubics[^1];
            for (int i = 0; i < cubics.Count; i++)
            {
                Cubic c = cubics[i];
                Assert.True(Math.Abs(c.X0 - previous.X3) <= 1e-4 && Math.Abs(c.Y0 - previous.Y3) <= 1e-4, $"shape {index}: curve {i} does not start where the one before ends");
                previous = c;
            }

            // Normalized shapes fit the unit square.
            (double left, double top, double right, double bottom) = all[index].Bounds(approximate: false);
            Assert.True(left >= -0.02 && top >= -0.02 && right <= 1.02 && bottom <= 1.02, $"shape {index} leaves the unit square: {left} {top} {right} {bottom}");
        }
    }

    [Fact]
    public void MorphsStartAtTheFirstShapeAndEndAtTheSecond()
    {
        (RoundedPolygon, RoundedPolygon)[] pairs =
        [
            (MaterialShapes.SoftBurst, MaterialShapes.Cookie9), (MaterialShapes.Cookie9, MaterialShapes.Pentagon),
            (MaterialShapes.Pentagon, MaterialShapes.Pill), (MaterialShapes.Pill, MaterialShapes.Sunny),
            (MaterialShapes.Sunny, MaterialShapes.Cookie4), (MaterialShapes.Cookie4, MaterialShapes.Oval),
            (MaterialShapes.Oval, MaterialShapes.SoftBurst), (MaterialShapes.Circle, MaterialShapes.Flower),
        ];
        for (int index = 0; index < pairs.Length; index++)
        {
            (RoundedPolygon from, RoundedPolygon to) = pairs[index];
            Morph morph = new(from, to);
            foreach ((double progress, RoundedPolygon target) in new[] { (0.0, from), (1.0, to) })
            {
                IReadOnlyList<Cubic> outline = morph.AsCubics(progress);
                Assert.True(outline.Count >= 3, $"pair {index}: {outline.Count} curves");
                double distance = Hausdorff(Sample(outline, 32), Sample(target.Cubics, 32));
                Assert.True(distance <= 0.01, $"pair {index} at {progress}: {distance:F4} away from the shape");
            }

            // The halfway shape is a shape of its own, inside the unit square, and the outline stays closed.
            IReadOnlyList<Cubic> middle = morph.AsCubics(0.5);
            foreach (Vec2 p in Sample(middle, 8))
            {
                Assert.True(p.X >= -0.1 && p.X <= 1.1 && p.Y >= -0.1 && p.Y <= 1.1, $"pair {index}: {p} leaves the square");
            }

            Assert.True(Math.Abs(middle[^1].X3 - middle[0].X0) <= 1e-9 && Math.Abs(middle[^1].Y3 - middle[0].Y0) <= 1e-9, $"pair {index}: the outline is not closed");
        }
    }

    [Fact]
    public void AMorphThatOvershootsStaysAClosedLoop()
    {
        IReadOnlyList<Cubic> outline = new Morph(MaterialShapes.Circle, MaterialShapes.Flower).AsCubics(1.15);
        Assert.Equal(outline[0].X0, outline[^1].X3);
        Assert.Equal(outline[0].Y0, outline[^1].Y3);
    }

    [Fact]
    public void PolygonsFollowTheirConstructionRules()
    {
        Assert.Throws<ArgumentException>(() => RoundedPolygon.FromVertices([0, 0, 1, 1], Rounding.Unrounded));
        Assert.Throws<ArgumentException>(() => RoundedPolygon.FromVertices([0, 0, 1, 1, 2, 0], Rounding.Unrounded, [Rounding.Unrounded]));
        Assert.Throws<ArgumentOutOfRangeException>(() => RoundedPolygon.Regular(2, 1, default, Rounding.Unrounded));
        Assert.Throws<ArgumentOutOfRangeException>(() => RoundedPolygon.Circle(2, 1, default));
        Assert.Throws<ArgumentOutOfRangeException>(() => RoundedPolygon.Star(5, 1, 1, Rounding.Unrounded, null, default));
    }

    [Fact]
    public void RectanglesAndStarsHaveTheCurvesOfTheirCorners()
    {
        RoundedPolygon square = RoundedPolygon.Rectangle(2, 2, new Rounding(0.5), null, default);
        Assert.True(square.Cubics.Count >= 8, $"a rounded square has curves for its corners, got {square.Cubics.Count}");
        RoundedPolygon star = RoundedPolygon.Star(5, 1, 0.5, Rounding.Unrounded, null, default);
        Assert.Equal(10, star.Cubics.Count);
        RoundedPolygon rotated = star.Rotated(90);
        Assert.True(Math.Abs(rotated.Center.X) <= 1e-9 && Math.Abs(rotated.Center.Y) <= 1e-9, $"{rotated.Center}");

        // Corners with an inner rounding of their own, and the vertices of a star around a centre.
        RoundedPolygon rounded = RoundedPolygon.Star(6, 1, 0.6, new Rounding(0.1), new Rounding(0.05, 0.5), new Vec2(2, 3));
        Assert.Equal(new Vec2(2, 3), rounded.Center);
        Assert.True(rounded.Cubics.Count > 12);
    }

    [Fact]
    public void ShapesAreMeasuredAndFittedToTheirCircle()
    {
        RoundedPolygon fitted = MaterialShapes.Cookie9.FitToCircle();
        (double left, double top, double right, double bottom) = fitted.MaxBounds();
        Assert.InRange(right - left, 0.99, 1.01);
        Assert.InRange(bottom - top, 0.99, 1.01);
        (double l, double t, double r, double b) = MaterialShapes.Gem.Bounds(approximate: true);
        (double l2, double t2, double r2, double b2) = MaterialShapes.Gem.Bounds(approximate: false);
        Assert.True(l <= l2 + 1e-9 && t <= t2 + 1e-9 && r >= r2 - 1e-9 && b >= b2 - 1e-9, "the approximation is never too small");
        Assert.Equal(12, MaterialShapes.AvatarShapes.Count);
    }

    [Fact]
    public void VectorsDoTheirMaths()
    {
        Vec2 a = new(3, 4);
        Assert.Equal(5, a.Length());
        Assert.Equal(25, a.LengthSquared());
        Assert.Equal(new Vec2(0.6, 0.8), a.Direction());
        Assert.Equal(new Vec2(-4, 3), a.Rotate90());
        Assert.Equal(new Vec2(4, 6), a + new Vec2(1, 2));
        Assert.Equal(new Vec2(2, 2), a - new Vec2(1, 2));
        Assert.Equal(new Vec2(6, 8), a * 2);
        Assert.Equal(new Vec2(1.5, 2), a / 2);
        Assert.Equal(11, a.Dot(new Vec2(1, 2)));
        Assert.True(new Vec2(1, 0).IsClockwise(new Vec2(0, 1)));
        Assert.False(new Vec2(0, 1).IsClockwise(new Vec2(1, 0)));
        Vec2 turned = new Vec2(1, 0).Rotate(90, default);
        Assert.True(Math.Abs(turned.X) < 1e-12 && Math.Abs(turned.Y - 1) < 1e-12);
        Assert.Equal(Math.PI / 2, new Vec2(0, 1).AngleRadians());
        Assert.Equal("Vec2 { X = 3, Y = 4 }", a.ToString());
    }
}
