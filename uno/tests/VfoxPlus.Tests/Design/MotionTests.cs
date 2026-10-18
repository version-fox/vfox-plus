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

public class MotionTests
{
    public static TheoryData<string, double> Springs() => new()
    {
        { "bouncy", 0.5 },
        { "critical", 1 },
        { "heavy", 2 },
    };

    [Theory]
    [MemberData(nameof(Springs))]
    public void SpringsStartAtZeroAndSettleAtTheirTarget(string name, double damping)
    {
        Spring spring = new(damping, 400);
        Assert.True(Math.Abs(spring.Position(0)) < 1e-9, $"{name} starts at 0");
        Assert.True(Math.Abs(spring.Position(10) - 1) < 1e-6, $"{name} settles at 1");
    }

    [Fact]
    public void OnlyUnderdampedSpringsOvershoot()
    {
        static double Peak(Spring spring)
        {
            double peak = 0;
            for (int i = 0; i < 400; i++)
            {
                peak = Math.Max(peak, spring.Position(i / 200.0));
            }

            return peak;
        }

        Assert.True(Peak(new Spring(0.5, 400)) > 1.1, "an underdamped spring overshoots");
        Assert.True(Peak(new Spring(1, 400)) <= 1 + 1e-9, "a critically damped one does not");
        Assert.True(Peak(new Spring(2, 400)) <= 1 + 1e-9, "an overdamped one does not");
    }

    [Fact]
    public void SettleTimeGrowsWithSoftnessAndIsRemembered()
    {
        TimeSpan stiff = new Spring(1, 3800).SettleTime;
        TimeSpan soft = new Spring(1, 200).SettleTime;
        Assert.True(stiff > TimeSpan.Zero && soft > stiff, $"{stiff} and {soft}");
        Assert.Equal(soft, new Spring(1, 200).SettleTime);
        Assert.Equal(TimeSpan.FromSeconds(5), new Spring(0.001, 1).SettleTime);
        Assert.True(MotionScheme.Expressive.FastSpatial.Damping < 1, "spatial springs bounce");
        Assert.Equal(1, MotionScheme.Expressive.DefaultEffects.Damping);
    }

    [Fact]
    public void AValueFollowsItsTargetAndKeepsItsVelocityWhenTheTargetChanges()
    {
        SpringValue value = new(MotionScheme.Expressive.DefaultSpatial, 5);
        Assert.False(value.IsMoving);
        Assert.False(value.Step(0.016));

        value.To(10);
        Assert.True(value.IsMoving);
        Assert.True(value.Step(0.016));
        double first = value.Value;
        Assert.InRange(first, 5.0001, 9.999);

        // The target moves back mid-flight: the value does not jump, and carries on in the direction it moved.
        value.To(0);
        Assert.Equal(first, value.Value);
        Assert.True(value.Step(0.016));
        Assert.True(value.Value > first - 0.5, "the velocity is kept");

        int steps = 0;
        while (value.Step(0.016) && steps++ < 1000)
        {
        }

        Assert.True(steps < 1000);
        Assert.Equal(0, value.Value);
        Assert.False(value.IsMoving);
    }

    [Fact]
    public void ASpringValueJumpsWhenToldToAndNeverMovesBackwardsInTime()
    {
        SpringValue value = new(MotionScheme.Expressive.SlowSpatial);
        value.To(1, immediately: true);
        Assert.Equal(1, value.Value);
        Assert.False(value.IsMoving);

        // Time that has not moved, or has gone back, moves nothing.
        value.To(0);
        Assert.True(value.Step(0));
        Assert.True(value.Step(-1));
        Assert.Equal(1, value.Value);
        Assert.True(value.IsMoving);
        value.Snap(0.25);
        Assert.Equal(0.25, value.Value);
        Assert.Equal(0.25, value.Target);
    }

    [Fact]
    public void LongPausesDoNotSkipAnAnimation()
    {
        SpringValue value = new(MotionScheme.Expressive.SlowSpatial);
        value.To(1);

        // A step after a long pause moves the value by one short step (50 ms of the animation), not to the end.
        Assert.True(value.Step(30));
        Assert.InRange(value.Value, 0.0001, 0.2);
    }
}
