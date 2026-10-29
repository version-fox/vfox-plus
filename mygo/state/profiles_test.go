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

package state_test

import (
	"errors"
	"reflect"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/state"
)

func linuxProfiles() *profile.Registry {
	return profile.NewRegistry(profile.Platform{OS: "linux", Arch: "amd64"})
}

func TestSdkDetailListsTheDistributionThatWasChosen(t *testing.T) {
	runner := testsupport.FixtureRunner()
	gate := make(chan struct{})
	runner.OnDynamic("search java tem", func() core.ProcessOutput {
		<-gate
		return testsupport.FixtureOutput("search-java-tem", 0)
	})
	controller := state.NewSdkDetailController(testClient(runner), "java", linuxProfiles())
	defer controller.Dispose()
	if !controller.HasProfile || controller.Search() != "open" || controller.SearchArguments() != nil {
		t.Fatalf("a search starts at the default: %q %v", controller.Search(), controller.SearchArguments())
	}
	<-controller.Versions.Load()
	if versions, _ := controller.Versions.Value(); len(versions) != 75 {
		t.Fatalf("OpenJDK lists %d versions", len(versions))
	}
	if call := runner.Calls("search java")[0]; !reflect.DeepEqual(call.Arguments, []string{"search", "java"}) {
		t.Errorf("the default passes no argument: %v", call.Arguments)
	}

	if err := controller.SetSearch("tem"); err != nil {
		t.Fatal(err)
	}
	if controller.Search() != "tem" || !reflect.DeepEqual(controller.SearchArguments(), []string{"tem"}) {
		t.Errorf("search = %q %v", controller.Search(), controller.SearchArguments())
	}
	// The versions of OpenJDK do not stay on show for Temurin.
	if controller.Versions.HasValue() || !controller.Versions.IsLoading() {
		t.Error("the list of the earlier choice goes away while the next one loads")
	}
	close(gate)
	waitFor(t, "the list of Temurin", func() bool { return controller.Versions.HasValue() })
	versions, _ := controller.Versions.Value()
	if len(versions) != 120 || versions[0].Version != "27+35-tem" {
		t.Fatalf("Temurin lists %d versions, first %v", len(versions), versions[0])
	}
	if calls := runner.Calls("search java tem"); len(calls) != 1 {
		t.Errorf("the choice is searched once: %d", len(calls))
	}

	// Choosing the same value again changes nothing; the default passes no argument again.
	before := len(runner.Invocations())
	if err := controller.SetSearch("tem"); err != nil || len(runner.Invocations()) != before {
		t.Errorf("the same choice is no new search: %v, %d calls", err, len(runner.Invocations())-before)
	}
	if err := controller.SetSearch("open"); err != nil || controller.SearchArguments() != nil || controller.Search() != "open" {
		t.Errorf("open is the default: %v %q", err, controller.Search())
	}
	waitFor(t, "the list of OpenJDK", func() bool {
		versions, ok := controller.Versions.Value()
		return ok && len(versions) == 75
	})

	if err := controller.SetSearch("nope"); err == nil {
		t.Error("a distribution that does not exist is refused")
	}
}

func TestSdkDetailOfAPluginWithoutAnArgumentRefusesOne(t *testing.T) {
	runner := testsupport.FixtureRunner()
	for _, c := range []struct {
		label, plugin string
		profiles      *profile.Registry
	}{
		{"nodejs", "nodejs", linuxProfiles()},
		{"maven", "maven", linuxProfiles()},
		{"nodejs without profiles", "nodejs", nil},
	} {
		controller := state.NewSdkDetailController(testClient(runner), c.plugin, c.profiles)
		if controller.Search() != "" || controller.SearchArguments() != nil {
			t.Errorf("%s has no search choice", c.label)
		}
		if err := controller.SetSearch("tem"); err == nil {
			t.Errorf("%s takes no argument", c.label)
		}
		if controller.Groups() != nil {
			t.Errorf("%s: nothing to filter by before a list is loaded", c.label)
		}
		controller.Dispose()
	}
}

func TestSdkDetailFiltersByTheNotesOfTheListedVersions(t *testing.T) {
	controller := state.NewSdkDetailController(testClient(testsupport.FixtureRunner()), "java", linuxProfiles())
	defer controller.Dispose()
	if err := controller.SetSearch("tem"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the list of Temurin", func() bool { return controller.Versions.HasValue() })
	groups := controller.Groups()
	if len(groups) != 1 || groups[0].ID != "support" || groups[0].Chips[0].Value != "LTS" || groups[0].Chips[0].Count == 0 {
		t.Fatalf("Temurin has releases with long-term support and no JavaFX build: %+v", groups)
	}
	all := len(controller.VisibleVersions(nil))
	controller.ToggleFacet("support", "LTS")
	if !controller.Selection().Has("support", "LTS") {
		t.Error("the value is chosen")
	}
	lts := controller.VisibleVersions(nil)
	if len(lts) != groups[0].Chips[0].Count || len(lts) >= all {
		t.Errorf("LTS leaves %d of %d versions, the group counts %d", len(lts), all, groups[0].Chips[0].Count)
	}
	for _, version := range lts {
		if !version.HasNote {
			t.Fatalf("%v passed the filter without a note", version)
		}
	}
	// The text filter and the group narrow each other.
	controller.SetQuery("25.0")
	if narrowed := controller.VisibleVersions(nil); len(narrowed) == 0 || len(narrowed) >= len(lts) {
		t.Errorf("25.0 narrows %d versions to %d", len(lts), len(narrowed))
	}
	controller.SetQuery("")
	controller.ClearFacets()
	if !controller.Selection().Empty() || len(controller.VisibleVersions(nil)) != all {
		t.Error("clearing the facets shows every version again")
	}

	// A new list has its own values: the choice does not carry over.
	controller.ToggleFacet("support", "LTS")
	if err := controller.SetSearch("zulu"); err == nil {
		// zulu has no fixture, so its list fails; the choice is gone all the same.
		waitFor(t, "the search of Zulu to settle", func() bool { return controller.Versions.Error() != nil })
	}
	if !controller.Selection().Empty() {
		t.Error("a new search forgets the facets")
	}
}

func TestSdkDetailOfAPluginWithoutAProfileLeavesTheListAlone(t *testing.T) {
	controller := state.NewSdkDetailController(testClient(testsupport.FixtureRunner()), "maven", nil)
	defer controller.Dispose()
	<-controller.Versions.Load()
	all := len(controller.VisibleVersions(nil))
	controller.ToggleFacet("channel", "stable")
	if len(controller.VisibleVersions(nil)) != all {
		t.Error("without a profile there are no facets to filter by")
	}
	controller.ClearFacets()
	controller.ClearFacets()
}

func TestPluginVariablesJoinTheEnvironmentOfVfox(t *testing.T) {
	runner := testsupport.FixtureRunner()
	model := startedModel(t, runner, "/home/u")
	const mirror = "https://npmmirror.com/mirrors/node"
	if err := model.SetPluginVariable("nodejs", "VFOX_NODEJS_MIRROR", " "+mirror+" "); err != nil {
		t.Fatal(err)
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != mirror {
		t.Errorf("the value is stored without its spaces: %q", got)
	}
	waitFor(t, "vfox to run with the mirror", func() bool {
		return model.Cli.LaunchConfig().ExtraEnvironment["VFOX_NODEJS_MIRROR"] == mirror
	})

	var invalid *profile.ValueError
	if err := model.SetPluginVariable("nodejs", "VFOX_NODEJS_MIRROR", "npmmirror.com"); !errors.As(err, &invalid) || invalid.Reason != profile.ReasonAddress {
		t.Errorf("an address needs its scheme: %v", err)
	}
	if err := model.SetPluginVariable("nodejs", "VFOX_NODEJS_LIBC", "gnu"); !errors.As(err, &invalid) || invalid.Reason != profile.ReasonChoice {
		t.Errorf("a choice takes its values only: %v", err)
	}
	if err := model.SetPluginVariable("nodejs", "PATH", "/x"); err == nil {
		t.Error("the plugin does not read PATH")
	}
	if err := model.SetPluginVariable("maven", "VFOX_NODEJS_MIRROR", mirror); err == nil {
		t.Error("maven has no profile")
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != mirror {
		t.Errorf("a refused value leaves the old one: %q", got)
	}

	if err := model.SetPluginVariable("nodejs", "VFOX_NODEJS_MIRROR", ""); err != nil {
		t.Fatal(err)
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != "" {
		t.Errorf("an empty value takes the variable away: %q", got)
	}
	waitFor(t, "vfox to run without the mirror", func() bool {
		_, set := model.Cli.LaunchConfig().ExtraEnvironment["VFOX_NODEJS_MIRROR"]
		return !set
	})
}
