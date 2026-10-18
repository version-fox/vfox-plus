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
using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Core.I18n;
using VfoxPlus.Core.Settings;
using VfoxPlus.Core.Vfox;
using VfoxPlus.Platform;
using VfoxPlus.State;

namespace VfoxPlus.App.Services;

/// <summary>
/// What every part of the user interface shares: the state of the application, the desktop, the theme, the language
/// and where the user is. There is one for the window.
/// </summary>
internal sealed class Env
{
    private readonly string[] locales;
    private readonly Dictionary<VfoxTask, IDisposable> watched = [];
    private (uint Seed, bool Dark) themeKey;

    public Env(AppModel model, IPlatform platform, UiThread dispatch, IReadOnlyDictionary<string, string> environment, string assets)
    {
        Current = this;
        Model = model;
        Platform = platform;
        Dispatch = dispatch;
        Environment = environment;
        Assets = assets;
        L10n = new L10n(Path.Combine(assets, "i18n"));
        Theme = new AppTheme(AppSettings.DefaultSeedColor, dark: false);
        Ui.Theme = Theme;
        themeKey = (AppSettings.DefaultSeedColor, false);
        locales = [.. Languages.SystemLocales(environment)];
        SyncAppearance();
        model.Settings.AddListener(() => dispatch.PostOnce(SyncAppearance));
        platform.AppearanceChanged += () => dispatch.PostOnce(SyncAppearance);
        model.Tasks.OnFinished(task => dispatch.Post(() => Announce(task)));
        model.Tasks.AddListener(() => dispatch.PostOnce(WatchTasks));
    }

    /// <summary>Gets the environment of the window.</summary>
    public static Env Current { get; private set; } = null!;

    /// <summary>Gets the state of the application.</summary>
    public AppModel Model { get; }

    /// <summary>Gets what the desktop offers.</summary>
    public IPlatform Platform { get; }

    /// <summary>Gets the thread of the window.</summary>
    public UiThread Dispatch { get; }

    /// <summary>Gets the environment variables of the process.</summary>
    public IReadOnlyDictionary<string, string> Environment { get; }

    /// <summary>Gets the directory of the files the build generates for the application.</summary>
    public string Assets { get; }

    /// <summary>Gets the messages of the user interface.</summary>
    public L10n L10n { get; }

    /// <summary>Gets the colours.</summary>
    public AppTheme Theme { get; }

    /// <summary>Gets where the user is.</summary>
    public Navigation Nav { get; } = new();

    /// <summary>Gets the root of the window, once the user interface exists.</summary>
    public OverlayHost? Overlays { get; set; }

    /// <summary>Returns a message in the language of the user interface.</summary>
    /// <param name="key">The key of the message.</param>
    /// <param name="args">The arguments of its placeholders: names and values.</param>
    /// <returns>The text.</returns>
    public string T(string key, params (string Name, object? Value)[] args) => L10n.T(key, args);

    /// <summary>Opens a web address in the browser of the user.</summary>
    /// <param name="address">The address.</param>
    public void OpenUrl(string address) => Platform.OpenUrl(address);

    /// <summary>Lets the user choose a file or a directory, and runs an action with it on the thread of the window.</summary>
    /// <param name="options">What to choose.</param>
    /// <param name="chosen">Runs with the path the user chose; it does not run when the user cancels.</param>
    public void Choose(ChooseOptions options, Action<string> chosen) => _ = ChooseAsync(options, chosen);

    /// <summary>Shows a short message in the snack bar.</summary>
    /// <param name="text">The message.</param>
    public void Notify(string text) => Overlays?.ShowSnackbar(new Snack { Text = text });

    /// <summary>Queues a command, or tells the user why it cannot be queued.</summary>
    /// <param name="build">Builds the command.</param>
    public void Run(Func<VfoxCommand> build)
    {
        try
        {
            Model.Run(build());
        }
        catch (VfoxUsageException error)
        {
            Notify(error.Message);
        }
    }

    /// <summary>Applies the theme, the motion and the language that the settings and the desktop ask for.</summary>
    public void SyncAppearance()
    {
        AppSettings settings = Model.Settings.Settings;
        Appearance appearance = Platform.Appearance;
        uint seed = settings.UseSystemColor && appearance.Accent is { } accent ? accent : settings.SeedColor;
        bool dark = settings.Theme switch
        {
            ThemePreference.Dark => true,
            ThemePreference.System => appearance.Dark == true,
            _ => false,
        };
        FrameClock.Instance.ReducedMotion = appearance.ReducedMotion == true;
        if (themeKey != (seed, dark))
        {
            themeKey = (seed, dark);
            Theme.Apply(seed, dark);
            ThemeChanged?.Invoke();
        }

        L10n.Select(settings.Language, locales);
    }

    /// <summary>Raised when the colours changed, which the elements that mix them themselves redo.</summary>
    public event Action? ThemeChanged;

    private async Task ChooseAsync(ChooseOptions options, Action<string> chosen)
    {
        try
        {
            string path = await Task.Run(() => Platform.ChooseAsync(options)).ConfigureAwait(false);
            if (path.Length > 0)
            {
                Dispatch.Post(() => chosen(path));
            }
        }
        catch (PlatformException error)
        {
            Dispatch.Post(() => Notify(error.Message));
        }
    }

    // Tells the user that a task has finished, in a snack bar.
    private void Announce(VfoxTask task)
    {
        if (task.Status == VfoxTaskStatus.Cancelled)
        {
            return;
        }

        string title = Presentation.TaskTitle(task.Command, T);
        bool failed = task.Status == VfoxTaskStatus.Failed;
        if (!failed && Nav.Current == Destination.Tasks)
        {
            return;
        }

        if (failed)
        {
            // Failures stay until the user dismisses them, so that they can be read in full.
            string text = $"{T("taskFailed", ("title", title))}\n{task.ErrorMessage}".Trim();
            Overlays?.ShowSnackbar(new Snack
            {
                Text = text,
                Persist = true,
                CloseLabel = T("actionClose"),
                Action = T("taskView"),
                OnAction = () => Nav.Go(Destination.Tasks),
            });
        }
        else
        {
            Overlays?.ShowSnackbar(new Snack { Text = T("taskSucceeded", ("title", title)), CloseLabel = T("actionClose") });
        }
    }

    // Asks for the refresh of a page whenever a task changes, such as when it reports its progress.
    private void WatchTasks()
    {
        IReadOnlyList<VfoxTask> tasks = Model.Tasks.Tasks;
        foreach (VfoxTask task in tasks)
        {
            if (!watched.ContainsKey(task))
            {
                watched[task] = task.AddListener(() => Dispatch.PostOnce(TaskChanged));
            }
        }

        foreach (VfoxTask gone in watched.Keys.Where(task => !tasks.Contains(task)).ToArray())
        {
            watched[gone].Dispose();
            watched.Remove(gone);
        }
    }

    /// <summary>Raised when a task changed.</summary>
    public event Action? TasksChanged;

    private void TaskChanged() => TasksChanged?.Invoke();
}
