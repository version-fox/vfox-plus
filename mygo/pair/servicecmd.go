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

package pair

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ServiceMain runs `vfox-plus service`: it manages the shared background
// server that owns the application state for this user. The desktop
// application registers itself while it runs; everywhere else one daemon
// (or one loopback `serve`) holds the file.
func ServiceMain(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus service <start|stop|status|set|unset>")
		return 2
	}
	switch args[0] {
	case "start":
		return serviceStart(args[1:])
	case "stop":
		return serviceStop(args[1:])
	case "status":
		return serviceStatus(args[1:])
	case "set":
		return serviceSet(args[1:])
	case "unset":
		return serviceUnset(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus service <start|stop|status|set|unset>")
		return 2
	}
}

func servicePort(args []string) (int, bool) {
	flags := flag.NewFlagSet("start", flag.ContinueOnError)
	port := flags.Int("port", 0, "port to listen on (0 picks a free one)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, false
		}
		return 2, false
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus service start [--port 0]")
		return 2, false
	}
	return *port, true
}

// serviceStart launches a detached loopback `serve` as the shared server and
// waits until it registers itself. It refuses when a live server — the
// desktop, a daemon or a loopback `serve` — already owns the state.
func serviceStart(args []string) int {
	port, ok := servicePort(args)
	if !ok {
		return port
	}
	settingsDir := LocalSettingsDir()
	if live := DiscoverService(settingsDir); live != nil {
		fmt.Fprintf(os.Stderr, "service: already served by %s (PID %d, port %d)\n", live.Owner, live.PID, live.Port)
		return 1
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "service:", err)
		return 1
	}
	logPath := serviceLogPath(settingsDir)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // The log path is the service log in the settings directory.
	if err != nil {
		fmt.Fprintln(os.Stderr, "service:", err)
		return 1
	}
	command := exec.Command(executable, "serve", "--bind", DefaultBind, "--port", fmt.Sprint(port)) //nolint:gosec // The executable is this binary; there is no shell.
	command.Stdout = logFile
	command.Stderr = logFile
	detachProcess(command)
	if err := command.Start(); err != nil {
		_ = logFile.Close()
		fmt.Fprintln(os.Stderr, "service:", err)
		return 1
	}
	_ = logFile.Close()
	_ = command.Process.Release()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if live := DiscoverService(settingsDir); live != nil {
			fmt.Printf("\n  Serving vfox-plus on http://127.0.0.1:%d\n", live.Port)
			fmt.Println("  The desktop, `vfox-plus pair` and browsers attach to this server.")
			fmt.Println()
			return 0
		}
		if time.Now().After(deadline) {
			fmt.Fprintln(os.Stderr, "service: the server did not register itself; see "+logPath)
			return 1
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// serviceStop ends the daemon recorded in the service file. It refuses to
// stop a desktop-owned server: closing the window stops that one.
func serviceStop(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus service stop")
		return 2
	}
	settingsDir := LocalSettingsDir()
	info, err := ReadServiceInfo(ServiceFilePath(settingsDir))
	if err != nil {
		fmt.Fprintln(os.Stderr, "service: no shared server is recorded")
		return 1
	}
	if !ProbeService(info) {
		_ = os.Remove(ServiceFilePath(settingsDir))
		fmt.Fprintln(os.Stderr, "service: no shared server is running")
		return 1
	}
	if info.Owner != ServiceOwnerService {
		fmt.Fprintf(os.Stderr, "service: served by %s (PID %d); stop it there\n", info.Owner, info.PID)
		return 1
	}
	if err := terminateProcess(info.PID); err != nil {
		fmt.Fprintln(os.Stderr, "service:", err)
		return 1
	}
	_ = os.Remove(ServiceFilePath(settingsDir))
	fmt.Println("  The shared server is stopped.")
	fmt.Println()
	return 0
}

// serviceStatus reports the shared server.
func serviceStatus(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus service status")
		return 2
	}
	settingsDir := LocalSettingsDir()
	if live := DiscoverService(settingsDir); live != nil {
		fmt.Printf("\n  Shared server: %s (PID %d) on http://127.0.0.1:%d since %s\n", live.Owner, live.PID, live.Port, live.StartedAt)
		if ServiceDisabled(settingsDir) {
			fmt.Println("  The background service is disabled.")
		}
		fmt.Println()
		return 0
	}
	fmt.Println()
	fmt.Println("  No shared server is running.")
	if ServiceDisabled(settingsDir) {
		fmt.Println("  The background service is disabled.")
	}
	fmt.Println()
	return 0
}

// serviceSet records service settings: `service set disabled true|false`.
func serviceSet(args []string) int {
	if len(args) != 2 || args[0] != "disabled" || (args[1] != "true" && args[1] != "false") {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus service set disabled true|false")
		return 2
	}
	if err := SetServiceDisabled(LocalSettingsDir(), args[1] == "true"); err != nil {
		fmt.Fprintln(os.Stderr, "service:", err)
		return 1
	}
	return 0
}

// serviceUnset clears service settings: `service unset disabled`.
func serviceUnset(args []string) int {
	if len(args) != 1 || args[0] != "disabled" {
		fmt.Fprintln(os.Stderr, "Usage: vfox-plus service unset disabled")
		return 2
	}
	if err := SetServiceDisabled(LocalSettingsDir(), false); err != nil {
		fmt.Fprintln(os.Stderr, "service:", err)
		return 1
	}
	return 0
}

func serviceLogPath(settingsDir string) string {
	return settingsDir + string(os.PathSeparator) + "service.log"
}
