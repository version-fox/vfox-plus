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

using System.Globalization;
using VfoxPlus.Core.Processes;

namespace VfoxPlus.Core.Vfox;

/// <summary>
/// How vfox processes are launched.
/// </summary>
public sealed record VfoxLaunchConfig
{
    /// <summary>Gets the path to the vfox executable, or a bare name to look up in <c>PATH</c>.</summary>
    public string Executable { get; init; } = VfoxClient.ExecutableName;

    /// <summary>Gets the environment inherited from the application.</summary>
    public IReadOnlyDictionary<string, string> BaseEnvironment { get; init; } = new Dictionary<string, string>();

    /// <summary>
    /// Gets the variables added by the user, for example download mirrors such as
    /// <c>VFOX_NODEJS_MIRROR</c>. They override <see cref="BaseEnvironment"/>.
    /// </summary>
    public IReadOnlyDictionary<string, string> ExtraEnvironment { get; init; } = new Dictionary<string, string>();

    /// <summary>
    /// Gets the working directory for commands that are not tied to a project. vfox reads a project
    /// <c>.vfox.toml</c> (and legacy files such as <c>.nvmrc</c>) from its working directory, so global
    /// queries must run somewhere that cannot contain one. The filesystem root is used by default.
    /// </summary>
    public string NeutralDirectory { get; init; } = "/";
}

/// <summary>
/// Drives the vfox CLI.
/// </summary>
/// <remarks>
/// Read-only queries are executed directly and parsed into models. State-changing operations are returned
/// as <see cref="VfoxCommand"/> values instead, so that the caller can queue them, stream their output and
/// retry them. Every command line built here is documented in <c>docs/vfox-cli-contract.md</c>. A client is
/// safe for use from several threads.
/// </remarks>
public sealed class VfoxClient
{
    /// <summary>The name of the vfox executable, looked up in <c>PATH</c> when no path is known.</summary>
    public const string ExecutableName = "vfox";

    /// <summary>
    /// The variables that tie a process to an activated shell session. They are removed so that a GUI
    /// started from an activated terminal does not act on that terminal's session.
    /// </summary>
    public static readonly IReadOnlyList<string> SessionVariables = ["__VFOX_SHELL", "__VFOX_PID", "__VFOX_CURTMPPATH", "__VFOX_INITIALIZED"];

    private readonly IProcessRunner runner;
    private readonly Lock gate = new();
    private VfoxLaunchConfig config;

    /// <summary>Initializes a new instance of the <see cref="VfoxClient"/> class.</summary>
    /// <param name="runner">Spawns vfox.</param>
    /// <param name="config">How vfox is launched.</param>
    public VfoxClient(IProcessRunner runner, VfoxLaunchConfig? config = null)
    {
        this.runner = runner;
        this.config = config ?? new VfoxLaunchConfig();
    }

    /// <summary>
    /// Gets or sets how vfox is launched. Changing it affects subsequent invocations only, for example after
    /// the user picked a different executable.
    /// </summary>
    public VfoxLaunchConfig LaunchConfig
    {
        get
        {
            lock (gate)
            {
                return config;
            }
        }

        set
        {
            lock (gate)
            {
                config = value;
            }
        }
    }

    /// <summary>Creates the complete environment of a vfox child process.</summary>
    /// <param name="config">How vfox is launched.</param>
    /// <param name="hookShell">Whether vfox must believe that it runs inside an activated shell.</param>
    /// <returns>The environment.</returns>
    public static Dictionary<string, string> Environment(VfoxLaunchConfig config, bool hookShell)
    {
        Dictionary<string, string> environment = [];
        foreach ((string key, string value) in config.BaseEnvironment.Concat(config.ExtraEnvironment))
        {
            if (!SessionVariables.Contains(key))
            {
                environment[key] = value;
            }
        }

        // Disable colours; vfox (through pterm) honours https://no-color.org.
        environment["NO_COLOR"] = "1";
        if (hookShell)
        {
            environment["__VFOX_SHELL"] = "bash";
        }

        return environment;
    }

    /// <summary>Returns the version of vfox, for example <c>1.0.12</c>.</summary>
    /// <returns>The version.</returns>
    public async Task<string> VersionAsync()
    {
        string[] arguments = ["--version"];
        ProcessOutput output = await QueryAsync(arguments, allowFailure: false).ConfigureAwait(false);
        string version = VfoxOutput.ParseVersion(output.Stdout + "\n" + output.Stderr);
        return version.Length > 0
            ? version
            : throw new VfoxException("Unrecognised version output", arguments, stdout: output.Stdout, stderr: output.Stderr);
    }

    /// <summary>
    /// Returns all installed plugins with their installed and current versions. It runs
    /// <c>vfox list</c> and <c>vfox current</c> and merges their results.
    /// </summary>
    /// <returns>The installed SDKs.</returns>
    public async Task<IReadOnlyList<InstalledSdk>> InstalledSdksAsync()
    {
        Task<ProcessOutput> list = QueryAsync(["list"], allowFailure: true);
        Task<ProcessOutput> currentTask = QueryAsync(["current"], allowFailure: false);
        try
        {
            await Task.WhenAll(list, currentTask).ConfigureAwait(false);
        }
        catch (Exception)
        {
            // The failures are thrown below, `vfox current` first.
        }

        IReadOnlyList<CurrentVersion> current = VfoxOutput.ParseCurrent((await currentTask.ConfigureAwait(false)).Stdout);
        ProcessOutput listOutput = await list.ConfigureAwait(false);
        if (listOutput.ExitCode != 0)
        {
            // `vfox list` fails with "you don't have any sdk installed yet" when no plugin is installed;
            // `vfox current` then prints nothing.
            return current.Count == 0 ? [] : throw Failure(["list"], listOutput);
        }

        return VfoxOutput.MergeInstalledSdks(VfoxOutput.ParseInstalledTree(listOutput.Stdout), current);
    }

    /// <summary>Returns the metadata of the installed plugin <paramref name="name"/>.</summary>
    /// <param name="name">The plugin name.</param>
    /// <returns>The metadata.</returns>
    public async Task<PluginInfo> PluginInfoAsync(string name)
    {
        string[] arguments = ["info", VfoxCommands.CheckName(name), "--format", VfoxOutput.PluginInfoTemplate];
        ProcessOutput output = await QueryAsync(arguments, allowFailure: false).ConfigureAwait(false);
        return VfoxOutput.ParsePluginInfo(output.Stdout)
            ?? throw new VfoxException("Unrecognised plugin information", arguments, stdout: output.Stdout, stderr: output.Stderr);
    }

    /// <summary>Returns the directory of an installed version.</summary>
    /// <param name="name">The plugin name.</param>
    /// <param name="version">The version.</param>
    /// <returns>The directory, or <see langword="null"/> if the version is not installed.</returns>
    public async Task<string?> InstallPathAsync(string name, string version)
    {
        ProcessOutput output = await QueryAsync(["info", $"{VfoxCommands.CheckName(name)}@{VfoxCommands.CheckVersion(version)}"], allowFailure: false).ConfigureAwait(false);
        return VfoxOutput.ParseInstallPath(output.Stdout);
    }

    /// <summary>
    /// Returns versions of <paramref name="name"/> that can be installed, newest first. It may take a
    /// while: plugins fetch the list from the network unless vfox has it cached (see the
    /// <c>cache.availableHookDuration</c> setting).
    /// </summary>
    /// <param name="name">The plugin name.</param>
    /// <returns>The versions.</returns>
    public async Task<IReadOnlyList<AvailableVersion>> SearchVersionsAsync(string name)
    {
        ProcessOutput output = await QueryAsync(["search", VfoxCommands.CheckName(name)], allowFailure: false).ConfigureAwait(false);
        return VfoxOutput.ParseSearch(output.Stdout);
    }

    /// <summary>Returns the plugins listed in the vfox registry.</summary>
    /// <returns>The plugins.</returns>
    public async Task<IReadOnlyList<AvailablePlugin>> AvailablePluginsAsync()
    {
        ProcessOutput output = await QueryAsync(["available"], allowFailure: false).ConfigureAwait(false);
        return VfoxOutput.ParseAvailablePlugins(output.Stdout);
    }

    /// <summary>Returns the vfox configuration as <c>key = value</c> entries.</summary>
    /// <returns>The entries.</returns>
    public async Task<IReadOnlyList<ConfigEntry>> ConfigEntriesAsync()
    {
        ProcessOutput output = await QueryAsync(["config", "--list"], allowFailure: false).ConfigureAwait(false);
        return VfoxOutput.ParseConfigList(output.Stdout);
    }

    /// <summary>Sets a configuration key.</summary>
    /// <param name="key">The dotted key.</param>
    /// <param name="value">The new value.</param>
    /// <returns>A task that completes when vfox is done.</returns>
    public async Task SetConfigAsync(string key, string value) =>
        await QueryAsync(["config", VfoxCommands.CheckConfigKey(key), value], allowFailure: false).ConfigureAwait(false);

    /// <summary>Resets a configuration key to its default value.</summary>
    /// <param name="key">The dotted key.</param>
    /// <returns>A task that completes when vfox is done.</returns>
    public async Task UnsetConfigAsync(string key) =>
        await QueryAsync(["config", "--unset", VfoxCommands.CheckConfigKey(key)], allowFailure: false).ConfigureAwait(false);

    /// <summary>Starts a command and returns the running process.</summary>
    /// <param name="command">The command.</param>
    /// <param name="listener">Receives the output while the process runs.</param>
    /// <returns>The running process.</returns>
    /// <exception cref="VfoxNotFoundException">vfox cannot be started.</exception>
    public IRunningProcess Start(VfoxCommand command, IProcessListener listener)
    {
        VfoxLaunchConfig current = LaunchConfig;
        string workingDirectory = command.WorkingDirectory.Length > 0 ? command.WorkingDirectory : current.NeutralDirectory;
        try
        {
            return runner.Start(
                current.Executable,
                command.Arguments,
                new ProcessOptions(workingDirectory, Environment(current, command.RequiresHookEnvironment)),
                listener);
        }
        catch (ProcessStartException error)
        {
            throw new VfoxNotFoundException(current.Executable, error);
        }
    }

    private static VfoxException Failure(IReadOnlyList<string> arguments, ProcessOutput output)
    {
        string summary = VfoxOutput.SummarizeError(output.Stdout, output.Stderr);
        if (summary.Length == 0)
        {
            summary = "vfox exited with code " + output.ExitCode.ToString(CultureInfo.InvariantCulture);
        }

        return new VfoxException(summary, arguments, output.ExitCode, output.Stdout, output.Stderr);
    }

    private async Task<ProcessOutput> QueryAsync(IReadOnlyList<string> arguments, bool allowFailure)
    {
        VfoxLaunchConfig current = LaunchConfig;
        ProcessOutput output;
        try
        {
            output = await runner.RunAsync(
                current.Executable,
                arguments,
                new ProcessOptions(current.NeutralDirectory, Environment(current, hookShell: false))).ConfigureAwait(false);
        }
        catch (ProcessStartException error)
        {
            throw new VfoxNotFoundException(current.Executable, error);
        }

        return output.ExitCode != 0 && !allowFailure ? throw Failure(arguments, output) : output;
    }
}
