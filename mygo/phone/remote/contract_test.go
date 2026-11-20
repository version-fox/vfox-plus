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

package remote

// Each method of the client makes at least one successful call against the real
// server, and the answer decodes strictly into the type that the client returns.

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestInventoryReportsVfoxAndEverySection(t *testing.T) {
	s := startServer(t)
	client, rec := s.paired(t)
	inventory, err := client.Inventory(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	strict := strictly[Inventory](t, rec)
	if !strict.Complete || strict.Vfox.Availability != AvailabilityReady {
		t.Fatalf("inventory = %+v, want complete with vfox ready", strict)
	}
	found := false
	for _, sdk := range inventory.Installed.Value {
		found = found || sdk.Name == "nodejs"
	}
	if !found {
		t.Errorf("nodejs is not among the installed SDKs: %+v", inventory.Installed.Value)
	}
	if len(inventory.Registry.Value) == 0 || len(inventory.Config.Value) == 0 {
		t.Errorf("registry or configuration is empty: %d plugins, %d keys",
			len(inventory.Registry.Value), len(inventory.Config.Value))
	}
}

func TestPluginCallsDecodeAsTheClientTypes(t *testing.T) {
	s := startServer(t)
	client, rec := s.paired(t)
	ctx := t.Context()

	if _, err := client.SdkInfo(ctx, "nodejs"); err != nil {
		t.Fatal(err)
	}
	if info := strictly[PluginInfo](t, rec); info.Name != "nodejs" {
		t.Errorf("SdkInfo name = %q", info.Name)
	}

	versions, err := client.Versions(ctx, "nodejs", "")
	if err != nil || len(versions) == 0 {
		t.Fatalf("Versions(nodejs) = %d versions, %v", len(versions), err)
	}
	strictly[[]AvailableVersion](t, rec)

	// The search asks Java for another distribution for this call only; the
	// answer then carries the parts and facets of the profile.
	java, err := client.Versions(ctx, "java", "tem")
	if err != nil || len(java) == 0 {
		t.Fatalf("Versions(java, tem) = %d versions, %v", len(java), err)
	}
	strictly[[]AvailableVersion](t, rec)
	if java[0].Parts == nil {
		t.Errorf("the versions of java carry no parts: %+v", java[0])
	}

	profile, err := client.Profile(ctx, "java")
	if err != nil || profile == nil {
		t.Fatalf("Profile(java) = %v, %v", profile, err)
	}
	strictly[*Profile](t, rec)
	if profile.Search == nil || len(profile.Search.Choices) == 0 {
		t.Errorf("the profile of java has no choice of what to list: %+v", profile.Search)
	}
	// The installed Java versions of the fixtures are taken apart by the profile,
	// with their distribution as the source.
	installed, ok := profile.Installed["17.0.9-tem"]
	if !ok || installed.Source == nil || installed.Source.Value != "tem" {
		t.Errorf("the installed versions of java are %+v, want 17.0.9-tem from tem", profile.Installed)
	}

	if _, err := client.Profile(ctx, "nodejs"); err != nil {
		t.Fatal(err)
	}
	strictly[*Profile](t, rec)

	// A plugin with no profile is not an error.
	if missing, err := client.Profile(ctx, "no-such-plugin"); err != nil || missing != nil {
		t.Errorf("Profile of a plugin without one = %v, %v; want nil, nil", missing, err)
	}

	aliases, err := client.Aliases(ctx, "nodejs")
	if err != nil {
		t.Fatal(err)
	}
	strictly[[]VersionAlias](t, rec)
	if len(aliases) != 1 || aliases[0].Alias != "lts" {
		t.Errorf("the aliases of nodejs = %+v, want lts", aliases)
	}
}

func TestDoctorReportsVfox(t *testing.T) {
	s := startServer(t)
	client, rec := s.paired(t)
	report, err := client.Doctor(t.Context(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	strictly[DoctorReport](t, rec)
	if report.OK || len(report.Findings) != 1 || report.Findings[0].Sdk != "nodejs" {
		t.Errorf("the doctor report = %+v, want one finding for nodejs", report)
	}
}

func TestChoicesVariablesAndConfigAreWritten(t *testing.T) {
	s := startServer(t)
	client, _ := s.paired(t)
	ctx := t.Context()

	if err := client.SetSearch(ctx, "java", "all"); err != nil {
		t.Fatalf("SetSearch: %v", err)
	}

	const mirror = "https://npmmirror.com/mirrors/node"
	if err := client.SetVariable(ctx, "nodejs", "VFOX_NODEJS_MIRROR", mirror); err != nil {
		t.Fatalf("SetVariable: %v", err)
	}
	profile, err := client.Profile(ctx, "nodejs")
	if err != nil || profile == nil {
		t.Fatalf("Profile(nodejs) = %v, %v", profile, err)
	}
	if got := variableValue(profile, "VFOX_NODEJS_MIRROR"); got != mirror {
		t.Errorf("the variable reads %q after SetVariable, want %q", got, mirror)
	}
	if err := client.SetVariable(ctx, "nodejs", "VFOX_NODEJS_MIRROR", ""); err != nil {
		t.Fatalf("SetVariable to the default: %v", err)
	}

	if err := client.PutConfig(ctx, "proxy.url", "http://proxy.example:8080"); err != nil {
		t.Fatalf("PutConfig: %v", err)
	}
	if err := client.DeleteConfig(ctx, "proxy.url"); err != nil {
		t.Fatalf("DeleteConfig: %v", err)
	}
}

func variableValue(profile *Profile, key string) string {
	for _, variable := range profile.Variables {
		if variable.Key == key {
			return variable.Value
		}
	}
	return ""
}

func TestSettingsWebAndQuit(t *testing.T) {
	s := startServer(t)
	client, rec := s.paired(t)
	ctx := t.Context()

	if _, err := client.Settings(ctx); err != nil {
		t.Fatal(err)
	}
	strictly[AppSettings](t, rec)

	// An empty map is sent too, so that a patch can clear the variables.
	english := LanguageEnglish
	noVariables := map[string]string{}
	if _, err := client.PutSettings(ctx, SettingsPatch{Language: &english, ExtraEnvironment: &noVariables}); err != nil {
		t.Fatal(err)
	}
	if updated := strictly[AppSettings](t, rec); updated.Language != LanguageEnglish {
		t.Errorf("language after the patch = %q, want %q", updated.Language, LanguageEnglish)
	}

	if _, err := client.Web(ctx); err != nil {
		t.Fatal(err)
	}
	strictly[WebState](t, rec)
	lightweight := true
	if _, err := client.PutWeb(ctx, WebPatch{Lightweight: &lightweight}); err != nil {
		t.Fatal(err)
	}
	if web := strictly[WebState](t, rec); !web.Lightweight {
		t.Errorf("lightweight after the patch = false")
	}

	code, err := client.WebPairing(ctx, "192.168.1.10")
	if err != nil {
		t.Fatal(err)
	}
	strictly[PairingCode](t, rec)
	if !strings.HasPrefix(code.URL, "https://192.168.1.10:8369/auth/connect/") || !strings.Contains(code.SVG, "<svg") {
		t.Errorf("the pairing code = %q, svg of %d bytes", code.URL, len(code.SVG))
	}

	if err := client.Quit(ctx); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for s.control.quits.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the server did not ask the application to quit")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTasksStartListFollowAndFinish(t *testing.T) {
	s := startServer(t)
	client, rec := s.paired(t)
	ctx := t.Context()

	started, err := client.StartTask(ctx, TaskRequest{Operation: RequestInstall, Sdk: "nodejs", Version: "22.21.0"})
	if err != nil {
		t.Fatal(err)
	}
	if task := strictly[Task](t, rec); task.ID != started.ID || task.Operation != OperationInstall {
		t.Fatalf("StartTask = %+v", task)
	}

	tasks, err := client.Tasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	strictly[[]Task](t, rec)
	listed := false
	for _, task := range tasks {
		listed = listed || task.ID == started.ID
	}
	if !listed {
		t.Errorf("task %d is not in the list", started.ID)
	}

	if _, err := client.Task(ctx, started.ID); err != nil {
		t.Fatal(err)
	}
	strictly[Task](t, rec)

	// A task that succeeded or is still running has no complete log.
	if _, err := client.TaskLog(ctx, started.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("TaskLog of a task without one = %v, want ErrNotFound", err)
	}
	if err := client.CancelTask(ctx, started.ID); err != nil {
		t.Fatal(err)
	}

	// A version that does not exist fails, and the desktop keeps its full log.
	failed, err := client.StartTask(ctx, TaskRequest{Operation: RequestInstall, Sdk: "nodejs", Version: "99.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	strictly[Task](t, rec)
	waitForStatus(t, client, failed.ID, TaskFailed)
	waitForFullLog(t, client, failed.ID)
	log, err := client.TaskLog(ctx, failed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "no such version") {
		t.Errorf("the full log is %q", log)
	}
	if err := client.RetryTask(ctx, failed.ID); err != nil {
		t.Fatal(err)
	}
	if err := client.ClearFinished(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := client.AddPlugin(ctx, "https://example.test/vfox-plugin-demo.git", "demo"); err != nil {
		t.Fatal(err)
	}
	if added := strictly[Task](t, rec); added.Operation != OperationAddPlugin {
		t.Errorf("AddPlugin operation = %q, want %q", added.Operation, OperationAddPlugin)
	}
}

// waitForFullLog polls until the desktop has kept the complete log of the task.
func waitForFullLog(t *testing.T, client *Client, id int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		task, err := client.Task(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if task.HasFullLog {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %d kept no full log", id)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
