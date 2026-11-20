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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// sdkDetailFake is the computer of the page of go: two versions, one in use, and a
// profile with variables, version files, an alias, a note and a document; and java,
// with no version installed and no profile.
func sdkDetailFake(t *testing.T) *sdkFake {
	t.Helper()
	fake := newSDKFake(t)
	fake.Inventory = remote.Inventory{
		Vfox: remote.VfoxStatus{Availability: remote.AvailabilityReady, Version: "0.9.0"},
		Installed: remote.Section[[]remote.InstalledSdk]{Value: []remote.InstalledSdk{
			{Name: "go", Versions: []string{"1.21.0", "1.22.0"}, Current: "1.22.0", HasCurrent: true},
			{Name: "java"},
		}},
		Config: remote.Section[[]remote.ConfigEntry]{Value: []remote.ConfigEntry{
			{Key: "legacyVersionFile.enable", Value: "false"},
			{Key: "legacyVersionFile.strategy", Value: "specified"},
		}},
		Complete: true,
	}
	fake.Infos["go"] = remote.PluginInfo{Name: "go", Version: "1.0.0", Homepage: "https://go.dev"}
	fake.Infos["java"] = remote.PluginInfo{Name: "java", Homepage: "file:///etc/passwd"}
	fake.Profiles["go"] = remote.Profile{
		Plugin: "go",
		Naming: remote.NamingPlain,
		Installed: map[string]remote.VersionParts{
			"1.22.0": {
				Version: "1.22.0",
				Source:  &remote.Choice{Value: "official", Name: remote.Text{EN: "Official", ZH: "官方"}},
				Flavors: []string{"amd64"},
			},
		},
		Variables: []remote.Variable{
			{
				Key:     "GOPROXY_MIRROR",
				Label:   remote.Text{EN: "Module mirror", ZH: "模块镜像"},
				Help:    remote.Text{EN: "Where the modules come from."},
				Kind:    remote.VariableURL,
				Example: "https://goproxy.io",
			},
			{
				Key:      "GO_CHANNEL",
				Label:    remote.Text{EN: "Channel"},
				Help:     remote.Text{EN: "Which releases are listed."},
				Kind:     remote.VariableChoice,
				Presets:  []remote.Preset{{Value: "stable", Name: remote.Text{EN: "Stable"}}, {Value: "beta", Name: remote.Text{EN: "Beta"}}},
				Value:    "beta",
				Advanced: true,
			},
		},
		LegacyFiles:    []string{".go-version"},
		LegacyStrategy: true,
		Tips:           []remote.Text{{EN: "Versions come from go.dev."}},
		Docs:           []remote.Link{{Title: remote.Text{EN: "Guide"}, URL: remote.Text{EN: "https://go.dev/doc"}}},
	}
	fake.Aliases["go"] = []remote.VersionAlias{{Sdk: "go", Alias: "latest", Target: "go@1.22.0"}}
	return fake
}

// sdkTaskRequests returns the tasks that the window started, in the order it started them.
func sdkTaskRequests(t *testing.T, fake *sdkFake) []remote.TaskRequest {
	t.Helper()
	var out []remote.TaskRequest
	for _, call := range fake.Requests("POST", "/api/tasks") {
		var request remote.TaskRequest
		if err := json.Unmarshal([]byte(call.Body), &request); err != nil {
			t.Fatalf("the task %q is not a task request: %v", call.Body, err)
		}
		out = append(out, request)
	}
	return out
}

func TestSDKPageShowsTheInstalledVersionsWithTheirOrigin(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	sdkWaitText(t, tt, "Official · amd64 · @latest")
	for _, text := range []string{"Available", "Options", "In use", "1.21.0", "1.22.0", "current", "Use globally"} {
		if !sdkHasText(tt, text) {
			t.Errorf("the page has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	sdkShot(t, tt, "page-installed")
}

func TestSDKPageSetsAVersionAsGlobal(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Use globally")
	if err := tt.Click("Use globally"); err != nil {
		t.Fatalf("Click(Use globally): %v", err)
	}
	sdkWait(t, tt, "the task to use the version is not started", func() bool {
		return len(sdkTaskRequests(t, fake)) == 1
	})
	want := remote.TaskRequest{Operation: remote.RequestUse, Sdk: "go", Version: "1.21.0", Scope: "global"}
	if got := sdkTaskRequests(t, fake)[0]; got != want {
		t.Errorf("the task is %+v, want %+v", got, want)
	}
	sdkWaitText(t, tt, "Queued. See the Tasks tab for its progress.")
}

func TestSDKPageReinstallsAVersionFromItsMenu(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Use globally")
	sdkClickBeside(t, tt, "Use globally", 4+24)
	sdkSettle(tt)
	sdkWaitText(t, tt, "Reinstall")
	if err := tt.Click("Reinstall"); err != nil {
		t.Fatalf("Click(Reinstall): %v", err)
	}
	sdkWait(t, tt, "the task to reinstall the version is not started", func() bool {
		return len(sdkTaskRequests(t, fake)) == 1
	})
	want := remote.TaskRequest{Operation: remote.RequestInstall, Sdk: "go", Version: "1.21.0", Reinstall: true}
	if got := sdkTaskRequests(t, fake)[0]; got != want {
		t.Errorf("the task is %+v, want %+v", got, want)
	}
}

func TestSDKPageUninstallsAVersionOnlyWhenConfirmed(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Use globally")
	sdkClickBeside(t, tt, "Use globally", 4+24)
	sdkSettle(tt)
	sdkWaitText(t, tt, "Uninstall")
	if err := tt.Click("Uninstall"); err != nil {
		t.Fatalf("Click(Uninstall) in the menu: %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Cancel")
	if !sdkHasText(tt, "1.21.0") {
		t.Errorf("the confirmation does not name the version; the frame shows %q", tt.Texts())
	}
	sdkShot(t, tt, "page-uninstall-confirm")
	if err := tt.Click("Cancel"); err != nil {
		t.Fatalf("Click(Cancel): %v", err)
	}
	sdkSettle(tt)
	if got := sdkTaskRequests(t, fake); len(got) != 0 {
		t.Fatalf("a task was started without the confirmation: %+v", got)
	}

	sdkClickBeside(t, tt, "Use globally", 4+24)
	sdkSettle(tt)
	sdkWaitText(t, tt, "Uninstall")
	if err := tt.Click("Uninstall"); err != nil {
		t.Fatalf("Click(Uninstall) in the menu: %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Cancel")
	sdkClickBeside(t, tt, "Cancel", 8+20)
	sdkWait(t, tt, "the confirmed uninstall is not started", func() bool {
		return len(sdkTaskRequests(t, fake)) == 1
	})
	want := remote.TaskRequest{Operation: remote.RequestUninstall, Sdk: "go", Version: "1.21.0"}
	if got := sdkTaskRequests(t, fake)[0]; got != want {
		t.Errorf("the task is %+v, want %+v", got, want)
	}
}

func TestSDKPageNamesAnAliasForTheVersionItWasOpenedFor(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Use globally")
	sdkClickBeside(t, tt, "Use globally", 4+24)
	sdkSettle(tt)
	sdkWaitText(t, tt, "Name alias")
	if err := tt.Click("Name alias"); err != nil {
		t.Fatalf("Click(Name alias) in the menu: %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Version name")
	if err := tt.Click("Alias"); err != nil {
		t.Fatalf("the alias field is not there: %v; the frame shows %q", err, tt.Texts())
	}
	tt.Type("j17")
	sdkSettle(tt)
	sdkShot(t, tt, "page-alias-dialog")
	sdkClickBeside(t, tt, "Cancel", 8+20)
	sdkWait(t, tt, "the alias is not named", func() bool {
		return len(sdkTaskRequests(t, fake)) == 1
	})
	want := remote.TaskRequest{Operation: remote.RequestAliasSet, Sdk: "go", Alias: "j17", Version: "1.21.0"}
	if got := sdkTaskRequests(t, fake)[0]; got != want {
		t.Errorf("the task is %+v, want %+v", got, want)
	}
}

func TestSDKPageNamesAnAliasForTheVersionInUseFromTheOptions(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Name alias")
	if err := tt.Click("Name alias"); err != nil {
		t.Fatalf("Click(Name alias): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Version name")
	if err := tt.Click("Alias"); err != nil {
		t.Fatalf("the alias field is not there: %v", err)
	}
	tt.Type("v2")
	sdkSettle(tt)
	sdkClickBeside(t, tt, "Cancel", 8+20)
	sdkWait(t, tt, "the alias is not named", func() bool {
		return len(sdkTaskRequests(t, fake)) == 1
	})
	want := remote.TaskRequest{Operation: remote.RequestAliasSet, Sdk: "go", Alias: "v2", Version: "1.22.0"}
	if got := sdkTaskRequests(t, fake)[0]; got != want {
		t.Errorf("the task is %+v, want %+v: a new alias names the version in use", got, want)
	}
}

func TestSDKPageShowsLongVersionListsAPageAtATime(t *testing.T) {
	fake := sdkDetailFake(t)
	var versions []string
	for i := range 60 {
		versions = append(versions, fmt.Sprintf("2.%d.0", i))
	}
	fake.Inventory.Installed.Value[0] = remote.InstalledSdk{Name: "go", Versions: versions, Current: "2.59.0", HasCurrent: true}
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "2.0.0")
	if sdkHasText(tt, "2.50.0") || !sdkHasText(tt, "2.49.0") {
		t.Fatalf("the first page is not the first %d versions; the frame shows %q", sdkPageSize, tt.Texts())
	}
	sdkClickScrolled(t, tt, "Show 10 more")
	sdkWaitText(t, tt, "2.50.0")
}

func TestSDKPageRemovesThePluginAfterItAsks(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Actions"); err != nil {
		t.Fatalf("the page has no menu: %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Remove")
	if err := tt.Click("Remove"); err != nil {
		t.Fatalf("Click(Remove) in the menu: %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Cancel")
	sdkClickBeside(t, tt, "Cancel", 8+20)
	sdkWait(t, tt, "the removal is not started", func() bool {
		return len(sdkTaskRequests(t, fake)) == 1
	})
	want := remote.TaskRequest{Operation: remote.RequestRemove, Sdk: "go"}
	if got := sdkTaskRequests(t, fake)[0]; got != want {
		t.Errorf("the task is %+v, want %+v", got, want)
	}
	sdkSettle(tt)
	if top, ok := root.App().Top(); ok {
		t.Errorf("the page %v is still shown after the plugin is removed", top)
	}
}

func TestSDKPageOpensItsHomepageAsALinkOnly(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Homepage")
	if err := tt.Click("Homepage"); err != nil {
		t.Fatalf("Click(Homepage): %v", err)
	}
	sdkSettle(tt)
	if got := tt.OpenedURLs(); len(got) != 1 || got[0] != "https://go.dev" {
		t.Errorf("opened %v, want the homepage of go", got)
	}

	root.App().Pop()
	sdkSettle(tt)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "java"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "No versions installed yet.")
	if err := tt.Click("Homepage"); err != nil {
		t.Fatalf("Click(Homepage) of java: %v", err)
	}
	sdkWaitText(t, tt, "This address is not a web address, so it is not opened.")
	if got := tt.OpenedURLs(); len(got) != 1 {
		t.Errorf("a homepage that is not http or https was opened: %v", got)
	}
}

func TestSDKPageWithNoVersionsOffersTheAvailableTab(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "java"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "No versions installed yet.")
	sdkShot(t, tt, "page-no-versions")
	if err := tt.Click("Browse versions"); err != nil {
		t.Fatalf("Click(Browse versions): %v", err)
	}
	sdkSettle(tt)
	if sdkHasText(tt, "No versions installed yet.") {
		t.Errorf("the installed tab is still shown; the frame shows %q", tt.Texts())
	}
	if sdkHasText(tt, "Options") {
		t.Error("a plugin with no profile has an options tab")
	}
}

func TestSDKPageWaitsForTheInventory(t *testing.T) {
	fake := sdkDetailFake(t)
	open := fake.Hold()
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Loading…")
	sdkShot(t, tt, "page-loading")
	open()
	sdkWaitText(t, tt, "Installed (2)")
}

func TestSDKPageOptionsShowTheVariablesTheFilesAndTheAliases(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Plugin variables")
	for _, text := range []string{"Module mirror", "Default", "Channel", "Beta · beta", "Advanced", "GOPROXY_MIRROR", "Read legacy version files", ".go-version", "Version used from legacy files", "As written", "Latest installed", "Latest available", "Aliases", "latest", "go@1.22.0", "Remove alias", "Name alias", "Notes", "Versions come from go.dev.", "Documentation", "Guide"} {
		if !sdkHasText(tt, text) {
			t.Errorf("the options have no %q; the frame shows %q", text, tt.Texts())
		}
	}
	sdkShot(t, tt, "page-options")
}

func TestSDKPageChangesAVariableWithItsAddress(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Module mirror")
	if err := tt.Click("Module mirror"); err != nil {
		t.Fatalf("Click(Module mirror): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Where the modules come from.")
	if err := tt.Click("Address"); err != nil {
		t.Fatalf("the address field is not there: %v; the frame shows %q", err, tt.Texts())
	}
	tt.Type("https://goproxy.cn")
	sdkSettle(tt)
	sdkShot(t, tt, "page-variable-dialog")
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("Click(Save): %v", err)
	}
	sdkWait(t, tt, "the variable is not saved", func() bool {
		return len(fake.Requests("PUT", "/api/sdks/go/variables")) == 1
	})
	var sent struct{ Key, Value string }
	if err := json.Unmarshal([]byte(fake.Requests("PUT", "/api/sdks/go/variables")[0].Body), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Key != "GOPROXY_MIRROR" || sent.Value != "https://goproxy.cn" {
		t.Errorf("the variable is sent as %+v, want GOPROXY_MIRROR set to the address", sent)
	}
	sdkWaitText(t, tt, "Saved")
}

func TestSDKPageSavesAPresetAndSaysWhyAVariableWasNotSaved(t *testing.T) {
	fake := sdkDetailFake(t)
	fake.Fail["PUT /api/sdks/go/variables"] = sdkFailure{Status: http.StatusBadRequest, Message: "not a channel"}
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Channel")
	if err := tt.Click("Channel"); err != nil {
		t.Fatalf("Click(Channel): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Which releases are listed.")
	if err := tt.Click("Stable"); err != nil {
		t.Fatalf("Click(Stable): %v", err)
	}
	sdkSettle(tt)
	if err := tt.Click("Save"); err != nil {
		t.Fatalf("Click(Save): %v", err)
	}
	sdkWaitText(t, tt, "not a channel")
	if sdkHasText(tt, "Saved") {
		t.Error("a variable that was refused says it is saved")
	}
	if !sdkHasText(tt, "Which releases are listed.") {
		t.Error("the dialog closed although the save failed")
	}
	sdkShot(t, tt, "page-variable-failed")
}

func TestSDKPageTurnsTheLegacyFilesOn(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Read legacy version files")
	sdkClickSwitch(t, tt, "Read legacy version files")
	sdkWait(t, tt, "the switch does not write the setting", func() bool {
		return len(fake.Requests("PUT", "/api/config")) == 1
	})
	var sent struct{ Key, Value string }
	if err := json.Unmarshal([]byte(fake.Requests("PUT", "/api/config")[0].Body), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Key != "legacyVersionFile.enable" || sent.Value != "true" {
		t.Errorf("the configuration is sent as %+v, want the version files read", sent)
	}
}

func TestSDKPageSwitchesTheLegacyFilesAndTheirStrategy(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Latest installed")
	if err := tt.Click("Latest installed"); err != nil {
		t.Fatalf("Click(Latest installed): %v", err)
	}
	sdkWait(t, tt, "the strategy is not written", func() bool {
		return len(fake.Requests("PUT", "/api/config")) == 1
	})
	var sent struct{ Key, Value string }
	if err := json.Unmarshal([]byte(fake.Requests("PUT", "/api/config")[0].Body), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Key != "legacyVersionFile.strategy" || sent.Value != "installed" {
		t.Errorf("the configuration is sent as %+v, want the strategy installed", sent)
	}
}

func TestSDKPageRemovesAnAlias(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Remove alias")
	if err := tt.Click("Remove alias"); err != nil {
		t.Fatalf("Click(Remove alias): %v", err)
	}
	sdkWait(t, tt, "the alias is not removed", func() bool {
		return len(sdkTaskRequests(t, fake)) == 1
	})
	want := remote.TaskRequest{Operation: remote.RequestAliasUnset, Sdk: "go", Alias: "latest"}
	if got := sdkTaskRequests(t, fake)[0]; got != want {
		t.Errorf("the task is %+v, want %+v", got, want)
	}
}

func TestSDKPageOpensADocumentInTheBrowser(t *testing.T) {
	fake := sdkDetailFake(t)
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Guide")
	sdkClickScrolled(t, tt, "Guide")
	sdkSettle(tt)
	if got := tt.OpenedURLs(); len(got) != 1 || got[0] != "https://go.dev/doc" {
		t.Errorf("opened %v, want the guide", got)
	}
}

func TestSDKPageHasNoOptionsTabWithoutAProfile(t *testing.T) {
	fake := sdkDetailFake(t)
	delete(fake.Profiles, "go")
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Installed (2)")
	if sdkHasText(tt, "Options") {
		t.Error("a plugin with no profile shows the options tab")
	}
}

func TestSDKPageAsksAgainWhenTheOptionsFail(t *testing.T) {
	fake := sdkDetailFake(t)
	fake.Fail["GET /api/sdks/go/profile"] = sdkFailure{Status: 503, Message: "profile is busy"}
	root := sdkRoot(t, fake)
	tt := sdkWindow(root)
	root.App().Push(app.Page{Kind: app.PageSDK, Name: "go"})
	sdkSettle(tt)
	sdkWaitText(t, tt, "Options")
	if err := tt.Click("Options"); err != nil {
		t.Fatalf("Click(Options): %v", err)
	}
	sdkSettle(tt)
	sdkWaitText(t, tt, "Request failed.")
	if !sdkHasText(tt, "profile is busy") {
		t.Errorf("the failure does not say why; the frame shows %q", tt.Texts())
	}
	sdkShot(t, tt, "page-options-failed")
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("the failure has no retry button: %v", err)
	}
}
