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

// Package testsupport holds the test doubles shared by the unit tests.
package testsupport

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

// SharedDir is the shared resources of the repository.
func SharedDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "shared")
}

// FixturesVersion is the captured vfox outputs in shared/fixtures that the
// tests use by default.
const FixturesVersion = "1.0.13"

// ReadCapture reads a capture of shared/fixtures, in which control characters
// are written as `\xHH` and backslashes as `\\`.
func ReadCapture(path string) string {
	content, err := os.ReadFile(path) //nolint:gosec // The path is a fixture of the tests.
	if err != nil {
		panic(err)
	}
	var out strings.Builder
	text := string(content)
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) {
			if text[i+1] == '\\' {
				out.WriteByte('\\')
				i++
			} else if text[i+1] == 'x' && i+3 < len(text) {
				if code, err := strconv.ParseUint(text[i+2:i+4], 16, 8); err == nil {
					out.WriteByte(byte(code))
					i += 3
				} else {
					out.WriteByte(text[i])
				}
			} else {
				out.WriteByte(text[i])
			}
		} else {
			out.WriteByte(text[i])
		}
	}
	return out.String()
}

// FixtureOutput returns the result of the fixture name, with exitCode.
func FixtureOutput(name string, exitCode int) core.ProcessOutput {
	base := filepath.Join(SharedDir(), "fixtures", "vfox-"+FixturesVersion, name)
	return core.ProcessOutput{
		ExitCode: exitCode,
		Stdout:   ReadCapture(base + ".stdout"),
		Stderr:   ReadCapture(base + ".stderr"),
	}
}

// ProcessCall is one recorded invocation of a FakeProcessRunner.
type ProcessCall struct {
	Executable       string
	Arguments        []string
	WorkingDirectory string
	Environment      map[string]string
	Streamed         bool
}

// chunk is output that a FakeRunningProcess wrote before its listener was
// attached.
type chunk struct {
	text    string
	isError bool
}

// FakeRunningProcess is a RunningProcess whose output and exit are driven by
// the test.
//
// Output written before the listener is attached is kept and delivered once
// it is, like the pipe of a real process.
type FakeRunningProcess struct {
	deliver  sync.Mutex // serialises the delivery of output
	mu       sync.Mutex
	listener core.ProcessListener
	backlog  []chunk
	exited   bool
	code     int
	killed   bool
	done     chan struct{}
}

// NewFakeRunningProcess creates a process that is still running.
func NewFakeRunningProcess() *FakeRunningProcess {
	return &FakeRunningProcess{done: make(chan struct{})}
}

// CompletedProcess returns a process that printed output and exited.
func CompletedProcess(output core.ProcessOutput) *FakeRunningProcess {
	process := NewFakeRunningProcess()
	process.WriteStdout(output.Stdout)
	process.WriteStderr(output.Stderr)
	process.Exit(output.ExitCode)
	return process
}

// Attach delivers the output written so far to listener and everything the
// process writes from now on.
func (p *FakeRunningProcess) Attach(listener core.ProcessListener) {
	p.deliver.Lock()
	defer p.deliver.Unlock()
	p.mu.Lock()
	p.listener = listener
	backlog := p.backlog
	p.backlog = nil
	p.mu.Unlock()
	for _, item := range backlog {
		send(listener, item)
	}
}

func send(listener core.ProcessListener, item chunk) {
	if item.isError {
		listener.Stderr([]byte(item.text))
	} else {
		listener.Stdout([]byte(item.text))
	}
}

func (p *FakeRunningProcess) write(item chunk) {
	if item.text == "" {
		return
	}
	p.deliver.Lock()
	defer p.deliver.Unlock()
	p.mu.Lock()
	listener := p.listener
	if listener == nil {
		p.backlog = append(p.backlog, item)
	}
	p.mu.Unlock()
	if listener != nil {
		send(listener, item)
	}
}

// WriteStdout feeds standard output.
func (p *FakeRunningProcess) WriteStdout(text string) { p.write(chunk{text: text}) }

// WriteStderr feeds standard error.
func (p *FakeRunningProcess) WriteStderr(text string) { p.write(chunk{text: text, isError: true}) }

// Exit ends the process with code.
func (p *FakeRunningProcess) Exit(code int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.exited {
		p.exited = true
		p.code = code
		close(p.done)
	}
}

// Wait implements core.RunningProcess.
func (p *FakeRunningProcess) Wait() int {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.code
}

// Kill implements core.RunningProcess; the process exits as if it had
// received SIGTERM.
func (p *FakeRunningProcess) Kill() {
	p.mu.Lock()
	p.killed = true
	p.mu.Unlock()
	p.Exit(-15)
}

// IsKilled reports whether Kill was called.
func (p *FakeRunningProcess) IsKilled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.killed
}

// FakeProcessRunner is a scripted core.ProcessRunner. It is safe for use from
// several goroutines.
//
// Responses are matched by the joined argument list, so tests read like the
// command lines they expect. Unmatched invocations fail with exit code 127.
type FakeProcessRunner struct {
	mu          sync.Mutex
	invocations []ProcessCall
	missing     bool
	failure     error
	responses   map[string]func() core.ProcessOutput
	streams     map[string]func() *FakeRunningProcess
}

// NewFakeProcessRunner creates a scripted runner.
func NewFakeProcessRunner() *FakeProcessRunner {
	return &FakeProcessRunner{
		responses: make(map[string]func() core.ProcessOutput),
		streams:   make(map[string]func() *FakeRunningProcess),
	}
}

// On responds to `vfox <args>` with output.
func (r *FakeProcessRunner) On(args string, output core.ProcessOutput) *FakeProcessRunner {
	return r.OnDynamic(args, func() core.ProcessOutput { return output })
}

// OnDynamic responds to `vfox <args>` with the result of respond.
func (r *FakeProcessRunner) OnDynamic(args string, respond func() core.ProcessOutput) *FakeProcessRunner {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses[args] = respond
	return r
}

// OnStart responds to a started `vfox <args>` with the process from create.
func (r *FakeProcessRunner) OnStart(args string, create func() *FakeRunningProcess) *FakeProcessRunner {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams[args] = create
	return r
}

// SetMissingExecutable makes every invocation fail as if the executable did
// not exist, or stops doing so.
func (r *FakeProcessRunner) SetMissingExecutable(missing bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.missing = missing
}

// SetFailure makes every invocation fail with err, which is not a failure to
// start the executable, or stops doing so when err is nil.
func (r *FakeProcessRunner) SetFailure(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failure = err
}

// Invocations returns every invocation, in order.
func (r *FakeProcessRunner) Invocations() []ProcessCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ProcessCall(nil), r.invocations...)
}

// Calls returns the calls whose arguments start with prefix.
func (r *FakeProcessRunner) Calls(prefix string) []ProcessCall {
	var result []ProcessCall
	for _, call := range r.Invocations() {
		if strings.HasPrefix(strings.Join(call.Arguments, " "), prefix) {
			result = append(result, call)
		}
	}
	return result
}

// record notes an invocation and returns the error it fails with, if any.
func (r *FakeProcessRunner) record(executable string, args []string, options core.ProcessOptions, streamed bool) error {
	environment := make(map[string]string, len(options.Environment))
	for key, value := range options.Environment {
		environment[key] = value
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invocations = append(r.invocations, ProcessCall{
		Executable:       executable,
		Arguments:        append([]string(nil), args...),
		WorkingDirectory: options.WorkingDirectory,
		Environment:      environment,
		Streamed:         streamed,
	})
	if r.failure != nil {
		return r.failure
	}
	if r.missing {
		return &core.ProcessStartError{Executable: executable, Err: errMissing}
	}
	return nil
}

// Run implements core.ProcessRunner.
func (r *FakeProcessRunner) Run(executable string, args []string, options core.ProcessOptions) (core.ProcessOutput, error) {
	if err := r.record(executable, args, options, false); err != nil {
		return core.ProcessOutput{}, err
	}
	r.mu.Lock()
	respond := r.responses[strings.Join(args, " ")]
	r.mu.Unlock()
	if respond != nil {
		return respond(), nil
	}
	return core.ProcessOutput{ExitCode: 127, Stderr: "unexpected command"}, nil
}

// Start implements core.ProcessRunner.
func (r *FakeProcessRunner) Start(executable string, args []string, options core.ProcessOptions, listener core.ProcessListener) (core.RunningProcess, error) {
	if err := r.record(executable, args, options, true); err != nil {
		return nil, err
	}
	r.mu.Lock()
	create := r.streams[strings.Join(args, " ")]
	respond := r.responses[strings.Join(args, " ")]
	r.mu.Unlock()
	var process *FakeRunningProcess
	switch {
	case create != nil:
		process = create()
	case respond != nil:
		process = CompletedProcess(respond())
	default:
		process = CompletedProcess(core.ProcessOutput{})
	}
	process.Attach(listener)
	return process, nil
}

type missingExecutable struct{}

func (missingExecutable) Error() string { return "No such file or directory" }

var errMissing = missingExecutable{}

// FixtureRunner returns a runner that answers the read-only commands like the
// `list` and `current` fixtures: maven 3.9.16 (current) and 3.8.9, mvn2
// without versions and nodejs 24.11.1 (current) and 22.21.0. It also answers
// the searches of java (`open` without an argument, `tem` and `all`) and of
// flutter, which the fixtures hold.
func FixtureRunner() *FakeProcessRunner {
	return NewFakeProcessRunner().
		On("--version", FixtureOutput("version", 0)).
		On("list", FixtureOutput("list", 0)).
		On("current", FixtureOutput("current", 0)).
		On("available", FixtureOutput("available", 0)).
		On("config --list", FixtureOutput("config-list", 0)).
		On("search maven", FixtureOutput("search-maven-installed", 0)).
		On("search nodejs", FixtureOutput("search-nodejs", 0)).
		On("search java", FixtureOutput("search-java", 0)).
		On("search java tem", FixtureOutput("search-java-tem", 0)).
		On("search java all", FixtureOutput("search-java-all", 0)).
		On("search flutter", FixtureOutput("search-flutter", 0)).
		On("info nodejs --format "+vfox.PluginInfoTemplate, FixtureOutput("info", 0))
}
