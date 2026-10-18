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
/// Says how a corner of a polygon is rounded.
/// </summary>
/// <param name="Radius">The radius of the circle that rounds the corner, in the units of the polygon.</param>
/// <param name="Smoothing">
/// From 0 to 1: spreads the curvature over a longer stretch of the edges, from a circular arc to a smooth
/// transition.
/// </param>
public readonly record struct Rounding(double Radius, double Smoothing = 0)
{
    /// <summary>Gets the rounding that leaves a corner sharp.</summary>
    public static Rounding Unrounded => default;
}

/// <summary>What a feature of a polygon outline is.</summary>
internal enum FeatureKind
{
    Edge,
    ConvexCorner,
    ConcaveCorner,
}

/// <summary>
/// A part of the outline of a polygon: an edge, which morphing ignores, or a corner, which morphing matches to a
/// corner of the other shape.
/// </summary>
internal sealed record Feature(IReadOnlyList<Cubic> Cubics, FeatureKind Kind)
{
    internal bool IsCorner => Kind != FeatureKind.Edge;

    internal bool IsConvex => Kind == FeatureKind.ConvexCorner;

    internal Feature Transformed(Func<double, double, (double X, double Y)> transform) =>
        new([.. Cubics.Select(cubic => cubic.Transformed(transform))], Kind);
}

/// <summary>
/// A polygon with rounded corners, drawn as a closed outline of cubic Bézier curves, as in the Android library
/// <c>androidx.graphics.shapes</c> (see <c>NOTICE.txt</c> for where this is ported from).
/// </summary>
public sealed class RoundedPolygon
{
    private RoundedPolygon(IReadOnlyList<Feature> features, Vec2 center)
    {
        Features = features;
        Center = center;
        Cubics = BuildOutline(features, center);
    }

    /// <summary>Gets the outline, a closed loop of curves.</summary>
    public IReadOnlyList<Cubic> Cubics { get; }

    /// <summary>Gets the centre of the polygon, from which the shape is measured.</summary>
    public Vec2 Center { get; }

    internal IReadOnlyList<Feature> Features { get; }

    /// <summary>
    /// Creates the polygon with the given vertices, all rounded with <paramref name="rounding"/>, or each with its own
    /// rounding of <paramref name="perVertex"/> when that is given. The centre is the average of the vertices unless
    /// given.
    /// </summary>
    /// <param name="vertices">The coordinates of the vertices, x and y one after the other.</param>
    /// <param name="rounding">The rounding of every vertex.</param>
    /// <param name="perVertex">The rounding of each vertex, or <see langword="null"/>.</param>
    /// <param name="center">The centre, or <see langword="null"/> for the average of the vertices.</param>
    /// <returns>The polygon.</returns>
    /// <exception cref="ArgumentException">There are fewer than three vertices, or a rounding is missing.</exception>
    public static RoundedPolygon FromVertices(IReadOnlyList<double> vertices, Rounding rounding, IReadOnlyList<Rounding>? perVertex = null, Vec2? center = null)
    {
        if (vertices.Count < 6 || vertices.Count % 2 != 0)
        {
            throw new ArgumentException("A polygon needs at least three vertices with two coordinates each.", nameof(vertices));
        }

        if (perVertex is not null && perVertex.Count * 2 != vertices.Count)
        {
            throw new ArgumentException("perVertex needs one rounding per vertex.", nameof(perVertex));
        }

        int n = vertices.Count / 2;
        Vec2 Vertex(int i) => new(vertices[((i % n) * 2)], vertices[((i % n) * 2) + 1]);

        RoundedCorner[] corners = new RoundedCorner[n];
        for (int i = 0; i < n; i++)
        {
            corners[i] = new RoundedCorner(Vertex(i + n - 1), Vertex(i), Vertex(i + 1), perVertex?[i] ?? rounding);
        }

        // Rounding two corners of a side may cut more than the side is long: the cuts of the corners are reduced
        // then, first the smoothing and then the rounding itself.
        (double RoundCutRatio, double CutRatio)[] cutAdjusts = new (double, double)[n];
        for (int i = 0; i < n; i++)
        {
            double expectedRoundCut = corners[i].ExpectedRoundCut + corners[(i + 1) % n].ExpectedRoundCut;
            double expectedCut = corners[i].ExpectedCut + corners[(i + 1) % n].ExpectedCut;
            Vec2 next = Vertex(i + 1);
            double sideSize = Maths.Distance(Vertex(i).X - next.X, Vertex(i).Y - next.Y);
            if (expectedRoundCut > sideSize)
            {
                cutAdjusts[i] = (sideSize / expectedRoundCut, 0);
            }
            else if (expectedCut > sideSize)
            {
                cutAdjusts[i] = (1, (sideSize - expectedRoundCut) / (expectedCut - expectedRoundCut));
            }
            else
            {
                cutAdjusts[i] = (1, 1);
            }
        }

        List<Cubic>[] cornerCubics = new List<Cubic>[n];
        for (int i = 0; i < n; i++)
        {
            double[] allowedCuts = new double[2];
            for (int delta = 0; delta <= 1; delta++)
            {
                (double roundCutRatio, double cutRatio) = cutAdjusts[(i + n - 1 + delta) % n];
                allowedCuts[delta] = (corners[i].ExpectedRoundCut * roundCutRatio)
                    + ((corners[i].ExpectedCut - corners[i].ExpectedRoundCut) * cutRatio);
            }

            cornerCubics[i] = corners[i].Cubics(allowedCuts[0], allowedCuts[1]);
        }

        List<Feature> features = new(n * 2);
        for (int i = 0; i < n; i++)
        {
            FeatureKind kind = Maths.IsConvex(Vertex(i + n - 1), Vertex(i), Vertex(i + 1)) ? FeatureKind.ConvexCorner : FeatureKind.ConcaveCorner;
            Cubic last = cornerCubics[i][^1];
            Cubic nextFirst = cornerCubics[(i + 1) % n][0];
            features.Add(new Feature(cornerCubics[i], kind));
            features.Add(new Feature([Cubic.StraightLine(last.X3, last.Y3, nextFirst.X0, nextFirst.Y0)], FeatureKind.Edge));
        }

        return new RoundedPolygon(features, center ?? AverageOf(vertices));
    }

    /// <summary>
    /// Creates the polygon with <paramref name="numVertices"/> vertices on a circle of <paramref name="radius"/> around
    /// <paramref name="center"/>.
    /// </summary>
    /// <param name="numVertices">The number of vertices, at least 3.</param>
    /// <param name="radius">The radius of the circle the vertices are on.</param>
    /// <param name="center">The centre.</param>
    /// <param name="rounding">The rounding of every vertex.</param>
    /// <param name="perVertex">The rounding of each vertex, or <see langword="null"/>.</param>
    /// <returns>The polygon.</returns>
    public static RoundedPolygon Regular(int numVertices, double radius, Vec2 center, Rounding rounding, IReadOnlyList<Rounding>? perVertex = null)
    {
        if (numVertices < 3)
        {
            throw new ArgumentOutOfRangeException(nameof(numVertices), "A polygon needs at least three vertices.");
        }

        List<double> vertices = new(numVertices * 2);
        for (int i = 0; i < numVertices; i++)
        {
            Vec2 vertex = Maths.RadialToCartesian(radius, Math.PI / numVertices * 2 * i, center);
            vertices.Add(vertex.X);
            vertices.Add(vertex.Y);
        }

        return FromVertices(vertices, rounding, perVertex, center);
    }

    /// <summary>Creates a polygon that approximates a circle with <paramref name="numVertices"/> rounded vertices.</summary>
    /// <param name="numVertices">The number of vertices, at least 3.</param>
    /// <param name="radius">The radius.</param>
    /// <param name="center">The centre.</param>
    /// <returns>The polygon.</returns>
    public static RoundedPolygon Circle(int numVertices, double radius, Vec2 center)
    {
        if (numVertices < 3)
        {
            throw new ArgumentOutOfRangeException(nameof(numVertices), "A circle needs at least three vertices.");
        }

        // Rounding the corners of the polygon with the radius of the circle makes the polygon the circle, once its
        // vertices lie on a larger circle.
        double theta = Math.PI / numVertices;
        double polygonRadius = radius / Math.Cos(theta);
        return Regular(numVertices, polygonRadius, center, new Rounding(radius));
    }

    /// <summary>
    /// Creates the rectangle of a width and a height around a centre. The vertices are the bottom right, bottom left,
    /// top left and top right ones.
    /// </summary>
    /// <param name="width">The width.</param>
    /// <param name="height">The height.</param>
    /// <param name="rounding">The rounding of every vertex.</param>
    /// <param name="perVertex">The rounding of each vertex, or <see langword="null"/>.</param>
    /// <param name="center">The centre.</param>
    /// <returns>The polygon.</returns>
    public static RoundedPolygon Rectangle(double width, double height, Rounding rounding, IReadOnlyList<Rounding>? perVertex, Vec2 center)
    {
        double left = center.X - (width / 2);
        double top = center.Y - (height / 2);
        double right = center.X + (width / 2);
        double bottom = center.Y + (height / 2);
        return FromVertices([right, bottom, left, bottom, left, top, right, top], rounding, perVertex, center);
    }

    /// <summary>
    /// Creates the polygon of a star with <paramref name="numVerticesPerRadius"/> points on a circle of
    /// <paramref name="radius"/>, between which the vertices of <paramref name="innerRadius"/> lie.
    /// </summary>
    /// <param name="numVerticesPerRadius">The number of points.</param>
    /// <param name="radius">The radius of the points.</param>
    /// <param name="innerRadius">The radius of the inner vertices.</param>
    /// <param name="rounding">The rounding of the points.</param>
    /// <param name="innerRounding">The rounding of the inner vertices, or <see langword="null"/> for the same.</param>
    /// <param name="center">The centre.</param>
    /// <returns>The polygon.</returns>
    public static RoundedPolygon Star(int numVerticesPerRadius, double radius, double innerRadius, Rounding rounding, Rounding? innerRounding, Vec2 center)
    {
        if (radius <= 0 || innerRadius <= 0 || innerRadius >= radius)
        {
            throw new ArgumentOutOfRangeException(nameof(innerRadius), "A star needs 0 < innerRadius < radius.");
        }

        List<Rounding>? perVertex = null;
        if (innerRounding is { } inner)
        {
            perVertex = [];
            for (int i = 0; i < numVerticesPerRadius; i++)
            {
                perVertex.Add(rounding);
                perVertex.Add(inner);
            }
        }

        List<double> vertices = new(numVerticesPerRadius * 4);
        for (int i = 0; i < numVerticesPerRadius; i++)
        {
            Vec2 outer = Maths.RadialToCartesian(radius, Math.PI / numVerticesPerRadius * 2 * i, default);
            Vec2 innerPoint = Maths.RadialToCartesian(innerRadius, Math.PI / numVerticesPerRadius * ((2 * i) + 1), default);
            vertices.AddRange([outer.X + center.X, outer.Y + center.Y, innerPoint.X + center.X, innerPoint.Y + center.Y]);
        }

        return FromVertices(vertices, rounding, perVertex, center);
    }

    /// <summary>Applies a function to every point of the polygon.</summary>
    /// <param name="transform">The function.</param>
    /// <returns>The transformed polygon.</returns>
    public RoundedPolygon Transformed(Func<double, double, (double X, double Y)> transform)
    {
        (double x, double y) = transform(Center.X, Center.Y);
        return new RoundedPolygon([.. Features.Select(feature => feature.Transformed(transform))], new Vec2(x, y));
    }

    /// <summary>Turns the polygon around the origin.</summary>
    /// <param name="degrees">The angle.</param>
    /// <returns>The turned polygon.</returns>
    public RoundedPolygon Rotated(double degrees) => Transformed(Maths.Rotation(degrees));

    /// <summary>
    /// Returns the bounding box of the outline. The approximation includes the control points and is never too small.
    /// </summary>
    /// <param name="approximate">Whether to approximate.</param>
    /// <returns>The left, top, right and bottom.</returns>
    public (double Left, double Top, double Right, double Bottom) Bounds(bool approximate)
    {
        double minX = double.MaxValue;
        double minY = double.MaxValue;
        double maxX = -double.MaxValue;
        double maxY = -double.MaxValue;
        foreach (Cubic cubic in Cubics)
        {
            (double left, double top, double right, double bottom) = cubic.CalculateBounds(approximate);
            minX = Math.Min(minX, left);
            minY = Math.Min(minY, top);
            maxX = Math.Max(maxX, right);
            maxY = Math.Max(maxY, bottom);
        }

        return (minX, minY, maxX, maxY);
    }

    /// <summary>
    /// Returns the bounding box of the circle around the centre that reaches the point of the outline farthest from
    /// it: the polygon stays inside it whatever it is turned by.
    /// </summary>
    /// <returns>The left, top, right and bottom.</returns>
    public (double Left, double Top, double Right, double Bottom) MaxBounds()
    {
        double maxDistance = 0;
        foreach (Cubic cubic in Cubics)
        {
            double anchor = Maths.DistanceSquared(cubic.X0 - Center.X, cubic.Y0 - Center.Y);
            Vec2 middle = cubic.PointOnCurve(0.5);
            maxDistance = Math.Max(maxDistance, Math.Max(anchor, Maths.DistanceSquared(middle.X - Center.X, middle.Y - Center.Y)));
        }

        double radius = Math.Sqrt(maxDistance);
        return (Center.X - radius, Center.Y - radius, Center.X + radius, Center.Y + radius);
    }

    /// <summary>
    /// Scales the polygon around its centre so that the circle that holds it fills the unit square, which keeps it
    /// inside its bounds while it spins.
    /// </summary>
    /// <returns>The scaled polygon.</returns>
    public RoundedPolygon FitToCircle()
    {
        (double left, _, double right, _) = MaxBounds();
        double scale = 0.5 / ((right - left) / 2);
        double cx = Center.X;
        double cy = Center.Y;
        return Transformed((x, y) => (((x - cx) * scale) + 0.5, ((y - cy) * scale) + 0.5));
    }

    /// <summary>Scales and moves the polygon into the unit square, keeping its aspect ratio and centring it.</summary>
    /// <returns>The normalized polygon.</returns>
    public RoundedPolygon Normalized()
    {
        (double left, double top, double right, double bottom) = Bounds(approximate: true);
        double width = right - left;
        double height = bottom - top;
        double side = Math.Max(width, height);
        double offsetX = ((side - width) / 2) - left;
        double offsetY = ((side - height) / 2) - top;
        return Transformed((x, y) => ((x + offsetX) / side, (y + offsetY) / side));
    }

    // The average of the vertices.
    private static Vec2 AverageOf(IReadOnlyList<double> vertices)
    {
        double x = 0;
        double y = 0;
        for (int i = 0; i < vertices.Count; i += 2)
        {
            x += vertices[i];
            y += vertices[i + 1];
        }

        double count = vertices.Count / 2.0;
        return new Vec2(x / count, y / count);
    }

    // Joins the cubics of the features into the outline, leaving out the ones without length. A corner of three
    // cubics at the start is cut in the middle of its arc, so that the outline starts and ends in the same corner.
    private static List<Cubic> BuildOutline(IReadOnlyList<Feature> features, Vec2 center)
    {
        List<Cubic> outline = [];
        Cubic? firstCubic = null;
        Cubic? lastCubic = null;
        List<Cubic>? firstSplitStart = null;
        List<Cubic>? firstSplitEnd = null;
        if (features.Count > 0 && features[0].Cubics.Count == 3)
        {
            (Cubic start, Cubic end) = features[0].Cubics[1].Split(0.5);
            firstSplitStart = [features[0].Cubics[0], start];
            firstSplitEnd = [end, features[0].Cubics[2]];
        }

        for (int i = 0; i <= features.Count; i++)
        {
            IReadOnlyList<Cubic> featureCubics;
            if (i == 0 && firstSplitEnd is not null)
            {
                featureCubics = firstSplitEnd;
            }
            else if (i == features.Count)
            {
                if (firstSplitStart is null)
                {
                    continue;
                }

                featureCubics = firstSplitStart;
            }
            else
            {
                featureCubics = features[i].Cubics;
            }

            foreach (Cubic cubic in featureCubics)
            {
                if (!cubic.ZeroLength())
                {
                    if (lastCubic is { } previous)
                    {
                        outline.Add(previous);
                    }

                    lastCubic = cubic;
                    firstCubic ??= cubic;
                }
                else if (lastCubic is { } last)
                {
                    // Move the end of the last curve to where the empty one ends.
                    lastCubic = last with { X3 = cubic.X3, Y3 = cubic.Y3 };
                }
            }
        }

        if (lastCubic is { } finalCubic && firstCubic is { } first)
        {
            outline.Add(finalCubic with { X3 = first.X0, Y3 = first.Y0 });
            return outline;
        }

        outline.Add(new Cubic(center.X, center.Y, center.X, center.Y, center.X, center.Y, center.X, center.Y));
        return outline;
    }

    /// <summary>Computes the curves that round one corner of a polygon.</summary>
    private sealed class RoundedCorner
    {
        private readonly Vec2 p0;
        private readonly Vec2 p1;
        private readonly Vec2 p2;
        private readonly Vec2 d1;
        private readonly Vec2 d2;
        private readonly double cornerRadius;
        private readonly double smoothing;

        internal RoundedCorner(Vec2 p0, Vec2 p1, Vec2 p2, Rounding rounding)
        {
            this.p0 = p0;
            this.p1 = p1;
            this.p2 = p2;
            Vec2 v01 = p0 - p1;
            Vec2 v21 = p2 - p1;
            double d01 = v01.Length();
            double d21 = v21.Length();
            if (d01 > 0 && d21 > 0)
            {
                d1 = v01 / d01;
                d2 = v21 / d21;
                cornerRadius = rounding.Radius;
                smoothing = rounding.Smoothing;

                // The corner of the two edges: how far from the vertex the round part starts.
                double cosAngle = d1.Dot(d2);
                double sinAngle = Math.Sqrt(1 - Maths.Square(cosAngle));
                if (sinAngle > 1e-3)
                {
                    ExpectedRoundCut = cornerRadius * (cosAngle + 1) / sinAngle;
                }
            }
        }

        /// <summary>Gets how far from the vertex the round part starts.</summary>
        internal double ExpectedRoundCut { get; }

        /// <summary>Gets how far the rounding, with its smoothing, reaches along the edges, if nothing is in its way.</summary>
        internal double ExpectedCut => (1 + smoothing) * ExpectedRoundCut;

        /// <summary>
        /// Returns the curves that round the corner, given how much of each of the two edges may be used: a flanking
        /// curve, the circular arc and the other flanking curve.
        /// </summary>
        internal List<Cubic> Cubics(double allowedCut0, double allowedCut1)
        {
            double allowedCut = Math.Min(allowedCut0, allowedCut1);
            if (ExpectedRoundCut < Maths.DistanceEpsilon || allowedCut < Maths.DistanceEpsilon || cornerRadius < Maths.DistanceEpsilon)
            {
                // Not rounded: the corner is a single point.
                return [Cubic.StraightLine(p1.X, p1.Y, p1.X, p1.Y)];
            }

            // Use the cut that fits, and scale the radius with it.
            double actualRoundCut = Math.Min(allowedCut, ExpectedRoundCut);
            double actualSmoothing0 = ActualSmoothing(allowedCut0);
            double actualSmoothing1 = ActualSmoothing(allowedCut1);
            double actualR = cornerRadius * actualRoundCut / ExpectedRoundCut;
            double centerDistance = Math.Sqrt(Maths.Square(actualR) + Maths.Square(actualRoundCut));
            Vec2 center = p1 + (((d1 + d2) / 2).Direction() * centerDistance);
            Vec2 circleIntersection0 = p1 + (d1 * actualRoundCut);
            Vec2 circleIntersection2 = p1 + (d2 * actualRoundCut);
            Cubic flanking0 = FlankingCurve(actualRoundCut, actualSmoothing0, p1, p0, circleIntersection0, circleIntersection2, center, actualR);
            Cubic flanking2 = FlankingCurve(actualRoundCut, actualSmoothing1, p1, p2, circleIntersection2, circleIntersection0, center, actualR).Reverse();
            return
            [
                flanking0,
                Cubic.CircularArc(center.X, center.Y, flanking0.X3, flanking0.Y3, flanking2.X0, flanking2.Y0),
                flanking2,
            ];
        }

        // The smoothing that fits in what an edge allows.
        private double ActualSmoothing(double allowedCut)
        {
            if (allowedCut > ExpectedCut)
            {
                return smoothing;
            }

            return allowedCut > ExpectedRoundCut ? smoothing * (allowedCut - ExpectedRoundCut) / (ExpectedCut - ExpectedRoundCut) : 0;
        }

        // The curve that joins an edge to the circular arc.
        private static Cubic FlankingCurve(
            double actualRoundCut,
            double actualSmoothing,
            Vec2 corner,
            Vec2 sideStart,
            Vec2 circleSegmentIntersection,
            Vec2 otherCircleSegmentIntersection,
            Vec2 circleCenter,
            double actualR)
        {
            Vec2 sideDirection = (sideStart - corner).Direction();
            Vec2 curveStart = corner + (sideDirection * (actualRoundCut * (1 + actualSmoothing)));

            // The end of the curve lies on the circle, between the intersection of the circle with the edge and with
            // the other edge.
            Vec2 p = Maths.Interpolate(circleSegmentIntersection, (circleSegmentIntersection + otherCircleSegmentIntersection) / 2, actualSmoothing);
            Vec2 curveEnd = circleCenter + (Maths.DirectionVector(p.X - circleCenter.X, p.Y - circleCenter.Y) * actualR);
            Vec2 circleTangent = (curveEnd - circleCenter).Rotate90();
            Vec2 anchorEnd = LineIntersection(sideStart, sideDirection, curveEnd, circleTangent) ?? circleSegmentIntersection;
            Vec2 anchorStart = (curveStart + (anchorEnd * 2)) / 3;
            return Cubic.From(curveStart, anchorStart, anchorEnd, curveEnd);
        }

        // Where the line through p0 in the direction d0 meets the line through p1 in the direction d1, if they are
        // not parallel.
        private static Vec2? LineIntersection(Vec2 p0, Vec2 d0, Vec2 p1, Vec2 d1)
        {
            Vec2 rotatedD1 = d1.Rotate90();
            double den = d0.Dot(rotatedD1);
            if (Math.Abs(den) < Maths.DistanceEpsilon)
            {
                return null;
            }

            double num = (p1 - p0).Dot(rotatedD1);
            if (Math.Abs(den) < Maths.DistanceEpsilon * Math.Abs(num))
            {
                return null;
            }

            return p0 + (d0 * (num / den));
        }
    }
}
