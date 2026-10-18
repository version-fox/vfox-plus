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

using System.Text;
using VfoxPlus.Core.Processes;
using VfoxPlus.Core.Vfox;

namespace VfoxPlus.Tests.Support;

/// <summary>One recorded invocation of a <see cref="FakeProcessRunner"/>.</summary>
/// <param name="Executable">The executable.</param>
/// <param name="Arguments">The arguments.</param>
/// <param name="WorkingDirectory">The working directory.</param>
/// <param name="Environment">The environment.</param>
/// <param name="Streamed">Whether the process was started rather than run to completion.</param>
internal sealed record ProcessCall(
    string Executable,
    IReadOnlyList<string> Arguments,
    string WorkingDirectory,
    IReadOnlyDictionary<string, string> Environment,
    bool Streamed)
{
    /// <summary>Gets the arguments joined by spaces.</summary>
    public string CommandLine => string.Join(' ', Arguments);
}

/// <summary>
/// A process whose output and exit are driven by the test. Output written before the listener is attached
/// is kept and delivered once it is, like the pipe of a real process.
/// </summary>
internal sealed class FakeRunningProcess : IRunningProcess
{
    private readonly Lock deliver = new();
    private readonly Lock gate = new();
    private readonly TaskCompletionSource<int> exit = new(TaskCreationOptions.RunContinuationsAsynchronously);
    private readonly List<(byte[] Bytes, bool IsError)> backlog = [];
    private IProcessListener? listener;
    private bool killed;

    /// <summary>Gets a value indicating whether <see cref="Kill"/> was called.</summary>
    public bool IsKilled
    {
        get
        {
            lock (gate)
            {
                return killed;
            }
        }
    }

    /// <summary>Creates a process that printed output and exited.</summary>
    /// <param name="output">The output.</param>
    /// <returns>The process.</returns>
    public static FakeRunningProcess Completed(ProcessOutput output)
    {
        FakeRunningProcess process = new();
        process.WriteStdout(output.Stdout);
        process.WriteStderr(output.Stderr);
        process.Exit(output.ExitCode);
        return process;
    }

    /// <summary>Delivers the output written so far and everything written from now on.</summary>
    /// <param name="newListener">The listener.</param>
    public void Attach(IProcessListener newListener)
    {
        lock (deliver)
        {
            List<(byte[] Bytes, bool IsError)> pending;
            lock (gate)
            {
                listener = newListener;
                pending = [.. backlog];
                backlog.Clear();
            }

            foreach ((byte[] bytes, bool isError) in pending)
            {
                Send(newListener, bytes, isError);
            }
        }
    }

    /// <summary>Feeds standard output.</summary>
    /// <param name="text">The text.</param>
    public void WriteStdout(string text) => Write(Encoding.UTF8.GetBytes(text), false);

    /// <summary>Feeds standard output, byte by byte as given.</summary>
    /// <param name="bytes">The bytes.</param>
    public void WriteStdout(byte[] bytes) => Write(bytes, false);

    /// <summary>Feeds standard error.</summary>
    /// <param name="text">The text.</param>
    public void WriteStderr(string text) => Write(Encoding.UTF8.GetBytes(text), true);

    /// <summary>Ends the process.</summary>
    /// <param name="code">The exit code.</param>
    public void Exit(int code) => exit.TrySetResult(code);

    /// <inheritdoc/>
    public Task<int> WaitAsync() => exit.Task;

    /// <inheritdoc/>
    public void Kill()
    {
        lock (gate)
        {
            killed = true;
        }

        Exit(-15);
    }

    private static void Send(IProcessListener listener, byte[] bytes, bool isError)
    {
        if (isError)
        {
            listener.Stderr(bytes);
        }
        else
        {
            listener.Stdout(bytes);
        }
    }

    private void Write(byte[] bytes, bool isError)
    {
        if (bytes.Length == 0)
        {
            return;
        }

        lock (deliver)
        {
            IProcessListener? target;
            lock (gate)
            {
                target = listener;
                if (target is null)
                {
                    backlog.Add((bytes, isError));
                }
            }

            if (target is not null)
            {
                Send(target, bytes, isError);
            }
        }
    }
}

/// <summary>
/// A scripted <see cref="IProcessRunner"/>, safe for use from several threads. Responses are matched by the
/// joined argument list, so tests read like the command lines they expect. Unmatched invocations fail with
/// exit code 127.
/// </summary>
internal sealed class FakeProcessRunner : IProcessRunner
{
    private readonly Lock gate = new();
    private readonly List<ProcessCall> invocations = [];
    private readonly Dictionary<string, Func<ProcessOutput>> responses = [];
    private readonly Dictionary<string, Func<FakeRunningProcess>> streams = [];
    private bool missing;
    private Exception? failure;

    /// <summary>Gets every invocation, in order.</summary>
    public IReadOnlyList<ProcessCall> Invocations
    {
        get
        {
            lock (gate)
            {
                return [.. invocations];
            }
        }
    }

    /// <summary>Creates a runner that answers the read-only commands like the fixtures.</summary>
    /// <remarks>
    /// <c>list</c> and <c>current</c> give maven 3.9.16 (current) and 3.8.9, mvn2 without versions and nodejs
    /// 24.11.1 (current) and 22.21.0.
    /// </remarks>
    /// <returns>The runner.</returns>
    public static FakeProcessRunner ForFixtures() => new FakeProcessRunner()
        .On("--version", Fixtures.Output("version"))
        .On("list", Fixtures.Output("list"))
        .On("current", Fixtures.Output("current"))
        .On("available", Fixtures.Output("available"))
        .On("config --list", Fixtures.Output("config-list"))
        .On("search maven", Fixtures.Output("search-maven-installed"))
        .On("search nodejs", Fixtures.Output("search-nodejs"))
        .On("info nodejs --format " + VfoxOutput.PluginInfoTemplate, Fixtures.Output("info"));

    /// <summary>Responds to <c>vfox &lt;args&gt;</c> with output.</summary>
    /// <param name="args">The arguments, joined by spaces.</param>
    /// <param name="output">The output.</param>
    /// <returns>This runner.</returns>
    public FakeProcessRunner On(string args, ProcessOutput output) => OnDynamic(args, () => output);

    /// <summary>Responds to <c>vfox &lt;args&gt;</c> with the result of a function.</summary>
    /// <param name="args">The arguments, joined by spaces.</param>
    /// <param name="respond">Makes the output.</param>
    /// <returns>This runner.</returns>
    public FakeProcessRunner OnDynamic(string args, Func<ProcessOutput> respond)
    {
        lock (gate)
        {
            responses[args] = respond;
        }

        return this;
    }

    /// <summary>Responds to a started <c>vfox &lt;args&gt;</c> with a process.</summary>
    /// <param name="args">The arguments, joined by spaces.</param>
    /// <param name="create">Makes the process.</param>
    /// <returns>This runner.</returns>
    public FakeProcessRunner OnStart(string args, Func<FakeRunningProcess> create)
    {
        lock (gate)
        {
            streams[args] = create;
        }

        return this;
    }

    /// <summary>Makes every invocation fail as if the executable did not exist, or stops doing so.</summary>
    /// <param name="value">Whether the executable is missing.</param>
    public void SetMissingExecutable(bool value)
    {
        lock (gate)
        {
            missing = value;
        }
    }

    /// <summary>Makes every invocation fail with an error that is not a failure to start the executable.</summary>
    /// <param name="error">The error, or <see langword="null"/> to stop failing.</param>
    public void SetFailure(Exception? error)
    {
        lock (gate)
        {
            failure = error;
        }
    }

    /// <summary>Returns the calls whose arguments start with a prefix.</summary>
    /// <param name="prefix">The prefix.</param>
    /// <returns>The calls.</returns>
    public IReadOnlyList<ProcessCall> Calls(string prefix) =>
        [.. Invocations.Where(call => call.CommandLine.StartsWith(prefix, StringComparison.Ordinal))];

    /// <inheritdoc/>
    public Task<ProcessOutput> RunAsync(string executable, IReadOnlyList<string> arguments, ProcessOptions options)
    {
        Record(executable, arguments, options, streamed: false);
        Func<ProcessOutput>? respond;
        lock (gate)
        {
            responses.TryGetValue(string.Join(' ', arguments), out respond);
        }

        return Task.FromResult(respond?.Invoke() ?? new ProcessOutput(127, string.Empty, "unexpected command"));
    }

    /// <inheritdoc/>
    public IRunningProcess Start(string executable, IReadOnlyList<string> arguments, ProcessOptions options, IProcessListener listener)
    {
        Record(executable, arguments, options, streamed: true);
        string key = string.Join(' ', arguments);
        Func<FakeRunningProcess>? create;
        Func<ProcessOutput>? respond;
        lock (gate)
        {
            streams.TryGetValue(key, out create);
            responses.TryGetValue(key, out respond);
        }

        FakeRunningProcess process = create?.Invoke()
            ?? FakeRunningProcess.Completed(respond?.Invoke() ?? new ProcessOutput(0, string.Empty, string.Empty));
        process.Attach(listener);
        return process;
    }

    private void Record(string executable, IReadOnlyList<string> arguments, ProcessOptions options, bool streamed)
    {
        lock (gate)
        {
            invocations.Add(new ProcessCall(executable, [.. arguments], options.WorkingDirectory, new Dictionary<string, string>(options.Environment), streamed));
            if (failure is not null)
            {
                throw failure;
            }

            if (missing)
            {
                throw new ProcessStartException(executable, new FileNotFoundException("No such file or directory"));
            }
        }
    }
}
