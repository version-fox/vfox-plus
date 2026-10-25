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

// Package display starts a virtual display with a session bus and
// smoke-tests a graphical application on it: the display the sandbox tasks
// and the `vfxtool smoke-test` command run their GUI checks on.
package display

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	// DefaultScreen is the -screen argument of Xvfb.
	DefaultScreen = "1280x800x24"
	// DefaultTimeout is how long SmokeTest waits before checking the application.
	DefaultTimeout = 15 * time.Second
	// displayName is the DISPLAY value.
	displayName = ":42"
)

// Display is a virtual display with a session bus.
type Display struct {
	pid int
	// Name is the DISPLAY value, e.g. ":42".
	Name string
	// BusAddress is DBUS_SESSION_BUS_ADDRESS.
	BusAddress string
}

// Environ exposes the display to a child process.
func (d *Display) Environ() []string {
	return []string{"DISPLAY=" + d.Name, "DBUS_SESSION_BUS_ADDRESS=" + d.BusAddress}
}

// Stop kills the virtual display. A nil display is a no-op.
func (d *Display) Stop() {
	if d == nil {
		return
	}
	proc, err := os.FindProcess(d.pid)
	if err != nil {
		return
	}
	_ = proc.Kill()
}

// Start starts Xvfb and a session bus. launchBus runs `dbus-launch
// --sh-syntax` and returns its output; nil runs it directly. Callers that
// need another user pass a launcher that switches to it.
func Start(screen string, launchBus func() ([]byte, error)) (*Display, error) {
	if screen == "" {
		screen = DefaultScreen
	}
	if launchBus == nil {
		launchBus = func() ([]byte, error) {
			return exec.Command("dbus-launch", "--sh-syntax").Output()
		}
	}
	xvfb := exec.Command("Xvfb", displayName, "-nolisten", "tcp", "-screen", "0", screen)
	xvfb.Stdout = nil
	xvfb.Stderr = nil
	if err := xvfb.Start(); err != nil {
		return nil, err
	}
	d := &Display{pid: xvfb.Process.Pid, Name: displayName}
	// Do not wait for Xvfb: like `&` in the shell, it runs until killed.
	go func() { _ = xvfb.Wait() }()
	time.Sleep(time.Second)
	out, err := launchBus()
	if err != nil {
		d.Stop()
		return nil, fmt.Errorf("dbus-launch: %w", err)
	}
	d.BusAddress = parseBusAddress(out)
	return d, nil
}

var busAssignment = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*);$`)

// parseBusAddress reads DBUS_SESSION_BUS_ADDRESS from the output of
// `dbus-launch --sh-syntax` (`VAR='value'; export VAR;` per line).
func parseBusAddress(out []byte) string {
	for _, line := range strings.Split(string(out), "\n") {
		m := busAssignment.FindStringSubmatch(line)
		if m == nil || m[1] != "DBUS_SESSION_BUS_ADDRESS" {
			continue
		}
		value := m[2]
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
			value = value[1 : len(value)-1]
		}
		return value
	}
	return ""
}

// Options configures SmokeTest.
type Options struct {
	// Screen is the -screen argument of Xvfb; empty means DefaultScreen.
	Screen string
	// Timeout is how long the application may take to stay up;
	// non-positive means DefaultTimeout.
	Timeout time.Duration
	// LogPath collects the output of the application; empty discards it.
	// The content is printed when the application exits early.
	LogPath string
	// Prefix runs before the executable, for example a user switch:
	// {"runuser", "-u", "tester", "--", "env", "HOME=/home/tester"}.
	Prefix []string
	// Args are passed to the executable.
	Args []string
	// ExtraEnv adds VAR=value entries to the environment of the application.
	ExtraEnv []string
	// LaunchBus runs `dbus-launch --sh-syntax`; nil runs it directly.
	LaunchBus func() ([]byte, error)
}

// SmokeTest starts the application on a virtual display and checks that it
// still runs after the timeout. An application that already exited fails the
// test, including one that crashed immediately: the exit is observed through
// Wait, because a `kill -0` probe alone cannot tell a live process from an
// unreaped zombie.
func SmokeTest(executable string, opt Options) error {
	if opt.Screen == "" {
		opt.Screen = DefaultScreen
	}
	if opt.Timeout <= 0 {
		opt.Timeout = DefaultTimeout
	}
	d, err := Start(opt.Screen, opt.LaunchBus)
	if err != nil {
		return err
	}
	defer d.Stop()
	log, err := logWriter(opt.LogPath)
	if err != nil {
		return err
	}
	defer log.Close()
	argv := append(append([]string{}, opt.Prefix...), executable)
	argv = append(argv, opt.Args...)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), append(d.Environ(), opt.ExtraEnv...)...)
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan int, 1)
	go func() {
		done <- exitCode(cmd.Wait())
	}()
	select {
	case code := <-done:
		dumpLog(opt.LogPath)
		return fmt.Errorf("vfox+ exited during startup (exit status %d)", code)
	case <-time.After(opt.Timeout):
	}
	_ = cmd.Process.Kill()
	<-done // Reap the killed process.
	fmt.Println("vfox+ started: " + executable)
	return nil
}

// dumpLog prints a collected application log for a failed smoke test.
func dumpLog(path string) {
	if path == "" {
		return
	}
	if data, err := os.ReadFile(path); err == nil {
		fmt.Fprint(os.Stderr, string(data))
	}
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
		if status, ok := exit.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return -int(status.Signal())
			}
			return status.ExitStatus()
		}
	}
	return 1
}

func logWriter(path string) (io.WriteCloser, error) {
	if path == "" {
		return discardCloser{}, nil
	}
	return os.Create(path)
}

// discardCloser is io.Discard with a no-op Close, for runs without --log.
type discardCloser struct{}

func (discardCloser) Write(p []byte) (int, error) { return len(p), nil }
func (discardCloser) Close() error                { return nil }
