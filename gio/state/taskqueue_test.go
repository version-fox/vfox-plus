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
	"sync"
	"testing"
	"time"

	core "github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/core/vfox"
	"github.com/version-fox/vfox-plus/gio/internal/testsupport"
	"github.com/version-fox/vfox-plus/gio/state"
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
