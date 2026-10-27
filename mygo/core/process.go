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

package core

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// ProcessOutput is everything a finished process wrote, and how it exited.
type ProcessOutput struct {
	// The exit code. Zero means success.
	ExitCode int
	// Everything written to standard output, decoded as UTF-8.
	Stdout string
	// Everything written to standard error, decoded as UTF-8.
	Stderr string
}

// ProcessOptions is where and how a child process runs.
type ProcessOptions struct {
	// The working directory of the child.
	WorkingDirectory string
	// The complete environment of the child: the environment of the
	// application is not inherited implicitly, so that callers can remove
	// variables as well as add them.
	Environment map[string]string
}

// ProcessListener receives the output of a started process while it runs.
type ProcessListener interface {
	// Stdout receives a chunk of raw standard output.
	Stdout(chunk []byte)
	// Stderr receives a chunk of raw standard error.
	Stderr(chunk []byte)
}

// RunningProcess is a child process whose output is consumed while it runs.
type RunningProcess interface {
	// Wait returns the exit code once the process has ended and every chunk
	// of its output was delivered to the listener.
	Wait() int
	// Kill asks the process to terminate with `SIGTERM`.
	Kill()
}

// ProcessRunner spawns child processes.
//
// The application never spawns processes directly, which keeps the vfox
// client testable with a fake runner and leaves room for a remote
// implementation (see `docs/en/dev/architecture.md`). Standard input is closed
// immediately: nothing is ever written to the child, and an open stdin would
// make an unexpected interactive prompt wait forever.
type ProcessRunner interface {
	// Run executes `executable` to completion and collects its output.
	//
	// It returns a *ProcessStartError if the executable cannot be started.
	Run(executable string, args []string, options ProcessOptions) (ProcessOutput, error)
	// Start launches `executable` and returns as soon as it runs.
	//
	// It returns a *ProcessStartError if the executable cannot be started.
	Start(executable string, args []string, options ProcessOptions, listener ProcessListener) (RunningProcess, error)
}

// ProcessStartError reports that an executable could not be started, for
// example because it is missing.
type ProcessStartError struct {
	// The path or name that could not be executed.
	Executable string
	Err        error
}

// Error is the message of the operating system, which names the executable.
func (e *ProcessStartError) Error() string { return e.Err.Error() }

func (e *ProcessStartError) Unwrap() error { return e.Err }

// OsProcessRunner is the ProcessRunner of the desktop, on os/exec.
type OsProcessRunner struct{}

func command(executable string, args []string, options ProcessOptions) *exec.Cmd {
	cmd := exec.Command(executable, args...) //nolint:gosec // Running vfox is the purpose; there is no shell.
	cmd.Dir = options.WorkingDirectory
	cmd.Env = make([]string, 0, len(options.Environment))
	for key, value := range options.Environment {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	// Nothing is ever written to the child; see ProcessRunner.
	cmd.Stdin = nil
	return cmd
}

// Run implements ProcessRunner.
func (OsProcessRunner) Run(executable string, args []string, options ProcessOptions) (ProcessOutput, error) {
	cmd := command(executable, args, options)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return ProcessOutput{}, &ProcessStartError{Executable: executable, Err: err}
	}
	err := cmd.Wait()
	return ProcessOutput{
		ExitCode: exitCode(err),
		// Malformed bytes are replaced, as with a UTF-8 decoder.
		Stdout: strings.ToValidUTF8(stdout.String(), "\uFFFD"),
		Stderr: strings.ToValidUTF8(stderr.String(), "\uFFFD"),
	}, nil
}

// Start implements ProcessRunner.
func (OsProcessRunner) Start(executable string, args []string, options ProcessOptions, listener ProcessListener) (RunningProcess, error) {
	cmd := command(executable, args, options)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, &ProcessStartError{Executable: executable, Err: err}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, &ProcessStartError{Executable: executable, Err: err}
	}
	if err := cmd.Start(); err != nil {
		return nil, &ProcessStartError{Executable: executable, Err: err}
	}
	process := &osRunningProcess{cmd: cmd, done: make(chan int, 1)}
	go process.pump(stdout, stderr, listener.Stdout, listener.Stderr)
	return process, nil
}

type osRunningProcess struct {
	cmd      *exec.Cmd
	killOnce sync.Once
	waitOnce sync.Once
	code     int
	done     chan int
}

func (p *osRunningProcess) pump(stdout, stderr io.Reader, deliverStdout, deliverStderr func([]byte)) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pumpReader(stdout, deliverStdout)
	}()
	go func() {
		defer wg.Done()
		pumpReader(stderr, deliverStderr)
	}()
	wg.Wait()
	err := p.cmd.Wait()
	p.done <- exitCode(err)
}

func pumpReader(stream io.Reader, deliver func([]byte)) {
	buffer := make([]byte, 65536)
	for {
		n, err := stream.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			deliver(chunk)
		}
		if err != nil {
			return
		}
	}
}

// Wait implements RunningProcess.
func (p *osRunningProcess) Wait() int {
	p.waitOnce.Do(func() { p.code = <-p.done })
	return p.code
}

// Kill implements RunningProcess.
//
// On Unix the process is asked to terminate with SIGTERM, so that vfox can
// clean up; on Windows it is killed, which vfox treats like an interrupted
// install (see docs/en/dev/vfox-cli-contract.md#cancelling).
func (p *osRunningProcess) Kill() {
	p.killOnce.Do(func() {
		killProcess(p.cmd)
	})
}

// exitCode maps the result of Wait to an exit code. A process killed by a
// signal has a negative status, which is a failure like any other non-zero
// code.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return platformExitCode(exit)
	}
	return 1
}
