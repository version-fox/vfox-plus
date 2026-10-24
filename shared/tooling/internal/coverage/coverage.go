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

// Package coverage reports what the unit tests cover, by area, from a
// coverage profile, and fails when the lines of core and state, which have
// no user interface, are covered less than the minimum, or the lines of the
// user interface, which the tests of its components and pages cover, less
// than 75%. The end-to-end scenarios do not count here.
//
// It replaces mygo/tools/coverage.sh.
package coverage

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// uiMinimum is the minimum percent of the user interface that the tests must
// cover.
const uiMinimum = 75

// areaStats counts the statements of one area and how many the tests cover.
type areaStats struct {
	statements int
	covered    int
}

// classify maps a file from a coverage profile to its area, like the awk
// script of coverage.sh does.
func classify(file string) string {
	switch {
	case strings.Contains(file, "/mygo/core/"):
		return "core"
	case strings.Contains(file, "/mygo/state/"):
		return "state"
	case strings.Contains(file, "/mygo/ui/"):
		return "ui"
	default:
		return "other"
	}
}

// summarize parses a coverage profile, skipping its header line, and counts
// the statements and covered statements of every area.
func summarize(r io.Reader) (map[string]*areaStats, error) {
	stats := map[string]*areaStats{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	header := true
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if header {
			header = false
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("line %d: expected a file, a statement count and a cover count: %q", lineNo, line)
		}
		file := fields[0]
		if i := strings.IndexByte(file, ':'); i >= 0 {
			file = file[:i]
		}
		statements, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: bad statement count: %w", lineNo, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("line %d: bad cover count: %w", lineNo, err)
		}
		area := classify(file)
		s, ok := stats[area]
		if !ok {
			s = &areaStats{}
			stats[area] = s
		}
		s.statements += statements
		if count > 0 {
			s.covered += statements
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

// percent returns the covered percent of statements, counting an area with
// no statements as fully covered.
func percent(covered, statements int) float64 {
	if statements == 0 {
		return 100
	}
	return float64(covered) / float64(statements) * 100
}

// areaOf returns the stats of an area, treating a missing area as empty.
func areaOf(stats map[string]*areaStats, area string) (covered, statements int) {
	if s, ok := stats[area]; ok {
		return s.covered, s.statements
	}
	return 0, 0
}

// report prints one line per area plus the two gate lines, like the awk
// script of coverage.sh does, and reports whether both gates pass.
func report(stats map[string]*areaStats, minimum float64, minimumRaw string, w io.Writer) bool {
	areas := make([]string, 0, len(stats))
	for area := range stats {
		areas = append(areas, area)
	}
	sort.Strings(areas)
	for _, area := range areas {
		s := stats[area]
		fmt.Fprintf(w, "%-6s %5.1f %% of %d statements\n", area, percent(s.covered, s.statements), s.statements)
	}
	ok := true
	coreCovered, coreStatements := areaOf(stats, "core")
	stateCovered, stateStatements := areaOf(stats, "state")
	gated := percent(coreCovered+stateCovered, coreStatements+stateStatements)
	fmt.Fprintf(w, "core and state: %.1f %% (minimum %s %%)\n", gated, minimumRaw)
	if gated < minimum {
		ok = false
	}
	uiCovered, uiStatements := areaOf(stats, "ui")
	ui := percent(uiCovered, uiStatements)
	fmt.Fprintf(w, "ui: %.1f %% (minimum %d %%)\n", ui, uiMinimum)
	if ui < uiMinimum {
		ok = false
	}
	return ok
}

// Run reports the coverage of profile and fails when it is below the
// minimum. It accepts PROFILE [minimum percent of core and state], which
// defaults to 85.
func Run(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: coverage PROFILE [minimum]")
	}
	minimumRaw := "85"
	if len(args) == 2 {
		minimumRaw = args[1]
	}
	minimum, err := strconv.ParseFloat(minimumRaw, 64)
	if err != nil {
		return fmt.Errorf("invalid minimum %q", minimumRaw)
	}
	profile, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer profile.Close()
	stats, err := summarize(profile)
	if err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	if !report(stats, minimum, minimumRaw, os.Stdout) {
		return fmt.Errorf("coverage below the minimum")
	}
	return nil
}
