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

package measurelightweight

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/measure"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/trayhost"
)

// What the icon in the tray may cost over the application without it, in KB. The icon adds
// 0.4 MB of resident memory and 0.08 MB of private memory. An icon made where no host shows it,
// with the fallback that libayatana-appindicator3 has for that, adds 7 MB and 1.8 MB: that is
// what the limits are for, and the room that they leave is the noise of the machine, which is
// that of the file pages that the system takes back from an idle process as it likes.
const (
	maxIconResident = 5 * 1024
	maxIconPrivate  = 1536
)

// How long the window is open before it is closed, so that the application has loaded what it
// loads when it starts, and the longest that each state is settled before it is measured: the
// state of the application with the window closed depends on what it was doing when the window
// closed, and the system takes back pages of an idle process the longer it waits.
const (
	loadWait    = 5 * time.Second
	maxSettle   = 5 * time.Second
	iconWait    = 30 * time.Second
	hostRelease = time.Second
)

// measureTray measures the application with its window closed twice, in the same way: where
// nothing shows an icon, and on a desktop that has a tray. The stand-in for the host of the tray
// icons (see package trayhost) owns the name of the watcher on the session bus in the second run,
// as the panel does on GNOME, so the application makes the icon that it makes there. It fails when
// the icon costs more than it may.
//
// The measurement is not driven through the automation plugin, which draws more of the window
// and with it grows the glyph atlas of the toolkit by 16 MB in some runs and not in others.
func measureTray(argv []string, fixturesDir, fakeVfox string, settle time.Duration) error {
	settle = min(settle, maxSettle)
	// There has to be a session bus, and no host on it, or the first run would show an icon too:
	// that is found out before the first run is spent on it.
	probe, err := startHost()
	if err != nil {
		return err
	}
	_ = probe.Close()
	time.Sleep(hostRelease)
	withoutIcon, err := closedWindow(argv, fixturesDir, fakeVfox, settle, nil)
	if err != nil {
		return err
	}
	host, err := startHost()
	if err != nil {
		return err
	}
	defer func() { _ = host.Close() }()
	withIcon, err := closedWindow(argv, fixturesDir, fakeVfox, settle, host)
	if err != nil {
		return err
	}
	fmt.Printf("memory (lightweight-no-tray): RSS %d MB, PSS %d MB in %d processes\n", withoutIcon.RSS/1024, withoutIcon.PSS/1024, withoutIcon.Processes)
	fmt.Printf("memory (lightweight-tray): RSS %d MB, PSS %d MB in %d processes\n", withIcon.RSS/1024, withIcon.PSS/1024, withIcon.Processes)
	return checkIconCost(withoutIcon, withIcon)
}

// startHost makes the stand-in for the host of the tray icons.
func startHost() (*trayhost.Host, error) {
	host, err := trayhost.Start()
	if err != nil {
		return nil, fmt.Errorf("start the stand-in for the host of the tray icons (it needs the session bus of a desktop session, "+
			"and no desktop that has a tray of its own; dbus-run-session gives one): %w", err)
	}
	return host, nil
}

// closedWindow runs the application with the web interface and the lightweight mode on, closes its
// window when it has loaded, waits for the icon to come into the tray when there is a host, and
// returns what it occupies when it has settled.
func closedWindow(argv []string, fixturesDir, fakeVfox string, settle time.Duration, host *trayhost.Host) (measure.Usage, error) {
	homeDir, err := os.MkdirTemp("", "vfox-plus-measure-tray-")
	if err != nil {
		return measure.Usage{}, err
	}
	defer func() { _ = os.RemoveAll(homeDir) }()
	env, linkFile, err := measureEnv(homeDir, fixturesDir, fakeVfox)
	if err != nil {
		return measure.Usage{}, err
	}
	app, err := startApplication(argv, env)
	if err != nil {
		return measure.Usage{}, err
	}
	if _, err := waitForLink(linkFile, app.done, 60*time.Second); err != nil {
		app.stop()
		return measure.Usage{}, err
	}
	time.Sleep(loadWait)
	if err := app.command.Process.Signal(closeSignal); err != nil {
		app.stop()
		return measure.Usage{}, fmt.Errorf("cannot ask the application to close its window: %w", err)
	}
	if host != nil {
		ctx, cancel := context.WithTimeout(context.Background(), iconWait)
		defer cancel()
		if _, err := host.NextItem(ctx); err != nil {
			app.stop()
			return measure.Usage{}, fmt.Errorf("the window is closed and the application made no icon in the tray (is the library of the icon, libayatana-appindicator3, installed?): %w", err)
		}
	}
	time.Sleep(settle)
	select {
	case err := <-app.done:
		return measure.Usage{}, fmt.Errorf("the application exited early: %v", err)
	default:
	}
	usage := measure.UsageOf(app.command.Process.Pid)
	app.stop()
	if host != nil {
		// The icon of this run is gone with its application, and the next one starts clean.
		time.Sleep(hostRelease)
	}
	return usage, nil
}

// checkIconCost says what the icon costs, and fails when that is more than it may.
func checkIconCost(withoutIcon, withIcon measure.Usage) error {
	resident, private := withIcon.RSS-withoutIcon.RSS, withIcon.Private-withoutIcon.Private
	fmt.Printf("the icon in the tray costs %+d KB of resident memory and %+d KB of private memory\n", resident, private)
	var problems []string
	if resident > maxIconResident {
		problems = append(problems, fmt.Sprintf("%d KB of resident memory, at most %d", resident, maxIconResident))
	}
	if private > maxIconPrivate {
		problems = append(problems, fmt.Sprintf("%d KB of private memory, at most %d", private, maxIconPrivate))
	}
	if len(problems) > 0 {
		return fmt.Errorf("the icon in the tray costs too much memory: %s", strings.Join(problems, "; "))
	}
	return nil
}
