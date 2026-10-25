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

// Package screenshots ports shared/tools/screenshots.sh: it runs the
// "screenshots" scenario twice (plain and with a slow fake vfox) and captures
// the screen with ImageMagick whenever the application prints "e2e: SHOT <name>".
package screenshots

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ParseShotName extracts the screenshot name from an application log line.
// Only lines starting with "e2e: SHOT " carry a shot. Names containing "/"
// or ".." are rejected to prevent path traversal outside the output
// directory, as are empty names.
func ParseShotName(line string) (name string, ok bool) {
	const prefix = "e2e: SHOT "
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	name = strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if name == "" {
		return "", false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", false
	}
	return name, true
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

func buildScreenshotsEnv(homeDir, fixturesDir string, extra []string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "VFOX_HOME=") || strings.HasPrefix(kv, "XDG_CONFIG_HOME=") {
			continue
		}
		env = append(env, kv)
	}
	env = append(env,
		"HOME="+homeDir,
		"LANG=C.UTF-8",
		"VFOX_PLUS_E2E_SCENARIO=screenshots",
		"VFOX_PLUS_FIXTURES="+fixturesDir,
	)
	return append(env, extra...)
}

func writeSettings(homeDir, vfoxPath string) error {
	dir := filepath.Join(homeDir, ".config", "vfox-plus")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("{\n  \"language\": \"english\",\n  \"vfoxExecutable\": \"%s\"\n}\n", vfoxPath)
	return os.WriteFile(filepath.Join(dir, "settings.json"), []byte(content), 0o644)
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

func captureShot(outputDir, name string) error {
	target := filepath.Join(outputDir, name+".png")
	cmd := exec.Command("import", "-window", "root", target)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("capture screenshot %s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func handleLogLine(outputDir, line string) error {
	if name, ok := ParseShotName(line); ok {
		if err := captureShot(outputDir, name); err != nil {
			return err
		}
		fmt.Println(name + ".png")
		return nil
	}
	if strings.HasPrefix(line, "e2e: ") {
		fmt.Println(line)
	}
	return nil
}

func runOnce(outputDir string, appCmd []string, extraEnv []string) error {
	root, err := findRepoRoot()
	if err != nil {
		return err
	}
	fixturesDir := filepath.Join(root, "shared", "fixtures", "vfox-1.0.13")
	sandbox, err := os.MkdirTemp("", "vfox-plus-screenshots.*")
	if err != nil {
		return err
	}
	appPid := 0
	defer func() {
		if appPid != 0 && procAlive(appPid) {
			killProcess(appPid)
		}
		removeSandbox(sandbox)
	}()
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
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	// extraEnv holds VAR=value assignments that precede the application command,
	// mirroring the second run with VFOX_PLUS_FAKE_DELAY=4 of the old script.
	cmd := exec.Command(appCmd[0], appCmd[1:]...)
	cmd.Env = buildScreenshotsEnv(homeDir, fixturesDir, extraEnv)
	// Interpret extraEnv entries of the form NAME=value as overrides; entries
	// without '=' are ignored by exec but already appended above.
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start application: %w", err)
	}
	defer logFile.Close()
	appPid = cmd.Process.Pid
	printed := 0
	// Reap the application in the background: a `kill -0` probe cannot
	// tell a live process from an unreaped zombie, so liveness is observed
	// through Wait instead of polling.
	done := make(chan int, 1)
	go func() { done <- exitStatus(cmd.Wait()) }()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case status := <-done:
			lines, _, err := newCompleteLines(logPath, printed)
			if err != nil {
				return err
			}
			for _, line := range lines {
				if herr := handleLogLine(outputDir, line); herr != nil {
					return herr
				}
			}
			appPid = 0
			if status != 0 {
				// Mirror screenshots.sh: dump the log on failure.
				if data, rerr := os.ReadFile(logPath); rerr == nil {
					fmt.Fprintf(os.Stderr, "%s", string(data))
				}
				return fmt.Errorf("application exited with status %d", status)
			}
			return nil
		case <-ticker.C:
			lines, next, err := newCompleteLines(logPath, printed)
			if err != nil {
				return err
			}
			for _, line := range lines {
				if herr := handleLogLine(outputDir, line); herr != nil {
					return herr
				}
			}
			printed = next
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

// Run implements `screenshots <output-dir> <app command...>`,
// mirroring shared/tools/screenshots.sh.
func Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: screenshots <output directory> <application command...>")
	}
	outputDir := args[0]
	appCmd := args[1:]
	if len(appCmd) == 0 {
		return errors.New("Usage: screenshots <output directory> <application command...>")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := runOnce(outputDir, appCmd, nil); err != nil {
		return err
	}
	return runOnce(outputDir, appCmd, []string{"VFOX_PLUS_FAKE_DELAY=4"})
}
