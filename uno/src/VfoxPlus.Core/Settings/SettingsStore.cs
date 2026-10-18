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

namespace VfoxPlus.Core.Settings;

/// <summary>
/// Loads and saves <see cref="AppSettings"/>.
/// </summary>
public interface ISettingsStore
{
    /// <summary>Loads the settings.</summary>
    /// <returns>The settings, or the defaults if none are stored.</returns>
    AppSettings Load();

    /// <summary>Persists settings.</summary>
    /// <param name="settings">The settings.</param>
    void Save(AppSettings settings);
}

/// <summary>
/// Stores settings as JSON in a file.
/// </summary>
/// <remarks>
/// Corrupted files are ignored rather than fatal: the application then starts with the defaults.
/// </remarks>
/// <param name="path">The settings file.</param>
public sealed class JsonFileSettingsStore(string path) : ISettingsStore
{
    /// <summary>
    /// Returns the settings file for an environment: <c>$XDG_CONFIG_HOME/vfox-plus/settings.json</c>, where
    /// <c>XDG_CONFIG_HOME</c> defaults to <c>~/.config</c> as defined by the XDG Base Directory Specification.
    /// </summary>
    /// <param name="environment">The environment.</param>
    /// <param name="fallbackHome">The home directory to use when the environment has none.</param>
    /// <returns>The path.</returns>
    public static string SettingsPath(IReadOnlyDictionary<string, string> environment, string fallbackHome)
    {
        if (environment.TryGetValue("XDG_CONFIG_HOME", out string? xdg) && xdg.StartsWith('/'))
        {
            return $"{xdg}/vfox-plus/settings.json";
        }

        string home = environment.TryGetValue("HOME", out string? value) && value.Length > 0 ? value : fallbackHome;
        return $"{home}/.config/vfox-plus/settings.json";
    }

    /// <inheritdoc/>
    public AppSettings Load()
    {
        try
        {
            return AppSettings.FromJson(File.ReadAllText(path));
        }
        catch (Exception error) when (error is IOException or UnauthorizedAccessException)
        {
            return new AppSettings();
        }
    }

    /// <inheritdoc/>
    public void Save(AppSettings settings)
    {
        // A temporary file is written first and renamed, so that a crash never leaves a truncated settings
        // file behind.
        string directory = Path.GetDirectoryName(path)!;
        Directory.CreateDirectory(directory);
        string temporary = Path.Combine(directory, $".settings-{Guid.NewGuid():N}.json");
        try
        {
            File.WriteAllText(temporary, settings.ToJson());
            File.Move(temporary, path, overwrite: true);
        }
        catch
        {
            File.Delete(temporary);
            throw;
        }
    }
}

/// <summary>
/// Keeps settings in memory. Used by tests and as a fallback. It is safe for use from several threads.
/// </summary>
/// <param name="settings">The settings it starts with.</param>
public sealed class MemorySettingsStore(AppSettings settings) : ISettingsStore
{
    private readonly Lock gate = new();
    private AppSettings stored = settings;
    private int saveCount;

    /// <summary>Gets how often <see cref="Save"/> was called.</summary>
    public int SaveCount
    {
        get
        {
            lock (gate)
            {
                return saveCount;
            }
        }
    }

    /// <inheritdoc/>
    public AppSettings Load()
    {
        lock (gate)
        {
            return stored;
        }
    }

    /// <inheritdoc/>
    public void Save(AppSettings settings)
    {
        lock (gate)
        {
            saveCount++;
            stored = settings;
        }
    }
}
