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

using VfoxPlus.Core;
using VfoxPlus.Core.Processes;
using VfoxPlus.Core.Settings;
using VfoxPlus.Core.Vfox;

namespace VfoxPlus.State;

/// <summary>
/// What an <see cref="AppModel"/> is made of.
/// </summary>
/// <param name="Runner">Spawns vfox.</param>
/// <param name="SettingsStore">Persists the settings.</param>
/// <param name="Environment">The environment of the application.</param>
/// <param name="Locator">Finds vfox; <see langword="null"/> uses a locator that searches <c>PATH</c> and the usual places.</param>
public sealed record AppModelOptions(
    IProcessRunner Runner,
    ISettingsStore SettingsStore,
    IReadOnlyDictionary<string, string> Environment,
    VfoxLocator? Locator = null);

/// <summary>
/// The application state: every controller, wired together.
/// </summary>
/// <remarks>
/// The model owns the <see cref="VfoxClient"/> and keeps the derived data consistent: installed SDKs are
/// reloaded whenever vfox becomes available or a task that changes them finishes.
/// </remarks>
public sealed class AppModel : IDisposable
{
    private readonly Lock gate = new();
    private readonly Dictionary<int, PreparedPluginSource> pluginCopies = [];
    private readonly List<IDisposable> stops;
    private VfoxAvailability lastAvailability = VfoxAvailability.Unknown;
    private AppSettings? appliedSettings;

    /// <summary>Initializes a new instance of the <see cref="AppModel"/> class.</summary>
    /// <param name="options">What the model is made of.</param>
    public AppModel(AppModelOptions options)
    {
        VfoxLocator locator = options.Locator ?? new VfoxLocator(options.Environment);
        Client = new VfoxClient(options.Runner, new VfoxLaunchConfig { BaseEnvironment = options.Environment });
        Settings = new SettingsController(options.SettingsStore);
        Vfox = new VfoxController(Client, locator, options.Environment);
        Installed = new AsyncResource<IReadOnlyList<InstalledSdk>>(Client.InstalledSdksAsync);
        Registry = new AsyncResource<IReadOnlyList<AvailablePlugin>>(Client.AvailablePluginsAsync);
        VfoxConfig = new VfoxConfigController(Client);
        Tasks = new TaskQueue(Client);
        stops =
        [
            Tasks.OnFinished(OnTaskFinished),
            Vfox.AddListener(OnVfoxChanged),
            Settings.AddListener(OnSettingsChanged),
        ];
    }

    /// <summary>Gets the vfox client shared by all controllers.</summary>
    public VfoxClient Client { get; }

    /// <summary>Gets the application preferences.</summary>
    public SettingsController Settings { get; }

    /// <summary>Gets the availability of the vfox executable.</summary>
    public VfoxController Vfox { get; }

    /// <summary>Gets the installed plugins and their versions.</summary>
    public AsyncResource<IReadOnlyList<InstalledSdk>> Installed { get; }

    /// <summary>Gets the plugins available in the registry.</summary>
    public AsyncResource<IReadOnlyList<AvailablePlugin>> Registry { get; }

    /// <summary>Gets the vfox configuration.</summary>
    public VfoxConfigController VfoxConfig { get; }

    /// <summary>Gets the queue that runs state-changing operations.</summary>
    public TaskQueue Tasks { get; }

    /// <summary>Loads the settings and probes vfox.</summary>
    /// <returns>A task that completes when vfox has been probed.</returns>
    public Task StartAsync()
    {
        Settings.Load();
        return ApplySettingsAsync();
    }

    /// <summary>Returns the installed SDK called <paramref name="name"/>, if it is installed.</summary>
    /// <param name="name">The plugin name.</param>
    /// <returns>The SDK, or <see langword="null"/>.</returns>
    public InstalledSdk? InstalledSdk(string name) => Installed.Value?.FirstOrDefault(sdk => sdk.Name == name);

    /// <summary>Creates the controller for the detail view of a plugin.</summary>
    /// <param name="name">The plugin name.</param>
    /// <returns>The controller.</returns>
    public SdkDetailController SdkDetail(string name) => new(Client, name);

    /// <summary>Queues a command.</summary>
    /// <param name="command">The command.</param>
    /// <returns>Its task.</returns>
    public VfoxTask Run(VfoxCommand command) => Tasks.Enqueue(command);

    /// <summary>
    /// Adds a plugin from a URL or a local <c>.zip</c> or <c>.lua</c> file, optionally under an alias. vfox
    /// moves a local <c>.lua</c> file away instead of reading it, so such a file is copied into the temporary
    /// directory of vfox first and vfox gets the copy; the copy is deleted once the task has finished.
    /// </summary>
    /// <param name="source">The URL or the path.</param>
    /// <param name="alias">The name to install the plugin under, or an empty string.</param>
    /// <returns>The task.</returns>
    /// <exception cref="VfoxUsageException">The alias is not a valid plugin name.</exception>
    public VfoxTask AddPluginFromSource(string source, string alias = "")
    {
        string temporaryRoot = PluginSources.VfoxTemporaryDirectory(VfoxClient.Environment(Client.LaunchConfig, hookShell: false));
        PreparedPluginSource prepared = PluginSources.Prepare(source, temporaryRoot);
        VfoxCommand command;
        try
        {
            command = VfoxCommands.AddPlugin(string.Empty, prepared.Path, alias);
        }
        catch (VfoxUsageException)
        {
            prepared.Dispose();
            throw;
        }

        VfoxTask task = Run(command);
        if (prepared.IsCopy)
        {
            lock (gate)
            {
                pluginCopies[task.Id] = prepared;
            }
        }

        return task;
    }

    /// <summary>Releases every resource.</summary>
    public void Dispose()
    {
        foreach (IDisposable stop in stops)
        {
            stop.Dispose();
        }

        List<PreparedPluginSource> copies;
        lock (gate)
        {
            copies = [.. pluginCopies.Values];
            pluginCopies.Clear();
        }

        copies.ForEach(copy => copy.Dispose());
        Tasks.Dispose();
        Installed.Dispose();
        Registry.Dispose();
        VfoxConfig.Dispose();
        Vfox.Dispose();
        Settings.Dispose();
    }

    private void OnVfoxChanged()
    {
        VfoxAvailability availability = Vfox.Availability;
        bool changed;
        lock (gate)
        {
            changed = availability != lastAvailability;
            lastAvailability = availability;
        }

        if (!changed || availability != VfoxAvailability.Ready)
        {
            return;
        }

        _ = Installed.LoadAsync();
        if (Registry.HasValue || Registry.Error is not null)
        {
            _ = Registry.LoadAsync();
        }

        if (VfoxConfig.Entries.HasValue || VfoxConfig.Entries.Error is not null)
        {
            _ = VfoxConfig.Entries.LoadAsync();
        }
    }

    private void OnSettingsChanged()
    {
        AppSettings current = Settings.Settings;
        AppSettings? previous;
        lock (gate)
        {
            previous = appliedSettings;
        }

        // Before the first apply, StartAsync applies the settings itself.
        if (previous is null
            || (previous.VfoxExecutable == current.VfoxExecutable
                && previous.ExtraEnvironment.Count == current.ExtraEnvironment.Count
                && previous.ExtraEnvironment.All(pair => current.ExtraEnvironment.TryGetValue(pair.Key, out string? value) && value == pair.Value)))
        {
            return;
        }

        _ = ApplySettingsAsync();
    }

    private Task ApplySettingsAsync()
    {
        AppSettings current = Settings.Settings;
        lock (gate)
        {
            appliedSettings = current;

            // Force a reload of the dependent data even if vfox stays available.
            lastAvailability = VfoxAvailability.Unknown;
        }

        return Vfox.ApplyAsync(current);
    }

    private void OnTaskFinished(VfoxTask task)
    {
        PreparedPluginSource? prepared;
        lock (gate)
        {
            pluginCopies.Remove(task.Id, out prepared);
        }

        prepared?.Dispose();
        if (task.Status == VfoxTaskStatus.Cancelled && task.StartedAt is null)
        {
            return;
        }

        _ = Installed.LoadAsync();
    }
}
