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

package vfox_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
)

func testConfig() vfox.VfoxLaunchConfig {
	return vfox.VfoxLaunchConfig{
		Executable: "/usr/local/bin/vfox",
		BaseEnvironment: map[string]string{
			"HOME": "/home/user", "PATH": "/usr/bin",
			"__VFOX_SHELL": "bash", "__VFOX_PID": "42", "__VFOX_CURTMPPATH": "/home/user/.vfox/tmp/x",
		},
		ExtraEnvironment: map[string]string{"VFOX_NODEJS_MIRROR": "https://mirror", "PATH": "/opt/bin"},
		NeutralDirectory: "/",
	}
}

func TestEnvironment(t *testing.T) {
	config := testConfig()
	environment := vfox.VfoxEnvironment(config, false)
	for _, key := range []string{"__VFOX_SHELL", "__VFOX_PID", "__VFOX_CURTMPPATH"} {
		if _, ok := environment[key]; ok {
			t.Errorf("%s must be removed", key)
		}
	}
	if environment["NO_COLOR"] != "1" || environment["HOME"] != "/home/user" {
		t.Errorf("got %#v", environment)
	}
	if environment["PATH"] != "/opt/bin" || environment["VFOX_NODEJS_MIRROR"] != "https://mirror" {
		t.Errorf("user variables must override inherited ones: %#v", environment)
	}
	if vfox.VfoxEnvironment(config, true)["__VFOX_SHELL"] != "bash" {
		t.Error("hook shell must be marked when asked to")
	}
}

type discardListener struct{}

func (discardListener) Stdout([]byte) {}
func (discardListener) Stderr([]byte) {}

func TestQueriesRunInTheNeutralDirectory(t *testing.T) {
	runner := testsupport.FixtureRunner()
	cli := vfox.NewClient(runner, testConfig())
	version, err := cli.Version()
	if err != nil || version != "1.0.13" {
		t.Fatalf("got %q %v", version, err)
	}
	if len(runner.Invocations()) != 1 {
		t.Fatalf("got %d invocations", len(runner.Invocations()))
	}
	call := runner.Invocations()[0]
	if call.Executable != "/usr/local/bin/vfox" || call.WorkingDirectory != "/" || call.Streamed {
		t.Errorf("got %#v", call)
	}
	if call.Environment["NO_COLOR"] != "1" {
		t.Errorf("got %#v", call.Environment)
	}
}

func TestVersionRejectsGarbage(t *testing.T) {
	runner := testsupport.FixtureRunner()
	runner.On("--version", core.ProcessOutput{Stdout: "hello\n"})
	cli := vfox.NewClient(runner, testConfig())
	if _, err := cli.Version(); err == nil {
		t.Error("expected an error")
	}
}

func TestInstalledSdks(t *testing.T) {
	runner := testsupport.FixtureRunner()
	cli := vfox.NewClient(runner, testConfig())
	got, err := cli.InstalledSdks()
	if err != nil {
		t.Fatal(err)
	}
	want := []core.InstalledSdk{
		{Name: "maven", Versions: []string{"3.9.16", "3.8.9"}, Current: "3.9.16", HasCurrent: true},
		{Name: "mvn2"},
		{Name: "nodejs", Versions: []string{"24.11.1", "22.21.0"}, Current: "24.11.1", HasCurrent: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v", got)
	}
}

func TestInstalledSdksEmpty(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("list", testsupport.FixtureOutput("list-empty", 1)).
		On("current", testsupport.FixtureOutput("current-empty", 0))
	cli := vfox.NewClient(runner, testConfig())
	got, err := cli.InstalledSdks()
	if err != nil || len(got) != 0 {
		t.Errorf("got %#v %v", got, err)
	}
}

func TestInstalledSdksFailure(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("list", core.ProcessOutput{ExitCode: 1, Stdout: "boom\n"})
	cli := vfox.NewClient(runner, testConfig())
	_, err := cli.InstalledSdks()
	var vfoxErr *vfox.VfoxError
	if !errors.As(err, &vfoxErr) || vfoxErr.Message != "boom" || vfoxErr.ExitCode != 1 {
		t.Errorf("got %#v", err)
	}
	if !reflect.DeepEqual(vfoxErr.Arguments, []string{"list"}) {
		t.Errorf("got %#v", vfoxErr.Arguments)
	}
}

func TestPluginInfo(t *testing.T) {
	runner := testsupport.FixtureRunner()
	cli := vfox.NewClient(runner, testConfig())
	info, err := cli.PluginInfo("nodejs")
	if err != nil || info.Version != "0.5.1" {
		t.Fatalf("got %#v %v", info, err)
	}
	call := runner.Invocations()[0]
	if !reflect.DeepEqual(call.Arguments[:3], []string{"info", "nodejs", "--format"}) {
		t.Errorf("got %q", call.Arguments)
	}
	if _, err := cli.PluginInfo("maven"); err == nil {
		t.Error("expected an error for output without fields")
	}
}

func TestSearchAndAvailable(t *testing.T) {
	runner := testsupport.FixtureRunner()
	cli := vfox.NewClient(runner, testConfig())
	versions, err := cli.SearchVersions("maven")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, version := range versions {
		if version.Version == "3.9.16" {
			found = true
		}
	}
	if !found {
		t.Errorf("got %#v", versions)
	}
	plugins, err := cli.AvailablePlugins()
	if err != nil || len(plugins) != 43 {
		t.Errorf("got %d %v", len(plugins), err)
	}
}

func TestSearchPassesItsArgumentsToThePlugin(t *testing.T) {
	const temurin = "Available versions:\n - 21.0.12.1+1-tem (LTS) (installed)\n - 25.0.4+9-tem (LTS, JavaFX)\n"
	runner := testsupport.NewFakeProcessRunner().
		On("search java tem", core.ProcessOutput{Stdout: temurin}).
		On("search java", core.ProcessOutput{Stdout: "Available versions:\n - 25.0.2+10\n"})
	cli := vfox.NewClient(runner, testConfig())
	versions, err := cli.SearchVersions("java", "tem")
	if err != nil || len(versions) != 2 || versions[0].Version != "21.0.12.1+1-tem" || !versions[0].Installed || versions[1].Note != "LTS, JavaFX" {
		t.Fatalf("got %#v %v", versions, err)
	}
	if call := runner.Invocations()[0]; !reflect.DeepEqual(call.Arguments, []string{"search", "java", "tem"}) || call.WorkingDirectory != "/" {
		t.Errorf("got %#v", call)
	}
	if versions, err := cli.SearchVersions("java"); err != nil || len(versions) != 1 {
		t.Errorf("without an argument: %#v %v", versions, err)
	}
	if call := runner.Invocations()[1]; !reflect.DeepEqual(call.Arguments, []string{"search", "java"}) {
		t.Errorf("got %#v", call)
	}
}

func TestSearchRefusesArgumentsThatAreNotPlain(t *testing.T) {
	runner := testsupport.NewFakeProcessRunner()
	cli := vfox.NewClient(runner, testConfig())
	for _, argument := range []string{"", "-h", "--version", "a b", "tem\n", "tem\t", "a\x00b"} {
		if _, err := cli.SearchVersions("java", argument); err == nil {
			t.Errorf("%q should be refused", argument)
		}
	}
	if _, err := cli.SearchVersions("--all", "tem"); err == nil {
		t.Error("the plugin name is checked as before")
	}
	if len(runner.Invocations()) != 0 {
		t.Errorf("nothing runs for a bad argument: %v", runner.Invocations())
	}
}

func TestConfiguration(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("config proxy.url http://proxy:8080", core.ProcessOutput{}).
		On("config --unset proxy.url", core.ProcessOutput{})
	cli := vfox.NewClient(runner, testConfig())
	entries, err := cli.ConfigEntries()
	if err != nil || len(entries) != 10 {
		t.Fatalf("got %d %v", len(entries), err)
	}
	if err := cli.SetConfig("proxy.url", "http://proxy:8080"); err != nil {
		t.Fatal(err)
	}
	if err := cli.UnsetConfig("proxy.url"); err != nil {
		t.Fatal(err)
	}
	var commands []string
	for _, call := range runner.Calls("config") {
		commands = append(commands, joinArgs(call.Arguments))
	}
	want := []string{"config --list", "config proxy.url http://proxy:8080", "config --unset proxy.url"}
	if !reflect.DeepEqual(commands, want) {
		t.Errorf("got %q", commands)
	}
}

func joinArgs(args []string) string {
	joined := ""
	for i, arg := range args {
		if i > 0 {
			joined += " "
		}
		joined += arg
	}
	return joined
}

func TestRejectedValue(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("config cache.availableHookDuration notaduration", testsupport.FixtureOutput("config-set-invalid-duration", 1))
	cli := vfox.NewClient(runner, testConfig())
	err := cli.SetConfig("cache.availableHookDuration", "notaduration")
	if err == nil || !strings.Contains(err.Error(), `time: invalid duration "notaduration"`) {
		t.Errorf("got %v", err)
	}
}

func TestEmptyFailureReportsExitCode(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("config --unset proxy.url", core.ProcessOutput{ExitCode: 3})
	cli := vfox.NewClient(runner, testConfig())
	if err := cli.UnsetConfig("proxy.url"); err == nil || err.Error() != "vfox exited with code 3" {
		t.Errorf("got %v", err)
	}
}

func TestInstallPath(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("info nodejs@24.11.1", testsupport.FixtureOutput("info-version-path", 0)).
		On("info nodejs@1.0.0", testsupport.FixtureOutput("info-version-not-installed", 0))
	cli := vfox.NewClient(runner, testConfig())
	path, ok, err := cli.InstallPath("nodejs", "24.11.1")
	if err != nil || !ok || path != "/tmp/vfox-sb/home/.vfox/cache/nodejs/v-24.11.1" {
		t.Errorf("got %q %v %v", path, ok, err)
	}
	if _, ok, err := cli.InstallPath("nodejs", "1.0.0"); err != nil || ok {
		t.Errorf("got %v %v", ok, err)
	}
}

func TestMissingExecutable(t *testing.T) {
	runner := testsupport.FixtureRunner()
	runner.SetMissingExecutable(true)
	cli := vfox.NewClient(runner, testConfig())
	_, err := cli.Version()
	var notFound *vfox.VfoxNotFoundError
	if !errors.As(err, &notFound) || notFound.Executable != "/usr/local/bin/vfox" {
		t.Errorf("got %#v", err)
	}
}

func TestInstallCommand(t *testing.T) {
	cli := vfox.NewClient(testsupport.FixtureRunner(), testConfig())
	command, err := cli.Install("nodejs", "24.11.1")
	if err != nil {
		t.Fatal(err)
	}
	if command.Operation != vfox.OperationInstall || command.Subject() != "nodejs@24.11.1" ||
		command.CommandLine() != "vfox install --yes nodejs@24.11.1" || command.WorkingDirectory != "" {
		t.Errorf("got %#v", command)
	}
}

func TestUseCommands(t *testing.T) {
	cli := vfox.NewClient(testsupport.FixtureRunner(), testConfig())
	command, err := cli.Use("nodejs", "24.11.1", core.UseScopeGlobal, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(command.Arguments, []string{"use", "--global", "nodejs@24.11.1"}) || !command.RequiresHookEnvironment {
		t.Errorf("got %#v", command)
	}
	command, err = cli.Use("java", "21", core.UseScopeProject, "/work/app")
	if err != nil || command.WorkingDirectory != "/work/app" {
		t.Errorf("got %#v %v", command, err)
	}
	if _, err := cli.Use("java", "21", core.UseScopeProject, ""); err == nil {
		t.Error("project scope requires a directory")
	}
	if _, err := cli.Unuse("java", core.UseScopeProject, " "); err == nil {
		t.Error("project scope requires a directory")
	}
}

func TestOtherCommands(t *testing.T) {
	cli := vfox.NewClient(testsupport.FixtureRunner(), testConfig())
	unuse, _ := cli.Unuse("java", core.UseScopeGlobal, "")
	if !reflect.DeepEqual(unuse.Arguments, []string{"unuse", "--global", "java"}) {
		t.Errorf("got %q", unuse.Arguments)
	}
	uninstall, _ := cli.Uninstall("java", "21")
	if !reflect.DeepEqual(uninstall.Arguments, []string{"uninstall", "java@21"}) {
		t.Errorf("got %q", uninstall.Arguments)
	}
	remove, _ := cli.RemovePlugin("java")
	if !reflect.DeepEqual(remove.Arguments, []string{"remove", "--yes", "java"}) {
		t.Errorf("got %q", remove.Arguments)
	}
	update, _ := cli.UpdatePlugin("java")
	if !reflect.DeepEqual(update.Arguments, []string{"update", "java"}) {
		t.Errorf("got %q", update.Arguments)
	}
	if all := cli.UpdateAllPlugins(); !reflect.DeepEqual(all.Arguments, []string{"update", "--all"}) || all.Subject() != "" {
		t.Errorf("got %#v", all)
	}
}

func TestAddPlugin(t *testing.T) {
	cli := vfox.NewClient(testsupport.FixtureRunner(), testConfig())
	command, err := cli.AddPlugin("nodejs", "", "")
	if err != nil || !reflect.DeepEqual(command.Arguments, []string{"add", "nodejs"}) {
		t.Errorf("got %#v %v", command, err)
	}
	custom, err := cli.AddPlugin("", " https://x/p.zip ", "mine")
	if err != nil || !reflect.DeepEqual(custom.Arguments, []string{"add", "--source", "https://x/p.zip", "--alias", "mine"}) || custom.Sdk != "mine" {
		t.Errorf("got %#v %v", custom, err)
	}
	bySource, err := cli.AddPlugin("", "/p.lua", "")
	if err != nil || bySource.Sdk != "" {
		t.Errorf("got %#v %v", bySource, err)
	}
	if _, err := cli.AddPlugin("  ", "", ""); err == nil {
		t.Error("expected an error")
	}
}

func TestNameValidation(t *testing.T) {
	cli := vfox.NewClient(testsupport.FixtureRunner(), testConfig())
	for name, version := range map[string]string{"--all": "1", "java": "-1", "java2": "1 2", "java3": "a@b", "java4": ""} {
		if _, err := cli.Install(name, version); err == nil {
			t.Errorf("expected an error for %q %q", name, version)
		}
	}
	if !vfox.IsValidPluginName("my-sdk.2") || vfox.IsValidPluginName("-x") {
		t.Error("name validation is wrong")
	}
	if err := cli.SetConfig("proxy..url", "x"); err == nil {
		t.Error("expected an error for a malformed key")
	}
}

func TestStart(t *testing.T) {
	runner := testsupport.FixtureRunner()
	cli := vfox.NewClient(runner, testConfig())
	command, err := cli.Use("java", "21", core.UseScopeProject, "/work/app")
	if err != nil {
		t.Fatal(err)
	}
	process, err := cli.Start(command, discardListener{})
	if err != nil {
		t.Fatal(err)
	}
	if code := process.Wait(); code != 0 {
		t.Errorf("got %d", code)
	}
	call := runner.Invocations()[0]
	if !call.Streamed || call.WorkingDirectory != "/work/app" || call.Environment["__VFOX_SHELL"] != "bash" {
		t.Errorf("got %#v", call)
	}
	runner.SetMissingExecutable(true)
	if _, err := cli.Start(cli.UpdateAllPlugins(), discardListener{}); err == nil {
		t.Error("expected a not-found error")
	}
}
