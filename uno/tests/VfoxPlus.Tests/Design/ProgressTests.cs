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

using VfoxPlus.Design.Motion;
using Xunit;

namespace VfoxPlus.Tests.Design;

public class ProgressTests
{
    [Fact]
    public void TheLoadingIndicatorMorphsEverySixHundredAndFiftyMilliseconds()
    {
        (int step0, double progress0, _) = Progress.Loading(TimeSpan.Zero);
        Assert.Equal(0, step0);
        Assert.True(Math.Abs(progress0) < 1e-9, "the morph starts at its first shape");

        (int step1, _, _) = Progress.Loading(TimeSpan.FromMilliseconds(649));
        (int step2, _, _) = Progress.Loading(TimeSpan.FromMilliseconds(651));
        Assert.Equal(0, step1);
        Assert.Equal(1, step2);

        // The spring overshoots, so the progress goes beyond 1 before it settles.
        double peak = Enumerable.Range(0, 650).Max(ms => Progress.Loading(TimeSpan.FromMilliseconds(ms)).Progress);
        Assert.InRange(peak, 1.0, 1.5);
    }

    [Fact]
    public void TheLoadingIndicatorTurnsOnceInAboutFourAndAHalfSecondsAndAQuarterPerMorph()
    {
        double atStart = Progress.Loading(TimeSpan.Zero).Rotation;
        double later = Progress.Loading(TimeSpan.FromMilliseconds(4666)).Rotation;

        // One constant turn, and 7 morphs of a quarter turn each in 4.6 seconds.
        Assert.True(later - atStart > 2 * Math.PI);
        Assert.InRange(later - atStart, 2 * Math.PI, 2 * Math.PI + (8 * Math.PI / 2));
    }

    [Theory]
    [InlineData(0.2, 0, 0.8, 1, 0, 0)]
    [InlineData(0.2, 0, 0.8, 1, 1, 1)]
    [InlineData(0, 0, 1, 1, 0.5, 0.5)]
    public void TheEasingCurvesStartAtZeroAndEndAtOne(double a, double b, double c, double d, double t, double expected)
    {
        Assert.Equal(expected, Progress.CubicBezier(a, b, c, d, t), 2);
    }

    [Fact]
    public void AnEaseInIsBelowTheDiagonalAndAnEaseOutAbove()
    {
        Assert.True(Progress.CubicBezier(0.4, 0, 1, 1, 0.5) < 0.5);
        Assert.True(Progress.CubicBezier(0, 0, 0.65, 1, 0.5) > 0.5);
    }

    [Fact]
    public void AnIntervalIsZeroBeforeItBeginsAndOneAfterItEnds()
    {
        Func<double, double> linear = t => t;
        Assert.Equal(0, Progress.Interval(0.1, 0.2, 0.6, linear));
        Assert.Equal(1, Progress.Interval(0.9, 0.2, 0.6, linear));
        Assert.Equal(0.5, Progress.Interval(0.4, 0.2, 0.6, linear), 9);
    }

    [Fact]
    public void TheBarsOfTheIndeterminateIndicatorMoveRightAndFollowTheirHeads()
    {
        (double head1, double tail1, double head2, double tail2) = Progress.Indeterminate(0);
        Assert.Equal((0, 0, 0, 0), (head1, tail1, head2, tail2));

        (head1, tail1, _, _) = Progress.Indeterminate(0.3);
        Assert.True(head1 > tail1, "the head leads");

        (head1, tail1, head2, tail2) = Progress.Indeterminate(1);
        Assert.Equal((1, 1, 1, 1), (head1, tail1, head2, tail2));
    }

    [Fact]
    public void TheWaveHasOnePointPerPixelAndTravelsWithItsPhase()
    {
        IReadOnlyList<(double X, double Y)> wave = Progress.Wave(2, 12, 6, 3, 40, 0);
        Assert.Equal(11, wave.Count);
        Assert.Equal(2, wave[0].X);
        Assert.Equal(12, wave[^1].X);
        Assert.All(wave, point => Assert.InRange(point.Y, 3, 9));

        IReadOnlyList<(double X, double Y)> moved = Progress.Wave(2, 12, 6, 3, 40, Math.PI / 2);
        Assert.NotEqual(wave[0].Y, moved[0].Y);

        // A part that is only a point still has two, so that it is drawn.
        Assert.Equal(2, Progress.Wave(5, 5, 6, 3, 40, 0).Count);
        Assert.Equal(2, Progress.Wave(7, 5, 6, 3, 40, 0).Count);
    }
}
