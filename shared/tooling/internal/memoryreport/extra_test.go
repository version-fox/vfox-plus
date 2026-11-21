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
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
)

func lightweightText(idle, loaded [3]int) string {
	return fmt.Sprintf("memory (lightweight-idle): RSS %d MB, PSS %d MB in %d processes\nmemory (lightweight-loaded): RSS %d MB, PSS %d MB in %d processes\n",
		idle[0], idle[1], idle[2], loaded[0], loaded[1], loaded[2])
}

func androidText(cold [3]int) string {
	return fmt.Sprintf("memory (apk-cold): RSS %d MB, PSS %d MB in %d processes\n", cold[0], cold[1], cold[2])
}

// privateText is the memory-lightweight-private artifact of a run: the private memory of the two points, in KB.
func privateText(idle, loaded int64) string {
	return fmt.Sprintf("memory (lightweight-idle): private %d KB in 1 processes\nmemory (lightweight-loaded): private %d KB in 1 processes\n",
		idle, loaded)
}

func TestParseExtras(t *testing.T) {
	lightweight := ParseLightweightMemory(lightweightText([3]int{12, 10, 1}, [3]int{13, 12, 1}))
	if lightweight == nil || lightweight.Idle.PSS != 10 || lightweight.Loaded.RSS != 13 {
		t.Fatalf("lightweight = %+v", lightweight)
	}
	for _, text := range []string{
		"",
		"memory (lightweight-idle): RSS 12 MB, PSS 10 MB in 1 processes\n",
		lightweightText([3]int{12, 10, 1}, [3]int{13, 12, 1}) + "memory (lightweight-idle): RSS 12 MB, PSS 10 MB in 1 processes\n",
		"memory (lightweight-idle): RSS 12 MB, PSS 10 MB in 1 processes\nmemory (home): RSS 13 MB, PSS 12 MB in 1 processes\n",
		"memory (lightweight-idle): RSS 12 MB, PSS 10 MB in 1 processes\nmemory (lightweight-loaded): RSS 13 MB, PSS 12 MB in 1 processes\nextra\n",
	} {
		if ParseLightweightMemory(text) != nil {
			t.Fatalf("ParseLightweightMemory(%q) parsed", text)
		}
	}
	apk := ParseAndroidMemory(androidText([3]int{210, 190, 1}))
	if apk == nil || apk.Cold.PSS != 190 {
		t.Fatalf("apk = %+v", apk)
	}
	for _, text := range []string{
		"",
		"memory (home): RSS 210 MB, PSS 190 MB in 1 processes\n",
		androidText([3]int{210, 190, 1}) + androidText([3]int{210, 190, 1}),
	} {
		if ParseAndroidMemory(text) != nil {
			t.Fatalf("ParseAndroidMemory(%q) parsed", text)
		}
	}
}

// extrasFixture builds the API with the mygo and phone workflows, one pull
// request, and baselines on main for the two extra artifacts.
func extrasFixture() *FakeAPI {
	f := newFakeAPI()
	f.addWorkflow("mygo", "mygo", "")
	f.addWorkflow("phone", "phone", "")
	f.addPullRequest(7, "", "")
	f.addArtifact("mygo", map[string]string{"memory.txt": lightweightText([3]int{11, 10, 1}, [3]int{12, 11, 1})},
		artifactOptions{name: LightweightArtifact})
	f.addArtifact("mygo", map[string]string{"memory.txt": privateText(14336, 14848)},
		artifactOptions{name: LightweightPrivateArtifact})
	f.addArtifact("phone", map[string]string{"memory.txt": androidText([3]int{200, 180, 1})},
		artifactOptions{name: AndroidArtifact})
	return f
}

func extrasRuns(t *testing.T, repo *Repository) map[string]*github.WorkflowRun {
	t.Helper()
	runs, err := LatestRuns(context.Background(), repo, testHead)
	if err != nil {
		t.Fatal(err)
	}
	return runs
}

func findExtra(extras []Extra, title string) *Extra {
	for i := range extras {
		if extras[i].Title == title {
			return &extras[i]
		}
	}
	return nil
}

func TestCollectExtras(t *testing.T) {
	t.Run("measured with baselines", func(t *testing.T) {
		f := extrasFixture()
		mygo := f.addRun("mygo", runOptions{})
		f.addArtifact("mygo", map[string]string{"memory.txt": lightweightText([3]int{12, 10, 1}, [3]int{20, 19, 1})},
			artifactOptions{runID: mygo, name: LightweightArtifact, branch: "feature", sha: testHead})
		f.addArtifact("mygo", map[string]string{"memory.txt": privateText(14848, 19456)},
			artifactOptions{runID: mygo, name: LightweightPrivateArtifact, branch: "feature", sha: testHead})
		android := f.addRun("phone", runOptions{})
		f.addArtifact("phone", map[string]string{"memory.txt": androidText([3]int{205, 182, 1})},
			artifactOptions{runID: android, name: AndroidArtifact, branch: "feature", sha: testHead})
		repo := f.repository(t)
		extras, err := CollectExtras(context.Background(), repo, extrasRuns(t, repo), "main")
		if err != nil {
			t.Fatal(err)
		}
		if len(extras) != 2 {
			t.Fatalf("extras = %d", len(extras))
		}
		lightweight := findExtra(extras, "Lightweight mode")
		if lightweight == nil || len(lightweight.Rows) != 6 {
			t.Fatalf("lightweight = %+v", lightweight)
		}
		// Resident memory is shown, not judged: +1 MB and +8 MB both carry no bold.
		if lightweight.Rows[0].PR != "12 MB (+1)" {
			t.Fatalf("idle RSS = %q", lightweight.Rows[0].PR)
		}
		if lightweight.Rows[3].PR != "20 MB (+8)" {
			t.Fatalf("loaded RSS = %q", lightweight.Rows[3].PR)
		}
		// Private memory is judged: +0.5 MB is within its noise, +4.5 MB is a change.
		if idle := lightweight.Rows[2]; idle.PR != "14.5 MB (+0.5)" || idle.Main != "14.0 MB" {
			t.Fatalf("idle private = %+v", idle)
		}
		if loaded := lightweight.Rows[5]; loaded.PR != "19.0 MB (**+4.5**)" || loaded.Main != "14.5 MB" {
			t.Fatalf("loaded private = %+v", loaded)
		}
		apk := findExtra(extras, "Phone app")
		if apk == nil || len(apk.Rows) != 2 {
			t.Fatalf("apk = %+v", apk)
		}
		// +5 MB is within the emulator noise: shown, not bold.
		if apk.Rows[0].PR != "205 MB (+5)" || apk.Rows[0].Main != "200 MB" {
			t.Fatalf("cold RSS = %+v", apk.Rows[0])
		}
		for _, extra := range extras {
			if !strings.Contains(extra.RunURL, "/actions/runs/") {
				t.Fatalf("%s has no run link", extra.Title)
			}
			if !strings.Contains(extra.BaselineURL, "/actions/runs/") {
				t.Fatalf("%s has no baseline link", extra.Title)
			}
		}
	})

	t.Run("running run is pending", func(t *testing.T) {
		f := extrasFixture()
		f.addRun("phone", runOptions{status: "in_progress", conclusion: ""})
		repo := f.repository(t)
		extras, err := CollectExtras(context.Background(), repo, extrasRuns(t, repo), "main")
		if err != nil {
			t.Fatal(err)
		}
		apk := findExtra(extras, "Phone app")
		if apk == nil || apk.Note != "Still running; the report is written again when it ends." {
			t.Fatalf("apk = %+v", apk)
		}
	})

	t.Run("missing numbers fail, missing everything is left out", func(t *testing.T) {
		f := newFakeAPI()
		f.addWorkflow("mygo", "mygo", "")
		f.addWorkflow("phone", "phone", "")
		f.addPullRequest(7, "", "")
		f.addRun("mygo", runOptions{})
		repo := f.repository(t)
		extras, err := CollectExtras(context.Background(), repo, extrasRuns(t, repo), "main")
		if err != nil {
			t.Fatal(err)
		}
		if len(extras) != 1 {
			t.Fatalf("extras = %+v", extras)
		}
		lightweight := findExtra(extras, "Lightweight mode")
		if lightweight == nil || lightweight.Note != "No numbers: the job did not upload them." {
			t.Fatalf("lightweight = %+v", lightweight)
		}
		// No phone run and no phone baseline: no section.
		if findExtra(extras, "Phone app") != nil {
			t.Fatal("want no Android section")
		}
	})

	t.Run("run without an artifact inherits the baseline", func(t *testing.T) {
		f := extrasFixture()
		f.addRun("mygo", runOptions{})
		repo := f.repository(t)
		extras, err := CollectExtras(context.Background(), repo, extrasRuns(t, repo), "main")
		if err != nil {
			t.Fatal(err)
		}
		lightweight := findExtra(extras, "Lightweight mode")
		if lightweight == nil || len(lightweight.Rows) != 6 || lightweight.Rows[0].PR != "11 MB (main)" {
			t.Fatalf("lightweight = %+v", lightweight)
		}
		if lightweight.Rows[2].PR != "14.0 MB (main)" {
			t.Fatalf("idle private = %+v", lightweight.Rows[2])
		}
		if lightweight.RunURL == "" || lightweight.BaselineURL == "" {
			t.Fatalf("lightweight has no links: %+v", lightweight)
		}
	})

	t.Run("no run inherits the baseline", func(t *testing.T) {
		f := extrasFixture()
		repo := f.repository(t)
		extras, err := CollectExtras(context.Background(), repo, extrasRuns(t, repo), "main")
		if err != nil {
			t.Fatal(err)
		}
		if len(extras) != 2 {
			t.Fatalf("extras = %d", len(extras))
		}
		lightweight := findExtra(extras, "Lightweight mode")
		if lightweight.Rows[0].PR != "11 MB (main)" {
			t.Fatalf("idle RSS = %q", lightweight.Rows[0].PR)
		}
	})

	t.Run("no baseline says so", func(t *testing.T) {
		f := newFakeAPI()
		f.addWorkflow("mygo", "mygo", "")
		f.addWorkflow("phone", "phone", "")
		f.addPullRequest(7, "", "")
		mygo := f.addRun("mygo", runOptions{})
		f.addArtifact("mygo", map[string]string{"memory.txt": lightweightText([3]int{12, 10, 1}, [3]int{13, 12, 1})},
			artifactOptions{runID: mygo, name: LightweightArtifact, branch: "feature", sha: testHead})
		repo := f.repository(t)
		extras, err := CollectExtras(context.Background(), repo, extrasRuns(t, repo), "main")
		if err != nil {
			t.Fatal(err)
		}
		lightweight := findExtra(extras, "Lightweight mode")
		if lightweight == nil || lightweight.Note != "No baseline: no run on main uploaded numbers yet." {
			t.Fatalf("lightweight = %+v", lightweight)
		}
	})
}

func TestRenderInsertExtras(t *testing.T) {
	extras := []Extra{{
		Title: "Lightweight mode",
		Intro: "How it is measured.",
		Rows:  []ExtraRow{{Measure: "Idle PSS", PR: "10 MB (+1)", Main: "9 MB"}},
	}}
	rendered := RenderExtras(extras)
	if !strings.Contains(rendered, "#### Lightweight mode") {
		t.Fatalf("rendered =\n%s", rendered)
	}
	if !strings.Contains(rendered, "| Idle PSS | 10 MB (+1) | 9 MB |") {
		t.Fatalf("rendered =\n%s", rendered)
	}
	body := Marker + "\ntable\n\n<details>\nrest\n"
	merged := InsertExtras(body, rendered)
	if !strings.Contains(merged, "#### Lightweight mode\n\n<details>") && !strings.Contains(merged, "#### Lightweight mode") {
		t.Fatalf("merged =\n%s", merged)
	}
	if strings.Index(merged, "#### Lightweight mode") > strings.Index(merged, "<details>") {
		t.Fatalf("extras are not before the details:\n%s", merged)
	}
	if InsertExtras(body, "") != body {
		t.Fatal("empty extras must leave the body alone")
	}
	if out := InsertExtras("no details here", rendered); !strings.HasSuffix(out, rendered) {
		t.Fatalf("out = %q", out)
	}
}

func TestReportWaitsForPhone(t *testing.T) {
	f := newFakeAPI()
	f.addWorkflow("mygo", "mygo", "")
	f.addWorkflow("phone", "phone", "")
	f.addPullRequest(7, "", "")
	f.addArtifact("mygo", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{})
	mygo := f.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
	f.addRun("phone", runOptions{status: "in_progress", conclusion: ""})
	var outBuffer, errBuffer bytes.Buffer
	oldOut, oldErr := stdout, stderr
	stdout, stderr = &outBuffer, &errBuffer
	defer func() { stdout, stderr = oldOut, oldErr }()
	if _, err := Report(context.Background(), f.repository(t), []string{"mygo"}, map[string]string{}, &mygo, 0, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errBuffer.String(), "Waiting for phone") {
		t.Fatalf("errors = %q", errBuffer.String())
	}
	if len(f.comments) != 0 {
		t.Fatal("want no comment")
	}
}

func TestParseLightweightPrivate(t *testing.T) {
	private := ParseLightweightPrivate(privateText(14336, 14848))
	if private == nil || private.Idle != 14336 || private.Loaded != 14848 {
		t.Fatalf("private = %+v", private)
	}
	for _, text := range []string{
		"",
		"memory (lightweight-idle): private 14336 KB in 1 processes\n",
		privateText(1, 2) + "memory (lightweight-idle): private 3 KB in 1 processes\n",
		"memory (lightweight-idle): private 14336 KB in 1 processes\nmemory (lightweight-loaded): RSS 13 MB, PSS 12 MB in 1 processes\n",
		"memory (lightweight-idle): private 14336 MB in 1 processes\nmemory (lightweight-loaded): private 14848 KB in 1 processes\n",
		privateText(1, 2) + "extra\n",
	} {
		if ParseLightweightPrivate(text) != nil {
			t.Fatalf("ParseLightweightPrivate(%q) parsed", text)
		}
	}
}

// TestTheMeasurementLinesAreTheArtifactsOfTheReport checks that the lines which vfxtool measure-lightweight
// writes are the ones that the report reads, in both artifacts: the usage of the processes as the measuring
// tool prints it, parsed back.
func TestTheMeasurementLinesAreTheArtifactsOfTheReport(t *testing.T) {
	idle := measure.Usage{RSS: 49 * 1024, PSS: 35 * 1024, Private: 14336, Processes: 1}
	loaded := measure.Usage{RSS: 51 * 1024, PSS: 37 * 1024, Private: 14848, Processes: 1}
	memory := "memory (lightweight-idle): " + idle.Summary() + "\nmemory (lightweight-loaded): " + loaded.Summary() + "\n"
	if parsed := ParseLightweightMemory(memory); parsed == nil || parsed.Idle.RSS != 49 || parsed.Loaded.PSS != 37 {
		t.Fatalf("the memory lines %q parse as %+v", memory, parsed)
	}
	private := "memory (lightweight-idle): " + idle.PrivateSummary() + "\nmemory (lightweight-loaded): " + loaded.PrivateSummary() + "\n"
	if parsed := ParseLightweightPrivate(private); parsed == nil || parsed.Idle != 14336 || parsed.Loaded != 14848 {
		t.Fatalf("the private lines %q parse as %+v", private, parsed)
	}
}

// TestPrivateRowsAreJudgedByTheirOwnNoise checks the private memory of the lightweight mode: the number in MB
// with one decimal, its difference from main, and bold only past the larger of 2 MB and 5 % of main.
func TestPrivateRowsAreJudgedByTheirOwnNoise(t *testing.T) {
	kb := func(v int64) *int64 { return &v }
	for _, c := range []struct {
		name      string
		pr, base  *int64
		inherited bool
		wantPR    string
		wantMain  string
	}{
		{"no change", kb(14336), kb(14336), false, "14.0 MB", "14.0 MB"},
		{"+0.5 MB is within the noise", kb(14848), kb(14336), false, "14.5 MB (+0.5)", "14.0 MB"},
		{"+4.5 MB is a change", kb(19456), kb(14848), false, "19.0 MB (**+4.5**)", "14.5 MB"},
		{"-3 MB is a change", kb(11264), kb(14336), false, "11.0 MB (**-3.0**)", "14.0 MB"},
		{"a difference that rounds to nothing", kb(14337), kb(14336), false, "14.0 MB (0.0)", "14.0 MB"},
		{"main has no private memory yet", kb(14336), nil, false, "14.0 MB", "-"},
		{"this pull request has none", nil, kb(14336), false, "-", "14.0 MB"},
		{"inherited from main", kb(14336), kb(14336), true, "14.0 MB (main)", "14.0 MB"},
	} {
		row := privateRow("Idle private memory", c.pr, c.base, c.inherited)
		if row.PR != c.wantPR || row.Main != c.wantMain {
			t.Errorf("%s: %q | %q, want %q | %q", c.name, row.PR, row.Main, c.wantPR, c.wantMain)
		}
	}
}

// TestPlainRowsAreShownNotJudged checks that the resident and the proportional memory show their difference
// from main without bold, and say so when the number comes from main.
func TestPlainRowsAreShownNotJudged(t *testing.T) {
	if row := plainRow("Idle RSS", 49, 42, false); row.PR != "49 MB (+7)" || row.Main != "42 MB" {
		t.Errorf("a large difference: %+v", row)
	}
	if row := plainRow("Idle PSS", 35, 35, false); row.PR != "35 MB" || row.Main != "35 MB" {
		t.Errorf("no difference: %+v", row)
	}
	if row := plainRow("Idle RSS", 11, 11, true); row.PR != "11 MB (main)" {
		t.Errorf("inherited: %+v", row)
	}
}

func TestRemarkIsUnderTheTable(t *testing.T) {
	rendered := RenderExtras([]Extra{{
		Title:  "Lightweight mode",
		Intro:  "How it is measured.",
		Rows:   []ExtraRow{{Measure: "Idle PSS", PR: "10 MB", Main: "9 MB"}},
		Remark: "Main has no private memory yet.",
	}})
	if !strings.Contains(rendered, "| Idle PSS | 10 MB | 9 MB |\n\n- Main has no private memory yet.") {
		t.Fatalf("rendered =\n%s", rendered)
	}
}
