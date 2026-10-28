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

package state_test

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/state"
)

func testClient(runner *testsupport.FakeProcessRunner) *vfox.Client {
	config := vfox.DefaultLaunchConfig()
	config.Executable = "vfox"
	return vfox.NewClient(runner, config)
}

// waitFor polls condition until it holds. The state changes on other
// goroutines, so the tests wait for what they assert.
func waitFor(t *testing.T, description string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// mustCommand unwraps the result of a command builder; the commands of the
// tests are valid, so an error is a bug of the test.
func mustCommand(command vfox.VfoxCommand, err error) vfox.VfoxCommand {
	if err != nil {
		panic(err)
	}
	return command
}

// finishedTasks collects the tasks a queue reports as finished.
type finishedTasks struct {
	mu    sync.Mutex
	tasks []*state.VfoxTask
}

func collectFinished(queue *state.TaskQueue) *finishedTasks {
	collected := &finishedTasks{}
	queue.OnFinished(func(task *state.VfoxTask) {
		collected.mu.Lock()
		defer collected.mu.Unlock()
		collected.tasks = append(collected.tasks, task)
	})
	return collected
}

func (f *finishedTasks) snapshot() []*state.VfoxTask {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*state.VfoxTask(nil), f.tasks...)
}

func (f *finishedTasks) count() int { return len(f.snapshot()) }

func TestRunsOneCommandAtATimeInOrder(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	first := testsupport.NewFakeRunningProcess()
	second := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes nodejs@24", func() *testsupport.FakeRunningProcess { return first })
	runner.OnStart("install --yes java@21", func() *testsupport.FakeRunningProcess { return second })
	a := queue.Enqueue(mustCommand(cli.Install("nodejs", "24")))
	b := queue.Enqueue(mustCommand(cli.Install("java", "21")))
	waitFor(t, "the first process", func() bool { return len(runner.Invocations()) == 1 })
	if a.Status() != state.TaskRunning || b.Status() != state.TaskQueued || queue.ActiveCount() != 2 {
		t.Fatalf("got %v %v %d", a.Status(), b.Status(), queue.ActiveCount())
	}
	first.Exit(0)
	waitFor(t, "a to succeed and b to run", func() bool {
		return a.Status() == state.TaskSucceeded && b.Status() == state.TaskRunning
	})
	waitFor(t, "the second process", func() bool { return len(runner.Invocations()) == 2 })
	second.Exit(0)
	waitFor(t, "both to finish", func() bool { return finished.count() == 2 })
	tasks := queue.Tasks()
	if len(tasks) != 2 || tasks[0] != b || tasks[1] != a {
		t.Errorf("newest first, got %v", tasks)
	}
	if got := finished.snapshot(); got[0] != a || got[1] != b {
		t.Errorf("finished out of order")
	}
	if _, ok := a.StartedAt(); !ok {
		t.Error("started")
	}
	if _, ok := a.FinishedAt(); !ok {
		t.Error("finished")
	}
}

func TestDoesNotQueueAnIdenticalActiveCommandTwice(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	runner.OnStart("update --all", testsupport.NewFakeRunningProcess)
	a := queue.Enqueue(cli.UpdateAllPlugins())
	if b := queue.Enqueue(cli.UpdateAllPlugins()); b != a || len(queue.Tasks()) != 1 {
		t.Error("identical commands must share a task")
	}
}

func TestTurnsProgressBarsIntoProgressAndKeepsABoundedLog(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes maven@3.9.16", func() *testsupport.FakeRunningProcess { return process })
	task := queue.Enqueue(mustCommand(cli.Install("maven", "3.9.16")))
	var notified int
	var mu sync.Mutex
	task.AddListener(func() {
		mu.Lock()
		defer mu.Unlock()
		notified++
	})
	process.WriteStdout("Preinstalling maven@3.9.16...\n")
	process.WriteStderr("\rDownloading...  42% [===>  ] (1 MB/s)\r")
	waitFor(t, "progress", func() bool {
		progress, ok := task.Progress()
		return ok && progress == 0.42
	})
	if got := task.Log(); len(got) != 1 || got[0] != "Preinstalling maven@3.9.16..." {
		t.Fatalf("got %q", got)
	}
	process.WriteStdout("one\ntwo\nthree")
	process.Exit(0)
	waitFor(t, "success", func() bool { return task.Status() == state.TaskSucceeded })
	got := task.Log()
	if last, ok := task.LastLine(); len(got) != 3 || got[0] != "one" || got[2] != "three" ||
		!ok || last != "three" || task.DroppedLines() != 1 {
		t.Errorf("got %q %d", got, task.DroppedLines())
	}
	if code, ok := task.ExitCode(); !ok || code != 0 {
		t.Errorf("got %d %v", code, ok)
	}
	mu.Lock()
	defer mu.Unlock()
	if notified == 0 {
		t.Error("expected notifications")
	}
}

func TestTrimsFinishedTaskLogsToTheKeptTail(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 0, 0)
	queue.KeptLogLines = 5
	defer queue.Dispose()
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes maven@3.9.16", func() *testsupport.FakeRunningProcess { return process })
	task := queue.Enqueue(mustCommand(cli.Install("maven", "3.9.16")))
	waitFor(t, "running", func() bool { return task.Status() == state.TaskRunning })
	process.WriteStdout("l0\nl1\nl2\nl3\nl4\nl5\nl6\n")
	waitFor(t, "streamed lines", func() bool { return len(task.Log()) == 7 })
	if got := task.Log(); len(got) != 7 || got[0] != "l0" {
		t.Fatalf("running tasks keep everything, got %q", got)
	}
	process.Exit(0)
	waitFor(t, "success", func() bool { return task.Status() == state.TaskSucceeded })
	if got := task.Log(); len(got) != 5 || got[0] != "l2" || got[4] != "l6" {
		t.Errorf("want the newest 5 lines, got %q", got)
	}
	if task.DroppedLines() != 2 {
		t.Errorf("want 2 dropped lines, got %d", task.DroppedLines())
	}
}

func TestSummarisesFailures(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	runner.On("install --yes maven@3.9.16", testsupport.FixtureOutput("install-failure", 1))
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	task := queue.Enqueue(mustCommand(cli.Install("maven", "3.9.16")))
	waitFor(t, "failure", func() bool { return finished.count() == 1 })
	if task.Status() != state.TaskFailed {
		t.Fatalf("got %v", task.Status())
	}
	// vfox 1.0.13 prints the reason on standard output and the generic
	// "failed to install maven" on standard error.
	message, _ := task.ErrorMessage()
	want := "mark installation in progress: open /tmp/vfox-sb/home/.vfox/cache/maven/.v-3.9.16.installing: permission denied"
	if message != want {
		t.Errorf("got %q", message)
	}
}

func TestReportsExitCodeAndMissingExecutables(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	runner.On("update java", core.ProcessOutput{ExitCode: 4})
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	task := queue.Enqueue(mustCommand(cli.UpdatePlugin("java")))
	waitFor(t, "failure", func() bool { return finished.count() == 1 })
	if message, _ := task.ErrorMessage(); message != "vfox exited with code 4" || task.Status() != state.TaskFailed {
		t.Errorf("got %q", message)
	}
	missing := testsupport.NewFakeProcessRunner()
	missing.SetMissingExecutable(true)
	missingQueue := state.NewTaskQueue(testClient(missing), 3, 2)
	defer missingQueue.Dispose()
	missingFinished := collectFinished(missingQueue)
	failed := missingQueue.Enqueue(mustCommand(cli.UpdatePlugin("java")))
	waitFor(t, "failure", func() bool { return missingFinished.count() == 1 })
	if message, ok := failed.ErrorMessage(); !ok || message != "No such file or directory" || failed.Status() != state.TaskFailed {
		t.Errorf("got %q %v", message, ok)
	}
	if _, ok := failed.ExitCode(); ok {
		t.Error("a process that never ran has no exit code")
	}
}

func TestCancelsQueuedTasksWithoutRunningThem(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("update --all", func() *testsupport.FakeRunningProcess { return process })
	running := queue.Enqueue(cli.UpdateAllPlugins())
	queued := queue.Enqueue(mustCommand(cli.UpdatePlugin("java")))
	queue.Cancel(queued)
	if queued.Status() != state.TaskCancelled || finished.count() != 1 || finished.snapshot()[0] != queued {
		t.Fatalf("got %v %d", queued.Status(), finished.count())
	}
	if _, ok := queued.StartedAt(); ok {
		t.Error("a cancelled queued task never started")
	}
	process.Exit(0)
	waitFor(t, "the running task", func() bool { return running.Status() == state.TaskSucceeded })
	if len(runner.Calls("update java")) != 0 {
		t.Error("a cancelled queued task must not run")
	}
}

func TestKillsCancelledRunningTasks(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes go@1", func() *testsupport.FakeRunningProcess { return process })
	task := queue.Enqueue(mustCommand(cli.Install("go", "1")))
	waitFor(t, "the process", func() bool { return len(runner.Invocations()) == 1 })
	queue.Cancel(task)
	if task.Status() != state.TaskCancelled {
		t.Fatalf("got %v", task.Status())
	}
	waitFor(t, "the process to be killed", process.IsKilled)
	waitFor(t, "the task to be announced", func() bool { return finished.count() == 1 })
	if message, ok := task.ErrorMessage(); ok {
		t.Errorf("a cancelled task did not fail: %q", message)
	}
	queue.Cancel(task)
	if task.Status() != state.TaskCancelled || finished.count() != 1 {
		t.Error("a finished task is not cancelled again")
	}
}

func TestKillsTasksCancelledWhileStarting(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes go@1", func() *testsupport.FakeRunningProcess { return process })
	task := queue.Enqueue(mustCommand(cli.Install("go", "1")))
	// Whether the process has started or not, it ends up killed.
	queue.Cancel(task)
	waitFor(t, "the task to be announced", func() bool { return finished.count() == 1 })
	if !process.IsKilled() {
		t.Error("the process must be killed")
	}
	if task.Status() != state.TaskCancelled {
		t.Errorf("got %v", task.Status())
	}
}

func TestRetriesAndClearsFinishedTasks(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	runner.On("update java", core.ProcessOutput{ExitCode: 1, Stdout: "java plugin not installed\n"})
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	failed := queue.Enqueue(mustCommand(cli.UpdatePlugin("java")))
	waitFor(t, "failure", func() bool { return finished.count() == 1 })
	if message, _ := failed.ErrorMessage(); failed.Status() != state.TaskFailed || message != "java plugin not installed" {
		t.Fatalf("got %v %q", failed.Status(), message)
	}
	runner.On("update java", testsupport.FixtureOutput("update", 0))
	retried := queue.Retry(failed)
	waitFor(t, "success", func() bool { return finished.count() == 2 })
	if retried == failed || retried.Status() != state.TaskSucceeded || !queue.HasFinished() {
		t.Error("a retry is a new task")
	}
	queue.ClearFinished()
	if len(queue.Tasks()) != 0 || queue.HasFinished() {
		t.Error("expected no tasks")
	}
}

func TestForgetsTheOldestFinishedTasks(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	for i, name := range []string{"a", "b", "c", "d"} {
		queue.Enqueue(mustCommand(cli.UpdatePlugin(name)))
		waitFor(t, name, func() bool { return finished.count() == i+1 })
	}
	var names []string
	for _, task := range queue.Tasks() {
		names = append(names, task.Command.Sdk)
	}
	if len(names) != 3 || names[0] != "d" || names[1] != "c" || names[2] != "b" {
		t.Errorf("got %q", names)
	}
}

func TestFindsTheActiveTask(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	runner.OnStart("install --yes go@1", testsupport.NewFakeRunningProcess)
	task := queue.Enqueue(mustCommand(cli.Install("go", "1")))
	if found := queue.ActiveTask(vfox.OperationInstall, "go", "1"); found != task {
		t.Error("expected the task")
	}
	if found := queue.ActiveTask(vfox.OperationUninstall, "go", ""); found != nil {
		t.Error("expected nothing")
	}
	if found := queue.ActiveTask("", "java", ""); found != nil {
		t.Error("expected nothing")
	}
	if found := queue.ActiveTask("", "", "2"); found != nil {
		t.Error("expected nothing")
	}
	if found := queue.ActiveTask("", "", ""); found != task {
		t.Error("expected the task")
	}
}

func TestDisposeKillsRunningProcesses(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes go@1", func() *testsupport.FakeRunningProcess { return process })
	queue := state.NewTaskQueue(cli, 3, 2)
	queue.Enqueue(mustCommand(cli.Install("go", "1")))
	// Disposed while the process is being started, it ends up killed.
	queue.Dispose()
	waitFor(t, "the process", func() bool { return len(runner.Invocations()) == 1 })
	waitFor(t, "the process to be killed", process.IsKilled)
	queue.Dispose()
	if !queue.IsDisposed() {
		t.Error("expected the queue to be disposed")
	}
}

func TestDisposeKillsStartedProcesses(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes go@1", func() *testsupport.FakeRunningProcess { return process })
	queue := state.NewTaskQueue(cli, 3, 2)
	queue.Enqueue(mustCommand(cli.Install("go", "1")))
	waitFor(t, "the process", func() bool { return len(runner.Invocations()) == 1 })
	queue.Dispose()
	waitFor(t, "the process to be killed", process.IsKilled)
	// A disposed queue starts nothing.
	queue.Enqueue(mustCommand(cli.Install("go", "2")))
	time.Sleep(20 * time.Millisecond)
	if len(runner.Invocations()) != 1 {
		t.Errorf("got %d invocations", len(runner.Invocations()))
	}
}

func TestFinalizerRunsWhenTheTaskFinishes(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	// An unmatched streamed command completes immediately.
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	finished := collectFinished(queue)
	var mu sync.Mutex
	calls := 0
	task := queue.EnqueueWithFinalizer(mustCommand(cli.Install("go", "1")), func() {
		mu.Lock()
		defer mu.Unlock()
		calls++
	})
	waitFor(t, "the task to finish", func() bool { return finished.count() == 1 })
	if task.Status() != state.TaskSucceeded {
		t.Fatalf("got %v", task.Status())
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("the finalizer ran %d times, want once", calls)
	}
}

func TestFinalizerRunsImmediatelyWhenTheCommandIsDeduplicated(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	runner.OnStart("update --all", testsupport.NewFakeRunningProcess)
	queue := state.NewTaskQueue(cli, 3, 2)
	defer queue.Dispose()
	first := queue.Enqueue(cli.UpdateAllPlugins())
	var mu sync.Mutex
	calls := 0
	second := queue.EnqueueWithFinalizer(cli.UpdateAllPlugins(), func() {
		mu.Lock()
		defer mu.Unlock()
		calls++
	})
	if second != first {
		t.Error("an identical active command must share its task")
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("the orphaned finalizer ran %d times, want once", calls)
	}
}

func TestFinalizerRunsForQueuedTasksOnDispose(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	runner.OnStart("update --all", testsupport.NewFakeRunningProcess)
	queue := state.NewTaskQueue(cli, 3, 2)
	queue.Enqueue(cli.UpdateAllPlugins())
	var mu sync.Mutex
	calls := 0
	queued := queue.EnqueueWithFinalizer(mustCommand(cli.Install("go", "1")), func() {
		mu.Lock()
		defer mu.Unlock()
		calls++
	})
	if queued.Status() != state.TaskQueued {
		t.Fatalf("got %v", queued.Status())
	}
	queue.Dispose()
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("the queued finalizer ran %d times, want once", calls)
	}
}

func TestFailedTasksKeepTheirFullLogOnDisk(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 0, 0)
	queue.KeptLogLines = 5
	queue.TaskLogDir = t.TempDir()
	defer queue.Dispose()
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes maven@3.9.16", func() *testsupport.FakeRunningProcess { return process })
	task := queue.Enqueue(mustCommand(cli.Install("maven", "3.9.16")))
	waitFor(t, "running", func() bool { return task.Status() == state.TaskRunning })
	var out strings.Builder
	for i := 0; i < 7; i++ {
		fmt.Fprintf(&out, "l%d\n", i)
	}
	process.WriteStdout(out.String())
	process.Exit(1)
	waitFor(t, "failure", func() bool { return task.Status() == state.TaskFailed })
	if got := task.Log(); len(got) != 5 || got[0] != "l2" {
		t.Fatalf("want the newest 5 lines in memory, got %q", got)
	}
	if task.DroppedLines() != 2 {
		t.Fatalf("want 2 dropped lines, got %d", task.DroppedLines())
	}
	path, ok := task.LogFile()
	if !ok {
		t.Fatal("want the full log kept for a failed task")
	}
	content, err := os.ReadFile(path) //nolint:gosec // The path comes from LogFile.
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7; i++ {
		if !strings.Contains(string(content), fmt.Sprintf("l%d\n", i)) {
			t.Fatalf("the full log misses l%d:\n%s", i, content)
		}
	}
}

func TestSuccessfulTasksDeleteTheirSpilledLog(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 0, 0)
	dir := t.TempDir()
	queue.TaskLogDir = dir
	defer queue.Dispose()
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes maven@3.9.16", func() *testsupport.FakeRunningProcess { return process })
	task := queue.Enqueue(mustCommand(cli.Install("maven", "3.9.16")))
	waitFor(t, "running", func() bool { return task.Status() == state.TaskRunning })
	process.WriteStdout("ok\n")
	process.Exit(0)
	waitFor(t, "success", func() bool { return task.Status() == state.TaskSucceeded })
	if _, ok := task.LogFile(); ok {
		t.Error("successful tasks keep no log file")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("want the log deleted, got %v", entries)
	}
}

func TestFailedTaskLogsArePrunedToTheNewest(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := testClient(runner)
	queue := state.NewTaskQueue(cli, 0, 0)
	dir := t.TempDir()
	queue.TaskLogDir = dir
	queue.KeptLogLines = 5
	defer queue.Dispose()
	for i := 0; i < 25; i++ {
		process := testsupport.NewFakeRunningProcess()
		version := fmt.Sprintf("9.%d", i)
		runner.OnStart("install --yes maven@"+version, func() *testsupport.FakeRunningProcess { return process })
		task := queue.Enqueue(mustCommand(cli.Install("maven", version)))
		waitFor(t, "running", func() bool { return task.Status() == state.TaskRunning })
		process.WriteStdout("boom\n")
		process.Exit(1)
		waitFor(t, "failure", func() bool { return task.Status() == state.TaskFailed })
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > 20 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("want at most 20 log files, got %d: %v", len(entries), names)
	}
}
