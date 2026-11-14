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

// Package runtask runs one task inside a sandbox container of the mygo
// implementation, replacing mygo/sandbox/run-task.sh and
// shared/sandbox/task-lib.sh.
//
// The steps mirror the jobs of .github/workflows/mygo.yml; keep them in sync.
//
// The repository is mounted read-only at /src and the results directory at
// /out. A task works on a private copy of the repository as the
// unprivileged user tester, so the only thing that reaches the host is what
// it copies to /out.
package runtask

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// TestTheCensusGroupsTakeEachScreenOnce holds the groups of the census to the tests of the census: each top-level test is in some
// group, and each window of the sweep (its scales, in English and in Chinese) is in exactly one of the two groups that take the sweep.
// A group that matched nothing would pass without a screen, so a rename of a test that is not here fails it.
func TestTheCensusGroupsTakeEachScreenOnce(t *testing.T) {
	// The patterns are matched as go test matches -run: the part before the slash against the top-level test, the part after it
	// against the subtest, and a pattern without a part after the slash takes every subtest of the test.
	type matcher struct{ top, sub *regexp.Regexp }
	matchers := map[string]matcher{}
	for _, group := range CensusGroups {
		top, sub, _ := strings.Cut(censusTests[group], "/")
		m := matcher{top: regexp.MustCompile(top)}
		if sub != "" {
			m.sub = regexp.MustCompile(sub)
		}
		matchers[group] = m
	}
	groupsOf := func(name string) []string {
		test, subtest, hasSub := strings.Cut(name, "/")
		var groups []string
		for _, group := range CensusGroups {
			m := matchers[group]
			if !m.top.MatchString(test) {
				continue
			}
			if m.sub == nil || (hasSub && m.sub.MatchString(subtest)) {
				groups = append(groups, group)
			}
		}
		return groups
	}
	for _, name := range []string{
		"TestTheScreensAreWithinTheBudgets",
		"TestTheScreensAreWithinTheBudgetsInTheFontsThatTheSystemChooses",
		"TestTheScreensAreWithinTheBudgetsInATextThatIsWider",
		"TestTheScreensAreWithinTheBudgetsInTheWidestFontsThatTheSystemHas",
	} {
		if len(groupsOf(name)) == 0 {
			t.Errorf("no group of the census takes %s", name)
		}
	}
	for _, window := range []string{"narrow", "compact", "medium", "wide"} {
		for _, scale := range []int{115, 130, 150, 175} {
			for _, suffix := range []string{"", "-zh"} {
				name := fmt.Sprintf("TestTheScreensAreWithinTheBudgetsAtEverySizeOfText/home.%s-%d%s", window, scale, suffix)
				if got := groupsOf(name); len(got) != 1 {
					t.Errorf("%s is taken by %v, want exactly one group", name, got)
				}
			}
		}
	}
}
