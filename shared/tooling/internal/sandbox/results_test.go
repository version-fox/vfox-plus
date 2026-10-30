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

package sandbox

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// fakeRepo is a session in a repository with the files that the names of the images
// are made of.
func fakeRepo(t *testing.T) *session {
	t.Helper()
	repo := t.TempDir()
	for path, content := range map[string]string{
		"shared/sandbox/base.Dockerfile": "FROM ubuntu\n",
		"shared/tooling/go.mod":          "module tooling\n",
		"mygo/sandbox/Dockerfile":        "FROM base\n",
		"mygo/versions.env":              "VFOX_VERSION=1.0.13\nVFOX_SHA256=abc\n",
	} {
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := newSession("mygo", repo)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// fakeDocker stands in for Docker: it says which images exist, records the commands,
// and lets the test act as the container.
type fakeDocker struct {
	images  map[string]bool
	calls   [][]string
	onRun   func(args []string) error
	listing string
}

func (f *fakeDocker) install(t *testing.T) {
	t.Helper()
	exists, run, output := imageExists, docker, dockerOutput
	t.Cleanup(func() { imageExists, docker, dockerOutput = exists, run, output })
	imageExists = func(image string) bool { return f.images[image] }
	docker = func(args ...string) error {
		f.calls = append(f.calls, args)
		if f.onRun != nil {
			return f.onRun(args)
		}
		return nil
	}
	dockerOutput = func(args ...string) ([]byte, error) {
		if len(args) > 1 && args[0] == "image" && args[1] == "ls" {
			return []byte(f.listing), nil
		}
		return nil, errors.New("not available")
	}
}

// stuckResults makes a directory of results that the user of the host cannot delete,
// as the stage of a package that a task wrote under rootless Docker: a directory
// without write permission, with a file in it.
func stuckResults(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("only a user who is not root is refused")
	}
	stage := filepath.Join(dir, "stage-x64")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "vfox-plus.exe"), []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(stage, 0o555); err != nil {
		t.Fatal(err)
	}
	// So that the directory of the test can be deleted if the test fails.
	t.Cleanup(func() { _ = os.Chmod(stage, 0o755) })
	if err := os.RemoveAll(dir); err == nil {
		t.Fatal("the host user can delete what the test made undeletable")
	}
	return stage
}

// root is what root in a container does to the directory that it is given.
func root(t *testing.T, mount, target string) {
	t.Helper()
	path := filepath.Join(mount, strings.TrimPrefix(target, "/parent/"))
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(p, 0o755)
		}
		return nil
	})
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveResultsDeletesWhatTheHostMayDelete(t *testing.T) {
	s := fakeRepo(t)
	fake := &fakeDocker{}
	fake.install(t)
	dir := filepath.Join(s.results, "check")
	if err := os.MkdirAll(filepath.Join(dir, "deep", "er"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deep", "er", "file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.removeResults(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the results are left: %v", err)
	}
	if len(fake.calls) != 0 {
		t.Errorf("a container was asked to delete what the host deletes: %v", fake.calls)
	}
	// What is not there is deleted, too.
	if err := s.removeResults(dir); err != nil {
		t.Errorf("results that are not there: %v", err)
	}
}

func TestRemoveResultsAsksAContainerWhenTheHostIsRefused(t *testing.T) {
	s := fakeRepo(t)
	implImage, _ := s.implImage()
	dir := filepath.Join(s.results, "package")
	stuckResults(t, dir)

	fake := &fakeDocker{images: map[string]bool{implImage: true}}
	fake.onRun = func(args []string) error {
		root(t, filepath.Dir(dir), args[len(args)-1])
		return nil
	}
	fake.install(t)
	if err := s.removeResults(dir); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{
		"run", "--rm", "--network", "none", "--volume", filepath.Dir(dir) + ":/parent", "--entrypoint", "rm",
		implImage, "-rf", "--", "/parent/package",
	}}
	if !reflect.DeepEqual(fake.calls, want) {
		t.Errorf("docker was run as\n%v\nwant\n%v", fake.calls, want)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the results are left: %v", err)
	}
}

func TestRemoveResultsTakesAnotherImageWhenThatOfTheImplementationIsGone(t *testing.T) {
	s := fakeRepo(t)
	implImage, _ := s.implImage()
	baseImage, _ := s.baseImage()
	cases := map[string]struct {
		images map[string]bool
		want   string
	}{
		"the image of the implementation": {map[string]bool{implImage: true, baseImage: true, cleanBaseImage: true}, implImage},
		"the base image":                  {map[string]bool{baseImage: true, cleanBaseImage: true}, baseImage},
		"the clean Ubuntu":                {map[string]bool{cleanBaseImage: true}, cleanBaseImage},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "screenshots")
			stuckResults(t, dir)
			fake := &fakeDocker{images: c.images}
			fake.onRun = func(args []string) error {
				root(t, filepath.Dir(dir), args[len(args)-1])
				return nil
			}
			fake.install(t)
			if err := s.removeResults(dir); err != nil {
				t.Fatal(err)
			}
			if len(fake.calls) != 1 || fake.calls[0][8] != c.want {
				t.Errorf("docker was run as %v, want the image %s", fake.calls, c.want)
			}
		})
	}
}

func TestRemoveResultsSaysWhatToDoWhenNothingCanDeleteThem(t *testing.T) {
	s := fakeRepo(t)
	dir := filepath.Join(t.TempDir(), "package")
	stuckResults(t, dir)
	fake := &fakeDocker{}
	fake.install(t)
	err := s.removeResults(dir)
	if err == nil || !strings.Contains(err.Error(), "permission denied") || !strings.Contains(err.Error(), "delete the directory as root") {
		t.Fatalf("got %v, want the refusal of the host with what to do", err)
	}
	if len(fake.calls) != 0 {
		t.Errorf("docker was run without an image: %v", fake.calls)
	}
}

func TestRemoveResultsReportsAContainerThatFails(t *testing.T) {
	s := fakeRepo(t)
	implImage, _ := s.implImage()
	dir := filepath.Join(t.TempDir(), "package")
	stuckResults(t, dir)
	fake := &fakeDocker{images: map[string]bool{implImage: true}, onRun: func([]string) error { return errors.New("boom") }}
	fake.install(t)
	err := s.removeResults(dir)
	if err == nil || !strings.Contains(err.Error(), "delete "+dir+" in a container: boom") {
		t.Fatalf("got %v", err)
	}
}

func TestCleanDeletesTheResultsBeforeTheImagesThatCanDeleteThem(t *testing.T) {
	s := fakeRepo(t)
	implImage, _ := s.implImage()
	stuckResults(t, filepath.Join(s.results, "package"))

	fake := &fakeDocker{
		images:  map[string]bool{implImage: true},
		listing: "vfox-plus-sandbox-mygo:aaa\nvfox-plus-sandbox-mygo:bbb\n\n",
	}
	fake.onRun = func(args []string) error {
		if args[0] == "run" {
			root(t, filepath.Dir(s.results), args[len(args)-1])
		}
		return nil
	}
	fake.install(t)
	if err := s.clean(); err != nil {
		t.Fatal(err)
	}
	if len(fake.calls) < 2 {
		t.Fatalf("clean ran %v", fake.calls)
	}
	var commands []string
	for _, call := range fake.calls {
		commands = append(commands, strings.Join(call[:min(len(call), 2)], " "))
	}
	for _, volume := range s.config.cacheVolumes {
		name, _, _ := strings.Cut(volume, ":")
		if !containsCall(fake.calls, "volume", "rm", "--force", name) {
			t.Errorf("the volume %s is not deleted: %v", name, fake.calls)
		}
	}
	if commands[0] != "run --rm" || commands[1] != "image rm" {
		t.Errorf("clean ran %v: the container that deletes the results must come before the images go", commands)
	}
	if !containsCall(fake.calls, "image", "rm", "vfox-plus-sandbox-mygo:aaa", "vfox-plus-sandbox-mygo:bbb") {
		t.Errorf("the images are not deleted: %v", fake.calls)
	}
	if _, err := os.Stat(s.results); !os.IsNotExist(err) {
		t.Errorf("the results are left: %v", err)
	}
}

func TestCleanDeletesImagesAndVolumesAlsoWhenTheResultsStay(t *testing.T) {
	s := fakeRepo(t)
	stuckResults(t, filepath.Join(s.results, "package"))
	// Without an image there is nothing to delete the results with.
	fake := &fakeDocker{listing: "vfox-plus-sandbox-mygo:aaa\n"}
	fake.install(t)
	err := s.clean()
	if err == nil || !strings.Contains(err.Error(), "delete the directory as root") {
		t.Fatalf("got %v, want the results to be reported", err)
	}
	if !containsCall(fake.calls, "image", "rm", "vfox-plus-sandbox-mygo:aaa") {
		t.Errorf("the image is not deleted: %v", fake.calls)
	}
}

func containsCall(calls [][]string, want ...string) bool {
	for _, call := range calls {
		if reflect.DeepEqual(call, want) {
			return true
		}
	}
	return false
}
