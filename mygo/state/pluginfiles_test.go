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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// pluginHome makes the home directory of a user with the plugin name, whose
// files are given by their path in the plugin.
func pluginHome(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	for path, content := range files {
		file := filepath.Join(home, ".vfox", "plugin", name, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

// infoRunner answers for the plugin name, which says that it is declared.
func infoRunner(name, declared string) *testsupport.FakeProcessRunner {
	info := declared + "\x1f1.0.0\x1fhttps://example.com/" + declared + "\x1f/home/u/.vfox/cache/" + name + "\x1fA plugin"
	return testsupport.FixtureRunner().
		On("info "+name+" --format "+vfox.PluginInfoTemplate, core.ProcessOutput{Stdout: info}).
		On("search "+name, testsupport.FixtureOutput("search-java", 0)).
		On("search "+name+" lts", testsupport.FixtureOutput("search-java-tem", 0))
}

func modelIn(t *testing.T, runner *testsupport.FakeProcessRunner, home string, options state.AppModelOptions) *state.AppModel {
	t.Helper()
	options.Runner = runner
	if options.SettingsStore == nil {
		options.SettingsStore = settings.NewMemorySettingsStore(settings.DefaultSettings())
	}
	options.Environment = map[string]string{"HOME": home}
	options.Locator = vfox.NewVfoxLocator(map[string]string{}, func(path string) bool { return path == "/usr/bin/vfox" })
	model := state.NewAppModel(options)
	t.Cleanup(model.Dispose)
	return model
}

func TestAPluginDescribesItself(t *testing.T) {
	home := pluginHome(t, "x", map[string]string{
		"metadata.lua":        `PLUGIN.legacyFilenames = { ".x-version" }`,
		"hooks/available.lua": `local mirror = os.getenv("VFOX_X_MIRROR")`,
		"lib/vfox-plus.json": `{"guide": "From the plugin", "search": {"label": "Line", "default": "all",
			"choices": [{"value": "all"}, {"value": "lts"}]}}`,
	})
	model := modelIn(t, infoRunner("x", "x"), home, state.AppModelOptions{Profiles: linuxProfiles(t)})
	detail := model.SdkDetail("x")
	defer detail.Dispose()
	if _, found := detail.Profile(); found || !detail.CanDetect() {
		t.Fatal("before the files are read, nothing is known")
	}
	<-detail.Versions.Load()
	notified := make(chan struct{}, 8)
	stop := detail.AddListener(func() { notified <- struct{}{} })
	defer stop()
	<-detail.Files.Load()
	known, found := detail.Profile()
	if !found || known.Guide.EN != "From the plugin" || known.Search == nil || detail.CanDetect() {
		t.Fatalf("profile = %+v, found = %v, detect = %v", known, found, detail.CanDetect())
	}
	select {
	case <-notified:
	default:
		t.Error("whoever shows the details is told")
	}
	// The plugin is no curated one: what its profile does not say is inferred.
	<-detail.Info.Load()
	effective, _ := detail.EffectiveProfile()
	if effective.Guide.EN != "From the plugin" || !reflect.DeepEqual(effective.LegacyFiles, []string{".x-version"}) ||
		len(effective.Variables) != 1 || effective.Variables[0].Key != "VFOX_X_MIRROR" || len(effective.Facets) == 0 {
		t.Errorf("effective = %+v", effective)
	}
	if err := detail.SetSearch("lts"); err != nil {
		t.Fatal(err)
	}
	<-detail.Versions.EnsureLoaded()
	if versions, _ := detail.Versions.Value(); len(versions) != 123 {
		t.Errorf("%d versions of the search of the plugin", len(versions))
	}
}

func TestTheFacetsThatAreShownFilter(t *testing.T) {
	// A profile of the application without facets: the interfaces offer those
	// that the versions give, and they filter the list.
	registry := profile.NewRegistry(profile.Platform{OS: "linux", Arch: "amd64"}, map[string]profile.Document{})
	registry.Register(profile.Profile{Plugin: "java"})
	controller := state.NewSdkDetailController(testClient(testsupport.FixtureRunner()), "java", registry, nil)
	defer controller.Dispose()
	<-controller.Versions.Load()
	all := len(controller.VisibleVersions(nil))
	groups := controller.Groups()
	if len(groups) == 0 {
		t.Fatal("the notes of java give facets")
	}
	controller.ToggleFacet(groups[0].ID, groups[0].Chips[0].Value)
	if got := len(controller.VisibleVersions(nil)); got != groups[0].Chips[0].Count || got == all {
		t.Errorf("%d of %d versions have %q, the filter says %d", got, all, groups[0].Chips[0].Value, groups[0].Chips[0].Count)
	}
}

func TestTheProfilesOfTheUserAndOfTheDirectoryReachTheDetails(t *testing.T) {
	files := map[string]string{"x": `{"plugin": "x", "guide": "online"}`, "nodejs": `{"plugin": "nodejs", "guide": "online"}`}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSuffix(r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:], ".json")
		if name == "index" {
			index := profile.Index{Schema: profile.SchemaVersion, Plugins: map[string]string{}}
			for name, content := range files {
				index.Plugins[name] = profile.SHA256([]byte(content))
			}
			_ = json.NewEncoder(w).Encode(index)
			return
		}
		_, _ = w.Write([]byte(files[name]))
	}))
	defer server.Close()
	userDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(userDirectory, "nodejs.json"), []byte(`{"plugin": "nodejs", "tips": ["mine"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog := &profile.Catalog{Directory: t.TempDir(), URL: server.URL + "/"}
	model := modelIn(t, testsupport.FixtureRunner(), "/home/u", state.AppModelOptions{Profiles: linuxProfiles(t), ProfilesDir: userDirectory, Catalog: catalog})
	detail := model.SdkDetail("nodejs")
	defer detail.Dispose()
	waitFor(t, "the profile of the directory", func() bool {
		known, _ := detail.Profile()
		return known.Guide.EN == "online"
	})
	known, _ := detail.Profile()
	if len(known.Tips) != 1 || known.Tips[0].EN != "mine" || len(known.Shorthands) == 0 {
		t.Errorf("each part comes from its source: %+v", known)
	}
	// The next run starts with the cache, without the network.
	server.Close()
	again := modelIn(t, testsupport.FixtureRunner(), "/home/u", state.AppModelOptions{Profiles: linuxProfiles(t), Catalog: catalog})
	other := again.SdkDetail("x")
	defer other.Dispose()
	if got, found := other.Profile(); !found || got.Guide.EN != "online" {
		t.Errorf("profile = %+v, %v", got, found)
	}
}

// A change of the profiles while the details are made reaches them.
func TestNoChangeOfTheProfilesIsMissed(t *testing.T) {
	registry := profile.NewRegistry(profile.Platform{OS: "linux", Arch: "amd64"}, nil)
	runner := testsupport.FixtureRunner()
	d, _, err := profile.ParseDocument([]byte(`{"plugin": "x", "guide": "new"}`))
	if err != nil {
		t.Fatal(err)
	}
	for i := range 2000 {
		registry.SetCatalog(nil)
		done := make(chan struct{})
		go func() {
			defer close(done)
			registry.SetCatalog(map[string]profile.Document{"x": d})
		}()
		controller := state.NewSdkDetailController(testClient(runner), "x", registry, nil)
		<-done
		known, _ := controller.Profile()
		controller.Dispose()
		if known.Guide.EN != "new" {
			t.Fatalf("the details of round %d missed the change: %+v", i, known)
		}
	}
}
