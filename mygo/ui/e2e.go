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

//go:build e2e

package ui

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// e2eHooks is what the end-to-end tests add to the application: the count of the
// builds of the view, which the scenario waits on, so that it acts on what the
// user sees.
type e2eHooks struct {
	builds atomic.Uint64
}

// startE2E runs the scenario that VFOX_PLUS_E2E_SCENARIO names, if there is one,
// as vfxtool e2e-run, measure and screenshots expect: every step is reported
// as `e2e: …` on the standard output, and the
// application ends with the exit code 0 when the scenario passed and 1 when it
// failed.
func (a *App) startE2E() {
	name := a.environment["VFOX_PLUS_E2E_SCENARIO"]
	if name == "" {
		return
	}
	a.env.Reg.CollectText = true
	a.model.Profiles.Register(testsdkProfile())
	go a.runScenario(name)
}

// testsdkProfile is the profile of the offline test plugin, which takes an argument
// like vfox-java does: `lts` lists the releases with long-term support only.
func testsdkProfile() profile.Profile {
	return profile.Profile{
		Plugin: "testsdk",
		Search: &profile.Search{
			Label: profile.Text{EN: "Channel"}, Default: "all",
			Choices: []profile.Choice{
				{Value: "all", Name: profile.Text{EN: "Every version"}, Popular: true},
				{Value: "lts", Name: profile.Text{EN: "Long-term support"}, Popular: true},
			},
		},
		Shorthands: []profile.Shorthand{{Name: "1.0.0", Meaning: profile.Text{EN: "The first release"}}},
		Facets:     []profile.Facet{{ID: "support", Label: profile.Text{EN: "Support"}, Kind: profile.FacetNote, Values: []string{"LTS"}}},
	}
}

// afterFrame counts the build of the view that has just ended, when the registry
// knows every widget of the frame.
func (a *App) afterFrame(kit.Ctx) { a.e2e.builds.Add(1) }

// report writes a line for the harness.
func report(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stdout, "e2e: "+format+"\n", args...)
}

func (a *App) runScenario(name string) {
	run, ok := scenarios[name]
	if !ok {
		names := make([]string, 0, len(scenarios))
		for known := range scenarios {
			names = append(names, known)
		}
		report("unknown scenario %q; use one of %s", name, strings.Join(names, ", "))
		os.Exit(2)
	}
	started := time.Now()
	report("scenario %s started", name)
	current := newScenario(a)
	if err := protect(func() { run(current) }); err != nil {
		report("scenario %s FAILED: %v", name, err)
		// What the user would have seen, to find out why.
		_ = protect(func() { current.dump() })
		os.Exit(1)
	}
	report("scenario %s passed in %d s", name, int(time.Since(started).Seconds()+0.5))
	os.Exit(0)
}

// protect turns the panic of a failed check into an error.
func protect(f func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%v", recovered)
		}
	}()
	f()
	return nil
}
