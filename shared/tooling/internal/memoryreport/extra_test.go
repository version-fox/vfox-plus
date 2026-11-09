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
)

func lightweightText(idle, loaded [3]int) string {
	return fmt.Sprintf("memory (lightweight-idle): RSS %d MB, PSS %d MB in %d processes\nmemory (lightweight-loaded): RSS %d MB, PSS %d MB in %d processes\n",
		idle[0], idle[1], idle[2], loaded[0], loaded[1], loaded[2])
}

func androidText(cold [3]int) string {
	return fmt.Sprintf("memory (apk-cold): RSS %d MB, PSS %d MB in %d processes\n", cold[0], cold[1], cold[2])
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

// extrasFixture builds the API with the mygo and android workflows, one pull
// request, and baselines on main for the two extra artifacts.
func extrasFixture() *FakeAPI {
	f := newFakeAPI()
	f.addWorkflow("mygo", "mygo", "")
	f.addWorkflow("android", "android", "")
	f.addPullRequest(7, "", "")
	f.addArtifact("mygo", map[string]string{"memory.txt": lightweightText([3]int{11, 10, 1}, [3]int{12, 11, 1})},
		artifactOptions{name: LightweightArtifact})
	f.addArtifact("android", map[string]string{"memory.txt": androidText([3]int{200, 180, 1})},
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
		android := f.addRun("android", runOptions{})
		f.addArtifact("android", map[string]string{"memory.txt": androidText([3]int{205, 182, 1})},
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
		if lightweight == nil || len(lightweight.Rows) != 4 {
			t.Fatalf("lightweight = %+v", lightweight)
		}
		// +1 MB is the noise of CI: shown, not bold.
		if lightweight.Rows[0].PR != "12 MB (+1)" {
			t.Fatalf("idle RSS = %q", lightweight.Rows[0].PR)
		}
		// +8 MB is a change: bold.
		if lightweight.Rows[2].PR != "20 MB (**+8**)" {
			t.Fatalf("loaded RSS = %q", lightweight.Rows[2].PR)
		}
		apk := findExtra(extras, "Android client")
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
		f.addRun("android", runOptions{status: "in_progress", conclusion: ""})
		repo := f.repository(t)
		extras, err := CollectExtras(context.Background(), repo, extrasRuns(t, repo), "main")
		if err != nil {
			t.Fatal(err)
		}
		apk := findExtra(extras, "Android client")
		if apk == nil || apk.Note != "Still running; the report is written again when it ends." {
			t.Fatalf("apk = %+v", apk)
		}
	})

	t.Run("missing numbers fail, missing everything is left out", func(t *testing.T) {
		f := newFakeAPI()
		f.addWorkflow("mygo", "mygo", "")
		f.addWorkflow("android", "android", "")
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
		// No android run and no android baseline: no section.
		if findExtra(extras, "Android client") != nil {
			t.Fatal("want no android section")
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
		if lightweight == nil || len(lightweight.Rows) != 4 || lightweight.Rows[0].PR != "11 MB (main)" {
			t.Fatalf("lightweight = %+v", lightweight)
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
		f.addWorkflow("android", "android", "")
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

func TestReportWaitsForAndroid(t *testing.T) {
	f := newFakeAPI()
	f.addWorkflow("mygo", "mygo", "")
	f.addWorkflow("android", "android", "")
	f.addPullRequest(7, "", "")
	f.addArtifact("mygo", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{})
	mygo := f.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
	f.addRun("android", runOptions{status: "in_progress", conclusion: ""})
	var outBuffer, errBuffer bytes.Buffer
	oldOut, oldErr := stdout, stderr
	stdout, stderr = &outBuffer, &errBuffer
	defer func() { stdout, stderr = oldOut, oldErr }()
	if _, err := Report(context.Background(), f.repository(t), []string{"mygo"}, map[string]string{}, &mygo, 0, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errBuffer.String(), "Waiting for android") {
		t.Fatalf("errors = %q", errBuffer.String())
	}
	if len(f.comments) != 0 {
		t.Fatal("want no comment")
	}
}
