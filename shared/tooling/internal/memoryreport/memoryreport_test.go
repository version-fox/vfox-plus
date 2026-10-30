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
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v93/github"
)

const testRepository = "version-fox/vfox-plus"
const testHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testMain = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestMain(m *testing.M) {
	// CI sets these. The tests must not write to the summary of the job that
	// runs them, or take the repository and the implementations from it.
	os.Unsetenv("GITHUB_STEP_SUMMARY")
	os.Unsetenv("GITHUB_REPOSITORY")
	os.Unsetenv("MEMORY_IMPLEMENTATIONS")
	os.Unsetenv("GH_TOKEN")
	// A request that fails is sent again at once.
	retryWait = func(int) time.Duration { return 0 }
	os.Exit(m.Run())
}

func memoryText(home, versions [3]int) string {
	return fmt.Sprintf("memory (home): RSS %d MB, PSS %d MB in %d processes\nmemory (nodejs): RSS %d MB, PSS %d MB in %d processes\n",
		home[0], home[1], home[2], versions[0], versions[1], versions[2])
}

func defaultMemoryText() string {
	return memoryText([3]int{55, 42, 1}, [3]int{64, 49, 1})
}

func makeZip(files map[string]string) []byte {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, content := range files {
		writer, err := archive.Create(name)
		if err != nil {
			panic(err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			panic(err)
		}
	}
	if err := archive.Close(); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}

func testMemory(homePSS, versionsPSS, processes int) *Memory {
	return &Memory{
		Home:     Point{RSS: homePSS + 10, PSS: homePSS, Processes: processes},
		Versions: Point{RSS: versionsPSS + 10, PSS: versionsPSS, Processes: processes},
	}
}

// FakeAPI is the parts of the API of GitHub that the report reads and writes,
// as data. ServeHTTP serves them over HTTP, as GitHub does, and repository
// connects the client of the report to that server.
type FakeAPI struct {
	defaultBranch string
	workflows     []map[string]any
	runs          []map[string]any
	artifacts     []map[string]any
	zips          map[int][]byte
	pulls         map[int]map[string]any
	comments      map[int][]map[string]any
	writes        [][2]string
	// requests holds the method and the path of every request, in order.
	requests []string
	// downloads counts the zips that were fetched.
	downloads int
	ids       int
	// failRaw makes the download of every artifact fail.
	failRaw bool
	onGet   func(path string, params map[string]any)
	// failing refuses the requests of the API that it says yes to, by method and path.
	failing func(request string) bool
}

func newFakeAPI() *FakeAPI {
	return &FakeAPI{
		defaultBranch: "main",
		zips:          map[int][]byte{},
		pulls:         map[int]map[string]any{},
		comments:      map[int][]map[string]any{},
		ids:           1000,
	}
}

func (f *FakeAPI) nextID() int {
	f.ids++
	return f.ids
}

func (f *FakeAPI) addWorkflow(implementation, name, state string) {
	if name == "" {
		name = strings.ToUpper(implementation[:1]) + implementation[1:]
	}
	if state == "" {
		state = "active"
	}
	f.workflows = append(f.workflows, map[string]any{
		"path":  ".github/workflows/" + implementation + ".yml",
		"name":  name,
		"state": state,
	})
}

func (f *FakeAPI) addPullRequest(number int, sha, state string) {
	if number == 0 {
		number = 7
	}
	if sha == "" {
		sha = testHead
	}
	if state == "" {
		state = "open"
	}
	f.pulls[number] = map[string]any{
		"number": number,
		"state":  state,
		"head":   map[string]any{"sha": sha, "ref": "feature", "owner": "contributor"},
	}
}

type runOptions struct {
	status        string
	conclusion    string
	sha           string
	event         string
	runID         int
	text          string
	hasText       bool
	artifactFiles map[string]string
}

func (f *FakeAPI) addRun(implementation string, options runOptions) int {
	status := options.status
	if status == "" {
		status = "completed"
	}
	conclusion := "success"
	if options.conclusion != "" || status != "completed" {
		conclusion = options.conclusion
	}
	sha := options.sha
	if sha == "" {
		sha = testHead
	}
	event := options.event
	if event == "" {
		event = "pull_request"
	}
	runID := options.runID
	if runID == 0 {
		runID = f.nextID()
	}
	f.runs = append(f.runs, map[string]any{
		"id":              runID,
		"path":            ".github/workflows/" + implementation + ".yml",
		"name":            strings.ToUpper(implementation[:1]) + implementation[1:],
		"event":           event,
		"status":          status,
		"conclusion":      conclusion,
		"head_sha":        sha,
		"head_branch":     "feature",
		"html_url":        fmt.Sprintf("https://github.com/%s/actions/runs/%d", testRepository, runID),
		"head_repository": map[string]any{"owner": map[string]any{"login": "contributor"}},
	})
	if options.artifactFiles != nil || options.hasText {
		files := options.artifactFiles
		if files == nil {
			files = map[string]string{"memory.txt": options.text}
		}
		f.addArtifact(implementation, files, artifactOptions{runID: runID, branch: "feature", sha: sha})
	}
	return runID
}

type artifactOptions struct {
	runID      int
	name       string
	branch     string
	sha        string
	created    string
	expired    bool
	headRepoID int
	size       int
	hasSize    bool
	useRawZip  bool
	rawZip     []byte
	noWorkflow bool
}

func (f *FakeAPI) addArtifact(implementation string, files map[string]string, options artifactOptions) int {
	if options.branch == "" {
		options.branch = "main"
	}
	if options.sha == "" {
		options.sha = testMain
	}
	if options.created == "" {
		options.created = "2026-10-20T10:00:00Z"
	}
	headRepoID := options.headRepoID
	if headRepoID == 0 && !options.noWorkflow {
		headRepoID = 1
	}
	archive := makeZip(files)
	artifactID := f.nextID()
	if options.useRawZip {
		f.zips[artifactID] = options.rawZip
	} else {
		f.zips[artifactID] = archive
	}
	size := len(archive)
	if options.hasSize {
		size = options.size
	}
	runID := options.runID
	if runID == 0 {
		runID = f.nextID()
	}
	var workflowRun any
	if !options.noWorkflow {
		workflowRun = map[string]any{
			"id":                 runID,
			"head_branch":        options.branch,
			"head_sha":           options.sha,
			"head_repository_id": headRepoID,
			"repository_id":      1,
		}
	}
	f.artifacts = append(f.artifacts, map[string]any{
		"id":            artifactID,
		"name":          firstNonEmpty(options.name, "memory-"+implementation),
		"size_in_bytes": size,
		"expired":       options.expired,
		"created_at":    options.created,
		"workflow_run":  workflowRun,
	})
	return artifactID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// The API as the client of the report sees it.

var (
	artifactZipPath = regexp.MustCompile(`^actions/artifacts/(\d+)/zip$`)
	commentsPath    = regexp.MustCompile(`^issues/(\d+)/comments$`)
)

// repository serves the fake and returns the repository whose client reads it.
func (f *FakeAPI) repository(t *testing.T) *Repository {
	t.Helper()
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	return repositoryAt(t, server.URL)
}

// typed converts a value of the fake to what the client returns for it.
func typed[T any](t *testing.T, value any) *T {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	result := new(T)
	if err := json.Unmarshal(data, result); err != nil {
		t.Fatal(err)
	}
	return result
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// ServeHTTP answers the requests of the client of the report.
func (f *FakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	if id, ok := strings.CutPrefix(r.URL.Path, "/download/"); ok {
		number, _ := strconv.Atoi(id)
		data, ok := f.zips[number]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f.downloads++
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(data)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, "/repos/"+testRepository)
	if !ok || (rest != "" && !strings.HasPrefix(rest, "/")) {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not Found"})
		return
	}
	if f.failing != nil && f.failing(r.Method+" "+rest) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": "refused by the test"})
		return
	}
	path := strings.TrimPrefix(rest, "/")
	if r.Method != http.MethodGet {
		f.serveWrite(w, r, path)
		return
	}
	if match := artifactZipPath.FindStringSubmatch(path); match != nil {
		if f.failRaw {
			writeJSON(w, http.StatusGone, map[string]any{"message": "Artifact has expired"})
			return
		}
		// The file is somewhere else, as it is at GitHub.
		w.Header().Set("Location", "http://"+r.Host+"/download/"+match[1])
		w.WriteHeader(http.StatusFound)
		return
	}
	if match := commentsPath.FindStringSubmatch(path); match != nil {
		f.serveComments(w, r, match[1])
		return
	}
	params := map[string]any{}
	for name, values := range r.URL.Query() {
		params[name] = values[0]
	}
	value, err := f.Get(path, params)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

// serveComments gives one page of the comments of an issue, with the link to
// the next that GitHub sends.
func (f *FakeAPI) serveComments(w http.ResponseWriter, r *http.Request, issue string) {
	number, _ := strconv.Atoi(issue)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	page = max(page, 1)
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}
	comments := f.comments[number]
	start := min((page-1)*perPage, len(comments))
	end := min(start+perPage, len(comments))
	if end < len(comments) {
		w.Header().Set("Link", fmt.Sprintf(`<http://%s%s?page=%d&per_page=%d>; rel="next"`, r.Host, r.URL.Path, page+1, perPage))
	}
	items := make([]any, 0, end-start)
	for _, comment := range comments[start:end] {
		items = append(items, comment)
	}
	writeJSON(w, http.StatusOK, items)
}

// serveWrite creates, updates and deletes comments.
func (f *FakeAPI) serveWrite(w http.ResponseWriter, r *http.Request, path string) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": err.Error()})
		return
	}
	result, err := f.Send(r.Method, path, payload)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": err.Error()})
		return
	}
	switch r.Method {
	case http.MethodPost:
		writeJSON(w, http.StatusCreated, result)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		writeJSON(w, http.StatusOK, result)
	}
}

// Get reads what a GET of the API of the repository answers with.
func (f *FakeAPI) Get(path string, params map[string]any) (any, error) {
	if f.onGet != nil {
		f.onGet(path, params)
	}
	switch {
	case path == "":
		return map[string]any{"default_branch": f.defaultBranch}, nil
	case path == "actions/workflows":
		return map[string]any{"workflows": toAnySlice(f.workflows)}, nil
	case path == "actions/runs":
		var runs []any
		for _, run := range f.runs {
			if asString(run["head_sha"]) == asString(params["head_sha"]) && asString(run["event"]) == asString(params["event"]) {
				runs = append(runs, run)
			}
		}
		if runs == nil {
			runs = []any{}
		}
		return map[string]any{"workflow_runs": runs}, nil
	case strings.HasPrefix(path, "actions/runs/") && strings.HasSuffix(path, "/artifacts"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "actions/runs/"), "/artifacts")
		// The artifacts of a run have no filter by name, as at GitHub.
		artifacts := []any{}
		for _, artifact := range f.artifacts {
			run, _ := artifact["workflow_run"].(map[string]any)
			if run != nil && fmt.Sprintf("%v", asInt(run["id"])) == id {
				artifacts = append(artifacts, artifact)
			}
		}
		return map[string]any{"artifacts": artifacts}, nil
	case strings.HasPrefix(path, "actions/runs/"):
		id := strings.TrimPrefix(path, "actions/runs/")
		for _, run := range f.runs {
			if fmt.Sprintf("%v", asInt(run["id"])) == id {
				return run, nil
			}
		}
		return nil, fmt.Errorf("no run %s", id)
	case path == "actions/artifacts":
		var artifacts []any
		for _, artifact := range f.artifacts {
			if asString(artifact["name"]) == asString(params["name"]) {
				artifacts = append(artifacts, artifact)
			}
		}
		if artifacts == nil {
			artifacts = []any{}
		}
		return map[string]any{"artifacts": artifacts}, nil
	case path == "pulls":
		var pulls []any
		head, _ := params["head"].(string)
		owner, branch, _ := strings.Cut(head, ":")
		for _, pull := range f.pulls {
			headMap, _ := pull["head"].(map[string]any)
			if asString(pull["state"]) == asString(params["state"]) &&
				asString(headMap["owner"]) == owner && asString(headMap["ref"]) == branch {
				pulls = append(pulls, pull)
			}
		}
		if pulls == nil {
			pulls = []any{}
		}
		return pulls, nil
	case strings.HasPrefix(path, "pulls/"):
		var number int
		fmt.Sscanf(strings.TrimPrefix(path, "pulls/"), "%d", &number)
		if pull, ok := f.pulls[number]; ok {
			return pull, nil
		}
		return nil, fmt.Errorf("no pull %d", number)
	case strings.HasPrefix(path, "commits/") && strings.HasSuffix(path, "/pulls"):
		sha := strings.TrimSuffix(strings.TrimPrefix(path, "commits/"), "/pulls")
		var pulls []any
		for _, pull := range f.pulls {
			headMap, _ := pull["head"].(map[string]any)
			if asString(headMap["sha"]) == sha {
				pulls = append(pulls, pull)
			}
		}
		if pulls == nil {
			pulls = []any{}
		}
		return pulls, nil
	}
	return nil, fmt.Errorf("unexpected GET %s %v", path, params)
}

func toAnySlice(maps []map[string]any) []any {
	items := make([]any, len(maps))
	for i, m := range maps {
		items[i] = m
	}
	return items
}

// Send writes through POST, PATCH or DELETE.
func (f *FakeAPI) Send(method, path string, payload map[string]any) (any, error) {
	f.writes = append(f.writes, [2]string{method, path})
	if match := commentsPath.FindStringSubmatch(path); match != nil && method == http.MethodPost {
		var number int
		fmt.Sscanf(match[1], "%d", &number)
		comment := map[string]any{"id": f.nextID(), "body": payload["body"], "user": map[string]any{"login": Bot}}
		f.comments[number] = append(f.comments[number], comment)
		return comment, nil
	}
	if match := regexp.MustCompile(`^issues/comments/(\d+)$`).FindStringSubmatch(path); match != nil {
		var id int
		fmt.Sscanf(match[1], "%d", &id)
		for number, comments := range f.comments {
			for i, comment := range comments {
				if asInt(comment["id"]) == int64(id) {
					if method == http.MethodPatch {
						comment["body"] = payload["body"]
						return comment, nil
					}
					f.comments[number] = append(comments[:i], comments[i+1:]...)
					return nil, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("unexpected %s %s", method, path)
}

// asInt reads an ID or a count out of the data of the fake.
func asInt(value any) int64 {
	switch number := value.(type) {
	case int:
		return int64(number)
	case int64:
		return number
	case float64:
		return int64(number)
	}
	return 0
}

// asString reads text out of the data of the fake.
func asString(value any) string {
	text, _ := value.(string)
	return text
}

func TestParseMemory(t *testing.T) {
	t.Run("reads both points", func(t *testing.T) {
		got := ParseMemory(memoryText([3]int{55, 42, 1}, [3]int{64, 49, 3}))
		want := &Memory{Home: Point{55, 42, 1}, Versions: Point{64, 49, 3}}
		if got == nil || *got != *want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})
	t.Run("order and blank lines do not matter", func(t *testing.T) {
		lines := strings.Split(defaultMemoryText(), "\n")
		text := "\n" + lines[1] + "\n\n" + lines[0] + "\n"
		if ParseMemory(text) == nil {
			t.Fatal("want a memory")
		}
	})
	t.Run("needs both points", func(t *testing.T) {
		if ParseMemory(strings.Split(defaultMemoryText(), "\n")[0]) != nil {
			t.Fatal("want nil")
		}
		if ParseMemory("") != nil {
			t.Fatal("want nil")
		}
	})
	t.Run("rejects a point twice", func(t *testing.T) {
		line := strings.Split(defaultMemoryText(), "\n")[0]
		if ParseMemory(line+"\n"+line+"\n"+strings.Split(defaultMemoryText(), "\n")[1]) != nil {
			t.Fatal("want nil")
		}
	})
	t.Run("rejects anything else", func(t *testing.T) {
		if ParseMemory(defaultMemoryText()+"e2e: scenario measure passed in 8 s\n") != nil {
			t.Fatal("want nil")
		}
		if ParseMemory(strings.Replace(defaultMemoryText(), "MB, PSS", "MB, PSS -", 1)) != nil {
			t.Fatal("want nil")
		}
		if ParseMemory(strings.Replace(defaultMemoryText(), "home", "other", 1)) != nil {
			t.Fatal("want nil")
		}
		big := strings.Replace(defaultMemoryText(), "55", "1000000", 1)
		if ParseMemory(big) != nil {
			t.Fatal("want nil")
		}
	})
	t.Run("rejects non-ascii digits", func(t *testing.T) {
		if ParseMemory(strings.Replace(defaultMemoryText(), "55", "٥٥", 1)) != nil {
			t.Fatal("want nil")
		}
	})
}

func TestTolerance(t *testing.T) {
	t.Run("small implementation judged by absolute floor", func(t *testing.T) {
		if ExceedsNoise("mygo", 42, 47) {
			t.Error("47 must be noise")
		}
		if !ExceedsNoise("mygo", 42, 48) {
			t.Error("48 must be a change")
		}
		if !ExceedsNoise("mygo", 42, 36) {
			t.Error("36 must be a change")
		}
		if ExceedsNoise("mygo", 42, 37) {
			t.Error("37 must be noise")
		}
	})
	t.Run("large measurement judged by percentage", func(t *testing.T) {
		if ExceedsNoise("mygo", 300, 315) {
			t.Error("315 must be noise")
		}
		if !ExceedsNoise("mygo", 300, 316) {
			t.Error("316 must be a change")
		}
	})
	t.Run("no difference is no change", func(t *testing.T) {
		for _, implementation := range []string{"mygo", "unknown"} {
			if ExceedsNoise(implementation, 100, 100) {
				t.Errorf("%s: 100 must be no change", implementation)
			}
		}
	})
}

func TestReadTechnologies(t *testing.T) {
	page := "# Memory use\n\nText with a | pipe.\n\n" +
		"| Implementation                   | Technology         | Processes |\n" +
		"| -------------------------------- | ------------------ | --------: |\n" +
		"| [mygo](../mygo/README.md#memory) | mygo 0.5, Go, GTK  |         1 |\n" +
		"| [Uno](../uno/README.md#memory)   | Uno Platform 6     |         1 |\n" +
		"\n" +
		"| Implementation | Other |\n" +
		"| --- | --- |\n" +
		"| [later](../later/README.md) | not read |\n"
	t.Run("reads the first table by directory", func(t *testing.T) {
		got := ReadTechnologies(page)
		if len(got) != 2 || got["mygo"] != "mygo 0.5, Go, GTK" || got["uno"] != "Uno Platform 6" {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("page without the table gives nothing", func(t *testing.T) {
		if got := ReadTechnologies("# Nothing\n\n| a | b |\n| - | - |\n"); len(got) != 0 {
			t.Fatalf("got %v", got)
		}
	})
	t.Run("reads the page of this repository", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "en", "dev", "memory.md"))
		if err != nil {
			t.Fatal(err)
		}
		technologies := ReadTechnologies(string(content))
		if len(technologies) == 0 {
			t.Fatal("want technologies")
		}
		for _, technology := range technologies {
			if technology == "" {
				t.Fatal("want no empty technology")
			}
		}
	})
}

func renderRows() []Row {
	return []Row{
		{Implementation: "tauri", Name: "Tauri", Technology: "Tauri 2, Rust", State: "measured", Memory: testMemory(283, 305, 3), Baseline: testMemory(283, 305, 3)},
		{Implementation: "mygo", Name: "mygo", Technology: "mygo 0.5, Go", State: "inherited", Baseline: testMemory(42, 49, 1)},
		{Implementation: "uno", Name: "Uno", Technology: "", State: "measured", Memory: testMemory(101, 128, 1), Baseline: testMemory(101, 127, 1)},
	}
}

func renderTable(text string) [][]string {
	var table [][]string
	for _, textLine := range strings.Split(text, "\n") {
		if strings.HasPrefix(textLine, "|") {
			var cells []string
			for _, cell := range strings.Split(strings.Trim(textLine, "|"), "|") {
				cells = append(cells, strings.TrimSpace(cell))
			}
			table = append(table, cells)
		}
	}
	return table
}

func TestRender(t *testing.T) {
	t.Run("starts with the marker that finds it again", func(t *testing.T) {
		text := Render(renderRows(), testHead)
		lines := strings.Split(text, "\n")
		if lines[0] != Marker {
			t.Fatalf("first line = %q", lines[0])
		}
		if !strings.Contains(lines[1], "`aaaaaaa`") {
			t.Fatalf("second line = %q", lines[1])
		}
	})
	t.Run("sorted by home PSS", func(t *testing.T) {
		table := renderTable(Render(renderRows(), testHead))
		var names []string
		for _, row := range table[2:] {
			names = append(names, row[0])
		}
		if strings.Join(names, ",") != "mygo (main),Uno,Tauri" {
			t.Fatalf("names = %v", names)
		}
	})
	t.Run("difference from main for what was measured only", func(t *testing.T) {
		table := renderTable(Render(renderRows(), testHead))
		byName := map[string][]string{}
		for _, row := range table[2:] {
			byName[row[0]] = row
		}
		if byName["Tauri"][3] != "283 (0)" {
			t.Fatalf("Tauri home = %q", byName["Tauri"][3])
		}
		if byName["Uno"][5] != "128 (+1)" {
			t.Fatalf("Uno versions = %q", byName["Uno"][5])
		}
		if byName["mygo (main)"][3] != "42" {
			t.Fatalf("mygo home = %q", byName["mygo (main)"][3])
		}
	})
	t.Run("the list adds what versions use beyond home", func(t *testing.T) {
		table := renderTable(Render(renderRows(), testHead))
		got := map[string]string{}
		for _, row := range table[2:] {
			got[row[0]] = row[len(row)-1]
		}
		if got["mygo (main)"] != "+7" || got["Uno"] != "+27" || got["Tauri"] != "+22" {
			t.Fatalf("adds = %v", got)
		}
	})
	t.Run("change beyond noise is bold and in summary", func(t *testing.T) {
		rows := []Row{{Implementation: "tauri", Name: "Tauri", State: "measured", Memory: testMemory(330, 305, 1), Baseline: testMemory(283, 305, 1)}}
		text := Render(rows, testHead)
		if !strings.Contains(text, "330 (**+47**)") {
			t.Fatalf("text =\n%s", text)
		}
		if !strings.Contains(text, "- Changed from main beyond the noise of CI: Tauri: Home PSS +47 MB (+17%).") {
			t.Fatalf("text =\n%s", text)
		}
		if strings.Contains(text, "Within the noise") {
			t.Fatalf("text =\n%s", text)
		}
	})
	t.Run("decrease counts too", func(t *testing.T) {
		rows := []Row{{Implementation: "tauri", Name: "Tauri", State: "measured", Memory: testMemory(200, 305, 1), Baseline: testMemory(283, 305, 1)}}
		if !strings.Contains(Render(rows, testHead), "200 (**-83**)") {
			t.Fatal("want a bold decrease")
		}
	})
	t.Run("processes change is a change", func(t *testing.T) {
		rows := []Row{{Implementation: "tauri", Name: "Tauri", State: "measured", Memory: testMemory(283, 305, 4), Baseline: testMemory(283, 305, 3)}}
		text := Render(rows, testHead)
		if !strings.Contains(text, "4 (**+1**)") {
			t.Fatalf("text =\n%s", text)
		}
		if !strings.Contains(text, "Tauri: processes 3 -> 4") {
			t.Fatalf("text =\n%s", text)
		}
	})
	t.Run("unchanged listed apart from changed", func(t *testing.T) {
		rows := []Row{
			{Implementation: "tauri", Name: "Tauri", State: "measured", Memory: testMemory(330, 305, 1), Baseline: testMemory(283, 305, 1)},
			{Implementation: "uno", Name: "Uno", State: "measured", Memory: testMemory(101, 127, 1), Baseline: testMemory(101, 127, 1)},
		}
		if !strings.Contains(Render(rows, testHead), "- Within the noise of CI: Uno.") {
			t.Fatal("want the within-noise line")
		}
	})
	t.Run("nothing changed", func(t *testing.T) {
		rows := []Row{{Implementation: "uno", Name: "Uno", State: "measured", Memory: testMemory(101, 127, 1), Baseline: testMemory(101, 127, 1)}}
		if !strings.Contains(Render(rows, testHead), "- No change from main beyond the noise of CI in Uno.") {
			t.Fatal("want the no-change line")
		}
	})
	t.Run("no baseline", func(t *testing.T) {
		rows := []Row{{Implementation: "uno", Name: "Uno", State: "measured", Memory: testMemory(101, 127, 1)}}
		text := Render(rows, testHead)
		if !strings.Contains(text, "Uno (no baseline)") {
			t.Fatalf("text =\n%s", text)
		}
		if !strings.Contains(text, "- No numbers on main to compare with yet: Uno.") {
			t.Fatalf("text =\n%s", text)
		}
		if !strings.Contains(text, "| 101 |") {
			t.Fatalf("text =\n%s", text)
		}
	})
	t.Run("failed and running come last with a link", func(t *testing.T) {
		rows := append(renderRows(),
			Row{Implementation: "flet", Name: "Flet", State: "failed", RunURL: "https://example.test/failed"},
			Row{Implementation: "gio", Name: "Gio", State: "running", RunURL: "https://example.test/running"},
			Row{Implementation: "wails", Name: "Wails", State: "missing"},
		)
		text := Render(rows, testHead)
		var names []string
		for _, row := range renderTable(text)[2:] {
			names = append(names, row[0])
		}
		if strings.Join(names, ",") != "mygo (main),Uno,Tauri,Gio,Flet,Wails" {
			t.Fatalf("names = %v", names)
		}
		for _, want := range []string{
			"[no result](https://example.test/failed)",
			"[running](https://example.test/running)",
			"- Still running: Gio.",
			"- No result, the run failed or was cancelled: Flet.",
			"- No numbers anywhere: Wails.",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("want %q in\n%s", want, text)
			}
		}
	})
	t.Run("rebuilt nothing", func(t *testing.T) {
		rows := []Row{{Implementation: "mygo", Name: "mygo", State: "inherited", Baseline: testMemory(42, 49, 1)}}
		if !strings.Contains(Render(rows, testHead), "did not rebuild any implementation") {
			t.Fatal("want the line")
		}
		if strings.Contains(Render(renderRows(), testHead), "did not rebuild any implementation") {
			t.Fatal("want no such line")
		}
	})
	t.Run("drops technology column without technologies", func(t *testing.T) {
		rows := []Row{{Implementation: "uno", Name: "Uno", State: "measured", Memory: testMemory(101, 127, 1), Baseline: testMemory(101, 127, 1)}}
		table := renderTable(Render(rows, testHead))
		if strings.Join(table[0][:3], ",") != "Implementation,Processes,Home PSS" {
			t.Fatalf("header = %v", table[0])
		}
		for _, row := range table {
			if len(row) != 7 {
				t.Fatalf("row = %v", row)
			}
		}
	})
	t.Run("keeps technology column with technologies", func(t *testing.T) {
		table := renderTable(Render(renderRows(), testHead))
		if strings.Join(table[0][:2], ",") != "Implementation,Technology" {
			t.Fatalf("header = %v", table[0])
		}
		for _, row := range table {
			if len(row) != 8 {
				t.Fatalf("row = %v", row)
			}
		}
	})
	t.Run("cell cannot break the table", func(t *testing.T) {
		rows := []Row{{Implementation: "uno", Name: "Un|o\n", Technology: "a | b\nc", State: "measured", Memory: testMemory(101, 127, 1), Baseline: testMemory(101, 127, 1)}}
		table := renderTable(strings.ReplaceAll(Render(rows, testHead), "\\|", "/"))
		for _, row := range table {
			if len(row) != 8 {
				t.Fatalf("row = %v", row)
			}
		}
	})
	t.Run("links the runs", func(t *testing.T) {
		rows := []Row{{
			Implementation: "uno", Name: "Uno", State: "measured",
			Memory: testMemory(101, 127, 1), Baseline: testMemory(101, 127, 1),
			RunURL: "https://example.test/pr", BaselineURL: "https://example.test/main", BaselineSHA: testMain,
		}}
		text := Render(rows, testHead)
		if !strings.Contains(text, "[Uno](https://example.test/pr)") {
			t.Fatalf("text =\n%s", text)
		}
		if !strings.Contains(text, "[Uno](https://example.test/main) (`bbbbbbb`)") {
			t.Fatalf("text =\n%s", text)
		}
	})
	t.Run("explains the noise", func(t *testing.T) {
		rows := []Row{{Implementation: "mygo", Name: "mygo", State: "measured", Memory: testMemory(300, 310, 1), Baseline: testMemory(300, 310, 1)}}
		if !strings.Contains(Render(rows, testHead), "more than 5 MB and more than 5% of the number on main.") {
			t.Fatal("want the explanation")
		}
	})
	t.Run("is ascii", func(t *testing.T) {
		for _, c := range Render(renderRows(), testHead) {
			if c > 127 {
				t.Fatalf("non-ascii %q", c)
			}
		}
	})
}

func TestReadArtifact(t *testing.T) {
	ctx := context.Background()
	read := func(t *testing.T, files map[string]string, options artifactOptions) *Memory {
		t.Helper()
		fake := newFakeAPI()
		fake.addArtifact("uno", files, options)
		return ReadArtifact(ctx, fake.repository(t), typed[github.Artifact](t, fake.artifacts[len(fake.artifacts)-1]))
	}
	t.Run("reads memory.txt", func(t *testing.T) {
		got := read(t, map[string]string{"memory.txt": memoryText([3]int{55, 42, 1}, [3]int{64, 49, 1})}, artifactOptions{})
		want := &Memory{Home: Point{55, 42, 1}, Versions: Point{64, 49, 1}}
		if got == nil || *got != *want {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("rejects other files", func(t *testing.T) {
		if read(t, map[string]string{"other.txt": defaultMemoryText()}, artifactOptions{}) != nil {
			t.Fatal("want nil")
		}
		if read(t, map[string]string{"memory.txt": defaultMemoryText(), "extra.txt": "x"}, artifactOptions{}) != nil {
			t.Fatal("want nil")
		}
		if read(t, map[string]string{"../memory.txt": defaultMemoryText()}, artifactOptions{}) != nil {
			t.Fatal("want nil")
		}
	})
	t.Run("rejects expired or large artifact without downloading", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{expired: true})
		fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{size: MaxArtifactBytes + 1, hasSize: true})
		repo := fake.repository(t)
		for _, artifact := range fake.artifacts {
			if ReadArtifact(ctx, repo, typed[github.Artifact](t, artifact)) != nil {
				t.Fatal("want nil")
			}
		}
		if len(fake.requests) != 0 {
			t.Fatalf("requests = %v", fake.requests)
		}
	})
	t.Run("rejects a large file inside the zip", func(t *testing.T) {
		if read(t, map[string]string{"memory.txt": defaultMemoryText() + strings.Repeat("x", MaxMemoryFileBytes+1)}, artifactOptions{}) != nil {
			t.Fatal("want nil")
		}
	})
	t.Run("rejects what is not a zip", func(t *testing.T) {
		fake := newFakeAPI()
		id := fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{})
		fake.zips[id] = []byte("not a zip")
		var buffer bytes.Buffer
		old := stderr
		stderr = &buffer
		defer func() { stderr = old }()
		if ReadArtifact(ctx, fake.repository(t), typed[github.Artifact](t, fake.artifacts[len(fake.artifacts)-1])) != nil {
			t.Fatal("want nil")
		}
		if !strings.Contains(buffer.String(), fmt.Sprintf("Cannot read the artifact memory-uno (%d)", id)) {
			t.Fatalf("errors = %q", buffer.String())
		}
	})
	t.Run("says when the download fails", func(t *testing.T) {
		fake := newFakeAPI()
		id := fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{})
		fake.failRaw = true
		var buffer bytes.Buffer
		old := stderr
		stderr = &buffer
		defer func() { stderr = old }()
		if ReadArtifact(ctx, fake.repository(t), typed[github.Artifact](t, fake.artifacts[len(fake.artifacts)-1])) != nil {
			t.Fatal("want nil")
		}
		if !strings.Contains(buffer.String(), fmt.Sprintf("Cannot read the artifact memory-uno (%d)", id)) || !strings.Contains(buffer.String(), "410 Gone") {
			t.Fatalf("errors = %q", buffer.String())
		}
	})
	t.Run("rejects wrong numbers", func(t *testing.T) {
		if read(t, map[string]string{"memory.txt": "memory (home): RSS many MB, PSS 1 MB in 1 processes\n"}, artifactOptions{}) != nil {
			t.Fatal("want nil")
		}
	})
}

func TestBaseline(t *testing.T) {
	t.Run("takes the newest valid artifact", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addArtifact("uno", map[string]string{"memory.txt": memoryText([3]int{10, 9, 1}, [3]int{10, 9, 1})},
			artifactOptions{created: "2026-10-18T00:00:00Z", sha: strings.Repeat("1", 40)})
		fake.addArtifact("uno", map[string]string{"memory.txt": memoryText([3]int{20, 19, 1}, [3]int{20, 19, 1})},
			artifactOptions{created: "2026-10-20T00:00:00Z", sha: strings.Repeat("2", 40)})
		baseline, artifact, err := BaselineOf(context.Background(), fake.repository(t), "uno", "main")
		if err != nil {
			t.Fatal(err)
		}
		if baseline.Home.PSS != 19 {
			t.Fatalf("pss = %d", baseline.Home.PSS)
		}
		if artifact.GetWorkflowRun().GetHeadSHA() != strings.Repeat("2", 40) {
			t.Fatal("want the newest")
		}
	})
	t.Run("ignores forks and expired artifacts", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()},
			artifactOptions{branch: "feature", created: "2026-10-22T00:00:00Z"})
		fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()},
			artifactOptions{headRepoID: 2, created: "2026-10-22T00:00:00Z"})
		fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()},
			artifactOptions{expired: true, created: "2026-10-22T00:00:00Z"})
		baseline, _, err := BaselineOf(context.Background(), fake.repository(t), "uno", "main")
		if err != nil {
			t.Fatal(err)
		}
		if baseline != nil {
			t.Fatal("want nil")
		}
	})
	t.Run("falls back to an older artifact", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addArtifact("uno", map[string]string{"memory.txt": memoryText([3]int{10, 9, 1}, [3]int{10, 9, 1})},
			artifactOptions{created: "2026-10-18T00:00:00Z"})
		fake.addArtifact("uno", map[string]string{"memory.txt": "garbage"},
			artifactOptions{created: "2026-10-20T00:00:00Z"})
		baseline, _, err := BaselineOf(context.Background(), fake.repository(t), "uno", "main")
		if err != nil {
			t.Fatal(err)
		}
		if baseline == nil || baseline.Home.PSS != 9 {
			t.Fatalf("baseline = %+v", baseline)
		}
	})
	t.Run("only reads the implementation", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addArtifact("tauri", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{})
		baseline, _, err := BaselineOf(context.Background(), fake.repository(t), "uno", "main")
		if err != nil {
			t.Fatal(err)
		}
		if baseline != nil {
			t.Fatal("want nil")
		}
	})
	t.Run("ignores an artifact that no workflow run made", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addArtifact("uno", map[string]string{"memory.txt": defaultMemoryText()}, artifactOptions{})
		fake.artifacts[len(fake.artifacts)-1]["workflow_run"] = nil
		baseline, _, err := BaselineOf(context.Background(), fake.repository(t), "uno", "main")
		if err != nil {
			t.Fatal(err)
		}
		if baseline != nil {
			t.Fatal("want nil")
		}
	})
}

func TestFindPullRequests(t *testing.T) {
	t.Run("finds it by the branch of the run", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addPullRequest(7, "", "")
		pulls, err := FindPullRequests(context.Background(), fake.repository(t), "contributor", "feature", testHead)
		if err != nil {
			t.Fatal(err)
		}
		if len(pulls) != 1 || pulls[0].GetNumber() != 7 {
			t.Fatalf("pulls = %v", pulls)
		}
	})
	t.Run("finds it by the commit without the owner", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addPullRequest(7, "", "")
		pulls, err := FindPullRequests(context.Background(), fake.repository(t), "", "feature", testHead)
		if err != nil {
			t.Fatal(err)
		}
		if len(pulls) != 1 {
			t.Fatalf("pulls = %v", pulls)
		}
	})
	t.Run("a newer commit is not the one of the run", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addPullRequest(7, strings.Repeat("c", 40), "")
		pulls, err := FindPullRequests(context.Background(), fake.repository(t), "contributor", "feature", testHead)
		if err != nil {
			t.Fatal(err)
		}
		if len(pulls) != 0 {
			t.Fatalf("pulls = %v", pulls)
		}
	})
	t.Run("a closed pull request is not found", func(t *testing.T) {
		fake := newFakeAPI()
		fake.addPullRequest(7, "", "closed")
		pulls, err := FindPullRequests(context.Background(), fake.repository(t), "contributor", "feature", testHead)
		if err != nil {
			t.Fatal(err)
		}
		if len(pulls) != 0 {
			t.Fatalf("pulls = %v", pulls)
		}
	})
}

func TestUpsertComment(t *testing.T) {
	comment := func(fake *FakeAPI, body, login string, number int) map[string]any {
		c := map[string]any{"id": fake.nextID(), "body": body, "user": map[string]any{"login": login}}
		fake.comments[number] = append(fake.comments[number], c)
		return c
	}
	t.Run("creates the first comment", func(t *testing.T) {
		fake := newFakeAPI()
		result, err := UpsertComment(context.Background(), fake.repository(t), 7, Marker+"\nnew")
		if err != nil {
			t.Fatal(err)
		}
		if result != "created" {
			t.Fatalf("result = %q", result)
		}
		if len(fake.writes) != 1 || fake.writes[0] != [2]string{"POST", "issues/7/comments"} {
			t.Fatalf("writes = %v", fake.writes)
		}
	})
	t.Run("replaces the earlier comment", func(t *testing.T) {
		fake := newFakeAPI()
		old := comment(fake, Marker+"\nold", Bot, 7)
		result, err := UpsertComment(context.Background(), fake.repository(t), 7, Marker+"\nnew")
		if err != nil {
			t.Fatal(err)
		}
		if result != "updated" {
			t.Fatalf("result = %q", result)
		}
		if len(fake.writes) != 1 || fake.writes[0] != [2]string{"PATCH", fmt.Sprintf("issues/comments/%d", asInt(old["id"]))} {
			t.Fatalf("writes = %v", fake.writes)
		}
		if asString(fake.comments[7][0]["body"]) != Marker+"\nnew" {
			t.Fatal("want the new body")
		}
	})
	t.Run("leaves an identical comment alone", func(t *testing.T) {
		fake := newFakeAPI()
		comment(fake, Marker+"\r\nsame\r\n", Bot, 7)
		result, err := UpsertComment(context.Background(), fake.repository(t), 7, Marker+"\nsame\n")
		if err != nil {
			t.Fatal(err)
		}
		if result != "unchanged" || len(fake.writes) != 0 {
			t.Fatalf("result = %q, writes = %v", result, fake.writes)
		}
	})
	t.Run("keeps the oldest and deletes the rest", func(t *testing.T) {
		fake := newFakeAPI()
		first := comment(fake, Marker+"\nfirst", Bot, 7)
		second := comment(fake, Marker+"\nsecond", Bot, 7)
		_, err := UpsertComment(context.Background(), fake.repository(t), 7, Marker+"\nnew")
		if err != nil {
			t.Fatal(err)
		}
		want := [][2]string{
			{"PATCH", fmt.Sprintf("issues/comments/%d", asInt(first["id"]))},
			{"DELETE", fmt.Sprintf("issues/comments/%d", asInt(second["id"]))},
		}
		if len(fake.writes) != 2 || fake.writes[0] != want[0] || fake.writes[1] != want[1] {
			t.Fatalf("writes = %v", fake.writes)
		}
		if len(fake.comments[7]) != 1 || asString(fake.comments[7][0]["body"]) != Marker+"\nnew" {
			t.Fatalf("comments = %v", fake.comments[7])
		}
	})
	t.Run("never touches other people's comments", func(t *testing.T) {
		fake := newFakeAPI()
		theirs := comment(fake, Marker+"\nquoted by a person", "contributor", 7)
		other := map[string]any{"id": fake.nextID(), "body": "a remark by the bot", "user": map[string]any{"login": Bot}}
		fake.comments[7] = append(fake.comments[7], other)
		result, err := UpsertComment(context.Background(), fake.repository(t), 7, Marker+"\nnew")
		if err != nil {
			t.Fatal(err)
		}
		if result != "created" {
			t.Fatalf("result = %q", result)
		}
		if len(fake.writes) != 1 || fake.writes[0] != [2]string{"POST", "issues/7/comments"} {
			t.Fatalf("writes = %v", fake.writes)
		}
		found := 0
		for _, c := range fake.comments[7] {
			if asInt(c["id"]) == asInt(theirs["id"]) || asInt(c["id"]) == asInt(other["id"]) {
				found++
			}
		}
		if found != 2 {
			t.Fatal("want both comments kept")
		}
	})
	t.Run("finds its comment on a later page", func(t *testing.T) {
		fake := newFakeAPI()
		for range 120 {
			comment(fake, "a remark", "contributor", 7)
		}
		mine := comment(fake, Marker+"\nold", Bot, 7)
		for range 20 {
			comment(fake, "another remark", "contributor", 7)
		}
		result, err := UpsertComment(context.Background(), fake.repository(t), 7, Marker+"\nnew")
		if err != nil {
			t.Fatal(err)
		}
		if result != "updated" || len(fake.writes) != 1 || fake.writes[0] != [2]string{"PATCH", fmt.Sprintf("issues/comments/%d", asInt(mine["id"]))} {
			t.Fatalf("result = %q, writes = %v", result, fake.writes)
		}
		pages := 0
		for _, request := range fake.requests {
			if request == "GET /repos/"+testRepository+"/issues/7/comments" {
				pages++
			}
		}
		if pages != 2 {
			t.Fatalf("requests = %v", fake.requests)
		}
	})
	t.Run("tolerates a comment of a deleted user", func(t *testing.T) {
		fake := newFakeAPI()
		fake.comments[7] = []map[string]any{{"id": 1, "body": "text", "user": nil}}
		result, err := UpsertComment(context.Background(), fake.repository(t), 7, Marker+"\nnew")
		if err != nil {
			t.Fatal(err)
		}
		if result != "created" {
			t.Fatalf("result = %q", result)
		}
	})
}

// reportFixture is the whole report against a fake GitHub.
type reportFixture struct {
	fake *FakeAPI
}

func newReportFixture() *reportFixture {
	fake := newFakeAPI()
	mainNumbers := map[string]struct {
		name     string
		home     [3]int
		versions [3]int
	}{
		"mygo":  {"mygo", [3]int{55, 42, 1}, [3]int{64, 49, 1}},
		"tauri": {"Tauri", [3]int{470, 283, 3}, [3]int{492, 305, 3}},
		"uno":   {"Uno", [3]int{110, 101, 1}, [3]int{136, 127, 1}},
	}
	for _, implementation := range []string{"mygo", "tauri", "uno"} {
		numbers := mainNumbers[implementation]
		fake.addWorkflow(implementation, numbers.name, "")
		fake.addArtifact(implementation, map[string]string{"memory.txt": memoryText(numbers.home, numbers.versions)}, artifactOptions{})
	}
	fake.addPullRequest(7, "", "")
	return &reportFixture{fake: fake}
}

func (f *reportFixture) run(t *testing.T, implementations []string, technologies map[string]string, runID *int, pullRequest int, dryRun bool) (int, string, string) {
	t.Helper()
	arguments := struct {
		implementations []string
		technologies    map[string]string
		runID           *int
		pullRequest     int
		dryRun          bool
	}{implementations, technologies, runID, pullRequest, dryRun}
	if arguments.implementations == nil {
		arguments.implementations = []string{"mygo", "tauri", "uno"}
	}
	if arguments.technologies == nil {
		arguments.technologies = map[string]string{"mygo": "mygo 0.5, Go"}
	}
	var outBuffer, errBuffer bytes.Buffer
	oldOut, oldErr := stdout, stderr
	stdout, stderr = &outBuffer, &errBuffer
	defer func() { stdout, stderr = oldOut, oldErr }()
	status, err := Report(context.Background(), f.fake.repository(t), arguments.implementations, arguments.technologies, arguments.runID, arguments.pullRequest, arguments.dryRun)
	if err != nil {
		t.Fatal(err)
	}
	return status, outBuffer.String(), errBuffer.String()
}

func (f *reportFixture) body(t *testing.T) string {
	t.Helper()
	comments := f.fake.comments[7]
	if len(comments) != 1 {
		t.Fatalf("comments = %v", comments)
	}
	body, _ := comments[0]["body"].(string)
	return body
}

func intPtr(value int) *int { return &value }

// An error of the API ends the report, whatever it was reading or writing, and
// nothing is written before the numbers are all there.
func TestReportStopsAtAnErrorOfTheAPI(t *testing.T) {
	stages := []struct {
		name   string
		manual bool
		fails  string
	}{
		{"reading the run", false, `^GET /actions/runs/\d+$`},
		{"listing the pull requests of the branch", false, `^GET /pulls$`},
		{"listing the workflows", false, `^GET /actions/workflows$`},
		{"listing the runs of the commit", false, `^GET /actions/runs$`},
		{"reading the repository", false, `^GET $`},
		{"listing the artifacts of a run", false, `^GET /actions/runs/\d+/artifacts$`},
		{"listing the artifacts of the default branch", false, `^GET /actions/artifacts$`},
		{"reading the pull request again", false, `^GET /pulls/7$`},
		{"listing the comments", false, `^GET /issues/7/comments$`},
		{"writing the comment", false, `^POST /issues/7/comments$`},
		{"reading the pull request", true, `^GET /pulls/7$`},
	}
	for _, stage := range stages {
		t.Run(stage.name, func(t *testing.T) {
			f := newReportFixture()
			run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
			fails := regexp.MustCompile(stage.fails)
			f.fake.failing = func(request string) bool { return fails.MatchString(request) }
			var runID *int
			pullRequest := 7
			if !stage.manual {
				runID, pullRequest = intPtr(run), 0
			}
			var outBuffer, errBuffer bytes.Buffer
			oldOut, oldErr := stdout, stderr
			stdout, stderr = &outBuffer, &errBuffer
			defer func() { stdout, stderr = oldOut, oldErr }()
			_, err := Report(context.Background(), f.fake.repository(t), []string{"mygo", "tauri", "uno"}, map[string]string{}, runID, pullRequest, false)
			if err == nil || !strings.Contains(err.Error(), "refused by the test") {
				t.Fatalf("err = %v", err)
			}
			if stage.name != "writing the comment" && len(f.fake.writes) != 0 {
				t.Fatalf("writes = %v", f.fake.writes)
			}
		})
	}
}

func TestShortSHA(t *testing.T) {
	if shortSHA(testHead) != "aaaaaaa" || shortSHA("abc") != "abc" {
		t.Fatalf("shortSHA = %q, %q", shortSHA(testHead), shortSHA("abc"))
	}
}

func TestReport(t *testing.T) {
	t.Run("measured and main for the rest", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: memoryText([3]int{56, 43, 1}, [3]int{65, 50, 1})})
		status, _, _ := f.run(t, nil, nil, intPtr(run), 0, false)
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
		var names []string
		for _, row := range renderTable(f.body(t))[2:] {
			names = append(names, strings.TrimSpace(row[1-1]))
		}
		if strings.Join(names, ",") != "mygo,Uno (main),Tauri (main)" {
			t.Fatalf("names = %v", names)
		}
		if !strings.Contains(f.body(t), "43 (+1)") {
			t.Fatal("want the difference")
		}
		if !strings.Contains(f.body(t), "No change from main beyond the noise of CI in mygo.") {
			t.Fatal("want the no-change line")
		}
	})
	t.Run("waits for the others still running", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		f.fake.addRun("tauri", runOptions{status: "in_progress", conclusion: ""})
		status, _, errors := f.run(t, nil, nil, intPtr(run), 0, false)
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
		if !strings.Contains(errors, "Waiting for Tauri") {
			t.Fatalf("errors = %q", errors)
		}
		if len(f.fake.comments) != 0 {
			t.Fatal("want no comment")
		}
	})
	t.Run("waiting takes few requests", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		f.fake.addRun("tauri", runOptions{status: "in_progress", conclusion: ""})
		var outBuffer, errBuffer bytes.Buffer
		oldOut, oldErr := stdout, stderr
		stdout, stderr = &outBuffer, &errBuffer
		defer func() { stdout, stderr = oldOut, oldErr }()
		repo := f.fake.repository(t)
		if _, err := Report(context.Background(), repo, []string{"mygo", "tauri", "uno"}, map[string]string{}, intPtr(run), 0, false); err != nil {
			t.Fatal(err)
		}
		requests := f.fake.requests
		if len(requests) > 6 {
			t.Fatalf("requests = %v", requests)
		}
		for _, request := range requests {
			if strings.Contains(request, "artifacts") {
				t.Fatalf("requests = %v", requests)
			}
		}
	})
	t.Run("last run to end writes all of them", func(t *testing.T) {
		f := newReportFixture()
		f.fake.addRun("tauri", runOptions{hasText: true, text: memoryText([3]int{470, 283, 3}, [3]int{492, 305, 3})})
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		f.run(t, nil, nil, intPtr(run), 0, false)
		var names []string
		for _, row := range renderTable(f.body(t))[2:] {
			names = append(names, strings.TrimSpace(row[0]))
		}
		if strings.Join(names, ",") != "mygo,Uno (main),Tauri" {
			t.Fatalf("names = %v", names)
		}
	})
	t.Run("ended run counts as ended when the list is late", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText(), status: "in_progress", conclusion: ""})
		status, _, errors := f.run(t, nil, nil, intPtr(run), 0, false)
		if strings.Contains(errors, "Waiting") {
			t.Fatalf("errors = %q", errors)
		}
		if !strings.Contains(f.body(t), "mygo") {
			t.Fatal("want mygo in the body")
		}
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
	})
	t.Run("failed run reported as such", func(t *testing.T) {
		f := newReportFixture()
		f.fake.addRun("tauri", runOptions{conclusion: "failure"})
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		f.run(t, nil, nil, intPtr(run), 0, false)
		if !strings.Contains(f.body(t), "No result, the run failed or was cancelled: Tauri.") {
			t.Fatalf("body =\n%s", f.body(t))
		}
	})
	t.Run("invalid artifact is no result", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: "not the numbers"})
		f.run(t, nil, nil, intPtr(run), 0, false)
		if !strings.Contains(f.body(t), "No result, the run failed or was cancelled: mygo.") {
			t.Fatalf("body =\n%s", f.body(t))
		}
	})
	t.Run("second report replaces the first", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: memoryText([3]int{56, 43, 1}, [3]int{65, 50, 1})})
		f.run(t, nil, nil, intPtr(run), 0, false)
		f.fake.pulls[7]["head"].(map[string]any)["sha"] = strings.Repeat("c", 40)
		later := f.fake.addRun("mygo", runOptions{hasText: true, text: memoryText([3]int{60, 47, 1}, [3]int{70, 54, 1}), sha: strings.Repeat("c", 40)})
		f.run(t, nil, nil, intPtr(later), 0, false)
		if len(f.fake.comments[7]) != 1 {
			t.Fatalf("comments = %v", f.fake.comments[7])
		}
		if !strings.Contains(f.body(t), "47 (+5)") {
			t.Fatalf("body =\n%s", f.body(t))
		}
		if !strings.Contains(f.body(t), "`ccccccc`") {
			t.Fatalf("body =\n%s", f.body(t))
		}
		var methods []string
		for _, write := range f.fake.writes {
			methods = append(methods, write[0])
		}
		if strings.Join(methods, ",") != "POST,PATCH" {
			t.Fatalf("methods = %v", methods)
		}
	})
	t.Run("ignores the run of an older commit", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		f.fake.pulls[7]["head"].(map[string]any)["sha"] = strings.Repeat("c", 40)
		status, _, errors := f.run(t, nil, nil, intPtr(run), 0, false)
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
		if !strings.Contains(errors, "No open pull request has aaaaaaa") {
			t.Fatalf("errors = %q", errors)
		}
		if len(f.fake.writes) != 0 {
			t.Fatal("want no writes")
		}
	})
	t.Run("no comment when the commit changes during the report", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		f.fake.onGet = func(path string, params map[string]any) {
			if path == "pulls/7" {
				f.fake.pulls[7]["head"].(map[string]any)["sha"] = strings.Repeat("c", 40)
			}
		}
		status, _, errors := f.run(t, nil, nil, intPtr(run), 0, false)
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
		if !strings.Contains(errors, "has moved on") {
			t.Fatalf("errors = %q", errors)
		}
		if len(f.fake.writes) != 0 {
			t.Fatal("want no writes")
		}
	})
	t.Run("ignores runs that are not of a pull request", func(t *testing.T) {
		f := newReportFixture()
		run := f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText(), event: "push", sha: testMain})
		status, _, errors := f.run(t, nil, nil, intPtr(run), 0, false)
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
		if !strings.Contains(errors, "not one of a pull request") {
			t.Fatalf("errors = %q", errors)
		}
		if len(f.fake.writes) != 0 {
			t.Fatal("want no writes")
		}
	})
	t.Run("skips a deleted workflow", func(t *testing.T) {
		f := newReportFixture()
		f.fake.workflows[1]["state"] = "deleted"
		f.run(t, nil, nil, nil, 7, false)
		if strings.Contains(f.body(t), "Tauri") {
			t.Fatalf("body =\n%s", f.body(t))
		}
	})
	t.Run("no workflow is skipped", func(t *testing.T) {
		f := newReportFixture()
		status, _, errors := f.run(t, []string{"nothing"}, map[string]string{}, nil, 7, false)
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
		if !strings.Contains(errors, "None of the implementations has a workflow") {
			t.Fatalf("errors = %q", errors)
		}
	})
	t.Run("manual mode does not wait", func(t *testing.T) {
		f := newReportFixture()
		f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		f.fake.addRun("tauri", runOptions{status: "in_progress", conclusion: ""})
		f.run(t, nil, nil, nil, 7, false)
		if !strings.Contains(f.body(t), "Still running: Tauri.") {
			t.Fatalf("body =\n%s", f.body(t))
		}
	})
	t.Run("closed pull request does nothing", func(t *testing.T) {
		f := newReportFixture()
		f.fake.pulls[7]["state"] = "closed"
		status, _, errors := f.run(t, nil, nil, nil, 7, false)
		if status != 0 {
			t.Fatalf("status = %d", status)
		}
		if !strings.Contains(errors, "is not open") {
			t.Fatalf("errors = %q", errors)
		}
	})
	t.Run("dry run prints and writes nothing", func(t *testing.T) {
		f := newReportFixture()
		f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		_, output, _ := f.run(t, nil, nil, nil, 7, true)
		if !strings.HasPrefix(output, Marker) {
			t.Fatalf("output = %q", output)
		}
		if len(f.fake.writes) != 0 {
			t.Fatal("want no writes")
		}
	})
	t.Run("new implementation has no baseline", func(t *testing.T) {
		f := newReportFixture()
		var kept []map[string]any
		for _, artifact := range f.fake.artifacts {
			if asString(artifact["name"]) != "memory-uno" {
				kept = append(kept, artifact)
			}
		}
		f.fake.artifacts = kept
		run := f.fake.addRun("uno", runOptions{hasText: true, text: defaultMemoryText()})
		f.run(t, nil, nil, intPtr(run), 0, false)
		if !strings.Contains(f.body(t), "Uno (no baseline)") {
			t.Fatalf("body =\n%s", f.body(t))
		}
	})
	t.Run("writes the step summary", func(t *testing.T) {
		f := newReportFixture()
		f.fake.addRun("mygo", runOptions{hasText: true, text: defaultMemoryText()})
		directory := t.TempDir()
		summary := filepath.Join(directory, "summary.md")
		t.Setenv("GITHUB_STEP_SUMMARY", summary)
		f.run(t, nil, nil, nil, 7, false)
		content, err := os.ReadFile(summary)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(content), "### Memory use") {
			t.Fatalf("summary = %q", content)
		}
		if strings.Contains(string(content), Marker) {
			t.Fatal("want no marker in the summary")
		}
	})
}

func TestRunValidation(t *testing.T) {
	t.Run("rejects a bad repository", func(t *testing.T) {
		if err := Run([]string{"--repository", "x;y", "--implementations", "mygo", "--pull-request", "1"}); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("rejects a bad implementation", func(t *testing.T) {
		if err := Run([]string{"--repository", testRepository, "--implementations", "../mygo", "--pull-request", "1"}); err == nil {
			t.Fatal("want an error")
		}
		if err := Run([]string{"--repository", testRepository, "--implementations", "", "--pull-request", "1"}); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("needs a run or a pull request", func(t *testing.T) {
		if err := Run([]string{"--repository", testRepository, "--implementations", "mygo"}); err == nil {
			t.Fatal("want an error")
		}
		if err := Run([]string{"--repository", testRepository, "--implementations", "mygo", "--run", "1", "--pull-request", "2"}); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("needs GH_TOKEN to read the API", func(t *testing.T) {
		t.Setenv("GH_TOKEN", "")
		if err := Run([]string{"--repository", testRepository, "--implementations", "mygo", "--pull-request", "1", "--docs", "missing.md"}); err == nil ||
			!strings.Contains(err.Error(), "GH_TOKEN") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("reports a failure of the api as an error", func(t *testing.T) {
		t.Setenv("GH_TOKEN", "test-token")
		old := reportFunc
		reportFunc = func(context.Context, *Repository, []string, map[string]string, *int, int, bool) (int, error) {
			return 0, errors.New("boom")
		}
		defer func() { reportFunc = old }()
		var errBuffer bytes.Buffer
		oldErr := stderr
		stderr = &errBuffer
		defer func() { stderr = oldErr }()
		if err := Run([]string{"--repository", testRepository, "--implementations", "mygo", "--pull-request", "1", "--docs", "missing.md"}); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("token that cannot write warns on one line", func(t *testing.T) {
		t.Setenv("GH_TOKEN", "test-token")
		old := reportFunc
		reportFunc = func(context.Context, *Repository, []string, map[string]string, *int, int, bool) (int, error) {
			return 0, errors.New("POST https://api.github.com/repos/o/r/issues/7/comments: 403 Resource not accessible by integration []\nsecond line 100%")
		}
		defer func() { reportFunc = old }()
		var buffer bytes.Buffer
		oldOut := stdout
		stdout = &buffer
		defer func() { stdout = oldOut }()
		if err := Run([]string{"--repository", testRepository, "--implementations", "mygo", "--pull-request", "1", "--docs", "missing.md"}); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(buffer.String(), "\n"), "\n")
		if len(lines) != 1 {
			t.Fatalf("output = %q", buffer.String())
		}
		if !strings.Contains(buffer.String(), "%0Asecond line 100%25") {
			t.Fatalf("output = %q", buffer.String())
		}
	})
	t.Run("token that cannot write is a warning not a failure", func(t *testing.T) {
		t.Setenv("GH_TOKEN", "test-token")
		old := reportFunc
		reportFunc = func(context.Context, *Repository, []string, map[string]string, *int, int, bool) (int, error) {
			return 0, errors.New("POST https://api.github.com/repos/o/r/issues/7/comments: 403 Resource not accessible by integration []")
		}
		defer func() { reportFunc = old }()
		var buffer bytes.Buffer
		oldOut := stdout
		stdout = &buffer
		defer func() { stdout = oldOut }()
		if err := Run([]string{"--repository", testRepository, "--implementations", "mygo", "--pull-request", "1", "--docs", "missing.md"}); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(buffer.String(), "::warning::") {
			t.Fatalf("output = %q", buffer.String())
		}
	})
}
