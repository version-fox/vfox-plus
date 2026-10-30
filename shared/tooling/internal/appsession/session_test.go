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

package appsession

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/automation"
)

// TestHelperApplication is the application of the tests below, which start the test
// binary again to run it: it listens on the socket of the automation plugin and shows
// a page, until it is ended. APPSESSION_HELPER is serve, or ignore-term for an
// application that does not end when it is asked to. Run by itself, it does nothing.
func TestHelperApplication(t *testing.T) {
	mode := os.Getenv("APPSESSION_HELPER")
	if mode == "" {
		t.Skip("the application of the other tests")
	}
	if mode == "ignore-term" {
		signal.Ignore(syscall.SIGTERM)
	}
	fmt.Printf("HELPER HOME=%s LANG=%s VFOX=%s\n", os.Getenv("HOME"), os.Getenv("LANG"), os.Getenv("VFOX_PLUS_E2E_VFOX"))
	listener, err := net.Listen("unix", os.Getenv("MYGO_AUTOMATION"))
	if err != nil {
		t.Fatal(err)
	}
	app := &fakeApp{nodes: []automation.Node{node("1", "", "text", "Versions in use")}}
	go app.serve(listener)
	time.Sleep(time.Minute)
}

// helperConfig is the configuration of a session of the helper application. The
// sandboxes of the sessions are made in a directory of the test, which is short:
// the path of a socket is limited.
func helperConfig(t *testing.T, mode string) Config {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the application of these tests is a Unix process with a Unix socket")
	}
	dir, err := os.MkdirTemp("", "as")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Setenv("TMPDIR", dir)
	return Config{
		Name:    "demo",
		Command: []string{os.Args[0], "-test.run=^TestHelperApplication$"},
		Vfox:    "/bin/true",
		Env:     []string{"APPSESSION_HELPER=" + mode},
	}
}

// sandboxes lists what the sessions of a test left in their directory.
func sandboxes(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(os.Getenv("TMPDIR"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestRunScenarioStartsTheApplicationAndRemovesItsSandbox(t *testing.T) {
	out := captureReports(t)
	config := helperConfig(t, "serve")
	var sandbox string
	err := RunScenario(config, "helper", func(s *Session) {
		sandbox = s.Sandbox
		s.WaitForText("Versions in use")
		s.Check(s.Alive() && s.PID() > 0 && s.Client() != nil, "the application runs")
		s.Check(strings.HasPrefix(filepath.Base(s.Sandbox), "vfox-plus-demo."), "the sandbox is named after the session")

		settings, err := os.ReadFile(filepath.Join(s.Home, ".config", "vfox-plus", "settings.json"))
		s.Check(err == nil && strings.Contains(string(settings), `"language": "english"`) && strings.Contains(string(settings), `"vfoxExecutable": "/bin/true"`),
			"the settings choose English and name vfox")

		log, err := os.ReadFile(s.Log)
		s.Check(err == nil && strings.Contains(string(log), fmt.Sprintf("HELPER HOME=%s LANG=C.UTF-8 VFOX=/bin/true\n", s.Home)),
			"the application runs at the home of the sandbox, in the language of the tests, and learns the vfox to use")
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "e2e: scenario helper passed in ") {
		t.Errorf("the pass is not reported:\n%s", out)
	}
	if _, err := os.Stat(sandbox); !os.IsNotExist(err) {
		t.Errorf("the sandbox %s is left (%v)", sandbox, err)
	}
	if left := sandboxes(t); len(left) != 0 {
		t.Errorf("left in the directory of the sandboxes: %v", left)
	}
}

func TestRunScenarioReportsAFailureWithWhatTheWindowShows(t *testing.T) {
	out := captureReports(t)
	err := RunScenario(helperConfig(t, "serve"), "helper", func(s *Session) { s.Check(false, "the page is right") })
	if err == nil || err.Error() != "Check failed: the page is right" {
		t.Fatalf("got %v, want the failure of the scenario", err)
	}
	if !strings.Contains(out.String(), "e2e: scenario helper FAILED: Check failed: the page is right\ne2e: on screen:\ntext \"Versions in use\"\n") {
		t.Errorf("the failure is not reported with the screen:\n%s", out)
	}
}

func TestRunScenarioKillsTheApplicationWhenTheScenarioTakesTooLong(t *testing.T) {
	captureReports(t)
	t.Setenv("VFOX_PLUS_E2E_TIMEOUT", "1")
	start := time.Now()
	err := RunScenario(helperConfig(t, "serve"), "slow", func(s *Session) {
		s.WaitUntil("a thing that never shows", time.Hour, func(automation.Tree) bool { return false })
	})
	if err == nil || err.Error() != "The scenario did not finish in time." {
		t.Fatalf("got %v, want the scenario to be late", err)
	}
	if elapsed := time.Since(start); elapsed > 30*time.Second {
		t.Fatalf("took %v: the application was not killed", elapsed)
	}
}

func TestRunScenarioRefusesAnInvalidTimeoutBeforeStartingAnything(t *testing.T) {
	t.Setenv("VFOX_PLUS_E2E_TIMEOUT", "soon")
	err := RunScenario(Config{Command: []string{"/nonexistent"}}, "x", func(*Session) { t.Error("the scenario ran") })
	if err == nil || !strings.Contains(err.Error(), `invalid VFOX_PLUS_E2E_TIMEOUT "soon"`) {
		t.Fatalf("got %v", err)
	}
}

func TestTimeout(t *testing.T) {
	cases := map[string]struct {
		value string
		want  time.Duration
		fails bool
	}{
		"the default":     {"", 300 * time.Second, false},
		"seconds":         {"7", 7 * time.Second, false},
		"with spaces":     {" 9 ", 9 * time.Second, false},
		"none":            {"0", 0, false},
		"no number":       {"soon", 0, true},
		"a negative time": {"-1", 0, true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("VFOX_PLUS_E2E_TIMEOUT", c.value)
			got, err := Timeout()
			if (err != nil) != c.fails || got != c.want {
				t.Fatalf("Timeout() = %v, %v; want %v (fails %v)", got, err, c.want, c.fails)
			}
		})
	}
}

func TestStartFailsAtOnceWhenTheApplicationEndsAndSaysWhy(t *testing.T) {
	config := helperConfig(t, "serve")
	config.Command = []string{"sh", "-c", "echo cannot open display >&2; exit 1"}
	start := time.Now()
	session, err := Start(config)
	if err == nil || session != nil {
		t.Fatalf("Start = %v, %v; want a failure", session, err)
	}
	if !strings.Contains(err.Error(), "The application wrote:\ncannot open display") {
		t.Errorf("the failure does not say what the application wrote: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 30*time.Second {
		t.Errorf("took %v: Start waited for an application that had ended", elapsed)
	}
	if left := sandboxes(t); len(left) != 0 {
		t.Errorf("left in the directory of the sandboxes: %v", left)
	}
}

func TestStartGivesUpOnAnApplicationThatNeverOpensAWindow(t *testing.T) {
	before := startTimeout
	startTimeout = 300 * time.Millisecond
	t.Cleanup(func() { startTimeout = before })
	config := helperConfig(t, "serve")
	config.Command = []string{"sleep", "30"}
	start := time.Now()
	if _, err := Start(config); err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("got %v, want the end of the time to start", err)
	}
	// Start ended the application: it did not wait for the sleep to be over.
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("took %v: the application was not ended", elapsed)
	}
	if left := sandboxes(t); len(left) != 0 {
		t.Errorf("left in the directory of the sandboxes: %v", left)
	}
}

func TestStartFailures(t *testing.T) {
	if _, err := Start(Config{}); err == nil || err.Error() != "no application to start" {
		t.Errorf("without a command: %v", err)
	}
	config := helperConfig(t, "serve")
	config.Command = []string{"/nonexistent/application"}
	if _, err := Start(config); err == nil || !strings.Contains(err.Error(), "start the application") {
		t.Errorf("a command that does not exist: %v", err)
	}
	config = helperConfig(t, "serve")
	config.Vfox, config.FakeVfox = "", "/nonexistent/vfox"
	if _, err := Start(config); err == nil || !strings.Contains(err.Error(), "install the fake vfox") {
		t.Errorf("a fake vfox that does not exist: %v", err)
	}
	if left := sandboxes(t); len(left) != 0 {
		t.Errorf("left in the directory of the sandboxes: %v", left)
	}
}

func TestCloseKillsAnApplicationThatDoesNotEndWhenAsked(t *testing.T) {
	before := terminateGrace
	terminateGrace = 300 * time.Millisecond
	t.Cleanup(func() { terminateGrace = before })
	session, err := Start(helperConfig(t, "ignore-term"))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	session.Close()
	if session.Alive() {
		t.Fatal("the application is still alive")
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("Close took %v", elapsed)
	}
	// Closing again, and killing what has ended, change nothing.
	session.Close()
	session.kill()
}

func TestCloseOfASessionThatNeverStarted(t *testing.T) {
	dir := t.TempDir()
	sandbox := filepath.Join(dir, "sandbox")
	if err := os.MkdirAll(filepath.Join(sandbox, "home"), 0o755); err != nil {
		t.Fatal(err)
	}
	(&Session{Sandbox: sandbox}).Close()
	(&Session{}).kill()
	if _, err := os.Stat(sandbox); !os.IsNotExist(err) {
		t.Fatalf("the sandbox is left (%v)", err)
	}
}

func TestStartInstallsTheFakeVfoxAndRunsItAsTheApplicationSeesIt(t *testing.T) {
	config := helperConfig(t, "serve")
	script := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$VFOX_PLUS_FIXTURES|$NO_COLOR|$HOME|$*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	config.Vfox, config.FakeVfox, config.Fixtures = "", script, "/the/fixtures"
	session, err := Start(config)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if session.Vfox != filepath.Join(session.Sandbox, "vfox") {
		t.Fatalf("the fake vfox is at %s", session.Vfox)
	}
	if info, err := os.Stat(session.Vfox); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("the fake vfox cannot be run: %v, %v", info, err)
	}
	settings, _ := os.ReadFile(filepath.Join(session.Home, ".config", "vfox-plus", "settings.json"))
	if !strings.Contains(string(settings), fmt.Sprintf("%q", session.Vfox)) {
		t.Errorf("the settings do not name the fake vfox:\n%s", settings)
	}
	want := fmt.Sprintf("/the/fixtures|1|%s|a b\n", session.Home)
	if got := session.Vfoxcmd("a", "b"); got != want {
		t.Fatalf("Vfoxcmd = %q, want %q", got, want)
	}
}

func TestVfoxcmdReturnsWhatVfoxPrintedAlsoWhenItFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a shell script")
	}
	script := filepath.Join(t.TempDir(), "vfox")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho partial\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := &Session{Vfox: script, env: os.Environ()}
	if got := s.Vfoxcmd("current"); got != "partial\n" {
		t.Fatalf("Vfoxcmd = %q", got)
	}
}

func TestEnvironmentSendsVfoxToTheSandbox(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	t.Setenv("VFOX_HOME", "/home/user/.vfox")
	t.Setenv("XDG_CONFIG_HOME", "/home/user/.config")
	t.Setenv("MYGO_AUTOMATION", "/elsewhere.sock")
	t.Setenv("PATH_KEPT", "yes")
	// The shell of a developer who uses vfox has the variables of its hook; the test
	// must not depend on them. (Setenv restores what Unsetenv takes away.)
	for _, name := range []string{"VFOX_PLUS_FIXTURES", "__VFOX_PID", "__VFOX_SHELL"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}

	// The last of the variables with a name is the one that the application gets.
	value := func(env []string, name string) (string, int) {
		found, count := "", 0
		for _, kv := range env {
			if strings.HasPrefix(kv, name+"=") {
				found, count = strings.TrimPrefix(kv, name+"="), count+1
			}
		}
		return found, count
	}

	env := environment("/sandbox/home", "/sandbox/auto.sock", "/sandbox/vfox", Config{Fixtures: "/fixtures", Env: []string{"EXTRA=1"}})
	for _, name := range []string{"VFOX_HOME", "XDG_CONFIG_HOME"} {
		if _, count := value(env, name); count != 0 {
			t.Errorf("%s of the user is passed on", name)
		}
	}
	wants := map[string]string{
		"HOME": "/sandbox/home", "LANG": "C.UTF-8", "MYGO_AUTOMATION": "/sandbox/auto.sock", "VFOX_PLUS_E2E_VFOX": "/sandbox/vfox",
		"VFOX_PLUS_FIXTURES": "/fixtures", "__VFOX_PID": "1", "__VFOX_SHELL": "bash", "EXTRA": "1", "PATH_KEPT": "yes",
	}
	for name, want := range wants {
		if got, _ := value(env, name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if _, count := value(env, "MYGO_AUTOMATION"); count != 1 {
		t.Errorf("MYGO_AUTOMATION occurs %d times", count)
	}

	// Without fixtures the fake vfox is not played; and the configuration wins.
	env = environment("/sandbox/home", "/s", "/v", Config{Env: []string{"LANG=de_DE.UTF-8"}})
	for _, name := range []string{"VFOX_PLUS_FIXTURES", "__VFOX_PID", "__VFOX_SHELL"} {
		if _, count := value(env, name); count != 0 {
			t.Errorf("%s is set without fixtures", name)
		}
	}
	if got, _ := value(env, "LANG"); got != "de_DE.UTF-8" {
		t.Errorf("LANG = %q: the environment of the configuration comes last", got)
	}
}

func TestMaterializeFakeVfox(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "vfox")
	if err := materializeFakeVfox(dest, source); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(dest); string(data) != "#!/bin/sh\n" {
		t.Errorf("copied %q", data)
	}
	if info, _ := os.Stat(dest); runtime.GOOS != "windows" && info.Mode().Perm() != 0o755 {
		t.Errorf("mode %v, want 0755", info.Mode())
	}

	// Without a source, the running program is the fake: vfxtool answers as vfox.
	self := filepath.Join(dir, "self")
	if err := materializeFakeVfox(self, ""); err != nil {
		t.Fatal(err)
	}
	executable, _ := os.Executable()
	want, _ := os.Stat(executable)
	if got, _ := os.Stat(self); got == nil || got.Size() != want.Size() {
		t.Errorf("the copy of the running program has another size: %v", got)
	}

	if err := materializeFakeVfox(dest, filepath.Join(dir, "missing")); err == nil {
		t.Error("a source that is not there was copied")
	}
	if err := materializeFakeVfox(filepath.Join(dir, "no", "such", "directory", "vfox"), source); err == nil {
		t.Error("a destination in a directory that is not there was written")
	}
}

func TestFindRepoRoot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "shared", "tooling", "internal", "x")
	for _, dir := range []string{deep, filepath.Join(root, "shared", "fixtures")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got, ok := searchUp(deep); !ok || got != root {
		t.Fatalf("searchUp = %q, %v; want %q", got, ok, root)
	}
	// A file with the name is no directory of the fixtures.
	elsewhere := filepath.Join(root, "elsewhere", "shared")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(elsewhere, "fixtures"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, ok := searchUp(filepath.Dir(elsewhere)); !ok || got != root {
		t.Fatalf("searchUp from a file named fixtures = %q, %v; want the root above, %q", got, ok, root)
	}
	if _, ok := searchUp(string(filepath.Separator)); ok {
		t.Fatal("the root of the file system has the fixtures")
	}

	t.Chdir(deep)
	got, err := FindRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got, _ = filepath.EvalSymlinks(got); got != root {
		t.Fatalf("FindRepoRoot = %q, want %q", got, root)
	}
	if want := filepath.Join(root, "shared", "fixtures", "vfox-1.0.13"); FixturesDir(root) != want {
		t.Fatalf("FixturesDir = %q, want %q", FixturesDir(root), want)
	}

	// Away from every repository, and with the program in none, it is not found.
	t.Chdir(t.TempDir())
	cwd, _ := os.Getwd()
	executable, _ := os.Executable()
	_, nearby := searchUp(cwd)
	_, nearTheProgram := searchUp(filepath.Dir(executable))
	if !nearby && !nearTheProgram {
		if root, err := FindRepoRoot(); err == nil {
			t.Fatalf("FindRepoRoot found %s where there is no repository", root)
		}
	}
}
