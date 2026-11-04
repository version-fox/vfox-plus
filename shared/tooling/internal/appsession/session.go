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

// Package appsession starts the built application against a fresh home directory
// and drives it through the automation plugin of mygo, as the end-to-end
// scenarios (e2e-run), the memory measurement (measure) and the screenshots
// share it. The application is built with the tag e2e (vfxtool build --e2e),
// which links the plugin; MYGO_AUTOMATION names the socket that it listens on.
package appsession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/automation"
)

// Config says how the application starts.
type Config struct {
	// Name is what the session is for (e2e, measure, screenshots): the directory of its
	// sandbox is named after it, and the path of the fake vfox, which the screenshots
	// show, with it.
	Name string
	// Command is the built application, with its arguments.
	Command []string
	// Vfox is the vfox executable that the settings name. Without one, the
	// session makes a fake of its own: a copy of the running vfxtool, which
	// answers as vfox when it is run under that name, with the captured output in
	// Fixtures.
	Vfox string
	// FakeVfox is the executable to copy as the fake vfox instead of vfxtool.
	FakeVfox string
	// Fixtures is the directory of the captured output (shared/fixtures/vfox-1.0.13)
	// that the fake vfox replays.
	Fixtures string
	// Env adds NAME=value variables to the environment of the application.
	Env []string
	// Settings adds keys to the settings that the application starts with: the language
	// and the vfox executable are always there.
	Settings map[string]any
	// Setup prepares the home directory of the application before it starts, and may add
	// variables to its environment: what the desktop of the user has, for a scenario that
	// needs more than the settings.
	Setup func(home string) (env []string, err error)
}

// Session is the application that is running, and the connection to it.
type Session struct {
	// Sandbox is the directory of the session: the home directory, the fake vfox,
	// the socket and the log of the application.
	Sandbox string
	// Home is the home directory of the application.
	Home string
	// Vfox is the vfox executable of the settings of the application.
	Vfox string
	// Log is the file with what the application wrote to its output.
	Log string

	// control is the file through which the tools ask the application to do what a person
	// does: see Tell.
	control string

	client *automation.Client
	env    []string
	cmd    *exec.Cmd
	// exited is closed when the application has ended, and exit is how.
	exited chan struct{}
	exit   error
}

// reportTo is where the progress of a scenario goes: the output of vfxtool, which
// the log of the CI shows.
var reportTo io.Writer = os.Stdout

// Report writes a line of the progress of the scenario, as `e2e: …`.
func Report(format string, args ...any) {
	fmt.Fprintf(reportTo, "e2e: "+format+"\n", args...)
}

// How long the application has to open its window, and how long it has to end when
// it is asked to, before it is killed. They are variables so that tests need not
// wait.
var (
	startTimeout   = 60 * time.Second
	terminateGrace = 5 * time.Second
)

// Start starts the application and connects to it when it has opened its
// window. Close the session when it is no longer needed.
func Start(cfg Config) (*Session, error) {
	if len(cfg.Command) == 0 {
		return nil, errors.New("no application to start")
	}
	name := cfg.Name
	if name == "" {
		name = "e2e"
	}
	sandbox, err := os.MkdirTemp("", "vfox-plus-"+name+".*")
	if err != nil {
		return nil, err
	}
	session := &Session{Sandbox: sandbox, Home: filepath.Join(sandbox, "home"), Log: filepath.Join(sandbox, "app.log")}
	if err := session.start(cfg); err != nil {
		session.Close()
		return nil, err
	}
	return session, nil
}

// SettingsDir is the directory of the settings of the application with the home directory:
// where the application looks for them on this system.
func SettingsDir(home string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Roaming", "vfox-plus")
	}
	return filepath.Join(home, ".config", "vfox-plus")
}

// fakeVfoxName is the name of the fake vfox: vfxtool answers as vfox when it runs under that
// name, and Windows runs only what has its extension.
func fakeVfoxName() string {
	if runtime.GOOS == "windows" {
		return "vfox.exe"
	}
	return "vfox"
}

func (s *Session) start(cfg Config) error {
	if err := os.MkdirAll(SettingsDir(s.Home), 0o755); err != nil {
		return err
	}
	s.Vfox = cfg.Vfox
	if s.Vfox == "" {
		s.Vfox = filepath.Join(s.Sandbox, fakeVfoxName())
		if err := materializeFakeVfox(s.Vfox, cfg.FakeVfox); err != nil {
			return fmt.Errorf("install the fake vfox: %w", err)
		}
	}
	settings := map[string]any{"language": "english", "vfoxExecutable": s.Vfox}
	for key, value := range cfg.Settings {
		settings[key] = value
	}
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(SettingsDir(s.Home), "settings.json"), append(content, '\n'), 0o644); err != nil {
		return err
	}
	socket := filepath.Join(s.Sandbox, "automation.sock")
	s.control = filepath.Join(s.Sandbox, "control")
	env := environment(s.Home, socket, s.Vfox, cfg)
	env = append(env, "VFOX_PLUS_E2E_CONTROL="+s.control)
	if cfg.Setup != nil {
		more, err := cfg.Setup(s.Home)
		if err != nil {
			return fmt.Errorf("prepare the home directory: %w", err)
		}
		env = append(env, more...)
	}
	s.env = env
	logFile, err := os.OpenFile(s.Log, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()
	s.cmd = exec.Command(cfg.Command[0], cfg.Command[1:]...) //nolint:gosec // The application of the test.
	s.cmd.Env = env
	s.cmd.Stdout, s.cmd.Stderr = logFile, logFile
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start the application: %w", err)
	}
	// Reap it in the background: that is how its end is noticed.
	s.exited = make(chan struct{})
	go func() {
		s.exit = s.cmd.Wait()
		close(s.exited)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()
	go func() {
		// An application that ends before it listens is no use waiting for.
		select {
		case <-s.exited:
			cancel()
		case <-ctx.Done():
		}
	}()
	client, err := automation.Dial(ctx, socket)
	if err != nil {
		return s.explain(err)
	}
	s.client = client
	if _, err := client.WaitForWindow(ctx); err != nil {
		return s.explain(err)
	}
	return nil
}

// explain adds what the application wrote to a failure of it: it is often why.
func (s *Session) explain(err error) error {
	data, _ := os.ReadFile(s.Log)
	text := strings.TrimSpace(string(data))
	if text == "" {
		return err
	}
	const limit = 4000
	if len(text) > limit {
		text = "…" + text[len(text)-limit:]
	}
	return fmt.Errorf("%w\nThe application wrote:\n%s", err, text)
}

// environment is the environment of the application: that of vfxtool without the
// variables that would send vfox to the home of the user, with the sandbox in
// their place.
func environment(home, socket, vfox string, cfg Config) []string {
	var env []string
	for _, kv := range os.Environ() {
		switch {
		case strings.HasPrefix(kv, "VFOX_HOME="), strings.HasPrefix(kv, "XDG_CONFIG_HOME="), strings.HasPrefix(kv, "MYGO_AUTOMATION="):
			continue
		}
		env = append(env, kv)
	}
	env = append(env,
		"HOME="+home,
		"LANG=C.UTF-8",
		"MYGO_AUTOMATION="+socket,
		"VFOX_PLUS_E2E_VFOX="+vfox,
	)
	if runtime.GOOS == "windows" {
		// Windows looks for the home and the settings of a user elsewhere than the variable HOME.
		env = append(env,
			"USERPROFILE="+home,
			"APPDATA="+filepath.Join(home, "AppData", "Roaming"),
			"LOCALAPPDATA="+filepath.Join(home, "AppData", "Local"),
		)
	}
	if cfg.Fixtures != "" {
		env = append(env, "VFOX_PLUS_FIXTURES="+cfg.Fixtures, "__VFOX_PID=1", "__VFOX_SHELL=bash")
	}
	return append(env, cfg.Env...)
}

// materializeFakeVfox installs the fake vfox at dest: source, or the running
// vfxtool, which answers vfox commands when it is run under the name vfox (see
// cmd/vfxtool).
func materializeFakeVfox(dest, source string) error {
	if source == "" {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		source = self
	}
	in, err := os.Open(source) //nolint:gosec // The executable that vfxtool was given or is.
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec // The fake vfox must run.
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// PID is the process of the application.
func (s *Session) PID() int { return s.cmd.Process.Pid }

// Client is the connection to the application.
func (s *Session) Client() *automation.Client { return s.client }

// Alive reports whether the application still runs.
func (s *Session) Alive() bool {
	select {
	case <-s.exited:
		return false
	default:
		return true
	}
}

// Tell asks the application to do what a person does, through the file that it follows:
// "close" closes its window, "show" chooses *Open vfox+* of its icon in the tray and "quit"
// *Quit vfox+* of it. The application removes the file when it has taken the word.
func (s *Session) Tell(word string) error {
	temporary := s.control + ".tmp"
	if err := os.WriteFile(temporary, []byte(word+"\n"), 0o600); err != nil {
		return err
	}
	// Windows refuses to replace a file that the application has open at that moment, which is
	// as long as it takes it to read the word that it was told before: a few tries are enough.
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		if err = os.Rename(temporary, s.control); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

// HasWindow reports whether the application has a window open.
func (s *Session) HasWindow() bool {
	windows, err := s.client.Windows()
	return err == nil && len(windows) > 0
}

// WaitExit waits for the application to end, and says whether it did in time.
func (s *Session) WaitExit(timeout time.Duration) bool {
	select {
	case <-s.exited:
		return true
	case <-time.After(timeout):
		return false
	}
}

// ExitError is how the application ended: nil when it ended by itself, with the code 0. It is
// only to be read once the application has ended (see WaitExit).
func (s *Session) ExitError() error { return s.exit }

// LogText is what the application has written to its output so far.
func (s *Session) LogText() string {
	data, _ := os.ReadFile(s.Log)
	return string(data)
}

// Close ends the application and removes the sandbox.
func (s *Session) Close() {
	if s.client != nil {
		_ = s.client.Close()
	}
	if s.exited != nil && s.Alive() {
		// Windows has no signal to end a program with: it is killed at once.
		if runtime.GOOS != "windows" {
			_ = s.cmd.Process.Signal(syscall.SIGTERM)
		} else {
			_ = s.cmd.Process.Kill()
		}
		select {
		case <-s.exited:
		case <-time.After(terminateGrace):
			_ = s.cmd.Process.Kill()
			<-s.exited
		}
	}
	for attempt := 1; attempt <= 5; attempt++ {
		if err := os.RemoveAll(s.Sandbox); err == nil {
			return
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	fmt.Fprintf(os.Stderr, "Could not remove %s\n", s.Sandbox)
}

// kill ends the application at once.
func (s *Session) kill() {
	if s.exited != nil && s.Alive() {
		_ = s.cmd.Process.Kill()
	}
}

// RunScenario starts the application, runs the scenario against it and ends it.
// The scenario has VFOX_PLUS_E2E_TIMEOUT seconds (300 by default); the application
// is killed after that.
func RunScenario(cfg Config, name string, scenario func(*Session)) error {
	timeout, err := Timeout()
	if err != nil {
		return err
	}
	session, err := Start(cfg)
	if err != nil {
		return err
	}
	defer session.Close()
	var late atomic.Bool
	timer := time.AfterFunc(timeout, func() {
		late.Store(true)
		session.kill()
	})
	defer timer.Stop()
	err = session.Run(name, scenario)
	if late.Load() {
		return errors.New("The scenario did not finish in time.")
	}
	return err
}

// Timeout is how long a scenario may take: VFOX_PLUS_E2E_TIMEOUT, in seconds.
func Timeout() (time.Duration, error) {
	text := os.Getenv("VFOX_PLUS_E2E_TIMEOUT")
	if text == "" {
		return 300 * time.Second, nil
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, fmt.Errorf("invalid VFOX_PLUS_E2E_TIMEOUT %q: %w", text, err)
	}
	if seconds < 0 {
		return 0, fmt.Errorf("invalid VFOX_PLUS_E2E_TIMEOUT %q: must not be negative", text)
	}
	return time.Duration(seconds) * time.Second, nil
}

// searchUp looks for the repository from dir upwards: the first directory that has
// shared/fixtures.
func searchUp(dir string) (string, bool) {
	for {
		if info, err := os.Stat(filepath.Join(dir, "shared", "fixtures")); err == nil && info.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// FindRepoRoot locates the repository: the directory with shared/fixtures that is
// the working directory or above it, else above the executable.
func FindRepoRoot() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		if root, ok := searchUp(cwd); ok {
			return root, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if root, ok := searchUp(filepath.Dir(exe)); ok {
			return root, nil
		}
	}
	return "", errors.New("cannot locate the repository (no shared/fixtures above the working directory)")
}

// FixturesDir is the directory of the captured output of vfox in a repository.
func FixturesDir(root string) string { return filepath.Join(root, "shared", "fixtures", "vfox-1.0.13") }

// Vfoxcmd runs vfox as the tests see it: in the environment of the application,
// from the root directory, without colours. It returns what vfox printed.
func (s *Session) Vfoxcmd(args ...string) string {
	command := exec.Command(s.Vfox, args...) //nolint:gosec // The vfox of the test.
	command.Dir = "/"
	command.Env = append(append([]string{}, s.env...), "NO_COLOR=1")
	output, _ := command.Output()
	return string(output)
}
