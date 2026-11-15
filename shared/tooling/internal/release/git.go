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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git is a checkout, read through the git command and the file system.
type Git struct {
	Dir string
}

// git runs git in the checkout, and returns its standard output.
func (g Git) git(args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", g.Dir}, args...)...).Output()
	return string(out), err
}

// TagExists reports whether the tag exists.
func (g Git) TagExists(tag string) (bool, error) {
	_, err := g.git("rev-parse", "-q", "--verify", "refs/tags/"+tag)
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil // rev-parse exits with 1 when the tag is not there
	}
	return false, err
}

// Tags lists the tags whose names start with v.
func (g Git) Tags() ([]string, error) {
	out, err := g.git("tag", "--list", "v*")
	if err != nil {
		return nil, err
	}
	return strings.Fields(out), nil
}

// ReadFile reads a file of the checkout, by its path from the root.
func (g Git) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(filepath.Join(g.Dir, filepath.FromSlash(path)))
	return string(data), err
}
