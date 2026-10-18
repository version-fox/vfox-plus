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

using System.Diagnostics;

namespace VfoxPlus.App.Motion;

/// <summary>
/// Calls the animations once per frame, while there are any: the window draws continuously only while something
/// moves, so that it is idle, and costs no processor time, otherwise.
/// </summary>
internal sealed class FrameClock
{
    private readonly Stopwatch clock = Stopwatch.StartNew();
    private readonly List<Func<double, bool>> steps = [];
    private TimeSpan last;
    private bool running;

    /// <summary>Gets the clock of the application.</summary>
    public static FrameClock Instance { get; } = new();

    /// <summary>Gets or sets a value indicating whether the desktop asks for less motion: springs jump to their targets.</summary>
    public bool ReducedMotion { get; set; }

    /// <summary>Gets the time since the clock started, which the frames of an indeterminate animation are drawn at.</summary>
    public TimeSpan Now => clock.Elapsed;

    /// <summary>Calls a function once per frame until it returns <see langword="false"/>.</summary>
    /// <param name="step">Takes the seconds since the last frame, and says whether it needs the next.</param>
    public void Run(Func<double, bool> step)
    {
        steps.Add(step);
        if (!running)
        {
            running = true;
            last = clock.Elapsed;
            CompositionTarget.Rendering += OnRendering;
        }
    }

    private void OnRendering(object? sender, object e)
    {
        TimeSpan now = clock.Elapsed;
        double seconds = (now - last).TotalSeconds;
        last = now;

        // A step may start another animation, which joins the next frame.
        Func<double, bool>[] current = [.. steps];
        steps.Clear();
        foreach (Func<double, bool> step in current)
        {
            if (step(seconds))
            {
                steps.Add(step);
            }
        }

        if (steps.Count == 0)
        {
            running = false;
            CompositionTarget.Rendering -= OnRendering;
        }
    }
}
