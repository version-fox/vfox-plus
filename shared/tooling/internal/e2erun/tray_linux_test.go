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

//go:build linux

package e2erun

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The scenario needs the session bus of a desktop session: the host of the tray icons is on it,
// and the application finds the host there.
func TestTheTrayScenarioNeedsTheSessionBusOfADesktop(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "no-bus"))
	err := Run([]string{"tray", "/nonexistent/application"})
	if err == nil || !strings.Contains(err.Error(), "session bus") {
		t.Fatalf("Run(tray) = %v, want a failure to reach the session bus", err)
	}
}

func TestTheTrayScenarioFailsWhenTheApplicationCannotBeStarted(t *testing.T) {
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon to start a session bus with")
	}
	cmd := exec.Command(daemon, "--session", "--nofork", "--print-address=1") //nolint:gosec // The daemon of the test.
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	address, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("the session bus gave no address: %v", err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(address))
	t.Setenv("TMPDIR", t.TempDir())
	if err := Run([]string{"tray", "/nonexistent/application"}); err == nil || !strings.Contains(err.Error(), "start the application") {
		t.Fatalf("Run(tray) = %v, want a failure to start the application", err)
	}
	if leftovers, _ := os.ReadDir(os.Getenv("TMPDIR")); len(leftovers) != 0 {
		t.Errorf("the scenario left %d things in the temporary directory", len(leftovers))
	}
}

// The GNOME scenario runs against the shell of the session, and says so when the session has none, before the
// application is started.
func TestTheGnomeScenarioNeedsAShellWithTheExtension(t *testing.T) {
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon to start a session bus with")
	}
	cmd := exec.Command(daemon, "--session", "--nofork", "--print-address=1") //nolint:gosec // The daemon of the test.
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	address, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("the session bus gave no address: %v", err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(address))
	err = Run([]string{"tray-gnome", "/nonexistent/application"})
	if err == nil || !strings.Contains(err.Error(), "no GNOME Shell with the AppIndicator extension") {
		t.Fatalf("Run(tray-gnome) = %v, want the lack of a shell", err)
	}
}

func TestTheGnomeScenarioNeedsASessionBus(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "no-bus"))
	err := Run([]string{"tray-gnome", "/nonexistent/application"})
	if err == nil || !strings.Contains(err.Error(), "no GNOME Shell with the AppIndicator extension") {
		t.Fatalf("Run(tray-gnome) = %v", err)
	}
}
