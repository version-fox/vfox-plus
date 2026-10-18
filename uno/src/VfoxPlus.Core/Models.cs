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

namespace VfoxPlus.Core;

/// <summary>
/// An installed vfox plugin together with the SDK versions it manages.
/// </summary>
/// <remarks>
/// vfox does not distinguish between a plugin and the SDK it installs: <c>vfox add nodejs</c> adds the
/// <c>nodejs</c> plugin, and <c>vfox install nodejs@24</c> installs a version of the <c>nodejs</c> SDK
/// through that plugin.
/// </remarks>
/// <param name="Name">The plugin name, for example <c>nodejs</c>.</param>
/// <param name="Versions">The installed versions, in the order reported by vfox (newest first).</param>
/// <param name="Current">The version that is currently in effect, or <see langword="null"/> if none is.</param>
public sealed record InstalledSdk(string Name, IReadOnlyList<string> Versions, string? Current = null)
{
    /// <summary>Gets a value indicating whether a version is currently in effect.</summary>
    public bool HasCurrent => Current is not null;
}

/// <summary>
/// The metadata of an installed plugin, as reported by <c>vfox info &lt;name&gt;</c>.
/// </summary>
/// <param name="Name">The plugin name.</param>
/// <param name="Version">The version of the plugin itself (not of the SDK).</param>
/// <param name="Homepage">The homepage of the plugin, usually a GitHub repository.</param>
/// <param name="InstallPath">The directory in which the SDK versions of this plugin are installed.</param>
/// <param name="Description">A human readable description.</param>
public sealed record PluginInfo(string Name, string Version, string Homepage, string InstallPath, string Description);

/// <summary>
/// A plugin listed in the vfox plugin registry (<c>vfox available</c>).
/// </summary>
/// <param name="Name">The plugin name that is passed to <c>vfox add</c>.</param>
/// <param name="Homepage">The homepage of the plugin.</param>
/// <param name="Official">Whether the plugin is maintained by the version-fox organisation.</param>
public sealed record AvailablePlugin(string Name, string Homepage, bool Official);

/// <summary>
/// A tool that is bundled with an SDK version, such as <c>npm</c> for Node.js.
/// </summary>
/// <param name="Name">The name of the bundled tool.</param>
/// <param name="Version">The version of the bundled tool. It is empty when the plugin omits it.</param>
public sealed record VersionAddition(string Name, string Version)
{
    /// <summary>Formats the tool as shown to users: <c>npm 11.19.1</c>, or just <c>npm</c>.</summary>
    /// <returns>The text.</returns>
    public override string ToString() => Version.Length == 0 ? Name : $"{Name} {Version}";
}

/// <summary>
/// A version that can be installed, as listed by <c>vfox search &lt;name&gt;</c>.
/// </summary>
/// <param name="Version">The version string to pass to <c>vfox install &lt;name&gt;@&lt;version&gt;</c>.</param>
/// <param name="Note">The remark from the plugin, for example <c>LTS</c>, or <see langword="null"/>.</param>
/// <param name="Additions">The tools bundled with this version.</param>
/// <param name="Installed">Whether this version is already installed.</param>
public sealed record AvailableVersion(
    string Version,
    string? Note = null,
    IReadOnlyList<VersionAddition>? Additions = null,
    bool Installed = false)
{
    /// <summary>Gets a value indicating whether the plugin gave a remark.</summary>
    public bool HasNote => Note is not null;

    /// <summary>Gets the tools bundled with this version.</summary>
    public IReadOnlyList<VersionAddition> Tools { get; } = Additions ?? [];
}

/// <summary>
/// One <c>key = value</c> line of <c>vfox config --list</c>.
/// </summary>
/// <param name="Key">The dotted configuration key, for example <c>proxy.url</c>.</param>
/// <param name="Value">The current value as printed by vfox. It is empty when the value is unset.</param>
public sealed record ConfigEntry(string Key, string Value = "");

/// <summary>
/// Where a version selected with <c>vfox use</c> takes effect.
/// </summary>
/// <remarks>
/// vfox also knows a <c>session</c> scope, which only exists inside an activated shell. A desktop
/// application has no shell session, so it is not offered.
/// </remarks>
public enum UseScope
{
    /// <summary>The default for the current user (<c>~/.vfox/.vfox.toml</c>).</summary>
    Global,

    /// <summary>Pinned in the <c>.vfox.toml</c> of a project directory.</summary>
    Project,
}
