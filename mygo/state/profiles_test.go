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
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
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
	controller := state.NewSdkDetailController(testClient(runner), "java", linuxProfiles(), nil)
	defer controller.Dispose()
	if _, known := controller.Profile(); !known || controller.Search() != "open" || controller.SearchArguments() != nil {
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
	// The versions of OpenJDK do not stay on show for Temurin, and nothing is
	// searched until somebody asks for the list.
	if controller.Versions.HasValue() || controller.Versions.IsLoading() || len(runner.Calls("search java tem")) != 0 {
		t.Error("the list of the earlier choice goes away, and the next one waits to be asked for")
	}
	controller.Versions.EnsureLoaded()
	if !controller.Versions.IsLoading() {
		t.Error("asking for the list loads it")
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
	controller.Versions.EnsureLoaded()
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
		controller := state.NewSdkDetailController(testClient(runner), c.plugin, c.profiles, nil)
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
	controller := state.NewSdkDetailController(testClient(testsupport.FixtureRunner()), "java", linuxProfiles(), nil)
	defer controller.Dispose()
	if err := controller.SetSearch("tem"); err != nil {
		t.Fatal(err)
	}
	controller.Versions.EnsureLoaded()
	waitFor(t, "the list of Temurin", func() bool { return controller.Versions.HasValue() })
	groups := controller.Groups()
	if len(groups) != 1 || groups[0].ID != "support" || groups[0].Chips[0].Value != "LTS" || groups[0].Chips[0].Count == 0 {
		t.Fatalf("Temurin has releases with long-term support and no JavaFX build: %+v", groups)
	}
	if again := controller.Groups(); &again[0] != &groups[0] {
		t.Error("the groups of a list are worked out once")
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
	if err := controller.SetSearch("zulu"); err != nil {
		t.Fatal(err)
	}
	if !controller.Selection().Empty() {
		t.Error("a new search forgets the facets")
	}
}

func TestSdkDetailOfAPluginWithoutAProfileLeavesTheListAlone(t *testing.T) {
	controller := state.NewSdkDetailController(testClient(testsupport.FixtureRunner()), "maven", nil, nil)
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
	nodejs, ok := linuxProfiles().Lookup("nodejs")
	if !ok {
		t.Fatal("nodejs has a profile")
	}
	const mirror = "https://npmmirror.com/mirrors/node"
	if err := model.SetPluginVariable(nodejs, "VFOX_NODEJS_MIRROR", " "+mirror+" "); err != nil {
		t.Fatal(err)
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != mirror {
		t.Errorf("the value is stored without its spaces: %q", got)
	}
	waitFor(t, "vfox to run with the mirror", func() bool {
		return model.Cli.LaunchConfig().ExtraEnvironment["VFOX_NODEJS_MIRROR"] == mirror
	})

	var invalid *profile.ValueError
	if err := model.SetPluginVariable(nodejs, "VFOX_NODEJS_MIRROR", "npmmirror.com"); !errors.As(err, &invalid) || invalid.Reason != profile.ReasonAddress {
		t.Errorf("an address needs its scheme: %v", err)
	}
	if err := model.SetPluginVariable(nodejs, "VFOX_NODEJS_LIBC", "gnu"); !errors.As(err, &invalid) || invalid.Reason != profile.ReasonChoice {
		t.Errorf("a choice takes its values only: %v", err)
	}
	if err := model.SetPluginVariable(nodejs, "PATH", "/x"); err == nil {
		t.Error("the plugin does not read PATH")
	}
	if err := model.SetPluginVariable(profile.Profile{Plugin: "maven"}, "VFOX_NODEJS_MIRROR", mirror); err == nil {
		t.Error("maven does not read the mirror of Node.js")
	}
	if got := model.PluginVariable("VFOX_NODEJS_MIRROR"); got != mirror {
		t.Errorf("a refused value leaves the old one: %q", got)
	}

	if err := model.SetPluginVariable(nodejs, "VFOX_NODEJS_MIRROR", ""); err != nil {
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

// memory is a SearchMemory that keeps its choices in a map.
type memory map[string]string

func (m memory) RememberedSearch(plugin string) string { return m[plugin] }

func (m memory) RememberSearch(plugin, value string) {
	if value == "" {
		delete(m, plugin)
		return
	}
	m[plugin] = value
}

func TestSdkDetailStartsAtTheChoiceThatWasRemembered(t *testing.T) {
	runner := testsupport.FixtureRunner()
	remembered := memory{"java": "tem", "nodejs": "tem"}
	controller := state.NewSdkDetailController(testClient(runner), "java", linuxProfiles(), remembered)
	defer controller.Dispose()
	if controller.Search() != "tem" || !reflect.DeepEqual(controller.SearchArguments(), []string{"tem"}) {
		t.Fatalf("the remembered choice is the search: %q %v", controller.Search(), controller.SearchArguments())
	}
	<-controller.Versions.Load()
	if calls := runner.Calls("search"); len(calls) != 1 || !reflect.DeepEqual(calls[0].Arguments, []string{"search", "java", "tem"}) {
		t.Errorf("the first list is the one of the remembered choice: %v", calls)
	}

	// A choice of the user is kept; a search that is only set is not.
	if err := controller.ChooseSearch("zulu"); err != nil || remembered["java"] != "zulu" {
		t.Errorf("a chosen distribution is remembered: %v %v", err, remembered)
	}
	if err := controller.SetSearch("tem"); err != nil || remembered["java"] != "zulu" {
		t.Errorf("setting a search is no choice: %v %v", err, remembered)
	}
	if err := controller.ChooseSearch("nope"); err == nil || remembered["java"] != "zulu" {
		t.Errorf("a refused choice is not remembered: %v %v", err, remembered)
	}
	// The default is no entry, so that it follows the plugin when the default changes.
	if err := controller.ChooseSearch("open"); err != nil {
		t.Fatal(err)
	}
	if _, kept := remembered["java"]; kept {
		t.Errorf("the default is forgotten, not kept: %v", remembered)
	}

	// What does not fit is ignored: a distribution that does not exist, the
	// default, and a plugin whose search takes no argument.
	for _, c := range []struct {
		label, plugin, value string
	}{
		{"an unknown distribution", "java", "nope"},
		{"the default", "java", "open"},
		{"a plugin without a choice", "nodejs", "tem"},
	} {
		other := state.NewSdkDetailController(testClient(runner), c.plugin, linuxProfiles(), memory{c.plugin: c.value})
		if got := other.SearchArguments(); got != nil {
			t.Errorf("%s: no argument is passed, got %v", c.label, got)
		}
		other.Dispose()
	}
	if got := state.NewSdkDetailController(testClient(runner), "java", linuxProfiles(), memory{"java": "nope"}).Search(); got != "open" {
		t.Errorf("an unknown distribution leaves the default: %q", got)
	}
}

func TestTheChoiceOfASearchIsKeptInTheSettings(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.DefaultSettings())
	open := func() *state.AppModel {
		model := state.NewAppModel(state.AppModelOptions{
			Runner: testsupport.FixtureRunner(), SettingsStore: store, Profiles: linuxProfiles(),
			Environment: map[string]string{"HOME": "/home/u"},
			Locator:     vfox.NewVfoxLocator(map[string]string{}, func(path string) bool { return path == "/usr/bin/vfox" }),
		})
		model.Start()
		t.Cleanup(model.Dispose)
		return model
	}
	model := open()
	detail := model.SdkDetail("java")
	if err := detail.ChooseSearch("tem"); err != nil {
		t.Fatal(err)
	}
	detail.Dispose()
	if got := store.Load().Searches; !reflect.DeepEqual(got, map[string]string{"java": "tem"}) {
		t.Fatalf("the choice is in the file: %v", got)
	}

	// The next details of the plugin, and the next run of the application, start at it.
	for _, model := range []*state.AppModel{model, open()} {
		again := model.SdkDetail("java")
		if again.Search() != "tem" {
			t.Errorf("the details start at the choice: %q", again.Search())
		}
		again.Dispose()
	}

	detail = model.SdkDetail("java")
	defer detail.Dispose()
	if err := detail.ChooseSearch("open"); err != nil {
		t.Fatal(err)
	}
	if got := store.Load().Searches; len(got) != 0 {
		t.Errorf("the default leaves no entry: %v", got)
	}
}

// aliasRunner answers for a plugin that was added under the alias, which says it
// is the plugin that declared.
func aliasRunner(alias, declared string) *testsupport.FakeProcessRunner {
	info := declared + "\x1f1.2.3\x1fhttps://example.com/" + declared + "\x1f/home/u/.vfox/cache/" + alias + "\x1fA plugin"
	return testsupport.FixtureRunner().
		On("info "+alias+" --format "+vfox.PluginInfoTemplate, core.ProcessOutput{Stdout: info}).
		On("search "+alias, testsupport.FixtureOutput("search-java", 0)).
		On("search "+alias+" tem", testsupport.FixtureOutput("search-java-tem", 0))
}

func TestSdkDetailOfAnAliasHasTheProfileOfThePluginItWasMadeFrom(t *testing.T) {
	runner := aliasRunner("coffee", "java")
	remembered := memory{"coffee": "tem"}
	controller := state.NewSdkDetailController(testClient(runner), "coffee", linuxProfiles(), remembered)
	defer controller.Dispose()
	if _, known := controller.Profile(); known || controller.Search() != "" || controller.SetSearch("tem") == nil {
		t.Fatal("the name alone says nothing about the plugin")
	}

	// The list that is loaded before the plugin says what it is answers no search.
	<-controller.Versions.Load()
	if versions, _ := controller.Versions.Value(); len(versions) != 75 {
		t.Fatalf("the plain list has %d versions", len(versions))
	}
	notified := make(chan struct{}, 8)
	stop := controller.AddListener(func() { notified <- struct{}{} })
	defer stop()

	<-controller.Info.Load()
	known, ok := controller.Profile()
	if !ok || known.Plugin != "java" || controller.Name != "coffee" {
		t.Fatalf("the plugin declares java: %v %v", known.Plugin, ok)
	}
	if controller.Search() != "tem" || !reflect.DeepEqual(controller.SearchArguments(), []string{"tem"}) {
		t.Errorf("the choice that was remembered applies: %q %v", controller.Search(), controller.SearchArguments())
	}
	if controller.Versions.HasValue() {
		t.Error("the list of the default is not the list of the choice")
	}
	select {
	case <-notified:
	default:
		t.Error("whoever shows the details is told that it has a profile now")
	}
	controller.Versions.EnsureLoaded()
	waitFor(t, "the list of Temurin", func() bool { return controller.Versions.HasValue() })
	if calls := runner.Calls("search coffee tem"); len(calls) != 1 {
		t.Errorf("the plugin is searched by its own name: %v", runner.Invocations())
	}

	// A choice of the user is kept for the name of the plugin.
	if err := controller.ChooseSearch("zulu"); err != nil || remembered["coffee"] != "zulu" {
		t.Errorf("the choice is kept under the alias: %v %v", err, remembered)
	}
}

func TestSdkDetailOfAPluginThatDeclaresAnotherNameKeepsWhatItHas(t *testing.T) {
	for _, c := range []struct {
		label, alias, declared string
		wantKnown              bool
	}{
		{"a plugin that has no profile", "mine", "maven", false},
		{"a profile of the name is kept", "java", "something-else", true},
	} {
		controller := state.NewSdkDetailController(testClient(aliasRunner(c.alias, c.declared)), c.alias, linuxProfiles(), nil)
		<-controller.Info.Load()
		if _, known := controller.Profile(); known != c.wantKnown {
			t.Errorf("%s: known = %v", c.label, known)
		}
		controller.Dispose()
	}

	// Metadata that cannot be had changes nothing.
	runner := testsupport.FixtureRunner().On("info coffee --format "+vfox.PluginInfoTemplate, testsupport.FixtureOutput("info-unknown", 1))
	controller := state.NewSdkDetailController(testClient(runner), "coffee", linuxProfiles(), nil)
	defer controller.Dispose()
	<-controller.Info.Load()
	if _, known := controller.Profile(); known || controller.Info.Error() == nil {
		t.Error("a plugin that cannot be described has no profile")
	}
}
