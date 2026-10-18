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

using VfoxPlus.Core.Processes;
using VfoxPlus.Core.Vfox;
using VfoxPlus.State;
using VfoxPlus.Tests.Support;
using Xunit;

namespace VfoxPlus.Tests.State;

public class TaskQueueTests
{
    private static VfoxClient TestClient(FakeProcessRunner runner) => new(runner, new VfoxLaunchConfig { Executable = "vfox" });

    private static Func<FakeRunningProcess> Returning(FakeRunningProcess process) => () => process;

    [Fact]
    public async Task RunsOneCommandAtATimeInOrder()
    {
        FakeProcessRunner runner = new();
        VfoxClient client = TestClient(runner);
        using TaskQueue queue = new(client, 3, 2);
        Finished finished = new(queue);
        FakeRunningProcess first = new();
        FakeRunningProcess second = new();
        runner.OnStart("install --yes nodejs@24", Returning(first));
        runner.OnStart("install --yes java@21", Returning(second));
        VfoxTask a = queue.Enqueue(VfoxCommands.Install("nodejs", "24"));
        VfoxTask b = queue.Enqueue(VfoxCommands.Install("java", "21"));
        await Wait.UntilAsync("the first process", () => runner.Invocations.Count == 1);
        Assert.Equal(VfoxTaskStatus.Running, a.Status);
        Assert.Equal(VfoxTaskStatus.Queued, b.Status);
        Assert.Equal(2, queue.ActiveCount);

        first.Exit(0);
        await Wait.UntilAsync("a to succeed and b to run", () => a.Status == VfoxTaskStatus.Succeeded && b.Status == VfoxTaskStatus.Running);
        await Wait.UntilAsync("the second process", () => runner.Invocations.Count == 2);
        second.Exit(0);
        await Wait.UntilAsync("both to finish", () => finished.Count == 2);

        Assert.Equal([b, a], queue.Tasks);
        Assert.Equal([a, b], finished.Snapshot());
        Assert.NotNull(a.StartedAt);
        Assert.NotNull(a.FinishedAt);
        await a.Completion;
    }

    [Fact]
    public void DoesNotQueueAnIdenticalActiveCommandTwice()
    {
        FakeProcessRunner runner = new();
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        runner.OnStart("update --all", () => new FakeRunningProcess());
        VfoxTask a = queue.Enqueue(VfoxCommands.UpdateAllPlugins());
        Assert.Same(a, queue.Enqueue(VfoxCommands.UpdateAllPlugins()));
        Assert.Single(queue.Tasks);
    }

    [Fact]
    public async Task TurnsProgressBarsIntoProgressAndKeepsABoundedLog()
    {
        FakeProcessRunner runner = new();
        VfoxClient client = TestClient(runner);
        using TaskQueue queue = new(client, 3, 2);
        FakeRunningProcess process = new();
        runner.OnStart("install --yes maven@3.9.16", Returning(process));
        VfoxTask task = queue.Enqueue(VfoxCommands.Install("maven", "3.9.16"));
        int notified = 0;
        task.AddListener(() => Interlocked.Increment(ref notified));

        process.WriteStdout("Preinstalling maven@3.9.16...\n");
        process.WriteStderr("\rDownloading...  42% [===>  ] (1 MB/s)\r");
        await Wait.UntilAsync("progress", () => task.Progress == 0.42);
        Assert.Equal(["Preinstalling maven@3.9.16..."], task.Log);

        process.WriteStdout("one\ntwo\nthree");
        process.Exit(0);
        await Wait.UntilAsync("success", () => task.Status == VfoxTaskStatus.Succeeded);
        Assert.Equal(3, task.Log.Count);
        Assert.Equal("one", task.Log[0]);
        Assert.Equal("three", task.Log[2]);
        Assert.Equal("three", task.LastLine);
        Assert.Equal(1, task.DroppedLines);
        Assert.Equal(0, task.ExitCode);
        Assert.True(Volatile.Read(ref notified) > 0);
    }

    [Fact]
    public async Task SummarisesFailures()
    {
        FakeProcessRunner runner = new();
        runner.On("install --yes maven@3.9.16", Fixtures.Output("install-failure", 1));
        VfoxClient client = TestClient(runner);
        using TaskQueue queue = new(client, 3, 2);
        Finished finished = new(queue);
        VfoxTask task = queue.Enqueue(VfoxCommands.Install("maven", "3.9.16"));
        await Wait.UntilAsync("failure", () => finished.Count == 1);
        Assert.Equal(VfoxTaskStatus.Failed, task.Status);

        // vfox 1.0.13 prints the reason on standard output and the generic "failed to install maven" on
        // standard error.
        Assert.Equal(
            "mark installation in progress: open /tmp/vfox-sb/home/.vfox/cache/maven/.v-3.9.16.installing: permission denied",
            task.ErrorMessage);
    }

    [Fact]
    public async Task ReportsTheExitCodeAndMissingExecutables()
    {
        FakeProcessRunner runner = new();
        runner.On("update java", new ProcessOutput(4, string.Empty, string.Empty));
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        Finished finished = new(queue);
        VfoxTask task = queue.Enqueue(VfoxCommands.UpdatePlugin("java"));
        await Wait.UntilAsync("failure", () => finished.Count == 1);
        Assert.Equal("vfox exited with code 4", task.ErrorMessage);
        Assert.Equal(VfoxTaskStatus.Failed, task.Status);

        FakeProcessRunner missing = new();
        missing.SetMissingExecutable(true);
        using TaskQueue missingQueue = new(TestClient(missing), 3, 2);
        Finished missingFinished = new(missingQueue);
        VfoxTask failed = missingQueue.Enqueue(VfoxCommands.UpdatePlugin("java"));
        await Wait.UntilAsync("failure", () => missingFinished.Count == 1);
        Assert.Equal("No such file or directory", failed.ErrorMessage);
        Assert.Equal(VfoxTaskStatus.Failed, failed.Status);
        Assert.Null(failed.ExitCode);
    }

    [Fact]
    public async Task CancelsQueuedTasksWithoutRunningThem()
    {
        FakeProcessRunner runner = new();
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        Finished finished = new(queue);
        FakeRunningProcess process = new();
        runner.OnStart("update --all", Returning(process));
        VfoxTask running = queue.Enqueue(VfoxCommands.UpdateAllPlugins());
        VfoxTask queued = queue.Enqueue(VfoxCommands.UpdatePlugin("java"));
        queue.Cancel(queued);
        Assert.Equal(VfoxTaskStatus.Cancelled, queued.Status);
        Assert.Equal([queued], finished.Snapshot());
        Assert.Null(queued.StartedAt);

        process.Exit(0);
        await Wait.UntilAsync("the running task", () => running.Status == VfoxTaskStatus.Succeeded);
        Assert.Empty(runner.Calls("update java"));
    }

    [Fact]
    public async Task KillsCancelledRunningTasks()
    {
        FakeProcessRunner runner = new();
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        Finished finished = new(queue);
        FakeRunningProcess process = new();
        runner.OnStart("install --yes go@1", Returning(process));
        VfoxTask task = queue.Enqueue(VfoxCommands.Install("go", "1"));
        await Wait.UntilAsync("the process", () => runner.Invocations.Count == 1);
        queue.Cancel(task);
        Assert.Equal(VfoxTaskStatus.Cancelled, task.Status);
        await Wait.UntilAsync("the process to be killed", () => process.IsKilled);
        await Wait.UntilAsync("the task to be announced", () => finished.Count == 1);
        Assert.Null(task.ErrorMessage);

        queue.Cancel(task);
        Assert.Equal(VfoxTaskStatus.Cancelled, task.Status);
        Assert.Equal(1, finished.Count);
    }

    [Fact]
    public async Task KillsTasksCancelledWhileStarting()
    {
        FakeProcessRunner runner = new();
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        Finished finished = new(queue);
        FakeRunningProcess process = new();
        runner.OnStart("install --yes go@1", Returning(process));
        VfoxTask task = queue.Enqueue(VfoxCommands.Install("go", "1"));

        // Whether the process has started or not, it ends up killed.
        queue.Cancel(task);
        await Wait.UntilAsync("the task to be announced", () => finished.Count == 1);
        Assert.True(process.IsKilled);
        Assert.Equal(VfoxTaskStatus.Cancelled, task.Status);
    }

    [Fact]
    public async Task RetriesAndClearsFinishedTasks()
    {
        FakeProcessRunner runner = new();
        runner.On("update java", new ProcessOutput(1, "java plugin not installed\n", string.Empty));
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        Finished finished = new(queue);
        VfoxTask failed = queue.Enqueue(VfoxCommands.UpdatePlugin("java"));
        await Wait.UntilAsync("failure", () => finished.Count == 1);
        Assert.Equal(VfoxTaskStatus.Failed, failed.Status);
        Assert.Equal("java plugin not installed", failed.ErrorMessage);

        runner.On("update java", Fixtures.Output("update"));
        VfoxTask retried = queue.Retry(failed);
        await Wait.UntilAsync("success", () => finished.Count == 2);
        Assert.NotSame(failed, retried);
        Assert.Equal(VfoxTaskStatus.Succeeded, retried.Status);
        Assert.True(queue.HasFinished);

        queue.ClearFinished();
        Assert.Empty(queue.Tasks);
        Assert.False(queue.HasFinished);
    }

    [Fact]
    public async Task ForgetsTheOldestFinishedTasks()
    {
        FakeProcessRunner runner = new();
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        Finished finished = new(queue);
        int count = 0;
        foreach (string name in new[] { "a", "b", "c", "d" })
        {
            queue.Enqueue(VfoxCommands.UpdatePlugin(name));
            count++;
            await Wait.UntilAsync(name, () => finished.Count == count);
        }

        Assert.Equal(["d", "c", "b"], queue.Tasks.Select(task => task.Command.Sdk));
    }

    [Fact]
    public void FindsTheActiveTask()
    {
        FakeProcessRunner runner = new();
        using TaskQueue queue = new(TestClient(runner), 3, 2);
        runner.OnStart("install --yes go@1", () => new FakeRunningProcess());
        VfoxTask task = queue.Enqueue(VfoxCommands.Install("go", "1"));
        Assert.Same(task, queue.ActiveTask(VfoxOperation.Install, "go", "1"));
        Assert.Null(queue.ActiveTask(VfoxOperation.Uninstall, "go"));
        Assert.Null(queue.ActiveTask(sdk: "java"));
        Assert.Null(queue.ActiveTask(version: "2"));
        Assert.Same(task, queue.ActiveTask());
    }

    [Fact]
    public async Task DisposeKillsProcessesStartedAfterwards()
    {
        FakeProcessRunner runner = new();
        FakeRunningProcess process = new();
        runner.OnStart("install --yes go@1", Returning(process));
        TaskQueue queue = new(TestClient(runner), 3, 2);
        queue.Enqueue(VfoxCommands.Install("go", "1"));

        // Disposed while the process is being started, it ends up killed.
        queue.Dispose();
        await Wait.UntilAsync("the process", () => runner.Invocations.Count == 1);
        await Wait.UntilAsync("the process to be killed", () => process.IsKilled);
        queue.Dispose();
        Assert.True(queue.IsDisposed);
    }

    [Fact]
    public async Task DisposeKillsStartedProcessesAndStartsNothingMore()
    {
        FakeProcessRunner runner = new();
        FakeRunningProcess process = new();
        runner.OnStart("install --yes go@1", Returning(process));
        TaskQueue queue = new(TestClient(runner), 3, 2);
        queue.Enqueue(VfoxCommands.Install("go", "1"));
        await Wait.UntilAsync("the process", () => runner.Invocations.Count == 1);
        queue.Dispose();
        await Wait.UntilAsync("the process to be killed", () => process.IsKilled);

        // A disposed queue starts nothing.
        queue.Enqueue(VfoxCommands.Install("go", "2"));
        await Task.Delay(20);
        Assert.Single(runner.Invocations);
    }

    private sealed class Finished
    {
        private readonly Lock gate = new();
        private readonly List<VfoxTask> tasks = [];

        public Finished(TaskQueue queue) =>
            queue.OnFinished(task =>
            {
                lock (gate)
                {
                    tasks.Add(task);
                }
            });

        public int Count => Snapshot().Count;

        public List<VfoxTask> Snapshot()
        {
            lock (gate)
            {
                return [.. tasks];
            }
        }
    }
}
