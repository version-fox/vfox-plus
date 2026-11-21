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

package screens

import (
	"reflect"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// Named white space and format characters, written as runes so the source holds
// only ASCII escapes: a byte order mark is illegal anywhere but the start of a file.
var (
	availNBSP        = string(rune(0x00A0))
	availIdeographic = string(rune(0x3000))
	availBOM         = string(rune(0xFEFF))
	availNEL         = string(rune(0x0085))
)

// TestAVersionNameFollowsItsMessage covers the check of a version name, which the message
// of the web says: a name has no spaces and does not start with a dash. So white space is
// refused anywhere in a name, and so is a name that starts with an @. The white space is
// JavaScript's, so U+0085, which Go counts, is an ordinary character.
func TestAVersionNameFollowsItsMessage(t *testing.T) {
	for _, name := range []string{"1.22.0", "go", "21.0.12.1+1-tem", "a-b", "x@y", availNEL + "go"} {
		if !availValidSpec(name) {
			t.Errorf("availValidSpec(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"", "-x", "--", "@go", " go", "\tgo", "go ", "17 system", "17\tsystem", "1.22" + availNBSP + "x", availIdeographic + "go", availBOM + "go"} {
		if availValidSpec(name) {
			t.Errorf("availValidSpec(%q) = true, want false", name)
		}
	}
}

// TestTrimRemovesWhiteSpaceAsJavaScriptDoes covers the trim of the web's inputs.
func TestTrimRemovesWhiteSpaceAsJavaScriptDoes(t *testing.T) {
	cases := map[string]string{
		"  go  ": "go",
		availNBSP + availIdeographic + "go" + availBOM: "go",
		"\n\t1.22.0\r\n":           "1.22.0",
		availNEL + "go" + availNEL: availNEL + "go" + availNEL,
		"17 system":                "17 system",
		"":                         "",
	}
	for in, want := range cases {
		if got := availTrim(in); got != want {
			t.Errorf("availTrim(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestAShorthandIsFoundByItsName covers the words that the install by name offers.
func TestAShorthandIsFoundByItsName(t *testing.T) {
	shorthands := []remote.Shorthand{
		{Name: "17", Meaning: remote.Text{EN: "Java 17."}},
		{Name: "lts", Meaning: remote.Text{EN: "The long-term support line."}},
	}
	if found, ok := availShorthand(shorthands, "lts"); !ok || found.Meaning.EN != "The long-term support line." {
		t.Errorf("availShorthand(lts) = %+v, %v", found, ok)
	}
	if _, ok := availShorthand(shorthands, "LTS"); ok {
		t.Error("a shorthand is matched without regard to case; the web compares the names exactly")
	}
	if _, ok := availShorthand(nil, "17"); ok {
		t.Error("a shorthand is found in no shorthands")
	}
}

// TestTheChoiceOfTheSearchIsTheCurrentOneOrTheDefault covers which search value the
// plugin lists, which names the versions that are read.
func TestTheChoiceOfTheSearchIsTheCurrentOneOrTheDefault(t *testing.T) {
	if got := availSearchValue(nil); got != "" {
		t.Errorf("no profile: value %q, want none", got)
	}
	if got := availSearchValue(&remote.Profile{}); got != "" {
		t.Errorf("no choice of search: value %q, want none", got)
	}
	profile := &remote.Profile{Search: &remote.ProfileSearch{Default: "all", Current: "lts"}}
	if got := availSearchValue(profile); got != "lts" {
		t.Errorf("current choice: value %q, want lts", got)
	}
	profile.Search.Current = ""
	if got := availSearchValue(profile); got != "all" {
		t.Errorf("no current choice: value %q, want the default all", got)
	}
}

// TestAChoiceMatchesItsValueAliasesNamesAndMakers covers the web's chooser search.
func TestAChoiceMatchesItsValueAliasesNamesAndMakers(t *testing.T) {
	choice := remote.Choice{
		Value:   "temurin-17",
		Name:    remote.Text{EN: "Temurin", ZH: "测试名"},
		Maker:   remote.Text{EN: "Eclipse Adoptium", ZH: "开源"},
		Aliases: []string{"tem"},
	}
	for _, q := range []string{"", "temurin", "eclipse", "tem", "17", "测试名", "开源", "adopt"} {
		if !availChoiceMatches(choice, q) {
			t.Errorf("a query %q does not match the choice", q)
		}
	}
	for _, q := range []string{"zulu", "lts"} {
		if availChoiceMatches(choice, q) {
			t.Errorf("a query %q matches the choice", q)
		}
	}
}

// TestTheFiltersAreAlternativesWithinAGroupAndRequiredAcrossGroups covers the
// versions' filters: a version has one of the values chosen in each group that has
// some, and the query matches the version or its note.
func TestTheFiltersAreAlternativesWithinAGroupAndRequiredAcrossGroups(t *testing.T) {
	versions := []remote.AvailableVersion{
		{Version: "1.22.0", Note: "stable", Facets: map[string][]string{"arch": {"amd64"}, "os": {"linux"}}},
		{Version: "1.23.0", Note: "beta", Facets: map[string][]string{"arch": {"arm64"}, "os": {"linux"}}},
		{Version: "1.24.0", Note: "stable", Facets: map[string][]string{"arch": {"amd64", "arm64"}, "os": {"windows"}}},
	}
	versionsOf := func(q string, chosen availChosen) []string {
		var out []string
		for _, v := range availRows(versions, q, chosen) {
			out = append(out, v.Version)
		}
		return out
	}
	if got := versionsOf("", availChosen{}); !reflect.DeepEqual(got, []string{"1.22.0", "1.23.0", "1.24.0"}) {
		t.Errorf("no filter: %v", got)
	}
	if got := versionsOf("stable", availChosen{}); !reflect.DeepEqual(got, []string{"1.22.0", "1.24.0"}) {
		t.Errorf("the note is searched: %v", got)
	}
	if got := versionsOf("1.2", availChosen{}); len(got) != 3 {
		t.Errorf("the version is searched: %v", got)
	}
	chosen := availChosen{}
	availToggle(chosen, "arch", "amd64")
	availToggle(chosen, "arch", "arm64")
	if got := versionsOf("", chosen); !reflect.DeepEqual(got, []string{"1.22.0", "1.23.0", "1.24.0"}) {
		t.Errorf("two values of one group are alternatives, so each version of them passes: %v", got)
	}
	availToggle(chosen, "os", "linux")
	if got := versionsOf("", chosen); !reflect.DeepEqual(got, []string{"1.22.0", "1.23.0"}) {
		t.Errorf("the groups are required together: %v", got)
	}
	if got := versionsOf("beta", chosen); !reflect.DeepEqual(got, []string{"1.23.0"}) {
		t.Errorf("the search and the filters together: %v", got)
	}
	availToggle(chosen, "os", "linux")
	availToggle(chosen, "arch", "amd64")
	availToggle(chosen, "arch", "arm64")
	if len(chosen) != 0 {
		t.Errorf("toggling every value off leaves %v", chosen)
	}
}

// TestTheFacetGroupsKeepOnlyTheValuesThatSomeVersionHas covers the chips of the
// filters: a value that no version has is not offered.
func TestTheFacetGroupsKeepOnlyTheValuesThatSomeVersionHas(t *testing.T) {
	profile := &remote.Profile{Facets: []remote.Facet{
		{ID: "arch", Label: remote.Text{EN: "Architecture"}, Values: []string{"amd64", "arm64", "386"}},
		{ID: "lib", Label: remote.Text{EN: "Library"}, Values: []string{"glibc"}},
	}}
	versions := []remote.AvailableVersion{
		{Version: "1", Facets: map[string][]string{"arch": {"arm64", "amd64"}}},
	}
	want := []availFacet{{ID: "arch", Label: remote.Text{EN: "Architecture"}, Values: []string{"amd64", "arm64"}}}
	if got := availFacets(profile, versions); !reflect.DeepEqual(got, want) {
		t.Errorf("availFacets = %+v, want %+v", got, want)
	}
	if got := availFacets(nil, versions); got != nil {
		t.Errorf("no profile: %+v", got)
	}
}

// TestTheInstalledVersionsAreTheOnesOfThePlugin covers the marks of the installed
// versions, which come from the inventory.
func TestTheInstalledVersionsAreTheOnesOfThePlugin(t *testing.T) {
	inventory := remote.Inventory{Installed: remote.Section[[]remote.InstalledSdk]{Value: []remote.InstalledSdk{
		{Name: "go", Versions: []string{"1.22.0", "1.23.0"}},
		{Name: "java", Versions: []string{"17"}},
	}}}
	want := map[string]bool{"1.22.0": true, "1.23.0": true}
	if got := availInstalled(inventory, "go"); !reflect.DeepEqual(got, want) {
		t.Errorf("availInstalled(go) = %v, want %v", got, want)
	}
	if got := availInstalled(remote.Inventory{}, "go"); len(got) != 0 {
		t.Errorf("an inventory that has not loaded marks %v", got)
	}
}

// TestAdditionsAndSourcesAreShownAsTheWebShowsThem covers the text of a version row.
func TestAdditionsAndSourcesAreShownAsTheWebShowsThem(t *testing.T) {
	additions := []remote.VersionAddition{{Name: "npm", Version: "10.2"}, {Name: "yarn"}}
	if got := availAdditions(additions); got != "npm 10.2, yarn" {
		t.Errorf("availAdditions = %q", got)
	}
	if got := availAdditions(nil); got != "" {
		t.Errorf("no additions: %q", got)
	}
	version := remote.AvailableVersion{Parts: &remote.VersionParts{Source: &remote.Choice{Name: remote.Text{EN: "Temurin"}}}}
	if got := availSourceName(version); got != "Temurin" {
		t.Errorf("availSourceName = %q", got)
	}
	if got := availSourceName(remote.AvailableVersion{}); got != "" {
		t.Errorf("a version with no parts names %q", got)
	}
}

// TestTheCountsAndTheLimitsAreTheWebs covers the texts of the list and its steps.
func TestTheCountsAndTheLimitsAreTheWebs(t *testing.T) {
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	if availPageSize != 200 || availPageStep != 400 {
		t.Errorf("the list shows %d and adds %d; the web shows 200 and adds 400", availPageSize, availPageStep)
	}
	if got := availCount("nVersions", 3); got != "3 versions" {
		t.Errorf("count = %q", got)
	}
	if got := availCount("showMore", 800); got != "Show 800 more" {
		t.Errorf("show more = %q", got)
	}
}
