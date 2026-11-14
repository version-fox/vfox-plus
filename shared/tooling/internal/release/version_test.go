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

package release

import (
	"errors"
	"strings"
	"testing"
)

// fakeRepo is a checkout with the given tags and files. A file that is not in
// files cannot be read.
type fakeRepo struct {
	tags  []string
	files map[string]string
}

func (r fakeRepo) TagExists(tag string) (bool, error) {
	for _, t := range r.tags {
		if t == tag {
			return true, nil
		}
	}
	return false, nil
}

func (r fakeRepo) Tags() ([]string, error) { return r.tags, nil }

func (r fakeRepo) ReadFile(path string) (string, error) {
	if text, ok := r.files[path]; ok {
		return text, nil
	}
	return "", errors.New("no such file: " + path)
}

// sayingVersion is the content of the three files of a commit that says version.
func sayingVersion(version string) map[string]string {
	return map[string]string{
		VersionFile:  version + "\n",
		AppFile:      "package ui\n\nconst Version = \"" + version + "\"\n",
		MetainfoFile: "<releases>\n    <release version=\"" + version + "\" date=\"2026-10-17\"/>\n</releases>\n",
	}
}

// checkResult fails the test unless the result is want, or, when wantErr is set,
// an error that contains wantErr.
func checkResult(t *testing.T, got string, err error, want, wantErr string) {
	t.Helper()
	if wantErr != "" {
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("got %q, %v; want an error containing %q", got, err, wantErr)
		}
		return
	}
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

func TestCheckVersionOfAManualRun(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		ref     string
		tags    []string
		want    string
		wantErr string
	}{
		{name: "a new version from main", input: "1.0.0", ref: "refs/heads/main", tags: []string{"v0.1.0"}, want: "1.0.0"},
		{name: "spaces are removed", input: " 1.0 .0\n", ref: "refs/heads/main", want: "1.0.0"},
		{name: "not from main", input: "1.0.0", ref: "refs/heads/feature",
			wantErr: "Run the release from main, not from feature"},
		{name: "not a version", input: "1.0", ref: "refs/heads/main",
			wantErr: "Not a version (major.minor.patch, without v): 1.0"},
		{name: "the tag exists", input: "0.1.0", ref: "refs/heads/main", tags: []string{"v0.1.0"},
			wantErr: "Tag v0.1.0 exists already"},
		{name: "not newer than the newest tag", input: "0.1.5", ref: "refs/heads/main", tags: []string{"v0.2.0"},
			wantErr: "v0.1.5 is not newer than v0.2.0"},
		{name: "tags that are not versions do not count", input: "0.3.0", ref: "refs/heads/main",
			tags: []string{"v0.2.0", "vendor", "v0.10.0-rc1"}, want: "0.3.0"},
		{name: "versions compare by number, not by text", input: "0.10.0", ref: "refs/heads/main",
			tags: []string{"v0.9.0"}, want: "0.10.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := Request{Event: EventManual, Ref: c.ref, RefName: strings.TrimPrefix(c.ref, "refs/heads/"), Input: c.input}
			got, err := CheckVersion(req, fakeRepo{tags: c.tags})
			checkResult(t, got, err, c.want, c.wantErr)
		})
	}
}

func TestCheckVersionOfATagRun(t *testing.T) {
	tagRun := func(files map[string]string) (string, error) {
		return CheckVersion(Request{Event: "push", Ref: "refs/tags/v1.0.0", RefName: "v1.0.0"}, fakeRepo{files: files})
	}
	t.Run("the commit says the version", func(t *testing.T) {
		got, err := tagRun(sayingVersion("1.0.0"))
		checkResult(t, got, err, "1.0.0", "")
	})
	t.Run("the commit says another version", func(t *testing.T) {
		_, err := tagRun(sayingVersion("0.9.0"))
		checkResult(t, "", err, "", "The commit of v1.0.0 must say 1.0.0")
	})
	t.Run("a file of the version is missing", func(t *testing.T) {
		files := sayingVersion("1.0.0")
		delete(files, AppFile)
		_, err := tagRun(files)
		checkResult(t, "", err, "", "must say 1.0.0 in mygo/VERSION, mygo/ui/app.go and the metainfo")
	})
}

func TestNewestTagComparesNumbers(t *testing.T) {
	if got := newestTag([]string{"v0.9.0", "v0.10.0", "vendor", "v0.10.0-rc1"}); got != "v0.10.0" {
		t.Errorf("newestTag = %q, want v0.10.0", got)
	}
	if got := newestTag([]string{"vendor"}); got != "" {
		t.Errorf("newestTag of no version = %q, want an empty string", got)
	}
}
