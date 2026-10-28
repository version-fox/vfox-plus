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

// Package measureserve implements `vfxtool measure-serve`: it starts the
// headless server with a fresh home directory and the fake vfox, pairs a
// browser, loads the inventory over server-sent events, and prints the
// resident (RSS) and proportional (PSS) set size of the server and all of its
// child processes at two measuring points: settled idle, and settled with a
// paired browser. Like `vfxtool measure`, it needs nothing but the binary and
// a loopback interface; unlike it, no display is involved.
package measureserve

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
	"regexp"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
)

// pairingLink finds the one-time pairing link the server prints at start.
var pairingLink = regexp.MustCompile(`http://127\.0\.0\.1:\d+/auth/connect/[0-9a-f]+`)

// pairingURLFromLine extracts the pairing link from a line of the server log.
func pairingURLFromLine(line string) (string, bool) {
	url := pairingLink.FindString(line)
	return url, url != ""
}

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

// Run implements `measure-serve [--output FILE] [--settle SECONDS] [--fixtures-root DIR] [--fake-vfox FILE] <server command...>`.
// The server command usually is the headless binary followed by
// `serve --bind 127.0.0.1 --port 0`.
func Run(args []string) error {
	flags := flag.NewFlagSet("measure-serve", flag.ContinueOnError)
	outputFlag := flags.String("output", "", "file collecting the memory(<point>) lines (also $VFOX_PLUS_MEASURE_OUTPUT)")
	settleFlag := flags.Int("settle", 10, "seconds to settle before each measuring point")
	fixturesRootFlag := flags.String("fixtures-root", "", "shared/fixtures directory (default: found via repository root)")
	fakeVfoxFlag := flags.String("fake-vfox", "", "explicit fake vfox executable (default: this vfxtool binary, executed as vfox)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	rest := flags.Args()
	if len(rest) == 0 {
		return errors.New("Usage: measure-serve [--output FILE] [--settle SECONDS] [--fixtures-root DIR] [--fake-vfox FILE] <server command...>")
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
	homeDir, err := os.MkdirTemp("", "vfox-plus-measure-serve-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(homeDir) }()
	fixturesDir, err := resolveFixturesDir(*fixturesRootFlag, *fakeVfoxFlag)
	if err != nil {
		return err
	}
	env, err := measureEnv(homeDir, fixturesDir, *fakeVfoxFlag)
	if err != nil {
		return err
	}
	command := exec.Command(rest[0], rest[1:]...)
	command.Env = env
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	command.Stderr = command.Stdout
	if err := command.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	lines := make(chan string, 64)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	pairingURL, err := waitForPairingURL(lines, done, 60*time.Second)
	if err != nil {
		_ = command.Process.Kill()
		<-done
		return err
	}
	report := func(point string) (string, error) {
		time.Sleep(settle)
		select {
		case err := <-done:
			return "", fmt.Errorf("the server exited early: %v", err)
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
	idle, err := report("serve-idle")
	if err != nil {
		_ = command.Process.Kill()
		<-done
		return err
	}
	if err := emit(idle); err != nil {
		_ = command.Process.Kill()
		<-done
		return err
	}
	events, err := pairAndLoad(pairingURL)
	if err != nil {
		_ = command.Process.Kill()
		<-done
		return err
	}
	defer func() { _ = events.Close() }()
	loaded, err := report("serve-loaded")
	_ = events.Close()
	_ = command.Process.Kill()
	<-done
	if err != nil {
		return err
	}
	return emit(loaded)
}

// waitForPairingURL reads server log lines until one holds the pairing link,
// or the server exits, or the timeout passes.
func waitForPairingURL(lines <-chan string, done <-chan error, timeout time.Duration) (string, error) {
	deadline := time.After(timeout)
	for {
		select {
		case err := <-done:
			return "", fmt.Errorf("the server exited before printing a pairing link: %v", err)
		case <-deadline:
			return "", errors.New("timed out waiting for the pairing link")
		case line, ok := <-lines:
			if !ok {
				select {
				case err := <-done:
					return "", fmt.Errorf("the server exited before printing a pairing link: %v", err)
				default:
					return "", errors.New("the server closed its output before printing a pairing link")
				}
			}
			if url, ok := pairingURLFromLine(line); ok {
				return url, nil
			}
		}
	}
}

// pairAndLoad pairs a browser through the one-time link, fetches the
// inventory (which loads everything the desktop would show) and opens the
// server-sent events stream past its first event. It returns the open stream.
func pairAndLoad(pairingURL string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 30 * time.Second}
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

// measureEnv builds the environment of the measured server: a fresh home
// directory, the fake vfox first on PATH, and the fixtures it replays,
// mirroring how the GUI drives vfox.
func measureEnv(homeDir, fixturesDir, fakeVfox string) ([]string, error) {
	fake := fakeVfox
	if fake == "" {
		executable, err := os.Executable()
		if err != nil {
			return nil, err
		}
		fake = executable
	}
	binDir := filepath.Join(homeDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return nil, err
	}
	content, err := os.ReadFile(fake)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(binDir, "vfox"), content, 0o755); err != nil {
		return nil, err
	}
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "VFOX_HOME=") || strings.HasPrefix(kv, "XDG_CONFIG_HOME=") ||
			strings.HasPrefix(kv, "__VFOX_SHELL=") || strings.HasPrefix(kv, "__VFOX_PID=") ||
			strings.HasPrefix(kv, "__VFOX_CURTMPPATH=") || strings.HasPrefix(kv, "__VFOX_INITIALIZED=") ||
			strings.HasPrefix(kv, "VFOX_PLUS_FIXTURES=") || strings.HasPrefix(kv, "NO_COLOR=") {
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
		"NO_COLOR=1",
		"VFOX_PLUS_FIXTURES="+fixturesDir,
	), nil
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
// directory, else its vfox-1.0.13 child.
func fixturesDirFromRoot(root string) string {
	if strings.HasPrefix(filepath.Base(root), "vfox-") {
		return root
	}
	return filepath.Join(root, "vfox-1.0.13")
}
