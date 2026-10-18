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

using VfoxPlus.App.Controls.Overlays;
using VfoxPlus.App.Services;

namespace VfoxPlus.App.E2E;

/// <summary>
/// What the end-to-end tests add to the application: the scenario that <c>VFOX_PLUS_E2E_SCENARIO</c> names runs in the
/// application, as <c>web/e2e/run.sh</c>, <c>shared/tools/measure.sh</c> and <c>shared/tools/screenshots.sh</c> expect:
/// every step is reported as <c>e2e: …</c> on the standard output, and the application ends with the exit code 0 when
/// the scenario passed and 1 when it failed.
/// </summary>
internal static class Agent
{
    /// <summary>Starts the scenario, if there is one.</summary>
    /// <param name="window">The window.</param>
    /// <param name="env">The environment of the application.</param>
    /// <param name="host">The root of the window.</param>
    public static void Start(Window window, Env env, OverlayHost host)
    {
        if (!env.Environment.TryGetValue("VFOX_PLUS_E2E_SCENARIO", out string? name) || name.Length == 0)
        {
            return;
        }

        Thread thread = new(() => Run(name, new Scenario(window, env, host), env)) { IsBackground = true, Name = "e2e" };
        thread.Start();
    }

    // Ends the application the way the user would: the window closes, and the exit code is the one of the scenario. A
    // process that ends while the graphics threads still run may crash, so it is only forced when the window does not close.
    private static void Exit(Env env, int code)
    {
        Console.Out.Flush();
        Environment.ExitCode = code;
        env.Dispatch.Post(() => Application.Current.Exit());
        Thread.Sleep(TimeSpan.FromSeconds(10));
        Environment.Exit(code);
    }

    private static void Run(string name, Scenario scenario, Env env)
    {
        if (!Scenarios.All.TryGetValue(name, out Action<Scenario>? run))
        {
            Scenario.Report($"unknown scenario \"{name}\"; use one of {string.Join(", ", Scenarios.All.Keys)}");
            Exit(env, 2);
            return;
        }

        DateTime started = DateTime.UtcNow;
        Scenario.Report($"scenario {name} started");
        try
        {
            run(scenario);
        }
        catch (Exception error) when (error is ScenarioException or InvalidOperationException or IOException or TimeoutException or KeyNotFoundException or AggregateException)
        {
            Scenario.Report($"scenario {name} FAILED: {error.Message}");
            try
            {
                // What the user would have seen, to find out why.
                scenario.Dump();
            }
            catch (ScenarioException)
            {
                // The window does not answer: there is nothing to dump.
            }

            Exit(env, 1);
            return;
        }

        Scenario.Report($"scenario {name} passed in {(int)(DateTime.UtcNow - started).TotalSeconds} s");
        Exit(env, 0);
    }
}
