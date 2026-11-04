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

package gatewaycheck

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeDocker puts a program named docker first on the PATH that records how it is called and answers as Docker
// does for the commands that the image program uses. It returns the file that has the calls, a line each.
func fakeDocker(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake Docker is a shell script")
	}
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls.txt")
	script := `#!/bin/sh
echo "$*" >> "$FAKE_DOCKER_CALLS"
if [ "$FAKE_DOCKER_FAIL" = "$1" ] || [ "$FAKE_DOCKER_FAIL" = "$1 $2" ]; then
  echo "docker said no to $1" >&2
  exit 1
fi
case "$1" in
  volume) [ "$2" = create ] && echo "$3" ;;
  run) echo "c0ffee0123456789" ;;
  port) echo "${FAKE_DOCKER_PORT:-127.0.0.1:49153}" ;;
  logs) echo "from stdout"; echo "from stderr" >&2 ;;
  inspect) echo "${FAKE_DOCKER_EXIT:-0}" ;;
esac
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755); err != nil { //nolint:gosec // A script of the test.
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DOCKER_CALLS", calls)
	return calls
}

// callsOf are the calls of the fake Docker so far, a line each.
func callsOf(t *testing.T, file string) []string {
	t.Helper()
	content, err := os.ReadFile(file) //nolint:gosec // The file of this test.
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(content)), "\n")
}

func TestAContainerIsRunWithAVolumeAHostNameAndAPortOnTheLoopback(t *testing.T) {
	calls := fakeDocker(t)
	program := newImage("vfox-plus-gateway:ci")
	started, err := program.start()
	if err != nil {
		t.Fatal(err)
	}
	if started.address() != "127.0.0.1:49153" {
		t.Errorf("address = %q", started.address())
	}
	if got := started.output(); !strings.Contains(got, "from stdout") || !strings.Contains(got, "from stderr") {
		t.Errorf("output = %q, want both of the outputs of the container", got)
	}
	all := callsOf(t, calls)
	run := ""
	for _, call := range all {
		if strings.HasPrefix(call, "run ") {
			run = call
		}
	}
	for _, want := range []string{
		"--detach", "--publish 127.0.0.1::8080", "--add-host host.docker.internal:host-gateway",
		"--volume " + program.volume + ":/home/vfoxplus/.config/vfox-plus-gateway",
	} {
		if !strings.Contains(run, want) {
			t.Errorf("run = %q, want it to have %q", run, want)
		}
	}
	if !strings.HasSuffix(run, " vfox-plus-gateway:ci") {
		t.Errorf("run = %q, want the image last", run)
	}
	if all[0] != "volume create "+program.volume {
		t.Errorf("the volume is made before the container: %v", all)
	}
	// A gateway that is started again is the same volume.
	if _, err := program.start(); err != nil {
		t.Fatal(err)
	}
	volumes := 0
	for _, call := range callsOf(t, calls) {
		if strings.HasPrefix(call, "volume create") {
			volumes++
		}
	}
	if volumes != 1 {
		t.Errorf("%d volumes were made, want 1: the gateway that is started again remembers in the same", volumes)
	}
}

func TestAContainerIsStoppedAsDockerStopsItAndMustHaveEndedWell(t *testing.T) {
	calls := fakeDocker(t)
	started, err := newImage("img").start()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := started.stop(); err != nil {
		t.Errorf("stop = %v", err)
	}
	all := callsOf(t, calls)
	tail := all[len(all)-3:]
	want := []string{"stop --time 10 c0ffee0123456789", "inspect --format {{.State.ExitCode}} c0ffee0123456789", "rm --force c0ffee0123456789"}
	for i := range want {
		if tail[i] != want[i] {
			t.Errorf("call %d is %q, want %q", i, tail[i], want[i])
		}
	}

	t.Setenv("FAKE_DOCKER_EXIT", "137")
	killed, err := newImage("img").start()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := killed.stop(); err == nil || !strings.Contains(err.Error(), "code 137") {
		t.Errorf("a container that was killed: %v", err)
	}
	t.Setenv("FAKE_DOCKER_EXIT", "0")
	t.Setenv("FAKE_DOCKER_FAIL", "stop")
	hung, err := newImage("img").start()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hung.stop(); err == nil || !strings.Contains(err.Error(), "docker said no") {
		t.Errorf("docker stop that fails: %v", err)
	}
	t.Setenv("FAKE_DOCKER_FAIL", "inspect")
	unknown, err := newImage("img").start()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unknown.stop(); err == nil {
		t.Error("a container whose exit code is not known ended well")
	}
}

func TestWhatDockerCannotDoIsSaid(t *testing.T) {
	calls := fakeDocker(t)
	t.Setenv("FAKE_DOCKER_FAIL", "volume create")
	if _, err := newImage("img").start(); err == nil || !strings.Contains(err.Error(), "docker said no") {
		t.Errorf("a volume that cannot be made: %v", err)
	}
	t.Setenv("FAKE_DOCKER_FAIL", "run")
	failing := newImage("no-such-image")
	if _, err := failing.start(); err == nil || !strings.Contains(err.Error(), "docker said no to run") {
		t.Errorf("an image that cannot run: %v", err)
	}
	failing.cleanup()
	last := callsOf(t, calls)
	if got := last[len(last)-1]; got != "volume rm --force "+failing.volume {
		t.Errorf("the volume that was made is removed, got %q", got)
	}

	// A container that was started and whose port is not known is not left running.
	t.Setenv("FAKE_DOCKER_FAIL", "port")
	if _, err := newImage("img").start(); err == nil {
		t.Error("a container without a port started")
	}
	t.Setenv("FAKE_DOCKER_FAIL", "")
	t.Setenv("FAKE_DOCKER_PORT", "garbage")
	if _, err := newImage("img").start(); err == nil || !strings.Contains(err.Error(), "garbage") {
		t.Errorf("a port that is not one: %v", err)
	}
	removed := 0
	for _, call := range callsOf(t, calls) {
		if call == "rm --force c0ffee0123456789" {
			removed++
		}
	}
	if removed != 2 {
		t.Errorf("%d containers were removed, want the 2 that could not be used", removed)
	}
}

func TestAnImageThatNeverRanLeavesNoVolumeToRemove(t *testing.T) {
	calls := fakeDocker(t)
	newImage("img").cleanup()
	if all := callsOf(t, calls); len(all) != 0 {
		t.Errorf("cleanup called Docker: %v", all)
	}
}

func TestAPortOfASecondFamilyOfAddressesIsNotTakenForTheFirst(t *testing.T) {
	fakeDocker(t)
	// docker port prints a line for each family of addresses that the port is published on.
	t.Setenv("FAKE_DOCKER_PORT", "127.0.0.1:49153\n[::1]:49153")
	started, err := newImage("img").start()
	if err != nil {
		t.Fatal(err)
	}
	if started.address() != "127.0.0.1:49153" {
		t.Errorf("address = %q", started.address())
	}
}
