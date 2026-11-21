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
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// sdkInventoryFixture is the inventory of two plugins: go, with two versions and one
// in use, and java, with one version and none in use.
func sdkInventoryFixture() remote.Inventory {
	return remote.Inventory{
		Vfox: remote.VfoxStatus{Availability: remote.AvailabilityReady, Version: "0.9.0"},
		Installed: remote.Section[[]remote.InstalledSdk]{Value: []remote.InstalledSdk{
			{Name: "go", Versions: []string{"1.21.0", "1.22.0"}, Current: "1.22.0", HasCurrent: true},
			{Name: "java", Versions: []string{"21.0.1"}},
		}},
		Config: remote.Section[[]remote.ConfigEntry]{Value: []remote.ConfigEntry{
			{Key: "legacyVersionFile.enable", Value: "false"},
		}},
		Complete: true,
	}
}

func TestSDKListShowsEachPluginWithTheVersionInUse(t *testing.T) {
	fake := newSDKFake(t)
	fake.Inventory = sdkInventoryFixture()
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "2 Versions")
	for _, text := range []string{"SDKs", "go", "java", "1.22.0", "— · 1 Versions"} {
		if !sdkHasText(tt, text) {
			t.Errorf("the list has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	if sdkHasText(tt, "1.21.0") {
		t.Error("the list shows the versions of a plugin; it shows the plugins only")
	}
	sdkShot(t, tt, "list")
}

func TestSDKListSearchMatchesNamesWithoutRegardToCase(t *testing.T) {
	fake := newSDKFake(t)
	fake.Inventory = sdkInventoryFixture()
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "2 Versions")
	if err := tt.Click("Search SDKs…"); err != nil {
		t.Fatalf("the search field is not there: %v; the frame shows %q", err, tt.Texts())
	}
	tt.Type("JA")
	sdkWait(t, tt, "the search for JA does not leave java alone", func() bool {
		return sdkHasText(tt, "java") && !sdkHasText(tt, "go")
	})
	sdkShot(t, tt, "list-search")
}

func TestSDKListSaysWhenNothingMatches(t *testing.T) {
	fake := newSDKFake(t)
	fake.Inventory = sdkInventoryFixture()
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "2 Versions")
	if err := tt.Click("Search SDKs…"); err != nil {
		t.Fatalf("the search field is not there: %v", err)
	}
	tt.Type("zzz")
	sdkWaitText(t, tt, "No plugin matches.")
	sdkShot(t, tt, "list-no-match")
}

func TestSDKListWithNoPluginOffersThePluginsTab(t *testing.T) {
	fake := newSDKFake(t)
	fake.Inventory = remote.Inventory{Complete: true}
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "No SDKs installed yet.")
	sdkShot(t, tt, "list-empty")
	if err := tt.Click("Add a plugin"); err != nil {
		t.Fatalf("the list has no button to add a plugin: %v; the frame shows %q", err, tt.Texts())
	}
	sdkSettle(tt)
	if root.App().Tab() != app.TabPlugins {
		t.Errorf("tab %v after the button, want the plugins tab", root.App().Tab())
	}
}

func TestSDKListWaitsForTheComputer(t *testing.T) {
	fake := newSDKFake(t)
	fake.Inventory = sdkInventoryFixture()
	open := fake.Hold()
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "Loading…")
	sdkShot(t, tt, "list-loading")
	open()
	sdkWaitText(t, tt, "go")
}

func TestSDKListAsksAgainWhenTheComputerFails(t *testing.T) {
	fake := newSDKFake(t)
	fake.Inventory = sdkInventoryFixture()
	fake.Fail["GET /api/inventory"] = sdkFailure{Status: 503, Message: "vfox is busy"}
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "Request failed.")
	if !sdkHasText(tt, "vfox is busy") {
		t.Errorf("the failure does not say why; the frame shows %q", tt.Texts())
	}
	sdkShot(t, tt, "list-failed")

	fake.mu.Lock()
	delete(fake.Fail, "GET /api/inventory")
	fake.mu.Unlock()
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("the failure has no retry button: %v; the frame shows %q", err, tt.Texts())
	}
	sdkWaitText(t, tt, "2 Versions")
}

func TestSDKListTapOpensThePageOfThePlugin(t *testing.T) {
	fake := newSDKFake(t)
	fake.Inventory = sdkInventoryFixture()
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "2 Versions")
	if err := tt.Click("go"); err != nil {
		t.Fatalf("Click(go): %v", err)
	}
	sdkSettle(tt)
	top, ok := root.App().Top()
	if !ok || top != (app.Page{Kind: app.PageSDK, Name: "go"}) {
		t.Errorf("the page on top is %v (%v), want the page of go", top, ok)
	}
}

func TestSDKListShowsLongListsAPageAtATime(t *testing.T) {
	fake := newSDKFake(t)
	var plugins []remote.InstalledSdk
	for i := range 120 {
		plugins = append(plugins, remote.InstalledSdk{Name: fmt.Sprintf("plugin-%03d", i), Versions: []string{"1"}})
	}
	fake.Inventory = remote.Inventory{Installed: remote.Section[[]remote.InstalledSdk]{Value: plugins}, Complete: true}
	root := sdkRoot(t, fake)
	root.App().SelectTab(app.TabSDKs)
	tt := sdkWindow(root)
	sdkWaitText(t, tt, "plugin-000")
	if sdkHasText(tt, "plugin-050") || !sdkHasText(tt, "plugin-049") {
		t.Fatalf("the first page is not the first %d plugins; the frame shows %q", sdkPageSize, tt.Texts())
	}
	sdkClickScrolled(t, tt, "Show 50 more")
	sdkWaitText(t, tt, "plugin-099")
	sdkClickScrolled(t, tt, "Show 20 more")
	sdkWaitText(t, tt, "plugin-119")
}
