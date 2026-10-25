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

package state

import (
	"container/list"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

// TaskStatus is the life cycle of a VfoxTask:
//
//   - TaskQueued: waiting for earlier tasks to finish;
//   - TaskRunning: the vfox process is running;
//   - TaskSucceeded: vfox exited with code zero;
//   - TaskFailed: vfox could not be started or exited with a non-zero code;
//   - TaskCancelled: the user cancelled the task.
type TaskStatus string

// The task states.
const (
	TaskQueued    TaskStatus = "queued"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// IsActive reports whether a task with this status has not finished yet.
func (s TaskStatus) IsActive() bool { return s == TaskQueued || s == TaskRunning }

const (
	// summaryLines is how many lines of each output stream a task keeps for
	// the summary of a failure, which only reads the last ones.
	summaryLines = 200
	// defaultMaxLogLines is how many log lines a task keeps by default.
	defaultMaxLogLines = 2000
	// defaultMaxFinishedTasks is how many finished tasks the queue keeps by
	// default.
	defaultMaxFinishedTasks = 50
)

// VfoxTask is one execution of a state-changing vfox.VfoxCommand. It is
// created by TaskQueue.
//
// Its fields change on the goroutines that read the output of the process;
// the accessors are safe to call from any goroutine. The status changes only
// through the methods below, each of which is one atomic transition, so that
// a task which was cancelled never turns into a success afterwards.
type VfoxTask struct {
	Notifier
	// ID is the sequential identifier, unique within a TaskQueue.
	ID int
	// Command is the command that is executed.
	Command vfox.VfoxCommand
	// CreatedAt is when the task was queued.
	CreatedAt    time.Time
	maxLogLines  int
	mu           sync.Mutex
	lines        []string
	stdoutLines  []string
	stderrLines  []string
	dropped      int
	process      core.RunningProcess
	status       TaskStatus
	progress     float64
	hasProgress  bool
	startedAt    time.Time
	hasStarted   bool
	finishedAt   time.Time
	hasFinished  bool
	exitCode     int
	hasExitCode  bool
	errorMessage string
	hasError     bool
	// finalizer releases a resource the task owns, such as the private copy
	// of a local plugin file. It runs exactly once, when the task is
	// announced as finished or when the queue is disposed. It is set when
	// the task is enqueued, so that a task which finishes immediately
	// cannot outrun the registration of the resource.
	finalizer func()
}

func newTask(id int, command vfox.VfoxCommand, maxLogLines int) *VfoxTask {
	if maxLogLines <= 0 {
		maxLogLines = defaultMaxLogLines
	}
	return &VfoxTask{
		ID: id, Command: command, CreatedAt: time.Now(),
		maxLogLines: maxLogLines, status: TaskQueued,
	}
}

// Status returns the current status.
func (t *VfoxTask) Status() TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status
}

// Progress returns the download progress between 0 and 1, if vfox reported
// any.
func (t *VfoxTask) Progress() (float64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.progress, t.hasProgress
}

// StartedAt returns when the process started.
func (t *VfoxTask) StartedAt() (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.startedAt, t.hasStarted
}

// FinishedAt returns when the task finished.
func (t *VfoxTask) FinishedAt() (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.finishedAt, t.hasFinished
}

// ExitCode returns the exit code of vfox, if it ran to completion.
func (t *VfoxTask) ExitCode() (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exitCode, t.hasExitCode
}

// ErrorMessage returns a short description of the failure, if the task
// failed.
func (t *VfoxTask) ErrorMessage() (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.errorMessage, t.hasError
}

// Log returns the output lines of the process (standard output and error
// interleaved), without progress bar redraws. Only the most recent lines
// are kept.
func (t *VfoxTask) Log() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.lines...)
}

// DroppedLines returns how many old lines were dropped from Log.
func (t *VfoxTask) DroppedLines() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.dropped
}

// LastLine returns the text that summarises a successful run, usually its
// last line, and whether there is one.
func (t *VfoxTask) LastLine() (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.lines) == 0 {
		return "", false
	}
	return t.lines[len(t.lines)-1], true
}

func (t *VfoxTask) append(line string, isError bool) {
	t.mu.Lock()
	if progress := vfox.ParseDownloadProgress(line); progress >= 0 {
		t.progress, t.hasProgress = progress, true
	} else {
		if isError {
			t.stderrLines = appendTail(t.stderrLines, line, summaryLines)
		} else {
			t.stdoutLines = appendTail(t.stdoutLines, line, summaryLines)
		}
		if len(t.lines) == t.maxLogLines {
			t.dropped++
		}
		t.lines = appendTail(t.lines, line, t.maxLogLines)
	}
	t.mu.Unlock()
	t.NotifyListeners()
}

// appendTail appends line to lines and drops the oldest ones beyond limit.
func appendTail(lines []string, line string, limit int) []string {
	if len(lines) >= limit {
		copy(lines, lines[len(lines)-limit+1:])
		lines = lines[:limit-1]
	}
	return append(lines, line)
}

// summaryLocked returns the error summary of the collected output.
func (t *VfoxTask) summaryLocked() string {
	return vfox.SummarizeError(strings.Join(t.stdoutLines, "\n"), strings.Join(t.stderrLines, "\n"))
}

// begin moves a queued task to running. It reports whether the task was
// queued: a task that was cancelled in the meantime stays cancelled.
func (t *VfoxTask) begin() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status != TaskQueued {
		return false
	}
	t.status = TaskRunning
	t.startedAt, t.hasStarted = time.Now(), true
	return true
}

// cancel moves an active task to cancelled. It returns the status the task
// had, and for a running task the process to kill, if it has started yet.
func (t *VfoxTask) cancel() (TaskStatus, core.RunningProcess) {
	t.mu.Lock()
	defer t.mu.Unlock()
	previous := t.status
	switch previous {
	case TaskQueued:
		t.status = TaskCancelled
		t.finishedAt, t.hasFinished = time.Now(), true
	case TaskRunning:
		t.status = TaskCancelled
		t.errorMessage, t.hasError = "", false
		return previous, t.process
	}
	return previous, nil
}

// attach records the started process. It reports whether the task was
// cancelled while the process was being spawned, in which case the caller
// kills the process.
func (t *VfoxTask) attach(process core.RunningProcess) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.process = process
	return t.status == TaskCancelled
}

// runningProcess returns the process of a running task.
func (t *VfoxTask) runningProcess() core.RunningProcess {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.process
}

// settle records how the process exited and moves the task to its final
// status, unless it was cancelled.
func (t *VfoxTask) settle(code int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.process = nil
	t.exitCode, t.hasExitCode = code, true
	t.finishedAt, t.hasFinished = time.Now(), true
	switch {
	case t.status == TaskCancelled:
	case code == 0:
		t.status = TaskSucceeded
	default:
		t.status = TaskFailed
		message := t.summaryLocked()
		if message == "" {
			message = "vfox exited with code " + strconv.Itoa(code)
		}
		t.errorMessage, t.hasError = message, true
	}
}

// fail moves the task to failed with message, unless it was cancelled.
func (t *VfoxTask) fail(message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.process = nil
	t.finishedAt, t.hasFinished = time.Now(), true
	if t.status != TaskCancelled {
		t.status = TaskFailed
		t.errorMessage, t.hasError = message, true
	}
}

// runFinalizer releases the resource the task owns, if any. It runs at most
// once, even when the queue is disposed while the task is running.
func (t *VfoxTask) runFinalizer() {
	t.mu.Lock()
	finalizer := t.finalizer
	t.finalizer = nil
	t.mu.Unlock()
	if finalizer != nil {
		finalizer()
	}
}

// taskOutput feeds the output of a process into a task, line by line.
type taskOutput struct {
	task   *VfoxTask
	mu     sync.Mutex
	stdout core.TerminalLineSplitter
	stderr core.TerminalLineSplitter
}

func (o *taskOutput) Stdout(chunk []byte) {
	o.mu.Lock()
	lines := o.stdout.Push(chunk)
	o.mu.Unlock()
	for _, line := range lines {
		o.task.append(line, false)
	}
}

func (o *taskOutput) Stderr(chunk []byte) {
	o.mu.Lock()
	lines := o.stderr.Push(chunk)
	o.mu.Unlock()
	for _, line := range lines {
		o.task.append(line, true)
	}
}

func (o *taskOutput) close() {
	o.mu.Lock()
	stdout, stderr := o.stdout.Close(), o.stderr.Close()
	o.mu.Unlock()
	for _, line := range stdout {
		o.task.append(line, false)
	}
	for _, line := range stderr {
		o.task.append(line, true)
	}
}

// TaskQueue runs state-changing vfox commands one after another.
//
// vfox does not lock its data directory, so concurrent installs, uses and
// removals could corrupt `.vfox.toml` or the symlinks. The queue therefore
// runs at most one command at a time, in the order they were enqueued.
type TaskQueue struct {
	Notifier
	cli *vfox.Client
	// MaxLogLines caps how many log lines each task keeps.
	MaxLogLines int
	// MaxFinishedTasks caps how many finished tasks are kept before the
	// oldest are forgotten.
	MaxFinishedTasks int

	mu       sync.Mutex
	all      *list.List // of *VfoxTask, in enqueue order
	finished Emitter[*VfoxTask]
	nextID   int
	running  *VfoxTask
}

// NewTaskQueue creates a queue on cli. A limit of zero takes the default: 2000
// log lines per task and 50 finished tasks.
func NewTaskQueue(cli *vfox.Client, maxLogLines, maxFinishedTasks int) *TaskQueue {
	if maxLogLines <= 0 {
		maxLogLines = defaultMaxLogLines
	}
	if maxFinishedTasks <= 0 {
		maxFinishedTasks = defaultMaxFinishedTasks
	}
	return &TaskQueue{
		cli: cli, MaxLogLines: maxLogLines, MaxFinishedTasks: maxFinishedTasks,
		all: list.New(), nextID: 1,
	}
}

// Tasks returns all known tasks, newest first.
func (q *TaskQueue) Tasks() []*VfoxTask {
	q.mu.Lock()
	defer q.mu.Unlock()
	tasks := make([]*VfoxTask, 0, q.all.Len())
	for element := q.all.Back(); element != nil; element = element.Prev() {
		tasks = append(tasks, element.Value.(*VfoxTask))
	}
	return tasks
}

// ActiveCount returns the number of queued and running tasks.
func (q *TaskQueue) ActiveCount() int {
	count := 0
	for _, task := range q.Tasks() {
		if task.Status().IsActive() {
			count++
		}
	}
	return count
}

// HasFinished reports whether any finished task can be cleared.
func (q *TaskQueue) HasFinished() bool {
	for _, task := range q.Tasks() {
		if !task.Status().IsActive() {
			return true
		}
	}
	return false
}

// OnFinished calls listener with every task once it has finished, failed or
// been cancelled; the result stops it.
func (q *TaskQueue) OnFinished(listener func(*VfoxTask)) Unsubscribe {
	return q.finished.Listen(listener)
}

// Enqueue queues command and returns its task.
//
// If an identical command is already queued or running, its task is returned
// instead of queueing the command twice.
func (q *TaskQueue) Enqueue(command vfox.VfoxCommand) *VfoxTask {
	return q.EnqueueWithFinalizer(command, nil)
}

// EnqueueWithFinalizer queues command like Enqueue, and runs finalizer
// exactly once when the task is announced as finished or when the queue is
// disposed, whichever comes first.
//
// The finalizer is bound to the task before it can start, so that a task
// which finishes immediately cannot outrun the registration of the resource
// it owns. When an identical command is already queued or running, the
// existing task is returned and the finalizer runs immediately, because no
// new task takes ownership of the resource.
func (q *TaskQueue) EnqueueWithFinalizer(command vfox.VfoxCommand, finalizer func()) *VfoxTask {
	q.mu.Lock()
	for element := q.all.Front(); element != nil; element = element.Next() {
		if task := element.Value.(*VfoxTask); task.Status().IsActive() && sameCommand(task.Command, command) {
			q.mu.Unlock()
			if finalizer != nil {
				finalizer()
			}
			return task
		}
	}
	task := newTask(q.nextID, command, q.MaxLogLines)
	if finalizer != nil {
		task.finalizer = finalizer
	}
	q.nextID++
	q.all.PushBack(task)
	q.trimFinishedLocked()
	q.mu.Unlock()
	q.NotifyListeners()
	q.pump()
	return task
}

// Retry queues the command of task again.
func (q *TaskQueue) Retry(task *VfoxTask) *VfoxTask {
	return q.Enqueue(task.Command)
}

// Cancel cancels task: a queued task is dropped, a running process is
// killed. A task that has finished stays as it is.
func (q *TaskQueue) Cancel(task *VfoxTask) {
	previous, process := task.cancel()
	switch previous {
	case TaskQueued:
		q.announce(task)
	case TaskRunning:
		if process != nil {
			process.Kill()
		}
		// The task is announced as finished once the process has exited, in
		// execute. Its status already tells the user interface.
		task.NotifyListeners()
		q.NotifyListeners()
	}
}

// ClearFinished forgets every finished task.
func (q *TaskQueue) ClearFinished() {
	q.mu.Lock()
	for element := q.all.Front(); element != nil; {
		next := element.Next()
		if !element.Value.(*VfoxTask).Status().IsActive() {
			q.all.Remove(element)
		}
		element = next
	}
	q.mu.Unlock()
	q.NotifyListeners()
}

// ActiveTask returns the active task matching the given fields, if any. An
// empty field matches every value.
func (q *TaskQueue) ActiveTask(operation vfox.VfoxOperation, sdk, version string) *VfoxTask {
	for _, task := range q.Tasks() {
		if !task.Status().IsActive() {
			continue
		}
		command := task.Command
		if operation != "" && command.Operation != operation {
			continue
		}
		if sdk != "" && command.Sdk != sdk {
			continue
		}
		if version != "" && command.Version != version {
			continue
		}
		return task
	}
	return nil
}

// pump starts the next queued task, unless one is running.
func (q *TaskQueue) pump() {
	q.mu.Lock()
	if q.running != nil || q.IsDisposed() {
		q.mu.Unlock()
		return
	}
	var upcoming *VfoxTask
	for element := q.all.Front(); element != nil; element = element.Next() {
		// The task is running from now on, so that cancelling it before its
		// process has started kills the process once it has.
		if task := element.Value.(*VfoxTask); task.begin() {
			upcoming = task
			break
		}
	}
	q.running = upcoming
	q.mu.Unlock()
	if upcoming == nil {
		return
	}
	upcoming.NotifyListeners()
	q.NotifyListeners()
	go q.execute(upcoming)
}

// execute runs task to its end, then starts the next one.
func (q *TaskQueue) execute(task *VfoxTask) {
	q.run(task)
	q.announce(task)
	q.mu.Lock()
	q.running = nil
	q.mu.Unlock()
	q.pump()
}

// run starts the process of task and waits for it. Whatever happens, the task
// has its final status afterwards.
func (q *TaskQueue) run(task *VfoxTask) {
	defer func() {
		if recovered := recover(); recovered != nil {
			task.fail(errorString(recovered))
		}
	}()
	output := &taskOutput{task: task}
	process, err := q.cli.Start(task.Command, output)
	if err != nil {
		task.fail(errorMessage(err))
		return
	}
	// Cancelled, or the queue disposed, while the process was being spawned.
	if task.attach(process) || q.IsDisposed() {
		process.Kill()
	}
	code := process.Wait()
	output.close()
	task.settle(code)
}

// announce tells the listeners that task has finished.
func (q *TaskQueue) announce(task *VfoxTask) {
	task.runFinalizer()
	task.NotifyListeners()
	if q.IsDisposed() {
		return
	}
	q.NotifyListeners()
	q.finished.Emit(task)
}

func (q *TaskQueue) trimFinishedLocked() {
	var finished []*list.Element
	for element := q.all.Front(); element != nil; element = element.Next() {
		if !element.Value.(*VfoxTask).Status().IsActive() {
			finished = append(finished, element)
		}
	}
	for _, element := range finished[:max(0, len(finished)-q.MaxFinishedTasks)] {
		q.all.Remove(element)
	}
}

// Dispose kills running processes and releases every resource.
func (q *TaskQueue) Dispose() {
	// The queue is marked first: a process that starts from now on is killed
	// by run, and one that runs is killed here.
	q.Notifier.Dispose()
	q.finished.Close()
	tasks := q.Tasks()
	for _, task := range tasks {
		if process := task.runningProcess(); process != nil {
			process.Kill()
		}
	}
	// A queued task never reaches execute, so its finalizer would never run.
	// runFinalizer runs at most once per task, even for a task that is
	// finishing concurrently.
	for _, task := range tasks {
		task.runFinalizer()
	}
}

func sameCommand(a, b vfox.VfoxCommand) bool {
	if a.WorkingDirectory != b.WorkingDirectory || len(a.Arguments) != len(b.Arguments) {
		return false
	}
	for i := range a.Arguments {
		if a.Arguments[i] != b.Arguments[i] {
			return false
		}
	}
	return true
}

// errorMessage is the short description of why a command could not run.
func errorMessage(err error) string {
	var vfoxErr *vfox.VfoxError
	if errors.As(err, &vfoxErr) {
		return vfoxErr.Message
	}
	return err.Error()
}

func errorString(recovered interface{}) string {
	if err, ok := recovered.(error); ok {
		return err.Error()
	}
	return "unknown failure"
}
