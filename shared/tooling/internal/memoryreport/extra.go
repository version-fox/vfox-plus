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
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// The artifacts of the measurements beyond the desktop table, by upload name:
// the headless server and the web client from the mygo workflow, the APK from
// the android workflow.
const (
	ServeArtifact   = "memory-serve"
	WebArtifact     = "memory-web"
	AndroidArtifact = "memory-android"
)

// serveLine matches one line of the memory-serve artifact: the server and all
// of its child processes, settled with no browser (serve-idle) or with a
// paired browser past its first inventory (serve-loaded).
var serveLine = regexp.MustCompile(`^memory \((serve-idle|serve-loaded)\): RSS ([0-9]{1,6}) MB, PSS ([0-9]{1,6}) MB in ([0-9]{1,4}) processes$`)

// webLine matches one line of the memory-web artifact: the JavaScript heap in
// bytes of the paired page, after the SDK list settles (app) and after twenty
// seconds idle on the open page (app-settled).
var webLine = regexp.MustCompile(`^webheap \((app|app-settled)\): ([0-9]{1,12}) bytes$`)

// androidLine matches the memory-android artifact: the release APK,
// cold-started on an emulator, from dumpsys meminfo after settling.
var androidLine = regexp.MustCompile(`^memory \((apk-cold)\): RSS ([0-9]{1,6}) MB, PSS ([0-9]{1,6}) MB in ([0-9]{1,4}) processes$`)

// ServeMemory is what vfxtool measure-serve prints.
type ServeMemory struct {
	Idle   Point
	Loaded Point
}

// WebHeap is what vfxtool webheap writes, in bytes.
type WebHeap struct {
	App     int64
	Settled int64
}

// AndroidMemory is what the measure job of android.yml writes.
type AndroidMemory struct {
	Cold Point
}

// ParseServeMemory parses the memory-serve artifact. It returns nil when the
// file is not exactly the two serve points.
func ParseServeMemory(text string) *ServeMemory {
	points := map[string]Point{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		match := serveLine.FindStringSubmatch(line)
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
	idle, okIdle := points["serve-idle"]
	loaded, okLoaded := points["serve-loaded"]
	if !okIdle || !okLoaded {
		return nil
	}
	return &ServeMemory{Idle: idle, Loaded: loaded}
}

// ParseWebHeap parses the memory-web artifact. It returns nil when the file
// is not exactly the two heap numbers.
func ParseWebHeap(text string) *WebHeap {
	numbers := map[string]int64{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		match := webLine.FindStringSubmatch(line)
		if match == nil {
			return nil
		}
		if _, dup := numbers[match[1]]; dup {
			return nil
		}
		value, _ := strconv.ParseInt(match[2], 10, 64)
		numbers[match[1]] = value
	}
	if len(numbers) != 2 {
		return nil
	}
	app, okApp := numbers["app"]
	settled, okSettled := numbers["app-settled"]
	if !okApp || !okSettled {
		return nil
	}
	return &WebHeap{App: app, Settled: settled}
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

// runArtifactText returns the content of the newest live artifact with a name
// of one run, or false when the run has none.
func runArtifactText(gh API, run map[string]any, name string) (string, bool, error) {
	value, err := gh.Get(fmt.Sprintf("actions/runs/%d/artifacts", asInt(run["id"])),
		map[string]any{"name": name, "per_page": 100})
	if err != nil {
		return "", false, err
	}
	response, _ := value.(map[string]any)
	var artifacts []map[string]any
	for _, item := range asList(response["artifacts"]) {
		if artifact, ok := item.(map[string]any); ok {
			artifacts = append(artifacts, artifact)
		}
	}
	if artifact := Newest(artifacts); artifact != nil {
		if text, ok := readArtifactText(gh, artifact); ok {
			return text, true, nil
		}
	}
	return "", false, nil
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

// serveNoise is the difference from main that counts as a change for the
// headless server: the same code varies as much as the desktop between runs.
func serveNoise(baseline int) float64 {
	return math.Max(MinChangeMB, DefaultTolerance*float64(baseline))
}

// webNoiseKB is the difference from main that counts as a change for the web
// heap in KB: the page is deterministic, but Chrome is not the same build
// forever.
func webNoiseKB(baseline int) float64 {
	return math.Max(100, DefaultTolerance*float64(baseline))
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
func baselineLink(extra *Extra, gh API, artifact map[string]any) {
	run, _ := artifact["workflow_run"].(map[string]any)
	if run == nil {
		return
	}
	extra.BaselineSHA = asString(run["head_sha"])
	extra.BaselineURL = fmt.Sprintf("https://github.com/%s/actions/runs/%d", gh.RepoName(), asInt(run["id"]))
}

// collectServe collects the headless-server section from the mygo run.
func collectServe(gh API, run map[string]any, defaultBranch string) (Extra, error) {
	extra := Extra{
		Title: "Headless server",
		Intro: "What `vfox-plus-serve serve` uses on loopback with the fake vfox, in MB for the server " +
			"and all of its child processes: `serve-idle` is settled with no browser, `serve-loaded` with a " +
			"paired browser past its first inventory over server-sent events (`vfxtool measure-serve`).",
	}
	if run != nil {
		extra.RunURL = asString(run["html_url"])
	}
	var memory *ServeMemory
	inherited := run == nil
	if run != nil && asString(run["status"]) != "completed" {
		extra.Note = "Still running; the report is written again when it ends."
		return extra, nil
	}
	if run != nil {
		if text, ok, err := runArtifactText(gh, run, ServeArtifact); err != nil {
			return extra, err
		} else if ok {
			memory = ParseServeMemory(text)
		}
	}
	var baseline *ServeMemory
	if text, artifact, err := BaselineArtifact(gh, ServeArtifact, defaultBranch); err != nil {
		return extra, err
	} else if text != "" {
		baseline = ParseServeMemory(text)
		if baseline != nil {
			baselineLink(&extra, gh, artifact)
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
		prCell, mainCell := extraCell(value, before, "MB", serveNoise(before), inherited)
		extra.Rows = append(extra.Rows, ExtraRow{Measure: point.label, PR: prCell, Main: mainCell})
	}
	return extra, nil
}

// collectWeb collects the web-client-heap section from the mygo run.
func collectWeb(gh API, run map[string]any, defaultBranch string) (Extra, error) {
	extra := Extra{
		Title: "Web client heap",
		Intro: "What the paired `/app` page uses in headless Chrome through the Chrome DevTools Protocol, " +
			"in KB: `app` after the SDK list settles, `app-settled` after twenty seconds idle on the open " +
			"page with live events (`vfxtool webheap`).",
	}
	if run != nil {
		extra.RunURL = asString(run["html_url"])
	}
	var heap *WebHeap
	inherited := run == nil
	if run != nil && asString(run["status"]) != "completed" {
		extra.Note = "Still running; the report is written again when it ends."
		return extra, nil
	}
	if run != nil {
		if text, ok, err := runArtifactText(gh, run, WebArtifact); err != nil {
			return extra, err
		} else if ok {
			heap = ParseWebHeap(text)
		}
	}
	var baseline *WebHeap
	if text, artifact, err := BaselineArtifact(gh, WebArtifact, defaultBranch); err != nil {
		return extra, err
	} else if text != "" {
		baseline = ParseWebHeap(text)
		if baseline != nil {
			baselineLink(&extra, gh, artifact)
		}
	}
	if heap == nil {
		if baseline != nil {
			inherited = true
			heap = baseline
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
	toKB := func(bytes int64) int { return int((bytes + 512) / 1024) }
	for _, point := range []struct {
		label string
		pr    int
		base  int
	}{
		{"App heap", toKB(heap.App), toKB(baseline.App)},
		{"App heap settled", toKB(heap.Settled), toKB(baseline.Settled)},
	} {
		prCell, mainCell := extraCell(point.pr, point.base, "KB", webNoiseKB(point.base), inherited)
		extra.Rows = append(extra.Rows, ExtraRow{Measure: point.label, PR: prCell, Main: mainCell})
	}
	return extra, nil
}

// collectAndroid collects the APK section from the android run.
func collectAndroid(gh API, run map[string]any, defaultBranch string) (Extra, error) {
	extra := Extra{
		Title: "Android client",
		Intro: "What the release APK uses cold-started on an emulator, in MB from `dumpsys meminfo` after " +
			"settling: `apk-cold`. Emulator boots vary, so only differences of more than 5 MB and 10% count " +
			"as changes.",
	}
	if run != nil {
		extra.RunURL = asString(run["html_url"])
	}
	var memory *AndroidMemory
	inherited := run == nil
	if run != nil && asString(run["status"]) != "completed" {
		extra.Note = "Still running; the report is written again when it ends."
		return extra, nil
	}
	if run != nil {
		if text, ok, err := runArtifactText(gh, run, AndroidArtifact); err != nil {
			return extra, err
		} else if ok {
			memory = ParseAndroidMemory(text)
		}
	}
	var baseline *AndroidMemory
	if text, artifact, err := BaselineArtifact(gh, AndroidArtifact, defaultBranch); err != nil {
		return extra, err
	} else if text != "" {
		baseline = ParseAndroidMemory(text)
		if baseline != nil {
			baselineLink(&extra, gh, artifact)
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
// the headless server and the web client from the mygo run, the APK from the
// android run. A section with neither numbers nor baseline is left out.
func CollectExtras(gh API, runs map[string]map[string]any, defaultBranch string) ([]Extra, error) {
	var extras []Extra
	serve, err := collectServe(gh, runs[workflowPath("mygo")], defaultBranch)
	if err != nil {
		return nil, err
	}
	web, err := collectWeb(gh, runs[workflowPath("mygo")], defaultBranch)
	if err != nil {
		return nil, err
	}
	android, err := collectAndroid(gh, runs[workflowPath("android")], defaultBranch)
	if err != nil {
		return nil, err
	}
	for _, extra := range []Extra{serve, web, android} {
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
