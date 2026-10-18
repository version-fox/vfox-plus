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

namespace VfoxPlus.Core.Processes;

/// <summary>
/// Everything a finished process wrote, and how it exited.
/// </summary>
/// <param name="ExitCode">The exit code. Zero means success.</param>
/// <param name="Stdout">Everything written to standard output, decoded as UTF-8.</param>
/// <param name="Stderr">Everything written to standard error, decoded as UTF-8.</param>
public sealed record ProcessOutput(int ExitCode, string Stdout, string Stderr);

/// <summary>
/// Where and how a child process runs.
/// </summary>
/// <param name="WorkingDirectory">The working directory of the child.</param>
/// <param name="Environment">
/// The complete environment of the child: the environment of the application is not inherited
/// implicitly, so that callers can remove variables as well as add them.
/// </param>
public sealed record ProcessOptions(string WorkingDirectory, IReadOnlyDictionary<string, string> Environment);

/// <summary>
/// Receives the output of a started process while it runs.
/// </summary>
public interface IProcessListener
{
    /// <summary>Receives a chunk of raw standard output.</summary>
    /// <param name="chunk">The bytes.</param>
    void Stdout(ReadOnlySpan<byte> chunk);

    /// <summary>Receives a chunk of raw standard error.</summary>
    /// <param name="chunk">The bytes.</param>
    void Stderr(ReadOnlySpan<byte> chunk);
}

/// <summary>
/// A child process whose output is consumed while it runs.
/// </summary>
public interface IRunningProcess
{
    /// <summary>
    /// Waits for the process to end and for every chunk of its output to be delivered to the listener.
    /// </summary>
    /// <returns>The exit code.</returns>
    Task<int> WaitAsync();

    /// <summary>Asks the process to terminate with <c>SIGTERM</c>.</summary>
    void Kill();
}

/// <summary>
/// Spawns child processes.
/// </summary>
/// <remarks>
/// The application never spawns processes directly, which keeps the vfox client testable with a fake
/// runner and leaves room for a remote implementation (see <c>docs/architecture.md</c>). Standard input is
/// closed immediately: nothing is ever written to the child, and an open stdin would make an unexpected
/// interactive prompt wait forever.
/// </remarks>
public interface IProcessRunner
{
    /// <summary>Executes a program to completion and collects its output.</summary>
    /// <param name="executable">The path or the name of the program.</param>
    /// <param name="arguments">The arguments.</param>
    /// <param name="options">Where and how to run.</param>
    /// <returns>The output and the exit code.</returns>
    /// <exception cref="ProcessStartException">The program cannot be started.</exception>
    Task<ProcessOutput> RunAsync(string executable, IReadOnlyList<string> arguments, ProcessOptions options);

    /// <summary>Launches a program and returns as soon as it runs.</summary>
    /// <param name="executable">The path or the name of the program.</param>
    /// <param name="arguments">The arguments.</param>
    /// <param name="options">Where and how to run.</param>
    /// <param name="listener">Receives the output while the process runs.</param>
    /// <returns>The running process.</returns>
    /// <exception cref="ProcessStartException">The program cannot be started.</exception>
    IRunningProcess Start(
        string executable,
        IReadOnlyList<string> arguments,
        ProcessOptions options,
        IProcessListener listener);
}

/// <summary>
/// Reports that an executable could not be started, for example because it is missing.
/// </summary>
public sealed class ProcessStartException : Exception
{
    /// <summary>Initializes a new instance of the <see cref="ProcessStartException"/> class.</summary>
    /// <param name="executable">The path or name that could not be executed.</param>
    /// <param name="inner">The error of the operating system.</param>
    public ProcessStartException(string executable, Exception inner)
        : base(inner.Message, inner) => Executable = executable;

    /// <summary>Gets the path or name that could not be executed.</summary>
    public string Executable { get; }
}
