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

// Package measure ports shared/tools/measure.sh: it starts the application with a
// fresh HOME and the fake vfox of shared/fixtures, drives it to the Home page and to
// the versions of Node.js through the automation plugin (see package appsession), and
// prints the resident (RSS) and proportional (PSS) set size of the application and
// all of its child processes at each measuring point.
package measure

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

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

// fixturesDirFromRoot resolves a --fixtures-root value to the directory with
// the captured outputs: the value itself when it already names a vfox-<v>
// directory, else its vfox-1.0.13 child.
func fixturesDirFromRoot(root string) string {
	if strings.HasPrefix(filepath.Base(root), "vfox-") {
		return root
	}
	return filepath.Join(root, "vfox-1.0.13")
}

func resolveFakeVfox(fixturesRootFlag, fakeVfoxFlag string) (fakeExplicit, fixturesDir string, err error) {
	if fakeVfoxFlag != "" {
		if _, serr := os.Stat(fakeVfoxFlag); serr != nil {
			return "", "", fmt.Errorf("fake vfox not found at %s: %w", fakeVfoxFlag, serr)
		}
		if fixturesRootFlag != "" {
			return fakeVfoxFlag, fixturesDirFromRoot(fixturesRootFlag), nil
		}
		if root, rerr := appsession.FindRepoRoot(); rerr == nil {
			return fakeVfoxFlag, appsession.FixturesDir(root), nil
		}
		return fakeVfoxFlag, filepath.Join(filepath.Dir(fakeVfoxFlag), "vfox-1.0.13"), nil
	}
	if fixturesRootFlag != "" {
		return "", fixturesDirFromRoot(fixturesRootFlag), nil
	}
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return "", "", fmt.Errorf("%w; pass --fake-vfox or --fixtures-root", err)
	}
	return "", appsession.FixturesDir(root), nil
}

// settle is how long the application rests at a measuring point before it is
// measured: what it loads after the page has been shown has been loaded by then.
const settle = 3 * time.Second

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
	command := fs.Args()
	if len(command) == 0 {
		return errors.New("Usage: measure [--output FILE] <application command...>")
	}
	output := *outputFlag
	if output == "" {
		output = os.Getenv("VFOX_PLUS_MEASURE_OUTPUT")
	}
	fakeExplicit, fixturesDir, err := resolveFakeVfox(*fixturesRootFlag, *fakeVfoxFlag)
	if err != nil {
		return err
	}
	if output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(output, nil, 0o644); err != nil {
			return err
		}
	}
	config := appsession.Config{Name: "measure", Command: command, FakeVfox: fakeExplicit, Fixtures: fixturesDir}
	return appsession.RunScenario(config, "measure", func(s *appsession.Session) { scenario(s, output) })
}

// scenario opens the Home page and the versions of Node.js, and measures the
// application at both points (see docs/en/dev/memory.md).
func scenario(s *appsession.Session, output string) {
	measureAt := func(point string) {
		line := fmt.Sprintf("memory (%s): %s", point, MemoryOf(s.PID()))
		fmt.Println(line)
		if output == "" {
			return
		}
		file, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
		if err == nil {
			_, err = fmt.Fprintln(file, line)
			if cerr := file.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			s.Fail("write %s: %v", output, err)
		}
	}
	s.WaitForText("Versions in use")
	s.Pause(settle)
	measureAt("home")
	s.Navigate(appsession.SDKs)
	s.ClickItem("nodejs")
	s.ClickButton("Available")
	s.WaitForText("872 versions")
	s.Pause(settle)
	measureAt("nodejs")
}
