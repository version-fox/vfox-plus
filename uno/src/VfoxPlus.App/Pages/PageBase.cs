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

using VfoxPlus.App.Services;
using VfoxPlus.State;

namespace VfoxPlus.App.Pages;

/// <summary>
/// A page: it shows the state of the application, and shows it again when the state changes while the page is on screen;
/// a page that is not shows it when it comes back.
/// </summary>
internal abstract class PageBase : Grid
{
    private readonly Action refresh;
    private readonly Action request;
    private bool active;
    private bool dirty = true;

    protected PageBase()
    {
        refresh = Refresh;
        request = () => Request();
        Env.L10n.Changed += () => Env.Dispatch.PostOnce(request);
        Env.ThemeChanged += () => Env.Dispatch.PostOnce(request);
    }

    /// <summary>Gets the environment of the window.</summary>
    protected static Services.Env Env => Services.Env.Current;

    /// <summary>Gets or sets a value indicating whether the page is the one that shows.</summary>
    public bool Active
    {
        get => active;
        set
        {
            active = value;
            if (value && dirty)
            {
                dirty = false;
                Refresh();
            }
        }
    }

    /// <summary>Shows the state of the application again.</summary>
    public abstract void Refresh();

    /// <summary>Returns a message in the language of the user interface.</summary>
    /// <param name="key">The key of the message.</param>
    /// <param name="args">The arguments of its placeholders.</param>
    /// <returns>The text.</returns>
    protected static string T(string key, params (string Name, object? Value)[] args) => Env.T(key, args);

    /// <summary>Shows the page again whenever a task changes, such as when it reports how far it has come.</summary>
    protected void WatchTaskProgress() => Env.TasksChanged += () => Env.Dispatch.PostOnce(request);

    /// <summary>Shows the page again whenever one of the objects changes.</summary>
    /// <param name="sources">The objects.</param>
    protected void Watch(params IListenable[] sources) =>
        Notifier.ListenAll(sources, () => Env.Dispatch.PostOnce(request));

    // Refreshes the page now when it shows, and when it comes back otherwise.
    private void Request()
    {
        if (active)
        {
            refresh();
        }
        else
        {
            dirty = true;
        }
    }
}
