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
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// availGoProfile is the profile of a plugin that has a search, filters, names to try and
// a guide: the profile of Java or Go in the desktop.
func availGoProfile() *remote.Profile {
	return &remote.Profile{
		Plugin: "go",
		Search: &remote.ProfileSearch{
			Label:   remote.Text{EN: "Distribution"},
			Default: "all",
			Current: "all",
			Choices: []remote.Choice{
				{Value: "all", Name: remote.Text{EN: "All distributions"}, Maker: remote.Text{EN: "Everyone"}, Everything: true},
				{Value: "lts", Name: remote.Text{EN: "LTS only"}, Maker: remote.Text{EN: "Vendors"}, Aliases: []string{"long"}},
			},
		},
		Facets: []remote.Facet{
			{ID: "arch", Label: remote.Text{EN: "Architecture"}, Values: []string{"amd64", "arm64", "386"}},
		},
		Shorthands: []remote.Shorthand{{Name: "1.22", Meaning: remote.Text{EN: "The 1.22 line."}}},
		Guide:      remote.Text{EN: "Type a full version."},
	}
}

// availGoVersions are three versions: one installed, one with a note and additions, and
// one that comes from the official source.
func availGoVersions() []remote.AvailableVersion {
	return []remote.AvailableVersion{
		{
			Version: "1.22.0",
			Parts:   &remote.VersionParts{Version: "1.22.0", Source: &remote.Choice{Value: "official", Name: remote.Text{EN: "Official"}}},
			Facets:  map[string][]string{"arch": {"amd64"}},
		},
		{
			Version:   "1.23.0",
			Note:      "stable",
			HasNote:   true,
			Additions: []remote.VersionAddition{{Name: "npm", Version: "10.2"}},
			Installed: true,
			Facets:    map[string][]string{"arch": {"arm64"}},
		},
		{
			Version: "1.24.0-rc",
			Facets:  map[string][]string{"arch": {"amd64", "arm64"}},
		},
	}
}

// availGoInventory is an inventory in which the plugin go has 1.23.0 installed.
func availGoInventory() remote.Inventory {
	return remote.Inventory{
		Installed: remote.Section[[]remote.InstalledSdk]{
			Value: []remote.InstalledSdk{{Name: "go", Versions: []string{"1.23.0"}, Current: "1.23.0", HasCurrent: true}},
		},
	}
}

// availOpenAvailable opens the SDK page of the plugin go and chooses its tab of the
// versions that can be installed, as a person does.
func availOpenAvailable(t *testing.T, root *Root, tt *ui.Tester) {
	t.Helper()
	if !root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"}) {
		t.Fatal("the page of go was refused")
	}
	settle(tt)
	if err := tt.Click("Available"); err != nil {
		t.Fatalf("Click(Available): %v", err)
	}
	settle(tt)
}

// availOpenVersions opens the versions that can be installed of the plugin go, and
// waits for the list.
func availOpenVersions(t *testing.T, root *Root, tt *ui.Tester, count string) {
	t.Helper()
	availOpenAvailable(t, root, tt)
	availWaitText(t, tt, count)
}

// availWaitTab draws frames until the window shows the tab, with no page above it.
func availWaitTab(t *testing.T, tt *ui.Tester, root *Root, tab app.Tab) {
	t.Helper()
	for i := 0; i < 500 && (root.App().Tab() != tab || len(root.App().Pages()) != 0); i++ {
		tt.Frame()
	}
	if root.App().Tab() != tab || len(root.App().Pages()) != 0 {
		t.Fatalf("the window shows tab %v with pages %v; want tab %v with none", root.App().Tab(), root.App().Pages(), tab)
	}
}

func TestTheAvailableTabListsTheVersionsAndMarksTheInstalled(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")
	for _, want := range []string{"1.22.0", "1.23.0", "1.24.0-rc", "stable", "npm 10.2", "Installed", "Official"} {
		if !hasText(tt, want) {
			t.Errorf("the page has no %q; the frame shows %q", want, tt.Texts())
		}
	}
	if !hasText(tt, "All distributions") {
		t.Errorf("the page does not show the choice in use; the frame shows %q", tt.Texts())
	}
	availShot(t, tt, "available-tab")
}

func TestTheVersionsPageShowsItsLoadingUntilTheComputerAnswers(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	release := f.holdReads(t)
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenAvailable(t, root, tt)
	if !hasText(tt, "Loading…") {
		t.Errorf("the page does not show its loading; the frame shows %q", tt.Texts())
	}
	availShot(t, tt, "available-loading")
	release()
	availWaitText(t, tt, "3 versions")
}

func TestTheSearchFiltersTheVersionsByVersionAndNote(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if err := tt.Click("Search versions…"); err != nil {
		t.Fatalf("Click(search): %v", err)
	}
	tt.Type("stable")
	availWaitText(t, tt, "1 versions")
	if hasText(tt, "1.22.0") {
		t.Errorf("a version that does not match is shown: %q", tt.Texts())
	}
	if !hasText(tt, "1.23.0") {
		t.Errorf("the version with the note is not shown: %q", tt.Texts())
	}
}

func TestTheFiltersOfTheVersionsAreTheValuesThatSomeVersionHas(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if hasText(tt, "386") {
		t.Errorf("a filter that no version has is offered: %q", tt.Texts())
	}
	if err := tt.Click("arm64"); err != nil {
		t.Fatalf("Click(arm64): %v", err)
	}
	availWaitText(t, tt, "2 versions")
	if hasText(tt, "1.22.0") || !hasText(tt, "1.23.0") || !hasText(tt, "1.24.0-rc") {
		t.Errorf("the filter arm64 shows %q", tt.Texts())
	}
}

func TestTheSearchSourceIsChosenOnTheComputer(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if err := tt.Click("All distributions"); err != nil {
		t.Fatalf("Click(the choice in use): %v", err)
	}
	availWaitText(t, tt, "Distribution")
	availShot(t, tt, "choice-dialog")
	if err := tt.Click("LTS only"); err != nil {
		t.Fatalf("Click(LTS only): %v", err)
	}
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0]; got.Method != "PUT" || got.Path != "/api/sdks/go/search" || got.Body != `{"value":"lts"}` {
		t.Errorf("the choice sent %+v", got)
	}
	// The list is read again for the new choice, and the filter of the old one is cleared.
	availWaitText(t, tt, "LTS only")
	availWaitText(t, tt, "3 versions")
}

func TestAChangedChoiceClearsTheFiltersOfTheOldOne(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")
	if err := tt.Click("arm64"); err != nil {
		t.Fatalf("Click(arm64): %v", err)
	}
	availWaitText(t, tt, "2 versions")

	if err := tt.Click("All distributions"); err != nil {
		t.Fatalf("Click(the choice in use): %v", err)
	}
	availWaitText(t, tt, "Distribution")
	if err := tt.Click("LTS only"); err != nil {
		t.Fatalf("Click(LTS only): %v", err)
	}
	availWaitText(t, tt, "3 versions")
	if hasText(tt, "2 versions") {
		t.Error("the filter of the old choice is still on")
	}
}

func TestInstallingAVersionStartsATaskAndShowsTheTasks(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if err := tt.Click("Install"); err != nil {
		t.Fatalf("Click(Install): %v", err)
	}
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0]; got.Method != "POST" || got.Path != "/api/tasks" || got.Body != `{"operation":"install","sdk":"go","version":"1.22.0"}` {
		t.Errorf("the install sent %+v", got)
	}
	availWaitTab(t, tt, root, app.TabTasks)
}

// availOpenInstallMenu opens the menu of the install button of the first version that
// can be installed. The menu button is labelled "Actions", as the SDK's own menu is, and
// the SDK's comes first, so the version's button is found from the label of its lead:
// the menu button follows the lead's right padding and a gap, and is 44 dp wide.
func availOpenInstallMenu(t *testing.T, tt *ui.Tester) {
	t.Helper()
	label, ok := tt.Find("Install")
	if !ok {
		t.Fatal("no install button is shown")
	}
	const leadPadding, gap, menuWidth = 12, 2, 44
	tt.ClickAt(label.X+label.W+leadPadding+gap+menuWidth/2, label.Y+label.H/2)
	tt.Frame()
}

func TestInstallAndUseSendsUseAndShowsTheTasks(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	availOpenInstallMenu(t, tt)
	if err := tt.Click("Install and use"); err != nil {
		t.Fatalf("Click(%s): %v", "Install and use", err)
	}
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0].Body; got != `{"operation":"install","sdk":"go","version":"1.22.0","use":true}` {
		t.Errorf("install and use sent %s", got)
	}
	availWaitTab(t, tt, root, app.TabTasks)
}

func TestInstallByNameChecksTheNameBeforeItIsSent(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if err := tt.Click("More ways to add"); err != nil {
		t.Fatalf("Click(More ways to add): %v", err)
	}
	tt.Frame()
	availShot(t, tt, "more-ways-menu")
	if err := tt.Click("Install by name…"); err != nil {
		t.Fatalf("Click(Install by name…): %v", err)
	}
	availWaitText(t, tt, "Install go by name")
	if !hasText(tt, "Type a full version.") {
		t.Errorf("the dialog does not show the guide of the plugin: %q", tt.Texts())
	}
	tt.Type("-x")
	tt.Key(0, ui.KeyEnter)
	availWaitText(t, tt, "A version name has no spaces and does not start with a dash.")
	availShot(t, tt, "install-by-name-error")
	if calls := f.recorded(); len(calls) != 0 {
		t.Fatalf("a refused name was sent: %+v", calls)
	}

	for i := 0; i < 2; i++ {
		tt.Key(0, ui.KeyBackspace)
	}
	tt.Type("1.22.0")
	tt.Key(0, ui.KeyEnter)
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0].Body; got != `{"operation":"install","sdk":"go","version":"1.22.0"}` {
		t.Errorf("the name sent %s", got)
	}
}

func TestTheNamesToTryFillTheFieldAndSayWhatTheyMean(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if err := tt.Click("More ways to add"); err != nil {
		t.Fatalf("Click(More ways to add): %v", err)
	}
	tt.Frame()
	if err := tt.Click("Install by name…"); err != nil {
		t.Fatalf("Click(%s): %v", "Install by name…", err)
	}
	availWaitText(t, tt, "Names to try")
	if err := tt.Click("1.22"); err != nil {
		t.Fatalf("Click(the names to try, 1.22): %v", err)
	}
	availWaitText(t, tt, "The 1.22 line.")
	availShot(t, tt, "install-by-name-names")
}

func TestLinkingAnExistingSDKSendsItsFolder(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if err := tt.Click("More ways to add"); err != nil {
		t.Fatalf("Click(More ways to add): %v", err)
	}
	tt.Frame()
	if err := tt.Click("Link existing…"); err != nil {
		t.Fatalf("Click(%s): %v", "Link existing…", err)
	}
	availWaitText(t, tt, "Link existing SDK")
	availShot(t, tt, "link-dialog")
	if !hasText(tt, "The folder is on the computer, not on this phone.") {
		t.Errorf("the link dialog does not say where the folder is: %q", tt.Texts())
	}
	tt.Type("17-system")
	if err := tt.Click("SDK directory"); err != nil {
		t.Fatalf("Click(SDK directory): %v", err)
	}
	tt.Type("/opt/go")
	tt.Key(0, ui.KeyEnter)
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0].Body; got != `{"operation":"link","sdk":"go","version":"17-system","path":"/opt/go"}` {
		t.Errorf("the link sent %s", got)
	}
}

func TestDetectingAndLinkingIsOneCall(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")

	if err := tt.Click("More ways to add"); err != nil {
		t.Fatalf("Click(More ways to add): %v", err)
	}
	tt.Frame()
	if err := tt.Click("Detect and link"); err != nil {
		t.Fatalf("Click(%s): %v", "Detect and link", err)
	}
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0].Body; got != `{"operation":"linkDetect","sdk":"go","yes":true}` {
		t.Errorf("detect and link sent %s", got)
	}
}

func TestAFailedListShowsItsErrorAndRetriesWithoutAnotherFailure(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	f.versionsStatus = 500
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenAvailable(t, root, tt)
	availWaitText(t, tt, "Request failed.")
	if !hasText(tt, "The computer failed to answer the request.") {
		t.Errorf("the failure does not say why: %q", tt.Texts())
	}
	availShot(t, tt, "versions-failed")

	f.mu.Lock()
	f.versionsStatus = 0
	f.mu.Unlock()
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("Click(Retry): %v", err)
	}
	availWaitText(t, tt, "3 versions")
}

func TestAVersionListWithNoVersionsSaysNoneMatch(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "0 versions")
	if !hasText(tt, "No version matches.") {
		t.Errorf("an empty list says nothing: %q", tt.Texts())
	}
	availShot(t, tt, "versions-empty")
}

func TestTheListShowsTwoHundredVersionsAndAddsFourHundredMore(t *testing.T) {
	f := availNewComputer(t)
	f.profiles["go"] = availGoProfile()
	var versions []remote.AvailableVersion
	for i := 0; i < 1000; i++ {
		versions = append(versions, remote.AvailableVersion{Version: fmt.Sprintf("2.%d.0", i)})
	}
	f.versions["go"] = versions
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "1000 versions")
	availShot(t, tt, "versions-long-list")
	if !hasText(tt, "2.199.0") || hasText(tt, "2.200.0") {
		t.Errorf("the first page is not the first 200 versions")
	}
	if !hasText(tt, "Show 800 more") {
		t.Fatalf("the list has no button to show more: %q", tt.Texts())
	}
	// The button is at the end of the list, which the window scrolls to.
	tt.Scroll(240, 500, 0, 40000)
	tt.Frame()
	if err := tt.Click("Show 800 more"); err != nil {
		t.Fatalf("Click(show more): %v", err)
	}
	availWaitText(t, tt, "Show 400 more")
	if !hasText(tt, "2.599.0") {
		t.Errorf("the second page does not show version 599")
	}
}

func TestAPluginWithoutAProfileHasNoChoiceAndNoFilters(t *testing.T) {
	f := availNewComputer(t)
	f.versions["go"] = availGoVersions()
	f.inventory = availGoInventory()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	availOpenVersions(t, root, tt, "3 versions")
	for _, unwanted := range []string{"All distributions", "amd64", "arm64"} {
		if hasText(tt, unwanted) {
			t.Errorf("a plugin with no profile shows %q", unwanted)
		}
	}
	if !hasText(tt, "1.22.0") {
		t.Errorf("the versions of a plugin with no profile are not shown: %q", tt.Texts())
	}
	if strings.Contains(strings.Join(tt.Texts(), "\n"), "Distribution") {
		t.Error("a plugin with no profile shows a search choice")
	}
}
