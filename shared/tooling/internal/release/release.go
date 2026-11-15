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

// Package release holds the decisions of the release workflow
// (.github/workflows/release.yml): the version a run publishes, the files that
// carry it, and the assets a release uploads. The workflow calls them through
// vfxtool, so that they are tested here rather than written in shell.
package release

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ErrReported marks an error that the command has already written as a workflow
// annotation, so that vfxtool does not print it a second time.
var ErrReported = errors.New("reported as an annotation")

// The files that carry the version, from the root of the checkout.
const (
	VersionFile  = "mygo/VERSION"
	AppFile      = "mygo/ui/app.go"
	MetainfoFile = "shared/packaging/linux/dev.vfox.VfoxPlus.metainfo.xml"
)

// EventManual is the event of a run that someone starts by hand, with the
// version as its input.
const EventManual = "workflow_dispatch"

// mainRef is the only ref that a release is started from by hand.
const mainRef = "refs/heads/main"

var (
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	versionConst   = regexp.MustCompile(`(?m)^const Version = "[^"]*"`)
)

// reported writes err as an annotation of the workflow, and returns ErrReported.
func reported(err error) error {
	fmt.Fprintf(os.Stderr, "::error::%v\n", err)
	return ErrReported
}

// parse reads a version of the form major.minor.patch into its three numbers.
func parse(version string) ([3]int, error) {
	var nums [3]int
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return nums, fmt.Errorf("not a version: %s", version)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nums, fmt.Errorf("not a version: %s", version)
		}
		nums[i] = n
	}
	return nums, nil
}

// compareVersions orders two versions by their numbers: -1, 0 or 1.
func compareVersions(a, b [3]int) int {
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}

// newestTag returns the highest tag of the form v<major>.<minor>.<patch>, compared
// by number, or "" when there is none. Other tags do not count.
func newestTag(tags []string) string {
	newest := ""
	var newestNums [3]int
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "v") || !versionPattern.MatchString(tag[1:]) {
			continue
		}
		nums, err := parse(tag[1:])
		if err != nil {
			continue
		}
		if newest == "" || compareVersions(nums, newestNums) > 0 {
			newest, newestNums = tag, nums
		}
	}
	return newest
}
