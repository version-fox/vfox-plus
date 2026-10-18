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
using Xunit;

namespace VfoxPlus.Tests.Core;

public class OsProcessRunnerTests
{
    private static readonly ProcessOptions Options = new("/", new Dictionary<string, string> { ["PATH"] = "/usr/bin:/bin", ["GREETING"] = "hello" });

    private readonly OsProcessRunner runner = new();

    [Fact]
    public async Task RunCollectsOutputAndClosesStandardInput()
    {
        ProcessOutput output = await runner.RunAsync(
            "/bin/sh",
            ["-c", "echo \"$GREETING $HOME\"; echo oops >&2; pwd; read line || echo 'no input'; exit 3"],
            new ProcessOptions("/tmp", new Dictionary<string, string> { ["GREETING"] = "hello" }));
        Assert.Equal(3, output.ExitCode);
        Assert.Equal("hello \n/tmp\nno input\n", output.Stdout);
        Assert.Equal("oops\n", output.Stderr);
    }

    [Fact]
    public async Task RunDecodesMalformedOutput()
    {
        ProcessOutput output = await runner.RunAsync("/bin/sh", ["-c", "printf 'a\\377b'"], Options);
        Assert.Equal("a\ufffdb", output.Stdout);
    }

    [Fact]
    public async Task RunReportsAMissingExecutable()
    {
        ProcessStartException error = await Assert.ThrowsAsync<ProcessStartException>(
            () => runner.RunAsync("/nonexistent/vfox", [], Options));
        Assert.Equal("/nonexistent/vfox", error.Executable);
    }

    [Fact]
    public async Task StartStreamsOutput()
    {
        Collector collected = new();
        IRunningProcess process = runner.Start("/bin/sh", ["-c", "echo one; echo two >&2; echo three"], Options, collected);
        Assert.Equal(0, await process.WaitAsync());
        Assert.Equal(("one\nthree\n", "two\n"), collected.Output());
        process.Kill();
    }

    [Fact]
    public async Task KillTerminatesTheProcessWithSigterm()
    {
        Collector collected = new();
        IRunningProcess process = runner.Start("/bin/sh", ["-c", "echo started; exec sleep 30"], Options, collected);
        DateTime deadline = DateTime.UtcNow.AddSeconds(5);
        while (collected.Output().Stdout.Length == 0 && DateTime.UtcNow < deadline)
        {
            await Task.Delay(10);
        }

        process.Kill();
        Task<int> done = process.WaitAsync();
        Assert.Same(done, await Task.WhenAny(done, Task.Delay(TimeSpan.FromSeconds(5))));
        Assert.Equal(128 + 15, await done);
    }

    [Fact]
    public void StartReportsAMissingExecutable()
    {
        Assert.Throws<ProcessStartException>(() => runner.Start("/nonexistent/vfox", [], Options, new Collector()));
    }

    private sealed class Collector : IProcessListener
    {
        private readonly Lock gate = new();
        private readonly StringBuilder stdout = new();
        private readonly StringBuilder stderr = new();

        public void Stdout(ReadOnlySpan<byte> chunk)
        {
            lock (gate)
            {
                stdout.Append(Encoding.UTF8.GetString(chunk));
            }
        }

        public void Stderr(ReadOnlySpan<byte> chunk)
        {
            lock (gate)
            {
                stderr.Append(Encoding.UTF8.GetString(chunk));
            }
        }

        public (string Stdout, string Stderr) Output()
        {
            lock (gate)
            {
                return (stdout.ToString(), stderr.ToString());
            }
        }
    }
}
