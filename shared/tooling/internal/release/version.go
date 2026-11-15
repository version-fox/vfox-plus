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
	"fmt"
	"strings"
)

// Request is what the workflow knows about the run that asks for a release.
type Request struct {
	Event   string // GITHUB_EVENT_NAME
	Ref     string // GITHUB_REF
	RefName string // GITHUB_REF_NAME
	Input   string // the version typed for a manual run
}

// Repository is the part of the checkout that CheckVersion reads.
type Repository interface {
	// TagExists reports whether the tag exists.
	TagExists(tag string) (bool, error)
	// Tags lists the tags that start with v.
	Tags() ([]string, error)
	// ReadFile reads a file, by its path from the root of the checkout.
	ReadFile(path string) (string, error)
}

// CheckVersion returns the version that the run publishes, or the reason why the
// run cannot release it.
//
// A run that someone starts by hand publishes the version in its input, from
// main: the version must be new, so no tag has it, and it is newer than the newest
// tag. A run that a tag starts publishes the version of the tag, and the commit of
// the tag must say that version in the files that carry it.
func CheckVersion(req Request, repo Repository) (string, error) {
	manual := req.Event == EventManual
	version := req.Input
	if manual {
		if req.Ref != mainRef {
			return "", fmt.Errorf("Run the release from main, not from %s", req.RefName)
		}
	} else {
		version = strings.TrimPrefix(req.RefName, "v")
	}
	// Every space goes, as tr -d '[:space:]' removed it in the shell.
	version = strings.Join(strings.Fields(version), "")
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("Not a version (major.minor.patch, without v): %s", version)
	}
	if !manual {
		if err := checkFiles(version, repo.ReadFile); err != nil {
			return "", fmt.Errorf("The commit of v%s must say %s in %s, %s and the metainfo: %w",
				version, version, VersionFile, AppFile, err)
		}
		return version, nil
	}
	exists, err := repo.TagExists("v" + version)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("Tag v%s exists already", version)
	}
	tags, err := repo.Tags()
	if err != nil {
		return "", err
	}
	if newest := newestTag(tags); newest != "" {
		mine, _ := parse(version)      // checked by versionPattern above
		theirs, _ := parse(newest[1:]) // newestTag returns only versions
		if compareVersions(mine, theirs) <= 0 {
			return "", fmt.Errorf("v%s is not newer than %s", version, newest)
		}
	}
	return version, nil
}

// checkFiles reports an error unless the files that carry the version all say it.
func checkFiles(version string, read func(path string) (string, error)) error {
	versionText, err := read(VersionFile)
	if err != nil {
		return err
	}
	app, err := read(AppFile)
	if err != nil {
		return err
	}
	metainfo, err := read(MetainfoFile)
	if err != nil {
		return err
	}
	if strings.Join(strings.Fields(versionText), "") != version {
		return fmt.Errorf("%s does not say %s", VersionFile, version)
	}
	if !strings.Contains(app, `const Version = "`+version+`"`) {
		return fmt.Errorf("%s does not say %s", AppFile, version)
	}
	if !strings.Contains(metainfo, `<release version="`+version+`"`) {
		return fmt.Errorf("%s has no release %s", MetainfoFile, version)
	}
	return nil
}
