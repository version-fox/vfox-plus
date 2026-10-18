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

package host

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
)

// Launch describes how to start a process, as docs/vfox-cli-contract.md asks:
// directly, with an argument vector, a complete environment, a working
// directory and a closed standard input.
type Launch struct {
	// Executable is a path, or a name looked up in the PATH of Environment.
	Executable string
	// Arguments are passed as they are; no shell is involved.
	Arguments []string
	// WorkingDirectory is the directory the process runs in.
	WorkingDirectory string
	// Environment is the complete environment: nothing is inherited.
	Environment map[string]string
}

// StartError reports that an executable could not be started at all.
type StartError struct {
	Err error
}

func (e *StartError) Error() string { return e.Err.Error() }

func (e *StartError) Unwrap() error { return e.Err }

// lookPath finds name in the directories of path, as a shell would. Names
// that contain a slash are returned as they are.
func lookPath(name, path string) (string, error) {
	if strings.Contains(name, "/") {
		return name, nil
	}
	for _, directory := range filepath.SplitList(path) {
		if directory == "" {
			continue
		}
		candidate := filepath.Join(directory, name)
		if IsExecutable(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s: executable file not found in PATH", name)
}

func (l Launch) command() (*exec.Cmd, error) {
	executable, err := lookPath(l.Executable, l.Environment["PATH"])
	if err != nil {
		return nil, &StartError{err}
	}
	command := exec.Command(executable, l.Arguments...) //nolint:gosec // Running vfox is the purpose of the host.
	command.Dir = l.WorkingDirectory
	command.Env = environmentList(l.Environment)
	// A nil Stdin is /dev/null: vfox detects that it does not run in a
	// terminal and never prompts; an open stdin would make a prompt hang.
	command.Stdin = nil
	return command, nil
}

// environmentList turns variables into the KEY=VALUE list of exec.Cmd, sorted
// so that processes see the same environment every time.
func environmentList(variables map[string]string) []string {
	list := make([]string, 0, len(variables))
	for key, value := range variables {
		list = append(list, key+"="+value)
	}
	sort.Strings(list)
	return list
}

// exitCode is the exit code of a finished process, or 128 plus the number of
// the signal that ended it, as a shell reports it.
func exitCode(state *os.ProcessState) int {
	if state == nil {
		return -1
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return state.ExitCode()
}

// RunResult is what a finished process wrote, decoded as UTF-8 with malformed
// bytes replaced, and how it exited.
type RunResult struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Run runs a process to completion. It fails with a StartError if the process
// cannot be started.
func Run(launch Launch) (RunResult, error) {
	command, err := launch.command()
	if err != nil {
		return RunResult{}, err
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return RunResult{}, &StartError{err}
	}
	// A non-zero exit is a result, not an error.
	_ = command.Wait()
	return RunResult{
		ExitCode: exitCode(command.ProcessState),
		Stdout:   strings.ToValidUTF8(stdout.String(), "�"),
		Stderr:   strings.ToValidUTF8(stderr.String(), "�"),
	}, nil
}

// Event is the output or the exit of a started process, in the shape the
// front end expects (web/src/hosts/protocol.ts).
type Event map[string]any

func outputEvent(process int, stream string, data []byte) Event {
	return Event{"event": "output", "process": process, "stream": stream, "data": base64.StdEncoding.EncodeToString(data)}
}

func exitEvent(process, code int) Event {
	return Event{"event": "exit", "process": process, "code": code}
}

// Processes are the processes started by the front end that still run.
type Processes struct {
	mu      sync.Mutex
	next    int
	running map[int]*os.Process
}

// NewProcesses creates an empty process table.
func NewProcesses() *Processes {
	return &Processes{running: map[int]*os.Process{}}
}

// Start starts a process and sends its output and then its exit to send, from
// background goroutines. It returns the identifier of the process, or a
// StartError if it cannot be started.
func (p *Processes) Start(launch Launch, send func(Event)) (int, error) {
	command, err := launch.command()
	if err != nil {
		return 0, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return 0, &StartError{err}
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return 0, &StartError{err}
	}
	if err := command.Start(); err != nil {
		return 0, &StartError{err}
	}
	p.mu.Lock()
	p.next++
	id := p.next
	p.running[id] = command.Process
	p.mu.Unlock()

	var readers sync.WaitGroup
	forward := func(stream io.Reader, name string) {
		defer readers.Done()
		buffer := make([]byte, 16*1024)
		for {
			read, err := stream.Read(buffer)
			if read > 0 {
				send(outputEvent(id, name, buffer[:read]))
			}
			if err != nil {
				return
			}
		}
	}
	readers.Add(2)
	go forward(stdout, "stdout")
	go forward(stderr, "stderr")
	go func() {
		// Wait closes the pipes, so it may only run once they are drained;
		// the exit is therefore always the last event of the process.
		readers.Wait()
		_ = command.Wait()
		p.mu.Lock()
		delete(p.running, id)
		p.mu.Unlock()
		send(exitEvent(id, exitCode(command.ProcessState)))
	}()
	return id, nil
}

// Kill asks the process to terminate with SIGTERM, as the CLI contract asks
// for cancelling vfox. Unknown or finished processes are ignored.
func (p *Processes) Kill(id int) {
	p.mu.Lock()
	process := p.running[id]
	p.mu.Unlock()
	if process != nil {
		if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
			fmt.Fprintf(os.Stderr, "vfox+: cannot terminate process %d: %v\n", id, err)
		}
	}
}

// KillAll terminates every process that still runs, when the application quits.
func (p *Processes) KillAll() {
	p.mu.Lock()
	ids := make([]int, 0, len(p.running))
	for id := range p.running {
		ids = append(ids, id)
	}
	p.mu.Unlock()
	for _, id := range ids {
		p.Kill(id)
	}
}
