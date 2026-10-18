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

using Microsoft.UI.Dispatching;

namespace VfoxPlus.App.Services;

/// <summary>
/// Hands work to the thread of the window, which is the only one that may touch the user interface: the state of the
/// application changes on the threads that read the output of vfox.
/// </summary>
internal sealed class UiThread
{
    private readonly DispatcherQueue queue;
    private readonly HashSet<Action> once = [];
    private readonly Lock gate = new();

    public UiThread(DispatcherQueue queue) => this.queue = queue;

    /// <summary>Gets a value indicating whether the caller is on the thread of the window.</summary>
    public bool IsCurrent => queue.HasThreadAccess;

    /// <summary>Runs an action on the thread of the window; it may be called from any thread.</summary>
    /// <param name="action">The action.</param>
    public void Post(Action action) => queue.TryEnqueue(() => action());

    /// <summary>
    /// Runs an action on the thread of the window, unless it is already waiting: many changes of the state in a row
    /// refresh a page once.
    /// </summary>
    /// <param name="action">The action; the same delegate is the same request.</param>
    public void PostOnce(Action action)
    {
        lock (gate)
        {
            if (!once.Add(action))
            {
                return;
            }
        }

        queue.TryEnqueue(() =>
        {
            lock (gate)
            {
                once.Remove(action);
            }

            action();
        });
    }
}
