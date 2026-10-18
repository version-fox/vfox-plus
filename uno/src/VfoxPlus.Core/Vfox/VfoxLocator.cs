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
/// Looks for the vfox executable in <c>PATH</c> and in common install locations.
/// </summary>
/// <remarks>
/// Applications started from a desktop launcher do not source the user's shell profile, so directories
/// such as <c>~/.local/bin</c> are often missing from their <c>PATH</c>. The well-known install locations of
/// the official installers and package managers are therefore searched as well.
/// </remarks>
public sealed class VfoxLocator
{
    private readonly IReadOnlyDictionary<string, string> environment;
    private readonly Func<string, bool> isExecutable;

    /// <summary>Initializes a new instance of the <see cref="VfoxLocator"/> class.</summary>
    /// <param name="environment">The environment whose <c>PATH</c> and <c>HOME</c> are searched.</param>
    /// <param name="isExecutable">Tells whether a path is executable; a stub in tests.</param>
    public VfoxLocator(IReadOnlyDictionary<string, string> environment, Func<string, bool>? isExecutable = null)
    {
        this.environment = environment;
        this.isExecutable = isExecutable ?? IsExecutableFile;
    }

    /// <summary>Gets the directories that are searched in addition to <c>PATH</c>, in order.</summary>
    public IReadOnlyList<string> FallbackDirectories
    {
        get
        {
            List<string> directories = ["/usr/local/bin", "/usr/bin"];
            if (environment.TryGetValue("HOME", out string? home) && home.Length > 0)
            {
                directories.AddRange([$"{home}/.local/bin", $"{home}/bin", $"{home}/.linuxbrew/bin"]);
            }

            directories.AddRange(["/home/linuxbrew/.linuxbrew/bin", "/snap/bin"]);
            return directories;
        }
    }

    /// <summary>Gets all directories that are searched, without duplicates.</summary>
    public IReadOnlyList<string> SearchDirectories
    {
        get
        {
            environment.TryGetValue("PATH", out string? path);
            return [.. (path ?? string.Empty).Split(':', StringSplitOptions.RemoveEmptyEntries).Concat(FallbackDirectories).Distinct()];
        }
    }

    /// <summary>Determines whether a path is an existing regular file that anybody may execute.</summary>
    /// <param name="path">The path.</param>
    /// <returns><see langword="true"/> if it can be executed.</returns>
    public static bool IsExecutableFile(string path)
    {
        try
        {
            FileInfo file = new(path);
            if (!file.Exists || file.Attributes.HasFlag(FileAttributes.Directory))
            {
                return false;
            }

            const UnixFileMode anyExecute = UnixFileMode.UserExecute | UnixFileMode.GroupExecute | UnixFileMode.OtherExecute;
            return (file.UnixFileMode & anyExecute) != 0;
        }
        catch (IOException)
        {
            return false;
        }
        catch (UnauthorizedAccessException)
        {
            return false;
        }
    }

    /// <summary>Returns the path of the first vfox executable found.</summary>
    /// <returns>The path, or an empty string.</returns>
    public string Locate()
    {
        foreach (string directory in SearchDirectories)
        {
            string candidate = $"{directory}/{VfoxClient.ExecutableName}";
            if (isExecutable(candidate))
            {
                return candidate;
            }
        }

        return string.Empty;
    }
}
