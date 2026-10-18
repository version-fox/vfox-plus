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

using System.Text.RegularExpressions;

namespace VfoxPlus.Core.Vfox;

/// <summary>
/// Builds the state-changing vfox commands, and checks their arguments.
/// </summary>
/// <remarks>
/// Every command line built here is documented in <c>docs/vfox-cli-contract.md</c>. The builders reject
/// names, versions and keys that vfox could mistake for options with a <see cref="VfoxUsageException"/>
/// whose message is shown to the user as it is.
/// </remarks>
public static partial class VfoxCommands
{
    /// <summary>Determines whether a name is accepted as a plugin name or alias.</summary>
    /// <param name="name">The name.</param>
    /// <returns><see langword="true"/> if it is valid.</returns>
    public static bool IsValidPluginName(string name) => NamePattern().IsMatch(name);

    /// <summary>Builds the command that installs a version.</summary>
    /// <param name="name">The plugin name.</param>
    /// <param name="version">The version.</param>
    /// <returns>The command.</returns>
    public static VfoxCommand Install(string name, string version) => new()
    {
        Operation = VfoxOperation.Install,
        Arguments = ["install", "--yes", $"{CheckName(name)}@{CheckVersion(version)}"],
        Sdk = name,
        Version = version,
    };

    /// <summary>Builds the command that uninstalls a version.</summary>
    /// <param name="name">The plugin name.</param>
    /// <param name="version">The version.</param>
    /// <returns>The command.</returns>
    public static VfoxCommand Uninstall(string name, string version) => new()
    {
        Operation = VfoxOperation.Uninstall,
        Arguments = ["uninstall", $"{CheckName(name)}@{CheckVersion(version)}"],
        Sdk = name,
        Version = version,
    };

    /// <summary>Builds the command that makes a version the current version in a scope.</summary>
    /// <param name="name">The plugin name.</param>
    /// <param name="version">The version.</param>
    /// <param name="scope">Where the version takes effect.</param>
    /// <param name="projectDirectory">The project directory, which the project scope requires.</param>
    /// <returns>The command.</returns>
    public static VfoxCommand Use(string name, string version, UseScope scope, string projectDirectory = "") => new()
    {
        Operation = VfoxOperation.Use,
        Arguments = ["use", ScopeFlag(scope), $"{CheckName(name)}@{CheckVersion(version)}"],
        Sdk = name,
        Version = version,
        WorkingDirectory = ScopeDirectory(scope, projectDirectory),
        RequiresHookEnvironment = true,
    };

    /// <summary>Builds the command that removes the current version of a plugin from a scope.</summary>
    /// <param name="name">The plugin name.</param>
    /// <param name="scope">The scope.</param>
    /// <param name="projectDirectory">The project directory, which the project scope requires.</param>
    /// <returns>The command.</returns>
    public static VfoxCommand Unuse(string name, UseScope scope, string projectDirectory = "") => new()
    {
        Operation = VfoxOperation.Unuse,
        Arguments = ["unuse", ScopeFlag(scope), CheckName(name)],
        Sdk = name,
        WorkingDirectory = ScopeDirectory(scope, projectDirectory),
    };

    /// <summary>
    /// Builds the command that adds the plugin <paramref name="name"/> from the registry, or from
    /// <paramref name="source"/> (a URL or a local path) when given.
    /// </summary>
    /// <param name="name">The plugin name.</param>
    /// <param name="source">The source, or an empty string.</param>
    /// <param name="alias">The name to install the plugin under, or an empty string.</param>
    /// <returns>The command.</returns>
    public static VfoxCommand AddPlugin(string name = "", string source = "", string alias = "")
    {
        string trimmedName = name.Trim();
        string trimmedSource = source.Trim();
        string trimmedAlias = alias.Trim();
        if (trimmedName.Length == 0 && trimmedSource.Length == 0)
        {
            throw new VfoxUsageException("A plugin name or a source is required");
        }

        List<string> arguments = ["add"];
        string sdk = string.Empty;
        if (trimmedSource.Length > 0)
        {
            arguments.AddRange(["--source", trimmedSource]);
        }

        if (trimmedAlias.Length > 0)
        {
            arguments.AddRange(["--alias", CheckName(trimmedAlias)]);
            sdk = trimmedAlias;
        }

        if (trimmedName.Length > 0)
        {
            arguments.Add(CheckName(trimmedName));
            if (sdk.Length == 0)
            {
                sdk = trimmedName;
            }
        }

        return new VfoxCommand { Operation = VfoxOperation.AddPlugin, Arguments = arguments, Sdk = sdk };
    }

    /// <summary>Builds the command that removes a plugin together with all of its installed versions.</summary>
    /// <param name="name">The plugin name.</param>
    /// <returns>The command.</returns>
    public static VfoxCommand RemovePlugin(string name) => new()
    {
        Operation = VfoxOperation.RemovePlugin,
        Arguments = ["remove", "--yes", CheckName(name)],
        Sdk = name,
    };

    /// <summary>Builds the command that updates a plugin to its latest version.</summary>
    /// <param name="name">The plugin name.</param>
    /// <returns>The command.</returns>
    public static VfoxCommand UpdatePlugin(string name) => new()
    {
        Operation = VfoxOperation.UpdatePlugin,
        Arguments = ["update", CheckName(name)],
        Sdk = name,
    };

    /// <summary>Builds the command that updates every installed plugin.</summary>
    /// <returns>The command.</returns>
    public static VfoxCommand UpdateAllPlugins() => new()
    {
        Operation = VfoxOperation.UpdateAllPlugins,
        Arguments = ["update", "--all"],
    };

    private static string ScopeFlag(UseScope scope) => scope == UseScope.Global ? "--global" : "--project";

    private static string ScopeDirectory(UseScope scope, string projectDirectory)
    {
        if (scope == UseScope.Global)
        {
            return string.Empty;
        }

        string directory = projectDirectory.Trim();
        return directory.Length > 0 ? directory : throw new VfoxUsageException("A project directory is required for the project scope");
    }

    // Rejects names that vfox could mistake for flags or that contain separators. Arguments never pass
    // through a shell, so this only guards against option injection and typos.
    internal static string CheckName(string name) =>
        IsValidPluginName(name) ? name : throw new VfoxUsageException($"Not a valid plugin name: {name}");

    internal static string CheckVersion(string version) =>
        version.Length == 0 || version.StartsWith('-') || VersionForbidden().IsMatch(version)
            ? throw new VfoxUsageException($"Not a valid version: {version}")
            : version;

    internal static string CheckConfigKey(string key) =>
        ConfigKeyPattern().IsMatch(key) ? key : throw new VfoxUsageException($"Not a valid configuration key: {key}");

    [GeneratedRegex("^[A-Za-z0-9][A-Za-z0-9._+-]*$", RegexOptions.CultureInvariant)]
    private static partial Regex NamePattern();

    [GeneratedRegex("^[A-Za-z][A-Za-z0-9]*(\\.[A-Za-z][A-Za-z0-9]*)*$", RegexOptions.CultureInvariant)]
    private static partial Regex ConfigKeyPattern();

    [GeneratedRegex("[\\s@]", RegexOptions.CultureInvariant)]
    private static partial Regex VersionForbidden();
}
