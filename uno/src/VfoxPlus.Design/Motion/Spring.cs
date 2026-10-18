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

using System.Collections.Concurrent;

namespace VfoxPlus.Design.Motion;

/// <summary>
/// A damped spring of unit mass: the Material 3 Expressive motion system uses springs instead of durations and
/// curves (see <c>docs/design-system.md#motion</c>).
/// </summary>
/// <param name="Damping">The damping ratio: 1 is critically damped, less bounces.</param>
/// <param name="Stiffness">The stiffness; higher settles faster.</param>
public readonly record struct Spring(double Damping, double Stiffness)
{
    // The tolerance of a spring at rest, as in Flutter's physics.
    private const double RestDistance = 0.001;
    private const double RestVelocity = 0.01;
    private static readonly ConcurrentDictionary<Spring, TimeSpan> SettleTimes = new();

    /// <summary>Gets how long the spring needs to travel from 0 to 1 and come to rest; page transitions, which bring their own curves, last that long.</summary>
    public TimeSpan SettleTime => SettleTimes.GetOrAdd(this, static spring => spring.ComputeSettleTime());

    /// <summary>Returns the position of the spring <paramref name="t"/> seconds after it started at 0, at rest, with 1 as its target.</summary>
    /// <param name="t">The time in seconds.</param>
    /// <returns>The position.</returns>
    public double Position(double t) => 1 + Step(-1, 0, t).Displacement;

    /// <summary>
    /// Returns the displacement from the target and the velocity of the spring after <paramref name="dt"/> seconds,
    /// when it started <paramref name="displacement"/> away from its target, moving at <paramref name="velocity"/>.
    /// It is the exact solution of the equation of the damped oscillator.
    /// </summary>
    /// <param name="displacement">The distance from the target at the start.</param>
    /// <param name="velocity">The velocity at the start.</param>
    /// <param name="dt">The time in seconds.</param>
    /// <returns>The displacement and the velocity at the end.</returns>
    public (double Displacement, double Velocity) Step(double displacement, double velocity, double dt)
    {
        double omega = Math.Sqrt(Stiffness);
        double zeta = Damping;
        if (zeta < 1)
        {
            double omegaD = omega * Math.Sqrt(1 - (zeta * zeta));
            double decay = Math.Exp(-zeta * omega * dt);
            double cos = Math.Cos(omegaD * dt);
            double sin = Math.Sin(omegaD * dt);
            double b = (velocity + (zeta * omega * displacement)) / omegaD;
            return (
                decay * ((displacement * cos) + (b * sin)),
                decay * ((velocity * cos) - (((displacement * omegaD) + (zeta * omega * b)) * sin)));
        }

        if (zeta == 1)
        {
            double decay = Math.Exp(-omega * dt);
            double b = velocity + (omega * displacement);
            return (decay * (displacement + (b * dt)), decay * (velocity - (omega * b * dt)));
        }

        double root = Math.Sqrt((zeta * zeta) - 1);
        double r1 = -omega * (zeta - root);
        double r2 = -omega * (zeta + root);
        double c2 = (velocity - (r1 * displacement)) / (r2 - r1);
        double c1 = displacement - c2;
        double e1 = Math.Exp(r1 * dt);
        double e2 = Math.Exp(r2 * dt);
        return ((c1 * e1) + (c2 * e2), (c1 * r1 * e1) + (c2 * r2 * e2));
    }

    /// <summary>Determines whether a displacement and a velocity are those of a spring at rest.</summary>
    /// <param name="displacement">The distance from the target.</param>
    /// <param name="velocity">The velocity.</param>
    /// <returns><see langword="true"/> if the spring is at rest.</returns>
    public static bool IsAtRest(double displacement, double velocity) =>
        Math.Abs(displacement) < RestDistance && Math.Abs(velocity) < RestVelocity;

    private TimeSpan ComputeSettleTime()
    {
        const double stepSize = 1.0 / 240;
        for (double t = 0; t < 5; t += stepSize)
        {
            (double x, double v) = Step(-1, 0, t);
            if (IsAtRest(x, v))
            {
                return TimeSpan.FromSeconds(t);
            }
        }

        return TimeSpan.FromSeconds(5);
    }
}

/// <summary>
/// The six springs of a motion scheme. <em>Spatial</em> springs move, resize and reshape and may overshoot;
/// <em>effects</em> springs change colour and opacity and never do. Each comes in three speeds: fast for small
/// components, default for partial-screen changes, slow for full-screen changes.
/// </summary>
/// <param name="FastSpatial">The fast spatial spring.</param>
/// <param name="DefaultSpatial">The default spatial spring.</param>
/// <param name="SlowSpatial">The slow spatial spring.</param>
/// <param name="FastEffects">The fast effects spring.</param>
/// <param name="DefaultEffects">The default effects spring.</param>
/// <param name="SlowEffects">The slow effects spring.</param>
public sealed record MotionScheme(
    Spring FastSpatial,
    Spring DefaultSpatial,
    Spring SlowSpatial,
    Spring FastEffects,
    Spring DefaultEffects,
    Spring SlowEffects)
{
    /// <summary>Gets the expressive scheme, which vfox-plus uses: bouncy spatial springs.</summary>
    public static MotionScheme Expressive { get; } = new(
        new Spring(0.6, 800),
        new Spring(0.8, 380),
        new Spring(0.8, 200),
        new Spring(1, 3800),
        new Spring(1, 1600),
        new Spring(1, 800));
}

/// <summary>
/// A value that follows a target on a spring. It keeps its velocity when the target changes, so that interrupted
/// animations stay smooth. Whoever owns the value calls <see cref="Step"/> once per frame while it moves.
/// </summary>
public sealed class SpringValue
{
    // A long pause between frames (a hidden window) must not skip the animation.
    private const double LongestStep = 0.05;

    private readonly Spring spring;
    private double velocity;

    /// <summary>Initializes a new instance of the <see cref="SpringValue"/> class, at rest.</summary>
    /// <param name="spring">The spring it follows its target on.</param>
    /// <param name="initial">The value it starts at, and its first target.</param>
    public SpringValue(Spring spring, double initial = 0)
    {
        this.spring = spring;
        Value = initial;
        Target = initial;
    }

    /// <summary>Gets the current value.</summary>
    public double Value { get; private set; }

    /// <summary>Gets the value the spring moves towards.</summary>
    public double Target { get; private set; }

    /// <summary>Gets a value indicating whether the value has not reached its target.</summary>
    public bool IsMoving => Value != Target || velocity != 0;

    /// <summary>Moves the value towards a new target, keeping its velocity.</summary>
    /// <param name="target">The target.</param>
    /// <param name="immediately">Whether to jump to it at once, as reduced motion asks.</param>
    public void To(double target, bool immediately = false)
    {
        Target = target;
        if (immediately)
        {
            Snap(target);
        }
    }

    /// <summary>Puts the value on a target at once.</summary>
    /// <param name="target">The new value and target.</param>
    public void Snap(double target)
    {
        Value = target;
        Target = target;
        velocity = 0;
    }

    /// <summary>Advances the value by a time.</summary>
    /// <param name="seconds">How much time has passed since the last step.</param>
    /// <returns><see langword="true"/> while the value is still moving.</returns>
    public bool Step(double seconds)
    {
        if (!IsMoving || seconds <= 0)
        {
            return IsMoving;
        }

        (double displacement, double newVelocity) = spring.Step(Value - Target, velocity, Math.Min(seconds, LongestStep));
        if (Spring.IsAtRest(displacement, newVelocity))
        {
            Snap(Target);
            return false;
        }

        Value = Target + displacement;
        velocity = newVelocity;
        return true;
    }
}
