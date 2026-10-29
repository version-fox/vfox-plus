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
	"os"
	"strings"
	"sync/atomic"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/state"
)

var errBoom = errors.New("boom")

// gatedLoader is a loader whose calls wait for the test: every call reports
// its number on started and returns what the test sends to its gate.
type gatedLoader struct {
	calls   atomic.Int32
	started chan int
	gates   [4]chan string
}

func newGatedLoader() *gatedLoader {
	loader := &gatedLoader{started: make(chan int, 4)}
	for i := range loader.gates {
		loader.gates[i] = make(chan string, 1)
	}
	return loader
}

func (l *gatedLoader) load() (string, error) {
	call := int(l.calls.Add(1))
	l.started <- call
	return <-l.gates[call-1], nil
}

func TestAsyncResourceKeepsThePreviousValueWhileReloadingAndRecordsErrors(t *testing.T) {
	var calls atomic.Int32
	resource := state.NewAsyncResource(func() (int, error) {
		call := int(calls.Add(1))
		if call == 2 {
			return 0, errBoom
		}
		return call, nil
	})
	if resource.HasValue() || resource.IsLoading() {
		t.Fatal("a new resource has nothing")
	}
	<-resource.Load()
	if value, ok := resource.Value(); !ok || value != 1 || !resource.HasValue() {
		t.Fatalf("got %v %v", value, ok)
	}
	reload := resource.Load()
	if !resource.IsLoading() {
		t.Fatal("expected loading")
	}
	<-reload
	if !errors.Is(resource.Error(), errBoom) {
		t.Fatalf("got %v", resource.Error())
	}
	if value, _ := resource.Value(); value != 1 || resource.IsLoading() {
		t.Errorf("keeps the previous value: %v", value)
	}
	<-resource.Load()
	if value, _ := resource.Value(); value != 3 || resource.Error() != nil {
		t.Errorf("a success clears the error: %v %v", value, resource.Error())
	}
}

func TestAsyncResourceLetsTheLatestOfOverlappingLoadsWin(t *testing.T) {
	for _, newerFirst := range []bool{true, false} {
		loader := newGatedLoader()
		resource := state.NewAsyncResource(loader.load)
		older := resource.Load()
		<-loader.started
		newer := resource.Load()
		<-loader.started
		if newerFirst {
			loader.gates[1] <- "new"
			<-newer
			loader.gates[0] <- "old"
			<-older
		} else {
			loader.gates[0] <- "old"
			<-older
			loader.gates[1] <- "new"
			<-newer
		}
		if value, _ := resource.Value(); value != "new" || resource.IsLoading() {
			t.Errorf("newerFirst=%v: got %q", newerFirst, value)
		}
	}
}

func TestAsyncResourceIgnoresTheFailureOfAReplacedLoad(t *testing.T) {
	var calls atomic.Int32
	started := make(chan int, 2)
	gates := [2]chan struct{}{make(chan struct{}), make(chan struct{})}
	resource := state.NewAsyncResource(func() (string, error) {
		call := int(calls.Add(1))
		started <- call
		<-gates[call-1]
		if call == 1 {
			return "", errBoom
		}
		return "fine", nil
	})
	older := resource.Load()
	<-started
	newer := resource.Load()
	<-started
	close(gates[1])
	<-newer
	close(gates[0])
	<-older
	if value, _ := resource.Value(); value != "fine" || resource.Error() != nil {
		t.Errorf("got %q %v", value, resource.Error())
	}
}

func TestAsyncResourceEnsuresItIsLoadedOnce(t *testing.T) {
	var calls atomic.Int32
	loader := newGatedLoader()
	resource := state.NewAsyncResource(func() (string, error) {
		calls.Add(1)
		return loader.load()
	})
	first := resource.EnsureLoaded()
	<-loader.started
	second := resource.EnsureLoaded()
	loader.gates[0] <- "value"
	<-first
	<-second
	<-resource.EnsureLoaded()
	if calls.Load() != 1 {
		t.Fatalf("got %d loads", calls.Load())
	}
	resource.SetValue("set")
	if value, _ := resource.Value(); value != "set" || resource.Error() != nil || resource.IsLoading() {
		t.Errorf("got %q", value)
	}
}

func TestSetValueReplacesALoadInFlight(t *testing.T) {
	loader := newGatedLoader()
	resource := state.NewAsyncResource(loader.load)
	load := resource.Load()
	<-loader.started
	resource.SetValue("optimistic")
	loader.gates[0] <- "loaded"
	<-load
	if value, _ := resource.Value(); value != "optimistic" {
		t.Errorf("got %q", value)
	}
}

func TestListenersAreNotifiedUntilTheyStopOrTheObjectIsDisposed(t *testing.T) {
	resource := state.NewAsyncResource(func() (int, error) { return 1, nil })
	var notified atomic.Int32
	stop := state.ListenAll([]state.Listenable{resource, nil}, func() { notified.Add(1) })
	<-resource.Load()
	if notified.Load() != 2 {
		t.Fatalf("got %d notifications", notified.Load())
	}
	stop()
	stop()
	<-resource.Load()
	if notified.Load() != 2 {
		t.Fatalf("got %d notifications", notified.Load())
	}
	resource.AddListener(func() { notified.Add(1) })
	resource.Dispose()
	if !resource.IsDisposed() {
		t.Error("expected disposed")
	}
	resource.NotifyListeners()
	<-resource.Load()
	if notified.Load() != 2 {
		t.Errorf("a disposed object is silent, got %d", notified.Load())
	}
}

func TestEmitterDeliversToListenersUntilTheyStop(t *testing.T) {
	var emitter state.Emitter[string]
	var first, second []string
	stopFirst := emitter.Listen(func(value string) { first = append(first, value) })
	emitter.Listen(func(value string) { second = append(second, value) })
	emitter.Emit("a")
	stopFirst()
	emitter.Emit("b")
	emitter.Close()
	emitter.Emit("c")
	if len(first) != 1 || first[0] != "a" || len(second) != 2 || second[1] != "b" {
		t.Errorf("got %q %q", first, second)
	}
}

func TestSettingsControllerPersistsChangesAndSkipsNoOps(t *testing.T) {
	dark := settings.DefaultSettings()
	dark.Theme = settings.ThemeDark
	store := settings.NewMemorySettingsStore(dark)
	controller := state.NewSettingsController(store)
	if controller.Settings().Theme != settings.ThemeSystem {
		t.Fatalf("got %#v", controller.Settings())
	}
	controller.Load()
	if controller.Settings().Theme != settings.ThemeDark {
		t.Fatalf("got %#v", controller.Settings())
	}
	var changes atomic.Int32
	stop := controller.AddListener(func() { changes.Add(1) })
	controller.Update(func(current settings.AppSettings) settings.AppSettings { return current })
	if store.SaveCount() != 0 || changes.Load() != 0 {
		t.Error("no-ops are neither saved nor announced")
	}
	controller.Update(func(current settings.AppSettings) settings.AppSettings {
		current.Theme = settings.ThemeLight
		current.ExtraEnvironment["A"] = "1"
		return current
	})
	saved := store.Load()
	if store.SaveCount() != 1 || changes.Load() != 1 || saved.Theme != settings.ThemeLight || saved.ExtraEnvironment["A"] != "1" {
		t.Errorf("changes are persisted, got %#v", saved)
	}
	stop()
}

func TestSettingsControllerHandsOutCopiesToChange(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.DefaultSettings())
	controller := state.NewSettingsController(store)
	controller.Load()
	controller.Update(func(current settings.AppSettings) settings.AppSettings {
		current.ExtraEnvironment["A"] = "1"
		return current
	})
	// A change that edits a map in place must still be seen as a change.
	controller.Update(func(current settings.AppSettings) settings.AppSettings {
		current.ExtraEnvironment["A"] = "2"
		return current
	})
	if store.SaveCount() != 2 || controller.Settings().ExtraEnvironment["A"] != "2" {
		t.Errorf("got %d saves", store.SaveCount())
	}
}

func vfoxController(runner *testsupport.FakeProcessRunner, found string) (*state.VfoxController, *vfox.Client) {
	cli := testClient(runner)
	environment := map[string]string{"HOME": "/home/u"}
	locator := vfox.NewVfoxLocator(environment, func(path string) bool { return path == found })
	return state.NewVfoxController(cli, locator, environment), cli
}

func TestVfoxUsesTheDetectedExecutableAndReadsTheVersion(t *testing.T) {
	runner := testsupport.FixtureRunner()
	controller, cli := vfoxController(runner, "/usr/local/bin/vfox")
	if controller.Availability() != state.VfoxUnknown {
		t.Fatalf("got %v", controller.Availability())
	}
	appSettings := settings.DefaultSettings()
	appSettings.ExtraEnvironment = map[string]string{"A": "1"}
	controller.Apply(appSettings)
	if controller.Availability() != state.VfoxReady {
		t.Fatalf("got %v", controller.Availability())
	}
	if version, ok := controller.Version(); !ok || version != "1.0.13" {
		t.Errorf("got %q %v", version, ok)
	}
	if executable, ok := controller.Executable(); !ok || executable != "/usr/local/bin/vfox" || controller.IsCustomExecutable() {
		t.Errorf("got %q %v", executable, ok)
	}
	if controller.Error() != nil {
		t.Errorf("got %v", controller.Error())
	}
	config := cli.LaunchConfig()
	if config.Executable != "/usr/local/bin/vfox" || config.ExtraEnvironment["A"] != "1" || config.BaseEnvironment["HOME"] != "/home/u" {
		t.Errorf("got %#v", config)
	}
	if calls := runner.Invocations(); len(calls) != 1 || calls[0].Executable != "/usr/local/bin/vfox" {
		t.Errorf("got %#v", calls)
	}
}

func TestVfoxPrefersTheExecutableChosenByTheUser(t *testing.T) {
	runner := testsupport.FixtureRunner()
	controller, _ := vfoxController(runner, "/usr/local/bin/vfox")
	custom := settings.DefaultSettings()
	custom.VfoxExecutable = "/opt/vfox"
	controller.Apply(custom)
	if executable, _ := controller.Executable(); executable != "/opt/vfox" || !controller.IsCustomExecutable() {
		t.Errorf("got %q", executable)
	}
	if controller.Availability() != state.VfoxReady {
		t.Errorf("got %v", controller.Availability())
	}
}

func TestVfoxReportsAMissingAndABrokenExecutable(t *testing.T) {
	runner := testsupport.FixtureRunner()
	controller, cli := vfoxController(runner, "")
	runner.SetMissingExecutable(true)
	controller.Apply(settings.DefaultSettings())
	if controller.Availability() != state.VfoxNotFound {
		t.Fatalf("got %v", controller.Availability())
	}
	if _, ok := controller.Executable(); ok {
		t.Error("expected no executable")
	}
	if cli.LaunchConfig().Executable != vfox.VfoxExecutableName {
		t.Errorf("a missing executable is looked up by name, got %q", cli.LaunchConfig().Executable)
	}
	if _, ok := controller.Version(); ok {
		t.Error("expected no version")
	}
	runner.SetMissingExecutable(false)
	runner.On("--version", core.ProcessOutput{Stdout: "hello"})
	controller.Check()
	var failure *vfox.VfoxError
	if controller.Availability() != state.VfoxBroken || !errors.As(controller.Error(), &failure) {
		t.Errorf("got %v %#v", controller.Availability(), controller.Error())
	}
}

func TestVfoxReportsUnexpectedFailuresAsBroken(t *testing.T) {
	runner := testsupport.FixtureRunner()
	controller, _ := vfoxController(runner, "/usr/bin/vfox")
	runner.SetFailure(errBoom)
	controller.Check()
	if controller.Availability() != state.VfoxBroken || !errors.Is(controller.Error(), errBoom) {
		t.Errorf("got %v %v", controller.Availability(), controller.Error())
	}
}

func TestSdkDetailFiltersTheAvailableVersionsByTextNoteAndInstallation(t *testing.T) {
	controller := state.NewSdkDetailController(testClient(testsupport.FixtureRunner()), "nodejs", nil)
	defer controller.Dispose()
	var changes atomic.Int32
	controller.AddListener(func() { changes.Add(1) })
	<-controller.Versions.Load()
	if changes.Load() == 0 {
		t.Error("loading the versions is announced by the controller")
	}
	installed := map[string]bool{"24.11.1": true}
	if got := controller.VisibleVersions(installed); len(got) != 872 {
		t.Fatalf("got %d", len(got))
	}
	controller.SetFilter(state.VersionFilterWithNote)
	withNote := controller.VisibleVersions(installed)
	if len(withNote) == 0 {
		t.Fatal("expected versions with a note")
	}
	for _, version := range withNote {
		if !version.HasNote {
			t.Fatalf("got %#v", version)
		}
	}
	controller.SetFilter(state.VersionFilterAll)
	controller.SetQuery(" 24.11. ")
	if controller.Query() != " 24.11. " {
		t.Fatalf("got %q", controller.Query())
	}
	got := controller.VisibleVersions(installed)
	if len(got) != 2 || got[0].Version != "24.11.1" || got[1].Version != "24.11.0" {
		t.Fatalf("got %#v", got)
	}
	controller.SetQuery("lts")
	if len(controller.VisibleVersions(installed)) <= 10 {
		t.Error("the query also searches the notes")
	}
	controller.SetQuery("")
	controller.SetFilter(state.VersionFilterInstalled)
	if controller.Filter() != state.VersionFilterInstalled {
		t.Fatalf("got %q", controller.Filter())
	}
	got = controller.VisibleVersions(installed)
	if len(got) != 1 || got[0].Version != "24.11.1" {
		t.Fatalf("got %#v", got)
	}
	before := changes.Load()
	controller.SetFilter(state.VersionFilterInstalled)
	controller.SetQuery("")
	if changes.Load() != before {
		t.Error("setting the same value announces nothing")
	}
}

func TestSdkDetailLoadsThePluginInformation(t *testing.T) {
	controller := state.NewSdkDetailController(testClient(testsupport.FixtureRunner()), "nodejs", nil)
	defer controller.Dispose()
	<-controller.Info.Load()
	if info, ok := controller.Info.Value(); !ok || info.Version != "0.5.1" {
		t.Errorf("got %#v %v", info, ok)
	}
}

func TestConfigSavesReloadsAndExposesValuesByKey(t *testing.T) {
	runner := testsupport.FixtureRunner()
	gate := make(chan struct{})
	runner.OnDynamic("config gitignore.enable false", func() core.ProcessOutput {
		<-gate
		return core.ProcessOutput{}
	})
	controller := state.NewVfoxConfigController(testClient(runner))
	<-controller.Load()
	if value, ok := controller.ValueFor("gitignore.enable"); !ok || value != "true" {
		t.Fatalf("got %q %v", value, ok)
	}
	if _, ok := controller.ValueFor("missing"); ok {
		t.Error("expected nothing")
	}
	done := make(chan error, 1)
	go func() { done <- controller.Set("gitignore.enable", "false") }()
	waitFor(t, "saving", func() bool { return controller.IsSaving("gitignore.enable") })
	close(gate)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if controller.IsSaving("gitignore.enable") || len(runner.Calls("config --list")) != 2 {
		t.Error("expected a reload")
	}
	runner.On("config --unset proxy.url", core.ProcessOutput{})
	if err := controller.Reset("proxy.url"); err != nil {
		t.Fatal(err)
	}
	if len(runner.Calls("config --unset proxy.url")) != 1 || len(runner.Calls("config --list")) != 3 {
		t.Error("expected an unset and a reload")
	}
}

func TestConfigRethrowsRejectedValuesAfterReloading(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("config cache.availableHookDuration x", testsupport.FixtureOutput("config-set-invalid-duration", 1))
	controller := state.NewVfoxConfigController(testClient(runner))
	err := controller.Set("cache.availableHookDuration", "x")
	var failure *vfox.VfoxError
	if !errors.As(err, &failure) {
		t.Fatalf("expected the vfox error, got %v", err)
	}
	if !controller.HasValue() || controller.IsSaving("cache.availableHookDuration") {
		t.Error("the configuration is reloaded either way")
	}
}

func newTestModel(runner *testsupport.FakeProcessRunner, home string) *state.AppModel {
	return state.NewAppModel(state.AppModelOptions{
		Runner:        runner,
		SettingsStore: settings.NewMemorySettingsStore(settings.DefaultSettings()),
		Environment:   map[string]string{"HOME": home},
		Locator: vfox.NewVfoxLocator(map[string]string{},
			func(path string) bool { return path == "/usr/bin/vfox" }),
	})
}

// startedModel starts a model and waits until it has loaded the installed
// SDKs, the way the first page that needs vfox triggers the probe.
func startedModel(t *testing.T, runner *testsupport.FakeProcessRunner, home string) *state.AppModel {
	t.Helper()
	model := newTestModel(runner, home)
	t.Cleanup(model.Dispose)
	model.Start()
	model.Vfox.EnsureChecked()
	waitFor(t, "the installed SDKs", func() bool { return model.Installed.HasValue() })
	return model
}

func TestModelStartsWithoutProbingVfox(t *testing.T) {
	runner := testsupport.FixtureRunner()
	model := newTestModel(runner, "/home/u")
	t.Cleanup(model.Dispose)
	model.Start()
	if calls := runner.Invocations(); len(calls) != 0 {
		t.Fatalf("start must not spawn vfox, got %v", calls)
	}
	if model.Vfox.Availability() != state.VfoxUnknown {
		t.Fatalf("got %v, want unknown", model.Vfox.Availability())
	}
	model.Vfox.EnsureChecked()
	if model.Vfox.Availability() != state.VfoxReady {
		t.Fatalf("got %v, want ready", model.Vfox.Availability())
	}
	// A second call probes nothing more.
	calls := len(runner.Invocations())
	model.Vfox.EnsureChecked()
	if got := len(runner.Invocations()); got != calls {
		t.Fatalf("got %d invocations, want %d", got, calls)
	}
}

func TestModelLoadsTheInstalledSdksAfterTheFirstProbe(t *testing.T) {
	model := startedModel(t, testsupport.FixtureRunner(), "/home/u")
	if model.Vfox.Availability() != state.VfoxReady {
		t.Fatalf("got %v", model.Vfox.Availability())
	}
	sdks, _ := model.Installed.Value()
	var names []string
	for _, sdk := range sdks {
		names = append(names, sdk.Name)
	}
	if strings.Join(names, ",") != "maven,mvn2,nodejs" {
		t.Errorf("got %q", names)
	}
	installed, ok := model.InstalledSdk("nodejs")
	if !ok || !installed.HasCurrent || installed.Current != "24.11.1" {
		t.Errorf("got %#v", installed)
	}
	if _, ok := model.InstalledSdk("python"); ok {
		t.Error("unexpected SDK")
	}
	detail := model.SdkDetail("nodejs")
	defer detail.Dispose()
	if detail.Name != "nodejs" {
		t.Error("expected a detail controller")
	}
}

func TestModelReloadsTheInstalledSdksAfterATask(t *testing.T) {
	runner := testsupport.FixtureRunner()
	model := startedModel(t, runner, "/home/u")
	before := len(runner.Calls("list"))
	use, err := model.Cli.Use("nodejs", "22.21.0", core.UseScopeGlobal, "")
	if err != nil {
		t.Fatal(err)
	}
	task := model.Run(use)
	waitFor(t, "the reload", func() bool { return len(runner.Calls("list")) == before+1 })
	waitFor(t, "the task", func() bool { return task.Status() == state.TaskSucceeded })
}

func TestModelReloadsTheRegistryAndTheConfigurationWhenVfoxBecomesAvailableAgain(t *testing.T) {
	runner := testsupport.FixtureRunner()
	model := startedModel(t, runner, "/home/u")
	<-model.Registry.Load()
	<-model.VfoxConfig.Load()
	model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.ExtraEnvironment["X"] = "1"
		return current
	})
	waitFor(t, "the registry reload", func() bool { return len(runner.Calls("available")) == 2 })
	waitFor(t, "the configuration reload", func() bool { return len(runner.Calls("config --list")) == 2 })
	waitFor(t, "the environment", func() bool {
		for _, call := range runner.Invocations() {
			if call.Environment["X"] == "1" {
				return true
			}
		}
		return false
	})
}

func TestModelLeavesTheRegistryAloneUntilItWasLoaded(t *testing.T) {
	runner := testsupport.FixtureRunner()
	model := startedModel(t, runner, "/home/u")
	model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.VfoxExecutable = "/opt/vfox"
		return current
	})
	waitFor(t, "the second probe", func() bool { return len(runner.Calls("--version")) == 2 })
	waitFor(t, "ready", func() bool { return model.Vfox.Availability() == state.VfoxReady })
	if len(runner.Calls("available")) != 0 || len(runner.Calls("config --list")) != 0 {
		t.Error("the registry and the configuration load on demand")
	}
}

func TestModelHandsVfoxACopyOfALocalLuaPluginAndDeletesItAfterwards(t *testing.T) {
	runner := testsupport.FixtureRunner()
	home := t.TempDir()
	model := startedModel(t, runner, home)
	original := home + "/mine.lua"
	if err := os.WriteFile(original, []byte("PLUGIN = {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	task, err := model.AddPluginFromSource(original, "mine")
	if err != nil {
		t.Fatal(err)
	}
	args := task.Command.Arguments
	source := ""
	for i, arg := range args {
		if arg == "--source" && i+1 < len(args) {
			source = args[i+1]
		}
	}
	if !strings.HasPrefix(source, home+"/.vfox/tmp/vfox-plus-plugin-") || !strings.HasSuffix(source, "/mine.lua") {
		t.Fatalf("got %q", args)
	}
	if args[len(args)-2] != "--alias" || args[len(args)-1] != "mine" {
		t.Fatalf("got %q", args)
	}
	waitFor(t, "success", func() bool { return task.Status() == state.TaskSucceeded })
	waitFor(t, "the copy to be deleted", func() bool {
		_, err := os.Stat(source)
		return os.IsNotExist(err)
	})
	if _, err := os.Stat(original); err != nil {
		t.Error("the original must survive")
	}
}

func TestModelPassesUrlsToVfoxAsTheyAre(t *testing.T) {
	model := startedModel(t, testsupport.FixtureRunner(), t.TempDir())
	task, err := model.AddPluginFromSource("https://example.com/p.zip", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := task.Command.Arguments; len(got) != 3 || got[1] != "--source" || got[2] != "https://example.com/p.zip" {
		t.Errorf("got %q", got)
	}
}

func TestModelRejectsAnInvalidAlias(t *testing.T) {
	model := startedModel(t, testsupport.FixtureRunner(), t.TempDir())
	if _, err := model.AddPluginFromSource("https://example.com/p.zip", "-bad"); err == nil {
		t.Error("expected an error")
	}
	if len(model.Tasks.Tasks()) != 0 {
		t.Error("nothing is queued")
	}
}

func TestModelAppliesTheSettingsAgainWhenTheExecutableChanges(t *testing.T) {
	runner := testsupport.FixtureRunner()
	model := startedModel(t, runner, "/home/u")
	model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.VfoxExecutable = "/opt/vfox"
		return current
	})
	waitFor(t, "the custom executable", func() bool {
		executable, ok := model.Vfox.Executable()
		return ok && executable == "/opt/vfox" && model.Vfox.Availability() == state.VfoxReady
	})
	invocations := runner.Invocations()
	if got := invocations[len(invocations)-1].Executable; got != "/opt/vfox" {
		t.Errorf("got %q", got)
	}
	// A change that does not concern vfox applies nothing.
	probes := len(runner.Calls("--version"))
	model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.Theme = settings.ThemeDark
		return current
	})
	if len(runner.Calls("--version")) != probes {
		t.Error("the theme does not concern vfox")
	}
}

func TestModelDoesNotReloadAfterATaskThatNeverRan(t *testing.T) {
	runner := testsupport.FixtureRunner()
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("update --all", func() *testsupport.FakeRunningProcess { return process })
	model := startedModel(t, runner, "/home/u")
	before := len(runner.Calls("list"))
	running := model.Run(model.Cli.UpdateAllPlugins())
	queued := model.Run(mustCommand(model.Cli.UpdatePlugin("b")))
	model.Tasks.Cancel(queued)
	process.Exit(0)
	waitFor(t, "the running task", func() bool { return running.Status() == state.TaskSucceeded })
	waitFor(t, "the reload", func() bool { return len(runner.Calls("list")) == before+1 })
	// The cancelled task never ran, so it did not change anything.
	if got := len(runner.Calls("list")); got != before+1 {
		t.Errorf("got %d calls", got)
	}
}

func TestModelDeletesPendingPluginCopiesWhenDisposed(t *testing.T) {
	runner := testsupport.FixtureRunner()
	// A task that keeps the queue busy, so that the plugin waits behind it.
	runner.OnStart("update --all", testsupport.NewFakeRunningProcess)
	home := t.TempDir()
	model := newTestModel(runner, home)
	model.Start()
	original := home + "/mine.lua"
	if err := os.WriteFile(original, []byte("PLUGIN = {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	busy := model.Run(model.Cli.UpdateAllPlugins())
	task, err := model.AddPluginFromSource(original, "")
	if err != nil {
		t.Fatal(err)
	}
	source := task.Command.Arguments[len(task.Command.Arguments)-1]
	if _, err := os.Stat(source); err != nil || task.Status() != state.TaskQueued || busy == task {
		t.Fatalf("the copy waits for its turn: %v %v", err, task.Status())
	}
	model.Dispose()
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Error("the copy must be deleted")
	}
	if _, err := os.Stat(original); err != nil {
		t.Error("the original must survive")
	}
}
