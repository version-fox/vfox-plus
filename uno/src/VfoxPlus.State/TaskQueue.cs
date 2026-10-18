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
using VfoxPlus.Core;
using VfoxPlus.Core.Processes;
using VfoxPlus.Core.Vfox;

namespace VfoxPlus.State;

/// <summary>The life cycle of a <see cref="VfoxTask"/>.</summary>
public enum VfoxTaskStatus
{
    /// <summary>Waiting for earlier tasks to finish.</summary>
    Queued,

    /// <summary>The vfox process is running.</summary>
    Running,

    /// <summary>vfox exited with code zero.</summary>
    Succeeded,

    /// <summary>vfox could not be started or exited with a non-zero code.</summary>
    Failed,

    /// <summary>The user cancelled the task.</summary>
    Cancelled,
}

/// <summary>Helpers for <see cref="VfoxTaskStatus"/>.</summary>
public static class VfoxTaskStatusExtensions
{
    /// <summary>Determines whether a task with this status has not finished yet.</summary>
    /// <param name="status">The status.</param>
    /// <returns><see langword="true"/> if the task is queued or running.</returns>
    public static bool IsActive(this VfoxTaskStatus status) => status is VfoxTaskStatus.Queued or VfoxTaskStatus.Running;
}

/// <summary>
/// One execution of a state-changing <see cref="VfoxCommand"/>. It is created by <see cref="TaskQueue"/>.
/// </summary>
/// <remarks>
/// Its fields change on the threads that read the output of the process; the accessors are safe to call from
/// any thread. The status changes only through methods that are each one atomic transition, so that a task
/// which was cancelled never turns into a success afterwards.
/// </remarks>
public sealed class VfoxTask : Notifier
{
    private const int SummaryLines = 200;
    private const int DefaultMaxLogLines = 2000;

    private readonly Lock gate = new();
    private readonly int maxLogLines;
    private readonly TaskCompletionSource finished = new(TaskCreationOptions.RunContinuationsAsynchronously);
    private readonly List<string> lines = [];
    private readonly List<string> stdoutLines = [];
    private readonly List<string> stderrLines = [];
    private int dropped;
    private IRunningProcess? process;
    private VfoxTaskStatus status = VfoxTaskStatus.Queued;
    private double? progress;
    private DateTime? startedAt;
    private DateTime? finishedAt;
    private int? exitCode;
    private string? errorMessage;

    internal VfoxTask(int id, VfoxCommand command, int maxLogLines)
    {
        Id = id;
        Command = command;
        CreatedAt = DateTime.Now;
        this.maxLogLines = maxLogLines > 0 ? maxLogLines : DefaultMaxLogLines;
    }

    /// <summary>Gets the sequential identifier, unique within a <see cref="TaskQueue"/>.</summary>
    public int Id { get; }

    /// <summary>Gets the command that is executed.</summary>
    public VfoxCommand Command { get; }

    /// <summary>Gets when the task was queued.</summary>
    public DateTime CreatedAt { get; }

    /// <summary>Gets a task that completes when this task has finished, failed or been cancelled.</summary>
    public Task Completion => finished.Task;

    /// <summary>Gets the current status.</summary>
    public VfoxTaskStatus Status
    {
        get
        {
            lock (gate)
            {
                return status;
            }
        }
    }

    /// <summary>Gets the download progress between 0 and 1, if vfox reported any.</summary>
    public double? Progress
    {
        get
        {
            lock (gate)
            {
                return progress;
            }
        }
    }

    /// <summary>Gets when the process started.</summary>
    public DateTime? StartedAt
    {
        get
        {
            lock (gate)
            {
                return startedAt;
            }
        }
    }

    /// <summary>Gets when the task finished.</summary>
    public DateTime? FinishedAt
    {
        get
        {
            lock (gate)
            {
                return finishedAt;
            }
        }
    }

    /// <summary>Gets the exit code of vfox, if it ran to completion.</summary>
    public int? ExitCode
    {
        get
        {
            lock (gate)
            {
                return exitCode;
            }
        }
    }

    /// <summary>Gets a short description of the failure, if the task failed.</summary>
    public string? ErrorMessage
    {
        get
        {
            lock (gate)
            {
                return errorMessage;
            }
        }
    }

    /// <summary>
    /// Gets the output lines of the process (standard output and error interleaved), without progress bar
    /// redraws. Only the most recent lines are kept.
    /// </summary>
    public IReadOnlyList<string> Log
    {
        get
        {
            lock (gate)
            {
                return [.. lines];
            }
        }
    }

    /// <summary>Gets how many old lines were dropped from <see cref="Log"/>.</summary>
    public int DroppedLines
    {
        get
        {
            lock (gate)
            {
                return dropped;
            }
        }
    }

    /// <summary>Gets the last line of the output, which summarises a successful run.</summary>
    public string? LastLine
    {
        get
        {
            lock (gate)
            {
                return lines.Count == 0 ? null : lines[^1];
            }
        }
    }

    /// <summary>Moves a queued task to running.</summary>
    /// <returns><see langword="true"/> if the task was queued: a task that was cancelled in the meantime stays cancelled.</returns>
    internal bool Begin()
    {
        lock (gate)
        {
            if (status != VfoxTaskStatus.Queued)
            {
                return false;
            }

            status = VfoxTaskStatus.Running;
            startedAt = DateTime.Now;
            return true;
        }
    }

    /// <summary>Moves an active task to cancelled.</summary>
    /// <returns>The status the task had, and for a running task the process to kill, if it has started yet.</returns>
    internal (VfoxTaskStatus Previous, IRunningProcess? Process) Cancel()
    {
        lock (gate)
        {
            VfoxTaskStatus previous = status;
            switch (previous)
            {
                case VfoxTaskStatus.Queued:
                    status = VfoxTaskStatus.Cancelled;
                    finishedAt = DateTime.Now;
                    break;
                case VfoxTaskStatus.Running:
                    status = VfoxTaskStatus.Cancelled;
                    errorMessage = null;
                    return (previous, process);
            }

            return (previous, null);
        }
    }

    /// <summary>Records the started process.</summary>
    /// <param name="started">The process.</param>
    /// <returns><see langword="true"/> if the task was cancelled while the process was being spawned, in which case the caller kills the process.</returns>
    internal bool Attach(IRunningProcess started)
    {
        lock (gate)
        {
            process = started;
            return status == VfoxTaskStatus.Cancelled;
        }
    }

    /// <summary>Gets the process of a running task.</summary>
    /// <returns>The process, or <see langword="null"/>.</returns>
    internal IRunningProcess? RunningProcess()
    {
        lock (gate)
        {
            return process;
        }
    }

    /// <summary>Records how the process exited and moves the task to its final status, unless it was cancelled.</summary>
    /// <param name="code">The exit code.</param>
    internal void Settle(int code)
    {
        lock (gate)
        {
            process = null;
            exitCode = code;
            finishedAt = DateTime.Now;
            if (status == VfoxTaskStatus.Cancelled)
            {
                return;
            }

            if (code == 0)
            {
                status = VfoxTaskStatus.Succeeded;
                return;
            }

            status = VfoxTaskStatus.Failed;
            string message = VfoxOutput.SummarizeError(string.Join('\n', stdoutLines), string.Join('\n', stderrLines));
            errorMessage = message.Length > 0 ? message : "vfox exited with code " + code.ToString(CultureInfo.InvariantCulture);
        }
    }

    /// <summary>Moves the task to failed, unless it was cancelled.</summary>
    /// <param name="message">What went wrong.</param>
    internal void Fail(string message)
    {
        lock (gate)
        {
            process = null;
            finishedAt = DateTime.Now;
            if (status != VfoxTaskStatus.Cancelled)
            {
                status = VfoxTaskStatus.Failed;
                errorMessage = message;
            }
        }
    }

    /// <summary>Tells the listeners and the waiting tasks that this task has finished.</summary>
    internal void Complete()
    {
        NotifyListeners();
        finished.TrySetResult();
    }

    internal void NotifyChanged() => NotifyListeners();

    internal void Append(string line, bool isError)
    {
        lock (gate)
        {
            if (VfoxOutput.ParseDownloadProgress(line) is { } fraction)
            {
                progress = fraction;
            }
            else
            {
                AppendTail(isError ? stderrLines : stdoutLines, line, SummaryLines);
                if (lines.Count == maxLogLines)
                {
                    dropped++;
                }

                AppendTail(lines, line, maxLogLines);
            }
        }

        NotifyListeners();
    }

    // Appends a line and drops the oldest ones beyond the limit.
    private static void AppendTail(List<string> list, string line, int limit)
    {
        if (list.Count >= limit)
        {
            list.RemoveRange(0, list.Count - limit + 1);
        }

        list.Add(line);
    }
}

/// <summary>
/// Runs state-changing vfox commands one after another.
/// </summary>
/// <remarks>
/// vfox does not lock its data directory, so concurrent installs, uses and removals could corrupt
/// <c>.vfox.toml</c> or the symlinks. The queue therefore runs at most one command at a time, in the order
/// they were enqueued.
/// </remarks>
public sealed class TaskQueue : Notifier
{
    private const int DefaultMaxLogLines = 2000;
    private const int DefaultMaxFinishedTasks = 50;

    private readonly VfoxClient client;
    private readonly Lock gate = new();
    private readonly List<VfoxTask> all = [];
    private readonly Emitter<VfoxTask> finished = new();
    private readonly int maxLogLines;
    private readonly int maxFinishedTasks;
    private int nextId = 1;
    private VfoxTask? running;

    /// <summary>Initializes a new instance of the <see cref="TaskQueue"/> class.</summary>
    /// <param name="client">The vfox client.</param>
    /// <param name="maxLogLines">How many log lines each task keeps; 0 for the default of 2000.</param>
    /// <param name="maxFinishedTasks">How many finished tasks are kept before the oldest are forgotten; 0 for the default of 50.</param>
    public TaskQueue(VfoxClient client, int maxLogLines = 0, int maxFinishedTasks = 0)
    {
        this.client = client;
        this.maxLogLines = maxLogLines > 0 ? maxLogLines : DefaultMaxLogLines;
        this.maxFinishedTasks = maxFinishedTasks > 0 ? maxFinishedTasks : DefaultMaxFinishedTasks;
    }

    /// <summary>Gets all known tasks, newest first.</summary>
    public IReadOnlyList<VfoxTask> Tasks
    {
        get
        {
            lock (gate)
            {
                return [.. Enumerable.Reverse(all)];
            }
        }
    }

    /// <summary>Gets the number of queued and running tasks.</summary>
    public int ActiveCount => Tasks.Count(task => task.Status.IsActive());

    /// <summary>Gets a value indicating whether any finished task can be cleared.</summary>
    public bool HasFinished => Tasks.Any(task => !task.Status.IsActive());

    /// <summary>Calls a listener with every task once it has finished, failed or been cancelled.</summary>
    /// <param name="listener">The listener.</param>
    /// <returns>A subscription whose disposal stops the listener.</returns>
    public IDisposable OnFinished(Action<VfoxTask> listener) => finished.Listen(listener);

    /// <summary>
    /// Queues a command. If an identical command is already queued or running, its task is returned instead of
    /// queueing the command twice.
    /// </summary>
    /// <param name="command">The command.</param>
    /// <returns>The task.</returns>
    public VfoxTask Enqueue(VfoxCommand command)
    {
        VfoxTask task;
        lock (gate)
        {
            VfoxTask? existing = all.Find(candidate => candidate.Status.IsActive() && candidate.Command.IsSameAs(command));
            if (existing is not null)
            {
                return existing;
            }

            task = new VfoxTask(nextId++, command, maxLogLines);
            all.Add(task);
            TrimFinished();
        }

        NotifyListeners();
        Pump();
        return task;
    }

    /// <summary>Queues the command of a task again.</summary>
    /// <param name="task">The task.</param>
    /// <returns>The new task.</returns>
    public VfoxTask Retry(VfoxTask task) => Enqueue(task.Command);

    /// <summary>
    /// Cancels a task: a queued task is dropped, a running process is killed. A task that has finished stays as
    /// it is.
    /// </summary>
    /// <param name="task">The task.</param>
    public void Cancel(VfoxTask task)
    {
        (VfoxTaskStatus previous, IRunningProcess? process) = task.Cancel();
        switch (previous)
        {
            case VfoxTaskStatus.Queued:
                Announce(task);
                break;
            case VfoxTaskStatus.Running:
                process?.Kill();

                // The task is announced as finished once the process has exited, in ExecuteAsync. Its status
                // already tells the user interface.
                task.NotifyChanged();
                NotifyListeners();
                break;
        }
    }

    /// <summary>Forgets every finished task.</summary>
    public void ClearFinished()
    {
        lock (gate)
        {
            all.RemoveAll(task => !task.Status.IsActive());
        }

        NotifyListeners();
    }

    /// <summary>Returns the active task matching the given fields, if any. An empty field matches every value.</summary>
    /// <param name="operation">The operation, or <see langword="null"/> for any.</param>
    /// <param name="sdk">The plugin, or an empty string for any.</param>
    /// <param name="version">The version, or an empty string for any.</param>
    /// <returns>The task, or <see langword="null"/>.</returns>
    public VfoxTask? ActiveTask(VfoxOperation? operation = null, string sdk = "", string version = "") =>
        Tasks.FirstOrDefault(task =>
            task.Status.IsActive()
            && (operation is null || task.Command.Operation == operation)
            && (sdk.Length == 0 || task.Command.Sdk == sdk)
            && (version.Length == 0 || task.Command.Version == version));

    /// <summary>Kills running processes and releases every resource.</summary>
    public override void Dispose()
    {
        // The queue is marked first: a process that starts from now on is killed by RunAsync, and one that
        // runs is killed here.
        base.Dispose();
        finished.Close();
        foreach (VfoxTask task in Tasks)
        {
            task.RunningProcess()?.Kill();
        }
    }

    // Starts the next queued task, unless one is running.
    private void Pump()
    {
        VfoxTask? upcoming = null;
        lock (gate)
        {
            if (running is not null || IsDisposed)
            {
                return;
            }

            // The task is running from now on, so that cancelling it before its process has started kills the
            // process once it has.
            upcoming = all.Find(task => task.Begin());
            running = upcoming;
        }

        if (upcoming is null)
        {
            return;
        }

        upcoming.NotifyChanged();
        NotifyListeners();
        _ = Task.Run(() => ExecuteAsync(upcoming));
    }

    // Runs a task to its end, then starts the next one.
    private async Task ExecuteAsync(VfoxTask task)
    {
        await RunAsync(task).ConfigureAwait(false);
        Announce(task);
        lock (gate)
        {
            running = null;
        }

        Pump();
    }

    // Starts the process of a task and waits for it. Whatever happens, the task has its final status afterwards.
    private async Task RunAsync(VfoxTask task)
    {
        try
        {
            TaskOutput output = new(task);
            IRunningProcess process;
            try
            {
                process = client.Start(task.Command, output);
            }
            catch (Exception error) when (error is VfoxNotFoundException or ProcessStartException or VfoxException)
            {
                task.Fail(error.Message);
                return;
            }

            // Cancelled, or the queue disposed, while the process was being spawned.
            if (task.Attach(process) || IsDisposed)
            {
                process.Kill();
            }

            int code = await process.WaitAsync().ConfigureAwait(false);
            output.Close();
            task.Settle(code);
        }
        catch (Exception error)
        {
            task.Fail(error.Message);
        }
    }

    // Tells the listeners that a task has finished.
    private void Announce(VfoxTask task)
    {
        task.Complete();
        if (IsDisposed)
        {
            return;
        }

        NotifyListeners();
        finished.Emit(task);
    }

    private void TrimFinished()
    {
        List<VfoxTask> done = [.. all.Where(task => !task.Status.IsActive())];
        foreach (VfoxTask task in done.Take(Math.Max(0, done.Count - maxFinishedTasks)))
        {
            all.Remove(task);
        }
    }

    // Feeds the output of a process into a task, line by line.
    private sealed class TaskOutput(VfoxTask task) : IProcessListener
    {
        private readonly Lock gate = new();
        private readonly TerminalLineSplitter stdout = new();
        private readonly TerminalLineSplitter stderr = new();

        public void Stdout(ReadOnlySpan<byte> chunk)
        {
            IReadOnlyList<string> lines;
            lock (gate)
            {
                lines = stdout.Push(chunk);
            }

            foreach (string line in lines)
            {
                task.Append(line, isError: false);
            }
        }

        public void Stderr(ReadOnlySpan<byte> chunk)
        {
            IReadOnlyList<string> lines;
            lock (gate)
            {
                lines = stderr.Push(chunk);
            }

            foreach (string line in lines)
            {
                task.Append(line, isError: true);
            }
        }

        public void Close()
        {
            IReadOnlyList<string> out1;
            IReadOnlyList<string> err1;
            lock (gate)
            {
                out1 = stdout.Close();
                err1 = stderr.Close();
            }

            foreach (string line in out1)
            {
                task.Append(line, isError: false);
            }

            foreach (string line in err1)
            {
                task.Append(line, isError: true);
            }
        }
    }
}
