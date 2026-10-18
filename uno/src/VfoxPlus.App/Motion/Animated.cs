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

namespace VfoxPlus.App.Motion;

/// <summary>
/// A number that follows a target on a spring and hands each value to a function that applies it to a property of an
/// element, such as an opacity or a corner radius.
/// </summary>
internal sealed class Animated
{
    private readonly SpringValue spring;
    private readonly Action<double> apply;
    private bool running;

    /// <summary>Initializes a new instance of the <see cref="Animated"/> class, at rest. Nothing is applied yet: the owner, whose other animations may not exist yet, applies the first value with <see cref="Apply"/>.</summary>
    /// <param name="spring">The spring the value follows its target on.</param>
    /// <param name="apply">Hands each value to the element.</param>
    /// <param name="initial">The first value.</param>
    public Animated(Spring spring, Action<double> apply, double initial = 0)
    {
        this.spring = new SpringValue(spring, initial);
        this.apply = apply;
    }

    /// <summary>Gets the current value.</summary>
    public double Value => spring.Value;

    /// <summary>Gets the target of the value.</summary>
    public double Target => spring.Target;

    /// <summary>Gets a value indicating whether the value has not reached its target.</summary>
    public bool IsMoving => spring.IsMoving;

    /// <summary>Hands the current value to the element.</summary>
    public void Apply() => apply(spring.Value);

    /// <summary>Moves the value towards a target, or puts it there when the desktop asks for less motion.</summary>
    /// <param name="target">The target.</param>
    public void To(double target)
    {
        if (FrameClock.Instance.ReducedMotion)
        {
            Snap(target);
            return;
        }

        spring.To(target);
        if (spring.IsMoving && !running)
        {
            running = true;
            FrameClock.Instance.Run(OnFrame);
        }
    }

    /// <summary>Puts the value on a target at once.</summary>
    /// <param name="target">The value.</param>
    public void Snap(double target)
    {
        spring.Snap(target);
        apply(target);
    }

    private bool OnFrame(double seconds)
    {
        bool moving = spring.Step(seconds);
        apply(spring.Value);
        running = moving;
        return moving;
    }
}
