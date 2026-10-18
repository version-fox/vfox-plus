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
/// The kind of state-changing operation the application performs.
/// </summary>
public enum VfoxOperation
{
    /// <summary><c>vfox install --yes &lt;name&gt;@&lt;version&gt;</c>.</summary>
    Install,

    /// <summary><c>vfox uninstall &lt;name&gt;@&lt;version&gt;</c>.</summary>
    Uninstall,

    /// <summary><c>vfox use --global|--project &lt;name&gt;@&lt;version&gt;</c>.</summary>
    Use,

    /// <summary><c>vfox unuse --global|--project &lt;name&gt;</c>.</summary>
    Unuse,

    /// <summary><c>vfox add [--source &lt;url&gt;] [--alias &lt;alias&gt;] &lt;name&gt;</c>.</summary>
    AddPlugin,

    /// <summary><c>vfox remove --yes &lt;name&gt;</c>.</summary>
    RemovePlugin,

    /// <summary><c>vfox update &lt;name&gt;</c>.</summary>
    UpdatePlugin,

    /// <summary><c>vfox update --all</c>.</summary>
    UpdateAllPlugins,
}

/// <summary>
/// A state-changing vfox invocation.
/// </summary>
/// <remarks>
/// Commands are values: they are built by <see cref="VfoxClient"/>, queued and executed by the task queue,
/// and can be retried by running them again.
/// </remarks>
public sealed record VfoxCommand
{
    /// <summary>Gets what the command does.</summary>
    public required VfoxOperation Operation { get; init; }

    /// <summary>Gets the arguments passed to the vfox executable.</summary>
    public required IReadOnlyList<string> Arguments { get; init; }

    /// <summary>Gets the plugin the command acts on, or an empty string.</summary>
    public string Sdk { get; init; } = string.Empty;

    /// <summary>Gets the SDK version the command acts on, or an empty string.</summary>
    public string Version { get; init; } = string.Empty;

    /// <summary>
    /// Gets the directory to run in. An empty string means a neutral directory, so that no project
    /// configuration is picked up.
    /// </summary>
    public string WorkingDirectory { get; init; } = string.Empty;

    /// <summary>
    /// Gets a value indicating whether vfox must believe that it runs inside an activated shell.
    /// </summary>
    /// <remarks>
    /// <c>vfox use</c> refuses to run on Unix without shell integration, but with <c>--global</c> or
    /// <c>--project</c> it only edits <c>.vfox.toml</c> and symlinks, which is exactly what a GUI needs. See
    /// <c>docs/vfox-cli-contract.md</c>.
    /// </remarks>
    public bool RequiresHookEnvironment { get; init; }

    /// <summary>Gets the SDK and version as <c>name@version</c>, or just the name.</summary>
    public string Subject => Sdk.Length > 0 && Version.Length > 0 ? $"{Sdk}@{Version}" : Sdk;

    /// <summary>Gets the command line, as shown in the task output.</summary>
    public string CommandLine => "vfox " + string.Join(' ', Arguments);

    /// <summary>Determines whether another command runs the same program with the same arguments.</summary>
    /// <param name="other">The other command.</param>
    /// <returns><see langword="true"/> if both would do the same.</returns>
    public bool IsSameAs(VfoxCommand other) =>
        WorkingDirectory == other.WorkingDirectory && Arguments.SequenceEqual(other.Arguments);
}

/// <summary>
/// Reports that a vfox invocation failed.
/// </summary>
public sealed class VfoxException : Exception
{
    /// <summary>Initializes a new instance of the <see cref="VfoxException"/> class.</summary>
    /// <param name="message">The short, human readable description.</param>
    /// <param name="arguments">The arguments vfox was started with.</param>
    /// <param name="exitCode">The exit code, or -1 if the process never ran.</param>
    /// <param name="stdout">The standard output of the failed process.</param>
    /// <param name="stderr">The standard error of the failed process.</param>
    public VfoxException(string message, IReadOnlyList<string> arguments, int exitCode = -1, string stdout = "", string stderr = "")
        : base(message)
    {
        Arguments = arguments;
        ExitCode = exitCode;
        Stdout = stdout;
        Stderr = stderr;
    }

    /// <summary>Gets the arguments vfox was started with.</summary>
    public IReadOnlyList<string> Arguments { get; }

    /// <summary>Gets the exit code, or -1 if the process never ran.</summary>
    public int ExitCode { get; }

    /// <summary>Gets the standard output of the failed process.</summary>
    public string Stdout { get; }

    /// <summary>Gets the standard error of the failed process.</summary>
    public string Stderr { get; }
}

/// <summary>
/// Reports that the vfox executable could not be started.
/// </summary>
public sealed class VfoxNotFoundException : Exception
{
    /// <summary>Initializes a new instance of the <see cref="VfoxNotFoundException"/> class.</summary>
    /// <param name="executable">The path or name that could not be executed.</param>
    /// <param name="inner">The error of the operating system.</param>
    public VfoxNotFoundException(string executable, Exception inner)
        : base(inner.Message, inner) => Executable = executable;

    /// <summary>Gets the path or name that could not be executed.</summary>
    public string Executable { get; }
}

/// <summary>
/// Reports that a name, a version, a key or another argument is not acceptable. Its message is shown to the
/// user as it is.
/// </summary>
public sealed class VfoxUsageException : Exception
{
    /// <summary>Initializes a new instance of the <see cref="VfoxUsageException"/> class.</summary>
    /// <param name="message">The message.</param>
    public VfoxUsageException(string message)
        : base(message)
    {
    }
}
