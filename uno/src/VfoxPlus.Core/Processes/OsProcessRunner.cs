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

using System.ComponentModel;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Text;

namespace VfoxPlus.Core.Processes;

/// <summary>
/// The <see cref="IProcessRunner"/> of the desktop, on <see cref="Process"/>.
/// </summary>
public sealed partial class OsProcessRunner : IProcessRunner
{
    private const int Sigterm = 15;
    private const int ChunkSize = 65536;

    /// <inheritdoc/>
    public async Task<ProcessOutput> RunAsync(string executable, IReadOnlyList<string> arguments, ProcessOptions options)
    {
        using Process process = Spawn(executable, arguments, options);
        Task<byte[]> stdout = ReadAllAsync(process.StandardOutput.BaseStream);
        Task<byte[]> stderr = ReadAllAsync(process.StandardError.BaseStream);
        await process.WaitForExitAsync().ConfigureAwait(false);
        return new ProcessOutput(
            process.ExitCode,
            Encoding.UTF8.GetString(await stdout.ConfigureAwait(false)),
            Encoding.UTF8.GetString(await stderr.ConfigureAwait(false)));
    }

    /// <inheritdoc/>
    public IRunningProcess Start(
        string executable,
        IReadOnlyList<string> arguments,
        ProcessOptions options,
        IProcessListener listener)
    {
        Process process = Spawn(executable, arguments, options);
        return new RunningProcess(process, listener);
    }

    [LibraryImport("libc", EntryPoint = "kill", SetLastError = true)]
    private static partial int SendSignal(int pid, int signal);

    private static Process Spawn(string executable, IReadOnlyList<string> arguments, ProcessOptions options)
    {
        ProcessStartInfo info = new(executable)
        {
            UseShellExecute = false,
            RedirectStandardInput = true,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            WorkingDirectory = options.WorkingDirectory,
        };
        foreach (string argument in arguments)
        {
            info.ArgumentList.Add(argument);
        }

        info.Environment.Clear();
        foreach ((string key, string value) in options.Environment)
        {
            info.Environment[key] = value;
        }

        Process process = new() { StartInfo = info };
        try
        {
            process.Start();
        }
        catch (Exception error) when (error is Win32Exception or InvalidOperationException or IOException)
        {
            process.Dispose();
            throw new ProcessStartException(executable, error);
        }

        // Nothing is ever written to the child; see IProcessRunner.
        process.StandardInput.Close();
        return process;
    }

    private static async Task<byte[]> ReadAllAsync(Stream stream)
    {
        using MemoryStream buffer = new();
        await stream.CopyToAsync(buffer).ConfigureAwait(false);
        return buffer.ToArray();
    }

    private static async Task PumpAsync(Stream stream, Action<byte[], int> deliver)
    {
        byte[] buffer = new byte[ChunkSize];
        int count;
        while ((count = await stream.ReadAsync(buffer).ConfigureAwait(false)) > 0)
        {
            deliver(buffer, count);
        }
    }

    private sealed class RunningProcess : IRunningProcess
    {
        private readonly Process process;
        private readonly Task<int> completion;
        private int killed;

        public RunningProcess(Process process, IProcessListener listener)
        {
            this.process = process;
            completion = Task.Run(async () =>
            {
                Task stdout = PumpAsync(process.StandardOutput.BaseStream, (chunk, count) => listener.Stdout(chunk.AsSpan(0, count)));
                Task stderr = PumpAsync(process.StandardError.BaseStream, (chunk, count) => listener.Stderr(chunk.AsSpan(0, count)));
                await Task.WhenAll(stdout, stderr).ConfigureAwait(false);
                await process.WaitForExitAsync().ConfigureAwait(false);
                int code = process.ExitCode;
                process.Dispose();
                return code;
            });
        }

        public Task<int> WaitAsync() => completion;

        public void Kill()
        {
            if (Interlocked.Exchange(ref killed, 1) != 0 || completion.IsCompleted)
            {
                return;
            }

            try
            {
                if (OperatingSystem.IsWindows())
                {
                    process.Kill(entireProcessTree: true);
                }
                else
                {
                    _ = SendSignal(process.Id, Sigterm);
                }
            }
            catch (InvalidOperationException)
            {
                // The process has ended in the meantime.
            }
        }
    }
}
