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

package memoryreport

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v93/github"
)

// The artifacts of the measurements beyond the desktop table, by upload name:
// the application in the lightweight mode from the mygo
// workflow, the APK from the android workflow.
const (
	LightweightArtifact = "memory-lightweight"
	AndroidArtifact     = "memory-android"
)

// lightweightLine matches one line of the memory-lightweight artifact: the
// application and all of its child processes, without its window, settled with no
// browser (lightweight-idle) or with a paired browser past its first inventory
// (lightweight-loaded).
var lightweightLine = regexp.MustCompile(`^memory \((lightweight-idle|lightweight-loaded)\): RSS ([0-9]{1,6}) MB, PSS ([0-9]{1,6}) MB in ([0-9]{1,4}) processes$`)

// androidLine matches the memory-android artifact: the release APK,
// cold-started on an emulator, from dumpsys meminfo after settling.
var androidLine = regexp.MustCompile(`^memory \((apk-cold)\): RSS ([0-9]{1,6}) MB, PSS ([0-9]{1,6}) MB in ([0-9]{1,4}) processes$`)

// LightweightMemory is what vfxtool measure-lightweight prints.
type LightweightMemory struct {
	Idle   Point
	Loaded Point
}

// AndroidMemory is what the measure job of android.yml writes.
type AndroidMemory struct {
	Cold Point
}

// ParseLightweightMemory parses the memory-lightweight artifact. It returns nil
// when the file is not exactly the two lightweight points.
func ParseLightweightMemory(text string) *LightweightMemory {
	points := map[string]Point{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		match := lightweightLine.FindStringSubmatch(line)
		if match == nil {
			return nil
		}
		if _, dup := points[match[1]]; dup {
			return nil
		}
		rss, _ := strconv.Atoi(match[2])
		pss, _ := strconv.Atoi(match[3])
		processes, _ := strconv.Atoi(match[4])
		points[match[1]] = Point{RSS: rss, PSS: pss, Processes: processes}
	}
	if len(points) != 2 {
		return nil
	}
	idle, okIdle := points["lightweight-idle"]
	loaded, okLoaded := points["lightweight-loaded"]
	if !okIdle || !okLoaded {
		return nil
	}
	return &LightweightMemory{Idle: idle, Loaded: loaded}
}

// ParseAndroidMemory parses the memory-android artifact. It returns nil when
// the file is not exactly the one APK point.
func ParseAndroidMemory(text string) *AndroidMemory {
	var cold *Point
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		match := androidLine.FindStringSubmatch(line)
		if match == nil || cold != nil {
			return nil
		}
		rss, _ := strconv.Atoi(match[2])
		pss, _ := strconv.Atoi(match[3])
		processes, _ := strconv.Atoi(match[4])
		cold = &Point{RSS: rss, PSS: pss, Processes: processes}
	}
	if cold == nil {
		return nil
	}
	return &AndroidMemory{Cold: *cold}
}

// ExtraRow is one measured number of an extra section: the pull request next
// to main.
type ExtraRow struct {
	Measure string
	PR      string
	Main    string
}

// Extra is one section of the report beyond the desktop table. Rows holds the
// table; when the section has no numbers, Note says why instead.
type Extra struct {
	Title       string
	Intro       string
	Rows        []ExtraRow
	Note        string
	RunURL      string
	BaselineURL string
	BaselineSHA string
}

// lightweightNoise is the difference from main that counts as a change for the
// lightweight mode: the same code varies as much as the desktop between runs.
func lightweightNoise(baseline int) float64 {
	return math.Max(MinChangeMB, DefaultTolerance*float64(baseline))
}

// androidNoise is the difference from main that counts as a change for the
// APK: emulator boots vary more than GitHub runners.
func androidNoise(baseline int) float64 {
	return math.Max(MinChangeMB, 0.10*float64(baseline))
}

// extraCell formats one number with its difference from main, in bold when
// more than the noise. Without a baseline of its own the pull request shows
// the numbers of main.
func extraCell(pr, baseline int, unit string, noise float64, inherited bool) (string, string) {
	main := fmt.Sprintf("%d %s", baseline, unit)
	if inherited {
		return fmt.Sprintf("%d %s (main)", pr, unit), main
	}
	if pr == baseline {
		return fmt.Sprintf("%d %s", pr, unit), main
	}
	difference := signed(pr - baseline)
	if math.Abs(float64(pr-baseline)) > noise {
		difference = "**" + difference + "**"
	}
	return fmt.Sprintf("%d %s (%s)", pr, unit, difference), main
}

// baselineLink records where the baseline numbers of an extra came from.
func baselineLink(extra *Extra, repo *Repository, artifact *github.Artifact) {
	run := artifact.GetWorkflowRun()
	if run == nil {
		return
	}
	extra.BaselineSHA = run.GetHeadSHA()
	extra.BaselineURL = runURL(repo, run.GetID())
}

// collectLightweight collects the lightweight-mode section from the mygo run.
func collectLightweight(ctx context.Context, repo *Repository, run *github.WorkflowRun, defaultBranch string) (Extra, error) {
	extra := Extra{
		Title: "Lightweight mode",
		Intro: "What the application uses with the web interface on and its window closed, with the fake vfox, in MB " +
			"for the application and all of its child processes: `lightweight-idle` is settled with no browser, " +
			"`lightweight-loaded` with a paired browser past its first inventory over server-sent events " +
			"(`vfxtool measure-lightweight`).",
	}
	if run != nil {
		extra.RunURL = run.GetHTMLURL()
	}
	var memory *LightweightMemory
	inherited := run == nil
	if run != nil && run.GetStatus() != "completed" {
		extra.Note = "Still running; the report is written again when it ends."
		return extra, nil
	}
	if run != nil {
		if text, ok, err := runArtifactText(ctx, repo, run, LightweightArtifact); err != nil {
			return extra, err
		} else if ok {
			memory = ParseLightweightMemory(text)
		}
	}
	var baseline *LightweightMemory
	if text, artifact, err := BaselineArtifact(ctx, repo, LightweightArtifact, defaultBranch); err != nil {
		return extra, err
	} else if text != "" {
		baseline = ParseLightweightMemory(text)
		if baseline != nil {
			baselineLink(&extra, repo, artifact)
		}
	}
	if memory == nil {
		if baseline != nil {
			inherited = true
			memory = baseline
		} else if run != nil {
			extra.Note = "No numbers: the job did not upload them."
			return extra, nil
		} else {
			return Extra{}, nil
		}
	}
	if baseline == nil {
		extra.Note = "No baseline: no run on main uploaded numbers yet."
		return extra, nil
	}
	for _, point := range []struct {
		label string
		now   Point
		base  Point
	}{
		{"Idle RSS", memory.Idle, baseline.Idle},
		{"Idle PSS", memory.Idle, baseline.Idle},
		{"Loaded RSS", memory.Loaded, baseline.Loaded},
		{"Loaded PSS", memory.Loaded, baseline.Loaded},
	} {
		value := point.now.RSS
		before := point.base.RSS
		if strings.HasSuffix(point.label, "PSS") {
			value = point.now.PSS
			before = point.base.PSS
		}
		prCell, mainCell := extraCell(value, before, "MB", lightweightNoise(before), inherited)
		extra.Rows = append(extra.Rows, ExtraRow{Measure: point.label, PR: prCell, Main: mainCell})
	}
	return extra, nil
}

// collectAndroid collects the APK section from the android run.
func collectAndroid(ctx context.Context, repo *Repository, run *github.WorkflowRun, defaultBranch string) (Extra, error) {
	extra := Extra{
		Title: "Android client",
		Intro: "What the release APK uses cold-started on an emulator, in MB from `dumpsys meminfo` after " +
			"settling: `apk-cold`. Emulator boots vary, so only differences of more than 5 MB and 10% count " +
			"as changes.",
	}
	if run != nil {
		extra.RunURL = run.GetHTMLURL()
	}
	var memory *AndroidMemory
	inherited := run == nil
	if run != nil && run.GetStatus() != "completed" {
		extra.Note = "Still running; the report is written again when it ends."
		return extra, nil
	}
	if run != nil {
		if text, ok, err := runArtifactText(ctx, repo, run, AndroidArtifact); err != nil {
			return extra, err
		} else if ok {
			memory = ParseAndroidMemory(text)
		}
	}
	var baseline *AndroidMemory
	if text, artifact, err := BaselineArtifact(ctx, repo, AndroidArtifact, defaultBranch); err != nil {
		return extra, err
	} else if text != "" {
		baseline = ParseAndroidMemory(text)
		if baseline != nil {
			baselineLink(&extra, repo, artifact)
		}
	}
	if memory == nil {
		if baseline != nil {
			inherited = true
			memory = baseline
		} else if run != nil {
			extra.Note = "No numbers: the job did not upload them."
			return extra, nil
		} else {
			return Extra{}, nil
		}
	}
	if baseline == nil {
		extra.Note = "No baseline: no run on main uploaded numbers yet."
		return extra, nil
	}
	for _, point := range []struct {
		label string
		value int
		base  int
	}{
		{"Cold RSS", memory.Cold.RSS, baseline.Cold.RSS},
		{"Cold PSS", memory.Cold.PSS, baseline.Cold.PSS},
	} {
		prCell, mainCell := extraCell(point.value, point.base, "MB", androidNoise(point.base), inherited)
		extra.Rows = append(extra.Rows, ExtraRow{Measure: point.label, PR: prCell, Main: mainCell})
	}
	return extra, nil
}

// CollectExtras returns the sections of the report beyond the desktop table:
// the lightweight mode from the mygo run, the APK from the
// android run. A section with neither numbers nor baseline is left out.
func CollectExtras(ctx context.Context, repo *Repository, runs map[string]*github.WorkflowRun, defaultBranch string) ([]Extra, error) {
	var extras []Extra
	lightweight, err := collectLightweight(ctx, repo, runs[workflowPath("mygo")], defaultBranch)
	if err != nil {
		return nil, err
	}
	android, err := collectAndroid(ctx, repo, runs[workflowPath("android")], defaultBranch)
	if err != nil {
		return nil, err
	}
	for _, extra := range []Extra{lightweight, android} {
		if extra.Title == "" {
			continue
		}
		extras = append(extras, extra)
	}
	return extras, nil
}

// RenderExtras returns the sections beyond the desktop table.
func RenderExtras(extras []Extra) string {
	if len(extras) == 0 {
		return ""
	}
	var lines []string
	for _, extra := range extras {
		lines = append(lines, "#### "+extra.Title, "")
		lines = append(lines, extra.Intro, "")
		if extra.Note != "" {
			lines = append(lines, "- "+extra.Note)
		} else {
			lines = append(lines,
				"| Measure | This pull request | Main |",
				"| --- | ---: | ---: |")
			for _, row := range extra.Rows {
				lines = append(lines, "| "+row.Measure+" | "+row.PR+" | "+row.Main+" |")
			}
		}
		var links []string
		if extra.RunURL != "" {
			links = append(links, fmt.Sprintf("[run](%s)", extra.RunURL))
		}
		if extra.BaselineURL != "" {
			short := extra.BaselineSHA
			if len(short) > 7 {
				short = short[:7]
			}
			links = append(links, fmt.Sprintf("[baseline](%s) (`%s`)", extra.BaselineURL, short))
		}
		if len(links) > 0 {
			lines = append(lines, "", "- "+strings.Join(links, ", ")+".")
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// InsertExtras splices the extra sections before the details block of the
// comment, so that the desktop table keeps its place at the top.
func InsertExtras(body, extras string) string {
	if extras == "" {
		return body
	}
	if i := strings.Index(body, "<details>"); i >= 0 {
		return body[:i] + extras + "\n" + body[i:]
	}
	return body + extras
}
