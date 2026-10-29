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

// Package memoryreport comments on a pull request with the memory that the
// application uses.
//
// It is a port of .github/scripts/memory_report.py to Go, using the go-github
// client library. The workflow of the implementation measures the application
// and uploads the result as the artifact memory-mygo, which holds one file,
// memory.txt, with the lines "memory (<point>): RSS <n> MB, PSS <n> MB in <n>
// processes" of the points home and nodejs. The memory workflow runs this tool
// when that workflow ends for the latest commit of a pull request, and the
// tool writes one comment: a table like the one of docs/en/dev/memory.md, with the
// difference from the latest measurement on the default branch next to the
// numbers. A later run replaces that comment instead of adding another one.
//
// Usage:
//
//	memory-report --run RUN_ID [--dry-run]
//	    Reports on the pull request of the workflow run RUN_ID.
//
//	memory-report --pull-request NUMBER [--dry-run]
//	    Reports on that pull request now, with the implementation still running
//	    marked as such.
//
// --dry-run prints the comment instead of writing it. The implementations come
// from --implementations or MEMORY_IMPLEMENTATIONS (names separated by spaces).
// GH_TOKEN authenticates the client of the API of GitHub. The artifacts come from the
// pull request, so they are not trusted: only the numbers in the lines above
// are read from them.
package memoryreport

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v93/github"
)

// Marker is the first line of the comment, which finds it again.
const Marker = "<!-- vfox-plus-memory-report -->"

// Bot is the author of the comments that GITHUB_TOKEN writes.
const Bot = "github-actions[bot]"

// workflowsDir holds the workflow of each implementation.
const workflowsDir = ".github/workflows"

// MemoryFile is the one file that a memory artifact holds.
const MemoryFile = "memory.txt"

// MAX_ARTIFACT_BYTES: memory.txt has two short lines; an artifact much larger
// than that is not one.
const MaxArtifactBytes = 64 * 1024

// MaxMemoryFileBytes caps the memory.txt read from an artifact.
const MaxMemoryFileBytes = 4 * 1024

// MinChangeMB and DefaultTolerance: the same code does not use the same
// memory in every CI run. Only a larger difference counts as a change.
const MinChangeMB = 5
const DefaultTolerance = 0.05

// Tolerances holds wider tolerances for implementations that vary more, by
// implementation. It mirrors TOLERANCES of the script.
var Tolerances = map[string]float64{}

// line matches one line of memory.txt.
var line = regexp.MustCompile(`^memory \((home|nodejs)\): RSS ([0-9]{1,6}) MB, PSS ([0-9]{1,6}) MB in ([0-9]{1,4}) processes$`)

// implementationName matches an implementation, such as "mygo".
var implementationName = regexp.MustCompile(`\A[a-z0-9][a-z0-9-]*\z`)

// repositoryName matches OWNER/NAME.
var repositoryName = regexp.MustCompile(`\A[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+\z`)

// whitespace collapses the text of a table cell onto one line.
var whitespace = regexp.MustCompile(`\s+`)

// technologyLink finds the implementation directory in a table cell. The
// documentation lives under docs/<language>/<audience>/, so the link climbs
// any number of directories before the implementation directory.
var technologyLink = regexp.MustCompile(`\]\((?:\.\./)+([A-Za-z0-9_-]+)/README\.md`)

// Point is the memory at one measuring point, in MB.
type Point struct {
	RSS       int
	PSS       int
	Processes int
}

// Memory is what an implementation uses on the Home page and with the versions
// of Node.js listed.
type Memory struct {
	Home     Point
	Versions Point
}

// Row is one implementation of the report.
//
// The state is "measured" (the pull request rebuilt it), "inherited" (it did
// not, so the numbers are those of the default branch), "running" or "failed"
// (the workflow of the pull request has not ended, or ended without numbers)
// or "missing" (no workflow ran and the default branch has no numbers).
type Row struct {
	Implementation string
	Name           string
	Technology     string
	State          string
	Memory         *Memory
	Baseline       *Memory
	RunURL         string
	BaselineURL    string
	BaselineSHA    string
}

// Shown returns the numbers of the table.
func (r Row) Shown() *Memory {
	if r.State == "measured" {
		return r.Memory
	}
	if r.State == "inherited" {
		return r.Baseline
	}
	return nil
}

// ParseMemory parses memory.txt: one line for the point home and one for
// nodejs, nothing else. It returns nil when the file is not exactly that.
func ParseMemory(text string) *Memory {
	points := map[string]Point{}
	for _, raw := range strings.Split(text, "\n") {
		lineText := strings.TrimSpace(raw)
		if lineText == "" {
			continue
		}
		match := line.FindStringSubmatch(lineText)
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
	home, okHome := points["home"]
	versions, okVersions := points["nodejs"]
	if !okHome || !okVersions {
		return nil
	}
	return &Memory{Home: home, Versions: versions}
}

// Tolerance is the difference in MB from the baseline that is still the noise
// of CI.
func Tolerance(implementation string, baseline int) float64 {
	tolerance, ok := Tolerances[implementation]
	if !ok {
		tolerance = DefaultTolerance
	}
	return math.Max(MinChangeMB, tolerance*float64(baseline))
}

// ExceedsNoise reports whether the difference is a change and not the
// variation between CI runs.
func ExceedsNoise(implementation string, baseline, current int) bool {
	return math.Abs(float64(current-baseline)) > Tolerance(implementation, baseline)
}

// signed formats a difference with its sign.
func signed(number int) string {
	if number > 0 {
		return fmt.Sprintf("+%d", number)
	}
	return strconv.Itoa(number)
}

// cell returns text that cannot break out of a table cell.
func cell(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(whitespace.ReplaceAllString(text, " "), "|", "\\|"))
}

// ReadTechnologies reads the Technology column of the first table of
// docs/en/dev/memory.md, by implementation directory.
func ReadTechnologies(text string) map[string]string {
	technologies := map[string]string{}
	var columns [2]int
	haveColumns := false
	inTable := false
	for _, raw := range strings.Split(text, "\n") {
		if !strings.HasPrefix(raw, "|") {
			if inTable {
				break
			}
			continue
		}
		inTable = true
		parts := strings.Split(strings.Trim(strings.TrimSpace(raw), "|"), "|")
		cells := make([]string, len(parts))
		for i, part := range parts {
			cells[i] = strings.TrimSpace(part)
		}
		if !haveColumns {
			implementation := -1
			technology := -1
			for i, c := range cells {
				if c == "Implementation" {
					implementation = i
				}
				if c == "Technology" {
					technology = i
				}
			}
			if implementation >= 0 && technology >= 0 {
				columns = [2]int{implementation, technology}
				haveColumns = true
			}
			continue
		}
		if len(cells) <= columns[0] || len(cells) <= columns[1] {
			continue
		}
		if link := technologyLink.FindStringSubmatch(cells[columns[0]]); link != nil {
			technologies[link[1]] = cells[columns[1]]
		}
	}
	return technologies
}

// PssCell returns the PSS of a point, with the difference from the default
// branch if it was measured.
func PssCell(row Row, point string) string {
	var shown, before int
	if point == "home" {
		shown = row.Shown().Home.PSS
	} else {
		shown = row.Shown().Versions.PSS
	}
	if row.State != "measured" || row.Baseline == nil {
		return strconv.Itoa(shown)
	}
	if point == "home" {
		before = row.Baseline.Home.PSS
	} else {
		before = row.Baseline.Versions.PSS
	}
	difference := signed(shown - before)
	if ExceedsNoise(row.Implementation, before, shown) {
		difference = "**" + difference + "**"
	}
	return fmt.Sprintf("%d (%s)", shown, difference)
}

// TableCells returns the cells of the row of the table.
func TableCells(row Row) []string {
	name := row.Name
	if row.State == "inherited" {
		name += " (main)"
	} else if row.State == "measured" && row.Baseline == nil {
		name += " (no baseline)"
	}
	shown := row.Shown()
	if shown == nil {
		labels := map[string]string{"running": "running", "failed": "no result", "missing": "no data"}
		label := labels[row.State]
		if row.RunURL != "" {
			label = fmt.Sprintf("[%s](%s)", label, row.RunURL)
		}
		return []string{cell(name), cell(row.Technology), "-", label, "-", "-", "-", "-"}
	}
	processes := strconv.Itoa(shown.Home.Processes)
	if row.State == "measured" && row.Baseline != nil {
		// The number of processes of an implementation does not vary between CI runs.
		if difference := shown.Home.Processes - row.Baseline.Home.Processes; difference != 0 {
			processes += fmt.Sprintf(" (**%s**)", signed(difference))
		}
	}
	return []string{
		cell(name),
		cell(row.Technology),
		processes,
		PssCell(row, "home"),
		strconv.Itoa(shown.Home.RSS),
		PssCell(row, "versions"),
		strconv.Itoa(shown.Versions.RSS),
		signed(shown.Versions.PSS - shown.Home.PSS),
	}
}

// sortRows sorts the rows by state and by the PSS of the Home page.
func sortRows(rows []Row) {
	group := func(state string) int {
		switch state {
		case "measured", "inherited":
			return 0
		case "running":
			return 1
		case "failed":
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(rows, func(a, b int) bool {
		ga, gb := group(rows[a].State), group(rows[b].State)
		if ga != gb {
			return ga < gb
		}
		var pa, pb int
		if shown := rows[a].Shown(); shown != nil {
			pa = shown.Home.PSS
		}
		if shown := rows[b].Shown(); shown != nil {
			pb = shown.Home.PSS
		}
		if pa != pb {
			return pa < pb
		}
		return strings.ToLower(rows[a].Name) < strings.ToLower(rows[b].Name)
	})
}

// ChangesOf returns the differences from the default branch that are more than
// the noise.
func ChangesOf(row Row) []string {
	if row.State != "measured" || row.Memory == nil || row.Baseline == nil {
		return nil
	}
	var changes []string
	for _, point := range []struct {
		label string
		point string
	}{
		{"Home PSS", "home"},
		{"Versions PSS", "versions"},
	} {
		var before, after int
		if point.point == "home" {
			before, after = row.Baseline.Home.PSS, row.Memory.Home.PSS
		} else {
			before, after = row.Baseline.Versions.PSS, row.Memory.Versions.PSS
		}
		if ExceedsNoise(row.Implementation, before, after) {
			percent := 0
			if before != 0 {
				percent = int(math.Round(100 * float64(after-before) / float64(before)))
			}
			changes = append(changes, fmt.Sprintf("%s %s MB (%s%%)", point.label, signed(after-before), signed(percent)))
		}
	}
	if row.Memory.Home.Processes != row.Baseline.Home.Processes {
		changes = append(changes, fmt.Sprintf("processes %d -> %d", row.Baseline.Home.Processes, row.Memory.Home.Processes))
	}
	return changes
}

// Summary returns the lines above the table: what changed, and what has no
// numbers.
func Summary(rows []Row) []string {
	var lines []string
	var compared []Row
	for _, row := range rows {
		if row.State == "measured" && row.Baseline != nil {
			compared = append(compared, row)
		}
	}
	changed := map[string][]string{}
	for _, row := range compared {
		if changes := ChangesOf(row); len(changes) > 0 {
			changed[row.Name] = changes
		}
	}
	if len(changed) > 0 {
		var names []string
		for _, row := range compared {
			if _, ok := changed[row.Name]; ok {
				names = append(names, row.Name)
			}
		}
		var details []string
		for _, name := range names {
			details = append(details, fmt.Sprintf("%s: %s", name, strings.Join(changed[name], ", ")))
		}
		lines = append(lines, fmt.Sprintf("Changed from main beyond the noise of CI: %s.", strings.Join(details, "; ")))
	}
	var same []string
	for _, row := range compared {
		if _, ok := changed[row.Name]; !ok {
			same = append(same, row.Name)
		}
	}
	if len(same) > 0 && len(changed) > 0 {
		lines = append(lines, fmt.Sprintf("Within the noise of CI: %s.", strings.Join(same, ", ")))
	} else if len(same) > 0 {
		lines = append(lines, fmt.Sprintf("No change from main beyond the noise of CI in %s.", strings.Join(same, ", ")))
	}
	var withoutBaseline, running, failed, missing []string
	for _, row := range rows {
		switch {
		case row.State == "measured" && row.Baseline == nil:
			withoutBaseline = append(withoutBaseline, row.Name)
		case row.State == "running":
			running = append(running, row.Name)
		case row.State == "failed":
			failed = append(failed, row.Name)
		case row.State == "missing":
			missing = append(missing, row.Name)
		}
	}
	for _, group := range []struct {
		text  string
		names []string
	}{
		{"No numbers on main to compare with yet", withoutBaseline},
		{"Still running", running},
		{"No result, the run failed or was cancelled", failed},
		{"No numbers anywhere", missing},
	} {
		if len(group.names) > 0 {
			lines = append(lines, fmt.Sprintf("%s: %s.", group.text, strings.Join(group.names, ", ")))
		}
	}
	allInheritedOrMissing := true
	for _, row := range rows {
		if row.State != "inherited" && row.State != "missing" {
			allInheritedOrMissing = false
			break
		}
	}
	if allInheritedOrMissing {
		lines = append(lines, "This pull request did not rebuild any implementation, so the numbers are those of main.")
	}
	return lines
}

// Explanation explains the noise that the report allows.
func Explanation(rows []Row) string {
	wider := map[float64][]string{}
	for _, row := range rows {
		if tolerance, ok := Tolerances[row.Implementation]; ok {
			wider[tolerance] = append(wider[tolerance], row.Name)
		}
	}
	var tolerances []float64
	for tolerance := range wider {
		tolerances = append(tolerances, tolerance)
	}
	sort.Float64s(tolerances)
	var exceptions []string
	for _, tolerance := range tolerances {
		exceptions = append(exceptions, fmt.Sprintf("%.0f%% for %s", tolerance*100, strings.Join(wider[tolerance], " and ")))
	}
	text := fmt.Sprintf("The same code does not use the same memory in every CI run, so a difference counts as a change only "+
		"when it is more than %d MB and more than %.0f%% of the number on main", MinChangeMB, DefaultTolerance*100)
	if len(exceptions) > 0 {
		text += fmt.Sprintf(" (%s, which vary more)", strings.Join(exceptions, ", "))
	}
	return text + "."
}

// Render returns the comment.
func Render(rows []Row, headSHA string) string {
	rows = append([]Row(nil), rows...)
	sortRows(rows)
	header := []string{"Implementation", "Technology", "Processes", "Home PSS", "Home RSS", "Versions PSS", "Versions RSS", "The list adds (PSS)"}
	alignment := []string{"---", "---", "---:", "---:", "---:", "---:", "---:", "---:"}
	var body [][]string
	for _, row := range rows {
		body = append(body, TableCells(row))
	}
	technologies := false
	for _, row := range rows {
		if row.Technology != "" {
			technologies = true
			break
		}
	}
	if !technologies {
		// Without the table of docs/en/dev/memory.md there is nothing to put there.
		header = append(header[:1], header[2:]...)
		alignment = append(alignment[:1], alignment[2:]...)
		for i := range body {
			body[i] = append(body[i][:1], body[i][2:]...)
		}
	}
	var table []string
	for _, line := range append([][]string{header, alignment}, body...) {
		table = append(table, "| "+strings.Join(line, " | ")+" |")
	}
	var runs []string
	for _, row := range rows {
		if row.RunURL != "" {
			runs = append(runs, fmt.Sprintf("[%s](%s)", row.Name, row.RunURL))
		}
	}
	var baselines []string
	for _, row := range rows {
		if row.BaselineURL != "" {
			short := row.BaselineSHA
			if len(short) > 7 {
				short = short[:7]
			}
			baselines = append(baselines, fmt.Sprintf("[%s](%s) (`%s`)", row.Name, row.BaselineURL, short))
		}
	}
	details := []string{
		"- Measured by `vfxtool measure` in the workflow of the " +
			"implementation, on a GitHub-hosted runner with Xvfb and no GPU. These are not the numbers of " +
			"`docs/en/dev/memory.md`, which come from another machine: compare the rows of one comment.",
		"- " + Explanation(rows),
	}
	if len(runs) > 0 {
		details = append(details, "- Runs of this pull request: "+strings.Join(runs, ", ")+".")
	}
	if len(baselines) > 0 {
		details = append(details, "- Latest runs on main: "+strings.Join(baselines, ", ")+".")
	}
	short := headSHA
	if len(short) > 7 {
		short = short[:7]
	}
	lines := []string{
		Marker,
		fmt.Sprintf("### Memory use at `%s`", short),
		"",
	}
	for _, line := range Summary(rows) {
		lines = append(lines, "- "+line)
	}
	lines = append(lines, "")
	lines = append(lines, table...)
	lines = append(lines, "",
		"In MB, for the application and all of its child processes, sorted by the PSS of the Home page; "+
			"\"Versions\" is the SDKs page with the available versions of Node.js listed. A number in parentheses is "+
			"the difference from main, in bold when it is more than the noise of CI. "+
			"\"(main)\" marks an implementation that this pull request did not rebuild: its numbers are those of main.",
		"",
		"<details>",
		"<summary>Details</summary>",
		"",
	)
	lines = append(lines, details...)
	lines = append(lines, "</details>", "")
	return strings.Join(lines, "\n")
}

// writers relate the report: the CLI prints the comment to stdout and logs to
// stderr. They are variables so that tests can capture them.
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

func logMessage(message string) {
	fmt.Fprintln(stderr, message)
}

// readArtifactText returns the content of an artifact, or false when it is not
// a readable memory.txt. ReadArtifact and the extra sections share it.
func readArtifactText(ctx context.Context, repo *Repository, artifact *github.Artifact) (string, bool) {
	if artifact.GetExpired() || artifact.GetSizeInBytes() > MaxArtifactBytes {
		return "", false
	}
	data, err := repo.DownloadArtifact(ctx, artifact.GetID())
	if err != nil {
		logMessage(fmt.Sprintf("Cannot read the artifact %s (%d): %v", artifact.GetName(), artifact.GetID(), err))
		return "", false
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		logMessage(fmt.Sprintf("Cannot read the artifact %s (%d): %v", artifact.GetName(), artifact.GetID(), err))
		return "", false
	}
	if len(reader.File) != 1 || reader.File[0].Name != MemoryFile {
		return "", false
	}
	file, err := reader.File[0].Open()
	if err != nil {
		return "", false
	}
	content, err := io.ReadAll(io.LimitReader(file, MaxMemoryFileBytes+1))
	file.Close()
	if err != nil {
		return "", false
	}
	if len(content) > MaxMemoryFileBytes {
		return "", false
	}
	return string(content), true
}

// ReadArtifact returns the numbers in an artifact, or nil if it is not a valid
// memory.txt.
func ReadArtifact(ctx context.Context, repo *Repository, artifact *github.Artifact) *Memory {
	text, ok := readArtifactText(ctx, repo, artifact)
	if !ok {
		return nil
	}
	return ParseMemory(text)
}

// Newest returns the newest live artifact, or nil when all have expired.
func Newest(artifacts []*github.Artifact) *github.Artifact {
	var newest *github.Artifact
	for _, artifact := range artifacts {
		if artifact.GetExpired() {
			continue
		}
		if newest == nil || artifact.GetCreatedAt().After(newest.GetCreatedAt().Time) {
			newest = artifact
		}
	}
	return newest
}

// runArtifacts returns the artifacts of a workflow run that have a name. The
// endpoint of a run has no filter for the name.
func runArtifacts(ctx context.Context, repo *Repository, runID int64, name string) ([]*github.Artifact, error) {
	list, _, err := repo.Client.Actions.ListWorkflowRunArtifacts(ctx, repo.Owner, repo.Name, runID, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, err
	}
	var named []*github.Artifact
	for _, artifact := range list.Artifacts {
		if artifact.GetName() == name {
			named = append(named, artifact)
		}
	}
	return named, nil
}

// runArtifactText returns the content of the newest live artifact with a name
// of one run, or false when the run has none.
func runArtifactText(ctx context.Context, repo *Repository, run *github.WorkflowRun, name string) (string, bool, error) {
	artifacts, err := runArtifacts(ctx, repo, run.GetID(), name)
	if err != nil {
		return "", false, err
	}
	if artifact := Newest(artifacts); artifact != nil {
		if text, ok := readArtifactText(ctx, repo, artifact); ok {
			return text, true, nil
		}
	}
	return "", false, nil
}

// baselineCandidate is one readable baseline artifact, newest first.
type baselineCandidate struct {
	text     string
	artifact *github.Artifact
}

// baselineCandidates returns the readable artifacts with a name that were
// measured on the default branch, newest first.
func baselineCandidates(ctx context.Context, repo *Repository, name, defaultBranch string) ([]baselineCandidate, error) {
	list, _, err := repo.Client.Actions.ListArtifacts(ctx, repo.Owner, repo.Name, &github.ListArtifactsOptions{
		Name:        &name,
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}
	var onDefaultBranch []*github.Artifact
	for _, artifact := range list.Artifacts {
		// A pull request from a fork can have a branch called like the default branch.
		run := artifact.GetWorkflowRun()
		if run == nil || run.GetHeadBranch() != defaultBranch || run.GetHeadRepositoryID() != run.GetRepositoryID() {
			continue
		}
		onDefaultBranch = append(onDefaultBranch, artifact)
	}
	sort.SliceStable(onDefaultBranch, func(a, b int) bool {
		return onDefaultBranch[a].GetCreatedAt().After(onDefaultBranch[b].GetCreatedAt().Time)
	})
	var candidates []baselineCandidate
	for _, artifact := range onDefaultBranch {
		if text, ok := readArtifactText(ctx, repo, artifact); ok {
			candidates = append(candidates, baselineCandidate{text: text, artifact: artifact})
		}
	}
	return candidates, nil
}

// BaselineArtifact returns the content of the latest artifact with a name that
// was measured on the default branch, with the artifact. BaselineOf and the
// extra sections share it; like Collect, the newest readable artifact wins,
// and parsing it is the caller's job.
func BaselineArtifact(ctx context.Context, repo *Repository, name, defaultBranch string) (string, *github.Artifact, error) {
	candidates, err := baselineCandidates(ctx, repo, name, defaultBranch)
	if err != nil {
		return "", nil, err
	}
	if len(candidates) == 0 {
		return "", nil, nil
	}
	return candidates[0].text, candidates[0].artifact, nil
}

// BaselineOf returns the latest numbers of an implementation that were
// measured on the default branch.
func BaselineOf(ctx context.Context, repo *Repository, implementation, defaultBranch string) (*Memory, *github.Artifact, error) {
	candidates, err := baselineCandidates(ctx, repo, "memory-"+implementation, defaultBranch)
	if err != nil {
		return nil, nil, err
	}
	for _, candidate := range candidates {
		if memory := ParseMemory(candidate.text); memory != nil {
			return memory, candidate.artifact, nil
		}
	}
	return nil, nil, nil
}

// runURL is the page of a workflow run.
func runURL(repo *Repository, runID int64) string {
	return fmt.Sprintf("https://github.com/%s/actions/runs/%d", repo.FullName(), runID)
}

// FindPullRequests returns the open pull requests whose latest commit is sha.
func FindPullRequests(ctx context.Context, repo *Repository, owner, branch, sha string) ([]*github.PullRequest, error) {
	var pulls []*github.PullRequest
	if owner != "" {
		listed, _, err := repo.Client.PullRequests.List(ctx, repo.Owner, repo.Name, &github.PullRequestListOptions{
			State:       "open",
			Head:        owner + ":" + branch,
			ListOptions: github.ListOptions{PerPage: 100},
		})
		if err != nil {
			return nil, err
		}
		pulls = listed
	}
	if len(pulls) == 0 {
		listed, _, err := repo.Client.PullRequests.ListPullRequestsWithCommit(ctx, repo.Owner, repo.Name, sha, &github.ListOptions{PerPage: 100})
		if err != nil {
			return nil, err
		}
		for _, pull := range listed {
			if pull.GetState() == "open" {
				pulls = append(pulls, pull)
			}
		}
	}
	var found []*github.PullRequest
	for _, pull := range pulls {
		if pull.GetHead().GetSHA() == sha {
			found = append(found, pull)
		}
	}
	return found, nil
}

// workflowPath is the path of the workflow of an implementation.
func workflowPath(implementation string) string {
	return workflowsDir + "/" + implementation + ".yml"
}

// ActiveWorkflows returns the workflows of the implementations, by
// implementation; one that is gone or disabled is left out.
func ActiveWorkflows(ctx context.Context, repo *Repository, implementations []string) (map[string]*github.Workflow, error) {
	list, _, err := repo.Client.Actions.ListWorkflows(ctx, repo.Owner, repo.Name, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, err
	}
	workflows := map[string]*github.Workflow{}
	for _, workflow := range list.Workflows {
		if path := workflow.GetPath(); path != "" {
			workflows[path] = workflow
		}
	}
	active := map[string]*github.Workflow{}
	for _, implementation := range implementations {
		if workflow, ok := workflows[workflowPath(implementation)]; ok && workflow.GetState() == "active" {
			active[implementation] = workflow
		}
	}
	return active, nil
}

// LatestRuns returns the latest run of every workflow of the pull request for
// a commit, by the path of the workflow.
func LatestRuns(ctx context.Context, repo *Repository, headSHA string) (map[string]*github.WorkflowRun, error) {
	list, _, err := repo.Client.Actions.ListRepositoryWorkflowRuns(ctx, repo.Owner, repo.Name, &github.ListWorkflowRunsOptions{
		HeadSHA:     headSHA,
		Event:       "pull_request",
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}
	runs := map[string]*github.WorkflowRun{}
	for _, run := range list.WorkflowRuns {
		path := run.GetPath()
		if current, ok := runs[path]; !ok || run.GetID() > current.GetID() {
			runs[path] = run
		}
	}
	return runs, nil
}

// Unfinished returns the names of the workflows with a run that has not ended,
// other than the run that has just ended.
func Unfinished(implementations []string, workflows map[string]*github.Workflow, runs map[string]*github.WorkflowRun, finishedRun *int) []string {
	var unfinished []string
	for _, implementation := range implementations {
		workflow, ok := workflows[implementation]
		if !ok {
			continue
		}
		run, ok := runs[workflowPath(implementation)]
		if !ok {
			continue
		}
		if run.GetStatus() == "completed" {
			continue
		}
		if finishedRun != nil && run.GetID() == int64(*finishedRun) {
			continue
		}
		unfinished = append(unfinished, workflow.GetName())
	}
	return unfinished
}

// Collect returns the rows of the report: the numbers of the pull request and
// of the default branch.
func Collect(ctx context.Context, repo *Repository, implementations []string, workflows map[string]*github.Workflow, runs map[string]*github.WorkflowRun, technologies map[string]string, defaultBranch string, finishedRun *int) ([]Row, error) {
	var rows []Row
	for _, implementation := range implementations {
		workflow, ok := workflows[implementation]
		if !ok {
			continue
		}
		row := Row{Implementation: implementation, Name: workflow.GetName(), Technology: technologies[implementation]}
		run, ok := runs[workflowPath(implementation)]
		if !ok {
			row.State = "inherited"
		} else {
			row.RunURL = run.GetHTMLURL()
			if run.GetStatus() != "completed" && (finishedRun == nil || run.GetID() != int64(*finishedRun)) {
				row.State = "running"
			} else {
				artifacts, err := runArtifacts(ctx, repo, run.GetID(), "memory-"+implementation)
				if err != nil {
					return nil, err
				}
				if artifact := Newest(artifacts); artifact != nil {
					row.Memory = ReadArtifact(ctx, repo, artifact)
				}
				if row.Memory != nil {
					row.State = "measured"
				} else {
					row.State = "failed"
				}
			}
		}
		if row.State == "measured" || row.State == "inherited" {
			memory, artifact, err := BaselineOf(ctx, repo, implementation, defaultBranch)
			if err != nil {
				return nil, err
			}
			if memory != nil {
				row.Baseline = memory
				row.BaselineSHA = artifact.GetWorkflowRun().GetHeadSHA()
				row.BaselineURL = runURL(repo, artifact.GetWorkflowRun().GetID())
			} else if row.State == "inherited" {
				row.State = "missing"
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// SameText reports whether two comments say the same, ignoring line endings
// and blank space around them.
func SameText(left, right string) bool {
	normalize := func(text string) string {
		return strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	}
	return normalize(left) == normalize(right)
}

// UpsertComment writes the comment of the pull request, or replaces the one
// that an earlier run wrote. It returns created, updated or unchanged.
func UpsertComment(ctx context.Context, repo *Repository, number int, body string) (string, error) {
	var mine []*github.IssueComment
	options := &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for comment, err := range repo.Client.Issues.ListCommentsIter(ctx, repo.Owner, repo.Name, number, options) {
		if err != nil {
			return "", err
		}
		if comment.GetUser().GetLogin() == Bot && strings.HasPrefix(comment.GetBody(), Marker) {
			mine = append(mine, comment)
		}
	}
	sort.Slice(mine, func(a, b int) bool { return mine[a].GetID() < mine[b].GetID() })
	request := github.IssueCommentRequest{Body: body}
	if len(mine) == 0 {
		if _, _, err := repo.Client.Issues.CreateComment(ctx, repo.Owner, repo.Name, number, request); err != nil {
			return "", err
		}
		return "created", nil
	}
	result := "unchanged"
	if !SameText(mine[0].GetBody(), body) {
		if _, _, err := repo.Client.Issues.UpdateComment(ctx, repo.Owner, repo.Name, mine[0].GetID(), request); err != nil {
			return "", err
		}
		result = "updated"
	}
	// Two runs at the same time can both have created one.
	for _, extra := range mine[1:] {
		if _, err := repo.Client.Issues.DeleteComment(ctx, repo.Owner, repo.Name, extra.GetID()); err != nil {
			return "", err
		}
	}
	return result, nil
}

// Report writes the comment of the pull request. runID is the workflow run
// that has just ended; pullRequest reports on that pull request now, with
// exactly one of the two set. dryRun prints the comment instead of writing
// it. It returns 0 when there is nothing to report on, like the script.
func Report(ctx context.Context, repo *Repository, implementations []string, technologies map[string]string, runID *int, pullRequest int, dryRun bool) (int, error) {
	// Most runs end here, on the way to the one that finds that all of them have: keep it to a few requests.
	var headSHA string
	var pulls []*github.PullRequest
	if runID != nil {
		run, _, err := repo.Client.Actions.GetWorkflowRunByID(ctx, repo.Owner, repo.Name, int64(*runID))
		if err != nil {
			return 0, err
		}
		if run.GetEvent() != "pull_request" {
			logMessage(fmt.Sprintf("The run %d is a %s run, not one of a pull request.", *runID, run.GetEvent()))
			return 0, nil
		}
		headSHA = run.GetHeadSHA()
		pulls, err = FindPullRequests(ctx, repo, run.GetHeadRepository().GetOwner().GetLogin(), run.GetHeadBranch(), headSHA)
		if err != nil {
			return 0, err
		}
	} else {
		pull, _, err := repo.Client.PullRequests.Get(ctx, repo.Owner, repo.Name, pullRequest)
		if err != nil {
			return 0, err
		}
		if pull.GetState() != "open" {
			logMessage(fmt.Sprintf("The pull request %d is not open.", pullRequest))
			return 0, nil
		}
		headSHA = pull.GetHead().GetSHA()
		pulls = []*github.PullRequest{pull}
	}
	if len(pulls) == 0 {
		logMessage(fmt.Sprintf("No open pull request has %s as its latest commit; there is nothing to report on.", shortSHA(headSHA)))
		return 0, nil
	}
	workflows, err := ActiveWorkflows(ctx, repo, append(append([]string{}, implementations...), "android"))
	if err != nil {
		return 0, err
	}
	if len(workflows) == 0 {
		logMessage("None of the implementations has a workflow.")
		return 0, nil
	}
	runs, err := LatestRuns(ctx, repo, headSHA)
	if err != nil {
		return 0, err
	}
	if runID != nil {
		var ordered []string
		for _, implementation := range implementations {
			if _, ok := workflows[implementation]; ok {
				ordered = append(ordered, implementation)
			}
		}
		// The android workflow has no row of its own, but its run carries
		// the APK section, so the report waits for it like for the rest.
		if waiting := Unfinished(append(ordered, "android"), workflows, runs, runID); len(waiting) > 0 {
			logMessage(fmt.Sprintf("Waiting for %s; the report is written when they have ended.", strings.Join(waiting, ", ")))
			return 0, nil
		}
	}
	info, _, err := repo.Client.Repositories.Get(ctx, repo.Owner, repo.Name)
	if err != nil {
		return 0, err
	}
	defaultBranch := info.GetDefaultBranch()
	rows, err := Collect(ctx, repo, implementations, workflows, runs, technologies, defaultBranch, runID)
	if err != nil {
		return 0, err
	}
	extras, err := CollectExtras(ctx, repo, runs, defaultBranch)
	if err != nil {
		return 0, err
	}
	body := InsertExtras(Render(rows, headSHA), RenderExtras(extras))
	if summary := os.Getenv("GITHUB_STEP_SUMMARY"); summary != "" {
		file, err := os.OpenFile(summary, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return 0, err
		}
		_, err = io.WriteString(file, strings.TrimPrefix(body, Marker+"\n"))
		file.Close()
		if err != nil {
			return 0, err
		}
	}
	if dryRun {
		fmt.Fprintln(stdout, body)
		return 0, nil
	}
	for _, listed := range pulls {
		number := listed.GetNumber()
		current, _, err := repo.Client.PullRequests.Get(ctx, repo.Owner, repo.Name, number)
		if err != nil {
			return 0, err
		}
		if current.GetState() != "open" || current.GetHead().GetSHA() != headSHA {
			logMessage(fmt.Sprintf("The pull request %d has moved on from %s.", number, shortSHA(headSHA)))
			continue
		}
		result, err := UpsertComment(ctx, repo, number, body)
		if err != nil {
			return 0, err
		}
		logMessage(fmt.Sprintf("Pull request %d: comment %s.", number, result))
	}
	return 0, nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// reportFunc runs the report. It is a variable so that tests can stub the report.
var reportFunc = Report

// Run runs the tool. args are the command line without the program name.
func Run(args []string) error {
	flags := flag.NewFlagSet("memory-report", flag.ContinueOnError)
	repository := flags.String("repository", os.Getenv("GITHUB_REPOSITORY"), "OWNER/NAME")
	implementations := flags.String("implementations", os.Getenv("MEMORY_IMPLEMENTATIONS"), "the implementation, such as 'mygo', separated by spaces")
	docs := flags.String("docs", "docs/en/dev/memory.md", "the page with the Technology column")
	dryRun := flags.Bool("dry-run", false, "print the comment instead of writing it")
	run := flags.Int("run", 0, "the ID of the workflow run that has just ended")
	pullRequest := flags.Int("pull-request", 0, "the number of the pull request")
	if err := flags.Parse(args); err != nil {
		return err
	}
	names := strings.Fields(*implementations)
	valid := len(names) > 0
	for _, name := range names {
		if !implementationName.MatchString(name) {
			valid = false
			break
		}
	}
	if !repositoryName.MatchString(*repository) {
		return errors.New("--repository (or GITHUB_REPOSITORY) must be OWNER/NAME")
	}
	if !valid {
		return errors.New("--implementations (or MEMORY_IMPLEMENTATIONS) must list names such as 'mygo'")
	}
	if (*run == 0) == (*pullRequest == 0) {
		return errors.New("one of --run and --pull-request is required")
	}
	technologies := map[string]string{}
	if info, err := os.Stat(*docs); err == nil && !info.IsDir() {
		content, err := os.ReadFile(*docs)
		if err != nil {
			return err
		}
		technologies = ReadTechnologies(string(content))
	}
	var runID *int
	if *run != 0 {
		runID = run
	}
	repo, err := NewRepository(*repository, os.Getenv("GH_TOKEN"))
	if err != nil {
		return err
	}
	_, err = reportFunc(context.Background(), repo, names, technologies, runID, *pullRequest, *dryRun)
	if err != nil && strings.Contains(err.Error(), "Resource not accessible by integration") {
		// A token that cannot write, such as that of a run for a bot's pull request: the report is
		// already in the summary of the run, which is as far as it can go.
		encoded := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(err.Error(), "%", "%25"), "\r", "%0D"), "\n", "%0A")
		fmt.Fprintln(stdout, "::warning::The token of this run cannot write the comment; the report is in the summary: "+encoded)
		return nil
	}
	return err
}
