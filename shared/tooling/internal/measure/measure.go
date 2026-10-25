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

// Package measure ports shared/tools/measure.sh: it runs the application in
// its "measure" scenario with a fresh HOME and the fake vfox of
// shared/fixtures, and prints the resident (RSS) and proportional (PSS) set
// size of the application and all of its child processes at each measuring
// point the application reports.
package measure

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ParseMeasurePoint extracts the measuring point from an application log line.
// It reports ok=false unless the line contains "e2e: MEASURE ". The point is
// everything after the last "MEASURE " marker, mirroring ${line##*MEASURE }
// in measure.sh.
func ParseMeasurePoint(line string) (point string, ok bool) {
	const marker = "e2e: MEASURE "
	if !strings.Contains(line, marker) {
		return "", false
	}
	idx := strings.LastIndex(line, "MEASURE ")
	if idx < 0 {
		return "", false
	}
	return line[idx+len("MEASURE "):], true
}

// IsMeasureLine reports whether line carries a measuring point.
func IsMeasureLine(line string) bool {
	_, ok := ParseMeasurePoint(line)
	return ok
}

// MemoryOf returns the resident and proportional set size of pid and all of
// its descendants, as "RSS <n> MB, PSS <n> MB in <n> processes", mirroring
// memory_of in measure.sh. Sizes are read from
// /proc/<pid>/task/*/children (process tree) and /proc/<pid>/smaps_rollup.
func MemoryOf(pid int) string {
	rss, pss, count := memoryOf(pid)
	return fmt.Sprintf("RSS %d MB, PSS %d MB in %d processes", rss/1024, pss/1024, count)
}

func memoryOf(pid int) (rss, pss int64, count int) {
	pids := []int{pid}
	for i := 0; i < len(pids); i++ {
		pattern := fmt.Sprintf("/proc/%d/task/*/children", pids[i])
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil {
				continue
			}
			for _, field := range strings.Fields(string(data)) {
				child, err := strconv.Atoi(field)
				if err != nil {
					continue
				}
				pids = append(pids, child)
			}
		}
	}
	for _, p := range pids {
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/smaps_rollup", p))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "Rss:") || strings.HasPrefix(line, "Pss:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if v, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						if strings.HasPrefix(line, "Rss:") {
							rss += v
						} else {
							pss += v
						}
					}
				}
			}
		}
	}
	return rss, pss, len(pids)
}

func procAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
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

func copyFileMode(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
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
	return "", errors.New("cannot locate repository root (no shared/fixtures above working directory); pass --fake-vfox or --fixtures-root")
}

// fixturesDirFromRoot resolves a --fixtures-root value to the directory with
// the captured outputs: the value itself when it already names a vfox-<v>
// directory, else its vfox-1.0.13 child.
func fixturesDirFromRoot(root string) string {
	if strings.HasPrefix(filepath.Base(root), "vfox-") {
		return root
	}
	return filepath.Join(root, "vfox-1.0.13")
}

// materializeFakeVfox installs the fake vfox executable at dest: explicit,
// when set, is copied as-is; otherwise the running vfxtool binary is copied,
// which answers vfox commands when executed under the name "vfox" (see
// cmd/vfxtool).
func materializeFakeVfox(dest, explicit string) error {
	src := explicit
	if src == "" {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		src = self
	}
	return copyFileMode(src, dest, 0o755)
}

func resolveFakeVfox(fixturesRootFlag, fakeVfoxFlag string) (fakeExplicit, fixturesDir string, err error) {
	if fakeVfoxFlag != "" {
		if _, serr := os.Stat(fakeVfoxFlag); serr != nil {
			return "", "", fmt.Errorf("fake vfox not found at %s: %w", fakeVfoxFlag, serr)
		}
		if fixturesRootFlag != "" {
			return fakeVfoxFlag, fixturesDirFromRoot(fixturesRootFlag), nil
		}
		if root, rerr := findRepoRoot(); rerr == nil {
			return fakeVfoxFlag, filepath.Join(root, "shared", "fixtures", "vfox-1.0.13"), nil
		}
		return fakeVfoxFlag, filepath.Join(filepath.Dir(fakeVfoxFlag), "vfox-1.0.13"), nil
	}
	if fixturesRootFlag != "" {
		return "", fixturesDirFromRoot(fixturesRootFlag), nil
	}
	root, err := findRepoRoot()
	if err != nil {
		return "", "", err
	}
	return "", filepath.Join(root, "shared", "fixtures", "vfox-1.0.13"), nil
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

func buildMeasureEnv(homeDir, vfoxPath, fixturesDir string) []string {
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
		"VFOX_PLUS_E2E_SCENARIO=measure",
		"VFOX_PLUS_E2E_VFOX="+vfoxPath,
		"VFOX_PLUS_FIXTURES="+fixturesDir,
		"__VFOX_PID=1",
		"__VFOX_SHELL=bash",
	)
	return env
}

// newCompleteLines returns the complete (newline-terminated) log lines added
// since printed, mirroring `wc -l` + `sed -n` in measure.sh. printed counts
// newline characters already shown.
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
	// First n entries are complete lines.
	added := append([]string(nil), lines[printed:n]...)
	return added, n, nil
}

// remainingTail returns any log content beyond printed complete lines,
// including a final partial line without a trailing newline.
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

// Run implements `measure [--output FILE] [--fixtures-root DIR] [--fake-vfox FILE] <app command...>`.
// It mirrors the old shared/tools/measure.sh.
func Run(args []string) error {
	fs := flag.NewFlagSet("measure", flag.ContinueOnError)
	outputFlag := fs.String("output", "", "file collecting the memory(<point>) lines (also $VFOX_PLUS_MEASURE_OUTPUT)")
	fixturesRootFlag := fs.String("fixtures-root", "", "shared/fixtures directory (default: found via repository root)")
	fakeVfoxFlag := fs.String("fake-vfox", "", "explicit fake vfox executable (default: this vfxtool binary, executed as vfox)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: measure [--output FILE] [--fixtures-root DIR] [--fake-vfox FILE] <application command...>\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	appCmd := fs.Args()
	if len(appCmd) == 0 {
		return errors.New("Usage: measure [--output FILE] <application command...>")
	}
	output := *outputFlag
	if output == "" {
		output = os.Getenv("VFOX_PLUS_MEASURE_OUTPUT")
	}
	timeout, err := e2eTimeout()
	if err != nil {
		return err
	}
	fakeExplicit, fixturesDir, err := resolveFakeVfox(*fixturesRootFlag, *fakeVfoxFlag)
	if err != nil {
		return err
	}

	sandbox, err := os.MkdirTemp("", "vfox-plus-measure.*")
	if err != nil {
		return err
	}
	homeDir := filepath.Join(sandbox, "home")
	logPath := filepath.Join(sandbox, "app.log")
	appPid := 0
	cleanup := func() {
		if appPid != 0 && procAlive(appPid) {
			_ = syscall.Kill(appPid, syscall.SIGKILL)
		}
		removeSandbox(sandbox)
	}
	defer cleanup()

	if output != "" {
		if dir := filepath.Dir(output); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		if err := os.WriteFile(output, nil, 0o644); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "vfox-plus"), 0o755); err != nil {
		return err
	}
	vfoxPath := filepath.Join(sandbox, "vfox")
	if err := materializeFakeVfox(vfoxPath, fakeExplicit); err != nil {
		return fmt.Errorf("install fake vfox: %w", err)
	}
	if err := writeSettings(homeDir, vfoxPath); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	cmd := exec.Command(appCmd[0], appCmd[1:]...)
	cmd.Env = buildMeasureEnv(homeDir, vfoxPath, fixturesDir)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start application: %w", err)
	}
	defer logFile.Close()
	appPid = cmd.Process.Pid
	printed := 0
	deadline := time.Now().Add(timeout)
	showNew := func() error {
		lines, next, err := newCompleteLines(logPath, printed)
		if err != nil {
			return err
		}
		for _, line := range lines {
			fmt.Println(line)
			if point, ok := ParseMeasurePoint(line); ok {
				result := fmt.Sprintf("memory (%s): %s", point, MemoryOf(appPid))
				fmt.Println(result)
				if output != "" {
					f, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
					if err != nil {
						return err
					}
					_, werr := fmt.Fprintln(f, result)
					_ = f.Close()
					if werr != nil {
						return werr
					}
				}
			}
		}
		printed = next
		return nil
	}
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
			if err := showNew(); err != nil {
				return err
			}
			appPid = 0
			// Print any trailing partial line the periodic scan skipped.
			for _, line := range remainingTail(logPath, printed) {
				fmt.Println(line)
			}
			data, err := os.ReadFile(logPath)
			if err != nil {
				return fmt.Errorf("read application log: %w", err)
			}
			if !strings.Contains(string(data), "e2e: scenario measure passed") {
				if status == 0 {
					status = 1
				}
				return fmt.Errorf("The application did not report the end of the measure scenario (exit status %d).", status)
			}
			if status != 0 {
				return fmt.Errorf("application exited with status %d", status)
			}
			return nil
		case <-ticker.C:
			if err := showNew(); err != nil {
				return err
			}
			if time.Now().After(deadline) {
				_ = cmd.Process.Kill()
				<-done
				appPid = 0
				return errors.New("The application did not finish the measure scenario in time.")
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
