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

//go:build linux

package gnomesession

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Run implements `gnome-session [--user NAME] [--extension ID] <command...>`.
func Run(args []string) error {
	o, err := parseArgs(args)
	if err != nil {
		return err
	}
	switch {
	case o.inSession:
		return runInSession(o)
	case os.Geteuid() == 0:
		return runAsRoot(o)
	default:
		return errors.New("gnome-session starts the system bus of the container, which takes root; run it as root in a container, or as the user inside dbus-run-session with --in-session")
	}
}

// runAsRoot makes what the shell wants of the system, and runs the session as the user. The exit
// status is that of the command.
func runAsRoot(o options) error {
	if err := prepareSystem(); err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	account, err := user.Lookup(o.user)
	if err != nil {
		return fmt.Errorf("the user of the session: %w", err)
	}
	uid, _ := strconv.Atoi(account.Uid)
	gid, _ := strconv.Atoi(account.Gid)
	// dbus-run-session makes the session bus and ends it with the command.
	argv := append([]string{"--", self, "gnome-session", "--in-session", "--extension", o.extension, "--user", o.user}, o.command...)
	command := exec.Command("dbus-run-session", argv...) //nolint:gosec // The tool of the session.
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}}
	command.Env = append(os.Environ(), "HOME="+account.HomeDir, "USER="+o.user, "LOGNAME="+o.user)
	return exitOf(command.Run())
}

// prepareSystem starts the system bus, if the container has none running, and takes away what makes the
// shell look for logind.
func prepareSystem() error {
	if err := os.MkdirAll("/run/dbus", 0o755); err != nil { //nolint:gosec // The directory of the system bus.
		return err
	}
	if _, err := os.Stat("/run/dbus/system_bus_socket"); err != nil {
		_ = os.Remove("/run/dbus/pid")
		if out, err := exec.Command("dbus-daemon", "--system", "--fork").CombinedOutput(); err != nil {
			return fmt.Errorf("start the system bus: %w\n%s", err, out)
		}
	}
	// A shell that finds these thinks that systemd runs, and asks logind about its session; a
	// container has no logind, and the shell stops at the first question.
	for _, dir := range []string{"/run/systemd/seats", "/run/systemd/sessions"} {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

// exitOf turns how a command ended into the error of this one: nil for 0, and the status otherwise.
func exitOf(err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return fmt.Errorf("the command ended with the status %d", exit.ExitCode())
	}
	return err
}

// runInSession runs inside the session bus of the user: it starts the shell, waits for the extension, runs the
// command on the display of the shell and ends the shell.
func runInSession(o options) error {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		dir, err := os.MkdirTemp("", "gnome-session-runtime-")
		if err != nil {
			return err
		}
		defer func() { _ = os.RemoveAll(dir) }()
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
		runtimeDir = dir
		_ = os.Setenv("XDG_RUNTIME_DIR", dir)
	}
	_ = os.Setenv("XDG_SESSION_TYPE", "wayland")
	_ = os.Setenv("XDG_CURRENT_DESKTOP", "ubuntu:GNOME")

	if out, err := exec.Command("gsettings", "set", "org.gnome.shell", "enabled-extensions", enabledExtensions(o.extension)).CombinedOutput(); err != nil { //nolint:gosec // The settings tool, with the extension that was asked for.
		return fmt.Errorf("enable the extension %s: %w\n%s", o.extension, err, out)
	}
	logFile, err := os.CreateTemp("", "gnome-shell-*.log")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(logFile.Name()) }()
	shell := exec.Command("gnome-shell", "--headless", "--unsafe-mode", "--virtual-monitor", "1280x800")
	shell.Stdout, shell.Stderr = logFile, logFile
	if err := shell.Start(); err != nil {
		return fmt.Errorf("start gnome-shell: %w", err)
	}
	exited := make(chan struct{})
	go func() {
		_ = shell.Wait()
		close(exited)
	}()
	defer stop(shell, exited)

	if err := waitForExtension(exited, logFile.Name()); err != nil {
		return err
	}
	display := waylandDisplay(readFile(logFile.Name()))
	fmt.Printf("gnome-session: GNOME Shell is up on %s, with %s\n", display, o.extension)

	command := exec.Command(o.command[0], o.command[1:]...) //nolint:gosec // The command that the caller asked for.
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.Env = append(os.Environ(), "WAYLAND_DISPLAY="+display, "GDK_BACKEND=wayland", "XDG_RUNTIME_DIR="+runtimeDir)
	err = command.Run()
	select {
	case <-exited:
		fmt.Printf("gnome-session: GNOME Shell ended while the command ran:\n%s\n", tail(readFile(logFile.Name()), 30))
	default:
	}
	return exitOf(err)
}

// waitForExtension waits for the extension to own the name of the watcher on the session bus, which
// it does once the shell has built its panel.
func waitForExtension(exited <-chan struct{}, logPath string) error {
	deadline := time.After(startTimeout)
	for {
		select {
		case <-exited:
			return fmt.Errorf("gnome-shell ended before the extension was up:\n%s", tail(readFile(logPath), 40))
		case <-deadline:
			return fmt.Errorf("the extension did not take the name of the watcher in %s:\n%s", startTimeout, tail(readFile(logPath), 40))
		case <-time.After(250 * time.Millisecond):
		}
		out, err := exec.Command("busctl", "--user", "status", "org.kde.StatusNotifierWatcher").CombinedOutput()
		if err == nil && strings.Contains(string(out), "PID=") {
			return nil
		}
	}
}

// stop ends the shell: asks it to, and kills it when it does not.
func stop(shell *exec.Cmd, exited <-chan struct{}) {
	select {
	case <-exited:
		return
	default:
	}
	_ = shell.Process.Signal(syscall.SIGTERM)
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		_ = shell.Process.Kill()
		<-exited
	}
}

func readFile(path string) string {
	data, _ := os.ReadFile(filepath.Clean(path))
	return string(data)
}
