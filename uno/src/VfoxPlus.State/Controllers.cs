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
using VfoxPlus.Core;
using VfoxPlus.Core.Settings;
using VfoxPlus.Core.Vfox;

namespace VfoxPlus.State;

/// <summary>
/// Holds the <see cref="AppSettings"/> and persists every change.
/// </summary>
/// <param name="store">Where the settings are kept.</param>
public sealed class SettingsController(ISettingsStore store) : Notifier
{
    private readonly Lock gate = new();
    private AppSettings settings = new();

    /// <summary>Gets the current settings.</summary>
    public AppSettings Settings
    {
        get
        {
            lock (gate)
            {
                return settings;
            }
        }
    }

    /// <summary>Loads the persisted settings.</summary>
    public void Load()
    {
        AppSettings loaded = store.Load();
        lock (gate)
        {
            settings = loaded;
        }

        NotifyListeners();
    }

    /// <summary>Applies a change to the settings and saves the result, unless nothing changed.</summary>
    /// <param name="change">Makes the new settings from the current ones.</param>
    public void Update(Func<AppSettings, AppSettings> change)
    {
        AppSettings updated;
        lock (gate)
        {
            updated = change(settings);
            if (updated.Equals(settings))
            {
                return;
            }

            settings = updated;
        }

        NotifyListeners();
        try
        {
            store.Save(updated);
        }
        catch (Exception error) when (error is IOException or UnauthorizedAccessException)
        {
            Trace.TraceError($"saving the settings failed: {error.Message}");
        }
    }
}

/// <summary>Tells whether vfox can be used.</summary>
public enum VfoxAvailability
{
    /// <summary>Not checked yet.</summary>
    Unknown,

    /// <summary>The executable is being probed.</summary>
    Checking,

    /// <summary>vfox runs and reported its version.</summary>
    Ready,

    /// <summary>No vfox executable was found.</summary>
    NotFound,

    /// <summary>An executable was found but did not behave like vfox.</summary>
    Broken,
}

/// <summary>
/// Locates vfox, configures the <see cref="VfoxClient"/> and checks that vfox works.
/// </summary>
public sealed class VfoxController : Notifier
{
    private readonly VfoxClient client;
    private readonly VfoxLocator locator;
    private readonly IReadOnlyDictionary<string, string> environment;
    private readonly Lock gate = new();
    private VfoxAvailability availability = VfoxAvailability.Unknown;
    private string? executable;
    private bool custom;
    private string? version;
    private Exception? error;

    /// <summary>Initializes a new instance of the <see cref="VfoxController"/> class.</summary>
    /// <param name="client">The client to configure.</param>
    /// <param name="locator">Finds vfox.</param>
    /// <param name="environment">The environment of the application.</param>
    public VfoxController(VfoxClient client, VfoxLocator locator, IReadOnlyDictionary<string, string> environment)
    {
        this.client = client;
        this.locator = locator;
        this.environment = environment;
    }

    /// <summary>Gets whether vfox can be used.</summary>
    public VfoxAvailability Availability
    {
        get
        {
            lock (gate)
            {
                return availability;
            }
        }
    }

    /// <summary>Gets the executable in use, if one was found.</summary>
    public string? Executable
    {
        get
        {
            lock (gate)
            {
                return executable;
            }
        }
    }

    /// <summary>Gets a value indicating whether the executable was chosen by the user rather than detected.</summary>
    public bool IsCustomExecutable
    {
        get
        {
            lock (gate)
            {
                return custom;
            }
        }
    }

    /// <summary>Gets the vfox version, once vfox is ready.</summary>
    public string? Version
    {
        get
        {
            lock (gate)
            {
                return version;
            }
        }
    }

    /// <summary>Gets why vfox is not available, if it is not.</summary>
    public Exception? Error
    {
        get
        {
            lock (gate)
            {
                return error;
            }
        }
    }

    /// <summary>Points the client at the executable and the environment of the settings and checks the result.</summary>
    /// <param name="appSettings">The settings.</param>
    /// <returns>A task that completes when vfox has been probed.</returns>
    public async Task ApplyAsync(AppSettings appSettings)
    {
        bool isCustom = appSettings.VfoxExecutable.Length > 0;
        string path = isCustom ? appSettings.VfoxExecutable : locator.Locate();
        client.LaunchConfig = client.LaunchConfig with
        {
            Executable = path.Length > 0 ? path : VfoxClient.ExecutableName,
            BaseEnvironment = environment,
            ExtraEnvironment = appSettings.ExtraEnvironment,
        };
        lock (gate)
        {
            custom = isCustom;
            executable = path.Length > 0 ? path : null;
        }

        await CheckAsync().ConfigureAwait(false);
    }

    /// <summary>Runs <c>vfox --version</c> to check that vfox works.</summary>
    /// <returns>A task that completes when vfox has been probed.</returns>
    public async Task CheckAsync()
    {
        lock (gate)
        {
            availability = VfoxAvailability.Checking;
        }

        NotifyListeners();
        string? found = null;
        Exception? failure = null;
        try
        {
            found = await client.VersionAsync().ConfigureAwait(false);
        }
        catch (Exception exception)
        {
            failure = exception;
        }

        lock (gate)
        {
            version = found;
            error = failure;
            availability = failure switch
            {
                null => VfoxAvailability.Ready,
                VfoxNotFoundException => VfoxAvailability.NotFound,
                _ => VfoxAvailability.Broken,
            };
        }

        if (failure is not null and not (VfoxNotFoundException or VfoxException))
        {
            Trace.TraceError($"checking vfox failed: {failure.Message}");
        }

        NotifyListeners();
    }
}

/// <summary>
/// The vfox configuration (<c>vfox config</c>), editable key by key.
/// </summary>
public sealed class VfoxConfigController : Notifier
{
    private readonly VfoxClient client;
    private readonly Lock gate = new();
    private readonly HashSet<string> saving = [];

    /// <summary>Initializes a new instance of the <see cref="VfoxConfigController"/> class.</summary>
    /// <param name="client">The vfox client.</param>
    public VfoxConfigController(VfoxClient client)
    {
        this.client = client;
        Entries = new AsyncResource<IReadOnlyList<ConfigEntry>>(client.ConfigEntriesAsync);
        Entries.AddListener(NotifyListeners);
    }

    /// <summary>Gets the configuration as it is loaded.</summary>
    public AsyncResource<IReadOnlyList<ConfigEntry>> Entries { get; }

    /// <summary>Returns the value of a key.</summary>
    /// <param name="key">The dotted key.</param>
    /// <returns>The value, or <see langword="null"/> if vfox does not know the key.</returns>
    public string? ValueFor(string key) => Entries.Value?.FirstOrDefault(entry => entry.Key == key)?.Value;

    /// <summary>Determines whether a key is being saved.</summary>
    /// <param name="key">The dotted key.</param>
    /// <returns><see langword="true"/> while vfox saves it.</returns>
    public bool IsSaving(string key)
    {
        lock (gate)
        {
            return saving.Contains(key);
        }
    }

    /// <summary>Sets a key to a new value and reloads the configuration.</summary>
    /// <param name="key">The dotted key.</param>
    /// <param name="newValue">The value.</param>
    /// <returns>A task that fails with a <see cref="VfoxException"/> if vfox rejects the value; the configuration is reloaded either way.</returns>
    public Task SetAsync(string key, string newValue) => SaveAsync(key, () => client.SetConfigAsync(key, newValue));

    /// <summary>Resets a key to its default and reloads the configuration.</summary>
    /// <param name="key">The dotted key.</param>
    /// <returns>A task that completes when vfox is done.</returns>
    public Task ResetAsync(string key) => SaveAsync(key, () => client.UnsetConfigAsync(key));

    /// <inheritdoc/>
    public override void Dispose()
    {
        Entries.Dispose();
        base.Dispose();
    }

    private async Task SaveAsync(string key, Func<Task> action)
    {
        lock (gate)
        {
            saving.Add(key);
        }

        NotifyListeners();
        try
        {
            await action().ConfigureAwait(false);
        }
        finally
        {
            lock (gate)
            {
                saving.Remove(key);
            }

            await Entries.LoadAsync().ConfigureAwait(false);
        }
    }
}

/// <summary>Selects which versions are listed on the "available" tab.</summary>
public enum VersionFilter
{
    /// <summary>Every version.</summary>
    All,

    /// <summary>Versions carrying a note such as <c>LTS</c>.</summary>
    WithNote,

    /// <summary>Versions that are installed already.</summary>
    Installed,
}

/// <summary>
/// The state of the detail view of one installed plugin.
/// </summary>
public sealed class SdkDetailController : Notifier
{
    private readonly Lock gate = new();
    private readonly IDisposable stop;
    private string query = string.Empty;
    private VersionFilter filter = VersionFilter.All;

    /// <summary>Initializes a new instance of the <see cref="SdkDetailController"/> class.</summary>
    /// <param name="client">The vfox client.</param>
    /// <param name="name">The plugin name.</param>
    public SdkDetailController(VfoxClient client, string name)
    {
        Name = name;
        Info = new AsyncResource<PluginInfo>(() => client.PluginInfoAsync(name));
        Versions = new AsyncResource<IReadOnlyList<AvailableVersion>>(() => client.SearchVersionsAsync(name));
        stop = Versions.AddListener(NotifyListeners);
    }

    /// <summary>Gets the plugin name.</summary>
    public string Name { get; }

    /// <summary>Gets the plugin metadata, loaded on demand.</summary>
    public AsyncResource<PluginInfo> Info { get; }

    /// <summary>Gets the installable versions, loaded on demand because the search can be slow.</summary>
    public AsyncResource<IReadOnlyList<AvailableVersion>> Versions { get; }

    /// <summary>Gets the text the available versions are filtered by.</summary>
    public string Query
    {
        get
        {
            lock (gate)
            {
                return query;
            }
        }
    }

    /// <summary>Gets which available versions are shown.</summary>
    public VersionFilter Filter
    {
        get
        {
            lock (gate)
            {
                return filter;
            }
        }
    }

    /// <summary>Sets the filter text.</summary>
    /// <param name="value">The text.</param>
    public void SetQuery(string value)
    {
        bool changed;
        lock (gate)
        {
            changed = value != query;
            query = value;
        }

        if (changed)
        {
            NotifyListeners();
        }
    }

    /// <summary>Sets which available versions are shown.</summary>
    /// <param name="value">The filter.</param>
    public void SetFilter(VersionFilter value)
    {
        bool changed;
        lock (gate)
        {
            changed = value != filter;
            filter = value;
        }

        if (changed)
        {
            NotifyListeners();
        }
    }

    /// <summary>
    /// Returns the available versions matching the query and the filter. The installed versions are passed in
    /// rather than taken from the search output, so that they stay current after installs without searching
    /// again.
    /// </summary>
    /// <param name="installed">The installed versions.</param>
    /// <returns>The versions to show.</returns>
    public IReadOnlyList<AvailableVersion> VisibleVersions(ISet<string> installed)
    {
        string text;
        VersionFilter current;
        lock (gate)
        {
            text = query.Trim().ToLowerInvariant();
            current = filter;
        }

        return [.. (Versions.Value ?? []).Where(version => MatchesFilter(version, current, installed) && MatchesQuery(version, text))];
    }

    /// <inheritdoc/>
    public override void Dispose()
    {
        stop.Dispose();
        Info.Dispose();
        Versions.Dispose();
        base.Dispose();
    }

    private static bool MatchesFilter(AvailableVersion version, VersionFilter filter, ISet<string> installed) => filter switch
    {
        VersionFilter.WithNote => version.HasNote,
        VersionFilter.Installed => installed.Contains(version.Version),
        _ => true,
    };

    private static bool MatchesQuery(AvailableVersion version, string needle) =>
        needle.Length == 0
        || version.Version.Contains(needle, StringComparison.OrdinalIgnoreCase)
        || (version.Note?.Contains(needle, StringComparison.OrdinalIgnoreCase) ?? false);
}
