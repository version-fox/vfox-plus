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

// Package e2erun ports shared/e2e/run.sh: it runs an end-to-end scenario of
// an application built with VFOX_PLUS_E2E=1 against a fresh HOME.
package e2erun

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/testserver"
)

// IsValidScenario reports whether s names a known end-to-end scenario.
func IsValidScenario(s string) bool {
	switch s {
	case "fake", "real", "measure":
		return true
	default:
		return false
	}
}

// ValidateScenario returns an error unless s is a known scenario.
func ValidateScenario(s string) error {
	if !IsValidScenario(s) {
		return fmt.Errorf("Unknown scenario: %s", s)
	}
	return nil
}

func removeSandbox(sandbox string) {
	for attempt := 1; attempt <= 5; attempt++ {
		if err := os.RemoveAll(sandbox); err == nil {
			return
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	fmt.Fprintf(os.Stderr, "Could not remove %s\n", sandbox)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(dst, data, mode)
}

func writeSettings(homeDir, vfoxPath string) error {
	dir := filepath.Join(homeDir, ".config", "vfox-plus")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("{\n  \"language\": \"english\",\n  \"vfoxExecutable\": \"%s\"\n}\n", vfoxPath)
	return os.WriteFile(filepath.Join(dir, "settings.json"), []byte(content), 0o644)
}

func searchRepoRootUp(dir string) (string, bool) {
	for {
		if fi, err := os.Stat(filepath.Join(dir, "shared", "fixtures")); err == nil && fi.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func findRepoRoot() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		if root, ok := searchRepoRootUp(cwd); ok {
			return root, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if root, ok := searchRepoRootUp(filepath.Dir(exe)); ok {
			return root, nil
		}
	}
	return "", errors.New("cannot locate repository root (no shared/fixtures above working directory)")
}

// materializeFakeVfox installs the fake vfox executable at dest: the running
// vfxtool binary is copied, which answers vfox commands when executed under
// the name "vfox" (see cmd/vfxtool).
func materializeFakeVfox(dest string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o755)
}

func e2eTimeout() (time.Duration, error) {
	s := os.Getenv("VFOX_PLUS_E2E_TIMEOUT")
	if s == "" {
		return 300 * time.Second, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("invalid VFOX_PLUS_E2E_TIMEOUT %q: %w", s, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid VFOX_PLUS_E2E_TIMEOUT %q: must not be negative", s)
	}
	return time.Duration(n) * time.Second, nil
}

func baseEnv(homeDir, scenario string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "VFOX_HOME=") || strings.HasPrefix(kv, "XDG_CONFIG_HOME=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"HOME="+homeDir,
		"VFOX_PLUS_E2E_SCENARIO="+scenario,
		"LANG=C.UTF-8",
	)
}

// newCompleteLines returns complete log lines added since printed.
func newCompleteLines(logPath string, printed int) ([]string, int, error) {
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, printed, nil
		}
		return nil, printed, err
	}
	n := strings.Count(string(data), "\n")
	if n <= printed {
		return nil, printed, nil
	}
	lines := strings.Split(string(data), "\n")
	return append([]string(nil), lines[printed:n]...), n, nil
}

func remainingTail(logPath string, printed int) []string {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil
	}
	text := string(data)
	n := strings.Count(text, "\n")
	lines := strings.Split(text, "\n")
	var out []string
	if n > printed {
		out = append(out, lines[printed:n]...)
		printed = n
	}
	if printed < len(lines) && lines[printed] != "" {
		out = append(out, lines[printed])
	}
	return out
}

func streamApp(scenario string, appCmd []string, env []string, sandbox, logPath string, timeout time.Duration) error {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd := exec.Command(appCmd[0], appCmd[1:]...)
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start application: %w", err)
	}
	// Reap the application in the background: a `kill -0` probe cannot
	// tell a live process from an unreaped zombie, so liveness is observed
	// through Wait instead of polling.
	done := make(chan int, 1)
	go func() { done <- exitStatus(cmd.Wait()) }()
	printed := 0
	deadline := time.Now().Add(timeout)
	tail := func() error {
		lines, next, err := newCompleteLines(logPath, printed)
		if err != nil {
			return err
		}
		for _, line := range lines {
			fmt.Println(line)
		}
		printed = next
		return nil
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	var status int
	for {
		select {
		case status = <-done:
			for _, line := range remainingTail(logPath, printed) {
				fmt.Println(line)
			}
			data, err := os.ReadFile(logPath)
			if err != nil {
				return fmt.Errorf("read application log: %w", err)
			}
			want := "e2e: scenario " + scenario + " passed"
			if !strings.Contains(string(data), want) {
				if status == 0 {
					status = 1
				}
				return fmt.Errorf("scenario %q did not pass (application exit status %d; missing %q)", scenario, status, want)
			}
			if status != 0 {
				return fmt.Errorf("application exited with status %d", status)
			}
			return nil
		case <-ticker.C:
			if err := tail(); err != nil {
				return err
			}
			if time.Now().After(deadline) {
				_ = cmd.Process.Kill()
				status = <-done
				return errors.New("The scenario did not finish in time.")
			}
		}
	}
}

// exitStatus maps the result of Wait to an exit code. A process killed by a
// signal has a non-zero status, like any other failure.
func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code != 0 {
			return code
		}
	}
	return 1
}

func runFake(appCmd []string, timeout time.Duration) error {
	root, err := findRepoRoot()
	if err != nil {
		return err
	}
	fixturesDir := filepath.Join(root, "shared", "fixtures", "vfox-1.0.13")
	sandbox, err := os.MkdirTemp("", "vfox-plus-e2e.*")
	if err != nil {
		return err
	}
	defer removeSandbox(sandbox)
	homeDir := filepath.Join(sandbox, "home")
	logPath := filepath.Join(sandbox, "app.log")
	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "vfox-plus"), 0o755); err != nil {
		return err
	}
	vfoxPath := filepath.Join(sandbox, "vfox")
	if err := materializeFakeVfox(vfoxPath); err != nil {
		return fmt.Errorf("install fake vfox: %w", err)
	}
	if err := writeSettings(homeDir, vfoxPath); err != nil {
		return err
	}
	env := baseEnv(homeDir, "fake")
	env = append(env,
		"VFOX_PLUS_FIXTURES="+fixturesDir,
		"__VFOX_PID=1",
		"__VFOX_SHELL=bash",
		"VFOX_PLUS_E2E_VFOX="+vfoxPath,
	)
	return streamApp("fake", appCmd, env, sandbox, logPath, timeout)
}

func runReal(appCmd []string, timeout time.Duration) error {
	vfox := os.Getenv("VFOX_PLUS_E2E_VFOX")
	if vfox == "" {
		return errors.New("Set VFOX_PLUS_E2E_VFOX to the vfox executable.")
	}
	root, err := findRepoRoot()
	if err != nil {
		return err
	}
	pluginSrc := filepath.Join(root, "shared", "test-plugin", "testsdk.lua")
	if _, err := os.Stat(pluginSrc); err != nil {
		return fmt.Errorf("test plugin not found at %s: %w", pluginSrc, err)
	}
	sandbox, err := os.MkdirTemp("", "vfox-plus-e2e.*")
	if err != nil {
		return err
	}
	defer removeSandbox(sandbox)
	homeDir := filepath.Join(sandbox, "home")
	logPath := filepath.Join(sandbox, "app.log")
	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "vfox-plus"), 0o755); err != nil {
		return err
	}
	pluginDst := filepath.Join(sandbox, "my-plugin.lua")
	if err := copyFile(pluginSrc, pluginDst); err != nil {
		return fmt.Errorf("copy test plugin: %w", err)
	}
	if err := writeSettings(homeDir, vfox); err != nil {
		return err
	}
	// The test server runs in-process: no node, no log scraping.
	serverURL, stopServer, err := testserver.Start(pluginSrc)
	if err != nil {
		return fmt.Errorf("start test server: %w", err)
	}
	defer stopServer()
	env := baseEnv(homeDir, "real")
	env = append(env,
		"VFOX_PLUS_TEST_SERVER="+serverURL,
		"VFOX_PLUS_E2E_LOCAL_PLUGIN="+pluginDst,
		"VFOX_PLUS_E2E_VFOX="+vfox,
	)
	return streamApp("real", appCmd, env, sandbox, logPath, timeout)
}

// Run implements `e2e-run <fake|real|measure> <app command...>`,
// mirroring shared/e2e/run.sh. The measure scenario delegates to the
// measure package in-process.
func Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: e2e-run <fake|real|measure> <application command...>")
	}
	scenario := args[0]
	appCmd := args[1:]
	if len(appCmd) == 0 {
		return errors.New("Usage: e2e-run <fake|real|measure> <application command...>")
	}
	if err := ValidateScenario(scenario); err != nil {
		return err
	}
	if scenario == "measure" {
		return measure.Run(appCmd)
	}
	timeout, err := e2eTimeout()
	if err != nil {
		return err
	}
	switch scenario {
	case "fake":
		return runFake(appCmd, timeout)
	case "real":
		return runReal(appCmd, timeout)
	default:
		return fmt.Errorf("Unknown scenario: %s", scenario)
	}
}
