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

namespace VfoxPlus.Core.Vfox;

/// <summary>
/// A plugin source that is safe to hand to vfox.
/// </summary>
/// <param name="Path">The path or URL to pass to <c>vfox add --source</c>.</param>
/// <param name="CopyDirectory">
/// The directory of the private copy of the original source, or an empty string when
/// <paramref name="Path"/> is the original source.
/// </param>
public sealed record PreparedPluginSource(string Path, string CopyDirectory = "") : IDisposable
{
    /// <summary>Gets a value indicating whether <see cref="Path"/> is a private copy of the original source.</summary>
    public bool IsCopy => CopyDirectory.Length > 0;

    /// <summary>Deletes the private copy, if one was made.</summary>
    public void Dispose()
    {
        if (IsCopy)
        {
            try
            {
                Directory.Delete(CopyDirectory, recursive: true);
            }
            catch (IOException)
            {
                // The copy is in the temporary directory of vfox and does no harm when it stays.
            }
            catch (UnauthorizedAccessException)
            {
            }
        }
    }
}

/// <summary>
/// Prepares plugin sources for <c>vfox add --source</c>.
/// </summary>
public static class PluginSources
{
    /// <summary>Determines whether a source is a URL rather than a local path.</summary>
    /// <param name="source">The source.</param>
    /// <returns><see langword="true"/> for an HTTP or HTTPS URL.</returns>
    public static bool IsRemote(string source) =>
        source.StartsWith("http://", StringComparison.Ordinal) || source.StartsWith("https://", StringComparison.Ordinal);

    /// <summary>
    /// Determines whether vfox would move a source instead of reading it.
    /// </summary>
    /// <remarks>
    /// vfox (still in 1.0.13) renames a local <c>.lua</c> plugin into its temporary directory, which deletes
    /// the user's file and fails when the file is on another filesystem; local <c>.zip</c> archives and URLs
    /// are only read. See <c>docs/vfox-cli-contract.md</c>.
    /// </remarks>
    /// <param name="source">The source.</param>
    /// <returns><see langword="true"/> if vfox moves it.</returns>
    public static bool VfoxMoves(string source) =>
        !IsRemote(source) && source.EndsWith(".lua", StringComparison.OrdinalIgnoreCase);

    /// <summary>
    /// Returns the temporary directory of vfox when it runs with an environment: <c>~/.version-fox/tmp</c> if
    /// the legacy <c>~/.version-fox</c> exists, <c>~/.vfox/tmp</c> otherwise. Without <c>HOME</c>, where vfox
    /// does not run at all, the system temporary directory.
    /// </summary>
    /// <param name="environment">The environment of vfox.</param>
    /// <returns>The directory.</returns>
    public static string VfoxTemporaryDirectory(IReadOnlyDictionary<string, string> environment)
    {
        if (environment.TryGetValue("HOME", out string? home) && home.Length > 0)
        {
            string legacy = $"{home}/.version-fox";
            return Directory.Exists(legacy) ? $"{legacy}/tmp" : $"{home}/.vfox/tmp";
        }

        return Path.GetTempPath();
    }

    /// <summary>
    /// Prepares a source for <c>vfox add --source</c>. A local <c>.lua</c> file is copied into a new
    /// directory under the temporary root, so that vfox moves the copy and the user's file stays where it
    /// is. Other sources are returned as they are. If the copy cannot be made, the original path is
    /// returned and vfox reports the problem.
    /// </summary>
    /// <param name="source">The source: a URL or a path.</param>
    /// <param name="temporaryRoot">
    /// The <see cref="VfoxTemporaryDirectory(IReadOnlyDictionary{string, string})"/>: vfox moves the file with
    /// a rename into a new directory there, which only works on the same filesystem.
    /// </param>
    /// <returns>The source to hand to vfox.</returns>
    public static PreparedPluginSource Prepare(string source, string temporaryRoot)
    {
        PreparedPluginSource original = new(source);
        if (!VfoxMoves(source))
        {
            return original;
        }

        string? directory = null;
        try
        {
            directory = Path.Combine(temporaryRoot, "vfox-plus-plugin-" + Guid.NewGuid().ToString("N"));
            Directory.CreateDirectory(directory);
            string name = Path.GetFileName(source);
            if (name.Length == 0)
            {
                name = "plugin.lua";
            }

            string copy = Path.Combine(directory, name);
            File.Copy(source, copy);
            PreparedPluginSource prepared = new(copy, directory);
            directory = null;
            return prepared;
        }
        catch (Exception error) when (error is IOException or UnauthorizedAccessException)
        {
            return original;
        }
        finally
        {
            if (directory is not null)
            {
                try
                {
                    Directory.Delete(directory, recursive: true);
                }
                catch (IOException)
                {
                }
            }
        }
    }
}
