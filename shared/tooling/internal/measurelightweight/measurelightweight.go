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

// Package measurelightweight implements `vfxtool measure-lightweight`: it starts
// the application with a fresh home directory and the fake vfox, with the web
// interface and the lightweight mode on, closes its window, pairs a browser, loads
// the inventory over server-sent events, and prints the resident (RSS) and
// proportional (PSS) set size of the application and all of its child processes at
// two measuring points: settled without its window (lightweight-idle), and settled
// with a paired browser (lightweight-loaded).
//
// With --tray it measures instead what the icon in the tray costs (see tray.go).
//
// It needs the end-to-end build of the application, which closes its window on
// SIGUSR1 and writes a pairing link to the file that VFOX_PLUS_E2E_LINK names, and a
// display, which the caller provides (xvfb-run); the application is a desktop
// application, and has no other way to start the web interface than its settings.
package measurelightweight

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
)

// sessionCookieValue reads the session cookie out of a Set-Cookie header.
// The value is passed back as a Cookie header by hand, so that pairing does
// not depend on cookie-jar handling of the Secure flag over plain HTTP.
func sessionCookieValue(header string) (string, bool) {
	for _, part := range strings.Split(header, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && name == "vfox-plus-session" && value != "" {
			return value, true
		}
	}
	return "", false
}

// settings are the settings of the measured application: the web interface is on, and
// the lightweight mode keeps the application running when its window is closed.
const settings = `{"webInterface":true,"lightweight":true}`

// Run implements `measure-lightweight [--output FILE] [--settle SECONDS] [--fixtures-root DIR] [--fake-vfox FILE] <application command...>`.
// The application command usually is the end-to-end build of the application.
func Run(args []string) error {
	flags := flag.NewFlagSet("measure-lightweight", flag.ContinueOnError)
	outputFlag := flags.String("output", "", "file collecting the memory(<point>) lines (also $VFOX_PLUS_MEASURE_OUTPUT)")
	settleFlag := flags.Int("settle", 10, "seconds to settle before each measuring point")
	fixturesRootFlag := flags.String("fixtures-root", "", "shared/fixtures directory (default: found via repository root)")
	fakeVfoxFlag := flags.String("fake-vfox", "", "explicit fake vfox executable (default: this vfxtool binary, executed as vfox)")
	trayFlag := flags.Bool("tray", false, "measure what the icon in the tray costs instead: run the application where nothing shows an icon and where a stand-in host does (Linux, in a desktop session), and fail when the icon costs more memory than it may")
	if err := flags.Parse(args); err != nil {
		return err
	}
	rest := flags.Args()
	if len(rest) == 0 {
		return errors.New("Usage: measure-lightweight [--output FILE] [--settle SECONDS] [--fixtures-root DIR] [--fake-vfox FILE] [--tray] <application command...>")
	}
	if *settleFlag < 0 {
		return errors.New("--settle must not be negative")
	}
	settle := time.Duration(*settleFlag) * time.Second
	output := *outputFlag
	if output == "" {
		output = os.Getenv("VFOX_PLUS_MEASURE_OUTPUT")
	}
	if output != "" {
		if dir := filepath.Dir(output); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		if err := os.WriteFile(output, nil, 0o644); err != nil {
			return err
		}
	}
	fixturesDir, err := resolveFixturesDir(*fixturesRootFlag, *fakeVfoxFlag)
	if err != nil {
		return err
	}
	if *trayFlag {
		return measureTray(rest, fixturesDir, *fakeVfoxFlag, settle)
	}
	homeDir, err := os.MkdirTemp("", "vfox-plus-measure-lightweight-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(homeDir) }()
	env, linkFile, err := measureEnv(homeDir, fixturesDir, *fakeVfoxFlag)
	if err != nil {
		return err
	}
	app, err := startApplication(rest, env)
	if err != nil {
		return err
	}
	command, done, stop := app.command, app.done, app.stop
	pairingURL, err := waitForLink(linkFile, done, 60*time.Second)
	if err != nil {
		stop()
		return err
	}
	// The window goes; the application stays, with the web interface it was asked for.
	if err := command.Process.Signal(closeSignal); err != nil {
		stop()
		return fmt.Errorf("cannot ask the application to close its window: %w", err)
	}
	report := func(point string) (string, error) {
		time.Sleep(settle)
		select {
		case err := <-done:
			return "", fmt.Errorf("the application exited early: %v", err)
		default:
		}
		return "memory (" + point + "): " + measure.MemoryOf(command.Process.Pid), nil
	}
	emit := func(line string) error {
		fmt.Println(line)
		if output == "" {
			return nil
		}
		file, err := os.OpenFile(output, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(file, line)
		return errors.Join(err, file.Close())
	}
	idle, err := report("lightweight-idle")
	if err != nil {
		stop()
		return err
	}
	if err := emit(idle); err != nil {
		stop()
		return err
	}
	events, err := pairAndLoad(pairingURL)
	if err != nil {
		stop()
		return err
	}
	defer func() { _ = events.Close() }()
	loaded, err := report("lightweight-loaded")
	_ = events.Close()
	stop()
	if err != nil {
		return err
	}
	return emit(loaded)
}

// application is the application that is running, and the way to end it.
type application struct {
	command *exec.Cmd
	// done receives how it ended, once.
	done chan error
}

// startApplication starts the application with the environment and its output on that of the tool.
func startApplication(argv, env []string) (*application, error) {
	command := exec.Command(argv[0], argv[1:]...) //nolint:gosec // The application that the caller names.
	command.Env = env
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	return &application{command: command, done: done}, nil
}

// stop ends the application at once, and waits until it has.
func (a *application) stop() {
	_ = a.command.Process.Kill()
	<-a.done
}

// waitForLink waits for the application to write its one-time pairing link to
// file, or to exit, or for the timeout to pass.
func waitForLink(file string, done <-chan error, timeout time.Duration) (string, error) {
	deadline := time.After(timeout)
	for {
		select {
		case err := <-done:
			return "", fmt.Errorf("the application exited before it wrote a pairing link: %v", err)
		case <-deadline:
			return "", errors.New("timed out waiting for the pairing link")
		case <-time.After(100 * time.Millisecond):
		}
		content, err := os.ReadFile(file) //nolint:gosec // The file is in the temporary directory of this run.
		if err != nil {
			continue
		}
		if link := strings.TrimSpace(string(content)); strings.Contains(link, "/auth/connect/") {
			return link, nil
		}
	}
}

// pairAndLoad pairs a browser through the one-time link, fetches the
// inventory (which loads everything the desktop would show) and opens the
// server-sent events stream past its first event. It returns the open stream.
func pairAndLoad(pairingURL string) (io.ReadCloser, error) {
	// The link answers with the session and a redirect to the interface: the cookie is in
	// the first answer, so the redirect is not followed.
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	pair, err := client.Get(pairingURL)
	if err != nil {
		return nil, err
	}
	var session string
	for _, header := range pair.Header["Set-Cookie"] {
		if value, ok := sessionCookieValue(header); ok {
			session = value
			break
		}
	}
	_, _ = io.Copy(io.Discard, pair.Body)
	_ = pair.Body.Close()
	if session == "" {
		return nil, errors.New("pairing did not issue a session cookie")
	}
	base := pairingURL[:strings.LastIndex(pairingURL, "/auth/connect/")]
	cookie := "vfox-plus-session=" + session
	inventory, err := get(client, base+"/api/inventory", cookie)
	if err != nil {
		return nil, err
	}
	defer func() { _ = inventory.Body.Close() }()
	var body struct {
		Complete bool `json:"complete"`
	}
	if err := json.NewDecoder(inventory.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("cannot read the inventory: %w", err)
	}
	if !body.Complete {
		return nil, errors.New("the inventory did not complete")
	}
	events, err := get(client, base+"/api/events", cookie)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(events.Body)
	deadline := time.Now().Add(30 * time.Second)
	for {
		if time.Now().After(deadline) {
			_ = events.Body.Close()
			return nil, errors.New("timed out waiting for server-sent events")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			_ = events.Body.Close()
			return nil, fmt.Errorf("cannot read server-sent events: %w", err)
		}
		if strings.HasPrefix(line, "event:") {
			return events.Body, nil
		}
	}
}

func get(client *http.Client, url, cookie string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Cookie", cookie)
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, response.Status)
	}
	return response, nil
}

// measureEnv builds the environment of the measured application: a fresh home
// directory with the settings of the measurement, the fake vfox first on PATH, the
// fixtures it replays, a runtime directory of its own, so that no other application
// of the user is the one that it starts beside, and the file that gets its pairing
// link. It returns that file.
func measureEnv(homeDir, fixturesDir, fakeVfox string) (env []string, linkFile string, err error) {
	fake := fakeVfox
	if fake == "" {
		executable, err := os.Executable()
		if err != nil {
			return nil, "", err
		}
		fake = executable
	}
	binDir := filepath.Join(homeDir, "bin")
	runtimeDir := filepath.Join(homeDir, "run")
	settingsDir := filepath.Join(homeDir, ".config", "vfox-plus")
	for _, dir := range []string{binDir, runtimeDir, settingsDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, "", err
		}
	}
	content, err := os.ReadFile(fake) //nolint:gosec // The fake vfox is this tool or the file that the caller named.
	if err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(filepath.Join(binDir, "vfox"), content, 0o755); err != nil { //nolint:gosec // The fake vfox is executable.
		return nil, "", err
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(settings), 0o600); err != nil {
		return nil, "", err
	}
	linkFile = filepath.Join(homeDir, "link.txt")
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "VFOX_HOME=") || strings.HasPrefix(kv, "HOME=") || strings.HasPrefix(kv, "XDG_CONFIG_HOME=") ||
			strings.HasPrefix(kv, "XDG_RUNTIME_DIR=") || strings.HasPrefix(kv, "XDG_CACHE_HOME=") ||
			strings.HasPrefix(kv, "__VFOX_SHELL=") || strings.HasPrefix(kv, "__VFOX_PID=") ||
			strings.HasPrefix(kv, "__VFOX_CURTMPPATH=") || strings.HasPrefix(kv, "__VFOX_INITIALIZED=") ||
			strings.HasPrefix(kv, "VFOX_PLUS_FIXTURES=") || strings.HasPrefix(kv, "VFOX_PLUS_E2E_LINK=") ||
			strings.HasPrefix(kv, "NO_COLOR=") {
			continue
		}
		if strings.HasPrefix(kv, "PATH=") {
			env = append(env, "PATH="+binDir+string(os.PathListSeparator)+strings.TrimPrefix(kv, "PATH="))
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"HOME="+homeDir,
		"XDG_RUNTIME_DIR="+runtimeDir,
		"NO_COLOR=1",
		"VFOX_PLUS_FIXTURES="+fixturesDir,
		"VFOX_PLUS_E2E_LINK="+linkFile,
	), linkFile, nil
}

// resolveFixturesDir locates the captured vfox output the fake vfox replays:
// the --fixtures-root flag (a vfox-<version> directory, or the shared
// fixtures directory holding one), or shared/fixtures found by walking up
// from the working directory.
func resolveFixturesDir(flag, fakeVfox string) (string, error) {
	if flag != "" {
		return fixturesDirFromRoot(filepath.Clean(flag)), nil
	}
	if fakeVfox != "" {
		return fixturesDirFromRoot(filepath.Dir(fakeVfox)), nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "shared", "fixtures")); err == nil && info.IsDir() {
			return fixturesDirFromRoot(filepath.Join(dir, "shared", "fixtures")), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot locate shared/fixtures above the working directory: pass --fixtures-root DIR")
		}
		dir = parent
	}
}

// fixturesDirFromRoot resolves a --fixtures-root value to the directory with
// the captured outputs: the value itself when it already names a vfox-<version>
// directory, else its vfox-1.1.0 child.
func fixturesDirFromRoot(root string) string {
	if strings.HasPrefix(filepath.Base(root), "vfox-") {
		return root
	}
	return filepath.Join(root, "vfox-1.1.0")
}
