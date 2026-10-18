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
/// The transition between two polygons. The outlines are measured, their corners are matched and both are cut into
/// the same number of curves, so that every curve of one has a partner on the other to interpolate with.
/// </summary>
public sealed class Morph
{
    private readonly List<(Cubic Start, Cubic End)> match;

    /// <summary>Initializes a new instance of the <see cref="Morph"/> class: the transition from a polygon to another.</summary>
    /// <param name="start">The polygon at progress 0.</param>
    /// <param name="end">The polygon at progress 1.</param>
    public Morph(RoundedPolygon start, RoundedPolygon end) => match = MatchPolygons(start, end);

    /// <summary>Returns the outline at a progress, from 0 (start) to 1 (end). Values beyond overshoot, which springs do.</summary>
    /// <param name="progress">The progress.</param>
    /// <returns>The outline, a closed loop of curves.</returns>
    public IReadOnlyList<Cubic> AsCubics(double progress)
    {
        List<Cubic> result = new(match.Count);
        Cubic? first = null;
        Cubic? last = null;
        foreach ((Cubic start, Cubic end) in match)
        {
            Cubic cubic = Cubic.Interpolate(start, end, progress);
            first ??= cubic;
            if (last is { } previous)
            {
                result.Add(previous);
            }

            last = cubic;
        }

        if (last is { } final && first is { } head)
        {
            // Make sure the outline closes exactly.
            result.Add(final with { X3 = head.X0, Y3 = head.Y0 });
        }

        return result;
    }

    // Cuts the outlines of two polygons into pairs of curves.
    private static List<(Cubic, Cubic)> MatchPolygons(RoundedPolygon p1, RoundedPolygon p2)
    {
        MeasuredPolygon measured1 = MeasuredPolygon.Measure(p1);
        MeasuredPolygon measured2 = MeasuredPolygon.Measure(p2);
        DoubleMapper mapper = FeatureMapper(measured1.Features, measured2.Features);

        // The second outline starts where the first corner of the first one maps to.
        double polygon2CutPoint = mapper.MapForward(0);
        MeasuredPolygon bs1 = measured1;
        MeasuredPolygon bs2 = measured2.CutAndShift(polygon2CutPoint);

        List<(Cubic, Cubic)> result = [];
        int i1 = 0;
        int i2 = 0;
        MeasuredCubic? b1 = bs1.Get(i1);
        i1++;
        MeasuredCubic? b2 = bs2.Get(i2);
        i2++;
        while (b1 is not null && b2 is not null)
        {
            // The progress at which each of the two current curves ends, in the space of the first outline.
            double b1a = i1 == bs1.Cubics.Count ? 1 : b1.EndProgress;
            double b2a = i2 != bs2.Cubics.Count
                ? mapper.MapBack(Maths.PositiveModulo(b2.EndProgress + polygon2CutPoint))
                : 1.0;
            double minb = Math.Min(b1a, b2a);

            // Cut the curve that goes on after minb; the other one is whole.
            MeasuredCubic seg1;
            MeasuredCubic? newB1;
            MeasuredCubic seg2;
            MeasuredCubic? newB2;
            if (b1a > minb + Maths.AngleEpsilon)
            {
                (seg1, newB1) = b1.CutAtProgress(minb);
            }
            else
            {
                seg1 = b1;
                newB1 = bs1.Get(i1);
                i1++;
            }

            if (b2a > minb + Maths.AngleEpsilon)
            {
                (seg2, newB2) = b2.CutAtProgress(Maths.PositiveModulo(mapper.MapForward(minb) - polygon2CutPoint));
            }
            else
            {
                seg2 = b2;
                newB2 = bs2.Get(i2);
                i2++;
            }

            result.Add((seg1.Cubic, seg2.Cubic));
            b1 = newB1;
            b2 = newB2;
        }

        return result;
    }

    /// <summary>Matches the corners of two outlines: nearby corners of the same kind are mapped to each other first, as long as the order of the corners along the outline stays the same.</summary>
    private static DoubleMapper FeatureMapper(IReadOnlyList<ProgressableFeature> features1, IReadOnlyList<ProgressableFeature> features2) =>
        new(DoMapping([.. features1.Where(f => f.Feature.IsCorner)], [.. features2.Where(f => f.Feature.IsCorner)]));

    private static List<(double, double)> DoMapping(List<ProgressableFeature> features1, List<ProgressableFeature> features2)
    {
        List<(double Distance, int F1, int F2)> vertices = [];
        for (int i = 0; i < features1.Count; i++)
        {
            for (int j = 0; j < features2.Count; j++)
            {
                double d = FeatureDistSquared(features1[i].Feature, features2[j].Feature);
                if (d != double.MaxValue)
                {
                    vertices.Add((d, i, j));
                }
            }
        }

        // A stable sort: corners at the same distance keep the order they were found in.
        List<(double Distance, int F1, int F2)> sorted = [.. vertices.OrderBy(vertex => vertex.Distance)];
        switch (sorted.Count)
        {
            case 0:
                return [(0, 0), (0.5, 0.5)];
            case 1:
                {
                    double f1 = features1[sorted[0].F1].Progress;
                    double f2 = features2[sorted[0].F2].Progress;
                    return [(f1, f2), ((f1 + 0.5) % 1, (f2 + 0.5) % 1)];
                }
        }

        MappingHelper helper = new();
        foreach ((_, int i1, int i2) in sorted)
        {
            helper.Add(features1, features2, i1, i2);
        }

        return helper.Mapping;
    }

    // The squared distance between the corners, or the largest number when one is convex and the other concave.
    private static double FeatureDistSquared(Feature f1, Feature f2)
    {
        if (f1.IsCorner && f2.IsCorner && f1.IsConvex != f2.IsConvex)
        {
            return double.MaxValue;
        }

        return (RepresentativePoint(f1) - RepresentativePoint(f2)).LengthSquared();
    }

    // The middle between the start and the end of the feature.
    private static Vec2 RepresentativePoint(Feature feature)
    {
        Cubic first = feature.Cubics[0];
        Cubic last = feature.Cubics[^1];
        return new Vec2((first.X0 + last.X3) / 2, (first.Y0 + last.Y3) / 2);
    }

    // The pairs of corners that are mapped to each other.
    private sealed class MappingHelper
    {
        private readonly HashSet<int> usedF1 = [];
        private readonly HashSet<int> usedF2 = [];

        internal List<(double, double)> Mapping { get; } = [];

        internal void Add(List<ProgressableFeature> features1, List<ProgressableFeature> features2, int i1, int i2)
        {
            if (usedF1.Contains(i1) || usedF2.Contains(i2))
            {
                return;
            }

            double p1 = features1[i1].Progress;
            double p2 = features2[i2].Progress;

            // Where the pair goes, to keep the first values ascending.
            int index = Mapping.FindIndex(pair => pair.Item1 >= p1);
            if (index < 0)
            {
                index = Mapping.Count;
            }

            if (index < Mapping.Count && Mapping[index].Item1 == p1)
            {
                throw new InvalidOperationException("Two corners cannot be at the same progress.");
            }

            int n = Mapping.Count;
            if (n >= 1)
            {
                (double beforeX, double beforeY) = Mapping[(index + n - 1) % n];
                (double afterX, double afterY) = Mapping[index % n];

                // Too close to a pair that exists: skip.
                if (ProgressDistance(p1, beforeX) < Maths.DistanceEpsilon || ProgressDistance(p1, afterX) < Maths.DistanceEpsilon
                    || ProgressDistance(p2, beforeY) < Maths.DistanceEpsilon || ProgressDistance(p2, afterY) < Maths.DistanceEpsilon)
                {
                    return;
                }

                // The second values must keep their order as well.
                if (n > 1 && !ProgressInRange(p2, beforeY, afterY))
                {
                    return;
                }
            }

            Mapping.Insert(index, (p1, p2));
            usedF1.Add(i1);
            usedF2.Add(i2);
        }
    }

    // Whether progress lies on the stretch of the outline from progressFrom to progressTo, which may wrap around.
    private static bool ProgressInRange(double progress, double progressFrom, double progressTo) =>
        progressTo >= progressFrom
            ? progress >= progressFrom && progress <= progressTo
            : progress >= progressFrom || progress <= progressTo;

    // How far apart two progress values are on the loop.
    private static double ProgressDistance(double p1, double p2)
    {
        double diff = Math.Abs(p1 - p2);
        return Math.Min(diff, 1 - diff);
    }

    // Maps x from the progress values xValues to yValues, linearly in between.
    private static double LinearMap(IReadOnlyList<double> xValues, IReadOnlyList<double> yValues, double x)
    {
        int segmentStart = -1;
        for (int i = 0; i < xValues.Count; i++)
        {
            if (ProgressInRange(x, xValues[i], xValues[(i + 1) % xValues.Count]))
            {
                segmentStart = i;
                break;
            }
        }

        if (segmentStart == -1)
        {
            throw new InvalidOperationException("No segment for the progress.");
        }

        int segmentEnd = (segmentStart + 1) % xValues.Count;
        double segmentSizeX = Maths.PositiveModulo(xValues[segmentEnd] - xValues[segmentStart]);
        double segmentSizeY = Maths.PositiveModulo(yValues[segmentEnd] - yValues[segmentStart]);
        double position = segmentSizeX >= 0.001 ? Maths.PositiveModulo(x - xValues[segmentStart]) / segmentSizeX : 0.5;
        return Maths.PositiveModulo(yValues[segmentStart] + (segmentSizeY * position));
    }

    // Maps progress values of one outline to those of the other and back, through a list of pairs of corresponding
    // values.
    private sealed class DoubleMapper(IReadOnlyList<(double, double)> mappings)
    {
        private readonly double[] source = [.. mappings.Select(pair => pair.Item1)];
        private readonly double[] target = [.. mappings.Select(pair => pair.Item2)];

        internal double MapForward(double x) => LinearMap(source, target, x);

        internal double MapBack(double x) => LinearMap(target, source, x);
    }

    // A feature and where on the outline, from 0 to 1, it lies.
    private sealed record ProgressableFeature(double Progress, Feature Feature);

    // A curve of an outline, with the stretch of the outline (from 0 to 1) that it covers.
    private sealed class MeasuredCubic
    {
        // The length of a curve is measured on a few segments; this is exact enough to match the outlines, and the
        // measure is the same on both.
        private const int MeasureSegments = 3;

        private readonly double size;

        internal MeasuredCubic(Cubic cubic, double startProgress, double endProgress)
        {
            Cubic = cubic;
            StartProgress = startProgress;
            EndProgress = endProgress;
            size = Measure(cubic);
        }

        internal Cubic Cubic { get; }

        internal double StartProgress { get; }

        internal double EndProgress { get; }

        // Cuts the curve where the outline has reached progress.
        internal (MeasuredCubic, MeasuredCubic) CutAtProgress(double progress)
        {
            double bounded = Math.Min(Math.Max(progress, StartProgress), EndProgress);
            double outlineSize = EndProgress - StartProgress;
            double relative = (bounded - StartProgress) / outlineSize;
            double t = ClosestProgressTo(Cubic, relative * size).Progress;
            (Cubic first, Cubic second) = Cubic.Split(t);
            return (new MeasuredCubic(first, StartProgress, bounded), new MeasuredCubic(second, bounded, EndProgress));
        }

        // Walks the curve until the length threshold is reached. Returns the progress at that point, and the length
        // that was walked.
        private static (double Progress, double Length) ClosestProgressTo(Cubic cubic, double threshold)
        {
            double total = 0;
            double remainder = threshold;
            Vec2 previous = cubic.Anchor0();
            for (int i = 0; i <= MeasureSegments; i++)
            {
                double progress = (double)i / MeasureSegments;
                Vec2 point = cubic.PointOnCurve(progress);
                double segment = (point - previous).Length();
                if (segment >= remainder)
                {
                    return (progress - ((1 - (remainder / segment)) / MeasureSegments), threshold);
                }

                remainder -= segment;
                total += segment;
                previous = point;
            }

            return (1, total);
        }

        // The length of a curve.
        internal static double Measure(Cubic cubic) => ClosestProgressTo(cubic, double.PositiveInfinity).Length;
    }

    // An outline whose curves know their share of its length.
    private sealed class MeasuredPolygon
    {
        private MeasuredPolygon(IReadOnlyList<ProgressableFeature> features) => Features = features;

        internal List<MeasuredCubic> Cubics { get; } = [];

        internal IReadOnlyList<ProgressableFeature> Features { get; }

        // Measures the outline of a polygon and where its corners are.
        internal static MeasuredPolygon Measure(RoundedPolygon polygon)
        {
            List<Cubic> cubics = [];
            List<(Feature Feature, int Cubic)> corners = [];
            foreach (Feature feature in polygon.Features)
            {
                for (int i = 0; i < feature.Cubics.Count; i++)
                {
                    // A corner is placed at the middle of its curves.
                    if (feature.IsCorner && i == feature.Cubics.Count / 2)
                    {
                        corners.Add((feature, cubics.Count));
                    }

                    cubics.Add(feature.Cubics[i]);
                }
            }

            double[] measures = new double[cubics.Count + 1];
            double total = 0;
            for (int i = 0; i < cubics.Count; i++)
            {
                total += MeasuredCubic.Measure(cubics[i]);
                measures[i + 1] = total;
            }

            double[] outlineProgress = [.. measures.Select(measure => measure / total)];
            List<ProgressableFeature> features =
            [
                .. corners.Select(corner => new ProgressableFeature(
                    Maths.PositiveModulo((outlineProgress[corner.Cubic] + outlineProgress[corner.Cubic + 1]) / 2),
                    corner.Feature)),
            ];
            return FromProgress(features, cubics, outlineProgress);
        }

        internal MeasuredCubic? Get(int index) => index < 0 || index >= Cubics.Count ? null : Cubics[index];

        // Makes the outline start at cuttingPoint (from 0 to 1).
        internal MeasuredPolygon CutAndShift(double cuttingPoint)
        {
            if (cuttingPoint < Maths.DistanceEpsilon)
            {
                return this;
            }

            int target = 0;
            for (int i = 0; i < Cubics.Count; i++)
            {
                if (cuttingPoint >= Cubics[i].StartProgress && cuttingPoint <= Cubics[i].EndProgress)
                {
                    target = i;
                    break;
                }
            }

            (MeasuredCubic b1, MeasuredCubic b2) = Cubics[target].CutAtProgress(cuttingPoint);
            List<Cubic> cubics = [b2.Cubic];
            for (int i = 1; i < Cubics.Count; i++)
            {
                cubics.Add(Cubics[(i + target) % Cubics.Count].Cubic);
            }

            cubics.Add(b1.Cubic);

            double[] progress = new double[Cubics.Count + 2];
            for (int i = 0; i < progress.Length; i++)
            {
                if (i == 0)
                {
                    progress[i] = 0;
                }
                else if (i == Cubics.Count + 1)
                {
                    progress[i] = 1;
                }
                else
                {
                    int index = (target + i - 1) % Cubics.Count;
                    progress[i] = Maths.PositiveModulo(Cubics[index].EndProgress - cuttingPoint);
                }
            }

            List<ProgressableFeature> features = [.. Features.Select(f => new ProgressableFeature(Maths.PositiveModulo(f.Progress - cuttingPoint), f.Feature))];
            return FromProgress(features, cubics, progress);
        }

        // Measures the curves given with the progress at which each of them (and the outline) ends.
        private static MeasuredPolygon FromProgress(IReadOnlyList<ProgressableFeature> features, List<Cubic> cubics, double[] outlineProgress)
        {
            MeasuredPolygon measured = new(features);
            double start = 0;
            for (int i = 0; i < cubics.Count; i++)
            {
                if (outlineProgress[i + 1] - outlineProgress[i] > Maths.DistanceEpsilon)
                {
                    measured.Cubics.Add(new MeasuredCubic(cubics[i], start, outlineProgress[i + 1]));
                    start = outlineProgress[i + 1];
                }
            }

            MeasuredCubic last = measured.Cubics[^1];
            measured.Cubics[^1] = new MeasuredCubic(last.Cubic, last.StartProgress, 1);
            return measured;
        }
    }
}
