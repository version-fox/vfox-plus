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
// workflow, the APK from the android workflow. The private memory of the
// lightweight mode is an artifact of its own, so that memory-lightweight keeps
// the one format that the report has always read.
const (
	LightweightArtifact        = "memory-lightweight"
	LightweightPrivateArtifact = "memory-lightweight-private"
	AndroidArtifact            = "memory-android"
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

// lightweightPrivateLine matches one line of the memory-lightweight-private artifact: the private memory
// (the heap and the stacks, in KB) of the application and all of its child processes, at the same two
// points as the memory-lightweight artifact.
var lightweightPrivateLine = regexp.MustCompile(`^memory \((lightweight-idle|lightweight-loaded)\): private ([0-9]{1,9}) KB in ([0-9]{1,4}) processes$`)

// LightweightPrivate is the private memory of the two lightweight points, in KB.
type LightweightPrivate struct {
	Idle   int64
	Loaded int64
}

// ParseLightweightPrivate parses the memory-lightweight-private artifact. It returns nil when the file is
// not exactly the two lightweight points.
func ParseLightweightPrivate(text string) *LightweightPrivate {
	points := map[string]int64{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		match := lightweightPrivateLine.FindStringSubmatch(line)
		if match == nil {
			return nil
		}
		if _, dup := points[match[1]]; dup {
			return nil
		}
		kb, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return nil
		}
		points[match[1]] = kb
	}
	idle, okIdle := points["lightweight-idle"]
	loaded, okLoaded := points["lightweight-loaded"]
	if len(points) != 2 || !okIdle || !okLoaded {
		return nil
	}
	return &LightweightPrivate{Idle: idle, Loaded: loaded}
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
	Title string
	Intro string
	Rows  []ExtraRow
	Note  string
	// Remark is a line under the table, for what the table cannot show.
	Remark      string
	RunURL      string
	BaselineURL string
	BaselineSHA string
}

// MinPrivateChangeKB and PrivateTolerance: the private memory of the lightweight mode counts as changed from
// main when the difference is more than the larger of the two, 2 MB and 5 % of main. They are a first
// estimate: the private memory moved by about 1 MB between runs of one build on a devcontainer, and CI has
// not measured it yet. Revisit them with the numbers of several runs on main.
const (
	MinPrivateChangeKB = 2 * 1024
	PrivateTolerance   = 0.05
)

// privateNoise is the difference from main that counts as a change of the private memory, in KB.
func privateNoise(baseline int64) float64 {
	return math.Max(MinPrivateChangeKB, PrivateTolerance*float64(baseline))
}

// megabytes formats a number of KB as MB with one decimal, the precision that the private memory needs.
func megabytes(kb int64) string {
	return fmt.Sprintf("%.1f MB", float64(kb)/1024)
}

// signedMegabytes formats a difference in KB as MB with its sign. A difference that rounds to nothing
// is 0.0, without a sign.
func signedMegabytes(kb int64) string {
	value := float64(kb) / 1024
	if math.Round(value*10) == 0 {
		return "0.0"
	}
	return fmt.Sprintf("%+.1f", value)
}

// plainCell formats one resident or proportional number with its difference from main. The number is not
// judged: both include the pages of files that the system takes back, which vary between runs of one build.
func plainCell(pr, baseline int, inherited bool) (string, string) {
	main := fmt.Sprintf("%d MB", baseline)
	if inherited {
		return fmt.Sprintf("%d MB (main)", pr), main
	}
	if pr == baseline {
		return fmt.Sprintf("%d MB", pr), main
	}
	return fmt.Sprintf("%d MB (%s)", pr, signed(pr-baseline)), main
}

// plainRow is one row of resident or proportional memory.
func plainRow(label string, pr, baseline int, inherited bool) ExtraRow {
	cell, main := plainCell(pr, baseline, inherited)
	return ExtraRow{Measure: label, PR: cell, Main: main}
}

// privateRow is one row of private memory: the pull request next to main, with the difference in bold when
// it is more than the noise. A number that main does not have is shown as a dash, and there is no
// difference to show without it.
func privateRow(label string, pr, baseline *int64, inherited bool) ExtraRow {
	row := ExtraRow{Measure: label, PR: "-", Main: "-"}
	if baseline != nil {
		row.Main = megabytes(*baseline)
	}
	switch {
	case inherited:
		if pr != nil {
			row.PR = megabytes(*pr) + " (main)"
		}
	case pr != nil && baseline != nil:
		row.PR = megabytes(*pr)
		if diff := *pr - *baseline; diff != 0 {
			text := signedMegabytes(diff)
			if math.Abs(float64(diff)) > privateNoise(*baseline) {
				text = "**" + text + "**"
			}
			row.PR += " (" + text + ")"
		}
	case pr != nil:
		row.PR = megabytes(*pr)
	}
	return row
}

// privateValues returns the private memory of the idle and the loaded point, or nils without numbers.
func privateValues(private *LightweightPrivate) (*int64, *int64) {
	if private == nil {
		return nil, nil
	}
	idle, loaded := private.Idle, private.Loaded
	return &idle, &loaded
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
			"(`vfxtool measure-lightweight`). A change is judged by the private memory, the heap and the stacks, " +
			"and only by a difference of more than the larger of 2 MB and 5 % of main. The resident and the " +
			"proportional memory include the pages of files that the system takes back, so they are shown " +
			"without a judgement.",
	}
	if run != nil {
		extra.RunURL = run.GetHTMLURL()
	}
	var memory *LightweightMemory
	var private *LightweightPrivate
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
		if text, ok, err := runArtifactText(ctx, repo, run, LightweightPrivateArtifact); err != nil {
			return extra, err
		} else if ok {
			private = ParseLightweightPrivate(text)
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
	// main has the private memory only from the runs that measured it: until then, its rows are dashes.
	var baselinePrivate *LightweightPrivate
	if text, _, err := BaselineArtifact(ctx, repo, LightweightPrivateArtifact, defaultBranch); err != nil {
		return extra, err
	} else if text != "" {
		baselinePrivate = ParseLightweightPrivate(text)
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
	if inherited {
		// The pull request did not measure, so its numbers are those of main, private memory included.
		private = baselinePrivate
	}
	if baselinePrivate == nil {
		extra.Remark = "Main has no private memory yet: the next run on main writes it, and the private rows compare from then on."
	} else if private == nil {
		extra.Remark = "This pull request uploaded no private memory."
	}
	idle, loaded := privateValues(private)
	baseIdle, baseLoaded := privateValues(baselinePrivate)
	extra.Rows = []ExtraRow{
		plainRow("Idle RSS", memory.Idle.RSS, baseline.Idle.RSS, inherited),
		plainRow("Idle PSS", memory.Idle.PSS, baseline.Idle.PSS, inherited),
		privateRow("Idle private memory", idle, baseIdle, inherited),
		plainRow("Loaded RSS", memory.Loaded.RSS, baseline.Loaded.RSS, inherited),
		plainRow("Loaded PSS", memory.Loaded.PSS, baseline.Loaded.PSS, inherited),
		privateRow("Loaded private memory", loaded, baseLoaded, inherited),
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
			if extra.Remark != "" {
				lines = append(lines, "", "- "+extra.Remark)
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
