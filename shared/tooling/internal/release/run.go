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
	"flag"
	"fmt"
	"time"
)

// CheckVersionRun is vfxtool check-version. It prints VERSION=<version>, which the
// workflow appends to its environment file: vfxtool check-version ... >> "$GITHUB_ENV".
func CheckVersionRun(args []string) error {
	fs := flag.NewFlagSet("check-version", flag.ContinueOnError)
	event := fs.String("event", "", "the event of the run (GITHUB_EVENT_NAME)")
	ref := fs.String("ref", "", "the ref of the run (GITHUB_REF)")
	refName := fs.String("ref-name", "", "the name of the ref (GITHUB_REF_NAME)")
	input := fs.String("input", "", "the version typed for a manual run")
	root := fs.String("root", ".", "the checkout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	version, err := CheckVersion(Request{Event: *event, Ref: *ref, RefName: *refName, Input: *input}, Git{Dir: *root})
	if err != nil {
		return reported(err)
	}
	fmt.Printf("VERSION=%s\n", version)
	return nil
}

// SetVersionRun is vfxtool set-version: it writes the version into the files that
// carry it, for a release.
func SetVersionRun(args []string) error {
	fs := flag.NewFlagSet("set-version", flag.ContinueOnError)
	version := fs.String("version", "", "the version, as major.minor.patch")
	date := fs.String("date", "", "the date of the release list entry, as YYYY-MM-DD (default: today, in UTC)")
	root := fs.String("root", ".", "the checkout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *date == "" {
		*date = time.Now().UTC().Format("2006-01-02")
	}
	if err := SetVersion(*root, *version, *date); err != nil {
		return reported(err)
	}
	return nil
}

// CheckAssetsRun is vfxtool check-assets: it prints the files that a release of the
// version uploads, one per line.
func CheckAssetsRun(args []string) error {
	fs := flag.NewFlagSet("check-assets", flag.ContinueOnError)
	version := fs.String("version", "", "the version of the release")
	root := fs.String("root", ".", "the directory that holds the dist directories")
	if err := fs.Parse(args); err != nil {
		return err
	}
	files, err := CheckAssets(*root, *version)
	if err != nil {
		return reported(err)
	}
	for _, file := range files {
		fmt.Println(file)
	}
	return nil
}
