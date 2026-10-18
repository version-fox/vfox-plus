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

package host

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func shell(script string) Launch {
	return Launch{
		Executable:       "sh",
		Arguments:        []string{"-c", script},
		WorkingDirectory: "/",
		Environment:      map[string]string{"ONLY": "this", "PATH": "/usr/bin:/bin"},
	}
}

func TestRunUsesExactlyTheGivenEnvironmentAndDirectory(t *testing.T) {
	result, err := Run(shell(`echo "$ONLY:$HOME:$(pwd)"; echo err >&2; exit 3`))
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 3 || result.Stdout != "this::/\n" || result.Stderr != "err\n" {
		t.Fatalf("unexpected result %+v", result)
	}
}

func TestRunClosesStandardInput(t *testing.T) {
	result, err := Run(shell("cat; echo done"))
	if err != nil || result.Stdout != "done\n" {
		t.Fatalf("got %+v, %v", result, err)
	}
}

func TestRunReplacesMalformedOutput(t *testing.T) {
	result, err := Run(shell(`printf 'a\377b'`))
	if err != nil || result.Stdout != "a�b" {
		t.Fatalf("got %q, %v", result.Stdout, err)
	}
}

func TestRunLooksUpNamesInTheChildPath(t *testing.T) {
	launch := shell("true")
	launch.Environment["PATH"] = "/nonexistent"
	_, err := Run(launch)
	var start *StartError
	if !errors.As(err, &start) {
		t.Fatalf("expected a StartError, got %v", err)
	}
	launch.Executable = "/nonexistent/vfox"
	if _, err := Run(launch); err == nil {
		t.Fatal("a missing executable must fail")
	}
	if _, err := NewProcesses().Start(launch, func(Event) {}); err == nil {
		t.Fatal("a missing executable must fail to start")
	}
}

// collect starts launch and returns its events once it has exited.
func collect(t *testing.T, processes *Processes, launch Launch, started func(int)) (int, []Event) {
	t.Helper()
	var mu sync.Mutex
	var events []Event
	done := make(chan struct{})
	id, err := processes.Start(launch, func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
		if event["event"] == "exit" {
			close(done)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if started != nil {
		started(id)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the process did not exit")
	}
	mu.Lock()
	defer mu.Unlock()
	return id, events
}

func TestStartStreamsOutputAndEndsWithTheExit(t *testing.T) {
	id, events := collect(t, NewProcesses(), shell("echo out; echo err >&2; exit 4"), nil)
	output := map[string]string{}
	for _, event := range events[:len(events)-1] {
		if event["event"] != "output" || event["process"] != id {
			t.Fatalf("unexpected event %v", event)
		}
		data, _ := base64.StdEncoding.DecodeString(event["data"].(string))
		output[event["stream"].(string)] += string(data)
	}
	if output["stdout"] != "out\n" || output["stderr"] != "err\n" {
		t.Fatalf("unexpected output %v", output)
	}
	last := events[len(events)-1]
	if last["event"] != "exit" || last["code"] != 4 {
		t.Fatalf("unexpected last event %v", last)
	}
}

func TestKillSendsSigterm(t *testing.T) {
	processes := NewProcesses()
	directory := t.TempDir()
	ready := filepath.Join(directory, "ready")
	script := `trap 'exit 42' TERM; touch "` + ready + `"; while :; do sleep 0.05; done`
	_, events := collect(t, processes, shell(script), func(id int) {
		for !Exists(ready) {
			time.Sleep(10 * time.Millisecond)
		}
		processes.Kill(id)
	})
	if last := events[len(events)-1]; last["code"] != 42 {
		t.Fatalf("expected the TERM trap to run, got %v", last)
	}
	// Killing a finished process is harmless.
	processes.Kill(1)
	processes.KillAll()
}

func TestSignalsBecome128PlusTheSignal(t *testing.T) {
	_, events := collect(t, NewProcesses(), shell("kill -KILL $$"), nil)
	if last := events[len(events)-1]; last["code"] != 128+9 {
		t.Fatalf("got %v", last)
	}
}

func TestFiles(t *testing.T) {
	base := t.TempDir()
	settings := filepath.Join(base, "nested", "settings.json")
	if text, err := ReadTextFile(settings); text != nil || err != nil {
		t.Fatalf("a missing file reads as nil, got %v, %v", text, err)
	}
	if err := WriteTextFile(settings, "{}\n"); err != nil {
		t.Fatal(err)
	}
	if text, _ := ReadTextFile(settings); text == nil || *text != "{}\n" {
		t.Fatalf("got %v", text)
	}
	if Exists(settings + ".tmp") {
		t.Fatal("the temporary file must be renamed")
	}
	script := filepath.Join(base, "vfox")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o644); err != nil { //nolint:gosec // A test file.
		t.Fatal(err)
	}
	if IsExecutable(script) {
		t.Fatal("not executable yet")
	}
	if err := os.Chmod(script, 0o755); err != nil { //nolint:gosec // A test file.
		t.Fatal(err)
	}
	if !IsExecutable(script) || IsExecutable(base) || !IsDirectory(base) || IsDirectory(script) {
		t.Fatal("files, directories and executables are told apart")
	}
	if path, err := lookPath("vfox", "/nonexistent::"+base); err != nil || path != script {
		t.Fatalf("got %q, %v", path, err)
	}
}

func TestPrivateCopies(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "mine.lua")
	if err := os.WriteFile(source, []byte("PLUGIN = {}"), 0o644); err != nil { //nolint:gosec // A test file.
		t.Fatal(err)
	}
	root := filepath.Join(base, ".vfox", "tmp")
	if err := CreateDirectory(root); err != nil {
		t.Fatal(err)
	}
	private, err := CreateTemporaryDirectory(root, "vfox-plus-plugin-")
	if err != nil || !strings.HasPrefix(filepath.Base(private), "vfox-plus-plugin-") {
		t.Fatalf("got %q, %v", private, err)
	}
	copied := filepath.Join(private, "mine.lua")
	if err := CopyFile(source, copied); err != nil {
		t.Fatal(err)
	}
	if text, _ := ReadTextFile(copied); text == nil || *text != "PLUGIN = {}" {
		t.Fatal("the copy has the content of the original")
	}
	if err := Remove(private); err != nil || Exists(private) {
		t.Fatal("the private directory is deleted")
	}
	if err := Remove(private); err != nil {
		t.Fatal("removing a missing path is not an error")
	}
	if !Exists(source) || SystemTemporaryDirectory() == "" {
		t.Fatal("the original stays")
	}
}

func TestAccentColours(t *testing.T) {
	colour := func(r, g, b float64) dbus.Variant { return dbus.MakeVariant([]any{r, g, b}) }
	if value, ok := argb(colour(1, 0.5, 0)); !ok || value != 0xFFFF8000 {
		t.Fatalf("got %x", value)
	}
	if value, ok := argb(dbus.MakeVariant(colour(0, 0, 1))); !ok || value != 0xFF0000FF {
		t.Fatalf("got %x", value)
	}
	if _, ok := argb(colour(-1, -1, -1)); ok {
		t.Fatal("an unset colour is rejected")
	}
	if _, ok := argb("blue"); ok {
		t.Fatal("other values are rejected")
	}
}

// fakeDesktop records what the service asks of the desktop.
type fakeDesktop struct {
	mu     sync.Mutex
	opened []string
	copied []string
	events []Event
	exit   int
}

func (d *fakeDesktop) OpenURL(url string) error {
	d.opened = append(d.opened, url)
	return nil
}

func (d *fakeDesktop) PickDirectory(string) (string, error) { return "", nil }

func (d *fakeDesktop) PickFile(initial, _ string, _ []string) (string, error) {
	return initial + "/chosen.lua", nil
}

func (d *fakeDesktop) CopyText(text string) error {
	d.copied = append(d.copied, text)
	return nil
}

func (d *fakeDesktop) Emit(event Event) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, event)
}

func (d *fakeDesktop) Exit(code int) { d.exit = code }

func call(t *testing.T, service *Service, method string, params ...any) map[string]any {
	t.Helper()
	raw := make([]json.RawMessage, len(params))
	for i, param := range params {
		raw[i], _ = json.Marshal(param)
	}
	response := service.Call(Request{ID: 7, Method: method, Params: raw})
	if response["id"] != int64(7) {
		t.Fatalf("the response must repeat the id: %v", response)
	}
	// What the front end receives.
	encoded, _ := json.Marshal(response)
	var decoded map[string]any
	_ = json.Unmarshal(encoded, &decoded)
	return decoded
}

func TestServiceDispatchesTheProtocol(t *testing.T) {
	desktop := &fakeDesktop{}
	service := NewService(desktop)
	base := t.TempDir()
	path := filepath.Join(base, "settings.json")

	if response := call(t, service, "readTextFile", path); response["result"] != nil || response["error"] != nil {
		t.Fatalf("a missing file is null: %v", response)
	}
	if _, ok := call(t, service, "readTextFile", path)["result"]; !ok {
		t.Fatal("a null result is still sent")
	}
	call(t, service, "writeTextFile", path, "{}")
	if response := call(t, service, "readTextFile", path); response["result"] != "{}" {
		t.Fatalf("got %v", response)
	}
	if call(t, service, "exists", path)["result"] != true || call(t, service, "isDirectory", path)["result"] != false {
		t.Fatal("exists and isDirectory")
	}
	if call(t, service, "environment")["result"].(map[string]any)["PATH"] == nil {
		t.Fatal("the environment includes PATH")
	}
	run := call(t, service, "run", "sh", []string{"-c", "echo hi"}, "/", map[string]string{"PATH": "/bin"})
	if run["result"].(map[string]any)["stdout"] != "hi\n" {
		t.Fatalf("got %v", run)
	}
	missing := call(t, service, "run", "/nonexistent/vfox", []string{}, "/", map[string]string{})
	if missing["error"].(map[string]any)["kind"] != "start" {
		t.Fatalf("a missing executable is a start error: %v", missing)
	}
	if call(t, service, "nope")["error"].(map[string]any)["kind"] != "error" {
		t.Fatal("unknown methods fail")
	}
	if call(t, service, "pickDirectory", nil)["result"] != nil {
		t.Fatal("a cancelled chooser is null")
	}
	if call(t, service, "pickFile", "/home", "Plugins", []string{"lua"})["result"] != "/home/chosen.lua" {
		t.Fatal("pickFile")
	}
	call(t, service, "openUrl", "https://vfox.dev")
	call(t, service, "copyText", "eval")
	call(t, service, "exit", 3)
	if desktop.opened[0] != "https://vfox.dev" || desktop.copied[0] != "eval" || desktop.exit != 3 {
		t.Fatal("the desktop operations are forwarded")
	}
	temporary := call(t, service, "createTemporaryDirectory", base, "x-")["result"].(string)
	if !IsDirectory(temporary) {
		t.Fatal("createTemporaryDirectory")
	}
	call(t, service, "remove", temporary)
	if Exists(temporary) || call(t, service, "systemTemporaryDirectory")["result"] == "" {
		t.Fatal("remove")
	}

	started := call(t, service, "start", "sh", []string{"-c", "echo streamed"}, "/", map[string]string{"PATH": "/bin"})
	id := started["result"].(float64)
	deadline := time.Now().Add(10 * time.Second)
	for {
		desktop.mu.Lock()
		count := len(desktop.events)
		desktop.mu.Unlock()
		if count >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	desktop.mu.Lock()
	defer desktop.mu.Unlock()
	if len(desktop.events) != 2 || desktop.events[1]["event"] != "exit" || float64(desktop.events[1]["process"].(int)) != id {
		t.Fatalf("got %v", desktop.events)
	}
}
